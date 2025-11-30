package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// StatusBarState holds the data needed to render a status bar.
type StatusBarState struct {
	Filename    string // Current file name (empty for new/unsaved)
	Line        int    // Current line number (1-indexed)
	TotalLines  int    // Total lines in document
	CalcCount   int    // Number of calc blocks
	Modified    bool   // Whether the document has unsaved changes
	Hints       string // Context-sensitive hints
	Mode        string // Current mode name (e.g., "NORMAL", "EDITING")
	StatusMsg   string // Status message (e.g., "Saved: file.cm")
	StatusIsErr bool   // Whether status message is an error
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
func DefaultStatusBarStyle() StatusBarStyle {
	return StatusBarStyle{
		Bar: lipgloss.NewStyle().
			Background(lipgloss.Color("#333333")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 1),
		Filename: lipgloss.NewStyle().
			Bold(true),
		Modified: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B")),
		Position: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")),
		Mode: lipgloss.NewStyle().
			Background(lipgloss.Color("#4ECDC4")).
			Foreground(lipgloss.Color("#000000")).
			Padding(0, 1),
		Hints: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Italic(true),
		StatusOK: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4ECDC4")),
		StatusErr: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B")),
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

	// Build center section: position info
	center := style.Position.Render(
		fmt.Sprintf("L%d/%d | %d calcs", state.Line, state.TotalLines, state.CalcCount),
	)

	// Build right section: mode + hints
	var right strings.Builder
	if state.Mode != "" {
		right.WriteString(style.Mode.Render(state.Mode))
		right.WriteString(" ")
	}
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

	// If there's room, space things out
	if totalContent < width-4 {
		padding := (width - totalContent) / 2
		leftPad := strings.Repeat(" ", padding)
		rightPad := strings.Repeat(" ", width-totalContent-padding)
		return style.Bar.Width(width).Height(statusBarHeight).Render(leftStr + leftPad + center + rightPad + rightStr)
	}

	// Otherwise, truncate hints first
	return style.Bar.Width(width).Height(statusBarHeight).Render(leftStr + " " + center + " " + rightStr)
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
