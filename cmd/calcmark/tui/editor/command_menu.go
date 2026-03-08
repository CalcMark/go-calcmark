package editor

import (
	tea "charm.land/bubbletea/v2"
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
	{Name: "New", Accelerator: "Ctrl+N", Description: "New empty document", Category: "file"},
	{Name: "Save", Accelerator: "Ctrl+S", Description: "Save document", Category: "file"},
	{Name: "Save As", Accelerator: "", Description: "Save with new name", Category: "file"},
	{Name: "Open", Accelerator: "Ctrl+O", Description: "Open file", Category: "file"},
	{Name: "Export", Accelerator: "Ctrl+E", Description: "Export to format", Category: "file"},
	{Name: "Quit", Accelerator: "Ctrl+Q", Description: "Quit editor", Category: "file"},

	// Edit commands
	{Name: "Undo", Accelerator: "Ctrl+Z", Description: "Undo last change", Category: "edit"},
	{Name: "Redo", Accelerator: "Ctrl+Y", Description: "Redo last change", Category: "edit"},
	{Name: "Delete Line", Accelerator: "Ctrl+K", Description: "Delete current line", Category: "edit"},
	{Name: "Insert Frontmatter", Accelerator: "Ctrl+F", Description: "Add exchange, globals, scale, and convert_to", Category: "edit"},

	// View commands
	{Name: "Toggle Preview", Accelerator: "Ctrl+P", Description: "Cycle preview mode", Category: "view"},

	// Navigation commands
	{Name: "Word Left", Accelerator: "Opt+Left", Description: "Move to previous word (also Opt+B)", Category: "navigation"},
	{Name: "Word Right", Accelerator: "Opt+Right", Description: "Move to next word (also Opt+F)", Category: "navigation"},
	{Name: "Doc Start", Accelerator: "⌘↑ / Opt+↑", Description: "Jump to document start", Category: "navigation"},
	{Name: "Doc End", Accelerator: "⌘↓ / Opt+↓", Description: "Jump to document end", Category: "navigation"},

	// Share commands
	{Name: "Share To Gist", Accelerator: "", Description: "Share document as GitHub Gist", Category: "share"},
	{Name: "Open From Gist", Accelerator: "", Description: "Open document from GitHub Gist", Category: "share"},

	// Help commands
	{Name: "Full Help", Accelerator: "F1", Description: "Show full help", Category: "help"},
}

// handleCommandMenuKey processes keys when the command menu popup is visible.
// Arrow keys navigate, Enter executes, Escape dismisses.
// Typing any character dismisses the menu (like autocomplete).
func (m Model) handleCommandMenuKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up":
		// Move selection up
		if m.commandMenuState.Selected > 0 {
			m.commandMenuState.Selected--
		}
		return m, nil

	case "down":
		// Move selection down
		if m.commandMenuState.Selected < len(EditorCommands)-1 {
			m.commandMenuState.Selected++
		}
		return m, nil

	case "enter":
		// Execute selected command
		return m.executeCommandMenuSelection()

	case "esc":
		// Dismiss menu without executing
		m.exitOverlay()
		return m, nil

	default:
		// Any other key (including typing) dismisses the menu
		// and is processed normally
		m.exitOverlay()
		return m.handleDefaultKey(msg)
	}
}

// executeCommandMenuSelection executes the currently selected command.
// Thin wrapper that resolves the selected item's name and delegates.
func (m Model) executeCommandMenuSelection() (tea.Model, tea.Cmd) {
	if m.commandMenuState.Selected < 0 || m.commandMenuState.Selected >= len(EditorCommands) {
		m.exitOverlay()
		return m, nil
	}

	cmd := EditorCommands[m.commandMenuState.Selected]
	m.exitOverlay()
	return m.executeCommandByName(cmd.Name)
}

// executeCommandByName dispatches a command by name.
// Shared by the command menu and help overlay.
func (m Model) executeCommandByName(name string) (tea.Model, tea.Cmd) {
	switch name {
	case "New":
		if m.promptSaveIfNeeded(PendingNew, "Unsaved changes! Save before new? (y/n/c)") {
			return m, nil
		}
		m.newFile()
		return m, nil

	case "Save":
		if m.filepath == "" {
			// No filepath: open file picker for save
			cmd := m.enterFilePicker(PickerForSave, FocusFilename)
			return m, cmd
		}
		m.saveFile("")
		return m, nil

	case "Save As":
		// Always open file picker for save-as
		cmd := m.enterFilePicker(PickerForSave, FocusFilename)
		return m, cmd

	case "Open":
		if m.promptSaveIfNeeded(PendingOpen, "Unsaved changes! Save before open? (y/n/c)") {
			return m, nil
		}
		cmd := m.enterFilePicker(PickerForOpen, FocusFileBrowser)
		return m, cmd

	case "Export":
		m.enterExportMode()
		return m, nil

	case "Quit":
		if m.promptSaveIfNeeded(PendingQuit, "Unsaved changes! Save before quit? (y/n/c)") {
			return m, nil
		}
		m.quitting = true
		return m, tea.Quit

	case "Undo":
		return m.handleUndo()

	case "Redo":
		return m.handleRedo()

	case "Delete Line":
		return m.handleDeleteLine()

	case "Insert Frontmatter":
		return m.insertFrontmatter()

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
		m.enterHelp()
		return m, nil

	case "Share To Gist":
		m.enterShareTo()
		return m, nil

	case "Open From Gist":
		if m.promptSaveIfNeeded(PendingOpenFromRemote, "Unsaved changes! Save before open? (y/n/c)") {
			return m, nil
		}
		m.enterOpenFrom()
		return m, nil
	}

	return m, nil
}

// GetCommandMenuState returns the command menu state for rendering.
func (m Model) GetCommandMenuState() CommandMenuState {
	return m.commandMenuState
}
