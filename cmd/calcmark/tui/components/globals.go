package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// GlobalVar represents a global variable for display.
type GlobalVar struct {
	Name       string // Variable name
	Value      string // Formatted value
	Expression string // Original expression (if any)
	IsExchange bool   // Is this an exchange rate?
}

// GlobalsPanelState holds the state for the globals panel.
type GlobalsPanelState struct {
	Globals    []GlobalVar
	Expanded   bool // Whether the panel is expanded
	FocusIndex int  // Currently focused item (-1 for none)
	Focused    bool // Whether this panel has focus
}

// GlobalsPanelStyle holds styles for rendering the globals panel.
type GlobalsPanelStyle struct {
	Container  lipgloss.Style // Panel container
	Header     lipgloss.Style // Panel header
	HeaderIcon lipgloss.Style // Header icon/emoji
	VarName    lipgloss.Style // Variable name
	VarValue   lipgloss.Style // Variable value
	Exchange   lipgloss.Style // Exchange rate indicator
	Focused    lipgloss.Style // Focused item
	Collapsed  lipgloss.Style // Collapsed indicator
}

// DefaultGlobalsPanelStyle returns the default globals panel styling.
func DefaultGlobalsPanelStyle() GlobalsPanelStyle {
	return GlobalsPanelStyle{
		Container: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#444444")).
			Padding(0, 1),
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")),
		HeaderIcon: lipgloss.NewStyle().
			MarginRight(1),
		VarName: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4ECDC4")),
		VarValue: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")),
		Exchange: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD93D")),
		Focused: lipgloss.NewStyle().
			Background(lipgloss.Color("#333333")),
		Collapsed: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Italic(true),
	}
}

// RenderGlobalsPanel renders the globals panel.
// Pure function: takes state and dimensions, returns string.
func RenderGlobalsPanel(state GlobalsPanelState, width int, style GlobalsPanelStyle) string {
	var b strings.Builder

	// Header
	icon := "⚙"
	if state.Focused {
		icon = "▶"
	}
	header := style.HeaderIcon.Render(icon) + style.Header.Render("Globals")

	if !state.Expanded {
		// Collapsed view
		count := len(state.Globals)
		hint := style.Collapsed.Render(fmt.Sprintf("(%d items, press g to expand)", count))
		return style.Container.Width(width).Render(header + " " + hint)
	}

	b.WriteString(header)
	b.WriteString("\n")

	if len(state.Globals) == 0 {
		b.WriteString(style.Collapsed.Render("(no globals defined)"))
		return style.Container.Width(width).Render(b.String())
	}

	// Variable list
	for i, g := range state.Globals {
		var line string

		// Name with optional @ prefix for exchange rates
		name := g.Name
		if g.IsExchange {
			name = "@exchange." + name
			line = style.Exchange.Render(name)
		} else {
			name = "@global." + name
			line = style.VarName.Render(name)
		}

		line += " = " + style.VarValue.Render(g.Value)

		// Apply focus styling if this is the focused item
		if state.Focused && i == state.FocusIndex {
			line = style.Focused.Render("→ " + line)
		} else {
			line = "  " + line
		}

		b.WriteString(line)
		if i < len(state.Globals)-1 {
			b.WriteString("\n")
		}
	}

	return style.Container.Width(width).Render(b.String())
}

// RenderCollapsedGlobals renders a minimal one-line globals summary.
func RenderCollapsedGlobals(state GlobalsPanelState, width int, style GlobalsPanelStyle) string {
	if len(state.Globals) == 0 {
		return ""
	}

	// Show first few globals inline
	var parts []string
	for i, g := range state.Globals {
		if i >= 3 {
			break
		}
		name := g.Name
		if g.IsExchange {
			parts = append(parts, style.Exchange.Render("@"+name))
		} else {
			parts = append(parts, style.VarName.Render("@"+name))
		}
	}

	content := strings.Join(parts, ", ")
	if len(state.Globals) > 3 {
		content += style.Collapsed.Render(fmt.Sprintf(" +%d more", len(state.Globals)-3))
	}

	return content
}
