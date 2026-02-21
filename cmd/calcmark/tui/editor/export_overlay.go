package editor

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ExportOverlayState holds the state for the export format selection overlay.
// After format selection, the file picker opens for directory/filename choice.
type ExportOverlayState struct {
	FormatIdx int // Selected index in exportFormatOpts
}

// handleExportOverlayKey processes keys when the export format overlay is visible.
// Up/Down navigate formats; Enter or number key selects format and opens file picker.
func (m Model) handleExportOverlayKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.mode = StateDefault
		m.statusMsg = "Export cancelled"
		return m, nil

	case tea.KeyUp:
		if m.exportState.FormatIdx > 0 {
			m.exportState.FormatIdx--
		}
		return m, nil

	case tea.KeyDown:
		if m.exportState.FormatIdx < len(m.exportFormatOpts)-1 {
			m.exportState.FormatIdx++
		}
		return m, nil

	case tea.KeyEnter:
		// Format selected — open file picker for export destination
		return m.openExportFilePicker()

	case tea.KeyRunes:
		// Number key shortcuts for format selection (1-5)
		if len(msg.Runes) > 0 {
			key := msg.Runes[0]
			if key >= '1' && key <= '5' {
				idx := int(key - '1')
				if idx < len(m.exportFormatOpts) {
					m.exportState.FormatIdx = idx
					return m.openExportFilePicker()
				}
			}
		}
		return m, nil
	}

	return m, nil
}

// openExportFilePicker transitions from format selection to the file picker
// with PickerForExport purpose.
func (m Model) openExportFilePicker() (tea.Model, tea.Cmd) {
	m.filePicker = initFilePicker()
	m.filePickerFocus = FocusFilename
	m.filePickerPurpose = PickerForExport
	m.mode = StateFilePicker
	return m, m.filePicker.Init()
}

// renderExportOverlay renders the format selection modal as a centered overlay.
func (m Model) renderExportOverlay() string {
	state := m.exportState
	formats := m.exportFormatOpts
	innerWidth := 44

	// Border and background colors (shared palette across all overlays)
	borderFg := lipgloss.Color("#5C5C5C")
	borderStyle := lipgloss.NewStyle().Foreground(borderFg)
	itemBg := lipgloss.Color("#1E1E1E")
	selectedBg := lipgloss.Color("#4A90D9")

	topBorder := borderStyle.Render("╭" + strings.Repeat("─", innerWidth) + "╮")
	bottomBorder := borderStyle.Render("╰" + strings.Repeat("─", innerWidth) + "╯")
	leftBorder := borderStyle.Render("│")
	rightBorder := borderStyle.Render("│")
	sepLine := borderStyle.Render(strings.Repeat("─", innerWidth))

	// pad creates a line padded to innerWidth with the given foreground/background
	pad := func(content string, fg, bg lipgloss.Color, bold bool) string {
		style := lipgloss.NewStyle().Foreground(fg).Background(bg)
		if bold {
			style = style.Bold(true)
		}
		visualWidth := lipgloss.Width(content)
		if visualWidth < innerWidth {
			content += strings.Repeat(" ", innerWidth-visualWidth)
		}
		return style.Render(content)
	}

	var lines []string
	lines = append(lines, topBorder)

	// Title
	lines = append(lines, leftBorder+pad(" Export — Select Format", "#FFFFFF", itemBg, true)+rightBorder)

	// Separator
	lines = append(lines, leftBorder+sepLine+rightBorder)

	// Format items
	for i, f := range formats {
		ext := formatToExtension(f)
		var prefix string
		if i == state.FormatIdx {
			prefix = " ▸ "
		} else {
			prefix = "   "
		}
		content := fmt.Sprintf("%s%d) %-8s (%s)", prefix, i+1, f, ext)

		if i == state.FormatIdx {
			lines = append(lines, leftBorder+pad(content, "#FFFFFF", selectedBg, true)+rightBorder)
		} else {
			lines = append(lines, leftBorder+pad(content, "#CCCCCC", itemBg, false)+rightBorder)
		}
	}

	// Separator before hints
	lines = append(lines, leftBorder+sepLine+rightBorder)

	// Hints
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Background(itemBg).
		Italic(true)
	hint := " ↑↓ navigate  Enter select  Esc cancel"
	visualWidth := lipgloss.Width(hint)
	if visualWidth < innerWidth {
		hint += strings.Repeat(" ", innerWidth-visualWidth)
	}
	lines = append(lines, leftBorder+hintStyle.Render(hint)+rightBorder)

	lines = append(lines, bottomBorder)
	return strings.Join(lines, "\n")
}

// formatToExtension returns the file extension for a given format name.
func formatToExtension(formatName string) string {
	switch formatName {
	case "text":
		return ".txt"
	case "cm":
		return ".cm"
	case "json":
		return ".json"
	case "html":
		return ".html"
	case "md":
		return ".md"
	default:
		return "." + formatName
	}
}
