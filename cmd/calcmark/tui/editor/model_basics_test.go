package editor

// model_basics_test.go — Model creation, basic accessors, status bar, line results.

import (
	"slices"
	"testing"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/config"
	"github.com/CalcMark/go-calcmark/spec/document"
	tea "charm.land/bubbletea/v2"
)

func init() {
	// Initialize config for tests
	config.Load()
}

func TestNew(t *testing.T) {
	// Test with nil document
	m := New(nil)
	if m.doc == nil {
		t.Error("Expected document to be initialized")
	}
	if m.eval == nil {
		t.Error("Expected evaluator to be initialized")
	}
	// Blank documents start in StateDefault for better UX
	if m.mode != StateDefault {
		t.Errorf("Expected StateDefault for blank document, got %v", m.mode)
	}

	// Test with existing document
	doc, _ := document.NewDocument("x = 10\ny = 20\n")
	m = New(doc)
	if m.doc != doc {
		t.Error("Expected document to be set")
	}
	if !m.pinnedVars["x"] || !m.pinnedVars["y"] {
		t.Error("Expected variables to be auto-pinned")
	}
}

func TestNewWithFile(t *testing.T) {
	doc, _ := document.NewDocument("x = 10\n")
	m := NewWithFile("test.cm", doc)

	if m.filepath != "test.cm" {
		t.Errorf("Expected filepath 'test.cm', got %q", m.filepath)
	}
}

func TestGetLines(t *testing.T) {
	doc, _ := document.NewDocument("x = 10\ny = 20\n")
	m := New(doc)

	lines := m.GetLines()
	// Document parser may add blank lines between blocks
	if len(lines) < 2 {
		t.Errorf("Expected at least 2 lines, got %d", len(lines))
	}
	// First non-empty line should be x = 10
	if !slices.Contains(lines, "x = 10") {
		t.Errorf("Expected to find 'x = 10' in lines: %v", lines)
	}
}

func TestTotalLines(t *testing.T) {
	doc, _ := document.NewDocument("a = 1\nb = 2\nc = 3\n")
	m := New(doc)

	// Document may have blank lines between blocks
	if m.TotalLines() < 3 {
		t.Errorf("Expected at least 3 total lines, got %d", m.TotalLines())
	}
}

func TestCalcBlockCount(t *testing.T) {
	doc, _ := document.NewDocument("x = 1\n\n\ny = 2\n")
	m := New(doc)

	// Each expression is a separate calc block
	if m.CalcBlockCount() < 1 {
		t.Errorf("Expected at least 1 calc block, got %d", m.CalcBlockCount())
	}
}

func TestMoveCursor(t *testing.T) {
	doc, _ := document.NewDocument("line1\nline2\nline3\n")
	m := New(doc)

	// Initial position
	if m.cursorLine != 0 {
		t.Errorf("Expected cursor at line 0, got %d", m.cursorLine)
	}

	// Move down
	m.moveCursor(1, 0)
	if m.cursorLine != 1 {
		t.Errorf("Expected cursor at line 1, got %d", m.cursorLine)
	}

	// Move up
	m.moveCursor(-1, 0)
	if m.cursorLine != 0 {
		t.Errorf("Expected cursor at line 0, got %d", m.cursorLine)
	}

	// Move beyond bounds
	m.moveCursor(-10, 0)
	if m.cursorLine != 0 {
		t.Error("Cursor should not go below 0")
	}

	m.moveCursor(100, 0)
	totalLines := m.TotalLines()
	if m.cursorLine != totalLines-1 {
		t.Errorf("Cursor should be at max line %d, got %d", totalLines-1, m.cursorLine)
	}
}

func TestHandleKeyQuit(t *testing.T) {
	// Ctrl+C with no selection should quit (standard interrupt signal)
	m := New(nil)
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	result := newModel.(Model)
	if !result.quitting {
		t.Error("Ctrl+C with no selection should set quitting=true")
	}
	if cmd == nil {
		t.Error("Ctrl+C with no selection should return quit command")
	}

	// Ctrl+C with selection should copy (not quit)
	doc, _ := document.NewDocument("test text\n")
	m = New(doc)
	m.SelectAll() // Select all text
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	result = newModel.(Model)
	if result.quitting {
		t.Error("Ctrl+C with selection should NOT quit")
	}
	if result.statusMsg != "Copied to clipboard" {
		t.Errorf("Expected 'Copied to clipboard' status, got: %s", result.statusMsg)
	}

	// Ctrl+Q should also quit (no unsaved changes)
	m = New(nil)
	newModel, cmd = m.Update(tea.KeyMsg{Type: tea.KeyCtrlQ})
	result = newModel.(Model)
	if !result.quitting {
		t.Error("Ctrl+Q should set quitting=true")
	}
	if cmd == nil {
		t.Error("Should return quit command")
	}
}

func TestGetStatusBarState(t *testing.T) {
	doc, _ := document.NewDocument("x = 10\ny = 20\n")
	m := NewWithFile("test.cm", doc)
	m.cursorLine = 1

	state := m.GetStatusBarState()

	if state.Filename != "test.cm" {
		t.Errorf("Expected filename 'test.cm', got %q", state.Filename)
	}
	if state.Line != 2 { // 1-indexed
		t.Errorf("Expected line 2, got %d", state.Line)
	}
	// TotalLines depends on how document parser creates blocks
	if state.TotalLines < 2 {
		t.Errorf("Expected at least 2 total lines, got %d", state.TotalLines)
	}
	// Mode is no longer shown to users - it's an internal implementation detail
	if state.Mode != "" {
		t.Errorf("Expected mode to be empty (not shown to users), got %q", state.Mode)
	}
}

func TestGetPinnedPanelState(t *testing.T) {
	doc, _ := document.NewDocument("x = 10\ny = 20\n")
	m := New(doc)

	state := m.GetPinnedPanelState(10)

	if len(state.Variables) != 2 {
		t.Errorf("Expected 2 pinned variables, got %d", len(state.Variables))
	}
}

func TestGetLineResults(t *testing.T) {
	doc, _ := document.NewDocument("x = 10\ny = x + 5\n")
	m := New(doc)

	results := m.GetLineResults()

	if len(results) < 2 {
		t.Fatalf("Expected at least 2 results, got %d", len(results))
	}

	// Should have results for calc blocks
	hasCalcResult := false
	for _, r := range results {
		if r.IsCalc && r.VarName != "" {
			hasCalcResult = true
			// Value should be a valid number
			if r.Value == "" && r.Error == "" {
				t.Errorf("Expected value or error for %s", r.VarName)
			}
		}
	}
	if !hasCalcResult {
		t.Error("Expected to find at least one calc result with variable")
	}
}

func TestGlobalsPanelState(t *testing.T) {
	// Test document with frontmatter globals
	// Note: exchange format requires FROM_TO keys like USD_EUR
	content := `---
globals:
  tax_rate: 0.25
exchange:
  USD_EUR: 0.85
  USD_GBP: 0.73
---
income = 5000
tax = income * tax_rate`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)

	state := m.GetGlobalsPanelState()

	// Should have globals from frontmatter
	if len(state.Globals) < 1 {
		t.Errorf("Expected at least 1 global, got %d", len(state.Globals))
	}

	// Check for expected globals
	var foundTaxRate, foundExchange bool
	for _, g := range state.Globals {
		if g.Name == "tax_rate" {
			foundTaxRate = true
		}
		if g.IsExchange && (g.Name == "USD_EUR" || g.Name == "USD_GBP") {
			foundExchange = true
		}
	}

	if !foundTaxRate {
		t.Error("Expected to find tax_rate global")
	}
	if !foundExchange {
		t.Error("Expected to find exchange rate globals")
	}

	// Test collapsed/expanded state
	if state.Expanded {
		t.Error("Globals should be collapsed by default")
	}

	// Expand globals
	m.globalsExpanded = true
	state = m.GetGlobalsPanelState()
	if !state.Expanded {
		t.Error("Globals should be expanded after setting flag")
	}
}

func TestLineResultsWithValues(t *testing.T) {
	// Test that GetLineResults returns per-statement values for each line
	doc, _ := document.NewDocument("x = 10\ny = x * 2\nz = y + 5\n")
	m := New(doc)

	// Log block structure for debugging
	blocks := doc.GetBlocks()
	t.Logf("Number of blocks: %d", len(blocks))
	for i, node := range blocks {
		switch b := node.Block.(type) {
		case *document.CalcBlock:
			t.Logf("Block %d: CalcBlock, source=%v, vars=%v", i, b.Source(), b.Variables())
		case *document.TextBlock:
			t.Logf("Block %d: TextBlock, source=%v", i, b.Source())
		}
	}

	results := m.GetLineResults()
	t.Logf("Number of results: %d", len(results))
	for i, r := range results {
		t.Logf("Result %d: line=%d, isCalc=%v, var=%q, value=%q, error=%q",
			i, r.LineNum, r.IsCalc, r.VarName, r.Value, r.Error)
	}

	// Collect values by variable name
	valuesByVar := make(map[string]string)
	for _, r := range results {
		if r.IsCalc && r.VarName != "" && r.Value != "" {
			valuesByVar[r.VarName] = r.Value
		}
	}

	// Verify per-line values are correct (not just LastValue)
	expected := map[string]string{
		"x": "10",
		"y": "20",
		"z": "25",
	}

	for varName, expectedVal := range expected {
		actual, ok := valuesByVar[varName]
		if !ok {
			t.Errorf("Expected to find variable %q in results", varName)
			continue
		}
		if actual != expectedVal {
			t.Errorf("Expected %s=%s, got %s", varName, expectedVal, actual)
		}
	}
}

func TestLineResultsMultiBlock(t *testing.T) {
	// Test with explicit blank lines to attempt separate blocks
	// Note: Current parser creates a single block even with blank lines.
	// This test documents the actual behavior: multi-statement blocks
	// show LastValue on first line, continuation lines are blank.
	content := `x = 10

y = x * 2

z = y + 5`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)

	// Log block structure
	blocks := doc.GetBlocks()
	t.Logf("Number of blocks: %d", len(blocks))
	for i, node := range blocks {
		switch b := node.Block.(type) {
		case *document.CalcBlock:
			t.Logf("Block %d: CalcBlock, vars=%v, lastValue=%v", i, b.Variables(), b.LastValue())
		case *document.TextBlock:
			t.Logf("Block %d: TextBlock, source=%v", i, b.Source())
		}
	}

	results := m.GetLineResults()
	t.Logf("Number of results: %d", len(results))

	// Current behavior: First line shows first variable with LastValue
	// Future improvement: Each line should show its own statement's value
	// using block.GetResults() indexed by statement position

	// Verify we have results
	if len(results) == 0 {
		t.Fatal("Expected at least one result")
	}

	// First result should have a value (even if it's LastValue)
	firstCalcResult := -1
	for i, r := range results {
		if r.IsCalc && r.Value != "" {
			firstCalcResult = i
			t.Logf("First calc result at index %d: var=%q, value=%q", i, r.VarName, r.Value)
			break
		}
	}

	if firstCalcResult == -1 {
		t.Error("Expected at least one calc result with a value")
	}

	// The final value should be 25 (z = y + 5 = 25)
	hasValue25 := false
	for _, r := range results {
		if r.Value == "25" {
			hasValue25 = true
			break
		}
	}
	if !hasValue25 {
		t.Error("Expected to find value 25 somewhere in results")
	}
}

func TestLineResultsWithError(t *testing.T) {
	// Test that errors are captured in results
	doc, _ := document.NewDocument("x = 10\ny = undefined_var\n")
	m := New(doc)

	results := m.GetLineResults()

	// Find the error result
	foundError := false
	for _, r := range results {
		if r.Error != "" {
			foundError = true
			t.Logf("Found error on line %d: %s", r.LineNum, r.Error)
		}
	}

	if !foundError {
		t.Error("Expected to find an error result for undefined variable")
	}
}

func TestVerticalAlignmentLineCount(t *testing.T) {
	// Test that source and preview have same line counts for 1:1 alignment
	content := `# Heading

x = 10
y = 20

## Section

z = x + y`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)
	m.width = 100
	m.height = 30

	// Get source lines
	sourceLines := m.GetLines()

	// Get preview results
	results := m.GetLineResults()

	// These should match for 1:1 vertical alignment
	if len(sourceLines) != len(results) {
		t.Errorf("Source lines (%d) and preview results (%d) should match for alignment",
			len(sourceLines), len(results))
	}

	// Each source line should have a corresponding result
	for i, line := range sourceLines {
		if i >= len(results) {
			t.Errorf("No result for line %d: %q", i, line)
			continue
		}
		r := results[i]
		if r.LineNum != i {
			t.Errorf("Line number mismatch: expected %d, got %d", i, r.LineNum)
		}
	}
}

func TestLiveUpdateCurrentLine(t *testing.T) {
	doc, _ := document.NewDocument("x = 10\n")
	m := New(doc)
	// User is always able to edit - load editBuf
	m.loadCurrentLineIntoEditBuffer()

	// Modify edit buffer
	m.editBuf = "x = 20"

	// Call live update
	m.updateCurrentLine(m.editBuf)
	m.reEvaluate()

	// Check that the document was updated
	lines := m.GetLines()
	if len(lines) == 0 || lines[0] != "x = 20" {
		t.Errorf("Expected line to be 'x = 20', got %v", lines)
	}

	// Check that results reflect the new value
	results := m.GetLineResults()
	for _, r := range results {
		if r.VarName == "x" {
			if r.Value != "20" {
				t.Errorf("Expected x=20 after live update, got %q", r.Value)
			}
			break
		}
	}
}
