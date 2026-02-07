package editor

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
)

// TestInsertLineAtStartPreservesContent verifies that pressing Enter at the
// start of a line (col 0) correctly preserves the line content.
//
// Bug: When cursor is at col 0 and Enter is pressed, the line content
// should move to the new line below. But the content was being set in
// editBuf without calling updateCurrentLine(), causing the document to
// have an empty line and triggering false "undefined variable" errors.
func TestInsertLineAtStartPreservesContent(t *testing.T) {
	source := `# Budget

## Income
total_income = $5000

## Savings Target

savings_rate = 0.20
savings = total_income * savings_rate
`

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	m := New(doc)
	m.reEvaluate()

	// Find the line with savings_rate
	lines := m.GetLines()
	var savingsRateLine int = -1
	for i, line := range lines {
		if strings.HasPrefix(line, "savings_rate") {
			savingsRateLine = i
			break
		}
	}

	if savingsRateLine == -1 {
		t.Fatal("Could not find savings_rate line")
	}

	t.Logf("savings_rate is on line %d", savingsRateLine)

	// Position cursor at start of savings_rate line (col 0)
	m.cursorLine = savingsRateLine
	m.cursorCol = 0
	m.editBuf = ""

	// Simulate pressing Enter at col 0
	// This should:
	// 1. Insert a blank line at the current position
	// 2. Move "savings_rate = 0.20" to the next line
	// 3. NOT cause any evaluation errors
	newModel, _ := m.handleEnterKey()
	m = newModel.(Model)

	// Get updated lines
	newLines := m.GetLines()
	t.Logf("After Enter, document has %d lines", len(newLines))
	for i, line := range newLines {
		t.Logf("  Line %d: %q", i, line)
	}

	// The line that was savings_rate should now be empty (the new line)
	// and savings_rate should be on the line AFTER
	if newLines[savingsRateLine] != "" {
		t.Errorf("Line %d should be empty after Enter at col 0, got: %q",
			savingsRateLine, newLines[savingsRateLine])
	}

	// The next line should have savings_rate = 0.20
	if !strings.HasPrefix(newLines[savingsRateLine+1], "savings_rate") {
		t.Errorf("Line %d should have savings_rate content, got: %q",
			savingsRateLine+1, newLines[savingsRateLine+1])
	}

	// Most importantly: the line after that (savings = ...) should NOT have an error
	// because savings_rate IS defined on the line above it
	results := m.GetLineResults()

	// Find the savings line (should be 2 lines after original savingsRateLine)
	var savingsLineError string
	for _, r := range results {
		if strings.HasPrefix(r.Source, "savings = total_income") {
			if r.Error != "" {
				savingsLineError = r.Error
			}
			break
		}
	}

	if savingsLineError != "" {
		t.Errorf("savings line should NOT have an error after inserting line above.\n"+
			"  Error: %s\n"+
			"  This indicates savings_rate was not properly preserved when Enter was pressed at col 0",
			savingsLineError)
	}
}

// TestInsertLineAtMiddlePreservesContent verifies that pressing Enter in the
// middle of a line correctly splits the content.
func TestInsertLineAtMiddlePreservesContent(t *testing.T) {
	source := `a = 10
b = a * 2
`

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	m := New(doc)
	m.reEvaluate()

	// Position cursor at middle of "a = 10" (after "a = ")
	m.cursorLine = 0
	m.cursorCol = 4 // After "a = "
	m.editBuf = "a = 10"

	// Simulate pressing Enter
	newModel, _ := m.handleEnterKey()
	m = newModel.(Model)

	// Get updated lines
	newLines := m.GetLines()

	// Line 0 should be "a = "
	if newLines[0] != "a = " {
		t.Errorf("Line 0 should be 'a = ', got: %q", newLines[0])
	}

	// Line 1 should be "10"
	if newLines[1] != "10" {
		t.Errorf("Line 1 should be '10', got: %q", newLines[1])
	}
}
