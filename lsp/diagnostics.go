package lsp

import (
	"fmt"
	"os"
	"strings"
	"time"

	specDoc "github.com/CalcMark/go-calcmark/spec/document"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func debugLog(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[calcmark-lsp] "+format+"\n", args...)
}

// textDocumentDidOpen handles the textDocument/didOpen notification.
func (s *Server) textDocumentDidOpen(ctx *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	uri := params.TextDocument.URI
	source := params.TextDocument.Text
	debugLog("didOpen: uri=%s len=%d", uri, len(source))

	// Enforce document size limit
	if len(source) > maxDocumentSize {
		s.publishSingleDiagnostic(ctx, uri, specDoc.Diagnostic{
			Severity: "Error",
			Code:     "document_too_large",
			Message:  "document exceeds 1 MB size limit",
			DocLine:  1,
		})
		return nil
	}

	snap := s.evaluate(source)
	debugLog("didOpen: evaluated, diagnostics=%d", len(snap.Diagnostics))

	ds := &documentState{}
	ds.setSource(source)
	ds.setSnapshot(snap)

	s.mu.Lock()
	s.documents[uri] = ds
	s.mu.Unlock()

	s.publishDiagnostics(ctx, uri, snap)
	debugLog("didOpen: published diagnostics")
	return nil
}

// textDocumentDidChange handles the textDocument/didChange notification.
func (s *Server) textDocumentDidChange(ctx *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	uri := params.TextDocument.URI

	ds := s.getDocument(uri)
	if ds == nil {
		return nil
	}

	// Full sync: take the last content change (should be the full document)
	if len(params.ContentChanges) == 0 {
		return nil
	}
	source := params.ContentChanges[len(params.ContentChanges)-1].(protocol.TextDocumentContentChangeEventWhole).Text

	// Enforce document size limit
	if len(source) > maxDocumentSize {
		return nil
	}

	// Store source immediately so read-only requests (completion, hover,
	// signature help) always see the latest text — even during debounce.
	ds.setSource(source)

	// Cancel any in-flight evaluation
	ds.mu.Lock()
	if ds.cancel != nil {
		ds.cancel()
	}

	// Debounce: reset timer for evaluation (parsing, diagnostics, variable state)
	if ds.timer != nil {
		ds.timer.Stop()
	}

	notifyCtx := ctx
	ds.timer = time.AfterFunc(debounceDelay, func() {
		snap := s.evaluate(source)
		ds.setSnapshot(snap)
		s.publishDiagnostics(notifyCtx, uri, snap)
	})
	ds.mu.Unlock()

	return nil
}

// textDocumentDidClose handles the textDocument/didClose notification.
func (s *Server) textDocumentDidClose(_ *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	uri := params.TextDocument.URI

	s.mu.Lock()
	if ds, ok := s.documents[uri]; ok {
		if ds.cancel != nil {
			ds.cancel()
		}
		if ds.timer != nil {
			ds.timer.Stop()
		}
		delete(s.documents, uri)
	}
	s.mu.Unlock()

	return nil
}

// publishDiagnostics sends diagnostics to the client for the given document.
func (s *Server) publishDiagnostics(ctx *glsp.Context, uri string, snap *DocumentSnapshot) {
	diags := make([]protocol.Diagnostic, 0, len(snap.Diagnostics))

	for _, d := range snap.Diagnostics {
		diags = append(diags, toLSPDiagnostic(d))
	}

	ctx.Notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diags,
	})
}

// publishSingleDiagnostic is a convenience to publish a single diagnostic without a full snapshot.
func (s *Server) publishSingleDiagnostic(ctx *glsp.Context, uri string, d specDoc.Diagnostic) {
	ctx.Notify(protocol.ServerTextDocumentPublishDiagnostics, protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: []protocol.Diagnostic{toLSPDiagnostic(d)},
	})
}

// toLSPDiagnostic converts a CalcMark diagnostic to an LSP diagnostic.
func toLSPDiagnostic(d specDoc.Diagnostic) protocol.Diagnostic {
	severity := toLSPSeverity(d.Severity)
	code := d.Code

	message := d.Message
	if d.Detailed != "" {
		message = d.Message + "\n" + d.Detailed
	}

	line := max(d.DocLine-1, 0)

	return protocol.Diagnostic{
		Range: protocol.Range{
			Start: protocol.Position{Line: protocol.UInteger(line), Character: 0},
			End:   protocol.Position{Line: protocol.UInteger(line), Character: 1000},
		},
		Severity: &severity,
		Code:     &protocol.IntegerOrString{Value: code},
		Source:   strPtr(serverName),
		Message:  message,
	}
}

// toLSPSeverity maps CalcMark severity strings to LSP DiagnosticSeverity.
func toLSPSeverity(severity string) protocol.DiagnosticSeverity {
	switch strings.ToLower(severity) {
	case "error":
		return protocol.DiagnosticSeverityError
	case "warning":
		return protocol.DiagnosticSeverityWarning
	case "hint":
		return protocol.DiagnosticSeverityHint
	default:
		return protocol.DiagnosticSeverityInformation
	}
}

func strPtr(s string) *string { return &s }
