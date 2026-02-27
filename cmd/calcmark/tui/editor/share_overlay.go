package editor

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/config/theme"
)

// handleShareOverlayKey processes keys when the Share To Gist overlay is visible.
// Tab cycles between fields, Up/Down toggle visibility, Enter confirms, Esc cancels.
func (m Model) handleShareOverlayKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.exitOverlay()
		m.statusMsg = "Share cancelled"
		return m, nil

	case "tab":
		// Cycle between visibility (0) and description (1)
		m.shareField = (m.shareField + 1) % 2
		return m, nil

	case "up", "down":
		if m.shareField == 0 {
			// Toggle visibility: 0 = public, 1 = secret
			m.shareVisibility = 1 - m.shareVisibility
		}
		return m, nil

	case "enter":
		// Confirm — launch the share operation
		return m.executeShareToGist()

	case "backspace":
		if m.shareField == 1 && len(m.shareDescription) > 0 {
			// Remove last rune from description
			runes := []rune(m.shareDescription)
			m.shareDescription = string(runes[:len(runes)-1])
		}
		return m, nil

	default:
		if m.shareField == 1 && msg.Text != "" {
			// Typing into description field
			m.shareDescription += msg.Text
		}
		return m, nil
	}
}

// renderShareOverlay renders the Share To Gist modal overlay.
func (m Model) renderShareOverlay() string {
	o := NewOverlayStyle(50)

	var lines []string
	lines = append(lines, o.TopBorder)

	// Title
	lines = append(lines, o.WrapRow(o.PadLine(" Share To GitHub Gist", theme.TextBright, o.ItemBg, true)))

	// Separator
	lines = append(lines, o.SepRow())

	// Visibility field
	var visLabel string
	if m.shareVisibility == 0 {
		visLabel = " Visibility: Public"
	} else {
		visLabel = " Visibility: Secret"
	}
	if m.shareField == 0 {
		lines = append(lines, o.WrapRow(o.PadLine(visLabel, theme.PopupSelectedFg, o.SelectedBg, true)))
	} else {
		lines = append(lines, o.WrapRow(o.PadLine(visLabel, theme.Text, o.ItemBg, false)))
	}

	// Description field
	descDisplay := m.shareDescription
	if m.shareField == 1 {
		descDisplay += "█"
	}
	if descDisplay == "" {
		descDisplay = "(optional)"
	}
	descContent := " Description: " + descDisplay
	// Truncate if too long
	if len(descContent) > o.InnerWidth {
		descContent = descContent[:o.InnerWidth-1] + "…"
	}
	if m.shareField == 1 {
		lines = append(lines, o.WrapRow(o.PadLine(descContent, theme.InputFg, theme.InputBg, false)))
	} else {
		lines = append(lines, o.WrapRow(o.PadLine(descContent, theme.Text, o.ItemBg, false)))
	}

	// Separator
	lines = append(lines, o.SepRow())

	// File info
	var filename string
	if m.filepath != "" {
		filename = resolveFilename(m.filepath)
	} else {
		filename = "untitled.cm"
	}
	fileInfo := fmt.Sprintf(" File: %s", filename)
	lines = append(lines, o.WrapRow(o.PadLine(fileInfo, theme.TextMuted, o.ItemBg, false)))

	// Empty line
	lines = append(lines, o.WrapRow(o.PadLine("", theme.Hint, o.ItemBg, false)))

	// Hints
	lines = append(lines, o.HintRow(" Tab switch  ↑↓ toggle  Enter share  Esc cancel"))

	lines = append(lines, o.BottomBorder)
	return strings.Join(lines, "\n")
}

// resolveFilename returns the base filename for sharing.
// If the document has a filepath, use its base name.
// Otherwise, return "untitled.cm".
func resolveFilename(fp string) string {
	if fp == "" {
		return "untitled.cm"
	}
	return filepath.Base(fp)
}
