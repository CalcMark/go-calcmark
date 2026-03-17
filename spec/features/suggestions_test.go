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
