package components

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/config/theme"
	"github.com/charmbracelet/lipgloss"
)

// Suggestion represents an autocompletion suggestion.
type Suggestion struct {
	Name        string // Display name (may include synonyms for display)
	Category    string // Category (function, unit, variable, etc.)
	Description string // Brief description
	Syntax      string // Syntax example
	InsertText  string // Actual text to insert (without synonyms/formatting)
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

// PopupStyle holds styles for rendering the popup overlay.
type PopupStyle struct {
	Border      lipgloss.Style // Border style for popup box
	Item        lipgloss.Style // Normal suggestion item
	Selected    lipgloss.Style // Selected/highlighted item
	Description lipgloss.Style // Description text
	Hint        lipgloss.Style // Keyboard hint at bottom
}

// DefaultPopupStyle returns the default popup styling.
func DefaultPopupStyle() PopupStyle {
	return PopupStyle{
		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.PopupBorder).
			Background(theme.PopupBg),
		Item: lipgloss.NewStyle().
			Foreground(theme.Text).
			Background(theme.PopupBg).
			Padding(0, 1),
		Selected: lipgloss.NewStyle().
			Foreground(theme.PopupSelectedFg).
			Background(theme.PopupSelectedBg).
			Bold(true).
			Padding(0, 1),
		Description: lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Italic(true),
		Hint: lipgloss.NewStyle().
			Foreground(theme.Hint).
			Italic(true).
			Background(theme.PopupBg).
			Padding(0, 1),
	}
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

// RenderPopup renders suggestions as a bordered popup box.
// Returns a slice of lines that should be overlaid at the popup position.
// Pure function: takes state and style, returns lines.
func RenderPopup(state AutosuggestState, style PopupStyle) []string {
	if !state.Visible || len(state.Suggestions) == 0 {
		return nil
	}

	maxVisible := state.PopupHeight
	if maxVisible <= 0 {
		maxVisible = min(len(state.Suggestions), 8)
	}

	// Ensure scroll keeps selected item visible
	scrollTop := min(state.ScrollTop, state.Selected)
	if state.Selected >= scrollTop+maxVisible {
		scrollTop = state.Selected - maxVisible + 1
	}

	// Calculate content width (for padding suggestions)
	contentWidth := max(15, state.PopupWidth-4) // Account for border and padding

	var lines []string

	// Render visible suggestions
	for i := scrollTop; i < scrollTop+maxVisible && i < len(state.Suggestions); i++ {
		s := state.Suggestions[i]

		// Format: prefer syntax (function signature) over description
		// Syntax is like "avg(a, b, ...)" which is more useful
		name := s.Name
		detail := ""
		if s.Syntax != "" {
			detail = " " + s.Syntax
		} else if s.Description != "" {
			detail = " " + s.Description
		}

		// Truncate if needed
		line := name + detail
		if len(line) > contentWidth {
			line = line[:contentWidth-1] + "…"
		}

		// Pad to content width
		for len(line) < contentWidth {
			line += " "
		}

		// Apply style based on selection
		if i == state.Selected {
			lines = append(lines, style.Selected.Render(line))
		} else {
			lines = append(lines, style.Item.Render(line))
		}
	}

	// Add scroll indicators if needed
	if len(state.Suggestions) > maxVisible {
		indicator := fmt.Sprintf("(%d/%d)", state.Selected+1, len(state.Suggestions))
		for len(indicator) < contentWidth {
			indicator += " "
		}
		lines = append(lines, style.Hint.Render(indicator))
	}

	// Add hint line
	hint := "Tab:accept ↑↓:select Esc:cancel"
	if len(hint) > contentWidth {
		hint = "Tab ↑↓ Esc"
	}
	for len(hint) < contentWidth {
		hint += " "
	}
	lines = append(lines, style.Hint.Render(hint))

	return lines
}

// RenderPopupBox renders the popup with a border around it.
// Returns the complete popup as a single string for overlay.
func RenderPopupBox(state AutosuggestState, style PopupStyle) string {
	lines := RenderPopup(state, style)
	if len(lines) == 0 {
		return ""
	}

	content := strings.Join(lines, "\n")
	return style.Border.Render(content)
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
