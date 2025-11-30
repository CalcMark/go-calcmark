package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// PinnedVar represents a pinned variable for display.
type PinnedVar struct {
	Name          string // Variable name
	Value         string // Formatted value
	Changed       bool   // Was modified in last operation
	IsFrontmatter bool   // Is this a frontmatter variable
}

// PinnedPanelState holds the state for the pinned variables panel.
type PinnedPanelState struct {
	Variables []PinnedVar
	ScrollY   int // Current scroll position
	Height    int // Visible height
}

// PinnedPanelStyle holds styles for rendering the pinned panel.
type PinnedPanelStyle struct {
	Container   lipgloss.Style // Panel container
	Header      lipgloss.Style // Panel header
	VarName     lipgloss.Style // Variable name
	VarValue    lipgloss.Style // Variable value
	Changed     lipgloss.Style // Changed indicator and value
	Frontmatter lipgloss.Style // Frontmatter indicator (@)
	Empty       lipgloss.Style // Empty state message
	ScrollHint  lipgloss.Style // Scroll indicator
}

// DefaultPinnedPanelStyle returns the default pinned panel styling.
func DefaultPinnedPanelStyle() PinnedPanelStyle {
	return PinnedPanelStyle{
		Container: lipgloss.NewStyle().
			BorderStyle(lipgloss.Border{Left: "│"}).
			BorderForeground(lipgloss.Color("#444444")).
			PaddingLeft(1),
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")),
		VarName: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4ECDC4")),
		VarValue: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")),
		Changed: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD93D")),
		Frontmatter: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")),
		Empty: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Italic(true),
		ScrollHint: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")),
	}
}

// RenderPinnedPanel renders the pinned variables panel.
// Pure function: takes state, width, and height, returns string.
func RenderPinnedPanel(state PinnedPanelState, width int, style PinnedPanelStyle) string {
	var b strings.Builder

	// Header
	b.WriteString(style.Header.Render("📌 Pinned Variables"))
	b.WriteString("\n\n")

	if len(state.Variables) == 0 {
		b.WriteString(style.Empty.Render("(no variables pinned)"))
		return style.Container.Width(width).Render(b.String())
	}

	// Calculate visible range
	visibleStart := state.ScrollY
	visibleEnd := min(visibleStart+state.Height, len(state.Variables))

	// Render visible variables
	for i := visibleStart; i < visibleEnd; i++ {
		v := state.Variables[i]

		// Build display name with optional @ prefix
		displayName := v.Name
		if v.IsFrontmatter {
			displayName = style.Frontmatter.Render("@") + v.Name
		}

		var line string
		if v.Changed {
			line = style.Changed.Render("* ")
			line += style.VarName.Bold(true).Render(displayName)
			line += " = "
			line += style.Changed.Render(v.Value)
		} else {
			line = "  "
			line += style.VarName.Render(displayName)
			line += " = "
			line += style.VarValue.Render(v.Value)
		}

		b.WriteString(line)
		if i < visibleEnd-1 {
			b.WriteString("\n")
		}
	}

	// Scroll indicator
	if len(state.Variables) > state.Height {
		scrollPct := 0
		if len(state.Variables)-state.Height > 0 {
			scrollPct = state.ScrollY * 100 / (len(state.Variables) - state.Height)
		}
		b.WriteString("\n")
		b.WriteString(style.ScrollHint.Render(fmt.Sprintf("↑↓ %d%%", scrollPct)))
	}

	return style.Container.Width(width).Render(b.String())
}

// RenderCompactPinned renders a single-line summary of pinned variables.
func RenderCompactPinned(state PinnedPanelState, width int, style PinnedPanelStyle) string {
	if len(state.Variables) == 0 {
		return style.Empty.Render("No pinned vars")
	}

	var parts []string
	totalWidth := 0

	for _, v := range state.Variables {
		part := v.Name + "=" + v.Value
		partWidth := len(part) + 2 // +2 for ", "

		if totalWidth+partWidth > width-10 {
			parts = append(parts, "...")
			break
		}

		if v.Changed {
			parts = append(parts, style.Changed.Render(part))
		} else {
			parts = append(parts, style.VarName.Render(v.Name)+"="+style.VarValue.Render(v.Value))
		}
		totalWidth += partWidth
	}

	return strings.Join(parts, ", ")
}
