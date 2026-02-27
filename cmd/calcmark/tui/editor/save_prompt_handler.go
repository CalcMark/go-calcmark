package editor

// save_prompt_handler.go — Save prompt and unsaved-changes guard.

import (
	tea "charm.land/bubbletea/v2"
)

// promptSaveIfNeeded checks for unsaved changes and either enters the save
// prompt (returning true) or returns false so the caller can proceed directly.
// This centralises the unsaved-changes guard used by Ctrl+Q, Ctrl+O, Ctrl+N,
// and the command menu equivalents.
func (m *Model) promptSaveIfNeeded(action PendingAction, promptMsg string) bool {
	if !m.hasUnsavedChanges() {
		return false
	}
	m.enterSavePrompt(action, promptMsg)
	return true
}

// handleSavePromptKey processes keys in save prompt mode.
// The prompt is triggered by either Ctrl+Q (quit) or Ctrl+O (open),
// with m.pendingSaveAction tracking which action to perform after
// the user responds.
func (m Model) handleSavePromptKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		// Save first, then perform the pending action
		if m.filepath == "" {
			cmd := m.enterFilePicker(PickerForSave, FocusFilename)
			return m, cmd
		}
		m.saveFile("")
		if m.statusIsErr {
			// Save failed — stay in prompt mode
			return m, nil
		}
		return m.completePendingSaveAction()
	case "n", "N":
		// Skip saving and perform the pending action directly
		return m.completePendingSaveAction()
	case "c", "C":
		// Cancel the pending action
		cancelMsg := actionCancelledMsg(m.pendingSaveAction)
		m.exitOverlay()
		m.statusMsg = cancelMsg
	case "esc":
		// Cancel the pending action
		cancelMsg := actionCancelledMsg(m.pendingSaveAction)
		m.exitOverlay()
		m.statusMsg = cancelMsg
	}
	return m, nil
}

// completePendingSaveAction finishes whatever action triggered the save prompt.
func (m Model) completePendingSaveAction() (tea.Model, tea.Cmd) {
	action := m.pendingSaveAction
	switch action {
	case PendingQuit:
		m.exitOverlay()
		m.quitting = true
		return m, tea.Quit
	case PendingOpen:
		m.exitOverlay()
		cmd := m.enterFilePicker(PickerForOpen, FocusFileBrowser)
		return m, cmd
	case PendingNew:
		m.exitOverlay()
		m.newFile()
		return m, nil
	case PendingOpenFromRemote:
		m.exitOverlay()
		m.enterOpenFrom()
		return m, nil
	default:
		m.exitOverlay()
		return m, nil
	}
}

// actionCancelledMsg returns the status message when a pending action is cancelled.
func actionCancelledMsg(action PendingAction) string {
	switch action {
	case PendingQuit:
		return "Quit cancelled"
	case PendingOpen:
		return "Open cancelled"
	case PendingNew:
		return "New cancelled"
	case PendingOpenFromRemote:
		return "Open from Gist cancelled"
	default:
		return ""
	}
}
