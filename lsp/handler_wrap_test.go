package lsp

import (
	"encoding/json"
	"testing"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// newServerWithDoc loads a source document into a fresh server and returns
// both the wrapping handler and the URI so interceptor tests can drive the
// full JSON-RPC dispatch path.
func newServerWithDoc(t *testing.T, source string) (*interceptingHandler, string) {
	t.Helper()
	s := NewServer()
	ds := &documentState{}
	ds.setSource(source)
	ds.setSnapshot(s.evaluate(source))

	uri := "test://wrap.cm"
	s.mu.Lock()
	s.documents[uri] = ds
	s.mu.Unlock()

	return &interceptingHandler{inner: &s.handler, server: s}, uri
}

func mustMarshal(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return json.RawMessage(b)
}

// TestInterceptingHandler_HoverDispatch proves that the full glsp-level
// dispatch path reaches hoverHandle, not the no-op stub.
func TestInterceptingHandler_HoverDispatch(t *testing.T) {
	h, uri := newServerWithDoc(t, "price = 100")

	params := protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 0},
		},
	}
	ctx := &glsp.Context{
		Method: protocol.MethodTextDocumentHover,
		Params: mustMarshal(t, params),
	}
	r, validMethod, validParams, err := h.Handle(ctx)
	if err != nil {
		t.Fatalf("dispatch error: %v", err)
	}
	if !validMethod {
		t.Error("validMethod = false, want true for MethodTextDocumentHover")
	}
	if !validParams {
		t.Error("validParams = false, want true after successful unmarshal")
	}
	hover, ok := r.(*lspHover)
	if !ok {
		t.Fatalf("result is not *lspHover: %T", r)
	}
	if hover == nil {
		t.Fatal("hover result is nil")
	}
	d, ok := hover.Data.(hoverData)
	if !ok {
		t.Fatalf("hover.Data is not hoverData: %T", hover.Data)
	}
	if d.Kind != "variable" || d.VariableType != "number" {
		t.Errorf("dispatched hover produced wrong data: %+v", d)
	}
}

// TestInterceptingHandler_SignatureHelpDispatch proves signatureHelp reaches
// the extended handler and returns the wrapped lspSignatureHelp shape.
func TestInterceptingHandler_SignatureHelpDispatch(t *testing.T) {
	h, uri := newServerWithDoc(t, "x = accumulate(10 MB/s, 1 hour)")

	params := protocol.SignatureHelpParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 0, Character: 15},
		},
	}
	ctx := &glsp.Context{
		Method: protocol.MethodTextDocumentSignatureHelp,
		Params: mustMarshal(t, params),
	}
	r, validMethod, validParams, err := h.Handle(ctx)
	if err != nil {
		t.Fatalf("dispatch error: %v", err)
	}
	if !validMethod || !validParams {
		t.Errorf("validMethod=%v validParams=%v, want both true", validMethod, validParams)
	}
	help, ok := r.(*lspSignatureHelp)
	if !ok {
		t.Fatalf("result is not *lspSignatureHelp: %T", r)
	}
	if help == nil || len(help.Signatures) == 0 {
		t.Fatal("signature help is nil or empty")
	}
	// The extended wire shape must carry wireParamData on each parameter.
	if _, ok := help.Signatures[0].Parameters[0].Data.(wireParamData); !ok {
		t.Errorf("parameter[0].Data is not wireParamData: %T",
			help.Signatures[0].Parameters[0].Data)
	}
}

// TestInterceptingHandler_FalthroughUnrecognizedMethod proves non-intercepted
// methods fall through to the inner glsp handler without panicking. The
// inner handler may signal "server not initialized" until Initialize has
// been called — what matters here is that the intercept does not crash
// and that the switch in interceptingHandler.Handle does not swallow the
// method as one of its own cases.
func TestInterceptingHandler_FalthroughUnrecognizedMethod(t *testing.T) {
	h, _ := newServerWithDoc(t, "x = 1")
	ctx := &glsp.Context{
		Method: "textDocument/definitionNotReal",
		Params: json.RawMessage(`{}`),
	}
	// The fallthrough must not panic. We don't assert specific validMethod
	// values because the inner glsp handler's own not-initialized guard
	// responds before per-method case matching.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("intercept fallthrough panicked: %v", r)
		}
	}()
	_, _, _, _ = h.Handle(ctx)
}

// TestInterceptingHandler_HoverUnmarshalError proves the error path when
// params JSON is malformed.
func TestInterceptingHandler_HoverUnmarshalError(t *testing.T) {
	h, _ := newServerWithDoc(t, "")
	ctx := &glsp.Context{
		Method: protocol.MethodTextDocumentHover,
		Params: json.RawMessage(`{"textDocument": "not-an-object"}`),
	}
	_, validMethod, validParams, err := h.Handle(ctx)
	if !validMethod {
		t.Error("validMethod should remain true for a recognized method even on param error")
	}
	if validParams {
		t.Error("validParams should be false after an unmarshal error")
	}
	if err == nil {
		t.Error("expected unmarshal error")
	}
}
