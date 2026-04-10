package lsp

import (
	"strings"
	"testing"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// hoverContent calls textDocumentHover with a freshly-loaded document and
// returns the hover body text (empty string when hover is nil).
func hoverContent(t *testing.T, source string, line, col uint32) string {
	t.Helper()
	s := NewServer()
	ds := &documentState{}
	ds.setSource(source)
	snap := s.evaluate(source)
	ds.setSnapshot(snap)

	s.mu.Lock()
	s.documents["test://doc.cm"] = ds
	s.mu.Unlock()

	h, err := s.textDocumentHover(nil, &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: "test://doc.cm"},
			Position:     protocol.Position{Line: line, Character: col},
		},
	})
	if err != nil {
		t.Fatalf("hover error: %v", err)
	}
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
