package editor

import "testing"

func TestSuggestionTag(t *testing.T) {
	tests := []struct {
		category string
		want     string
	}{
		// Explicit function categories
		{"Math", "fn"},
		{"Conversion", "fn"},
		{"Network", "fn"},
		{"Storage", "fn"},
		{"Capacity", "fn"},
		{"Growth", "fn"},

		// Special categories
		{"example", "nl"},
		{"variable", "var"},

		// Short categories pass through
		{"kg", "kg"},
		{"m", "m"},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			got := suggestionTag(tt.category)
			if got != tt.want {
				t.Errorf("suggestionTag(%q) = %q, want %q", tt.category, got, tt.want)
			}
		})
	}
}
