package editor

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/config/theme"
)

// renderAutocompletePopup renders the autocomplete popup box using manual borders
// with explicit backgrounds on every cell, matching the OverlayStyle pattern used
// by command menu, help, file picker, and export overlays. This prevents terminal
// colors from bleeding through the popup.
func (m Model) renderAutocompletePopup() string {
	if !m.autocompleteState.Visible || len(m.autocompleteState.Suggestions) == 0 {
		return ""
	}

	state := m.autocompleteState

	// Inner width = popup width minus 2 border characters (left + right)
	innerWidth := max(state.PopupWidth-2, 15)
	o := NewOverlayStyle(innerWidth)

	maxVisible := state.PopupHeight
	if maxVisible <= 0 {
		maxVisible = min(len(state.Suggestions), 8)
	}

	// Ensure scroll keeps selected item visible
	scrollTop := min(state.ScrollTop, state.Selected)
	if state.Selected >= scrollTop+maxVisible {
		scrollTop = state.Selected - maxVisible + 1
	}

	var lines []string
	lines = append(lines, o.TopBorder)

	// Render visible suggestion items
	for i := scrollTop; i < scrollTop+maxVisible && i < len(state.Suggestions); i++ {
		s := state.Suggestions[i]

		// Category tag prefix: fn/nl/var or abbreviated category name
		tag := suggestionTag(s.Category)

		// Format: prefer syntax (function signature) over description.
		// Syntax already includes the function name (e.g. "capacity(demand, ...)"),
		// so we use it directly to avoid showing the name twice.
		// Synonym info in Name (e.g. "avg (average)") is appended separately.
		var content string
		if s.Syntax != "" {
			content = " " + tag + " " + s.Syntax
			// Preserve synonym hint from Name if present, e.g. " (average)"
			if idx := strings.Index(s.Name, " ("); idx >= 0 {
				content += " " + s.Name[idx+1:]
			}
		} else if s.Description != "" {
			content = " " + tag + " " + s.Name + " " + s.Description
		} else {
			content = " " + tag + " " + s.Name
		}
		if len(content) > innerWidth {
			content = content[:innerWidth-1] + "…"
		}

		if i == state.Selected {
			lines = append(lines, o.WrapRow(o.PadLine(content, theme.PopupSelectedFg, o.SelectedBg, true)))
		} else {
			lines = append(lines, o.WrapRow(o.PadLine(content, theme.Text, o.ItemBg, false)))
		}
	}

	// Scroll indicator if needed
	if len(state.Suggestions) > maxVisible {
		indicator := fmt.Sprintf(" (%d/%d)", state.Selected+1, len(state.Suggestions))
		lines = append(lines, o.HintRow(indicator))
	}

	// Keyboard hints
	lines = append(lines, o.HintRow(" Tab ↑↓ Esc"))

	lines = append(lines, o.BottomBorder)
	return strings.Join(lines, "\n")
}

// calculatePopupScreenPosition computes where to place the popup on screen.
// Returns (row, col) as screen coordinates.
func (m Model) calculatePopupScreenPosition(contentHeight int) (row, col int) {
	// The cursor visual position in the content area
	visualCursorRow := m.cursorLine - m.scrollOffset

	// Account for source header row (1 line: "Source")
	headerRows := 1

	// Screen row for popup (below cursor)
	row = headerRows + visualCursorRow + 1

	// Ensure popup fits on screen
	// Height = top border (1) + items + hint (1) + bottom border (1) + optional scroll indicator (1)
	hasScroll := len(m.autocompleteState.Suggestions) > m.autocompleteState.PopupHeight
	popupHeight := m.autocompleteState.PopupHeight + 3 // borders (2) + hint (1)
	if hasScroll {
		popupHeight++ // scroll indicator line
	}
	if row+popupHeight > contentHeight {
		// Place above cursor instead
		row = max(headerRows+visualCursorRow-popupHeight, headerRows)
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

// suggestionTag returns a short tag for display in the popup (e.g., "fn", "nl", "var").
func suggestionTag(category string) string {
	switch category {
	case "example":
		return "nl"
	case "variable":
		return "var"
	case "Math", "Conversion", "Network", "Storage", "Capacity", "Growth":
		return "fn"
	default:
		// Unit categories and others: show abbreviated category
		if len(category) > 3 {
			return strings.ToLower(category[:3])
		}
		return strings.ToLower(category)
	}
}

// renderCommandMenuPopup renders the command menu as a popup overlay.
func (m Model) renderCommandMenuPopup() string {
	commands := EditorCommands
	selected := m.commandMenuState.Selected

	o := NewOverlayStyle(44)

	var lines []string
	lines = append(lines, o.TopBorder)

	// Title row
	lines = append(lines, o.WrapRow(o.PadLine(" Commands", theme.TextBright, o.ItemBg, true)))

	// Separator
	lines = append(lines, o.SepRow())

	// Command items — show accelerator and name
	for i, cmd := range commands {
		content := fmt.Sprintf(" %-12s %s", cmd.Accelerator, cmd.Name)

		if i == selected {
			lines = append(lines, o.WrapRow(o.PadLine(content, theme.PopupSelectedFg, o.SelectedBg, true)))
		} else {
			lines = append(lines, o.WrapRow(o.PadLine(content, theme.Text, o.ItemBg, false)))
		}
	}

	// Hint row
	lines = append(lines, o.HintRow(" ↑↓ navigate  Enter select  Esc close"))

	lines = append(lines, o.BottomBorder)
	return strings.Join(lines, "\n")
}

// renderFilePickerOverlay renders the file picker as a modal overlay
// with fixed-width manual borders matching other overlays.
func (m Model) renderFilePickerOverlay() string {
	o := NewOverlayStyle(70)

	// truncatePath shortens a directory path to fit within maxLen,
	// keeping the end (most relevant part) with leading "..."
	truncatePath := func(path string, maxLen int) string {
		if len(path) <= maxLen {
			return path
		}
		return "..." + path[len(path)-(maxLen-3):]
	}

	var lines []string
	lines = append(lines, o.TopBorder)

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
	maxPathLen := o.InnerWidth - len(titlePrefix) - 1
	dirPath := truncatePath(m.filePicker.CurrentDirectory, maxPathLen)
	lines = append(lines, o.WrapRow(o.PadLine(titlePrefix+dirPath, theme.TextBright, o.ItemBg, true)))

	// Format subheading for export mode
	if m.filePickerPurpose == PickerForExport {
		format := m.exportFormatOpts[m.exportState.FormatIdx]
		ext := formatToExtension(format)
		content := fmt.Sprintf(" Format: %s (%s)", format, ext)
		lines = append(lines, o.WrapRow(o.PadLine(content, theme.TextMuted, o.ItemBg, false)))
	}

	// Separator
	lines = append(lines, o.SepRow())

	// File picker view — split into lines, pad each with ANSI-aware padding.
	// The filepicker View() output contains ANSI escape codes for directory
	// coloring and cursor highlighting, so we use overlayPadLine which strips
	// ANSI resets and applies background consistently.
	pickerView := m.filePicker.View()

	for pl := range strings.SplitSeq(pickerView, "\n") {
		content := " " + pl
		if m.filePickerFocus == FocusFilename {
			// Dim the browser when filename has focus
			dimStyle := lipgloss.NewStyle().Foreground(theme.Hint).Background(o.ItemBg)
			stripped := stripResetCodes(content)
			visualWidth := lipgloss.Width(stripped)
			if visualWidth < o.InnerWidth {
				stripped += strings.Repeat(" ", o.InnerWidth-visualWidth)
			}
			lines = append(lines, o.WrapRow(dimStyle.Render(stripped)))
		} else {
			// Active browser — use overlayPadLine for ANSI-safe background
			lines = append(lines, o.WrapRow(overlayPadLine(content, o.InnerWidth, o.ItemBg)))
		}
	}

	// Separator before filename
	lines = append(lines, o.SepRow())

	// Filename input field
	if m.filePickerFocus == FocusFilename {
		cursor := "█"
		display := m.newFileName + cursor
		lines = append(lines, o.WrapRow(o.PadLine(" Filename: "+display, theme.InputFg, theme.InputBg, false)))
	} else {
		display := m.newFileName
		if display == "" {
			display = "(type filename)"
		}
		lines = append(lines, o.WrapRow(o.PadLine(" Filename: "+display, theme.Hint, o.ItemBg, false)))
	}

	// Empty line
	lines = append(lines, o.WrapRow(o.PadLine("", theme.Hint, o.ItemBg, false)))

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
	lines = append(lines, o.HintRow(hint))

	lines = append(lines, o.BottomBorder)
	return strings.Join(lines, "\n")
}
