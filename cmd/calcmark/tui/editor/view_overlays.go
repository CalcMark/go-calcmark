package editor

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/components"
	"github.com/charmbracelet/lipgloss"
)

// renderGlobalsPanel renders the collapsible globals panel.
func (m Model) renderGlobalsPanel(width int) string {
	state := m.GetGlobalsPanelState()
	globalsCount := len(state.Globals)
	hasError := state.Error != ""

	if !state.Expanded {
		// Collapsed: show count (or warning if error)
		indicator := "▸"
		var text string
		if hasError {
			text = " Globals ⚠"
		} else {
			text = fmt.Sprintf(" Globals (%d)", globalsCount)
		}
		hint := "[g]"

		headerFg := lipgloss.Color("252")
		if hasError {
			headerFg = lipgloss.Color("208") // amber for error
		}

		left := lipgloss.NewStyle().
			Bold(true).
			Foreground(headerFg).
			Render(indicator + text)

		right := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render(hint)

		// Space between left and right with background
		space := width - lipgloss.Width(left) - lipgloss.Width(right)
		if space < 0 {
			space = 0
		}

		// Use centralized StyledPadding utility
		header := left + components.StyledPadding(space, lipgloss.Color("236")) + right
		return ensureFullWidth(header, width, lipgloss.Color("236"))
	}

	// Expanded: show all globals
	var allLines []string

	indicator := "▾"
	text := " Globals"
	hint := "[g]"

	headerFg := lipgloss.Color("252")
	if hasError {
		headerFg = lipgloss.Color("208")
		text = " Globals ⚠"
	}

	left := lipgloss.NewStyle().
		Bold(true).
		Foreground(headerFg).
		Render(indicator + text)

	right := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(hint)

	space := width - lipgloss.Width(left) - lipgloss.Width(right)
	if space < 0 {
		space = 0
	}

	// Use centralized StyledPadding utility
	headerLine := left + components.StyledPadding(space, lipgloss.Color("236")) + right
	headerLine = ensureFullWidth(headerLine, width, lipgloss.Color("236"))
	allLines = append(allLines, headerLine)

	// Show error details when YAML is malformed
	if hasError {
		errStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("208")).
			Italic(true)
		errLine := errStyle.Render("  " + state.Error)
		errLine = ensureFullWidth(errLine, width, lipgloss.Color("236"))
		allLines = append(allLines, errLine)
		return strings.Join(allLines, "\n")
	}

	if globalsCount == 0 {
		noGlobalsLine := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true).
			Render("  (no globals defined)")
		noGlobalsLine = ensureFullWidth(noGlobalsLine, width, lipgloss.Color("236"))
		allLines = append(allLines, noGlobalsLine)
		return strings.Join(allLines, "\n")
	}

	for i, g := range state.Globals {
		prefix := "  "
		if state.Focused && i == state.FocusIndex {
			prefix = "> "
		}

		nameStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
		valueStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("6"))

		if g.IsExchange {
			nameStyle = nameStyle.Foreground(lipgloss.Color("5"))
		}

		// Format: "  name          value"
		name := fmt.Sprintf("%-18s", g.Name)
		globalLine := prefix + nameStyle.Render(name) + valueStyle.Render(g.Value)
		globalLine = ensureFullWidth(globalLine, width, lipgloss.Color("236"))
		allLines = append(allLines, globalLine)
	}

	return strings.Join(allLines, "\n")
}

// renderAutocompletePopup renders the autocomplete popup box.
// Returns the popup as a styled string with border.
func (m Model) renderAutocompletePopup() string {
	if !m.autocompleteState.Visible || len(m.autocompleteState.Suggestions) == 0 {
		return ""
	}

	style := components.DefaultPopupStyle()
	return components.RenderPopupBox(m.autocompleteState, style)
}

// calculatePopupScreenPosition computes where to place the popup on screen.
// Returns (row, col) as screen coordinates.
func (m Model) calculatePopupScreenPosition(contentHeight int) (row, col int) {
	// The cursor visual position in the content area
	visualCursorRow := m.cursorLine - m.scrollOffset

	// Account for headers: source header (1) + globals padding if preview visible
	headerRows := 1
	if m.previewMode != PreviewHidden {
		globalsHeight := 1
		if m.globalsExpanded {
			globalsHeight = 1 + m.getGlobalsCount()
			if m.getGlobalsCount() == 0 {
				globalsHeight = 2
			}
		}
		globalsHeight++ // separator
		headerRows += globalsHeight
	}

	// Screen row for popup (below cursor)
	row = headerRows + visualCursorRow + 1

	// Ensure popup fits on screen
	popupHeight := m.autocompleteState.PopupHeight + 2 // +2 for hint and border
	if row+popupHeight > contentHeight {
		// Place above cursor instead
		row = headerRows + visualCursorRow - popupHeight
		if row < headerRows {
			row = headerRows
		}
	}

	// Column: align with cursor, adjusted for line number gutter
	gutterWidth := 5 // "  N→" format
	col = gutterWidth + m.cursorCol

	// Ensure popup doesn't go off right edge
	leftWidth, _ := m.GetPaneWidths(m.width)
	if col+m.autocompleteState.PopupWidth > leftWidth {
		col = leftWidth - m.autocompleteState.PopupWidth
	}
	if col < gutterWidth {
		col = gutterWidth
	}

	return row, col
}

// renderCommandMenuPopup renders the command menu as a popup overlay.
func (m Model) renderCommandMenuPopup() string {
	commands := EditorCommands
	selected := m.commandMenuState.Selected

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

	// Title row
	lines = append(lines, leftBorder+pad(" Commands", "#FFFFFF", itemBg, true)+rightBorder)

	// Separator
	lines = append(lines, leftBorder+sepLine+rightBorder)

	// Command items — show accelerator and name
	for i, cmd := range commands {
		content := fmt.Sprintf(" %-12s %s", cmd.Accelerator, cmd.Name)

		if i == selected {
			lines = append(lines, leftBorder+pad(content, "#FFFFFF", selectedBg, true)+rightBorder)
		} else {
			lines = append(lines, leftBorder+pad(content, "#CCCCCC", itemBg, false)+rightBorder)
		}
	}

	// Hint row
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Background(itemBg).
		Italic(true)
	hint := " ↑↓ navigate  Enter select  Esc close"
	visualWidth := lipgloss.Width(hint)
	if visualWidth < innerWidth {
		hint += strings.Repeat(" ", innerWidth-visualWidth)
	}
	lines = append(lines, leftBorder+hintStyle.Render(hint)+rightBorder)

	lines = append(lines, bottomBorder)
	return strings.Join(lines, "\n")
}

// renderFilePickerOverlay renders the file picker as a modal overlay
// with fixed-width manual borders matching other overlays.
func (m Model) renderFilePickerOverlay() string {
	innerWidth := 70

	// Border and background colors (shared palette across all overlays)
	borderFg := lipgloss.Color("#5C5C5C")
	borderStyle := lipgloss.NewStyle().Foreground(borderFg)
	itemBg := lipgloss.Color("#1E1E1E")

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

	// truncatePath shortens a directory path to fit within maxLen,
	// keeping the end (most relevant part) with leading "..."
	truncatePath := func(path string, maxLen int) string {
		if len(path) <= maxLen {
			return path
		}
		return "..." + path[len(path)-(maxLen-3):]
	}

	var lines []string
	lines = append(lines, topBorder)

	// Purpose-aware title with truncated directory path
	var titlePrefix string
	switch m.filePickerPurpose {
	case PickerForOpen:
		titlePrefix = " Open: "
	case PickerForExport:
		titlePrefix = " Export to: "
	default:
		titlePrefix = " Save to: "
	}
	maxPathLen := innerWidth - len(titlePrefix) - 1
	dirPath := truncatePath(m.filePicker.CurrentDirectory, maxPathLen)
	lines = append(lines, leftBorder+pad(titlePrefix+dirPath, "#FFFFFF", itemBg, true)+rightBorder)

	// Format subheading for export mode
	if m.filePickerPurpose == PickerForExport {
		format := m.exportFormatOpts[m.exportState.FormatIdx]
		ext := formatToExtension(format)
		content := fmt.Sprintf(" Format: %s (%s)", format, ext)
		formatStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#AAAAAA")).Background(itemBg).Italic(true)
		visualWidth := lipgloss.Width(content)
		if visualWidth < innerWidth {
			content += strings.Repeat(" ", innerWidth-visualWidth)
		}
		lines = append(lines, leftBorder+formatStyle.Render(content)+rightBorder)
	}

	// Separator
	lines = append(lines, leftBorder+sepLine+rightBorder)

	// File picker view — split into lines, pad each with ANSI-aware padding.
	// The filepicker View() output contains ANSI escape codes for directory
	// coloring and cursor highlighting, so we use overlayPadLine which strips
	// ANSI resets and applies background consistently.
	pickerView := m.filePicker.View()
	pickerLines := strings.Split(pickerView, "\n")

	dimBg := lipgloss.Color("#1E1E1E")
	for _, pl := range pickerLines {
		content := " " + pl
		if m.filePickerFocus == FocusFilename {
			// Dim the browser when filename has focus
			dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Background(dimBg)
			stripped := stripResetCodes(content)
			visualWidth := lipgloss.Width(stripped)
			if visualWidth < innerWidth {
				stripped += strings.Repeat(" ", innerWidth-visualWidth)
			}
			lines = append(lines, leftBorder+dimStyle.Render(stripped)+rightBorder)
		} else {
			// Active browser — use overlayPadLine for ANSI-safe background
			lines = append(lines, leftBorder+overlayPadLine(content, innerWidth, itemBg)+rightBorder)
		}
	}

	// Separator before filename
	lines = append(lines, leftBorder+sepLine+rightBorder)

	// Filename input field
	activeInputBg := lipgloss.Color("#2A2A3A")

	if m.filePickerFocus == FocusFilename {
		cursor := "█"
		display := m.newFileName + cursor
		lines = append(lines, leftBorder+pad(" Filename: "+display, "#FFFFFF", activeInputBg, false)+rightBorder)
	} else {
		display := m.newFileName
		if display == "" {
			display = "(type filename)"
		}
		lines = append(lines, leftBorder+pad(" Filename: "+display, "#666666", itemBg, false)+rightBorder)
	}

	// Empty line
	lines = append(lines, leftBorder+pad("", "#666666", itemBg, false)+rightBorder)

	// Purpose-aware hints
	var hint string
	switch m.filePickerPurpose {
	case PickerForOpen:
		hint = " Tab switch  ← up dir  Enter open  Esc cancel"
	case PickerForExport:
		hint = " Tab switch  ← up dir  Enter export  Esc cancel"
	default:
		hint = " Tab switch  ← up dir  Enter save  Esc cancel"
	}
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Background(itemBg).
		Italic(true)
	visualWidth := lipgloss.Width(hint)
	if visualWidth < innerWidth {
		hint += strings.Repeat(" ", innerWidth-visualWidth)
	}
	lines = append(lines, leftBorder+hintStyle.Render(hint)+rightBorder)

	lines = append(lines, bottomBorder)
	return strings.Join(lines, "\n")
}
