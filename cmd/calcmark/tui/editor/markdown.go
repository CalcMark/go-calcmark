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
	renderer     *glamour.TermRenderer
	width        int
	hiddenLinkRe *regexp.Regexp // strips invisible URL text from links
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

	// Build a regex to strip invisible link URLs from glamour output.
	// The Link style sets fg=bg=pvBg, so the URL text is invisible but takes space.
	// We match the space + ANSI-styled invisible text and remove it entirely.
	hiddenLinkRe := buildHiddenLinkRegex()

	return &MarkdownRenderer{
		renderer:     renderer,
		width:        width,
		hiddenLinkRe: hiddenLinkRe,
	}, nil
}

// buildHiddenLinkRegex creates a regex that matches the invisible URL portion
// of rendered links. Glamour outputs: [LinkText styled] [reset] [space] [Link styled URL] [reset]
// The Link style has fg=bg=pvBg, producing a specific ANSI SGR sequence.
// We match that sequence, any text until reset, and the preceding space.
func buildHiddenLinkRegex() *regexp.Regexp {
	pvBg := resolveThemeColor(theme.PreviewPaneBg)
	// Convert hex #RRGGBB to decimal R;G;B for ANSI matching
	r, g, b := hexToRGB(pvBg)
	// The ANSI sequence for fg=bg=pvBg is: \x1b[38;2;R;G;B;48;2;R;G;Bm
	// Match: optional space + this SGR + any non-ESC text + reset
	pattern := fmt.Sprintf(` ?\x1b\[38;2;%d;%d;%d;48;2;%d;%d;%dm[^\x1b]*\x1b\[0m`, r, g, b, r, g, b)
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil // graceful degradation: URLs will be invisible but take space
	}
	return re
}

// hexToRGB converts a hex color string (#RRGGBB) to decimal R, G, B values.
func hexToRGB(hex string) (int, int, int) {
	var r, g, b int
	fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	return r, g, b
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

	// Strip invisible link URLs (fg=bg styled text) before line splitting.
	if m.hiddenLinkRe != nil {
		result = m.hiddenLinkRe.ReplaceAllString(result, "")
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

// RenderBlock renders a multi-line markdown block, preserving blank lines.
// Glamour normalizes inter-block spacing (collapsing blank lines between headings
// and paragraphs). To preserve the user's intentional blank lines, we split the
// input at blank lines, render each segment independently, then re-join.
func (m *MarkdownRenderer) RenderBlock(content string) []string {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return []string{""}
	}

	// Split into segments separated by blank lines.
	var segments [][]string
	var current []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			// End current segment (if any), add a blank line marker.
			if len(current) > 0 {
				segments = append(segments, current)
				current = nil
			}
			segments = append(segments, nil) // nil = blank line
		} else {
			current = append(current, line)
		}
	}
	if len(current) > 0 {
		segments = append(segments, current)
	}

	// Render each segment through glamour, preserving blank lines.
	var output []string
	for _, seg := range segments {
		if seg == nil {
			// Blank line — preserved as-is.
			output = append(output, "")
			continue
		}
		rendered := m.RenderLine(strings.Join(seg, "\n"))
		output = append(output, rendered...)
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
// the cm Pearish theme colors. Maps glamour's color system to cm theme colors so
// rendered markdown is visible and attractive against the preview pane background.
//
// Theme color usage:
//   - H1: Primary (pear green, bold) — most prominent heading
//   - H2: Accent (lime green, bold) — secondary heading
//   - H3: TextBright (white, bold) — tertiary heading
//   - H4-H6: Text/TextMuted — diminishing emphasis
//   - Links: MdLink (blue, underlined) — only link text shown, URL hidden
//   - Code: MdCode (warm red) on MdCodeBg — inline code stands out
//   - Tables: Separator-themed borders
func createMinimalStyle() ansi.StyleConfig {
	zero := uint(0)

	// Resolve cm theme adaptive colors to hex strings for glamour.
	textColor := resolveThemeColor(theme.Text)
	textBright := resolveThemeColor(theme.TextBright)
	textMuted := resolveThemeColor(theme.TextMuted)
	pvBg := resolveThemeColor(theme.PreviewPaneBg)

	// Heading colors — pear green hierarchy
	h1Color := resolveThemeColor(theme.Primary)
	h2Color := resolveThemeColor(theme.Accent)

	// Semantic element colors
	linkColor := resolveThemeColor(theme.MdLink)
	codeColor := resolveThemeColor(theme.MdCode)
	codeBg := resolveThemeColor(theme.MdCodeBg)
	quoteColor := resolveThemeColor(theme.MdQuote)
	separatorColor := resolveThemeColor(theme.Separator)

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
				Color:           &h1Color,
				BackgroundColor: &pvBg,
			},
		},
		H2: ansi.StyleBlock{
			Margin: &zero,
			StylePrimitive: ansi.StylePrimitive{
				Bold:            boolPtr(true),
				Color:           &h2Color,
				BackgroundColor: &pvBg,
			},
		},
		H3: ansi.StyleBlock{
			Margin: &zero,
			StylePrimitive: ansi.StylePrimitive{
				Bold:            boolPtr(true),
				Color:           &textBright,
				BackgroundColor: &pvBg,
			},
		},
		H4: ansi.StyleBlock{
			Margin: &zero,
			StylePrimitive: ansi.StylePrimitive{
				Bold:            boolPtr(true),
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
				Color:           &quoteColor,
				BackgroundColor: &pvBg,
				Italic:          boolPtr(true),
			},
		},
		HorizontalRule: ansi.StylePrimitive{
			Format:          "────────",
			Color:           &separatorColor,
			BackgroundColor: &pvBg,
		},
		Code: ansi.StyleBlock{
			Margin: &zero,
			StylePrimitive: ansi.StylePrimitive{
				Prefix:          "`",
				Suffix:          "`",
				Color:           &codeColor,
				BackgroundColor: &codeBg,
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
		// Link: styles the URL portion that glamour renders after the link text.
		// We hide it by setting foreground = background so it's invisible.
		// The space prefix from glamour's renderHrefPart is unavoidable but minimal.
		Link: ansi.StylePrimitive{
			Color:           &pvBg,
			BackgroundColor: &pvBg,
		},
		// LinkText: styles the visible "[text]" portion of markdown links.
		LinkText: ansi.StylePrimitive{
			Color:           &linkColor,
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
				StylePrimitive: ansi.StylePrimitive{
					Color:           &textColor,
					BackgroundColor: &pvBg,
				},
			},
			CenterSeparator: stringPtr("┼"),
			ColumnSeparator: stringPtr("│"),
			RowSeparator:    stringPtr("─"),
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
