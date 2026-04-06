package editor

import (
	"slices"
	"strings"
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

// TestCascadingErrorDiagnosticSetsIsBlocked verifies that a cascading_error
// diagnostic code from the evaluator causes IsBlocked=true on the affected line.
func TestCascadingErrorDiagnosticSetsIsBlocked(t *testing.T) {
	// Block 1: a = 1/0 (root cause error)
	// Block 2: b = a * 2 (cascading error - depends on errored "a")
	source := "a = 1 / 0\n\nSome text\n\nb = a * 2\n"
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}
	m := New(doc)
	results := m.GetLineResults()

	// Find the line for "b = a * 2"
	var bResult *LineResult
	for i := range results {
		if strings.Contains(results[i].Source, "b = a * 2") {
			bResult = &results[i]
			break
		}
	}
	if bResult == nil {
		t.Fatal("could not find result for 'b = a * 2'")
	}

	if !bResult.IsBlocked {
		t.Errorf("expected IsBlocked=true for cascading error on 'b = a * 2', got false; error=%q", bResult.Error)
	}

	// Find the line for "a = 1 / 0" - root cause should NOT be blocked
	var aResult *LineResult
	for i := range results {
		if strings.Contains(results[i].Source, "a = 1 / 0") {
			aResult = &results[i]
			break
		}
	}
	if aResult == nil {
		t.Fatal("could not find result for 'a = 1 / 0'")
	}
	if aResult.IsBlocked {
		t.Error("expected IsBlocked=false for root-cause error on 'a = 1 / 0'")
	}
}

// TestCascadingErrorWithinSameBlock verifies cascading detection within a single block.
func TestCascadingErrorWithinSameBlock(t *testing.T) {
	// Both statements in one block: a = 1/0 fails, b = a*2 cascading
	source := "a = 1 / 0\nb = a * 2\n"
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}
	m := New(doc)
	results := m.GetLineResults()

	var aResult, bResult *LineResult
	for i := range results {
		switch {
		case strings.Contains(results[i].Source, "a = 1 / 0"):
			aResult = &results[i]
		case strings.Contains(results[i].Source, "b = a * 2"):
			bResult = &results[i]
		}
	}

	if aResult == nil || bResult == nil {
		t.Fatalf("could not find both results; a=%v b=%v", aResult, bResult)
	}

	if aResult.IsBlocked {
		t.Error("root-cause 'a = 1 / 0' should NOT be blocked")
	}
	if !bResult.IsBlocked {
		t.Errorf("cascading 'b = a * 2' should be blocked; error=%q", bResult.Error)
	}
}

// TestCascadingErrorNilResultPlaceholder verifies that nil result placeholders
// (for failed statements) are handled gracefully without panics.
func TestCascadingErrorNilResultPlaceholder(t *testing.T) {
	// a = 1/0 produces a nil result placeholder; b = a * 2 also nil
	source := "a = 1 / 0\nb = a * 2\n"
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}
	m := New(doc)
	// Should not panic
	results := m.GetLineResults()

	// Both lines should have errors (not blank values)
	for _, r := range results {
		if r.IsCalc && strings.TrimSpace(r.Source) != "" {
			if r.Error == "" && r.Value == "" {
				t.Errorf("line %q has neither error nor value", r.Source)
			}
		}
	}
}

// TestMultipleCascadingErrorsAcrossBlocks verifies that cascading errors
// from multiple blocks all show as blocked.
func TestMultipleCascadingErrorsAcrossBlocks(t *testing.T) {
	// Block 1: a = 1/0
	// Block 2: b = a * 2
	// Block 3: c = a + 1
	source := "a = 1 / 0\n\ntext\n\nb = a * 2\n\ntext\n\nc = a + 1\n"
	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}
	m := New(doc)
	results := m.GetLineResults()

	blocked := 0
	for _, r := range results {
		if r.IsBlocked {
			blocked++
		}
	}

	if blocked < 2 {
		t.Errorf("expected at least 2 blocked lines (b and c), got %d", blocked)
		for _, r := range results {
			if r.IsCalc && r.Error != "" {
				t.Logf("  line=%q error=%q blocked=%v", r.Source, r.Error, r.IsBlocked)
			}
		}
	}
}
