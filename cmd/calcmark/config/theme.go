package config

import (
	"github.com/CalcMark/go-calcmark/cmd/calcmark/config/theme"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
)

// Styles holds pre-built lipgloss styles derived from the semantic palette.
// This avoids rebuilding styles on every render call.
// All colors use lipgloss.AdaptiveColor from the theme package, so they
// automatically resolve to the correct light/dark value at Render() time.
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
	CurrentLine lipgloss.Style // Current line highlight
	LineNumber  lipgloss.Style // Line number style
	SourceText  lipgloss.Style // Normal source text color

	// Calculation result display styles
	CalcVarName lipgloss.Style // Variable name in result
	CalcArrow   lipgloss.Style // Arrow in result
	CalcValue   lipgloss.Style // Calculated value in result

	// Markdown preview styles
	MdText   lipgloss.Style // Body text
	MdH1     lipgloss.Style // H1 heading
	MdH2     lipgloss.Style // H2 heading
	MdH3Plus lipgloss.Style // H3+ headings
	MdLink   lipgloss.Style // Links
	MdQuote  lipgloss.Style // Block quotes
	MdCode   lipgloss.Style // Inline code
	MdCodeBg lipgloss.Style // Code with background

	// Pane backgrounds
	SourcePane    lipgloss.Style // Source pane background
	PreviewPane   lipgloss.Style // Preview pane background
	StatusBar     lipgloss.Style // Status bar background
	ContextFooter lipgloss.Style // Context footer background

	// Input and prompt styles
	PromptLabel lipgloss.Style // Prompt label style (e.g., "Save as:")
	InputText   lipgloss.Style // User input text style
	InputCursor lipgloss.Style // Input cursor style

	// Source pane syntax highlighting (block-level)
	SourceFrontmatter lipgloss.Style // Frontmatter lines (YAML between ---)
	SourceMarkdown    lipgloss.Style // Markdown prose lines
	SourceCalc        lipgloss.Style // Calculation lines
}

// BuildStyles creates lipgloss.Style instances from the semantic palette.
// User overrides from ThemeConfig are applied by overriding the appropriate
// palette slot before building. Since AdaptiveColor resolves at Render() time,
// this function only needs to be called once.
func (t ThemeConfig) BuildStyles() Styles {
	// Apply user overrides to palette. Each override replaces the palette slot
	// matching the current color_mode (dark or light). Since we can't know the
	// mode here, we set both slots when the user provides an override — the user
	// has explicitly chosen this color for their mode.
	primary := overrideColor(theme.Primary, t.Primary)
	accent := overrideColor(theme.Accent, t.Accent)
	errColor := overrideColor(theme.Error, t.Error)
	warning := overrideColor(theme.Warning, t.Warning)
	muted := overrideColor(theme.TextMuted, t.Muted)
	dimmed := overrideColor(theme.Hint, t.Dimmed)
	output := overrideColor(theme.Result, t.Output)
	bright := overrideColor(theme.TextBright, t.Bright)
	separator := overrideColor(theme.Separator, t.Separator)
	sourcePaneBg := overrideColor(theme.SourcePaneBg, t.SourcePaneBg)
	previewPaneBg := overrideColor(theme.PreviewPaneBg, t.PreviewPaneBg)
	statusBarBg := overrideColor(theme.StatusBg, t.StatusBarBg)

	return Styles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(primary).
			Margin(1, 0),

		PinnedPanel: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(accent).
			PaddingLeft(1),

		Error: lipgloss.NewStyle().
			Foreground(errColor),

		ErrorOutput: lipgloss.NewStyle().
			Foreground(errColor),

		Help: lipgloss.NewStyle().
			Foreground(muted).
			Margin(1, 0),

		Prompt: lipgloss.NewStyle().
			Foreground(primary),

		Output: lipgloss.NewStyle().
			Foreground(output),

		Changed: lipgloss.NewStyle().
			Foreground(warning),

		Var: lipgloss.NewStyle().
			Foreground(primary),

		Hint: lipgloss.NewStyle().
			Foreground(dimmed).
			Italic(true),

		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(primary),

		Syntax: lipgloss.NewStyle().
			Bold(true).
			Foreground(bright),

		Example: lipgloss.NewStyle().
			Italic(true).
			Foreground(dimmed),

		Separator: lipgloss.NewStyle().
			Foreground(separator),

		ModeIndicator: lipgloss.NewStyle().
			Foreground(primary),

		// Editor styles — sourced from palette
		EditLine: lipgloss.NewStyle().
			Background(theme.EditLineBg).
			Foreground(theme.EditLineFg),

		Cursor: lipgloss.NewStyle().
			Background(theme.Cursor).
			Foreground(theme.CursorFg),

		CurrentLine: lipgloss.NewStyle().
			Background(theme.CurrentLineBg).
			Foreground(theme.CurrentLineFg),

		LineNumber: lipgloss.NewStyle().
			Foreground(theme.LineNumber).
			Background(sourcePaneBg),

		SourceText: lipgloss.NewStyle().
			Foreground(theme.Text).
			Background(sourcePaneBg),

		// Calculation result display styles
		CalcVarName: lipgloss.NewStyle().
			Foreground(theme.ResultMuted).
			Background(previewPaneBg),

		CalcArrow: lipgloss.NewStyle().
			Foreground(theme.ResultArrow).
			Background(previewPaneBg),

		CalcValue: lipgloss.NewStyle().
			Foreground(output).
			Background(previewPaneBg),

		// Markdown preview styles
		MdText: lipgloss.NewStyle().
			Foreground(theme.Text),

		MdH1: lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Text).
			Background(theme.MdH1Bg).
			Padding(0, 1),

		MdH2: lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Text).
			Background(theme.MdH2Bg).
			Padding(0, 1),

		MdH3Plus: lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.MdHeading),

		MdLink: lipgloss.NewStyle().
			Foreground(theme.MdLink).
			Underline(true),

		MdQuote: lipgloss.NewStyle().
			Foreground(theme.MdQuote).
			Italic(true),

		MdCode: lipgloss.NewStyle().
			Foreground(theme.MdCode),

		MdCodeBg: lipgloss.NewStyle().
			Foreground(theme.MdCode).
			Background(theme.MdCodeBg).
			Padding(0, 1),

		// Pane backgrounds
		SourcePane: lipgloss.NewStyle().
			Background(sourcePaneBg),
		PreviewPane: lipgloss.NewStyle().
			Background(previewPaneBg),
		StatusBar: lipgloss.NewStyle().
			Background(statusBarBg),
		ContextFooter: lipgloss.NewStyle().
			Background(theme.ContextFooterBg),

		// Input and prompt styles
		PromptLabel: lipgloss.NewStyle().
			Foreground(theme.PromptFg).
			Background(theme.PromptBg).
			Bold(true).
			Padding(0, 1),

		InputText: lipgloss.NewStyle().
			Foreground(theme.InputFg).
			Background(theme.InputBg),

		InputCursor: lipgloss.NewStyle().
			Foreground(theme.Cursor).
			Bold(true),

		// Source pane syntax highlighting (block-level)
		SourceFrontmatter: lipgloss.NewStyle().
			Foreground(theme.SourceFrontmatter).
			Background(sourcePaneBg),
		SourceMarkdown: lipgloss.NewStyle().
			Foreground(theme.SourceMarkdown).
			Background(sourcePaneBg),
		SourceCalc: lipgloss.NewStyle().
			Foreground(theme.SourceCalc).
			Background(sourcePaneBg),
	}
}

// overrideColor returns an AdaptiveColor with the user's hex override applied
// to both slots if non-empty, otherwise returns the palette default.
func overrideColor(palette compat.AdaptiveColor, userHex string) compat.AdaptiveColor {
	if userHex == "" {
		return palette
	}
	// User override applies to the slot matching their color_mode.
	// Since we build once and AdaptiveColor resolves at render time,
	// we set both slots — the user explicitly chose this color.
	c := lipgloss.Color(userHex)
	return compat.AdaptiveColor{
		Light: c,
		Dark:  c,
	}
}
