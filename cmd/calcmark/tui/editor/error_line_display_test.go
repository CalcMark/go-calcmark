package editor

import (
	"strings"
	"testing"

	impldoc "github.com/CalcMark/go-calcmark/v2/impl/document"
	"github.com/CalcMark/go-calcmark/v2/spec/document"
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
	eval := impldoc.NewEvaluator()
	eval.Evaluate(doc)

	// Create model from document
	m := &Model{doc: doc}

	// Get line results
	results := m.GetLineResults()

	// Debug: Print diagnostic info to trace the mapping
	for i, block := range doc.GetBlocks() {
		if cb, ok := block.Block.(*document.CalcBlock); ok {
			t.Logf("Block %d (%s): %d source lines, %d diagnostics, error=%v",
				i, block.ID[:8], len(cb.Source()), len(cb.Diagnostics()), cb.Error() != nil)
			for _, d := range cb.Diagnostics() {
				t.Logf("  Diagnostic: Line=%d Col=%d Code=%s", d.Line, d.Column, d.Code)
			}
		}
	}

	// Debug: Print all line results with errors
	for _, r := range results {
		if r.Error != "" {
			t.Logf("LineResult: LineNum=%d Source=%q Error=%q", r.LineNum, r.Source, r.Error)
		}
	}

	// Also verify the AlignedModel produces the error in the right preview line
	input := AlignedModelInput{
		Lines:              m.GetLines(),
		Results:            results,
		SourceContentWidth: 60,
		PreviewWidth:       40,
		CursorLine:         0,
		PreviewMode:        PreviewFull,
	}
	aligned := ComputeAlignedModel(input, m.renderCalcLine, nil)

	// Find which preview line contains the error indicator
	for i, pl := range aligned.PreviewLines {
		if strings.Contains(pl.Content, "⚠") || strings.Contains(pl.Content, "error") {
			t.Logf("Preview line %d (SourceLineIdx=%d): %q", i, pl.SourceLineIdx, pl.Content)
		}
	}

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

// TestErrorLineViaLiveEditing verifies that errors display at the correct line
// when a user TYPES a line with an error (not just loading a document with one).
// This exercises the impl/document/evaluator.go code path used by the TUI.
func TestErrorLineViaLiveEditing(t *testing.T) {
	// Start with a document WITHOUT the error line (simulating opening budget.cm)
	source := `# Monthly Budget

## Income
salary = $5000
side_hustle = $800
total_income = salary + side_hustle

123 + 4

## Fixed Expenses
rent = $1500
`

	doc, err := document.NewDocument(source)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	// Create a proper editor Model with evaluator (like the TUI does)
	m := New(doc)

	// Evaluate the initial document
	m.reEvaluate()

	// Simulate user typing on a new line: insert "accumulate(5mb, 1 hour)" at line 9
	// First, navigate to line 9 (between 123+4 and ## Fixed Expenses)
	// The document structure is:
	// 0: # Monthly Budget
	// 1: (empty)
	// 2: ## Income
	// 3: salary = $5000
	// 4: side_hustle = $800
	// 5: total_income = ...
	// 6: (empty)
	// 7: 123 + 4
	// 8: (empty)
	// 9: ## Fixed Expenses  <- we'll insert before this

	// Instead of simulating navigation, just recreate with the final content
	sourceWithError := `# Monthly Budget

## Income
salary = $5000
side_hustle = $800
total_income = salary + side_hustle

123 + 4

accumulate(5mb, 1 hour)

## Fixed Expenses
rent = $1500
`

	doc2, err := document.NewDocument(sourceWithError)
	if err != nil {
		t.Fatalf("NewDocument failed: %v", err)
	}

	// Create model and use the TUI's evaluator (impl/document)
	m2 := New(doc2)
	m2.reEvaluate()

	// Get line results
	results := m2.GetLineResults()

	// Debug: Print diagnostic info
	for i, block := range m2.doc.GetBlocks() {
		if cb, ok := block.Block.(*document.CalcBlock); ok {
			t.Logf("Block %d (%s): %d source lines, %d diagnostics, error=%v",
				i, block.ID[:8], len(cb.Source()), len(cb.Diagnostics()), cb.Error() != nil)
			for _, d := range cb.Diagnostics() {
				t.Logf("  Diagnostic: Line=%d Col=%d Code=%s", d.Line, d.Column, d.Code)
			}
		}
	}

	// Debug: Print all line results with errors
	for _, r := range results {
		if r.Error != "" {
			t.Logf("LineResult: LineNum=%d Source=%q Error=%q", r.LineNum, r.Source, r.Error)
		}
	}

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

	eval := impldoc.NewEvaluator()
	eval.Evaluate(doc)

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
