package lsp

import (
	"encoding/json"
	"strings"
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// hoverResult drives textDocument/hover through the full extended path
// (including the interceptingHandler) and returns the *lspHover wire shape.
func hoverResult(t *testing.T, source string, line, col uint32) *lspHover {
	t.Helper()
	s := NewServer()
	ds := &documentState{}
	ds.setSource(source)
	snap := s.evaluate(source)
	ds.setSnapshot(snap)

	s.mu.Lock()
	s.documents["test://doc.cm"] = ds
	s.mu.Unlock()

	r, err := s.hoverHandle(&protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "test://doc.cm"},
			Position:     protocol.Position{Line: line, Character: col},
		},
	})
	if err != nil {
		t.Fatalf("hover error: %v", err)
	}
	if r == nil {
		return nil
	}
	h, ok := r.(*lspHover)
	if !ok {
		t.Fatalf("hover result is not *lspHover: %T", r)
	}
	return h
}

// hoverContent returns just the markdown body text (empty string when nil).
func hoverContent(t *testing.T, source string, line, col uint32) string {
	t.Helper()
	h := hoverResult(t, source, line, col)
	if h == nil {
		return ""
	}
	mc, ok := h.Contents.(protocol.MarkupContent)
	if !ok {
		t.Fatalf("hover contents not MarkupContent: %T", h.Contents)
	}
	return mc.Value
}

// TestHover_FunctionGrow asserts R5: hovering on grow in
// `goal = grow(100, 20, 5)` returns the signature, description, and
// at least one example.
func TestHover_FunctionGrow(t *testing.T) {
	// Cursor on 'g' of grow (col 7)
	content := hoverContent(t, "goal = grow(100, 20, 5)", 0, 7)
	if content == "" {
		t.Fatal("expected hover on grow, got nil")
	}
	wantSubstrings := []string{"grow(", "grow"}
	for _, want := range wantSubstrings {
		if !strings.Contains(content, want) {
			t.Errorf("hover content missing %q\nfull: %s", want, content)
		}
	}
	// Must include at least one example — any example text from the spec
	// ('100', '20 GB', '1000 users', etc.) is acceptable.
	if !strings.Contains(content, "Example") && !strings.Contains(content, "example") {
		t.Errorf("hover content has no 'Example' section:\n%s", content)
	}
}

// TestHover_VariableWithType asserts R6: hovering on price in a doc with
// `price = 100` returns markdown containing "number" and "100".
func TestHover_VariableWithType(t *testing.T) {
	content := hoverContent(t, "price = 100", 0, 0)
	if content == "" {
		t.Fatal("expected hover on price, got nil")
	}
	if !strings.Contains(content, "number") {
		t.Errorf("hover content missing 'number':\n%s", content)
	}
	if !strings.Contains(content, "100") {
		t.Errorf("hover content missing '100':\n%s", content)
	}
}

// TestHover_VariablePercentage asserts percentage type is shown.
func TestHover_VariablePercentage(t *testing.T) {
	content := hoverContent(t, "tax_rate = 8%", 0, 2)
	if content == "" {
		t.Fatal("expected hover on tax_rate")
	}
	if !strings.Contains(content, "percentage") {
		t.Errorf("hover content missing 'percentage':\n%s", content)
	}
	if !strings.Contains(content, "8%") {
		t.Errorf("hover content missing '8%%':\n%s", content)
	}
}

// TestHover_VariableRate asserts rate type is shown.
func TestHover_VariableRate(t *testing.T) {
	content := hoverContent(t, "bandwidth = 10 MB/s", 0, 3)
	if content == "" {
		t.Fatal("expected hover on bandwidth")
	}
	if !strings.Contains(content, "rate") {
		t.Errorf("hover content missing 'rate':\n%s", content)
	}
}

// TestHover_VariableData asserts the structured data field on a variable hover.
func TestHover_VariableData(t *testing.T) {
	h := hoverResult(t, "price = 100", 0, 0)
	if h == nil {
		t.Fatal("expected hover on price")
	}
	d, ok := h.Data.(hoverData)
	if !ok {
		t.Fatalf("h.Data is not hoverData: %T", h.Data)
	}
	if d.Kind != "variable" {
		t.Errorf("kind = %q, want variable", d.Kind)
	}
	if d.Name != "price" {
		t.Errorf("name = %q, want price", d.Name)
	}
	if d.VariableType != "number" {
		t.Errorf("variableType = %q, want number", d.VariableType)
	}
	if d.Value == "" {
		t.Error("value is empty")
	}
}

// TestHover_FunctionData asserts the structured data field on a function hover.
func TestHover_FunctionData(t *testing.T) {
	h := hoverResult(t, "x = accumulate(10 MB/s, 1 hour)", 0, 4)
	if h == nil {
		t.Fatal("expected hover on accumulate")
	}
	d, ok := h.Data.(hoverData)
	if !ok {
		t.Fatalf("h.Data is not hoverData: %T", h.Data)
	}
	if d.Kind != "function" {
		t.Errorf("kind = %q, want function", d.Kind)
	}
	if d.Name != "accumulate" {
		t.Errorf("name = %q, want accumulate", d.Name)
	}
	if len(d.Params) != 2 {
		t.Errorf("expected 2 params, got %d", len(d.Params))
	}
	if len(d.Params) >= 1 && d.Params[0].Type != "rate" {
		t.Errorf("param[0].type = %q, want rate", d.Params[0].Type)
	}
}

// TestHover_WireJSONShape asserts the final JSON emitted by MarshalJSON on
// lspHover includes both contents (standard) and data (extension).
func TestHover_WireJSONShape(t *testing.T) {
	h := hoverResult(t, "price = 100", 0, 0)
	if h == nil {
		t.Fatal("expected hover")
	}
	b, err := json.Marshal(h)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(b)
	for _, want := range []string{
		`"contents"`,
		`"data"`,
		`"kind":"variable"`,
		`"variableType":"number"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("wire JSON missing %q\n%s", want, s)
		}
	}
}

// TestHover_FunctionParamsListed asserts the hover for a function includes
// the parameter names and types from its spec.
func TestHover_FunctionParamsListed(t *testing.T) {
	// Cursor on 'a' of accumulate (col 4)
	content := hoverContent(t, "x = accumulate(10 MB/s, 1 hour)", 0, 4)
	if content == "" {
		t.Fatal("expected hover on accumulate")
	}
	if !strings.Contains(content, "rate") {
		t.Errorf("hover content missing 'rate' param type:\n%s", content)
	}
	if !strings.Contains(content, "duration") {
		t.Errorf("hover content missing 'duration' param type:\n%s", content)
	}
}
