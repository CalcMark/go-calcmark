package cmd

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/CalcMark/go-calcmark/format"
	implDoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
	"golang.org/x/net/websocket"
)

var watchPort int

var watchCmd = &cobra.Command{
	Use:   "watch <file.cm>",
	Short: "Watch a CalcMark file and serve a live preview",
	Long: `Watch a CalcMark file for changes and serve a live HTML preview
in the browser. The preview updates automatically on every save.

Security: binds to 127.0.0.1 only, uses a random session token in the URL,
and validates WebSocket origins.

Example:
  cm watch budget.cm     Open http://127.0.0.1:3141/<token> in your browser`,
	Args: cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		return runWatch(args[0])
	},
}

func init() {
	watchCmd.Flags().IntVar(&watchPort, "port", 3141, "Port to listen on")
	rootCmd.AddCommand(watchCmd)
}

func runWatch(filename string) error {
	if err := validateReadFilePath(filename); err != nil {
		return fmt.Errorf("invalid file: %w", err)
	}

	// Generate random session token for URL path
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Errorf("generate session token: %w", err)
	}
	sessionToken := hex.EncodeToString(tokenBytes)

	// Initial render
	html, err := renderFile(filename)
	if err != nil {
		return fmt.Errorf("initial render: %w", err)
	}

	srv := &watchServer{
		filename:     filename,
		sessionToken: sessionToken,
		html:         html,
	}

	// Set up file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}
	defer watcher.Close()

	if err := watcher.Add(filename); err != nil {
		return fmt.Errorf("watch file: %w", err)
	}

	// Start watching for changes in background
	go srv.watchLoop(watcher)

	// Set up HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/"+sessionToken, srv.handlePage)
	mux.Handle("/"+sessionToken+"/ws", websocket.Handler(srv.handleWebSocket))

	addr := fmt.Sprintf("127.0.0.1:%d", watchPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}

	httpServer := &http.Server{
		Handler:      srv.securityMiddleware(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	fmt.Fprintf(os.Stderr, "Watching %s\n", filename)
	fmt.Fprintf(os.Stderr, "Preview: http://%s/%s\n", addr, sessionToken)

	// Graceful shutdown on interrupt
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		httpServer.Shutdown(shutdownCtx)
	}()

	return httpServer.Serve(listener)
}

// watchServer holds state for the live preview server.
type watchServer struct {
	filename     string
	sessionToken string

	mu      sync.RWMutex
	html    string
	clients map[*websocket.Conn]struct{}
}

// watchLoop listens for file changes and re-renders.
func (s *watchServer) watchLoop(watcher *fsnotify.Watcher) {
	// Debounce: wait for writes to settle before re-rendering
	var debounceTimer *time.Timer

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(100*time.Millisecond, func() {
					html, err := renderFile(s.filename)
					if err != nil {
						fmt.Fprintf(os.Stderr, "render error: %v\n", err)
						return
					}
					s.mu.Lock()
					s.html = html
					clients := make(map[*websocket.Conn]struct{}, len(s.clients))
					for c := range s.clients {
						clients[c] = struct{}{}
					}
					s.mu.Unlock()

					// Notify all WebSocket clients
					for conn := range clients {
						if err := websocket.Message.Send(conn, "reload"); err != nil {
							s.removeClient(conn)
						}
					}
				})
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fmt.Fprintf(os.Stderr, "watch error: %v\n", err)
		}
	}
}

func (s *watchServer) addClient(conn *websocket.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.clients == nil {
		s.clients = make(map[*websocket.Conn]struct{})
	}
	s.clients[conn] = struct{}{}
}

func (s *watchServer) removeClient(conn *websocket.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, conn)
	conn.Close()
}

// securityMiddleware enforces security headers and origin validation.
func (s *watchServer) securityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only allow loopback origins for WebSocket upgrade requests
		if r.Header.Get("Upgrade") == "websocket" {
			origin := r.Header.Get("Origin")
			if origin != "" && !isLoopbackOrigin(origin) {
				http.Error(w, "forbidden origin", http.StatusForbidden)
				return
			}
		}

		// Require session token in path
		if !strings.HasPrefix(r.URL.Path, "/"+s.sessionToken) {
			http.NotFound(w, r)
			return
		}

		// Security headers
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; connect-src 'self' ws://127.0.0.1:* ws://localhost:*")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")

		next.ServeHTTP(w, r)
	})
}

// handlePage serves the preview HTML page with embedded WebSocket client.
func (s *watchServer) handlePage(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	html := s.html
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, watchPageTemplate, html, s.sessionToken)
}

// handleWebSocket handles WebSocket connections for live reload.
func (s *watchServer) handleWebSocket(conn *websocket.Conn) {
	s.addClient(conn)
	defer s.removeClient(conn)

	// Keep connection alive — read loop blocks until client disconnects
	for {
		var msg string
		if err := websocket.Message.Receive(conn, &msg); err != nil {
			return
		}
	}
}

// isLoopbackOrigin checks if the origin is from a loopback address.
func isLoopbackOrigin(origin string) bool {
	return strings.Contains(origin, "127.0.0.1") ||
		strings.Contains(origin, "localhost") ||
		strings.Contains(origin, "[::1]")
}

// renderFile reads, parses, evaluates, and renders a CalcMark file to HTML.
func renderFile(filename string) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}

	doc, err := document.NewDocument(string(data))
	if err != nil {
		return "", err
	}

	eval := implDoc.NewEvaluator()
	eval.Evaluate(doc)

	var buf bytes.Buffer
	formatter := &format.HTMLFormatter{}
	if err := formatter.Format(&buf, doc, format.Options{}); err != nil {
		return "", err
	}

	return buf.String(), nil
}

const watchPageTemplate = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>CalcMark Preview</title>
  <style>
    body { font-family: system-ui, sans-serif; max-width: 800px; margin: 2rem auto; padding: 0 1rem; }
    #status { position: fixed; top: 8px; right: 8px; padding: 4px 8px; border-radius: 4px; font-size: 12px; }
    .connected { background: #d4edda; color: #155724; }
    .disconnected { background: #f8d7da; color: #721c24; }
  </style>
</head>
<body>
  <div id="status" class="disconnected">disconnected</div>
  <div id="content">%s</div>
  <script>
    (function() {
      var proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
      var ws;
      var statusEl = document.getElementById('status');
      function connect() {
        ws = new WebSocket(proto + '//' + location.host + '/%s/ws');
        ws.onopen = function() { statusEl.textContent = 'live'; statusEl.className = 'connected'; };
        ws.onclose = function() { statusEl.textContent = 'disconnected'; statusEl.className = 'disconnected'; setTimeout(connect, 1000); };
        ws.onmessage = function(e) { if (e.data === 'reload') location.reload(); };
      }
      connect();
    })();
  </script>
</body>
</html>`
