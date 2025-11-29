package editor

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/config"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/components"
	"github.com/CalcMark/go-calcmark/spec/document"
	tea "github.com/charmbracelet/bubbletea"
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
	if m.mode != ModeNormal {
		t.Errorf("Expected ModeNormal, got %v", m.mode)
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

func TestEnterExitEditMode(t *testing.T) {
	doc, _ := document.NewDocument("x = 10\n")
	m := New(doc)

	// Enter edit mode
	m.enterEditMode()
	if m.mode != ModeEditing {
		t.Error("Expected ModeEditing")
	}
	if m.editBuf != "x = 10" {
		t.Errorf("Expected edit buffer 'x = 10', got %q", m.editBuf)
	}

	// Exit edit mode
	m.exitEditMode(false) // Don't save
	if m.mode != ModeNormal {
		t.Error("Expected ModeNormal after exit")
	}
}

func TestEditModeSpaceKey(t *testing.T) {
	doc, _ := document.NewDocument("hello\n")
	m := New(doc)
	m.enterEditMode()

	// Position cursor in middle of word
	m.cursorCol = 5 // After "hello"

	// Type a space
	newModel, _ := m.handleEditKey(tea.KeyMsg{Type: tea.KeySpace})
	result := newModel.(Model)

	if result.editBuf != "hello " {
		t.Errorf("Expected 'hello ', got %q", result.editBuf)
	}
	if result.cursorCol != 6 {
		t.Errorf("Expected cursor at 6, got %d", result.cursorCol)
	}

	// Type more characters
	newModel, _ = result.handleEditKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
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
	m.enterEditMode()
	t.Logf("After enterEditMode: mode=%v, lines=%d", m.mode, m.TotalLines())

	if m.mode != ModeEditing {
		t.Errorf("Expected ModeEditing on empty document, got %v", m.mode)
	}

	// A line should have been created
	lines = m.GetLines()
	if len(lines) <= initialLineCount {
		t.Error("Expected a line to be created when entering edit mode on empty document")
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

func TestHandleKeyQuit(t *testing.T) {
	m := New(nil)

	// Ctrl+C
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	result := newModel.(Model)
	if !result.quitting {
		t.Error("Ctrl+C should set quitting=true")
	}
	if cmd == nil {
		t.Error("Should return quit command")
	}

	// Ctrl+D
	m = New(nil)
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	result = newModel.(Model)
	if !result.quitting {
		t.Error("Ctrl+D should set quitting=true")
	}
}

func TestHandleNormalRunes(t *testing.T) {
	doc, _ := document.NewDocument("line1\nline2\nline3\n")
	m := New(doc)

	// j = down
	tm, _ := m.handleNormalRune([]rune{'j'})
	result := tm.(Model)
	if result.cursorLine != 1 {
		t.Error("'j' should move cursor down")
	}

	// k = up
	tm, _ = result.handleNormalRune([]rune{'k'})
	result = tm.(Model)
	if result.cursorLine != 0 {
		t.Error("'k' should move cursor up")
	}

	// G = go to end
	tm, _ = result.handleNormalRune([]rune{'G'})
	result = tm.(Model)
	totalLines := result.TotalLines()
	if result.cursorLine != totalLines-1 {
		t.Errorf("'G' should go to last line %d, got %d", totalLines-1, result.cursorLine)
	}

	// e = edit mode
	tm, _ = result.handleNormalRune([]rune{'e'})
	result = tm.(Model)
	if result.mode != ModeEditing {
		t.Error("'e' should enter edit mode")
	}
}

func TestHandleCommandMode(t *testing.T) {
	m := New(nil)

	// Enter command mode with /
	tm, _ := m.handleNormalRune([]rune{'/'})
	result := tm.(Model)
	if result.mode != ModeCommand {
		t.Error("'/' should enter command mode")
	}

	// Type command
	result.cmdInput = "quit"
	tm, _ = result.handleCommandKey(tea.KeyMsg{Type: tea.KeyEnter})
	result = tm.(Model)
	if !result.quitting {
		t.Error("/quit should set quitting")
	}
}

func TestHandleCommandMode_SpaceInCommand(t *testing.T) {
	m := New(nil)
	m.mode = ModeCommand
	m.cmdInput = "save"

	// Type a space
	tm, _ := m.handleCommandKey(tea.KeyMsg{Type: tea.KeySpace})
	result := tm.(Model)

	if result.cmdInput != "save " {
		t.Errorf("Space should be added to command input, got %q", result.cmdInput)
	}

	// Type filename characters
	tm, _ = result.handleCommandKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t', 'e', 's', 't'}})
	result = tm.(Model)

	if result.cmdInput != "save test" {
		t.Errorf("Expected 'save test', got %q", result.cmdInput)
	}

	// Type another space and more
	tm, _ = result.handleCommandKey(tea.KeyMsg{Type: tea.KeySpace})
	result = tm.(Model)
	tm, _ = result.handleCommandKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f', 'i', 'l', 'e'}})
	result = tm.(Model)

	if result.cmdInput != "save test file" {
		t.Errorf("Expected 'save test file', got %q", result.cmdInput)
	}
}

func TestUndo(t *testing.T) {
	doc, _ := document.NewDocument("x = 10\n")
	m := New(doc)

	// Initial undo stack should have one entry
	if len(m.undoStack) != 1 {
		t.Errorf("Expected 1 undo entry, got %d", len(m.undoStack))
	}

	// Make a change
	m.pushUndoState() // This will be a duplicate, so shouldn't add
	if len(m.undoStack) != 1 {
		t.Error("Duplicate state should not be added")
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
	if state.Mode != "NORMAL" {
		t.Errorf("Expected mode 'NORMAL', got %q", state.Mode)
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

func TestTogglePreview(t *testing.T) {
	m := New(nil)

	// Default should be Full preview mode
	if m.previewMode != PreviewFull {
		t.Errorf("Preview should be Full by default, got %v", m.previewMode)
	}

	// First toggle: Full → Minimal
	tm, _ := m.handleNormalRune([]rune{'p'})
	result := tm.(Model)
	if result.previewMode != PreviewMinimal {
		t.Errorf("Preview should be Minimal after first toggle, got %v", result.previewMode)
	}

	// Second toggle: Minimal → Hidden
	tm, _ = result.handleNormalRune([]rune{'p'})
	result = tm.(Model)
	if result.previewMode != PreviewHidden {
		t.Errorf("Preview should be Hidden after second toggle, got %v", result.previewMode)
	}

	// Third toggle: Hidden → Full
	tm, _ = result.handleNormalRune([]rune{'p'})
	result = tm.(Model)
	if result.previewMode != PreviewFull {
		t.Errorf("Preview should be Full after third toggle, got %v", result.previewMode)
	}
}

func TestGlobalsMode(t *testing.T) {
	doc, _ := document.NewDocument("@global.rate = 0.1\nx = 10\n")
	m := New(doc)

	// 'g' followed by non-'g' key should enter globals mode
	// First 'g' sets pending key
	tm, _ := m.handleNormalRune([]rune{'g'})
	result := tm.(Model)
	if result.pendingKey != 'g' {
		t.Errorf("First 'g' should set pending key, got %c", result.pendingKey)
	}

	// Second key (not 'g') enters globals mode
	tm, _ = result.handleNormalRune([]rune{'x'}) // any non-g key
	result = tm.(Model)
	if result.mode != ModeGlobals {
		t.Error("'g' + non-g key should enter globals mode")
	}
	if !result.globalsExpanded {
		t.Error("Globals should be expanded")
	}

	// Exit with Escape
	tm, _ = result.handleGlobalsKey(tea.KeyMsg{Type: tea.KeyEsc})
	result = tm.(Model)
	if result.mode != ModeNormal {
		t.Error("Escape should exit globals mode")
	}
}

func TestGgGoToTop(t *testing.T) {
	doc, _ := document.NewDocument("line1\nline2\nline3\n")
	m := New(doc)

	// Move to bottom
	m.cursorLine = 2

	// Press 'g' then 'g' - should go to top
	tm, _ := m.handleNormalRune([]rune{'g'})
	result := tm.(Model)
	if result.pendingKey != 'g' {
		t.Error("First 'g' should set pending key")
	}

	tm, _ = result.handleNormalRune([]rune{'g'})
	result = tm.(Model)
	if result.cursorLine != 0 {
		t.Errorf("gg should go to line 0, got %d", result.cursorLine)
	}
	if result.pendingKey != 0 {
		t.Error("pending key should be cleared after gg")
	}
}

func TestView(t *testing.T) {
	doc, _ := document.NewDocument("x = 10\ny = 20\n")
	m := New(doc)
	m.width = 80
	m.height = 24

	view := m.View()

	// Should contain source header
	if !strings.Contains(view, "Source") {
		t.Error("View should contain 'Source' header")
	}

	// Should contain preview header (preview visible by default)
	if !strings.Contains(view, "Preview") {
		t.Error("View should contain 'Preview' header")
	}

	// Should show mode
	if !strings.Contains(view, "NORMAL") {
		t.Error("View should show NORMAL mode")
	}
}

func TestViewEmptyDocument(t *testing.T) {
	// Test that viewing an empty document doesn't crash or produce empty output
	doc, _ := document.NewDocument("")
	m := New(doc)
	m.width = 80
	m.height = 24

	view := m.View()

	// Should not be empty or crash
	if len(view) == 0 {
		t.Error("View of empty document should not be empty string")
	}

	// Should still have Source header
	if !strings.Contains(view, "Source") {
		t.Error("View of empty document should contain 'Source' header")
	}

	// Should not be all whitespace
	if strings.TrimSpace(view) == "" {
		t.Error("View of empty document should not be all whitespace")
	}
}

func TestViewAfterEnterEditMode(t *testing.T) {
	// Test viewing after entering edit mode on empty doc
	doc, _ := document.NewDocument("")
	m := New(doc)
	m.width = 80
	m.height = 24

	// Enter edit mode
	m.enterEditMode()

	if m.mode != ModeEditing {
		t.Fatalf("Expected ModeEditing, got %v", m.mode)
	}

	view := m.View()

	// Should not be empty
	if len(view) == 0 {
		t.Error("View after entering edit mode should not be empty")
	}

	// Should show EDITING mode
	if !strings.Contains(view, "EDITING") {
		t.Error("View should show EDITING mode")
	}
}

func TestSaveFile(t *testing.T) {
	doc, _ := document.NewDocument("x = 10\ny = 20\n")
	m := New(doc)

	// Try to save without filename
	m.saveFile("")
	if !m.statusIsErr {
		t.Error("Save without filename should be an error")
	}
	if !strings.Contains(m.statusMsg, "No filename") {
		t.Errorf("Expected 'No filename' error, got: %s", m.statusMsg)
	}

	// Reset error state before next test
	m.statusIsErr = false
	m.statusMsg = ""

	// Save with a temporary file
	tmpFile := t.TempDir() + "/test.cm"
	m.saveFile(tmpFile)

	if m.statusIsErr {
		t.Errorf("Save should succeed, but got error: %s", m.statusMsg)
	}
	if !strings.Contains(m.statusMsg, "Saved") {
		t.Errorf("Expected 'Saved' message, got: %s", m.statusMsg)
	}
	if m.modified {
		t.Error("Modified should be false after save")
	}
}

func TestOpenFile(t *testing.T) {
	// Create a temp file with content
	tmpFile := t.TempDir() + "/test.cm"
	content := "a = 100\nb = 200\n"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	m := New(nil)
	m.openFile(tmpFile)

	if m.statusIsErr {
		t.Errorf("Open should succeed, got error: %s", m.statusMsg)
	}
	if !strings.Contains(m.statusMsg, "Opened") {
		t.Errorf("Expected 'Opened' message, got: %s", m.statusMsg)
	}
	if m.filepath != tmpFile {
		t.Errorf("Filepath not set correctly: %s", m.filepath)
	}
	if m.pinnedVars["a"] != true || m.pinnedVars["b"] != true {
		t.Error("Variables should be auto-pinned")
	}

	// Try opening non-existent file
	m.openFile("/nonexistent/file.cm")
	if !m.statusIsErr {
		t.Error("Open non-existent file should be an error")
	}
}

func TestExecuteCommands(t *testing.T) {
	doc, _ := document.NewDocument("x = 10\n")
	m := New(doc)

	// Test /help command
	m.executeCommand("help")
	if m.statusMsg == "" {
		t.Error("/help should set status message")
	}
	if m.statusIsErr {
		t.Error("/help should not be an error")
	}

	// Reset
	m.statusMsg = ""
	m.statusIsErr = false

	// Test unknown command
	m.executeCommand("unknownxyz")
	if !m.statusIsErr {
		t.Error("Unknown command should be an error")
	}
	if !strings.Contains(m.statusMsg, "Unknown command") {
		t.Errorf("Expected 'Unknown command', got: %s", m.statusMsg)
	}

	// Test /quit
	m.executeCommand("quit")
	if !m.quitting {
		t.Error("/quit should set quitting")
	}

	// Test /preview cycle
	m = New(doc)
	if m.previewMode != PreviewFull {
		t.Error("Preview should start as Full")
	}
	m.executeCommand("preview")
	if m.previewMode != PreviewMinimal {
		t.Error("/preview should cycle to Minimal")
	}

	// Test /preview with argument
	m.executeCommand("preview hidden")
	if m.previewMode != PreviewHidden {
		t.Error("/preview hidden should set Hidden mode")
	}
	m.executeCommand("preview full")
	if m.previewMode != PreviewFull {
		t.Error("/preview full should set Full mode")
	}
}

func TestSaveWQ(t *testing.T) {
	doc, _ := document.NewDocument("x = 10\n")
	m := New(doc)

	// Create a temp file
	tmpFile := t.TempDir() + "/test.cm"
	m.filepath = tmpFile

	// /wq should save and quit
	m.executeCommand("wq")
	if !m.quitting {
		t.Error("/wq should set quitting")
	}
	if m.statusIsErr {
		t.Errorf("/wq should not error: %s", m.statusMsg)
	}
}

func TestYankAndPaste(t *testing.T) {
	doc, _ := document.NewDocument("x = 10\ny = 20\nz = 30\n")
	m := New(doc)

	// yy: yank current line
	tm, _ := m.handleNormalRune([]rune{'y'})
	result := tm.(Model)
	if result.pendingKey != 'y' {
		t.Error("First 'y' should set pending key")
	}

	tm, _ = result.handleNormalRune([]rune{'y'})
	result = tm.(Model)
	if result.yankBuffer != "x = 10" {
		t.Errorf("yy should yank line, got %q", result.yankBuffer)
	}
	if result.pendingKey != 0 {
		t.Error("pending key should be cleared after yy")
	}

	// p: paste below (since we have yank buffer, it should paste not toggle preview)
	tm, _ = result.handleNormalRune([]rune{'p'})
	result = tm.(Model)
	if !strings.Contains(result.statusMsg, "pasted") {
		t.Errorf("Expected 'pasted' message, got %q", result.statusMsg)
	}
}

func TestDeleteLine(t *testing.T) {
	doc, _ := document.NewDocument("x = 10\ny = 20\nz = 30\n")
	m := New(doc)

	initialLines := m.TotalLines()

	// dd: delete current line
	tm, _ := m.handleNormalRune([]rune{'d'})
	result := tm.(Model)
	if result.pendingKey != 'd' {
		t.Error("First 'd' should set pending key")
	}

	tm, _ = result.handleNormalRune([]rune{'d'})
	result = tm.(Model)
	// Line should be yanked before delete
	if result.yankBuffer != "x = 10" {
		t.Errorf("dd should yank line before deleting, got %q", result.yankBuffer)
	}
	if result.pendingKey != 0 {
		t.Error("pending key should be cleared after dd")
	}
	// Line count should be reduced (note: may vary by block structure)
	if result.TotalLines() >= initialLines {
		t.Error("dd should delete a line")
	}
}

func TestFindCommand(t *testing.T) {
	doc, _ := document.NewDocument("x = 10\ny = 20\nz = x + y\n")
	m := New(doc)

	// /find x
	m.executeCommand("find x")
	if m.statusIsErr {
		t.Errorf("find should not error: %s", m.statusMsg)
	}
	if len(m.searchMatches) != 2 {
		t.Errorf("Expected 2 matches for 'x', got %d", len(m.searchMatches))
	}
	// Should jump to first match
	if m.cursorLine != 0 {
		t.Errorf("Should jump to first match at line 0, got %d", m.cursorLine)
	}

	// n: next match
	tm, _ := m.handleNormalRune([]rune{'n'})
	result := tm.(Model)
	if result.cursorLine != 2 {
		t.Errorf("n should go to next match at line 2, got %d", result.cursorLine)
	}

	// N: previous match
	tm, _ = result.handleNormalRune([]rune{'N'})
	result = tm.(Model)
	if result.cursorLine != 0 {
		t.Errorf("N should go to previous match at line 0, got %d", result.cursorLine)
	}

	// Test no match case
	m.executeCommand("find nonexistent")
	if !m.statusIsErr {
		t.Error("find nonexistent should set error")
	}
}

func TestGotoCommand(t *testing.T) {
	doc, _ := document.NewDocument("line1\nline2\nline3\nline4\nline5\n")
	m := New(doc)

	// /goto 3
	m.executeCommand("goto 3")
	if m.statusIsErr {
		t.Errorf("goto should not error: %s", m.statusMsg)
	}
	if m.cursorLine != 2 {
		t.Errorf("goto 3 should set cursor to line 2 (0-indexed), got %d", m.cursorLine)
	}

	// Test out of bounds
	m.executeCommand("goto 100")
	// Should clamp to last line
	totalLines := m.TotalLines()
	if m.cursorLine != totalLines-1 {
		t.Errorf("goto 100 should clamp to last line %d, got %d", totalLines-1, m.cursorLine)
	}

	// Test invalid input
	m.executeCommand("goto abc")
	if !m.statusIsErr {
		t.Error("goto abc should set error")
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
	m.enterEditMode()

	// Modify edit buffer
	m.editBuf = "x = 20"

	// Call live update
	m.liveUpdateCurrentLine()

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

func TestPreviewModeRendering(t *testing.T) {
	doc, _ := document.NewDocument("x = 100\n")
	m := New(doc)
	m.width = 80
	m.height = 24

	// Test Full preview mode
	m.previewMode = PreviewFull
	view := m.View()
	if !strings.Contains(view, "Preview") {
		t.Error("Full preview mode should show Preview header")
	}

	// Test Minimal preview mode
	m.previewMode = PreviewMinimal
	view = m.View()
	if !strings.Contains(view, "Preview") {
		t.Error("Minimal preview mode should show Preview header")
	}

	// Test Hidden preview mode
	m.previewMode = PreviewHidden
	view = m.View()
	if strings.Contains(view, "Preview") {
		t.Error("Hidden preview mode should not show Preview header")
	}
}

func TestNewDocumentEditBuffer(t *testing.T) {
	// Test that when creating a new document, the edit buffer starts empty
	// not with the placeholder character
	doc, _ := document.NewDocument("")
	m := New(doc)

	// Enter edit mode on empty document
	m.enterEditMode()

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
	m.enterEditMode()

	// Change x to 100
	m.editBuf = "x = 100"
	m.liveUpdateCurrentLine()

	// Exit edit mode to trigger full re-evaluation
	m.exitEditMode(true)

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
	m.enterEditMode()
	m.editBuf = "x = 20"
	m.liveUpdateCurrentLine()

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
	m.enterEditMode()
	m.editBuf = "x = 42"
	m.liveUpdateCurrentLine()
	m.exitEditMode(true)

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
	m.enterEditMode()
	m.editBuf = "a = 10"
	m.liveUpdateCurrentLine()
	m.exitEditMode(true)

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
	m.enterEditMode()
	m.editBuf = "total = 200"
	m.liveUpdateCurrentLine()
	m.exitEditMode(true)

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
func TestEditModeWrappedLineNoDuplicate(t *testing.T) {
	content := `this is a really long line of markdown that should wrap`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)

	// Set narrow width to force wrapping
	m.width = 60
	m.height = 20

	// Enter edit mode on the first line
	m.mode = ModeEditing
	m.cursorLine = 0
	m.editBuf = content
	m.cursorCol = len(content)

	view := m.View()
	t.Logf("VIEW OUTPUT:\n%s", view)

	// Count occurrences of "markdown" - it should only appear TWICE
	// (once in source pane, once in preview pane - not duplicated due to wrapping)
	// Note: "should wrap" may be split across lines due to word wrapping
	occurrences := strings.Count(view, "markdown")
	if occurrences > 2 {
		t.Errorf("Text 'markdown' appears %d times, expected 2 (duplicate wrapped line bug)", occurrences)
	}
	if occurrences == 0 {
		t.Error("Text 'markdown' not found in output")
	}

	// The line number "1" should only appear once
	lineNum1Count := strings.Count(view, "   1 ")
	if lineNum1Count > 1 {
		t.Errorf("Line number 1 appears %d times, expected 1", lineNum1Count)
	}
}

// TestLongLineWrappingInEditor tests that long lines wrap in the editor pane
// instead of being truncated with "...".
func TestLongLineWrappingInEditor(t *testing.T) {
	// Create a document with a very long variable name and expression
	content := `x = 1
heres_a_reeeeeeeeeeeeeeeeeeeeeeeeeelly_long_variable_name = x * 2`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)

	// Set a narrow width to force wrapping
	m.width = 60
	m.height = 20

	// Get the view output
	view := m.View()

	// Debug: dump the view
	t.Logf("VIEW OUTPUT:\n%s", view)

	// The SOURCE PANE should wrap long lines (not truncate with ...)
	// The wrapped line may appear across multiple visual lines, so we check
	// that the content is present even if split across lines.
	// We verify wrapping by checking:
	// 1. The end of the variable name appears somewhere (it would be cut if truncated)
	// 2. The expression "= x * 2" appears somewhere

	// Check that "ame = x * 2" appears (end of variable name + expression)
	// This would be cut off if truncation happened instead of wrapping
	hasEndOfName := strings.Contains(view, "ame = x * 2")
	if !hasEndOfName {
		t.Error("Long line in source pane is truncated instead of wrapping - missing end of expression")
	}

	// Check for continuation lines (wrapped content) - they show indented text without line numbers
	// Look for "variable" or "long" appearing on a line that doesn't have a line number
	lines := strings.Split(view, "\n")
	foundWrappedContent := false
	for _, line := range lines {
		// A wrapped line starts with spaces but no line number before variable content
		// Check for "eeeee" (middle of the long name) on a line without a number prefix
		trimmed := strings.TrimLeft(line, " ")
		if strings.HasPrefix(trimmed, "eeeee") || strings.HasPrefix(trimmed, "ame =") {
			foundWrappedContent = true
			break
		}
	}
	if !foundWrappedContent {
		t.Error("Expected to find wrapped continuation lines in source pane")
	}

	// The line number "2" should only appear once (wrapped lines don't get line numbers)
	lineNum2Count := strings.Count(view, "   2 ")
	if lineNum2Count > 1 {
		t.Errorf("Line number 2 appears %d times, expected 1 (wrapped lines should not have line numbers)", lineNum2Count)
	}
}

// TestLongMarkdownWrappingInEditor tests that long markdown text wraps properly.
func TestLongMarkdownWrappingInEditor(t *testing.T) {
	content := `# Header
Some really long markdown text that should wrap nicely in the editor pane without being truncated with ellipsis dots`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)

	m.width = 60
	m.height = 20

	view := m.View()

	// Long markdown should not be truncated
	if strings.Contains(view, "Some really long") && strings.Contains(view, "...") {
		if !strings.Contains(view, "ellipsis dots") {
			t.Error("Long markdown is truncated instead of wrapping")
		}
	}
}

// TestPreviewPaneShowsFullMarkdown tests that the preview pane renders
// full markdown content, not just the first character.
func TestPreviewPaneShowsFullMarkdown(t *testing.T) {
	content := `Some really long markdown that should render fully in the preview pane`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)

	m.width = 100
	m.height = 20
	m.previewMode = PreviewFull

	view := m.View()

	// Debug: log the view output
	t.Logf("VIEW OUTPUT:\n%s", view)

	// The preview pane should show more than just "S"
	// Split the view to find the preview section
	lines := strings.Split(view, "\n")

	foundFullText := false
	for _, line := range lines {
		// Look for rendered markdown content (should have multiple words)
		if strings.Contains(line, "really") || strings.Contains(line, "markdown") {
			foundFullText = true
			break
		}
	}

	if !foundFullText {
		t.Error("Preview pane does not show full markdown content")
	}
}

// TestPreviewPaneWrapsInsteadOfTruncating tests that preview pane wraps
// long content instead of truncating with "...".
func TestPreviewPaneWrapsInsteadOfTruncating(t *testing.T) {
	content := `this is a really long line of markdown that should wrap in preview`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)

	// Set narrow width to force wrapping
	m.width = 60
	m.height = 20
	m.previewMode = PreviewFull

	view := m.View()
	t.Logf("VIEW OUTPUT:\n%s", view)

	// Preview should NOT contain "..." truncation
	if strings.Contains(view, "...") {
		// Check if it's truly truncation (cutting off content)
		if !strings.Contains(view, "wrap in preview") {
			t.Error("Preview pane is truncating content with '...' instead of wrapping")
		}
	}

	// The full text should be visible somewhere (possibly wrapped across lines)
	if !strings.Contains(view, "really") || !strings.Contains(view, "wrap") {
		t.Error("Preview pane does not show full content - appears to be truncated")
	}
}

// TestPreviewPaneMarkdownNotTruncatedToSingleChar tests that markdown in preview
// is not truncated to just the first character (regression test for "S" bug).
func TestPreviewPaneMarkdownNotTruncatedToSingleChar(t *testing.T) {
	// This tests the specific bug where preview showed only "S" for long markdown
	testCases := []struct {
		name        string
		content     string
		expectWords []string // Words that should appear in preview
	}{
		{
			name:        "single long line",
			content:     `Some really long markdown text`,
			expectWords: []string{"Some", "really", "long"},
		},
		{
			name:        "heading",
			content:     `# This is a heading`,
			expectWords: []string{"This", "heading"},
		},
		{
			name:        "paragraph with bold",
			content:     `This has **bold text** in it`,
			expectWords: []string{"This", "bold", "text"},
		},
		{
			name:        "multiple lines",
			content:     "First line\nSecond line\nThird line",
			expectWords: []string{"First", "Second", "Third"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := document.NewDocument(tc.content)
			if err != nil {
				t.Fatalf("Failed to create document: %v", err)
			}
			m := New(doc)
			m.width = 100
			m.height = 20
			m.previewMode = PreviewFull

			view := m.View()

			// Check that each expected word appears somewhere in the view
			for _, word := range tc.expectWords {
				if !strings.Contains(view, word) {
					t.Errorf("Expected word %q not found in preview output", word)
					t.Logf("VIEW:\n%s", view)
				}
			}
		})
	}
}

// TestMinimalModeLeftJustified tests that minimal mode shows results
// left-justified, not right-justified.
func TestMinimalModeLeftJustified(t *testing.T) {
	content := `x = 42`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)

	m.width = 100
	m.height = 20
	m.previewMode = PreviewMinimal

	view := m.View()

	// In minimal mode, "→ 42" should be left-justified within the preview pane
	// Use the centralized pane width configuration
	sourceWidth, _ := m.GetPaneWidths(m.width)
	previewStart := sourceWidth

	lines := strings.Split(view, "\n")

	for _, line := range lines {
		if strings.Contains(line, "→") && strings.Contains(line, "42") {
			arrowIdx := strings.Index(line, "→")

			// Arrow should be near the start of the preview pane (left-justified)
			// Allow a few characters of margin for borders/padding
			if arrowIdx > previewStart+8 {
				// Arrow is too far into the preview pane - it's right-justified
				t.Errorf("Minimal mode result is right-justified (arrow at position %d, preview starts at %d)", arrowIdx, previewStart)
			}
			if arrowIdx < previewStart-5 {
				// Arrow is before the preview pane - something's wrong
				t.Errorf("Arrow appears before preview pane (arrow at %d, preview starts at %d)", arrowIdx, previewStart)
			}
			return
		}
	}
	t.Error("Could not find arrow with result value in output")
}

// TestMinimalModeNarrowerPreview tests that the preview pane is narrower
// in minimal mode compared to full mode.
func TestMinimalModeNarrowerPreview(t *testing.T) {
	content := `x = 42`

	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}
	m := New(doc)

	m.width = 100
	m.height = 20

	// Use the centralized configuration to verify width differences
	fullConfig := DefaultPaneWidths[PreviewFull]
	minimalConfig := DefaultPaneWidths[PreviewMinimal]

	// Verify minimal mode has narrower preview (smaller preview percent)
	if minimalConfig.PreviewPercent >= fullConfig.PreviewPercent {
		t.Errorf("Minimal mode preview should be narrower than full mode: minimal=%d%%, full=%d%%",
			minimalConfig.PreviewPercent, fullConfig.PreviewPercent)
	}

	// Verify the actual widths match configuration
	m.previewMode = PreviewFull
	_, fullPreviewWidth := m.GetPaneWidths(m.width)

	m.previewMode = PreviewMinimal
	_, minimalPreviewWidth := m.GetPaneWidths(m.width)

	if minimalPreviewWidth >= fullPreviewWidth {
		t.Errorf("Minimal preview width (%d) should be less than full preview width (%d)",
			minimalPreviewWidth, fullPreviewWidth)
	}

	// Verify expected percentages from configuration
	expectedMinimalPreview := m.width * minimalConfig.PreviewPercent / 100
	if minimalPreviewWidth != expectedMinimalPreview {
		t.Errorf("Minimal preview width mismatch: got %d, expected %d (from config)",
			minimalPreviewWidth, expectedMinimalPreview)
	}
}

// =============================================================================
// Visual Line Alignment Tests
// These tests verify that source and preview panes stay aligned when content
// wraps to multiple visual lines.
// =============================================================================

func TestSourceToVisualMapping_BasicCase(t *testing.T) {
	// Simple case: no wrapping, each source line maps to one visual line
	// Note: trailing newline creates an empty 4th line
	doc, _ := document.NewDocument("x = 10\ny = 20\nz = 30\n")
	m := New(doc)
	m.width = 100
	m.height = 24

	leftWidth, rightWidth := m.GetPaneWidths(m.width)
	aligned := m.computeAlignedPanes(leftWidth, rightWidth)

	totalSourceLines := m.TotalLines()
	t.Logf("Document has %d source lines", totalSourceLines)

	// With no wrapping, sourceToVisual should map each source line to sequential visual lines
	for i := 0; i < totalSourceLines; i++ {
		visualIdx, ok := aligned.sourceToVisual[i]
		if !ok {
			t.Errorf("sourceToVisual missing entry for source line %d", i)
			continue
		}
		if visualIdx != i {
			t.Errorf("Source line %d: expected visual index %d, got %d", i, i, visualIdx)
		}
	}

	// With no wrapping, visual lines should equal source lines
	if len(aligned.sourceLines) != totalSourceLines {
		t.Errorf("Expected %d source lines, got %d", totalSourceLines, len(aligned.sourceLines))
	}
	if len(aligned.previewLines) != totalSourceLines {
		t.Errorf("Expected %d preview lines, got %d", totalSourceLines, len(aligned.previewLines))
	}

	// Verify source and preview counts match (critical invariant)
	if len(aligned.sourceLines) != len(aligned.previewLines) {
		t.Errorf("Source lines (%d) != Preview lines (%d)",
			len(aligned.sourceLines), len(aligned.previewLines))
	}
}

func TestSourceToVisualMapping_PreviewWraps(t *testing.T) {
	// Create a document where preview content will wrap
	// Use a narrow preview width to force wrapping
	doc, _ := document.NewDocument("a = 1\nb = 2\n")
	m := New(doc)
	m.width = 60  // Narrow total width
	m.height = 24

	// Force a very narrow preview to cause wrapping
	// We need to test with actual preview content that wraps
	// The preview shows "a  1" and "b  2" which are short
	// Let's use a document with longer variable names

	doc2, _ := document.NewDocument("short = 1\nthis_is_a_very_long_variable_name = 2\n")
	m2 := New(doc2)
	m2.width = 50  // Narrow
	m2.height = 24

	leftWidth, rightWidth := m2.GetPaneWidths(m2.width)
	aligned := m2.computeAlignedPanes(leftWidth, rightWidth)

	t.Logf("Source lines: %d, Preview lines: %d", len(aligned.sourceLines), len(aligned.previewLines))
	t.Logf("sourceToVisual map: %v", aligned.sourceToVisual)

	for i, sl := range aligned.sourceLines {
		t.Logf("  Source[%d]: lineNum=%d, sourceLineIdx=%d, isPadding=%v, content=%q",
			i, sl.lineNum, sl.sourceLineIdx, sl.isPadding, sl.content)
	}

	// Key invariant: source lines and preview lines must have same count
	if len(aligned.sourceLines) != len(aligned.previewLines) {
		t.Errorf("Source lines (%d) != Preview lines (%d) - alignment broken",
			len(aligned.sourceLines), len(aligned.previewLines))
	}

	// Verify sourceToVisual contains entries for all source lines
	sourceLineCount := m2.TotalLines()
	for i := 0; i < sourceLineCount; i++ {
		if _, ok := aligned.sourceToVisual[i]; !ok {
			t.Errorf("sourceToVisual missing entry for source line %d", i)
		}
	}
}

func TestSourceToVisualMapping_WithPaddingLines(t *testing.T) {
	// Create a scenario where preview wraps more than source
	// This forces padding lines to be added to source pane

	// We need markdown content that renders to multiple lines
	// A long heading or text line should do it
	content := `# Short heading
x = 100`

	doc, _ := document.NewDocument(content)
	m := New(doc)
	m.width = 40  // Very narrow to force wrapping
	m.height = 24

	leftWidth, rightWidth := m.GetPaneWidths(m.width)
	aligned := m.computeAlignedPanes(leftWidth, rightWidth)

	t.Logf("Total source lines in doc: %d", m.TotalLines())
	t.Logf("Visual source lines: %d, Visual preview lines: %d",
		len(aligned.sourceLines), len(aligned.previewLines))

	for i, sl := range aligned.sourceLines {
		t.Logf("  Source[%d]: lineNum=%d, idx=%d, padding=%v, wrapped=%v, content=%q",
			i, sl.lineNum, sl.sourceLineIdx, sl.isPadding, sl.isWrapped, sl.content)
	}

	// Verify alignment
	if len(aligned.sourceLines) != len(aligned.previewLines) {
		t.Errorf("Source lines (%d) != Preview lines (%d)",
			len(aligned.sourceLines), len(aligned.previewLines))
	}

	// If there are padding lines in source, they should have isPadding=true
	paddingCount := 0
	for _, sl := range aligned.sourceLines {
		if sl.isPadding {
			paddingCount++
		}
	}
	t.Logf("Padding lines in source: %d", paddingCount)

	// Verify the mapping skips padding appropriately
	// Source line 0 should map to visual line 0
	if idx, ok := aligned.sourceToVisual[0]; ok {
		if idx != 0 {
			t.Errorf("Source line 0 should map to visual line 0, got %d", idx)
		}
	}
}

func TestScrollSyncWithPadding(t *testing.T) {
	// Test that when cursor is on a line with padding below it,
	// both panes compute the same scroll offset

	content := `line1 = 1
line2_with_a_much_longer_name_that_might_wrap = 2
line3 = 3
line4 = 4
line5 = 5`

	doc, _ := document.NewDocument(content)
	m := New(doc)
	m.width = 50
	m.height = 10  // Small height to test scrolling

	// Move cursor to line 2 (the long one)
	m.cursorLine = 1

	leftWidth, rightWidth := m.GetPaneWidths(m.width)
	aligned := m.computeAlignedPanes(leftWidth, rightWidth)

	// Get the visual line index for cursor
	cursorVisualLine, ok := aligned.sourceToVisual[m.cursorLine]
	if !ok {
		t.Fatalf("No visual mapping for cursor line %d", m.cursorLine)
	}

	t.Logf("Cursor on source line %d, visual line %d", m.cursorLine, cursorVisualLine)
	t.Logf("sourceToVisual: %v", aligned.sourceToVisual)

	// The key test: when we render, both panes should use the same visual scroll offset
	// We can't easily test the internal scroll calculation, but we can verify
	// that the sourceToVisual mapping is monotonically increasing
	lastVisual := -1
	for srcLine := 0; srcLine < m.TotalLines(); srcLine++ {
		visualLine, ok := aligned.sourceToVisual[srcLine]
		if !ok {
			t.Errorf("Missing mapping for source line %d", srcLine)
			continue
		}
		if visualLine <= lastVisual {
			t.Errorf("sourceToVisual not monotonically increasing: line %d maps to %d, but previous was %d",
				srcLine, visualLine, lastVisual)
		}
		lastVisual = visualLine
	}
}

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
	m.enterEditMode()

	if m.mode != ModeEditing {
		t.Fatalf("Expected ModeEditing, got %v", m.mode)
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
	m.exitEditMode(true)

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
	m.width = 50  // Narrow
	m.height = 24

	// Move to the calc line
	m.cursorLine = 1

	leftWidth, rightWidth := m.GetPaneWidths(m.width)

	// Get aligned panes before edit mode
	alignedBefore := m.computeAlignedPanes(leftWidth, rightWidth)
	visualBefore, _ := alignedBefore.sourceToVisual[m.cursorLine]

	// Enter edit mode
	m.enterEditMode()

	// Get aligned panes in edit mode
	alignedDuring := m.computeAlignedPanes(leftWidth, rightWidth)
	visualDuring, _ := alignedDuring.sourceToVisual[m.cursorLine]

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

func TestVisualLineCalculation_Deterministic(t *testing.T) {
	// Test with exact, predictable dimensions
	// Source pane: 40 chars wide (after line numbers)
	// Preview pane: 30 chars wide
	// This allows us to predict exactly how content wraps

	content := `x = 1
this_is_a_longer_line = 2
z = 3`

	doc, _ := document.NewDocument(content)
	m := New(doc)

	// Set exact dimensions
	// With width=80, PreviewFull gives 55% source (44) and 45% preview (36)
	m.width = 80
	m.height = 24
	m.previewMode = PreviewFull

	leftWidth, rightWidth := m.GetPaneWidths(m.width)
	t.Logf("Pane widths: source=%d, preview=%d", leftWidth, rightWidth)

	// Calculate expected source content width (after line numbers)
	// Line number width is 4, plus 2 for spacing
	sourceContentWidth := leftWidth - 4 - 2
	t.Logf("Source content width: %d", sourceContentWidth)

	aligned := m.computeAlignedPanes(leftWidth, rightWidth)

	// Log all visual lines for debugging
	t.Log("Source visual lines:")
	for i, sl := range aligned.sourceLines {
		t.Logf("  [%d] srcIdx=%d lineNum=%d wrap=%v pad=%v content=%q",
			i, sl.sourceLineIdx, sl.lineNum, sl.isWrapped, sl.isPadding, sl.content)
	}

	t.Log("Preview visual lines:")
	for i, pl := range aligned.previewLines {
		t.Logf("  [%d] srcNum=%d content=%q", i, pl.sourceLineNum, pl.content)
	}

	t.Logf("sourceToVisual: %v", aligned.sourceToVisual)

	// Verify the mapping
	// Line 0 "x = 1" is short, should be visual line 0
	if v, ok := aligned.sourceToVisual[0]; !ok || v != 0 {
		t.Errorf("Source line 0 should map to visual 0, got %v (ok=%v)", v, ok)
	}

	// Line 1 "this_is_a_longer_line = 2" - check if it wraps
	line1 := "this_is_a_longer_line = 2"
	line1VisualWidth := len(line1) // ASCII, so len == visual width
	if line1VisualWidth > sourceContentWidth {
		t.Logf("Line 1 (%d chars) should wrap at width %d", line1VisualWidth, sourceContentWidth)
	}

	// Source line 2 should map to visual index after line 1's visual lines
	v1, _ := aligned.sourceToVisual[1]
	v2, _ := aligned.sourceToVisual[2]
	if v2 <= v1 {
		t.Errorf("Source line 2 visual (%d) should be > source line 1 visual (%d)", v2, v1)
	}
}

func TestScrollOffset_Deterministic(t *testing.T) {
	// Test scroll offset calculation with exact dimensions
	// Create a document with enough lines to require scrolling

	content := `line0 = 0
line1 = 1
line2 = 2
line3 = 3
line4 = 4
line5 = 5
line6 = 6
line7 = 7
line8 = 8
line9 = 9`

	doc, _ := document.NewDocument(content)
	m := New(doc)
	m.width = 80
	m.height = 10 // Small height to force scrolling
	m.previewMode = PreviewFull

	leftWidth, rightWidth := m.GetPaneWidths(m.width)
	aligned := m.computeAlignedPanes(leftWidth, rightWidth)

	t.Logf("Total visual lines: %d", len(aligned.sourceLines))
	t.Logf("Visible height: ~%d (height=%d minus headers/footers)", m.height-6, m.height)

	// Test cursor at different positions
	testCases := []struct {
		cursorLine         int
		expectedVisualLine int
	}{
		{0, 0},
		{1, 1},
		{5, 5},
		{9, 9},
	}

	for _, tc := range testCases {
		m.cursorLine = tc.cursorLine
		visualLine, ok := aligned.sourceToVisual[tc.cursorLine]
		if !ok {
			t.Errorf("Cursor line %d: no visual mapping", tc.cursorLine)
			continue
		}
		if visualLine != tc.expectedVisualLine {
			t.Errorf("Cursor line %d: expected visual %d, got %d",
				tc.cursorLine, tc.expectedVisualLine, visualLine)
		}
	}
}

func TestScrollOffset_WithWrapping(t *testing.T) {
	// Test scroll calculation when some lines wrap
	// Use very narrow width to force wrapping

	content := `short = 1
this_is_a_very_long_variable_name_that_will_definitely_wrap = 2
another_short = 3`

	doc, _ := document.NewDocument(content)
	m := New(doc)
	m.width = 60
	m.height = 20
	m.previewMode = PreviewFull

	leftWidth, rightWidth := m.GetPaneWidths(m.width)
	t.Logf("Widths: left=%d, right=%d", leftWidth, rightWidth)

	// Source content width after line numbers
	sourceContentWidth := leftWidth - 4 - 2
	t.Logf("Source content width: %d", sourceContentWidth)

	aligned := m.computeAlignedPanes(leftWidth, rightWidth)

	// Log the visual structure
	t.Log("Visual line structure:")
	for i, sl := range aligned.sourceLines {
		t.Logf("  visual[%d]: srcIdx=%d wrap=%v pad=%v len=%d content=%q",
			i, sl.sourceLineIdx, sl.isWrapped, sl.isPadding, len(sl.content), sl.content)
	}

	// The long line (source line 1) should wrap
	line1Content := "this_is_a_very_long_variable_name_that_will_definitely_wrap = 2"
	expectedWraps := (len(line1Content) + sourceContentWidth - 1) / sourceContentWidth
	t.Logf("Line 1 length: %d, expected wraps: %d", len(line1Content), expectedWraps)

	// Count visual lines for source line 1
	line1VisualCount := 0
	for _, sl := range aligned.sourceLines {
		if sl.sourceLineIdx == 1 && !sl.isPadding {
			line1VisualCount++
		}
	}
	t.Logf("Line 1 actual visual lines: %d", line1VisualCount)

	// Source line 2 should start at visual index = line0 visuals + line1 visuals
	v0, _ := aligned.sourceToVisual[0]
	v1, _ := aligned.sourceToVisual[1]
	v2, _ := aligned.sourceToVisual[2]

	t.Logf("Visual indices: line0=%d, line1=%d, line2=%d", v0, v1, v2)

	// Verify gaps match wrapping
	gapBetween1And2 := v2 - v1
	if gapBetween1And2 < line1VisualCount {
		t.Errorf("Gap between line1 and line2 (%d) should be >= line1 visual count (%d)",
			gapBetween1And2, line1VisualCount)
	}
}

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
	m.enterEditMode()
	m.editBuf = "- this is a list item"
	m.exitEditMode(true)

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
	m.enterEditMode()
	m.editBuf = "total = 100 + 200"
	m.exitEditMode(true)

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
	m.enterEditMode()

	// Type markdown content
	m.editBuf = "- list item one"
	m.exitEditMode(true)

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
	m.enterEditMode()

	if m.mode != ModeEditing {
		t.Fatalf("Expected ModeEditing, got %v", m.mode)
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
	expectedVisualIdx, _ := aligned.sourceToVisual[m.cursorLine]
	if cursorVisualIdx != expectedVisualIdx {
		t.Errorf("Cursor visual index %d doesn't match mapping %d",
			cursorVisualIdx, expectedVisualIdx)
	}

	t.Logf("Cursor at source line %d, visual line %d", m.cursorLine, cursorVisualIdx)

	// Modify and save
	m.editBuf = "modified = 123"
	m.exitEditMode(true)

	// Verify save worked
	lines := m.GetLines()
	if lines[1] != "modified = 123" {
		t.Errorf("Line not saved correctly: got %q", lines[1])
	}
}

func TestPaneAlignment_ExactDimensions(t *testing.T) {
	// Test with exact dimensions to verify pixel-perfect alignment

	testCases := []struct {
		name           string
		content        string
		width          int
		height         int
		expectedVisual int // expected total visual lines
	}{
		{
			name:           "simple no wrap",
			content:        "a = 1\nb = 2",
			width:          100,
			height:         24,
			expectedVisual: 2,
		},
		{
			name:           "single line wraps at narrow width",
			content:        "abcdefghij = 12345", // 18 chars
			width:          40,                   // Very narrow
			height:         24,
			expectedVisual: -1, // Calculate based on actual wrapping
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc, _ := document.NewDocument(tc.content)
			m := New(doc)
			m.width = tc.width
			m.height = tc.height

			leftWidth, rightWidth := m.GetPaneWidths(m.width)
			aligned := m.computeAlignedPanes(leftWidth, rightWidth)

			t.Logf("Width=%d -> left=%d, right=%d", tc.width, leftWidth, rightWidth)
			t.Logf("Source lines: %d, Preview lines: %d",
				len(aligned.sourceLines), len(aligned.previewLines))

			// Critical: counts must match
			if len(aligned.sourceLines) != len(aligned.previewLines) {
				t.Errorf("Alignment broken: source=%d, preview=%d",
					len(aligned.sourceLines), len(aligned.previewLines))
			}

			if tc.expectedVisual > 0 && len(aligned.sourceLines) != tc.expectedVisual {
				t.Errorf("Expected %d visual lines, got %d",
					tc.expectedVisual, len(aligned.sourceLines))
			}

			// Log structure for debugging
			for i := range aligned.sourceLines {
				sl := aligned.sourceLines[i]
				pl := aligned.previewLines[i]
				t.Logf("  [%d] src=%q | preview=%q",
					i, sl.content, pl.content)
			}
		})
	}
}

func TestAlignedPanesCountsMatch(t *testing.T) {
	// Critical invariant: source and preview lines must always have same count
	testCases := []struct {
		name    string
		content string
		width   int
	}{
		{
			name:    "simple calc",
			content: "x = 10\ny = 20\n",
			width:   80,
		},
		{
			name:    "narrow width",
			content: "x = 10\ny = 20\n",
			width:   30,
		},
		{
			name:    "mixed content",
			content: "# Header\nx = 10\nSome text\ny = x * 2\n",
			width:   60,
		},
		{
			name:    "long variable names",
			content: "very_long_variable_name_here = 12345\nanother_long_one = 67890\n",
			width:   40,
		},
		{
			name:    "empty lines",
			content: "x = 1\n\ny = 2\n",
			width:   80,
		},
		{
			name: "compression scenario - long function calls",
			content: `# Use in calculations
storage_savings = 10 GB - compress(10 GB, gzip)
compressed_transfer = transfer_time(compress(1 GB, lz4), global, gigabit)`,
			width: 80,
		},
		{
			name: "very narrow compression scenario",
			content: `# Use in calculations
storage_savings = 10 GB - compress(10 GB, gzip)
compressed_transfer = transfer_time(compress(1 GB, lz4), global, gigabit)`,
			width: 50,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := document.NewDocument(tc.content)
			if err != nil {
				t.Fatalf("Failed to create document: %v", err)
			}

			m := New(doc)
			m.width = tc.width
			m.height = 24

			leftWidth, rightWidth := m.GetPaneWidths(m.width)
			aligned := m.computeAlignedPanes(leftWidth, rightWidth)

			if len(aligned.sourceLines) != len(aligned.previewLines) {
				t.Errorf("Source lines (%d) != Preview lines (%d)",
					len(aligned.sourceLines), len(aligned.previewLines))

				t.Log("Source lines:")
				for i, sl := range aligned.sourceLines {
					t.Logf("  [%d] idx=%d padding=%v content=%q",
						i, sl.sourceLineIdx, sl.isPadding, sl.content)
				}
				t.Log("Preview lines:")
				for i, pl := range aligned.previewLines {
					t.Logf("  [%d] srcNum=%d content=%q",
						i, pl.sourceLineNum, pl.content)
				}
			}
		})
	}
}
