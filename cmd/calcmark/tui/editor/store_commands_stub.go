//go:build js || wasm

package editor

import (
	tea "charm.land/bubbletea/v2"
)

// executeShareToGist is a stub for WASM builds where subprocess execution is unavailable.
func (m Model) executeShareToGist() (tea.Model, tea.Cmd) {
	m.statusMsg = "Share to Gist is not available in the browser"
	m.statusIsErr = true
	m.exitOverlay()
	return m, nil
}

// executeOpenFromGist is a stub for WASM builds where subprocess execution is unavailable.
func (m Model) executeOpenFromGist(_ string) (tea.Model, tea.Cmd) {
	m.statusMsg = "Open from Gist is not available in the browser"
	m.statusIsErr = true
	m.exitOverlay()
	return m, nil
}
