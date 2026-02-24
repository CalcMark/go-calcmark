package editor

// globals_handler.go — Key handling for globals panel.

import (
	tea "charm.land/bubbletea/v2"
)

// handleGlobalsKey processes keys when globals panel is focused.
func (m Model) handleGlobalsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = StateDefault
		m.globalsExpanded = false
	case "up", "k":
		if m.globalsFocusIdx > 0 {
			m.globalsFocusIdx--
		}
	case "down", "j":
		globalsCount := m.getGlobalsCount()
		if m.globalsFocusIdx < globalsCount-1 {
			m.globalsFocusIdx++
		}
	case "enter":
		// Could edit focused global
		m.mode = StateDefault
	}

	return m, nil
}
