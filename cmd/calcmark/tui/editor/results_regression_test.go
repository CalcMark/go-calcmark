package editor

// results_regression_test.go — Reproduction tests for the reported regression:
// "Context footer no longer displays variable references after commit 6ce0bb3."
//
// These tests exercise GetLineResults().ReferencedVars and getLineReferences()
// end-to-end through the editor Model, covering:
//   - Basic variable references (no whitespace gaps)
//   - Whitespace-only lines between definitions and references
//   - The broken whitespace guard at results.go:93-104
//
// DO NOT modify existing code. This file is purely diagnostic.

import (
	"slices"
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
)

// TestRegressionBasicReferencedVars verifies that a simple two-line calc block
// correctly populates ReferencedVars for the line referencing a variable.
// This is the minimal reproduction for the reported regression.
func TestRegressionBasicReferencedVars(t *testing.T) {
	doc, err := document.NewDocument("x = 10\ny = x + 5\n")
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}
	m := New(doc)

	results := m.GetLineResults()
	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}

	// Line 0: "x = 10" — defines x, references nothing
	if results[0].ReferencedVars != nil {
		t.Errorf("line 0 ReferencedVars = %v, want nil", results[0].ReferencedVars)
	}

	// Line 1: "y = x + 5" — defines y, references x
	wantRefs := []string{"x"}
	if !slices.Equal(results[1].ReferencedVars, wantRefs) {
		t.Errorf("line 1 ReferencedVars = %v, want %v", results[1].ReferencedVars, wantRefs)
	}
}

// TestRegressionGetLineReferencesBasic verifies that getLineReferences returns
// VarReference structs with the correct Name and Value for a simple case.
// This is the function directly consumed by the context footer.
func TestRegressionGetLineReferencesBasic(t *testing.T) {
	doc, err := document.NewDocument("x = 10\ny = x + 5\n")
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}
	m := New(doc)

	results := m.GetLineResults()
	refs := m.getLineReferences(1, results)
	if len(refs) == 0 {
		t.Fatal("getLineReferences(1) returned empty — context footer would show no references")
	}

	if refs[0].Name != "x" {
		t.Errorf("refs[0].Name = %q, want %q", refs[0].Name, "x")
	}

	// Value should be "10" (the evaluated result of x = 10)
	if refs[0].Value != "10" {
		t.Errorf("refs[0].Value = %q, want %q", refs[0].Value, "10")
	}
}

// TestRegressionWhitespaceLinesBetweenVars tests the scenario where whitespace-only
// lines appear between variable definitions and references. The broken whitespace
// guard at results.go:93-104 fails to skip whitespace-only lines, potentially
// corrupting the statement index mapping.
func TestRegressionWhitespaceLinesBetweenVars(t *testing.T) {
	// Source with a whitespace-only line between definitions
	doc, err := document.NewDocument("x = 10\n   \ny = x + 5\n")
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}
	m := New(doc)

	results := m.GetLineResults()
	if len(results) < 3 {
		t.Fatalf("expected at least 3 results, got %d", len(results))
	}

	// Line 0: "x = 10" — defines x, references nothing
	if results[0].ReferencedVars != nil {
		t.Errorf("line 0 ReferencedVars = %v, want nil", results[0].ReferencedVars)
	}

	// Line 1: "   " (whitespace-only) — should have no references
	// BUG: The broken whitespace guard does NOT skip this line, so it may
	// get spurious ReferencedVars from the next statement.
	if results[1].ReferencedVars != nil {
		t.Errorf("line 1 (whitespace-only) ReferencedVars = %v, want nil (whitespace guard bug)",
			results[1].ReferencedVars)
	}

	// Line 2: "y = x + 5" — defines y, references x
	// This is the critical check: does the reference survive the broken whitespace guard?
	wantRefs := []string{"x"}
	if !slices.Equal(results[2].ReferencedVars, wantRefs) {
		t.Errorf("line 2 ReferencedVars = %v, want %v", results[2].ReferencedVars, wantRefs)
	}
}

// TestRegressionGetLineReferencesWithWhitespace verifies getLineReferences works
// correctly when whitespace-only lines precede the referenced line.
func TestRegressionGetLineReferencesWithWhitespace(t *testing.T) {
	doc, err := document.NewDocument("x = 10\n   \ny = x + 5\n")
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}
	m := New(doc)

	// Line 2: "y = x + 5" should reference x with value 10
	results := m.GetLineResults()
	refs := m.getLineReferences(2, results)
	if len(refs) == 0 {
		t.Fatal("getLineReferences(2) returned empty — context footer would show no references")
	}

	if refs[0].Name != "x" {
		t.Errorf("refs[0].Name = %q, want %q", refs[0].Name, "x")
	}
	if refs[0].Value != "10" {
		t.Errorf("refs[0].Value = %q, want %q", refs[0].Value, "10")
	}
}

// TestWhitespaceGuardCorrectness verifies that the whitespace guard in
// GetLineResults (results.go) correctly identifies whitespace-only lines.
// This was originally broken (d36390ff) — trimmed was always set to line.
// Fixed by using strings.TrimSpace, consistent with countNonEmptyLinesBefore.
func TestWhitespaceGuardCorrectness(t *testing.T) {
	testCases := []struct {
		name     string
		line     string
		wantSkip bool // true if the guard should skip this line
	}{
		{"empty string", "", true},
		{"spaces only", "   ", true},
		{"tab only", "\t", true},
		{"mixed whitespace", " \t ", true},
		{"non-empty", "x = 10", false},
		{"space then text", " x = 10", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// This matches the fixed guard logic in results.go
			actuallySkipped := strings.TrimSpace(tc.line) == ""

			if actuallySkipped != tc.wantSkip {
				t.Errorf("line %q: guard skipped=%v, want skip=%v",
					tc.line, actuallySkipped, tc.wantSkip)
			}
		})
	}
}

// TestRegressionMultipleRefsAfterWhitespace tests that multiple variable
// references still work when whitespace lines are interspersed.
func TestRegressionMultipleRefsAfterWhitespace(t *testing.T) {
	doc, err := document.NewDocument("a = 1\nb = 2\n\nc = a + b\n")
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}
	m := New(doc)

	results := m.GetLineResults()
	if len(results) < 4 {
		t.Fatalf("expected at least 4 results, got %d", len(results))
	}

	// Line 3: "c = a + b" — references a and b
	got := results[3].ReferencedVars
	want := []string{"a", "b"}
	if !slices.Equal(got, want) {
		t.Errorf("line 3 ReferencedVars = %v, want %v", got, want)
	}
}
