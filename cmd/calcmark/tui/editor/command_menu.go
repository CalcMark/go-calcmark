package editor

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Command defines a single editor command displayed in the command menu.
// Commands are informational - accelerators still execute directly.
// The menu is for discovery, not the primary action path.
type Command struct {
	Name        string // Display name: "Save", "Export", "Quit"
	Accelerator string // Key hint: "Ctrl+S", "Ctrl+E"
	Description string // Brief description
	Category    string // "file", "edit", "view", "navigation", "help"
}

// CommandMenuState tracks the state of the command menu popup.
type CommandMenuState struct {
	Selected int // Currently selected command index
}

// EditorCommands contains all commands organized by category.
// These are displayed in the command menu for discovery.
var EditorCommands = []Command{
	// File commands
	{Name: "Save", Accelerator: "Ctrl+S", Description: "Save document", Category: "file"},
	{Name: "Export", Accelerator: "Ctrl+E", Description: "Export to format", Category: "file"},
	{Name: "Quit", Accelerator: "Ctrl+Q", Description: "Quit editor", Category: "file"},

	// Edit commands
	{Name: "Undo", Accelerator: "Ctrl+Z", Description: "Undo last change", Category: "edit"},
	{Name: "Redo", Accelerator: "Ctrl+Y", Description: "Redo last change", Category: "edit"},

	// View commands
	{Name: "Toggle Preview", Accelerator: "Ctrl+P", Description: "Cycle preview mode", Category: "view"},

	// Navigation commands
	{Name: "Word Left", Accelerator: "Ctrl+Left/Alt+B", Description: "Move to previous word", Category: "navigation"},
	{Name: "Word Right", Accelerator: "Ctrl+Right/Alt+F", Description: "Move to next word", Category: "navigation"},
	{Name: "Doc Start", Accelerator: "Ctrl+Home", Description: "Jump to document start", Category: "navigation"},
	{Name: "Doc End", Accelerator: "Ctrl+End", Description: "Jump to document end", Category: "navigation"},

	// Help commands
	{Name: "Full Help", Accelerator: "F1", Description: "Show full help", Category: "help"},
}

// handleCommandMenuKey processes keys when the command menu popup is visible.
// Arrow keys navigate, Enter executes, Escape dismisses.
// Typing any character dismisses the menu (like autocomplete).
func (m Model) handleCommandMenuKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		// Move selection up
		if m.commandMenuState.Selected > 0 {
			m.commandMenuState.Selected--
		}
		return m, nil

	case tea.KeyDown:
		// Move selection down
		if m.commandMenuState.Selected < len(EditorCommands)-1 {
			m.commandMenuState.Selected++
		}
		return m, nil

	case tea.KeyEnter:
		// Execute selected command
		return m.executeCommandMenuSelection()

	case tea.KeyEsc:
		// Dismiss menu without executing
		m.mode = StateDefault
		return m, nil

	default:
		// Any other key (including typing) dismisses the menu
		// and is processed normally
		m.mode = StateDefault
		return m.handleDefaultKey(msg)
	}
}

// executeCommandMenuSelection executes the currently selected command.
func (m Model) executeCommandMenuSelection() (tea.Model, tea.Cmd) {
	if m.commandMenuState.Selected < 0 || m.commandMenuState.Selected >= len(EditorCommands) {
		m.mode = StateDefault
		return m, nil
	}

	cmd := EditorCommands[m.commandMenuState.Selected]
	m.mode = StateDefault

	// Execute the command based on its name
	switch cmd.Name {
	case "Save":
		if m.filepath == "" {
			m.mode = StateSaveAsPath
			m.saveAsPath = ""
			m.statusMsg = "Save as (filename):"
			return m, nil
		}
		m.saveFile("")
		return m, nil

	case "Export":
		m.enterExportMode()
		return m, nil

	case "Quit":
		if m.hasUnsavedChanges() {
			m.mode = StateSavePrompt
			m.statusMsg = "Unsaved changes! Save before quit? (y/n/c)"
			return m, nil
		}
		m.quitting = true
		return m, tea.Quit

	case "Undo":
		m.undo()
		return m, nil

	case "Redo":
		m.redo()
		return m, nil

	case "Toggle Preview":
		m.cyclePreviewMode()
		return m, nil

	case "Word Left":
		return m.handleCtrlLeftKey()

	case "Word Right":
		return m.handleCtrlRightKey()

	case "Doc Start":
		return m.handleCtrlHomeKey()

	case "Doc End":
		return m.handleCtrlEndKey()

	case "Full Help":
		m.mode = StateHelp
		return m, nil
	}

	return m, nil
}

// GetCommandMenuState returns the command menu state for rendering.
func (m Model) GetCommandMenuState() CommandMenuState {
	return m.commandMenuState
}
