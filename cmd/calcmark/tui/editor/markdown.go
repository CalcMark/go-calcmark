package editor

import (
	"fmt"
	"image/color"
	"regexp"
	"strings"

	"charm.land/lipgloss/v2/compat"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/config/theme"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
)

// ansiEscapeRe matches ANSI escape sequences (CSI sequences).
var ansiEscapeRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// MarkdownRenderer provides line-by-line markdown rendering with 1:1 line mapping.
// This is essential for maintaining vertical alignment between source and preview panes.
type MarkdownRenderer struct {
	renderer *glamour.TermRenderer
	width    int
}

// NewMarkdownRenderer creates a renderer with a minimal style that doesn't add extra lines.
func NewMarkdownRenderer(width int) (*MarkdownRenderer, error) {
	style := createMinimalStyle()

	renderer, err := glamour.NewTermRenderer(
		glamour.WithStyles(style),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return nil, err
	}

	return &MarkdownRenderer{
		renderer: renderer,
		width:    width,
	}, nil
}

// RenderLine renders a single line of markdown, returning wrapped lines as a slice.
// Glamour handles wrapping to the configured width.
func (m *MarkdownRenderer) RenderLine(line string) []string {
	if strings.TrimSpace(line) == "" {
		return []string{""}
	}

	result, err := m.renderer.Render(line)
	if err != nil {
		return []string{line}
	}

	// Trim trailing whitespace/newlines and split into lines
	trimmed := strings.TrimRight(result, "\n ")
	lines := strings.Split(trimmed, "\n")

	// Filter and clean lines
	var output []string
	for i, l := range lines {
		cleaned := strings.TrimRight(l, " ")
		// Skip leading empty line (glamour adds this for lists/blockquotes).
		// With styled output, "empty" lines contain ANSI codes for background color,
		// so we strip escape sequences before checking.
		if i == 0 && isVisuallyEmpty(cleaned) && len(lines) > 1 {
			continue
		}
		output = append(output, cleaned)
	}

	// If empty, check for horizontal rule
	if len(output) == 0 || (len(output) == 1 && output[0] == "") {
		trimmedInput := strings.TrimSpace(line)
		if isHorizontalRule(trimmedInput) {
			return []string{"────────"}
		}
		return []string{""}
	}

	return output
}

// isVisuallyEmpty returns true if a string contains only whitespace after
// stripping ANSI escape sequences. Glamour's styled output produces lines that
// look empty but contain background-color escapes.
func isVisuallyEmpty(s string) bool {
	return strings.TrimSpace(ansiEscapeRe.ReplaceAllString(s, "")) == ""
}

// isHorizontalRule checks if a line is a markdown horizontal rule.
func isHorizontalRule(line string) bool {
	if len(line) < 3 {
		return false
	}
	// Must be 3+ of the same character (-, *, _) possibly with spaces
	ruleChar := rune(0)
	count := 0
	for _, c := range line {
		if c == ' ' {
			continue
		}
		if c == '-' || c == '*' || c == '_' {
			if ruleChar == 0 {
				ruleChar = c
			}
			if c == ruleChar {
				count++
			} else {
				return false
			}
		} else {
			return false
		}
	}
	return count >= 3
}

// createMinimalStyle creates a glamour style with zero margins/padding that uses
// the cm theme colors. This bridges glamour's color system to the Pearish theme,
// ensuring rendered markdown is visible against the preview pane background.
func createMinimalStyle() ansi.StyleConfig {
	zero := uint(0)

	// Resolve cm theme adaptive colors to hex strings for glamour.
	textColor := resolveThemeColor(theme.Text)
	textBright := resolveThemeColor(theme.TextBright)
	textMuted := resolveThemeColor(theme.TextMuted)
	primaryColor := resolveThemeColor(theme.Primary)
	pvBg := resolveThemeColor(theme.PreviewPaneBg)

	return ansi.StyleConfig{
		Document: ansi.StyleBlock{
			Margin: &zero,
			StylePrimitive: ansi.StylePrimitive{
				Color:           &textColor,
				BackgroundColor: &pvBg,
			},
		},
		Paragraph: ansi.StyleBlock{
			Margin: &zero,
			StylePrimitive: ansi.StylePrimitive{
				Color:           &textColor,
				BackgroundColor: &pvBg,
			},
		},
		Heading: ansi.StyleBlock{
			Margin: &zero,
		},
		H1: ansi.StyleBlock{
			Margin: &zero,
			StylePrimitive: ansi.StylePrimitive{
				Bold:            boolPtr(true),
				Color:           &textBright,
				BackgroundColor: &pvBg,
			},
		},
		H2: ansi.StyleBlock{
			Margin: &zero,
			StylePrimitive: ansi.StylePrimitive{
				Bold:            boolPtr(true),
				Color:           &textBright,
				BackgroundColor: &pvBg,
			},
		},
		H3: ansi.StyleBlock{
			Margin: &zero,
			StylePrimitive: ansi.StylePrimitive{
				Bold:            boolPtr(true),
				Color:           &textColor,
				BackgroundColor: &pvBg,
			},
		},
		H4: ansi.StyleBlock{
			Margin: &zero,
			StylePrimitive: ansi.StylePrimitive{
				Color:           &textColor,
				BackgroundColor: &pvBg,
			},
		},
		H5: ansi.StyleBlock{
			Margin: &zero,
			StylePrimitive: ansi.StylePrimitive{
				Color:           &textColor,
				BackgroundColor: &pvBg,
			},
		},
		H6: ansi.StyleBlock{
			Margin: &zero,
			StylePrimitive: ansi.StylePrimitive{
				Color:           &textMuted,
				BackgroundColor: &pvBg,
			},
		},
		List: ansi.StyleList{
			StyleBlock: ansi.StyleBlock{
				Margin: &zero,
				Indent: &zero,
			},
			LevelIndent: 2,
		},
		Item: ansi.StylePrimitive{
			Prefix:          "• ",
			Color:           &textColor,
			BackgroundColor: &pvBg,
		},
		Enumeration: ansi.StylePrimitive{
			BlockPrefix:     ". ",
			Color:           &textColor,
			BackgroundColor: &pvBg,
		},
		BlockQuote: ansi.StyleBlock{
			Margin: &zero,
			Indent: &zero,
			StylePrimitive: ansi.StylePrimitive{
				Prefix:          "│ ",
				Color:           &textMuted,
				BackgroundColor: &pvBg,
				Italic:          boolPtr(true),
			},
		},
		HorizontalRule: ansi.StylePrimitive{
			Format:          "────────",
			Color:           &textMuted,
			BackgroundColor: &pvBg,
		},
		Code: ansi.StyleBlock{
			Margin: &zero,
			StylePrimitive: ansi.StylePrimitive{
				Prefix:          "`",
				Suffix:          "`",
				Color:           &primaryColor,
				BackgroundColor: &pvBg,
			},
		},
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{
				Margin: &zero,
			},
		},
		Emph: ansi.StylePrimitive{
			Italic:          boolPtr(true),
			Color:           &textColor,
			BackgroundColor: &pvBg,
		},
		Strong: ansi.StylePrimitive{
			Bold:            boolPtr(true),
			Color:           &textBright,
			BackgroundColor: &pvBg,
		},
		Strikethrough: ansi.StylePrimitive{
			CrossedOut:      boolPtr(true),
			Color:           &textMuted,
			BackgroundColor: &pvBg,
		},
		Link: ansi.StylePrimitive{
			Color:           stringPtr("39"),
			BackgroundColor: &pvBg,
			Underline:       boolPtr(true),
		},
		LinkText: ansi.StylePrimitive{
			Color:           stringPtr("39"),
			BackgroundColor: &pvBg,
			Underline:       boolPtr(true),
		},
		Image: ansi.StylePrimitive{
			Color:           &textMuted,
			BackgroundColor: &pvBg,
			Prefix:          "🖼 ",
		},
		ImageText: ansi.StylePrimitive{
			Color:           &textMuted,
			BackgroundColor: &pvBg,
		},
		Table: ansi.StyleTable{
			StyleBlock: ansi.StyleBlock{
				Margin: &zero,
			},
		},
		Text: ansi.StylePrimitive{
			Color:           &textColor,
			BackgroundColor: &pvBg,
		},
	}
}

// resolveThemeColor converts a cm theme AdaptiveColor to a hex string for glamour.
// Resolves Light/Dark based on the current compat.HasDarkBackground setting.
func resolveThemeColor(ac compat.AdaptiveColor) string {
	var c color.Color
	if compat.HasDarkBackground {
		c = ac.Dark
	} else {
		c = ac.Light
	}
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8)
}

func boolPtr(v bool) *bool {
	return &v
}

func stringPtr(v string) *string {
	return &v
}
