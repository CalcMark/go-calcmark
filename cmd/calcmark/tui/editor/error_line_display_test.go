package editor

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
)

// TestErrorLineDisplaysAtCorrectPosition verifies that errors display
// at the correct source line in the preview pane.
//
// Bug: When there's an error at line 10 (`accumulate(5mb, 1 hour)`),
// the error indicator "⚠ error" displays at line 4 in the preview
// pane instead of line 10.
func TestErrorLineDisplaysAtCorrectPosition(t *testing.T) {
	// Reproduce exact scenario from user bug report
	source := `# Monthly Budget

## Income
salary = $5000
side_hustle = $800
total_income = salary + side_hustle

123 + 4

accumulate(5mb, 1 hour)

## Fixed Expenses
rent = $1500
`

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	// Evaluate the document (this should produce an error on line 10)
	doc.Evaluate()

	// Create model from document
	m := &Model{doc: doc}

	// Get line results
	results := m.GetLineResults()

	// Find which line has the error
	var errorLineNum int = -1
	var errorLine string
	for _, r := range results {
		if r.Error != "" {
			errorLineNum = r.LineNum
			errorLine = r.Source
			break
		}
	}

	if errorLineNum == -1 {
		t.Fatal("Expected an error but none found")
	}

	// The error should be on line 10 (0-indexed: 9) which contains "accumulate(5mb, 1 hour)"
	// NOT on line 4 (0-indexed: 3) which contains "salary = $5000"
	expectedErrorLine := 9 // 0-indexed line number for line 10
	expectedErrorContent := "accumulate(5mb, 1 hour)"

	if errorLineNum != expectedErrorLine {
		t.Errorf("Error displayed on wrong line.\n"+
			"  Expected: line %d (0-indexed) containing %q\n"+
			"  Got:      line %d (0-indexed) containing %q\n"+
			"  Error:    %s",
			expectedErrorLine, expectedErrorContent,
			errorLineNum, errorLine,
			results[errorLineNum].Error)
	}

	// Also verify the error content is what we expect
	if !strings.Contains(errorLine, "accumulate") {
		t.Errorf("Error should be on the accumulate line, got: %q", errorLine)
	}
}

// TestErrorLineWithMultipleBlocks verifies error line tracking works
// correctly across multiple CalcBlocks.
func TestErrorLineWithMultipleBlocks(t *testing.T) {
	// Document with multiple CalcBlocks separated by 2+ empty lines
	// Error should appear in the correct block at the correct line
	source := `a = 10
b = 20


c = undefined_var
`

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	doc.Evaluate()

	m := &Model{doc: doc}
	results := m.GetLineResults()

	// Find error
	var errorLineNum int = -1
	for _, r := range results {
		if r.Error != "" {
			errorLineNum = r.LineNum
			break
		}
	}

	if errorLineNum == -1 {
		t.Fatal("Expected an error on line with undefined_var")
	}

	// Error should be on line 5 (0-indexed: 4) which contains "c = undefined_var"
	// NOT on line 1 (0-indexed: 0) which is the first calc line
	expectedErrorLine := 4

	if errorLineNum != expectedErrorLine {
		t.Errorf("Error displayed on wrong line. Expected line %d (0-indexed), got line %d",
			expectedErrorLine, errorLineNum)
	}
}
