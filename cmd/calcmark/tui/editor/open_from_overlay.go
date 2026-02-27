package editor

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/config/theme"
)

// handleOpenFromOverlayKey processes keys when the Open From Gist overlay is visible.
// User types a gist URL or ID, Enter confirms, Esc cancels.
func (m Model) handleOpenFromOverlayKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.exitOverlay()
		m.statusMsg = "Open from Gist cancelled"
		return m, nil

	case "enter":
		input := strings.TrimSpace(m.openFromInput)
		if input == "" {
			m.statusMsg = "Enter a Gist URL or ID"
			m.statusIsErr = true
			return m, nil
		}
		return m.executeOpenFromGist(input)

	case "backspace":
		if len(m.openFromInput) > 0 {
			runes := []rune(m.openFromInput)
			m.openFromInput = string(runes[:len(runes)-1])
		}
		return m, nil

	default:
		if msg.Text != "" {
			m.openFromInput += msg.Text
		}
		return m, nil
	}
}

// renderOpenFromOverlay renders the Open From Gist modal overlay.
func (m Model) renderOpenFromOverlay() string {
	o := NewOverlayStyle(56)

	var lines []string
	lines = append(lines, o.TopBorder)

	// Title
	lines = append(lines, o.WrapRow(o.PadLine(" Open From GitHub Gist", theme.TextBright, o.ItemBg, true)))

	// Separator
	lines = append(lines, o.SepRow())

	// Input field with cursor
	display := m.openFromInput + "█"
	inputContent := " Gist URL or ID: " + display
	// Truncate if too long
	if len(inputContent) > o.InnerWidth {
		inputContent = inputContent[:o.InnerWidth-1] + "…"
	}
	lines = append(lines, o.WrapRow(o.PadLine(inputContent, theme.InputFg, theme.InputBg, false)))

	// Helper text
	lines = append(lines, o.WrapRow(o.PadLine(" e.g. abc123 or https://gist.github.com/user/abc123", theme.TextMuted, o.ItemBg, false)))

	// Empty line
	lines = append(lines, o.WrapRow(o.PadLine("", theme.Hint, o.ItemBg, false)))

	// Hints
	lines = append(lines, o.HintRow(" Enter open  Esc cancel"))

	lines = append(lines, o.BottomBorder)
	return strings.Join(lines, "\n")
}
