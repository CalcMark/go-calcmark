package editor

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
)

// TestEditingVariableShowsFalseRedefinition reproduces the user's bug:
// Editing "b = 5" to "b = 6" shows a redefinition error on "a = 3".
func TestEditingVariableShowsFalseRedefinition(t *testing.T) {
	// Initial document: two variables, no redefinition
	source := `a = 3

b = 5

# Hello`

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	evalErr := doc.Evaluate()
	if evalErr != nil {
		t.Fatalf("Initial document should have no errors, got: %v", evalErr)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Verify initial state has no errors
	results := m.GetLineResults()
	t.Logf("Initial state (%d lines):", len(results))
	for i, r := range results {
		if r.Error != "" {
			t.Logf("  Line %d: %q -> ERROR: %s", i, r.Source, r.Error)
		} else if r.IsCalc {
			t.Logf("  Line %d: %q -> %s", i, r.Source, r.Value)
		} else {
			t.Logf("  Line %d: %q (text)", i, r.Source)
		}
	}

	// Verify no errors initially
	for i, r := range results {
		if r.Error != "" {
			t.Errorf("Initial line %d should have no error, got: %s", i, r.Error)
		}
	}

	// Now simulate editing line 2 (b = 5) to (b = 6)
	// Steps:
	// 1. Navigate to line 2 (b = 5)
	// 2. Enter edit mode and modify the line
	// 3. Exit edit mode (which triggers re-evaluation)

	t.Log("\n=== Simulating edit: changing 'b = 5' to 'b = 6' ===")

	// Navigate to line 2
	m.cursorLine = 2
	m.loadCurrentLineIntoEditBuffer()
	t.Logf("Loaded line 2 into editBuf: %q", m.editBuf)

	// Set state to default (user is always editing - no modal editing like vim)
	m.mode = StateDefault
	m.userIsTyping = true

	// Simulate typing: change '5' to '6'
	// editBuf is "b = 5", we want "b = 6"
	m.editBuf = "b = 6"
	m.cursorCol = len(m.editBuf)

	t.Logf("Modified editBuf to: %q", m.editBuf)

	// Save the line (this triggers re-evaluation)
	m.saveCurrentLine(true)

	t.Logf("After save: mode=%d, cursorLine=%d", m.mode, m.cursorLine)

	// Get updated results
	results = m.GetLineResults()
	t.Logf("\nAfter edit (%d lines):", len(results))
	for i, r := range results {
		if r.Error != "" {
			t.Logf("  Line %d: %q -> ERROR: %s", i, r.Source, r.Error)
		} else if r.IsCalc {
			t.Logf("  Line %d: %q -> %s", i, r.Source, r.Value)
		} else {
			t.Logf("  Line %d: %q (text)", i, r.Source)
		}
	}

	// Verify: NO errors should appear
	// Line 0: a = 3 (no error)
	// Line 2: b = 6 (no error - this is NOT a redefinition!)

	if results[0].Error != "" {
		t.Errorf("Line 0 (a = 3) should have no error after editing b, got: %s", results[0].Error)
	}

	if len(results) > 2 && results[2].Error != "" {
		t.Errorf("Line 2 (b = 6) should have no error (not a redefinition), got: %s", results[2].Error)
	}

	// Check that b's value updated to 6
	if len(results) > 2 && results[2].IsCalc {
		if results[2].Value != "6" {
			t.Errorf("Line 2 (b = 6) should have value '6', got: %s", results[2].Value)
		}
	}

	// Most importantly: NO line should have any errors
	for i, r := range results {
		if r.Error != "" {
			t.Errorf("FAIL: Line %d should have no error, got: %s", i, r.Error)
		}
	}
}
