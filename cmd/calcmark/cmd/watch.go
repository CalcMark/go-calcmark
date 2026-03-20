package cmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/CalcMark/go-calcmark"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
	"golang.org/x/net/websocket"
)

var watchPort int

var watchCmd = &cobra.Command{
	Use:   "watch <file>",
	Short: "Watch a CalcMark file and serve a live preview",
	Long: `Watch a CalcMark or Markdown file for changes and serve a live HTML preview
in the browser. The preview updates automatically on every save.

For .cm/.calcmark files, the entire file is evaluated as CalcMark.
For .md/.markdown files, embedded cm/calcmark fenced code blocks are evaluated
and the surrounding Markdown prose is rendered to HTML.

Security: binds to 127.0.0.1 only, uses a random session token in the URL,
and validates WebSocket origins.

Examples:
  cm watch budget.cm       CalcMark live preview
  cm watch report.md       Embedded mode live preview`,
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
	// Detect mode from file extension.
	mode := calcmark.CM
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == ".md" || ext == ".markdown" {
		mode = calcmark.Embedded
		if err := validateReadFilePathEmbedded(filename); err != nil {
			return fmt.Errorf("invalid file: %w", err)
		}
	} else {
		if err := validateReadFilePath(filename); err != nil {
			return fmt.Errorf("invalid file: %w", err)
		}
	}

	// Generate random session token for URL path
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Errorf("generate session token: %w", err)
	}
	sessionToken := hex.EncodeToString(tokenBytes)

	// Initial render
	html, err := renderFile(filename, mode)
	if err != nil {
		return fmt.Errorf("initial render: %w", err)
	}

	srv := &watchServer{
		filename:     filename,
		mode:         mode,
		sessionToken: sessionToken,
		html:         html,
	}

	// Set up file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}
	defer watcher.Close()

	if err := srv.addWatch(watcher); err != nil {
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
	mode         calcmark.Mode
	sessionToken string

	mu      sync.RWMutex
	html    string
	clients map[*websocket.Conn]struct{}

	logw io.Writer // destination for log output; nil means os.Stderr
}

// logf writes a formatted log line to the server's log writer.
func (s *watchServer) logf(format string, args ...any) {
	w := s.logw
	if w == nil {
		w = os.Stderr
	}
	fmt.Fprintf(w, format+"\n", args...)
}

// addWatch registers the parent directory with the fsnotify watcher.
// Watching the directory instead of the file ensures atomic saves
// (write-temp-rename) are detected on all platforms.
func (s *watchServer) addWatch(watcher *fsnotify.Watcher) error {
	absPath, err := filepath.Abs(s.filename)
	if err != nil {
		return err
	}
	s.filename = absPath
	return watcher.Add(filepath.Dir(absPath))
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
			// Filter: only react to events on the target file.
			// We watch the directory to catch atomic saves (write-temp-rename).
			if event.Name != s.filename {
				continue
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) {
				s.logf("[watch] change detected: %s", filepath.Base(s.filename))
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(100*time.Millisecond, func() {
					start := time.Now()
					html, err := renderFile(s.filename, s.mode)
					if err != nil {
						s.logf("[watch] render error: %v", err)
						return
					}
					s.logf("[watch] re-rendered (%s)", time.Since(start).Round(time.Millisecond))

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
					if n := len(clients); n > 0 {
						s.logf("[watch] notified %d client(s)", n)
					}
				})
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			s.logf("[watch] error: %v", err)
		}
	}
}

func (s *watchServer) addClient(conn *websocket.Conn) {
	s.mu.Lock()
	if s.clients == nil {
		s.clients = make(map[*websocket.Conn]struct{})
	}
	s.clients[conn] = struct{}{}
	n := len(s.clients)
	s.mu.Unlock()
	s.logf("[watch] client connected (%d total)", n)
}

func (s *watchServer) removeClient(conn *websocket.Conn) {
	s.mu.Lock()
	delete(s.clients, conn)
	n := len(s.clients)
	s.mu.Unlock()
	conn.Close()
	s.logf("[watch] client disconnected (%d total)", n)
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
	page := strings.Replace(watchPageTemplate, "{{CONTENT}}", html, 1)
	page = strings.Replace(page, "{{TOKEN}}", s.sessionToken, 1)
	io.WriteString(w, page)
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

// renderFile reads and converts a CalcMark or Markdown file to HTML.
func renderFile(filename string, mode calcmark.Mode) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}

	result, err := calcmark.Convert(string(data), calcmark.Options{
		Mode:   mode,
		Format: "html",
	})
	// For embedded mode, partial errors still produce useful output.
	if err != nil && result == "" {
		return "", err
	}
	return result, nil
}

const watchPageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>CalcMark Preview</title>
  <style>
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
      max-width: 900px;
      margin: 0 auto;
      padding: 2rem;
      line-height: 1.6;
      color: #333;
    }

    #status {
      position: fixed;
      top: 8px;
      right: 8px;
      padding: 4px 8px;
      border-radius: 4px;
      font-size: 12px;
      z-index: 1000;
    }
    .connected { background: #d4edda; color: #155724; }
    .disconnected { background: #f8d7da; color: #721c24; }

    .calc-block {
      margin: 1.5em 0;
      padding: 1em;
      background: #f8f9fa;
      border-left: 4px solid #0066cc;
      border-radius: 4px;
    }

    .calc-line {
      display: flex;
      justify-content: space-between;
      align-items: baseline;
      margin: 0.25em 0;
    }

    .calc-source {
      font-family: 'SF Mono', Monaco, 'Cascadia Code', 'Roboto Mono', Consolas, 'Courier New', monospace;
      font-size: 0.95em;
      color: #24292e;
      flex: 1;
    }

    .calc-inline-result {
      font-weight: 600;
      color: #0066cc;
      margin-left: 2em;
      font-size: 0.9em;
    }

    .calc-inline-result::before {
      content: "= ";
    }

    .calc-result {
      font-weight: 600;
      color: #0066cc;
      margin-top: 0.5em;
      padding: 0.5em;
      background: white;
      border-radius: 3px;
    }

    .calc-error {
      color: #d73a49;
      background: #ffeef0;
      padding: 0.5em;
      border-radius: 3px;
      border-left: 3px solid #d73a49;
      margin-top: 0.5em;
    }

    .text-block {
      margin: 1.5em 0;
    }

    .text-block p {
      margin: 0.75em 0;
    }

    .cm-interpolated {
      font-weight: 600;
    }

    .text-block h1, .text-block h2, .text-block h3 {
      margin-top: 1.5em;
      margin-bottom: 0.5em;
    }

    .text-block code {
      background: #f6f8fa;
      padding: 0.2em 0.4em;
      border-radius: 3px;
      font-family: 'SF Mono', Monaco, monospace;
      font-size: 0.9em;
    }

    .text-block pre {
      background: #f6f8fa;
      padding: 1em;
      border-radius: 6px;
      overflow-x: auto;
    }

    .text-block pre code {
      background: none;
      padding: 0;
    }

    .text-block blockquote {
      border-left: 3px solid #0066cc;
      padding-left: 1em;
      color: #57606a;
      margin: 1em 0;
    }

    .text-block blockquote p {
      margin: 0.5em 0;
    }

    .text-block table {
      border-collapse: collapse;
      width: 100%;
      margin: 1em 0;
    }

    .text-block th, .text-block td {
      border: 1px solid #d0d7de;
      padding: 0.5em 0.75em;
      text-align: left;
    }

    .text-block th {
      background: #f0f4f8;
      font-weight: 600;
    }

    .frontmatter {
      margin-bottom: 2em;
      padding: 1em 1.5em;
      background: #f0f4f8;
      border-radius: 6px;
      border: 1px solid #d0d7de;
    }

    .frontmatter h3 {
      margin: 0 0 0.75em 0;
      font-size: 0.9em;
      color: #57606a;
      text-transform: uppercase;
      letter-spacing: 0.05em;
    }

    .frontmatter dl {
      margin: 0;
      display: grid;
      grid-template-columns: auto 1fr;
      gap: 0.25em 1em;
    }

    .frontmatter dt {
      font-family: 'SF Mono', Monaco, 'Cascadia Code', 'Roboto Mono', Consolas, monospace;
      font-size: 0.9em;
      color: #0550ae;
    }

    .frontmatter dt::before {
      content: "@";
      color: #6e7781;
    }

    .frontmatter dd {
      margin: 0;
      font-family: 'SF Mono', Monaco, 'Cascadia Code', 'Roboto Mono', Consolas, monospace;
      font-size: 0.9em;
      color: #24292e;
    }

    .frontmatter .exchange dt::before {
      content: "";
    }

    .frontmatter .exchange dt {
      color: #6e7781;
    }

    .frontmatter hr {
      border: none;
      border-top: 1px solid #d0d7de;
      margin: 0.75em 0;
    }

    .frontmatter-value {
      margin: 0;
      font-family: 'SF Mono', Monaco, 'Cascadia Code', 'Roboto Mono', Consolas, monospace;
      font-size: 0.9em;
      color: #24292e;
    }

    .frontmatter .extra dt::before {
      content: "";
    }

    .frontmatter .extra dt {
      color: #57606a;
    }
  </style>
</head>
<body>
  <div id="status" class="disconnected">disconnected</div>
  <div id="content">{{CONTENT}}</div>
  <script>
    (function() {
      var proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
      var ws;
      var statusEl = document.getElementById('status');
      function connect() {
        ws = new WebSocket(proto + '//' + location.host + '/{{TOKEN}}/ws');
        ws.onopen = function() { statusEl.textContent = 'live'; statusEl.className = 'connected'; };
        ws.onclose = function() { statusEl.textContent = 'disconnected'; statusEl.className = 'disconnected'; setTimeout(connect, 1000); };
        ws.onmessage = function(e) { if (e.data === 'reload') location.reload(); };
      }
      connect();
    })();
  </script>
</body>
</html>`
