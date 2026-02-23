package editor

// command_menu_test.go — Command menu, help overlay, export overlay, file picker overlay.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
	tea "github.com/charmbracelet/bubbletea"
)

func TestAllCommandMenuActionsViaUpdate(t *testing.T) {
	t.Run("Export from command menu", func(t *testing.T) {
		doc, _ := document.NewDocument("x = 42\n")
		m := New(doc)

		// Open command menu (F1 / Ctrl+H)
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyF1})
		m = newModel.(Model)
		if m.mode != StateCommandMenu {
			t.Fatalf("Expected StateCommandMenu, got %v", m.mode)
		}

		// Navigate to Export — find it by name since indices changed
		for m.commandMenuState.Selected < len(EditorCommands)-1 {
			if EditorCommands[m.commandMenuState.Selected].Name == "Export" {
				break
			}
			newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
			m = newModel.(Model)
		}
		if EditorCommands[m.commandMenuState.Selected].Name != "Export" {
			t.Fatalf("Could not find Export in command menu")
		}

		// Execute Export
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = newModel.(Model)
		if m.mode != StateExport {
			t.Errorf("Expected StateExport after selecting Export from command menu, got %v", m.mode)
		}
	})

	t.Run("Toggle Preview from command menu", func(t *testing.T) {
		doc, _ := document.NewDocument("x = 42\n")
		m := New(doc)
		origMode := m.previewMode

		// Open command menu
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyF1})
		m = newModel.(Model)

		// Navigate to Toggle Preview by name
		for m.commandMenuState.Selected < len(EditorCommands)-1 {
			if EditorCommands[m.commandMenuState.Selected].Name == "Toggle Preview" {
				break
			}
			newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
			m = newModel.(Model)
		}
		if EditorCommands[m.commandMenuState.Selected].Name != "Toggle Preview" {
			t.Fatalf("Expected 'Toggle Preview', got %q", EditorCommands[m.commandMenuState.Selected].Name)
		}

		// Execute
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = newModel.(Model)

		if m.mode != StateDefault {
			t.Errorf("Expected StateDefault after Toggle Preview, got %v", m.mode)
		}
		if m.previewMode == origMode {
			t.Errorf("Expected preview mode to change from %v", origMode)
		}
	})

	t.Run("Undo from command menu", func(t *testing.T) {
		doc, _ := document.NewDocument("x = 42\n")
		m := New(doc)

		// Make a change to create undo history
		m.cursorLine = 0
		m.loadCurrentLineIntoEditBuffer()
		m.editBuf = "y = 99"
		m.transitionToProcessing()

		// Open command menu
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyF1})
		m = newModel.(Model)

		// Navigate to Undo by name
		for m.commandMenuState.Selected < len(EditorCommands)-1 {
			if EditorCommands[m.commandMenuState.Selected].Name == "Undo" {
				break
			}
			newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
			m = newModel.(Model)
		}
		if EditorCommands[m.commandMenuState.Selected].Name != "Undo" {
			t.Fatalf("Expected 'Undo', got %q", EditorCommands[m.commandMenuState.Selected].Name)
		}

		// Execute
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = newModel.(Model)

		if m.mode != StateDefault {
			t.Errorf("Expected StateDefault after Undo, got %v", m.mode)
		}
	})

	t.Run("Save new file from command menu", func(t *testing.T) {
		doc, _ := document.NewDocument("x = 42\n")
		m := New(doc)
		// No filepath set — should trigger file picker

		// Open command menu
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyF1})
		m = newModel.(Model)

		// Execute Save (first item, index 0)
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = newModel.(Model)

		// Should be in file picker
		if m.mode != StateFilePicker {
			t.Errorf("Expected StateFilePicker for new file save, got %v", m.mode)
		}
	})

	t.Run("Quit from command menu (no changes)", func(t *testing.T) {
		doc, _ := document.NewDocument("x = 42\n")
		m := New(doc)

		// Open command menu
		newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyF1})
		m = newModel.(Model)

		// Navigate to Quit by name
		for m.commandMenuState.Selected < len(EditorCommands)-1 {
			if EditorCommands[m.commandMenuState.Selected].Name == "Quit" {
				break
			}
			newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
			m = newModel.(Model)
		}

		// Execute Quit
		newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = newModel.(Model)

		if !m.quitting {
			t.Error("Expected quitting=true")
		}
		if cmd == nil {
			t.Error("Expected tea.Quit command")
		}
	})
}

// TestExecuteCommandByName tests the shared command dispatch function.
func TestExecuteCommandByName(t *testing.T) {
	t.Run("Save with filepath saves immediately", func(t *testing.T) {
		tmpDir := t.TempDir()
		fpath := filepath.Join(tmpDir, "test.cm")
		os.WriteFile(fpath, []byte("x = 1\n"), 0644)

		doc, _ := document.NewDocument("x = 1\n")
		m := NewWithFile(fpath, doc)

		newModel, _ := m.executeCommandByName("Save")
		m = newModel.(Model)

		if m.mode != StateDefault {
			t.Errorf("Expected StateDefault after save, got %v", m.mode)
		}
	})

	t.Run("Save without filepath opens file picker", func(t *testing.T) {
		m := New(nil)

		newModel, _ := m.executeCommandByName("Save")
		m = newModel.(Model)

		if m.mode != StateFilePicker {
			t.Errorf("Expected StateFilePicker for new file save, got %v", m.mode)
		}
		if m.filePickerPurpose != PickerForSave {
			t.Errorf("Expected PickerForSave, got %v", m.filePickerPurpose)
		}
	})

	t.Run("Save As always opens file picker", func(t *testing.T) {
		tmpDir := t.TempDir()
		fpath := filepath.Join(tmpDir, "test.cm")
		doc, _ := document.NewDocument("x = 1\n")
		m := NewWithFile(fpath, doc)

		newModel, _ := m.executeCommandByName("Save As")
		m = newModel.(Model)

		if m.mode != StateFilePicker {
			t.Errorf("Expected StateFilePicker for Save As, got %v", m.mode)
		}
		if m.filePickerPurpose != PickerForSave {
			t.Errorf("Expected PickerForSave, got %v", m.filePickerPurpose)
		}
	})

	t.Run("Open opens file picker for browsing", func(t *testing.T) {
		m := New(nil)

		newModel, _ := m.executeCommandByName("Open")
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
	})

	t.Run("Export enters export mode", func(t *testing.T) {
		m := New(nil)

		newModel, _ := m.executeCommandByName("Export")
		m = newModel.(Model)

		if m.mode != StateExport {
			t.Errorf("Expected StateExport, got %v", m.mode)
		}
		if m.exportState.FormatIdx != 0 {
			t.Errorf("Expected format index 0, got %d", m.exportState.FormatIdx)
		}
	})

	t.Run("Quit without changes quits immediately", func(t *testing.T) {
		m := New(nil)

		newModel, cmd := m.executeCommandByName("Quit")
		m = newModel.(Model)

		if !m.quitting {
			t.Error("Expected quitting=true")
		}
		if cmd == nil {
			t.Error("Expected tea.Quit command")
		}
	})

	t.Run("Quit with unsaved changes shows prompt", func(t *testing.T) {
		m := New(nil)
		// Make a change
		m.cursorLine = 0
		m.loadCurrentLineIntoEditBuffer()
		m.editBuf = "x = 100"
		m.saveCurrentLine(true)

		newModel, _ := m.executeCommandByName("Quit")
		m = newModel.(Model)

		if m.mode != StateSavePrompt {
			t.Errorf("Expected StateSavePrompt, got %v", m.mode)
		}
	})

	t.Run("Toggle Preview changes mode", func(t *testing.T) {
		m := New(nil)
		orig := m.previewMode

		newModel, _ := m.executeCommandByName("Toggle Preview")
		m = newModel.(Model)

		if m.previewMode == orig {
			t.Error("Expected preview mode to change")
		}
	})

	t.Run("Delete Line executes without panic", func(t *testing.T) {
		doc, _ := document.NewDocument("x = 1\ny = 2\nz = 3\n")
		m := New(doc)

		// Should not panic and should return to default state
		newModel, _ := m.executeCommandByName("Delete Line")
		m = newModel.(Model)

		if m.mode != StateDefault {
			t.Errorf("Expected StateDefault after Delete Line, got %v", m.mode)
		}
	})

	t.Run("Full Help opens help overlay", func(t *testing.T) {
		m := New(nil)

		newModel, _ := m.executeCommandByName("Full Help")
		m = newModel.(Model)

		if m.mode != StateHelp {
			t.Errorf("Expected StateHelp, got %v", m.mode)
		}
		if m.helpState.Selected != 0 {
			t.Errorf("Expected selected=0, got %d", m.helpState.Selected)
		}
	})

	t.Run("Unknown command is no-op", func(t *testing.T) {
		m := New(nil)

		newModel, cmd := m.executeCommandByName("NonexistentCommand")
		m = newModel.(Model)

		if m.mode != StateDefault {
			t.Errorf("Expected StateDefault for unknown command, got %v", m.mode)
		}
		if cmd != nil {
			t.Error("Expected nil cmd for unknown command")
		}
	})
}

// TestHelpOverlayNavigation tests interactive help overlay navigation.
func TestHelpOverlayNavigation(t *testing.T) {
	t.Run("Open help overlay via executeCommandByName", func(t *testing.T) {
		m := New(nil)

		// Open help via executeCommandByName (as triggered from command menu "Full Help")
		newModel, _ := m.executeCommandByName("Full Help")
		m = newModel.(Model)

		if m.mode != StateHelp {
			t.Fatalf("Expected StateHelp, got %v", m.mode)
		}

		// First actionable item should be selected (index 0 = Save)
		indices := actionableIndices()
		if m.helpState.Selected != indices[0] {
			t.Errorf("Expected first actionable at index %d, got %d", indices[0], m.helpState.Selected)
		}
	})

	t.Run("Down arrow moves to next actionable", func(t *testing.T) {
		m := New(nil)
		m.mode = StateHelp
		m.helpState = HelpOverlayState{Selected: 0, ScrollOffset: 0}

		indices := actionableIndices()
		if len(indices) < 2 {
			t.Fatal("Need at least 2 actionable items")
		}

		// Press Down to move to next actionable
		newModel, _ := m.handleHelpOverlayKey(tea.KeyMsg{Type: tea.KeyDown})
		m = newModel.(Model)

		if m.helpState.Selected != indices[1] {
			t.Errorf("Expected next actionable at index %d, got %d", indices[1], m.helpState.Selected)
		}
	})

	t.Run("Up arrow moves to previous actionable", func(t *testing.T) {
		m := New(nil)
		indices := actionableIndices()
		m.mode = StateHelp
		// Start at second actionable
		m.helpState = HelpOverlayState{Selected: indices[1], ScrollOffset: 0}

		newModel, _ := m.handleHelpOverlayKey(tea.KeyMsg{Type: tea.KeyUp})
		m = newModel.(Model)

		if m.helpState.Selected != indices[0] {
			t.Errorf("Expected previous actionable at index %d, got %d", indices[0], m.helpState.Selected)
		}
	})

	t.Run("Esc closes help overlay", func(t *testing.T) {
		m := New(nil)
		m.mode = StateHelp
		m.helpState = HelpOverlayState{Selected: 0, ScrollOffset: 0}

		newModel, _ := m.handleHelpOverlayKey(tea.KeyMsg{Type: tea.KeyEsc})
		m = newModel.(Model)

		if m.mode != StateDefault {
			t.Errorf("Expected StateDefault after Esc, got %v", m.mode)
		}
	})

	t.Run("Enter executes selected actionable item", func(t *testing.T) {
		m := New(nil)
		m.mode = StateHelp

		// Find the Export item's index in the flat list
		items := flatHelpItems()
		exportIdx := -1
		for i, item := range items {
			if item.CommandName == "Export" {
				exportIdx = i
				break
			}
		}
		if exportIdx == -1 {
			t.Fatal("Export not found in help items")
		}

		m.helpState = HelpOverlayState{Selected: exportIdx, ScrollOffset: 0}

		newModel, _ := m.handleHelpOverlayKey(tea.KeyMsg{Type: tea.KeyEnter})
		m = newModel.(Model)

		// Should have executed Export, which enters StateExport
		if m.mode != StateExport {
			t.Errorf("Expected StateExport after executing Export from help, got %v", m.mode)
		}
	})

	t.Run("Navigation wraps around", func(t *testing.T) {
		m := New(nil)
		indices := actionableIndices()
		m.mode = StateHelp
		// Start at first actionable
		m.helpState = HelpOverlayState{Selected: indices[0], ScrollOffset: 0}

		// Press Up — should wrap to last actionable
		newModel, _ := m.handleHelpOverlayKey(tea.KeyMsg{Type: tea.KeyUp})
		m = newModel.(Model)

		if m.helpState.Selected != indices[len(indices)-1] {
			t.Errorf("Expected wrap to last actionable at index %d, got %d",
				indices[len(indices)-1], m.helpState.Selected)
		}
	})

	t.Run("Help categories structure is valid", func(t *testing.T) {
		cats := helpCategories()
		if len(cats) < 3 {
			t.Errorf("Expected at least 3 categories, got %d", len(cats))
		}

		// Verify we have both actionable and advisory items
		var actionCount, advisoryCount int
		for _, cat := range cats {
			for _, item := range cat.Items {
				if item.Kind == HelpActionable {
					actionCount++
					if item.CommandName == "" {
						t.Errorf("Actionable item %q has empty CommandName", item.Name)
					}
				} else {
					advisoryCount++
				}
			}
		}
		if actionCount == 0 {
			t.Error("No actionable items found")
		}
		if advisoryCount == 0 {
			t.Error("No advisory items found")
		}
	})
}

// TestExportOverlayNumberShortcuts tests number key shortcuts for format selection.
// Number keys select format and transition to file picker.
func TestHelpOverlayRendering(t *testing.T) {
	m := New(nil)
	m.width = 80
	m.height = 24
	m.mode = StateHelp
	m.helpState = HelpOverlayState{Selected: 0, ScrollOffset: 0}

	output := m.renderHelpOverlay()
	if output == "" {
		t.Error("Expected non-empty help overlay output")
	}

	// Should contain help title
	if !strings.Contains(output, "CalcMark Help") {
		t.Error("Expected help overlay to contain title 'CalcMark Help'")
	}

	// Should contain hint line
	if !strings.Contains(output, "navigate") {
		t.Error("Expected help overlay to contain navigation hint")
	}
}

// TestExportOverlayRendering tests the export overlay renders without panic.
func TestHelpOverlayHelpers(t *testing.T) {
	t.Run("flatHelpItems returns all items", func(t *testing.T) {
		items := flatHelpItems()
		if len(items) == 0 {
			t.Error("Expected non-empty flat items list")
		}

		// Count items from categories to verify
		var expected int
		for _, cat := range helpCategories() {
			expected += len(cat.Items)
		}
		if len(items) != expected {
			t.Errorf("flatHelpItems returned %d items, expected %d", len(items), expected)
		}
	})

	t.Run("actionableIndices returns only actionable items", func(t *testing.T) {
		indices := actionableIndices()
		items := flatHelpItems()

		for _, idx := range indices {
			if idx >= len(items) {
				t.Errorf("actionableIndex %d out of range (max %d)", idx, len(items)-1)
				continue
			}
			if items[idx].Kind != HelpActionable {
				t.Errorf("Index %d is not actionable: %q", idx, items[idx].Name)
			}
		}
	})

	t.Run("nextActionableIndex wraps around", func(t *testing.T) {
		indices := actionableIndices()
		if len(indices) < 2 {
			t.Skip("Need at least 2 actionable items")
		}

		last := indices[len(indices)-1]
		next := nextActionableIndex(indices, last+1)
		if next != indices[0] {
			t.Errorf("Expected wrap to first index %d, got %d", indices[0], next)
		}
	})

	t.Run("prevActionableIndex wraps around", func(t *testing.T) {
		indices := actionableIndices()
		if len(indices) < 2 {
			t.Skip("Need at least 2 actionable items")
		}

		first := indices[0]
		prev := prevActionableIndex(indices, first-1)
		if prev != indices[len(indices)-1] {
			t.Errorf("Expected wrap to last index %d, got %d", indices[len(indices)-1], prev)
		}
	})
}

// TestSaveAsUsesFilePicker tests that Save As redirects to file picker.
func TestExportOverlayRendering(t *testing.T) {
	m := New(nil)
	m.width = 80
	m.height = 24
	m.enterExportMode()

	output := m.renderExportOverlay()
	if output == "" {
		t.Error("Expected non-empty export overlay output")
	}

	// Should contain Export title
	if !strings.Contains(output, "Export") {
		t.Error("Expected export overlay to contain title 'Export'")
	}

	// Should contain format hint
	if !strings.Contains(output, "Format") {
		t.Error("Expected export overlay to contain 'Format'")
	}
}

// TestFilePickerOverlayRendering tests the file picker overlay renders with
// consistent width and purpose-aware titles.
func TestFilePickerOverlayRendering(t *testing.T) {
	m := New(nil)
	m.width = 80
	m.height = 24

	purposes := []struct {
		purpose FilePickerPurpose
		title   string
	}{
		{PickerForSave, "Save to"},
		{PickerForOpen, "Open"},
		{PickerForExport, "Export to"},
	}

	for _, tc := range purposes {
		m.filePicker = initFilePicker()
		m.filePickerPurpose = tc.purpose
		m.mode = StateFilePicker

		output := m.renderFilePickerOverlay()
		if output == "" {
			t.Errorf("Expected non-empty file picker overlay for %v", tc.purpose)
		}
		if !strings.Contains(output, tc.title) {
			t.Errorf("Expected file picker overlay to contain %q for purpose %v", tc.title, tc.purpose)
		}
	}
}

// TestSaveAsStatusMessage tests that Save As shows a clear status message.
