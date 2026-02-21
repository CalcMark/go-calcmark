package editor

// globals_handler.go — Key handling for globals panel and save prompt.

import (
	tea "github.com/charmbracelet/bubbletea"
)

// handleGlobalsKey processes keys when globals panel is focused.
func (m Model) handleGlobalsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.mode = StateDefault
		m.globalsExpanded = false
	case tea.KeyUp, tea.KeyRunes:
		if msg.Type == tea.KeyUp || (len(msg.Runes) > 0 && msg.Runes[0] == 'k') {
			if m.globalsFocusIdx > 0 {
				m.globalsFocusIdx--
			}
		}
	case tea.KeyDown:
		globalsCount := m.getGlobalsCount()
		if m.globalsFocusIdx < globalsCount-1 {
			m.globalsFocusIdx++
		}
	case tea.KeyEnter:
		// Could edit focused global
		m.mode = StateDefault
	}

	// Handle 'j' for down
	if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 && msg.Runes[0] == 'j' {
		globalsCount := m.getGlobalsCount()
		if m.globalsFocusIdx < globalsCount-1 {
			m.globalsFocusIdx++
		}
	}

	return m, nil
}

// handleSavePromptKey processes keys in save prompt mode (before quit).
func (m Model) handleSavePromptKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
		switch msg.Runes[0] {
		case 'y', 'Y':
			// Save and quit - but if no filename, open file picker first
			if m.filepath == "" {
				m.filePicker = initFilePicker()
				m.filePickerFocus = FocusFilename
				m.filePickerPurpose = PickerForSave
				m.mode = StateFilePicker
				return m, m.filePicker.Init()
			}
			m.saveFile("")
			if !m.statusIsErr {
				m.quitting = true
				return m, tea.Quit
			}
			// If save failed, stay in prompt mode
			return m, nil
		case 'n', 'N':
			// Quit without saving
			m.quitting = true
			return m, tea.Quit
		case 'c', 'C':
			// Cancel quit
			m.mode = StateDefault
			m.statusMsg = "Quit cancelled"
		}
	} else if msg.Type == tea.KeyEsc {
		// Cancel quit
		m.mode = StateDefault
		m.statusMsg = "Quit cancelled"
	}
	return m, nil
}
