package editor

import (
	"strings"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/config/theme"
	"github.com/charmbracelet/lipgloss"
)

// OverlayStyle holds the shared rendering primitives for all modal overlays
// (help, export, command menu, file picker). Using a shared struct eliminates
// duplicated border/background definitions and ensures visual consistency.
type OverlayStyle struct {
	InnerWidth  int
	BorderStyle lipgloss.Style
	ItemBg      lipgloss.TerminalColor
	SelectedBg  lipgloss.TerminalColor
	TopBorder   string
	BottomBorder string
	LeftBorder  string
	RightBorder string
	SepLine     string
	HintStyle   lipgloss.Style
}

// NewOverlayStyle creates an OverlayStyle with the given inner width.
// All colors come from the palette so they adapt to light/dark mode.
func NewOverlayStyle(innerWidth int) OverlayStyle {
	bs := lipgloss.NewStyle().Foreground(theme.OverlayBorder)

	return OverlayStyle{
		InnerWidth:   innerWidth,
		BorderStyle:  bs,
		ItemBg:       theme.OverlayBg,
		SelectedBg:   theme.PopupSelectedBg,
		TopBorder:    bs.Render("╭" + strings.Repeat("─", innerWidth) + "╮"),
		BottomBorder: bs.Render("╰" + strings.Repeat("─", innerWidth) + "╯"),
		LeftBorder:   bs.Render("│"),
		RightBorder:  bs.Render("│"),
		SepLine:      bs.Render(strings.Repeat("─", innerWidth)),
		HintStyle: lipgloss.NewStyle().
			Foreground(theme.Hint).
			Background(theme.OverlayBg).
			Italic(true),
	}
}

// PadLine creates a line padded to InnerWidth with the given foreground/background.
func (o OverlayStyle) PadLine(content string, fg lipgloss.TerminalColor, bg lipgloss.TerminalColor, bold bool) string {
	style := lipgloss.NewStyle().Foreground(fg).Background(bg)
	if bold {
		style = style.Bold(true)
	}
	visualWidth := lipgloss.Width(content)
	if visualWidth < o.InnerWidth {
		content += strings.Repeat(" ", o.InnerWidth-visualWidth)
	}
	return style.Render(content)
}

// WrapRow wraps content with left and right border characters.
func (o OverlayStyle) WrapRow(content string) string {
	return o.LeftBorder + content + o.RightBorder
}

// HintRow renders a hint line wrapped in borders.
func (o OverlayStyle) HintRow(hint string) string {
	visualWidth := lipgloss.Width(hint)
	if visualWidth < o.InnerWidth {
		hint += strings.Repeat(" ", o.InnerWidth-visualWidth)
	}
	return o.WrapRow(o.HintStyle.Render(hint))
}

// SepRow renders a separator line wrapped in borders.
func (o OverlayStyle) SepRow() string {
	return o.WrapRow(o.SepLine)
}
