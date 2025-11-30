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

	// Editor styles
	EditLine    lipgloss.Style // Background for line being edited
	Cursor      lipgloss.Style // Cursor style
	CurrentLine lipgloss.Style // Current line highlight in normal mode
	LineNumber  lipgloss.Style // Line number style

	// Markdown preview styles
	MdText   lipgloss.Style // Body text
	MdH1     lipgloss.Style // H1 heading
	MdH2     lipgloss.Style // H2 heading
	MdH3Plus lipgloss.Style // H3+ headings
	MdLink   lipgloss.Style // Links
	MdQuote  lipgloss.Style // Block quotes
	MdCode   lipgloss.Style // Inline code
	MdCodeBg lipgloss.Style // Code with background
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
			Border(lipgloss.NormalBorder(), false, false, false, true). // Left border only
			BorderForeground(lipgloss.Color(t.Accent)).
			PaddingLeft(1),

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

		// Editor styles
		EditLine: lipgloss.NewStyle().
			Background(lipgloss.Color(t.EditLineBg)).
			Foreground(lipgloss.Color(t.EditLineFg)),

		Cursor: lipgloss.NewStyle().
			Background(lipgloss.Color(t.CursorBg)).
			Foreground(lipgloss.Color(t.CursorFg)),

		CurrentLine: lipgloss.NewStyle().
			Background(lipgloss.Color(t.CurrentLineBg)).
			Foreground(lipgloss.Color(t.CurrentLineFg)),

		LineNumber: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.LineNumber)),

		// Markdown preview styles
		MdText: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.MdText)),

		MdH1: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(t.MdText)).
			Background(lipgloss.Color(t.MdH1Bg)).
			Padding(0, 1),

		MdH2: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(t.MdText)).
			Background(lipgloss.Color(t.MdH2Bg)).
			Padding(0, 1),

		MdH3Plus: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(t.MdHeading)),

		MdLink: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.MdLink)).
			Underline(true),

		MdQuote: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.MdQuote)).
			Italic(true),

		MdCode: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.MdCode)),

		MdCodeBg: lipgloss.NewStyle().
			Foreground(lipgloss.Color(t.MdCode)).
			Background(lipgloss.Color(t.MdCodeBg)).
			Padding(0, 1),
	}
}
