package components

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/config/theme"
	"github.com/CalcMark/go-calcmark/spec/features"
)

// Suggestion is an alias for the shared suggestion type in spec/features.
type Suggestion = features.Suggestion

// SuggestionSource is an alias for the shared interface in spec/features.
type SuggestionSource = features.SuggestionSource

// AutosuggestState holds the state for the autosuggestion component.
type AutosuggestState struct {
	Suggestions []Suggestion
	Selected    int    // Currently selected index (-1 for none)
	Visible     bool   // Whether the suggestions are visible
	Prefix      string // The prefix being completed

	// Popup positioning (computed when triggered)
	PopupRow    int // Screen row for popup (0-indexed)
	PopupCol    int // Screen column for popup (0-indexed)
	PopupWidth  int // Width of popup
	PopupHeight int // Height of popup (number of visible items)
	ScrollTop   int // First visible suggestion index (for scrolling long lists)
}

// AutosuggestStyle holds styles for rendering suggestions.
type AutosuggestStyle struct {
	Container lipgloss.Style // Container for all suggestions
	Item      lipgloss.Style // Normal suggestion item
	Selected  lipgloss.Style // Selected suggestion item
	Category  lipgloss.Style // Category label
	Syntax    lipgloss.Style // Syntax/example text
	Separator lipgloss.Style // Separator between items
}

// DefaultAutosuggestStyle returns the default suggestion styling.
func DefaultAutosuggestStyle() AutosuggestStyle {
	return AutosuggestStyle{
		Container: lipgloss.NewStyle().
			Foreground(theme.TextMuted),
		Item: lipgloss.NewStyle().
			Foreground(theme.AutosuggestText),
		Selected: lipgloss.NewStyle().
			Foreground(theme.TextBright).
			Background(theme.AutosuggestSelectedBg),
		Category: lipgloss.NewStyle().
			Foreground(theme.Hint).
			Italic(true),
		Syntax: lipgloss.NewStyle().
			Foreground(theme.AutosuggestSyntax),
		Separator: lipgloss.NewStyle().
			Foreground(theme.AutosuggestSeparator),
	}
}

// RenderSuggestions renders suggestions as a single-line hint.
// Pure function: takes state and width, returns string.
func RenderSuggestions(state AutosuggestState, width int, style AutosuggestStyle) string {
	if !state.Visible || len(state.Suggestions) == 0 {
		return ""
	}

	var parts []string
	for i, s := range state.Suggestions {
		var part string
		if i == state.Selected {
			part = style.Selected.Render(s.Name)
		} else {
			part = style.Item.Render(s.Name)
		}

		if s.Syntax != "" {
			part += " " + style.Syntax.Render(s.Syntax)
		}

		parts = append(parts, part)

		// Limit to fit width
		current := strings.Join(parts, style.Separator.Render(" │ "))
		if lipgloss.Width(current) > width-10 {
			// Remove last part if too wide
			parts = parts[:len(parts)-1]
			break
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return style.Container.Render("Hints: " + strings.Join(parts, style.Separator.Render(" │ ")))
}

// RenderDropdownSuggestions renders suggestions as a dropdown menu.
// Use this for more detailed suggestion display.
func RenderDropdownSuggestions(state AutosuggestState, maxItems int, style AutosuggestStyle) string {
	if !state.Visible || len(state.Suggestions) == 0 {
		return ""
	}

	var b strings.Builder
	shown := min(len(state.Suggestions), maxItems)

	for i := range shown {
		s := state.Suggestions[i]

		var line string
		if i == state.Selected {
			line = style.Selected.Render(fmt.Sprintf("→ %-12s %s", s.Name, s.Description))
		} else {
			name := style.Syntax.Render(fmt.Sprintf("  %-12s", s.Name))
			desc := style.Item.Render(s.Description)
			line = name + " " + desc
		}

		b.WriteString(line)
		if i < shown-1 {
			b.WriteString("\n")
		}
	}

	if len(state.Suggestions) > maxItems {
		b.WriteString(fmt.Sprintf("\n%s", style.Category.Render(
			fmt.Sprintf("... and %d more", len(state.Suggestions)-maxItems),
		)))
	}

	return b.String()
}

// FilterSuggestions delegates to the shared implementation in spec/features.
var FilterSuggestions = features.FilterSuggestions
