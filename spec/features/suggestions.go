package features

import "strings"

// Suggestion represents an autocompletion suggestion.
// This type is shared between the TUI editor and the LSP server.
type Suggestion struct {
	Name         string // Display name (may include synonyms for display)
	Category     string // Category (function, unit, variable, etc.)
	Description  string // Brief description
	Syntax       string // Syntax example
	InsertText   string // Actual text to insert (without synonyms/formatting)
	SortCategory string // Override category for sorting (e.g., NL rows use parent fn category)
}

// SuggestionSource provides suggestions for a given prefix.
type SuggestionSource interface {
	// GetSuggestions returns suggestions matching the given prefix.
	GetSuggestions(prefix string) []Suggestion
}

// MatchesPrefix checks if s starts with lowerPrefix (case-insensitive).
// lowerPrefix must already be lowercased by the caller.
func MatchesPrefix(s string, lowerPrefix string) bool {
	return strings.HasPrefix(strings.ToLower(s), lowerPrefix)
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
		if MatchesPrefix(s.Name, prefix) {
			matches = append(matches, s)
		}
	}

	return matches
}
