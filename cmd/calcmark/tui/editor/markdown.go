package editor

import (
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
)

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
		// Skip leading empty line (glamour adds this for lists/blockquotes)
		if i == 0 && cleaned == "" && len(lines) > 1 {
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

// createMinimalStyle creates a glamour style with zero margins/padding.
// This ensures markdown renders without extra blank lines.
func createMinimalStyle() ansi.StyleConfig {
	zero := uint(0)

	return ansi.StyleConfig{
		Document: ansi.StyleBlock{
			Margin: &zero,
		},
		Paragraph: ansi.StyleBlock{
			Margin: &zero,
		},
		Heading: ansi.StyleBlock{
			Margin: &zero,
		},
		H1: ansi.StyleBlock{
			Margin: &zero,
			StylePrimitive: ansi.StylePrimitive{
				Bold:  boolPtr(true),
				Color: stringPtr("15"), // Bright white
			},
		},
		H2: ansi.StyleBlock{
			Margin: &zero,
			StylePrimitive: ansi.StylePrimitive{
				Bold:  boolPtr(true),
				Color: stringPtr("15"),
			},
		},
		H3: ansi.StyleBlock{
			Margin: &zero,
			StylePrimitive: ansi.StylePrimitive{
				Bold:  boolPtr(true),
				Color: stringPtr("252"),
			},
		},
		H4: ansi.StyleBlock{
			Margin: &zero,
			StylePrimitive: ansi.StylePrimitive{
				Color: stringPtr("252"),
			},
		},
		H5: ansi.StyleBlock{
			Margin: &zero,
			StylePrimitive: ansi.StylePrimitive{
				Color: stringPtr("252"),
			},
		},
		H6: ansi.StyleBlock{
			Margin: &zero,
			StylePrimitive: ansi.StylePrimitive{
				Color: stringPtr("240"),
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
			Prefix: "• ",
		},
		Enumeration: ansi.StylePrimitive{
			BlockPrefix: ". ",
		},
		BlockQuote: ansi.StyleBlock{
			Margin: &zero,
			Indent: &zero,
			StylePrimitive: ansi.StylePrimitive{
				Prefix: "│ ",
				Color:  stringPtr("244"),
				Italic: boolPtr(true),
			},
		},
		HorizontalRule: ansi.StylePrimitive{
			Format: "────────",
			Color:  stringPtr("240"),
		},
		Code: ansi.StyleBlock{
			Margin: &zero,
			StylePrimitive: ansi.StylePrimitive{
				Prefix:          "`",
				Suffix:          "`",
				Color:           stringPtr("203"), // Coral/orange for code
				BackgroundColor: stringPtr("236"),
			},
		},
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{
				Margin: &zero,
			},
		},
		Emph: ansi.StylePrimitive{
			Italic: boolPtr(true),
		},
		Strong: ansi.StylePrimitive{
			Bold: boolPtr(true),
		},
		Strikethrough: ansi.StylePrimitive{
			CrossedOut: boolPtr(true),
		},
		Link: ansi.StylePrimitive{
			Color:     stringPtr("39"), // Blue
			Underline: boolPtr(true),
		},
		LinkText: ansi.StylePrimitive{
			Color:     stringPtr("39"),
			Underline: boolPtr(true),
		},
		Image: ansi.StylePrimitive{
			Color:  stringPtr("245"),
			Prefix: "🖼 ",
		},
		ImageText: ansi.StylePrimitive{
			Color: stringPtr("245"),
		},
		Table: ansi.StyleTable{
			StyleBlock: ansi.StyleBlock{
				Margin: &zero,
			},
		},
		Text: ansi.StylePrimitive{},
	}
}

func boolPtr(v bool) *bool {
	return &v
}

func stringPtr(v string) *string {
	return &v
}
