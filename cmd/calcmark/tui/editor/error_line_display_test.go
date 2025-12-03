package editor

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
)

// TestErrorLineDisplay tests that errors appear on the correct line in GetLineResults.
// This reproduces the user's bug where changing "b = 5" to "b = 6" shows an error on "a = 3".
func TestErrorLineDisplay(t *testing.T) {
	// Exact scenario from user's screenshot
	source := `a = 3

b = 6

# Hello`

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	evalErr := doc.Evaluate()
	if evalErr != nil {
		t.Logf("Evaluate error: %v", evalErr)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	results := m.GetLineResults()

	t.Logf("Got %d line results:", len(results))
	for i, r := range results {
		if r.Error != "" {
			t.Logf("  Line %d: %q -> ERROR: %s", i, r.Source, r.Error)
		} else {
			t.Logf("  Line %d: %q -> %s", i, r.Source, r.Value)
		}
	}

	// There should be NO errors in this document
	// Line 0: a = 3 (valid)
	// Line 1: (empty)
	// Line 2: b = 6 (valid, NOT a redefinition)
	// Line 3: (empty)
	// Line 4: # Hello (markdown)

	if evalErr != nil {
		t.Errorf("Document should evaluate without errors, got: %v", evalErr)
	}

	// Check that line 0 (a = 3) has no error
	if results[0].Error != "" {
		t.Errorf("Line 0 (a = 3) should have no error, got: %s", results[0].Error)
	}

	// Check that line 2 (b = 6) has no error
	if len(results) > 2 && results[2].Error != "" {
		t.Errorf("Line 2 (b = 6) should have no error, got: %s", results[2].Error)
	}

	// Check all calc lines have no errors
	for i, r := range results {
		if r.IsCalc && r.Error != "" {
			t.Errorf("Line %d should have no error, got: %s", i, r.Error)
		}
	}
}

// TestSingleAssignmentPerVariable tests that a single assignment of each variable is valid.
func TestSingleAssignmentPerVariable(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name:   "Two different variables",
			source: "a = 3\n\nb = 6",
		},
		{
			name:   "Three different variables",
			source: "a = 1\n\nb = 2\n\nc = 3",
		},
		{
			name:   "Variable used in expression",
			source: "a = 3\n\nb = a * 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := document.NewDocument(tt.source)
			if err != nil {
				t.Fatalf("NewDocument failed: %v", err)
			}

			evalErr := doc.Evaluate()
			if evalErr != nil {
				t.Errorf("Document should evaluate without errors, got: %v", evalErr)
			}

			m := New(doc)
			results := m.GetLineResults()

			// Check no calc lines have errors
			for i, r := range results {
				if r.IsCalc && r.Error != "" {
					t.Errorf("Line %d (%q) should have no error, got: %s", i, r.Source, r.Error)
				}
			}
		})
	}
}
