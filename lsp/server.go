package lsp

import (
	"context"
	"fmt"
	"sync"
	"time"

	implDoc "github.com/CalcMark/go-calcmark/impl/document"
	specDoc "github.com/CalcMark/go-calcmark/spec/document"
	"github.com/tliron/commonlog"
	"github.com/tliron/glsp"
	glspServer "github.com/tliron/glsp/server"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

var serverVersionStr = "0.1.0"

const (
	serverName = "calcmark-lsp"

	// maxDocumentSize is the maximum document size the LSP server will process.
	maxDocumentSize = 1 * 1024 * 1024 // 1 MB

	// evalTimeout is the maximum time allowed for a single evaluation.
	evalTimeout = 1 * time.Second

	// debounceDelay is the delay before triggering re-evaluation after a change.
	debounceDelay = 150 * time.Millisecond
)

// DocumentSnapshot holds the immutable evaluation results for a document.
// Request handlers read from the latest snapshot without locks.
type DocumentSnapshot struct {
	Source      string
	Document    *specDoc.Document
	Evaluator   *implDoc.Evaluator
	Diagnostics []specDoc.Diagnostic
	HTML        string
}

// documentState holds the mutable state for a single open document.
type documentState struct {
	mu       sync.RWMutex
	snapshot *DocumentSnapshot
	cancel   context.CancelFunc // cancel in-flight evaluation
	timer    *time.Timer        // debounce timer
}

func (ds *documentState) getSnapshot() *DocumentSnapshot {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.snapshot
}

func (ds *documentState) setSnapshot(snap *DocumentSnapshot) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.snapshot = snap
}

// Server is the CalcMark LSP server.
type Server struct {
	handler protocol.Handler
	server  *glspServer.Server
	log     commonlog.Logger

	mu        sync.RWMutex
	documents map[string]*documentState // URI -> state
}

// NewServer creates a new CalcMark LSP server.
func NewServer() *Server {
	s := &Server{
		documents: make(map[string]*documentState),
		log:       commonlog.GetLogger(serverName),
	}

	s.handler = protocol.Handler{
		Initialize:  s.initialize,
		Initialized: s.initialized,
		Shutdown:    s.shutdown,
		SetTrace:    s.setTrace,

		TextDocumentDidOpen:   s.textDocumentDidOpen,
		TextDocumentDidChange: s.textDocumentDidChange,
		TextDocumentDidClose:  s.textDocumentDidClose,

		TextDocumentCompletion:    s.textDocumentCompletion,
		TextDocumentHover:         s.textDocumentHover,
		TextDocumentDefinition:    s.textDocumentDefinition,
		TextDocumentDocumentSymbol: s.textDocumentDocumentSymbol,
	}

	s.server = glspServer.NewServer(&s.handler, serverName, false)

	return s
}

// RunStdio starts the LSP server over stdio.
func (s *Server) RunStdio() error {
	return s.server.RunStdio()
}

// initialize handles the LSP initialize request.
func (s *Server) initialize(_ *glsp.Context, params *protocol.InitializeParams) (any, error) {
	if params.ClientInfo != nil {
		s.log.Infof("client: %s", params.ClientInfo.Name)
	}

	syncKind := protocol.TextDocumentSyncKindFull

	return protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{
			TextDocumentSync: &protocol.TextDocumentSyncOptions{
				OpenClose: boolPtr(true),
				Change:    &syncKind,
			},
			CompletionProvider: &protocol.CompletionOptions{
				TriggerCharacters: []string{},
			},
			HoverProvider:          &protocol.HoverOptions{},
			DefinitionProvider:     &protocol.DefinitionOptions{},
			DocumentSymbolProvider: &protocol.DocumentSymbolOptions{},
		},
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    serverName,
			Version: &serverVersionStr,
		},
	}, nil
}

// initialized handles the LSP initialized notification.
func (s *Server) initialized(_ *glsp.Context, _ *protocol.InitializedParams) error {
	return nil
}

// shutdown handles the LSP shutdown request.
func (s *Server) shutdown(_ *glsp.Context) error {
	// Cancel all in-flight evaluations
	s.mu.RLock()
	for _, ds := range s.documents {
		if ds.cancel != nil {
			ds.cancel()
		}
	}
	s.mu.RUnlock()
	return nil
}

// setTrace handles the LSP $/setTrace notification.
func (s *Server) setTrace(_ *glsp.Context, _ *protocol.SetTraceParams) error {
	return nil
}

// getDocument returns the document state for the given URI.
func (s *Server) getDocument(uri string) *documentState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.documents[uri]
}

// evaluate parses and evaluates a document, producing an immutable snapshot.
// It wraps evaluation in a recover() to ensure the LSP server never crashes.
func (s *Server) evaluate(source string) *DocumentSnapshot {
	snap := &DocumentSnapshot{Source: source}

	defer func() {
		if r := recover(); r != nil {
			s.log.Errorf("panic during evaluation: %v", r)
			// Return partial snapshot with a diagnostic about the panic
			snap.Diagnostics = append(snap.Diagnostics, specDoc.Diagnostic{
				Severity: "Error",
				Code:     "internal_error",
				Message:  fmt.Sprintf("internal error: %v", r),
				DocLine:  1,
			})
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), evalTimeout)
	defer cancel()

	doc, err := specDoc.NewDocument(source)
	if err != nil {
		snap.Diagnostics = append(snap.Diagnostics, specDoc.Diagnostic{
			Severity: "Error",
			Code:     "parse_error",
			Message:  err.Error(),
			DocLine:  1,
		})
		return snap
	}
	snap.Document = doc

	eval := implDoc.NewEvaluator()

	// Check context before evaluation
	select {
	case <-ctx.Done():
		snap.Diagnostics = append(snap.Diagnostics, specDoc.Diagnostic{
			Severity: "Warning",
			Code:     "timeout",
			Message:  "evaluation timed out",
			DocLine:  1,
		})
		return snap
	default:
	}

	eval.Evaluate(doc)
	snap.Evaluator = eval

	// Collect diagnostics from all calc blocks
	for _, node := range doc.GetBlocks() {
		if cb, ok := node.Block.(*specDoc.CalcBlock); ok {
			snap.Diagnostics = append(snap.Diagnostics, cb.Diagnostics()...)
		}
	}

	return snap
}

func boolPtr(b bool) *bool { return &b }
