package config

import "github.com/charmbracelet/lipgloss"

// Styles holds pre-built lipgloss styles derived from theme config.
// This avoids rebuilding styles on every render call.
type Styles struct {
	Title         lipgloss.Style
	PinnedPanel   lipgloss.Style
	Error         lipgloss.Style
	ErrorOutput   lipgloss.Style
	Help          lipgloss.Style
	Prompt        lipgloss.Style
	Output        lipgloss.Style
	Changed       lipgloss.Style
	Var           lipgloss.Style
	Hint          lipgloss.Style
	Header        lipgloss.Style
	Syntax        lipgloss.Style
	Example       lipgloss.Style
	Separator     lipgloss.Style
	ModeIndicator lipgloss.Style
}

// BuildStyles creates lipgloss.Style instances from ThemeConfig.
// Call this once after loading config, then reuse the Styles struct.
func (t ThemeConfig) BuildStyles() Styles {
	return Styles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(t.Primary)).
			Margin(1, 0),

		PinnedPanel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(t.Accent)).
			Padding(0, 1).
			Margin(1, 0),

		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Error)),

		ErrorOutput: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Error)),

		Help: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Muted)).
			Margin(1, 0),

		Prompt: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Primary)),

		Output: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Output)),

		Changed: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Warning)),

		Var: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Primary)),

		Hint: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Dimmed)).
			Italic(true),

		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(t.Primary)),

		Syntax: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(t.Bright)),

		Example: lipgloss.NewStyle().
			Italic(true).
			Foreground(lipgloss.Color(t.Dimmed)),

		Separator: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Separator)),

		ModeIndicator: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.Primary)),
	}
}
