package components

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/config/theme"
	"github.com/charmbracelet/lipgloss"
)

// StyledPadding creates styled spaces with the given background color.
// This is a pure function that prevents terminal default background bleed-through.
// Use this whenever you need spacing/padding between UI elements.
func StyledPadding(width int, bg lipgloss.TerminalColor) string {
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

// statusBarHeight is the fixed height for all status bar renderings.
// This must be consistent to avoid bubbletea rendering artifacts.
// See: https://github.com/charmbracelet/bubbletea/issues/1004
const statusBarHeight = 2

// RenderStatusBar renders a status bar as a string.
// Pure function: takes state and width, returns string.
// IMPORTANT: Always returns consistent height (statusBarHeight lines) to avoid
// bubbletea rendering artifacts when view height changes between renders.
func RenderStatusBar(state StatusBarState, width int, style StatusBarStyle) string {
	// If there's a status message, show it prominently
	if state.StatusMsg != "" {
		var msgStyle lipgloss.Style
		if state.StatusIsErr {
			msgStyle = style.StatusErr
		} else {
			msgStyle = style.StatusOK
		}
		return style.Bar.Width(width).Height(statusBarHeight).Render(msgStyle.Render(state.StatusMsg))
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

	// Build center section: position info with column and optional EVAL indicator
	var centerText string
	if state.EvalInProgress {
		centerText = fmt.Sprintf("L%d:%d | EVAL...", state.Line, state.Column)
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

	// Get the background from Bar style for padding
	barBg := style.Bar.GetBackground()

	// Account for Bar style's horizontal padding when computing content width.
	// Bar has Padding(0, 1) = 2 chars total, so content must fit in (width - 2).
	barHPad := style.Bar.GetHorizontalPadding()
	contentWidth := width - barHPad

	// If there's room, space things out
	if totalContent < contentWidth-4 {
		padding := (contentWidth - totalContent) / 2
		leftPad := StyledPadding(padding, barBg)
		rightPad := StyledPadding(contentWidth-totalContent-padding, barBg)
		return style.Bar.Width(width).Height(statusBarHeight).Render(leftStr + leftPad + center + rightPad + rightStr)
	}

	// Otherwise, truncate hints first - use styled spaces
	styledSpace := StyledPadding(1, barBg)
	return style.Bar.Width(width).Height(statusBarHeight).Render(leftStr + styledSpace + center + styledSpace + rightStr)
}

// RenderMinimalStatusBar renders a compact status bar for narrow terminals.
func RenderMinimalStatusBar(state StatusBarState, width int, style StatusBarStyle) string {
	// Just show filename (truncated if needed) and modified indicator
	name := state.Filename
	if name == "" {
		name = "[New]"
	}

	maxNameLen := width - 10
	if len(name) > maxNameLen && maxNameLen > 3 {
		name = name[:maxNameLen-3] + "..."
	}

	result := style.Filename.Render(name)
	if state.Modified {
		result += style.Modified.Render(" [+]")
	}

	return style.Bar.Width(width).Render(result)
}
