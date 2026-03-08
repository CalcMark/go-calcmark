package editor

// file_operations_test.go — Save, open, export, file picker, unsaved changes.

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/CalcMark/go-calcmark/format"
	"github.com/CalcMark/go-calcmark/format/display"
	"github.com/CalcMark/go-calcmark/spec/document"
)

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

// TestOpenFileWithParseErrorsShowsParseError verifies that .cm files with
// invalid content show parse errors, not extension errors.
func TestOpenFileWithParseErrorsShowsParseError(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "bad.cm")
	// YAML frontmatter with unknown key triggers parse error
	content := "---\ntitle: bad\n---\nx = 1\n"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	m := New(nil)
	m.openFile(tmpFile)

	if !m.statusIsErr {
		t.Fatal("expected parse error for .cm file with bad frontmatter")
	}
	if !strings.Contains(m.statusMsg, "Parse error") {
		t.Errorf("expected 'Parse error' prefix, got: %s", m.statusMsg)
	}
}

// Note: TestExecuteCommands and TestSaveWQ were removed along with
// the executeCommand() dead code. The editor uses keyboard accelerators
// (Ctrl+S, Ctrl+E, etc.) and the command menu (Ctrl+H) instead.

func TestExportFile(t *testing.T) {
	doc, _ := document.NewDocument("x = 100\ny = 200\n")
	m := New(doc)

	// Test export to text format
	tmpDir := t.TempDir()
	textFile := tmpDir + "/test"
	m.exportFile(textFile, "text")

	if m.statusIsErr {
		t.Errorf("Export to text failed: %s", m.statusMsg)
	}

	// Check file exists with .txt extension
	if _, err := os.Stat(tmpDir + "/test.txt"); os.IsNotExist(err) {
		t.Error("Expected test.txt to be created")
	}

	// Test export to cm format
	cmFile := tmpDir + "/test_cm"
	m.exportFile(cmFile, "cm")

	if m.statusIsErr {
		t.Errorf("Export to cm failed: %s", m.statusMsg)
	}

	// Check file exists with .cm extension
	if _, err := os.Stat(tmpDir + "/test_cm.cm"); os.IsNotExist(err) {
		t.Error("Expected test_cm.cm to be created")
	}

	// Test export to json format
	jsonFile := tmpDir + "/test_json"
	m.exportFile(jsonFile, "json")

	if m.statusIsErr {
		t.Errorf("Export to json failed: %s", m.statusMsg)
	}

	// Check file exists with .json extension
	if _, err := os.Stat(tmpDir + "/test_json.json"); os.IsNotExist(err) {
		t.Error("Expected test_json.json to be created")
	}
}

// Test export overlay transitions (format selection → file picker)
func TestExportModeTransitions(t *testing.T) {
	m := New(nil)

	// Test entering export mode
	m.enterExportMode()
	if m.mode != StateExport {
		t.Errorf("Expected StateExport, got %v", m.mode)
	}
	if m.exportState.FormatIdx != 0 {
		t.Errorf("Expected initial format index 0, got %d", m.exportState.FormatIdx)
	}

	// Test selecting format with key '1' (text) — transitions to file picker
	newModel, _ := m.handleExportOverlayKey(tea.KeyPressMsg{Code: '1', Text: "1"})
	m = newModel.(Model)

	if m.mode != StateFilePicker {
		t.Errorf("Expected StateFilePicker after selecting format, got %v", m.mode)
	}
	if m.filePickerPurpose != PickerForExport {
		t.Errorf("Expected PickerForExport, got %v", m.filePickerPurpose)
	}
	if m.exportState.FormatIdx != 0 {
		t.Errorf("Expected format index 0 (text), got %d", m.exportState.FormatIdx)
	}

	// Test canceling export
	m = New(nil)
	m.enterExportMode()
	newModel, _ = m.handleExportOverlayKey(tea.KeyPressMsg{Code: tea.KeyEsc})
	m = newModel.(Model)

	if m.mode != StateDefault {
		t.Errorf("Expected StateDefault after cancel, got %v", m.mode)
	}
}

// Test unsaved changes detection
func TestHasUnsavedChanges(t *testing.T) {
	m := New(nil)

	// Initially should not have unsaved changes
	if m.hasUnsavedChanges() {
		t.Error("New document should not have unsaved changes")
	}

	// Make an actual content change
	m.cursorLine = 0
	// User is always able to edit - load editBuf
	m.loadCurrentLineIntoEditBuffer()
	m.editBuf = "x = 100"
	m.saveCurrentLine(true)

	// Should now have unsaved changes
	if !m.hasUnsavedChanges() {
		t.Error("Modified document should have unsaved changes")
	}

	// Save the file
	tmpDir := t.TempDir()
	testFile := tmpDir + "/test.cm"
	m.saveFile(testFile)

	// After save, should not have unsaved changes
	if m.hasUnsavedChanges() {
		t.Error("Saved document should not have unsaved changes")
	}

	// Make another change
	m.cursorLine = 0
	// User is always able to edit - load editBuf
	m.loadCurrentLineIntoEditBuffer()
	m.editBuf = "y = 200"
	m.saveCurrentLine(true)

	// Should have unsaved changes again
	if !m.hasUnsavedChanges() {
		t.Error("Document with new changes should have unsaved changes")
	}
}

// Test save prompt mode
func TestSavePromptMode(t *testing.T) {
	doc, _ := document.NewDocument("x = 100\n")
	m := New(doc)

	// Make an actual edit to trigger unsaved changes
	m.cursorLine = 0
	// User is always able to edit - load editBuf
	m.loadCurrentLineIntoEditBuffer()
	m.editBuf = "y = 200"
	m.saveCurrentLine(true)

	// Trigger quit with unsaved changes using Ctrl+Q
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'q', Mod: tea.ModCtrl})
	m = newModel.(Model)

	if m.mode != StateSavePrompt {
		t.Errorf("Expected StateSavePrompt, got %v", m.mode)
	}

	// Test pressing 'n' (quit without save)
	newModel, cmd := m.handleSavePromptKey(tea.KeyPressMsg{Code: 'n', Text: "n"})
	m = newModel.(Model)

	if !m.quitting {
		t.Error("Expected quitting=true after pressing 'n'")
	}
	if cmd == nil {
		t.Error("Expected tea.Quit command")
	}

	// Test pressing 'c' (cancel)
	doc2, _ := document.NewDocument("x = 100\n")
	m = New(doc2)
	// Make an edit
	m.cursorLine = 0
	// User is always able to edit - load editBuf
	m.loadCurrentLineIntoEditBuffer()
	m.editBuf = "z = 300"
	m.saveCurrentLine(true)
	m.mode = StateSavePrompt

	newModel, _ = m.handleSavePromptKey(tea.KeyPressMsg{Code: 'c', Text: "c"})
	m = newModel.(Model)

	if m.mode != StateDefault {
		t.Errorf("Expected StateDefault after cancel, got %v", m.mode)
	}
	if m.quitting {
		t.Error("Expected quitting=false after cancel")
	}
}

func TestCtrlEDoesNotTriggerExport(t *testing.T) {
	m := New(nil)

	// Ctrl+E was moved to Ctrl+T to prevent Cmd+Right collision in legacy terminals.
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'e', Mod: tea.ModCtrl})
	m = newModel.(Model)

	if m.mode == StateExport {
		t.Errorf("Ctrl+E should NOT trigger Export (moved to Ctrl+T to fix Cmd+Right collision), got %v", m.mode)
	}
}

func TestCtrlTTriggersExport(t *testing.T) {
	m := New(nil)

	newModel, _ := m.Update(tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
	m = newModel.(Model)

	if m.mode != StateExport {
		t.Errorf("Expected StateExport after Ctrl+T, got %v", m.mode)
	}
}

// Test export flow through the command menu
func TestExportFlowThroughUpdate(t *testing.T) {
	doc, _ := document.NewDocument("x = 42\n")
	m := New(doc)

	// Step 1: Enter export via command dispatch (as the command menu does)
	newModel, _ := m.executeCommandByName("Export")
	m = newModel.(Model)

	if m.mode != StateExport {
		t.Fatalf("Step 1: Expected StateExport after Export command, got %v", m.mode)
	}

	// Step 2: Press '1' to select text format → transitions to file picker
	newModel, _ = m.Update(tea.KeyPressMsg{Code: '1', Text: "1"})
	m = newModel.(Model)

	if m.mode != StateFilePicker {
		t.Fatalf("Step 2: Expected StateFilePicker after selecting format, got %v", m.mode)
	}
	if m.filePickerPurpose != PickerForExport {
		t.Fatalf("Step 2: Expected PickerForExport, got %v", m.filePickerPurpose)
	}
	if m.exportFormatOpts[m.exportState.FormatIdx] != "text" {
		t.Fatalf("Step 2: Expected format 'text', got %q", m.exportFormatOpts[m.exportState.FormatIdx])
	}

	// Step 3: Type filename in file picker
	tmpDir := t.TempDir()
	m.filePicker.CurrentDirectory = tmpDir

	for _, ch := range "output" {
		newModel, _ = m.handleFilePickerKey(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		m = newModel.(Model)
	}

	if m.newFileName != "output" {
		t.Fatalf("Step 3: Expected filename 'output', got %q", m.newFileName)
	}

	// Step 4: Enter exports the file
	newModel, _ = m.handleFilePickerKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = newModel.(Model)

	if m.mode != StateDefault {
		t.Fatalf("Step 4: Expected StateDefault after Enter, got %v", m.mode)
	}

	// Verify file was created with .txt extension
	if _, err := os.Stat(filepath.Join(tmpDir, "output.txt")); os.IsNotExist(err) {
		t.Error("Step 4: Export file was not created")
	}
}

// TestExportFlowEscCancel tests that Esc cancels the export overlay.
func TestExportFlowEscCancel(t *testing.T) {
	doc, _ := document.NewDocument("x = 42\n")

	t.Run("cancel at format selection", func(t *testing.T) {
		m := New(doc)
		newModel, _ := m.executeCommandByName("Export")
		m = newModel.(Model)

		newModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
		m = newModel.(Model)

		if m.mode != StateDefault {
			t.Errorf("Expected StateDefault after Esc, got %v", m.mode)
		}
	})

	t.Run("cancel at file picker after format selection", func(t *testing.T) {
		m := New(doc)
		newModel, _ := m.executeCommandByName("Export")
		m = newModel.(Model)

		// Select format → transitions to file picker
		newModel, _ = m.Update(tea.KeyPressMsg{Code: '1', Text: "1"})
		m = newModel.(Model)

		if m.mode != StateFilePicker {
			t.Fatalf("Expected StateFilePicker, got %v", m.mode)
		}

		// Cancel from file picker
		newModel, _ = m.handleFilePickerKey(tea.KeyPressMsg{Code: tea.KeyEsc})
		m = newModel.(Model)

		if m.mode != StateDefault {
			t.Errorf("Expected StateDefault after Esc, got %v", m.mode)
		}
	})
}

// TestAllCommandMenuActionsViaUpdate tests that every command menu action
// correctly transitions state when triggered through the full Update chain.
func TestExportOverlayNumberShortcuts(t *testing.T) {
	m := New(nil)
	m.enterExportMode()

	// Number shortcuts 1-5 select format and open file picker
	for num := 1; num <= len(m.exportFormatOpts); num++ {
		m.enterExportMode() // reset
		key := rune('0' + num)
		newModel, _ := m.handleExportOverlayKey(tea.KeyPressMsg{Code: key, Text: string(key)})
		m = newModel.(Model)

		if m.exportState.FormatIdx != num-1 {
			t.Errorf("Key '%c': expected format index %d, got %d", key, num-1, m.exportState.FormatIdx)
		}
		if m.mode != StateFilePicker {
			t.Errorf("Key '%c': expected StateFilePicker, got %v", key, m.mode)
		}
		if m.filePickerPurpose != PickerForExport {
			t.Errorf("Key '%c': expected PickerForExport, got %v", key, m.filePickerPurpose)
		}
	}
}

// TestExportOverlayFormatNavigation tests up/down navigation of format list.
func TestExportOverlayFormatNavigation(t *testing.T) {
	m := New(nil)
	m.enterExportMode()

	// Initial format is index 0
	if m.exportState.FormatIdx != 0 {
		t.Fatalf("Expected initial format index 0, got %d", m.exportState.FormatIdx)
	}

	// Down moves to next format
	newModel, _ := m.handleExportOverlayKey(tea.KeyPressMsg{Code: tea.KeyDown})
	m = newModel.(Model)
	if m.exportState.FormatIdx != 1 {
		t.Errorf("Expected format index 1 after down, got %d", m.exportState.FormatIdx)
	}

	// Up moves back
	newModel, _ = m.handleExportOverlayKey(tea.KeyPressMsg{Code: tea.KeyUp})
	m = newModel.(Model)
	if m.exportState.FormatIdx != 0 {
		t.Errorf("Expected format index 0 after up, got %d", m.exportState.FormatIdx)
	}

	// Up at top stays at top
	newModel, _ = m.handleExportOverlayKey(tea.KeyPressMsg{Code: tea.KeyUp})
	m = newModel.(Model)
	if m.exportState.FormatIdx != 0 {
		t.Errorf("Expected format index 0 (clamped), got %d", m.exportState.FormatIdx)
	}
}

// TestExportOverlayEnterOnFormat tests Enter on format opens file picker.
func TestExportOverlayEnterOnFormat(t *testing.T) {
	m := New(nil)
	m.enterExportMode()

	// Navigate to index 2 (json)
	newModel, _ := m.handleExportOverlayKey(tea.KeyPressMsg{Code: tea.KeyDown})
	m = newModel.(Model)
	newModel, _ = m.handleExportOverlayKey(tea.KeyPressMsg{Code: tea.KeyDown})
	m = newModel.(Model)

	if m.exportState.FormatIdx != 2 {
		t.Fatalf("Expected format index 2 (json), got %d", m.exportState.FormatIdx)
	}

	// Enter opens file picker with the selected format preserved
	newModel, _ = m.handleExportOverlayKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = newModel.(Model)

	if m.mode != StateFilePicker {
		t.Errorf("Expected StateFilePicker after Enter on format, got %v", m.mode)
	}
	if m.filePickerPurpose != PickerForExport {
		t.Errorf("Expected PickerForExport, got %v", m.filePickerPurpose)
	}
	if m.exportState.FormatIdx != 2 {
		t.Errorf("Expected format index 2 preserved, got %d", m.exportState.FormatIdx)
	}
}

// TestExportViaFilePickerFlow tests the full export-via-file-picker flow.
func TestExportViaFilePickerFlow(t *testing.T) {
	doc, _ := document.NewDocument("x = 42\n")
	m := New(doc)

	// Enter export mode and select HTML format
	m.enterExportMode()
	m.exportState.FormatIdx = 3 // html

	// Open file picker
	newModel, _ := m.handleExportOverlayKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = newModel.(Model)

	// Set directory and type filename
	tmpDir := t.TempDir()
	m.filePicker.CurrentDirectory = tmpDir

	for _, ch := range "report" {
		newModel, _ = m.handleFilePickerKey(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		m = newModel.(Model)
	}

	// Enter exports with correct format extension
	newModel, _ = m.handleFilePickerKey(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = newModel.(Model)

	if m.mode != StateDefault {
		t.Errorf("Expected StateDefault after export, got %v", m.mode)
	}

	// Verify file was created with .html extension
	expectedPath := filepath.Join(tmpDir, "report.html")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected export file at %s", expectedPath)
	}
}

// TestCtrlOOpensFilePicker tests Ctrl+O opens the file picker with PickerForOpen
// and AllowedTypes restricted to CalcMark extensions.
func TestCtrlOOpensFilePicker(t *testing.T) {
	m := New(nil)
	m.width = 80
	m.height = 24

	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'o', Mod: tea.ModCtrl})
	m = newModel.(Model)

	if m.mode != StateFilePicker {
		t.Errorf("Expected StateFilePicker after Ctrl+O, got %v", m.mode)
	}
	if m.filePickerPurpose != PickerForOpen {
		t.Errorf("Expected PickerForOpen, got %v", m.filePickerPurpose)
	}
	if m.filePickerFocus != FocusFileBrowser {
		t.Errorf("Expected FocusFileBrowser, got %v", m.filePickerFocus)
	}

	// File picker should only allow .cm and .calcmark files for open
	allowed := m.filePicker.AllowedTypes
	if len(allowed) != 2 || allowed[0] != ".cm" || allowed[1] != ".calcmark" {
		t.Errorf("Expected AllowedTypes [.cm .calcmark], got %v", allowed)
	}
}

// TestCtrlOWithUnsavedChanges tests Ctrl+O shows save prompt when there are unsaved changes.
func TestCtrlOWithUnsavedChanges(t *testing.T) {
	doc, _ := document.NewDocument("x = 100\n")
	m := New(doc)
	m.width = 80
	m.height = 24

	// Make an edit to trigger unsaved changes
	m.cursorLine = 0
	m.loadCurrentLineIntoEditBuffer()
	m.editBuf = "y = 200"
	m.saveCurrentLine(true)

	// Ctrl+O should show save prompt instead of file picker
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'o', Mod: tea.ModCtrl})
	m = newModel.(Model)

	if m.mode != StateSavePrompt {
		t.Errorf("Expected StateSavePrompt, got %v", m.mode)
	}
	if m.pendingSaveAction != PendingOpen {
		t.Errorf("Expected PendingOpen, got %v", m.pendingSaveAction)
	}
	if m.statusMsg != "Unsaved changes! Save before open? (y/n/c)" {
		t.Errorf("Expected open save prompt message, got %q", m.statusMsg)
	}

	// Press 'n' — should proceed to file picker for open (not quit)
	newModel, cmd := m.handleSavePromptKey(tea.KeyPressMsg{Code: 'n', Text: "n"})
	m = newModel.(Model)

	if m.mode != StateFilePicker {
		t.Errorf("Expected StateFilePicker after pressing 'n', got %v", m.mode)
	}
	if m.filePickerPurpose != PickerForOpen {
		t.Errorf("Expected PickerForOpen, got %v", m.filePickerPurpose)
	}
	if m.quitting {
		t.Error("Expected quitting=false (should open, not quit)")
	}
	_ = cmd
}

// TestCtrlOWithUnsavedChangesCancel tests cancelling the save prompt from Ctrl+O.
func TestCtrlOWithUnsavedChangesCancel(t *testing.T) {
	doc, _ := document.NewDocument("x = 100\n")
	m := New(doc)
	m.width = 80
	m.height = 24

	// Make an edit
	m.cursorLine = 0
	m.loadCurrentLineIntoEditBuffer()
	m.editBuf = "y = 200"
	m.saveCurrentLine(true)

	// Trigger Ctrl+O → save prompt
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'o', Mod: tea.ModCtrl})
	m = newModel.(Model)

	// Press 'c' — should cancel and return to editing
	newModel, _ = m.handleSavePromptKey(tea.KeyPressMsg{Code: 'c', Text: "c"})
	m = newModel.(Model)

	if m.mode != StateDefault {
		t.Errorf("Expected StateDefault after cancel, got %v", m.mode)
	}
	if m.statusMsg != "Open cancelled" {
		t.Errorf("Expected 'Open cancelled', got %q", m.statusMsg)
	}
}

// TestOpenFileResetsEditBuf tests that openFile resets all editing state.
func TestOpenFileResetsEditBuf(t *testing.T) {
	m := New(nil)
	m.width = 80
	m.height = 24

	// Simulate stale state from a previous editing session
	m.editBuf = "some typed text"
	m.userIsTyping = true
	m.selectionAnchorLine = 1
	m.selectionAnchorCol = 5
	m.pendingSaveAction = PendingOpen

	// Create a temp file to open
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.cm")
	if err := os.WriteFile(tmpFile, []byte("a = 42\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Open the file
	m.openFile(tmpFile)

	// Verify all editing state was reset
	if m.editBuf != "" {
		t.Errorf("Expected editBuf to be empty, got %q", m.editBuf)
	}
	if m.userIsTyping {
		t.Error("Expected userIsTyping=false after openFile")
	}
	if m.selectionAnchorLine != -1 {
		t.Errorf("Expected selectionAnchorLine=-1, got %d", m.selectionAnchorLine)
	}
	if m.selectionAnchorCol != -1 {
		t.Errorf("Expected selectionAnchorCol=-1, got %d", m.selectionAnchorCol)
	}
	if m.cursorLine != 0 || m.cursorCol != 0 {
		t.Errorf("Expected cursor at (0,0), got (%d,%d)", m.cursorLine, m.cursorCol)
	}
	if m.modified {
		t.Error("Expected modified=false after openFile")
	}
	if m.pendingSaveAction != PendingNone {
		t.Errorf("Expected PendingNone, got %v", m.pendingSaveAction)
	}
	if m.state != StateReady {
		t.Errorf("Expected state=StateReady after openFile, got %v", m.state)
	}
}

// TestOpenFileResetsStateFromEditing verifies that openFile transitions the
// state machine from StateEditing to StateReady. This prevents the bug where
// opening a new file while actively editing left the editor in StateEditing
// with an empty editBuf, violating the StateEditing invariant.
func TestOpenFileResetsStateFromEditing(t *testing.T) {
	// Start with a document and simulate typing to enter StateEditing
	doc, _ := document.NewDocument("x = 10\ny = 20\n")
	m := New(doc)
	m.width = 80
	m.height = 24

	// Type some text to enter StateEditing
	for _, r := range "hello" {
		newModel, _ := m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		m = newModel.(Model)
	}

	if m.state != StateEditing {
		t.Fatalf("Expected StateEditing after typing, got %v", m.state)
	}

	// Create a different file to open
	tmpFile := filepath.Join(t.TempDir(), "other.cm")
	if err := os.WriteFile(tmpFile, []byte("a = 42\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Open the new file
	m.openFile(tmpFile)

	// State must be StateReady, not StateEditing
	if m.state != StateReady {
		t.Errorf("Expected StateReady after openFile, got %v", m.state)
	}
	if m.editBufLoaded {
		t.Error("Expected editBufLoaded=false after openFile")
	}
	if m.userIsTyping {
		t.Error("Expected userIsTyping=false after openFile")
	}

	// Document should be the newly opened file
	lines := m.GetLines()
	if len(lines) == 0 || lines[0] != "a = 42" {
		t.Errorf("Expected first line 'a = 42', got %v", lines)
	}
}

// TestTypeThenCtrlODetectsUnsaved traces how hasUnsavedChanges works after typing via Update.
func TestTypeThenCtrlODetectsUnsaved(t *testing.T) {
	content := "# Header\nx = 10\ny = 20\nz = 30"
	doc, _ := document.NewDocument(content)
	m := New(doc)
	m.width = 80
	m.height = 24

	// Type "hello world" through Update (simulates catwalk)
	for _, r := range "hello world" {
		newModel, _ := m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		m = newModel.(Model)
	}

	t.Logf("After typing: editBuf=%q modified=%v userIsTyping=%v", m.editBuf, m.modified, m.userIsTyping)
	t.Logf("After typing: hasUnsavedChanges=%v", m.hasUnsavedChanges())

	// Ctrl+O should show save prompt
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'o', Mod: tea.ModCtrl})
	m = newModel.(Model)

	if m.mode != StateSavePrompt {
		t.Errorf("Expected StateSavePrompt after Ctrl+O with unsaved editBuf, got %v", m.mode)
	}
}

// TestHelpOverlayRendering tests the help overlay renders without panic.
func TestSaveAsStatusMessage(t *testing.T) {
	doc, _ := document.NewDocument("x = 42\n")
	m := New(doc)

	// First save to establish a filepath
	tmpDir := t.TempDir()
	m.saveFile(filepath.Join(tmpDir, "original.cm"))
	if m.statusIsErr {
		t.Fatalf("First save failed: %s", m.statusMsg)
	}
	originalPath := m.filepath

	// Save As to a different file
	m.saveFile(filepath.Join(tmpDir, "copy.cm"))
	if m.statusIsErr {
		t.Fatalf("Save As failed: %s", m.statusMsg)
	}

	// Filepath should have changed
	if m.filepath == originalPath {
		t.Error("Expected filepath to change after Save As")
	}

	// Status should indicate the new file
	if !strings.Contains(m.statusMsg, "Now editing") {
		t.Errorf("Expected 'Now editing' in status after Save As, got %q", m.statusMsg)
	}
	if !strings.Contains(m.statusMsg, "copy.cm") {
		t.Errorf("Expected new filename in status, got %q", m.statusMsg)
	}
}

// TestInitialEvalErrors tests that evaluation errors are shown on startup.
func TestAddExportExtension(t *testing.T) {
	tests := []struct {
		filename string
		format   string
		expected string
	}{
		{"report", "html", "report.html"},
		{"data", "json", "data.json"},
		{"doc", "text", "doc.txt"},
		{"file", "cm", "file.cm"},
		{"file", "md", "file.md"},
		{"report.html", "html", "report.html"}, // already has extension
	}

	for _, tt := range tests {
		got := addExportExtension(tt.filename, tt.format)
		if got != tt.expected {
			t.Errorf("addExportExtension(%q, %q) = %q, want %q", tt.filename, tt.format, got, tt.expected)
		}
	}
}

// TestFormatToExtension tests the format-to-extension mapping.
func TestFormatToExtension(t *testing.T) {
	tests := []struct {
		format string
		ext    string
	}{
		{"text", ".txt"},
		{"cm", ".cm"},
		{"json", ".json"},
		{"html", ".html"},
		{"md", ".md"},
		{"unknown", ".unknown"},
	}

	for _, tt := range tests {
		got := formatToExtension(tt.format)
		if got != tt.ext {
			t.Errorf("formatToExtension(%q) = %q, want %q", tt.format, got, tt.ext)
		}
	}
}

// TestHelpOverlayHelpers tests help overlay helper functions.
func TestSaveAsUsesFilePicker(t *testing.T) {
	t.Run("Save As via command menu", func(t *testing.T) {
		doc, _ := document.NewDocument("x = 42\n")
		m := New(doc)

		// Open command menu
		newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyF1})
		m = newModel.(Model)

		// Navigate to Save As by name
		for m.commandMenuState.Selected < len(EditorCommands)-1 {
			if EditorCommands[m.commandMenuState.Selected].Name == "Save As" {
				break
			}
			newModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
			m = newModel.(Model)
		}
		if EditorCommands[m.commandMenuState.Selected].Name != "Save As" {
			t.Fatalf("Could not find 'Save As' in command menu")
		}

		// Execute
		newModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		m = newModel.(Model)

		if m.mode != StateFilePicker {
			t.Errorf("Expected StateFilePicker for Save As, got %v", m.mode)
		}
		if m.filePickerPurpose != PickerForSave {
			t.Errorf("Expected PickerForSave, got %v", m.filePickerPurpose)
		}
	})
}

// TestNewFile tests that newFile resets the editor to an empty document.
func TestNewFile(t *testing.T) {
	doc, _ := document.NewDocument("x = 100\ny = 200\n")
	m := New(doc)
	m.width = 80
	m.height = 24

	// Save to establish a filepath
	tmpFile := t.TempDir() + "/test.cm"
	m.saveFile(tmpFile)

	// Simulate some editing state
	m.cursorLine = 1
	m.cursorCol = 5
	m.loadCurrentLineIntoEditBuffer()
	m.editBuf = "changed"
	m.saveCurrentLine(true)

	// Invoke newFile
	m.newFile()

	// Filepath should be cleared
	if m.filepath != "" {
		t.Errorf("Expected empty filepath, got %q", m.filepath)
	}
	// Should not be modified
	if m.modified {
		t.Error("Expected modified=false after newFile")
	}
	// Cursor should be at origin
	if m.cursorLine != 0 || m.cursorCol != 0 {
		t.Errorf("Expected cursor at (0,0), got (%d,%d)", m.cursorLine, m.cursorCol)
	}
	// Edit state should be reset
	if m.editBuf != "" {
		t.Errorf("Expected empty editBuf, got %q", m.editBuf)
	}
	if m.userIsTyping {
		t.Error("Expected userIsTyping=false")
	}
	// Document should be essentially empty
	lines := m.GetLines()
	if len(lines) != 1 || lines[0] != "" {
		t.Errorf("Expected single empty line, got %v", lines)
	}
	// Status message
	if m.statusMsg != "New document" {
		t.Errorf("Expected 'New document' status, got %q", m.statusMsg)
	}
	// State machine should be ready
	if m.state != StateReady {
		t.Errorf("Expected StateReady, got %v", m.state)
	}
}

// TestCtrlNCreatesNewDocument tests that Ctrl+N creates a new document.
func TestCtrlNCreatesNewDocument(t *testing.T) {
	doc, _ := document.NewDocument("x = 42\n")
	m := New(doc)
	m.width = 80
	m.height = 24

	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl})
	m = newModel.(Model)

	// No unsaved changes, so should go straight to new document
	if m.filepath != "" {
		t.Errorf("Expected empty filepath, got %q", m.filepath)
	}
	if m.statusMsg != "New document" {
		t.Errorf("Expected 'New document' status, got %q", m.statusMsg)
	}
}

// TestCtrlNWithUnsavedChanges tests that Ctrl+N shows save prompt with unsaved changes.
func TestCtrlNWithUnsavedChanges(t *testing.T) {
	doc, _ := document.NewDocument("x = 100\n")
	m := New(doc)
	m.width = 80
	m.height = 24

	// Make an edit
	m.cursorLine = 0
	m.loadCurrentLineIntoEditBuffer()
	m.editBuf = "y = 200"
	m.saveCurrentLine(true)

	// Ctrl+N should show save prompt
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl})
	m = newModel.(Model)

	if m.mode != StateSavePrompt {
		t.Errorf("Expected StateSavePrompt, got %v", m.mode)
	}
	if m.pendingSaveAction != PendingNew {
		t.Errorf("Expected PendingNew, got %v", m.pendingSaveAction)
	}
	if m.statusMsg != "Unsaved changes! Save before new? (y/n/c)" {
		t.Errorf("Expected new save prompt message, got %q", m.statusMsg)
	}
}

// TestCtrlNSavePromptNo tests pressing 'n' at save prompt proceeds to new document.
func TestCtrlNSavePromptNo(t *testing.T) {
	doc, _ := document.NewDocument("x = 100\n")
	m := New(doc)
	m.width = 80
	m.height = 24

	// Make an edit
	m.cursorLine = 0
	m.loadCurrentLineIntoEditBuffer()
	m.editBuf = "y = 200"
	m.saveCurrentLine(true)

	// Ctrl+N → save prompt
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl})
	m = newModel.(Model)

	// Press 'n' to discard and create new
	newModel, _ = m.handleSavePromptKey(tea.KeyPressMsg{Code: 'n', Text: "n"})
	m = newModel.(Model)

	if m.mode != StateDefault {
		t.Errorf("Expected StateDefault after 'n', got %v", m.mode)
	}
	if m.filepath != "" {
		t.Errorf("Expected empty filepath, got %q", m.filepath)
	}
	if m.statusMsg != "New document" {
		t.Errorf("Expected 'New document' status, got %q", m.statusMsg)
	}
}

// TestCtrlNSavePromptCancel tests pressing 'c' cancels the new operation.
func TestCtrlNSavePromptCancel(t *testing.T) {
	doc, _ := document.NewDocument("x = 100\n")
	m := New(doc)
	m.width = 80
	m.height = 24

	// Make an edit
	m.cursorLine = 0
	m.loadCurrentLineIntoEditBuffer()
	m.editBuf = "y = 200"
	m.saveCurrentLine(true)

	// Ctrl+N → save prompt
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl})
	m = newModel.(Model)

	// Press 'c' to cancel
	newModel, _ = m.handleSavePromptKey(tea.KeyPressMsg{Code: 'c', Text: "c"})
	m = newModel.(Model)

	if m.mode != StateDefault {
		t.Errorf("Expected StateDefault after cancel, got %v", m.mode)
	}
	if m.statusMsg != "New cancelled" {
		t.Errorf("Expected 'New cancelled', got %q", m.statusMsg)
	}
}

// TestCtrlNSavePromptYes tests pressing 'y' saves then creates new document.
func TestCtrlNSavePromptYes(t *testing.T) {
	doc, _ := document.NewDocument("x = 100\n")
	m := New(doc)
	m.width = 80
	m.height = 24

	// Save to establish filepath first
	tmpFile := t.TempDir() + "/test.cm"
	m.saveFile(tmpFile)

	// Make an edit
	m.cursorLine = 0
	m.loadCurrentLineIntoEditBuffer()
	m.editBuf = "y = 200"
	m.saveCurrentLine(true)

	// Ctrl+N → save prompt
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl})
	m = newModel.(Model)

	// Press 'y' to save then new
	newModel, _ = m.handleSavePromptKey(tea.KeyPressMsg{Code: 'y', Text: "y"})
	m = newModel.(Model)

	// Should have saved and then created new document
	if m.filepath != "" {
		t.Errorf("Expected empty filepath after save+new, got %q", m.filepath)
	}
	if m.statusMsg != "New document" {
		t.Errorf("Expected 'New document' status, got %q", m.statusMsg)
	}

	// Verify original file was saved
	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to read saved file: %v", err)
	}
	if !strings.Contains(string(data), "y = 200") {
		t.Errorf("Expected saved content to contain 'y = 200', got %q", string(data))
	}
}

// TestOpenViaCommandMenu tests opening files from the command menu.
func TestOpenViaCommandMenu(t *testing.T) {
	doc, _ := document.NewDocument("x = 42\n")
	m := New(doc)

	// Open command menu
	newModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyF1})
	m = newModel.(Model)

	// Navigate to Open by name
	for m.commandMenuState.Selected < len(EditorCommands)-1 {
		if EditorCommands[m.commandMenuState.Selected].Name == "Open" {
			break
		}
		newModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		m = newModel.(Model)
	}
	if EditorCommands[m.commandMenuState.Selected].Name != "Open" {
		t.Fatalf("Could not find 'Open' in command menu")
	}

	// Execute
	newModel, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = newModel.(Model)

	if m.mode != StateFilePicker {
		t.Errorf("Expected StateFilePicker for Open, got %v", m.mode)
	}
	if m.filePickerPurpose != PickerForOpen {
		t.Errorf("Expected PickerForOpen, got %v", m.filePickerPurpose)
	}
	if m.filePickerFocus != FocusFileBrowser {
		t.Errorf("Expected FocusFileBrowser for Open, got %v", m.filePickerFocus)
	}
}

// TestExportFileLocale verifies that export uses the model's locale formatter.
func TestExportFileLocale(t *testing.T) {
	// Separate blocks with a text line so each calc block has its own LastValue.
	doc, _ := document.NewDocument("x = 1500 USD\n# separator\ny = 42.50 CNY\n")
	m := New(doc)

	// Inject de-DE formatter
	deCfg, err := display.NewConfig("de-DE")
	if err != nil {
		t.Fatalf("NewConfig(de-DE): %v", err)
	}
	m.SetFormatter(display.NewFormatter(deCfg))

	tmpDir := t.TempDir()

	t.Run("text export uses locale", func(t *testing.T) {
		textFile := filepath.Join(tmpDir, "locale_test")
		m.exportFile(textFile, "text")
		if m.statusIsErr {
			t.Fatalf("Export failed: %s", m.statusMsg)
		}

		data, err := os.ReadFile(filepath.Join(tmpDir, "locale_test.txt"))
		if err != nil {
			t.Fatalf("Read exported file: %v", err)
		}
		content := string(data)

		// de-DE: comma decimal, dot thousand
		if !strings.Contains(content, "$1.500,00") {
			t.Errorf("Expected de-DE formatted USD ($1.500,00) in text export, got:\n%s", content)
		}
		if !strings.Contains(content, "CNY 42,50") {
			t.Errorf("Expected de-DE formatted CNY (CNY 42,50) in text export, got:\n%s", content)
		}
	})

	t.Run("json export uses locale for value and ASCII for raw_value", func(t *testing.T) {
		jsonFile := filepath.Join(tmpDir, "locale_test_json")
		m.exportFile(jsonFile, "json")
		if m.statusIsErr {
			t.Fatalf("Export failed: %s", m.statusMsg)
		}

		data, err := os.ReadFile(filepath.Join(tmpDir, "locale_test_json.json"))
		if err != nil {
			t.Fatalf("Read exported file: %v", err)
		}

		var jsonDoc format.JSONDocument
		if err := json.Unmarshal(data, &jsonDoc); err != nil {
			t.Fatalf("Parse JSON: %v", err)
		}

		// Find the results across blocks
		for _, block := range jsonDoc.Blocks {
			for _, result := range block.Results {
				switch result.Variable {
				case "x":
					if result.Value != "$1.500,00" {
						t.Errorf("JSON value for x: got %q, want %q", result.Value, "$1.500,00")
					}
					// numeric_value is always locale-independent
					if result.NumericValue == nil || *result.NumericValue != 1500 {
						t.Errorf("JSON numeric_value should be 1500, got %v", result.NumericValue)
					}
					if result.Unit != "USD" {
						t.Errorf("JSON unit should be USD, got %q", result.Unit)
					}
				case "y":
					if result.Value != "CNY 42,50" {
						t.Errorf("JSON value for y: got %q, want %q", result.Value, "CNY 42,50")
					}
				}
			}
		}
	})
}
