package editor

import (
	"slices"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
)

// TestLineResultReferencedVars verifies that GetLineResults populates
// ReferencedVars from the AST rather than text matching.
// This is the core regression test for issue #10 (E matched inside EUR).
func TestLineResultReferencedVars(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		lineIdx  int // which line to check (0-indexed)
		wantRefs []string
	}{
		{
			name:     "assignment referencing another variable",
			source:   "x = 10\ny = x + 5\n",
			lineIdx:  1,
			wantRefs: []string{"x"},
		},
		{
			name:     "anonymous expression with no variables",
			source:   "2 + 2\n",
			lineIdx:  0,
			wantRefs: nil,
		},
		{
			name:     "currency literal not mistaken for variable E",
			source:   "1000 EUR - 250 EUR\n",
			lineIdx:  0,
			wantRefs: nil,
		},
		{
			name:     "assignment line defines var, no refs on RHS literal",
			source:   "x = 42\n",
			lineIdx:  0,
			wantRefs: nil,
		},
		{
			name:     "multiple refs sorted",
			source:   "a = 1\nb = 2\nc = b + a\n",
			lineIdx:  2,
			wantRefs: []string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := document.NewDocument(tt.source)
			if err != nil {
				t.Fatalf("NewDocument(%q): %v", tt.source, err)
			}
			m := New(doc)
			results := m.GetLineResults()

			if tt.lineIdx >= len(results) {
				t.Fatalf("lineIdx %d out of range (got %d results)", tt.lineIdx, len(results))
			}

			got := results[tt.lineIdx].ReferencedVars
			if !slices.Equal(got, tt.wantRefs) {
				t.Errorf("ReferencedVars = %v, want %v", got, tt.wantRefs)
			}
		})
	}
}

// TestReferencedVarsWithBuiltinConstants is the end-to-end regression test
// for issue #10. The built-in constant E exists in the environment, but
// "1000 EUR" is a CurrencyLiteral in the AST — E must not appear as a
// referenced variable.
func TestReferencedVarsWithBuiltinConstants(t *testing.T) {
	doc, err := document.NewDocument("budget = 1000 EUR - 250 EUR\n")
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}
	m := New(doc)

	// Confirm E is actually in the environment (the precondition for the bug)
	env := m.eval.GetEnvironment()
	allVars := env.GetAllVariables()
	if _, hasE := allVars["E"]; !hasE {
		t.Fatal("precondition: expected built-in constant E in environment")
	}

	results := m.GetLineResults()
	if len(results) == 0 {
		t.Fatal("expected at least 1 result")
	}

	refs := results[0].ReferencedVars
	for _, ref := range refs {
		if ref == "E" {
			t.Errorf("ReferencedVars contains 'E' — currency literal EUR was misidentified as variable reference")
		}
	}
	if len(refs) != 0 {
		t.Errorf("expected no references for pure currency arithmetic, got %v", refs)
	}
}

// TestIssue10_EURNotMatchedAsE is the exact regression scenario from issue #10.
// Built-in constant E (Euler's number) exists in the environment, but
// "1000 EUR - 250 EUR" must not report E as a referenced variable because
// EUR is a currency literal token, not an identifier.
func TestIssue10_EURNotMatchedAsE(t *testing.T) {
	doc, err := document.NewDocument("budget = 1000 EUR - 250 EUR\n")
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}
	m := New(doc)

	// Confirm E is in the environment (the precondition that caused the bug)
	env := m.eval.GetEnvironment()
	allVars := env.GetAllVariables()
	if _, hasE := allVars["E"]; !hasE {
		t.Fatal("precondition: built-in constant E should be in the environment")
	}

	results := m.GetLineResults()
	if len(results) == 0 {
		t.Fatal("expected at least 1 result")
	}

	refs := results[0].ReferencedVars
	if len(refs) != 0 {
		t.Errorf("expected no variable references for currency arithmetic, got %v", refs)
	}
}
