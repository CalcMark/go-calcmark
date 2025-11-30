package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Suggestion represents an autocompletion suggestion.
type Suggestion struct {
	Name        string // Display name
	Category    string // Category (function, unit, variable, etc.)
	Description string // Brief description
	Syntax      string // Syntax example
}

// SuggestionSource provides suggestions for a given prefix.
type SuggestionSource interface {
	// GetSuggestions returns suggestions matching the given prefix.
	GetSuggestions(prefix string) []Suggestion
}

// AutosuggestState holds the state for the autosuggestion component.
type AutosuggestState struct {
	Suggestions []Suggestion
	Selected    int    // Currently selected index (-1 for none)
	Visible     bool   // Whether the suggestions are visible
	Prefix      string // The prefix being completed
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
			Foreground(lipgloss.Color("#888888")),
		Item: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#AAAAAA")),
		Selected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#444444")),
		Category: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Italic(true),
		Syntax: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4ECDC4")),
		Separator: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555555")),
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

// FilterSuggestions returns suggestions that match the given prefix.
// This is a pure helper function for suggestion sources.
func FilterSuggestions(suggestions []Suggestion, prefix string) []Suggestion {
	if prefix == "" {
		return suggestions
	}

	prefix = strings.ToLower(prefix)
	var matches []Suggestion

	for _, s := range suggestions {
		if strings.HasPrefix(strings.ToLower(s.Name), prefix) {
			matches = append(matches, s)
		}
	}

	return matches
}

// VariableSuggestionSource provides variable name suggestions.
type VariableSuggestionSource struct {
	Variables map[string]string // variable name -> formatted value
}

// GetSuggestions implements SuggestionSource for variables.
func (v *VariableSuggestionSource) GetSuggestions(prefix string) []Suggestion {
	var suggestions []Suggestion
	for name, value := range v.Variables {
		suggestions = append(suggestions, Suggestion{
			Name:        name,
			Category:    "variable",
			Description: value,
		})
	}
	return FilterSuggestions(suggestions, prefix)
}
