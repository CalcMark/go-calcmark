package editor

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/config/theme"
	tea "github.com/charmbracelet/bubbletea"
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
		// For any other rune, cancel Export mode
		m.mode = StateDefault
		m.statusMsg = "Export cancelled"
		return m.handleDefaultKey(msg)

	// For any navigation keys (arrows with modifiers, home, end, etc.),
	// cancel Export mode and process the key normally
	case tea.KeyLeft, tea.KeyRight, tea.KeyCtrlLeft, tea.KeyCtrlRight,
		tea.KeyHome, tea.KeyEnd, tea.KeyCtrlHome, tea.KeyCtrlEnd,
		tea.KeyPgUp, tea.KeyPgDown:
		m.mode = StateDefault
		m.statusMsg = "Export cancelled"
		return m.handleDefaultKey(msg)
	}

	// For any other unexpected keys, just ignore them (don't exit Export mode)
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

	o := NewOverlayStyle(44)

	var lines []string
	lines = append(lines, o.TopBorder)

	// Title
	lines = append(lines, o.WrapRow(o.PadLine(" Export — Select Format", theme.TextBright, o.ItemBg, true)))

	// Separator
	lines = append(lines, o.SepRow())

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
			lines = append(lines, o.WrapRow(o.PadLine(content, theme.PopupSelectedFg, o.SelectedBg, true)))
		} else {
			lines = append(lines, o.WrapRow(o.PadLine(content, theme.Text, o.ItemBg, false)))
		}
	}

	// Separator before hints
	lines = append(lines, o.SepRow())

	// Hints
	lines = append(lines, o.HintRow(" ↑↓ navigate  Enter select  Esc cancel"))

	lines = append(lines, o.BottomBorder)
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
