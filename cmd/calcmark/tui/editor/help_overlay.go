package editor

import (
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/shared"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/lipgloss"
)

// renderHelpOverlay renders a centered help panel showing all keybindings.
// The overlay is displayed when the user presses F1 to toggle help mode.
func (m Model) renderHelpOverlay() string {
	// Create help model with full keybindings shown
	h := help.New()
	h.ShowAll = true
	h.Width = m.width - 20 // Leave margins

	// Get the keybindings to display
	keys := shared.DefaultKeyMap()
	helpContent := h.View(keys)

	// Style the help content box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(m.width - 10).
		MaxHeight(m.height - 4)

	// Title style
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("212"))

	title := titleStyle.Render("CalcMark Help")

	// Footer style
	footerStyle := lipgloss.NewStyle().
		Faint(true)

	footer := footerStyle.Render("Press F1 or Esc to close")

	// Assemble content vertically
	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		helpContent,
		"",
		footer,
	)

	return boxStyle.Render(content)
}
