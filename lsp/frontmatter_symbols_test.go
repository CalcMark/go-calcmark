package lsp

import (
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/ast"
	specDoc "github.com/CalcMark/go-calcmark/v2/spec/document"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// mkRange is a small local helper so tests read as specs, not arithmetic.
func mkRange(startLine, startCol, endLine, endCol int) ast.Range {
	return ast.Range{
		Start: ast.Position{Line: startLine, Column: startCol},
		End:   ast.Position{Line: endLine, Column: endCol},
	}
}

func TestBuildFrontmatterSymbols_HappyRegisteredKeys(t *testing.T) {
	fm := specDoc.Frontmatter{
		KeyRanges: map[string]ast.Range{
			"convert_to": mkRange(2, 1, 2, 12),
			"globals":    mkRange(3, 1, 5, 20),
		},
	}
	syms := buildFrontmatterSymbols(fm)
	if len(syms) != 2 {
		t.Fatalf("expected 2 symbols, got %d", len(syms))
	}
	// Source-order: convert_to on line 2 comes before globals on line 3.
	if syms[0].Name != "convert_to" {
		t.Errorf("expected first symbol convert_to, got %q", syms[0].Name)
	}
	if syms[1].Name != "globals" {
		t.Errorf("expected second symbol globals, got %q", syms[1].Name)
	}
	if syms[0].Kind != protocol.SymbolKindProperty {
		t.Errorf("expected Kind Property (7), got %v", syms[0].Kind)
	}
	// 0-indexed LSP conversion: line 2 -> 1.
	if syms[0].Range.Start.Line != 1 {
		t.Errorf("expected convert_to Range.Start.Line=1, got %d", syms[0].Range.Start.Line)
	}
}

func TestBuildFrontmatterSymbols_ExtraOnly(t *testing.T) {
	fm := specDoc.Frontmatter{
		KeyRanges: map[string]ast.Range{
			"title": mkRange(2, 1, 2, 12),
		},
	}
	syms := buildFrontmatterSymbols(fm)
	if len(syms) != 0 {
		t.Errorf("expected no symbols for Extra-only frontmatter, got %d: %+v", len(syms), syms)
	}
}

func TestBuildFrontmatterSymbols_SortedBySourceLine(t *testing.T) {
	// Map iteration order is randomized; the helper must emit in line order.
	fm := specDoc.Frontmatter{
		KeyRanges: map[string]ast.Range{
			"globals":    mkRange(5, 1, 7, 5),
			"convert_to": mkRange(2, 1, 2, 12),
			"exchange":   mkRange(3, 1, 4, 15),
		},
	}
	syms := buildFrontmatterSymbols(fm)
	if len(syms) != 3 {
		t.Fatalf("expected 3 symbols, got %d", len(syms))
	}
	want := []string{"convert_to", "exchange", "globals"}
	for i, name := range want {
		if syms[i].Name != name {
			t.Errorf("position %d: want %q, got %q", i, name, syms[i].Name)
		}
	}
}

func TestBuildFrontmatterSymbols_EmptyFrontmatter(t *testing.T) {
	syms := buildFrontmatterSymbols(specDoc.Frontmatter{})
	if len(syms) != 0 {
		t.Errorf("expected empty slice for empty frontmatter, got %d symbols", len(syms))
	}
}

func TestBuildFrontmatterSymbols_RegisteredKeyPresentEvenIfTypedFieldsAbsent(t *testing.T) {
	// Only KeyRanges has "convert_to"; typed field ConvertTo is nil. The key
	// physically appeared in the source, so we still emit a symbol for it.
	fm := specDoc.Frontmatter{
		KeyRanges: map[string]ast.Range{
			"convert_to": mkRange(2, 1, 2, 12),
		},
	}
	syms := buildFrontmatterSymbols(fm)
	if len(syms) != 1 || syms[0].Name != "convert_to" {
		t.Fatalf("expected one convert_to symbol, got %+v", syms)
	}
}

func TestBuildFrontmatterSymbols_MixedRegisteredAndExtra(t *testing.T) {
	fm := specDoc.Frontmatter{
		KeyRanges: map[string]ast.Range{
			"title":      mkRange(2, 1, 2, 12),
			"convert_to": mkRange(3, 1, 3, 12),
			"author":     mkRange(4, 1, 4, 12),
		},
	}
	syms := buildFrontmatterSymbols(fm)
	if len(syms) != 1 || syms[0].Name != "convert_to" {
		t.Fatalf("expected only convert_to, got %+v", syms)
	}
}

func TestBuildFrontmatterSymbols_SelectionRangeIsKeyOnly(t *testing.T) {
	// For a key "convert_to" on line 2 col 1, SelectionRange should cover the
	// identifier itself (same line, col 1 .. col 1+len("convert_to")).
	fm := specDoc.Frontmatter{
		KeyRanges: map[string]ast.Range{
			"convert_to": mkRange(2, 1, 3, 5),
		},
	}
	syms := buildFrontmatterSymbols(fm)
	if len(syms) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(syms))
	}
	sel := syms[0].SelectionRange
	if sel.Start.Line != 1 || sel.End.Line != 1 {
		t.Errorf("SelectionRange should stay on key line (line 1 in LSP 0-indexed), got %+v", sel)
	}
	// convert_to is 10 chars, start col 1 → LSP char 0; end char 10.
	if sel.Start.Character != 0 || sel.End.Character != 10 {
		t.Errorf("SelectionRange columns want 0..10, got %d..%d", sel.Start.Character, sel.End.Character)
	}
}
