package components

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/config/theme"
)

// StyledPadding creates styled spaces with the given background color.
// This is a pure function that prevents terminal default background bleed-through.
// Use this whenever you need spacing/padding between UI elements.
func StyledPadding(width int, bg color.Color) string {
	if width <= 0 {
		return ""
	}
	style := lipgloss.NewStyle()
	if _, ok := bg.(lipgloss.NoColor); !ok {
		style = style.Background(bg)
	}
	return style.Render(strings.Repeat(" ", width))
}

// StatusBarState holds the data needed to render a status bar.
type StatusBarState struct {
	Filename       string // Current file name (empty for new/unsaved)
	Line           int    // Current line number (1-indexed)
	Column         int    // Current column number (1-indexed)
	TotalLines     int    // Total lines in document
	CalcCount      int    // Number of calc blocks
	Modified       bool   // Whether the document has unsaved changes
	Hints          string // Context-sensitive hints
	Mode           string // Current mode name (e.g., "NORMAL", "EDITING")
	StatusMsg      string // Status message (e.g., "Saved: file.cm")
	StatusIsErr    bool   // Whether status message is an error
	EvalInProgress bool   // True during evaluation debounce period
	SelectionCount int    // Number of selected characters (0 = no selection)
}

// StatusBarStyle holds styles for rendering the status bar.
type StatusBarStyle struct {
	Bar       lipgloss.Style // Overall bar style
	Filename  lipgloss.Style // Filename style
	Modified  lipgloss.Style // Modified indicator style
	Position  lipgloss.Style // Line position style
	Mode      lipgloss.Style // Mode indicator style
	Hints     lipgloss.Style // Hints style
	StatusOK  lipgloss.Style // Status message (success)
	StatusErr lipgloss.Style // Status message (error)
}

// DefaultStatusBarStyle returns the default status bar styling.
// Note: The Bar background should be overridden with the themed color.
func DefaultStatusBarStyle() StatusBarStyle {
	return StatusBarStyle{
		Bar: lipgloss.NewStyle().
			Foreground(theme.StatusFg).
			Padding(0, 1),
		Filename: lipgloss.NewStyle().
			Foreground(theme.StatusFg).
			Bold(true),
		Modified: lipgloss.NewStyle().
			Foreground(theme.Error),
		Position: lipgloss.NewStyle().
			Foreground(theme.TextMuted),
		Mode: lipgloss.NewStyle().
			Background(theme.ModeIndicatorBg).
			Foreground(theme.ModeIndicatorFg).
			Padding(0, 1),
		Hints: lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Italic(true),
		StatusOK: lipgloss.NewStyle().
			Foreground(theme.Success),
		StatusErr: lipgloss.NewStyle().
			Foreground(theme.Error),
	}
}

// StatusBarHeight is the fixed height for all status bar renderings.
// This must be consistent to avoid bubbletea rendering artifacts.
// See: https://charm.land/bubbletea/v2/issues/1004
const StatusBarHeight = 2

// RenderStatusBar renders a status bar as a string.
// Pure function: takes state and width, returns string.
// IMPORTANT: Always returns consistent height (StatusBarHeight lines) to avoid
// bubbletea rendering artifacts when view height changes between renders.
//
// Background strategy: each segment and all padding is rendered with explicit
// backgrounds. We do NOT rely on lipgloss outer Render() for background
// coverage because inner ANSI resets from sub-components clear the outer
// background, causing terminal color bleed-through.
func RenderStatusBar(state StatusBarState, width int, style StatusBarStyle) string {
	barBg := style.Bar.GetBackground()

	// Helper: build a line from styled segments, padded to full width with barBg
	buildLine := func(content string, contentVisualWidth int) string {
		// Add 1-char padding on each side (matching Bar's Padding(0,1))
		line := StyledPadding(1, barBg) + content
		currentWidth := 1 + contentVisualWidth
		remaining := width - currentWidth
		if remaining > 0 {
			line += StyledPadding(remaining, barBg)
		}
		return line
	}

	// Status message takes the full line
	if state.StatusMsg != "" {
		var msgStyle lipgloss.Style
		if state.StatusIsErr {
			msgStyle = style.StatusErr
		} else {
			msgStyle = style.StatusOK
		}
		// Truncate long messages to fit within the bar width.
		// Account for 1-char left padding added by buildLine.
		maxMsgWidth := width - 2
		msg := TruncateWithEllipsis(state.StatusMsg, maxMsgWidth)
		msgStr := msgStyle.Render(msg)
		line1 := buildLine(msgStr, lipgloss.Width(msg))
		line2 := StyledPadding(width, barBg)
		return line1 + "\n" + line2
	}

	// Build left section: filename + modified indicator
	var left strings.Builder
	if state.Filename != "" {
		left.WriteString(style.Filename.Render(state.Filename))
	} else {
		left.WriteString(style.Filename.Render("[New]"))
	}
	if state.Modified {
		left.WriteString(style.Modified.Render(" [+]"))
	}

	// Build center section: position info with column, selection count, and optional EVAL indicator
	var centerText string
	if state.EvalInProgress {
		centerText = fmt.Sprintf("L%d:%d | EVAL...", state.Line, state.Column)
	} else if state.SelectionCount > 0 {
		centerText = fmt.Sprintf("L%d:%d | %d selected", state.Line, state.Column, state.SelectionCount)
	} else {
		centerText = fmt.Sprintf("L%d:%d | %d calcs", state.Line, state.Column, state.CalcCount)
	}
	center := style.Position.Render(centerText)

	// Build right section: hints only (mode is internal implementation detail)
	var right strings.Builder
	if state.Hints != "" {
		right.WriteString(style.Hints.Render(state.Hints))
	}

	leftStr := left.String()
	rightStr := right.String()

	// Calculate spacing
	leftWidth := lipgloss.Width(leftStr)
	centerWidth := lipgloss.Width(center)
	rightWidth := lipgloss.Width(rightStr)
	totalContent := leftWidth + centerWidth + rightWidth

	// Content width = total width minus Bar's horizontal padding (1 on each side)
	barHPad := style.Bar.GetHorizontalPadding()
	contentWidth := width - barHPad

	// Assemble line 1 with explicit styled padding between segments
	var line1 string
	if totalContent < contentWidth-4 {
		// Spread: left | pad | center | pad | right
		padding := (contentWidth - totalContent) / 2
		leftPad := StyledPadding(padding, barBg)
		rightPad := StyledPadding(contentWidth-totalContent-padding, barBg)
		line1 = buildLine(leftStr+leftPad+center+rightPad+rightStr, contentWidth)
	} else {
		// Tight: left | 1sp | center | 1sp | right
		styledSpace := StyledPadding(1, barBg)
		visWidth := totalContent + 2
		line1 = buildLine(leftStr+styledSpace+center+styledSpace+rightStr, visWidth)
	}

	// Line 2: empty line with full background (consistent StatusBarHeight)
	line2 := StyledPadding(width, barBg)

	return line1 + "\n" + line2
}

// RenderMinimalStatusBar renders a compact status bar for narrow terminals.
func RenderMinimalStatusBar(state StatusBarState, width int, style StatusBarStyle) string {
	barBg := style.Bar.GetBackground()

	// Just show filename (truncated if needed) and modified indicator
	name := state.Filename
	if name == "" {
		name = "[New]"
	}

	maxNameLen := width - 10
	if len(name) > maxNameLen && maxNameLen > 3 {
		name = name[:maxNameLen-3] + "..."
	}

	content := style.Filename.Render(name)
	contentWidth := lipgloss.Width(name)
	if state.Modified {
		content += style.Modified.Render(" [+]")
		contentWidth += 4
	}

	// Build with explicit padding — 1 char left pad, then content, then fill
	line := StyledPadding(1, barBg) + content
	remaining := width - 1 - contentWidth
	if remaining > 0 {
		line += StyledPadding(remaining, barBg)
	}
	return line
}
