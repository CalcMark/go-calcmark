package features

import "testing"

func TestFilterSuggestions(t *testing.T) {
	suggestions := []Suggestion{
		{Name: "alpha", Category: "variable"},
		{Name: "beta", Category: "variable"},
		{Name: "average", Category: "function"},
	}

	t.Run("empty prefix returns all", func(t *testing.T) {
		got := FilterSuggestions(suggestions, "")
		if len(got) != 3 {
			t.Errorf("expected 3 suggestions, got %d", len(got))
		}
	})

	t.Run("prefix filters by name", func(t *testing.T) {
		got := FilterSuggestions(suggestions, "a")
		if len(got) != 2 {
			t.Errorf("expected 2 suggestions for 'a', got %d", len(got))
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		got := FilterSuggestions(suggestions, "A")
		if len(got) != 2 {
			t.Errorf("expected 2 suggestions for 'A', got %d", len(got))
		}
	})

	t.Run("no match returns empty", func(t *testing.T) {
		got := FilterSuggestions(suggestions, "z")
		if len(got) != 0 {
			t.Errorf("expected 0 suggestions for 'z', got %d", len(got))
		}
	})
}

func TestMatchesPrefix(t *testing.T) {
	tests := []struct {
		s, prefix string
		want      bool
	}{
		{"Average", "av", true},    // case-insensitive
		{"average", "av", true},    // exact case
		{"avg", "av", true},        // shorter name
		{"beta", "av", false},      // no match
		{"average", "", true},      // empty prefix matches all
		{"", "av", false},          // empty string never matches
		{"@globals", "@g", true},   // directive prefix
		{"Meter", "meter", true},   // unit name case fold
	}
	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.prefix, func(t *testing.T) {
			if got := MatchesPrefix(tt.s, tt.prefix); got != tt.want {
				t.Errorf("MatchesPrefix(%q, %q) = %v, want %v", tt.s, tt.prefix, got, tt.want)
			}
		})
	}
}
