package editor

// globals_handler.go — Key handling for globals panel.

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
