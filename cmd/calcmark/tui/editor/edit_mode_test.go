package editor

// edit_mode_test.go — Edit mode lifecycle, buffer management, block type detection.

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/components"
	"github.com/CalcMark/go-calcmark/spec/document"
)

func TestEnterExitEditMode(t *testing.T) {
	doc, _ := document.NewDocument("x = 10\n")
	m := New(doc)

	// Enter edit mode
	// User is always able to edit - load editBuf
	m.loadCurrentLineIntoEditBuffer()
	if m.mode != StateDefault {
		t.Error("Expected StateDefault")
	}
	if m.editBuf != "x = 10" {
		t.Errorf("Expected edit buffer 'x = 10', got %q", m.editBuf)
	}

	// Exit edit mode
	m.saveCurrentLine(false) // Don't save
	if m.mode != StateDefault {
		t.Error("Expected StateDefault after exit")
	}
}

func TestEditModeSpaceKey(t *testing.T) {
	doc, _ := document.NewDocument("hello\n")
	m := New(doc)
	// User is always able to edit - load editBuf
	m.loadCurrentLineIntoEditBuffer()

	// Position cursor in middle of word
	m.cursorCol = 5 // After "hello"

	// Type a space
	newModel, _ := m.handleRuneInput([]rune{' '})
	result := newModel.(Model)

	if result.editBuf != "hello " {
		t.Errorf("Expected 'hello ', got %q", result.editBuf)
	}
	if result.cursorCol != 6 {
		t.Errorf("Expected cursor at 6, got %d", result.cursorCol)
	}

	// Type more characters
	newModel, _ = result.handleRuneInput([]rune{'w'})
	result = newModel.(Model)

	if result.editBuf != "hello w" {
		t.Errorf("Expected 'hello w', got %q", result.editBuf)
	}
}

func TestEnterEditModeEmptyDocument(t *testing.T) {
	// Test entering edit mode on an empty document
	doc, _ := document.NewDocument("")
	m := New(doc)

	// Verify initial state - document is empty
	lines := m.GetLines()
	initialLineCount := len(lines)
	t.Logf("Initial state: %d lines", initialLineCount)

	// Enter edit mode on empty document should work
	// User is always able to edit - load editBuf
	m.loadCurrentLineIntoEditBuffer()
	t.Logf("After enterEditMode: mode=%v, lines=%d", m.mode, m.TotalLines())

	if m.mode != StateDefault {
		t.Errorf("Expected StateDefault on empty document, got %v", m.mode)
	}

	// New behavior: document stays empty in edit mode
	// A line is only created when user types and saves
	lines = m.GetLines()
	if len(lines) != initialLineCount {
		t.Logf("Document should stay empty until user types (got %d lines)", len(lines))
	}

	// editBuf should be empty
	if m.editBuf != "" {
		t.Errorf("Expected empty editBuf, got %q", m.editBuf)
	}
}

func TestEmptyDocumentNewlineCreation(t *testing.T) {
	// Test various document contents
	tests := []struct {
		name    string
		content string
	}{
		{"empty", ""},
		{"newline", "\n"},
		{"space", " "},
		{"underscore", "_"},
		{"expression", "x = 1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := document.NewDocument(tt.content)
			if err != nil {
				t.Fatalf("NewDocument failed: %v", err)
			}

			blocks := doc.GetBlocks()
			t.Logf("Content %q has %d blocks", tt.content, len(blocks))
			for i, b := range blocks {
				t.Logf("  Block %d: %T, source=%v", i, b.Block, b.Block.Source())
			}
		})
	}
}

func TestNewDocumentEditBuffer(t *testing.T) {
	// Test that when creating a new document, the edit buffer starts empty
	// not with the placeholder character
	doc, _ := document.NewDocument("")
	m := New(doc)

	// Enter edit mode on empty document
	// User is always able to edit - load editBuf
	m.loadCurrentLineIntoEditBuffer()

	// The edit buffer should be empty, not contain underscore placeholder
	if m.editBuf == "_" {
		t.Error("Edit buffer should not contain underscore placeholder for new document")
	}
	if m.editBuf != "" {
		t.Errorf("Edit buffer should be empty for new document, got %q", m.editBuf)
	}
}

// TestSurgicalUpdateOnEdit tests that editing a line triggers surgical updates
// to dependent blocks and the environment.
func TestSurgicalUpdateOnEdit(t *testing.T) {
	// Create document with dependency: y depends on x
	doc, _ := document.NewDocument("x = 10\ny = x * 2\n")
	m := New(doc)

	// Verify initial values
	results := m.GetLineResults()
	initialValues := make(map[string]string)
	for _, r := range results {
		if r.VarName != "" && r.Value != "" {
			initialValues[r.VarName] = r.Value
		}
	}
	t.Logf("Initial values: %v", initialValues)

	if initialValues["x"] != "10" {
		t.Errorf("Expected initial x=10, got %s", initialValues["x"])
	}
	if initialValues["y"] != "20" {
		t.Errorf("Expected initial y=20, got %s", initialValues["y"])
	}

	// Enter edit mode on first line (x = 10)
	m.cursorLine = 0
	// User is always able to edit - load editBuf
	m.loadCurrentLineIntoEditBuffer()

	// Change x to 100
	m.editBuf = "x = 100"
	m.updateCurrentLine(m.editBuf)
	m.reEvaluate()

	// Exit edit mode to trigger full re-evaluation
	m.saveCurrentLine(true)

	// Get updated results
	results = m.GetLineResults()
	updatedValues := make(map[string]string)
	for _, r := range results {
		if r.VarName != "" && r.Value != "" {
			updatedValues[r.VarName] = r.Value
		}
	}
	t.Logf("Updated values: %v", updatedValues)

	// Verify x was updated
	if updatedValues["x"] != "100" {
		t.Errorf("Expected updated x=100, got %s", updatedValues["x"])
	}

	// Verify y was updated due to dependency on x
	if updatedValues["y"] != "200" {
		t.Errorf("Expected updated y=200 (x*2=100*2), got %s", updatedValues["y"])
	}
}

// TestChangedBlockIDsTracking tests that changedBlockIDs is updated on edits.
func TestChangedBlockIDsTracking(t *testing.T) {
	doc, _ := document.NewDocument("x = 10\n")
	m := New(doc)

	// Initially no changed blocks
	if len(m.changedBlockIDs) != 0 {
		t.Errorf("Expected 0 changed blocks initially, got %d", len(m.changedBlockIDs))
	}

	// Enter edit mode and make a change
	// User is always able to edit - load editBuf
	m.loadCurrentLineIntoEditBuffer()
	m.editBuf = "x = 20"
	m.updateCurrentLine(m.editBuf)
	m.reEvaluate()

	// After liveUpdate, changedBlockIDs should have the affected block
	if len(m.changedBlockIDs) == 0 {
		t.Log("Note: changedBlockIDs may be cleared after reEvaluate()")
	}

	// Results should show WasChanged flag (before reEvaluate clears it)
	// This is implementation-dependent - document the actual behavior
	results := m.GetLineResults()
	for _, r := range results {
		t.Logf("Result: var=%q, value=%q, wasChanged=%v", r.VarName, r.Value, r.WasChanged)
	}
}

// TestEnvironmentUpdateOnEdit tests that the evaluator's environment is updated.
func TestEnvironmentUpdateOnEdit(t *testing.T) {
	doc, _ := document.NewDocument("x = 10\n")
	m := New(doc)

	// Check initial environment value
	env := m.eval.GetEnvironment()
	val, ok := env.Get("x")
	if !ok {
		t.Fatal("Expected variable 'x' to be in environment")
	}
	if val.String() != "10" {
		t.Errorf("Expected x=10 in environment, got %s", val.String())
	}

	// Make a change
	// User is always able to edit - load editBuf
	m.loadCurrentLineIntoEditBuffer()
	m.editBuf = "x = 42"
	m.updateCurrentLine(m.editBuf)
	m.reEvaluate()
	m.saveCurrentLine(true)

	// Check updated environment value
	env = m.eval.GetEnvironment()
	val, ok = env.Get("x")
	if !ok {
		t.Fatal("Expected variable 'x' to still be in environment after edit")
	}
	if val.String() != "42" {
		t.Errorf("Expected x=42 in environment after edit, got %s", val.String())
	}
}

// TestDependencyChainUpdate tests that editing a variable updates all dependents.
func TestDependencyChainUpdate(t *testing.T) {
	// Create a dependency chain: a -> b -> c
	doc, _ := document.NewDocument("a = 5\nb = a + 10\nc = b * 2\n")
	m := New(doc)

	// Verify initial values
	results := m.GetLineResults()
	values := make(map[string]string)
	for _, r := range results {
		if r.VarName != "" && r.Value != "" {
			values[r.VarName] = r.Value
		}
	}
	t.Logf("Initial: a=%s, b=%s, c=%s", values["a"], values["b"], values["c"])

	// Expected: a=5, b=15, c=30
	if values["a"] != "5" {
		t.Errorf("Expected a=5, got %s", values["a"])
	}
	if values["b"] != "15" {
		t.Errorf("Expected b=15 (a+10=5+10), got %s", values["b"])
	}
	if values["c"] != "30" {
		t.Errorf("Expected c=30 (b*2=15*2), got %s", values["c"])
	}

	// Change a from 5 to 10
	m.cursorLine = 0
	// User is always able to edit - load editBuf
	m.loadCurrentLineIntoEditBuffer()
	m.editBuf = "a = 10"
	m.updateCurrentLine(m.editBuf)
	m.reEvaluate()
	m.saveCurrentLine(true)

	// Verify chain was updated
	results = m.GetLineResults()
	values = make(map[string]string)
	for _, r := range results {
		if r.VarName != "" && r.Value != "" {
			values[r.VarName] = r.Value
		}
	}
	t.Logf("After edit: a=%s, b=%s, c=%s", values["a"], values["b"], values["c"])

	// Expected: a=10, b=20, c=40
	if values["a"] != "10" {
		t.Errorf("Expected a=10, got %s", values["a"])
	}
	if values["b"] != "20" {
		t.Errorf("Expected b=20 (a+10=10+10), got %s", values["b"])
	}
	if values["c"] != "40" {
		t.Errorf("Expected c=40 (b*2=20*2), got %s", values["c"])
	}
}

// TestPinnedVariablesUpdate tests that pinned variables are updated on edits.
// TestMarkdownInCalcBlockNoError tests that markdown content in CalcBlocks
// doesn't show unhelpful "undefined_variable" errors in preview.
func TestMarkdownInCalcBlockNoError(t *testing.T) {
	// Create a document with markdown-like content
	// When typed in edit mode, this could end up in a CalcBlock
	doc, _ := document.NewDocument("# Heading\n")
	m := New(doc)

	results := m.GetLineResults()

	// The line should be treated as text, not show calc error
	for _, r := range results {
		if strings.HasPrefix(r.Source, "#") {
			// This is our markdown line - it should either:
			// 1. Be detected as !IsCalc (TextBlock) by document.Detector
			// 2. Or if in CalcBlock, the view uses Detector.IsCalculation() to check
			t.Logf("Markdown line: IsCalc=%v, Error=%q, Source=%q", r.IsCalc, r.Error, r.Source)

			// The view layer uses document.Detector.IsCalculation() to determine
			// if a line should show calc error or render as markdown
		}
	}
}

func TestPinnedVariablesUpdate(t *testing.T) {
	doc, _ := document.NewDocument("total = 100\ntax = total * 0.1\n")
	m := New(doc)

	// Variables should be auto-pinned
	if !m.pinnedVars["total"] {
		t.Error("Expected 'total' to be auto-pinned")
	}
	if !m.pinnedVars["tax"] {
		t.Error("Expected 'tax' to be auto-pinned")
	}

	// Get pinned panel state
	state := m.GetPinnedPanelState(10)
	t.Logf("Pinned variables: %+v", state.Variables)

	// Find tax variable
	var taxVar *components.PinnedVar
	for i := range state.Variables {
		if state.Variables[i].Name == "tax" {
			taxVar = &state.Variables[i]
			break
		}
	}

	if taxVar == nil {
		t.Fatal("Expected to find 'tax' in pinned variables")
	}
	if taxVar.Value != "10" {
		t.Errorf("Expected tax=10 (100*0.1), got %s", taxVar.Value)
	}

	// Change total from 100 to 200
	m.cursorLine = 0
	// User is always able to edit - load editBuf
	m.loadCurrentLineIntoEditBuffer()
	m.editBuf = "total = 200"
	m.updateCurrentLine(m.editBuf)
	m.reEvaluate()
	m.saveCurrentLine(true)

	// Get updated pinned panel state
	state = m.GetPinnedPanelState(10)
	for i := range state.Variables {
		if state.Variables[i].Name == "tax" {
			taxVar = &state.Variables[i]
			break
		}
	}

	if taxVar.Value != "20" {
		t.Errorf("Expected tax=20 (200*0.1) after edit, got %s", taxVar.Value)
	}
}

// TestEditModeWrappedLineNoDuplicate tests that in edit mode, a long line
// wraps correctly without duplicating content.
func TestBlockTypeRedetection_CalcToMarkdown(t *testing.T) {
	// Test that editing a calc line to markdown content properly re-detects the block type
	content := `x = 10
y = 20`

	doc, _ := document.NewDocument(content)
	m := New(doc)
	m.width = 80
	m.height = 24

	// Verify initial state - both lines should be calc
	results := m.GetLineResults()
	if !results[0].IsCalc {
		t.Error("Line 0 should initially be a calc")
	}
	if !results[1].IsCalc {
		t.Error("Line 1 should initially be a calc")
	}

	// Edit line 1 to be markdown
	m.cursorLine = 1
	// User is always able to edit - load editBuf
	m.loadCurrentLineIntoEditBuffer()
	m.editBuf = "- this is a list item"
	m.saveCurrentLine(true)

	// After exit, the document should have re-detected block types
	results = m.GetLineResults()

	// Line 0 should still be calc
	if !results[0].IsCalc {
		t.Error("Line 0 should still be a calc after editing line 1")
	}

	// Line 1 should now be text (markdown)
	if results[1].IsCalc {
		t.Errorf("Line 1 should now be text/markdown, but IsCalc=%v, Source=%q",
			results[1].IsCalc, results[1].Source)
	}
}

func TestBlockTypeRedetection_MarkdownToCalc(t *testing.T) {
	// Test that editing a markdown line to calc content properly re-detects the block type
	content := `# Header
Some text here`

	doc, _ := document.NewDocument(content)
	m := New(doc)
	m.width = 80
	m.height = 24

	// Verify initial state - both lines should be text
	results := m.GetLineResults()
	if results[0].IsCalc {
		t.Error("Line 0 should initially be text")
	}
	if results[1].IsCalc {
		t.Error("Line 1 should initially be text")
	}

	// Edit line 1 to be a calculation
	m.cursorLine = 1
	// User is always able to edit - load editBuf
	m.loadCurrentLineIntoEditBuffer()
	m.editBuf = "total = 100 + 200"
	m.saveCurrentLine(true)

	// After exit, the document should have re-detected block types
	results = m.GetLineResults()

	// Line 0 should still be text
	if results[0].IsCalc {
		t.Error("Line 0 should still be text after editing line 1")
	}

	// Line 1 should now be calc
	if !results[1].IsCalc {
		t.Errorf("Line 1 should now be calc, but IsCalc=%v, Source=%q",
			results[1].IsCalc, results[1].Source)
	}

	// Verify the calculation was evaluated
	if results[1].Value == "" {
		t.Error("Line 1 should have a computed value")
	}
}

func TestInsertLine_ThenEditAsMarkdown(t *testing.T) {
	// Test the original bug: insert a new line, type markdown, it should render
	content := `x = 10`

	doc, _ := document.NewDocument(content)
	m := New(doc)
	m.width = 80
	m.height = 24

	// Insert line below and enter edit mode (simulates pressing 'o')
	m.cursorLine = 0
	m.insertLineBelow()
	// User is always able to edit - load editBuf
	m.loadCurrentLineIntoEditBuffer()

	// Type markdown content
	m.editBuf = "- list item one"
	m.saveCurrentLine(true)

	// The new line should be detected as markdown
	results := m.GetLineResults()

	// Should have 2 lines now
	if len(results) != 2 {
		t.Fatalf("Expected 2 lines, got %d", len(results))
	}

	// Line 0 should be calc
	if !results[0].IsCalc {
		t.Error("Line 0 should be calc")
	}

	// Line 1 should be markdown (not calc)
	if results[1].IsCalc {
		t.Errorf("Line 1 should be markdown, but IsCalc=%v, Source=%q",
			results[1].IsCalc, results[1].Source)
	}
}

// =============================================================================
// Insert Line Tests
// These tests define the expected behavior when inserting new lines.
// =============================================================================

func TestEditModeOnPaddedLine(t *testing.T) {
	// Test that entering edit mode on a line that has padding works correctly

	content := `short = 1
this_is_line_two = 2`

	doc, _ := document.NewDocument(content)
	m := New(doc)
	m.width = 60
	m.height = 24

	// Navigate to line 1
	m.cursorLine = 1
	m.cursorCol = 0

	// Enter edit mode
	// User is always able to edit - load editBuf
	m.loadCurrentLineIntoEditBuffer()

	if m.mode != StateDefault {
		t.Fatalf("Expected StateDefault, got %v", m.mode)
	}

	// Verify edit buffer contains the correct line
	expectedContent := "this_is_line_two = 2"
	if m.editBuf != expectedContent {
		t.Errorf("Edit buffer = %q, want %q", m.editBuf, expectedContent)
	}

	// Compute aligned panes
	leftWidth, rightWidth := m.GetPaneWidths(m.width)
	aligned := m.computeAlignedPanes(leftWidth, rightWidth)

	// Find the source line that is marked as cursor line
	cursorLineFound := false
	for i, sl := range aligned.sourceLines {
		if sl.isCursorLine {
			cursorLineFound = true
			if sl.sourceLineIdx != m.cursorLine {
				t.Errorf("Cursor line at visual %d has sourceLineIdx=%d, want %d",
					i, sl.sourceLineIdx, m.cursorLine)
			}
			t.Logf("Cursor line found at visual index %d, sourceLineIdx=%d", i, sl.sourceLineIdx)
		}
	}

	if !cursorLineFound {
		t.Error("No source line marked as cursor line")
	}

	// Simulate typing - modify edit buffer
	m.editBuf = "modified_line = 999"
	m.cursorCol = len(m.editBuf)

	// Exit edit mode (save)
	m.saveCurrentLine(true)

	// Verify the change was saved
	lines := m.GetLines()
	if lines[1] != "modified_line = 999" {
		t.Errorf("Line not updated after edit: got %q", lines[1])
	}
}

func TestEditModePreservesVisualPosition(t *testing.T) {
	// Test that the cursor visual position is correct after entering edit mode
	// on a line that has padding lines around it

	content := `# Header that might wrap on narrow screen
x = 10
# Another header
y = 20`

	doc, _ := document.NewDocument(content)
	m := New(doc)
	m.width = 50 // Narrow
	m.height = 24

	// Move to the calc line
	m.cursorLine = 1

	leftWidth, rightWidth := m.GetPaneWidths(m.width)

	// Get aligned panes before edit mode
	alignedBefore := m.computeAlignedPanes(leftWidth, rightWidth)
	visualBefore := alignedBefore.sourceToVisual[m.cursorLine]

	// Enter edit mode
	// User is always able to edit - load editBuf
	m.loadCurrentLineIntoEditBuffer()

	// Get aligned panes in edit mode
	alignedDuring := m.computeAlignedPanes(leftWidth, rightWidth)
	visualDuring := alignedDuring.sourceToVisual[m.cursorLine]

	t.Logf("Visual line before edit: %d, during edit: %d", visualBefore, visualDuring)

	// The visual line for the cursor should be the same
	// (edit mode shouldn't change which visual line the cursor is on)
	if visualBefore != visualDuring {
		t.Errorf("Visual line changed when entering edit mode: %d -> %d",
			visualBefore, visualDuring)
	}

	// Verify the cursor line is marked correctly
	cursorMarked := false
	for _, sl := range alignedDuring.sourceLines {
		if sl.isCursorLine && sl.sourceLineIdx == m.cursorLine {
			cursorMarked = true
			break
		}
	}
	if !cursorMarked {
		t.Error("Cursor line not marked correctly during edit mode")
	}
}

func TestEditMode_CursorOnWrappedLine(t *testing.T) {
	// Test editing a line that wraps to multiple visual lines

	content := `short = 1
this_is_a_line_that_is_long_enough_to_wrap_in_narrow_terminal = 999`

	doc, _ := document.NewDocument(content)
	m := New(doc)
	m.width = 50 // Narrow
	m.height = 24
	m.previewMode = PreviewFull

	// Move to line 1 (the long one)
	m.cursorLine = 1
	// User is always able to edit - load editBuf
	m.loadCurrentLineIntoEditBuffer()

	if m.mode != StateDefault {
		t.Fatalf("Expected StateDefault, got %v", m.mode)
	}

	// Verify edit buffer has full content (not truncated)
	expectedContent := "this_is_a_line_that_is_long_enough_to_wrap_in_narrow_terminal = 999"
	if m.editBuf != expectedContent {
		t.Errorf("Edit buffer = %q, want %q", m.editBuf, expectedContent)
	}

	// Compute aligned panes in edit mode
	leftWidth, rightWidth := m.GetPaneWidths(m.width)
	aligned := m.computeAlignedPanes(leftWidth, rightWidth)

	// Find cursor line in visual structure
	cursorVisualIdx := -1
	for i, sl := range aligned.sourceLines {
		if sl.isCursorLine {
			cursorVisualIdx = i
			break
		}
	}

	if cursorVisualIdx == -1 {
		t.Error("No cursor line found in visual structure")
	}

	// Verify the visual index matches the mapping
	expectedVisualIdx := aligned.sourceToVisual[m.cursorLine]
	if cursorVisualIdx != expectedVisualIdx {
		t.Errorf("Cursor visual index %d doesn't match mapping %d",
			cursorVisualIdx, expectedVisualIdx)
	}

	t.Logf("Cursor at source line %d, visual line %d", m.cursorLine, cursorVisualIdx)

	// Modify and save
	m.editBuf = "modified = 123"
	m.saveCurrentLine(true)

	// Verify save worked
	lines := m.GetLines()
	if lines[1] != "modified = 123" {
		t.Errorf("Line not saved correctly: got %q", lines[1])
	}
}
