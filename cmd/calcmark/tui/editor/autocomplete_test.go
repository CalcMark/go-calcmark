package editor

import (
	"testing"
)

func TestFunctionSuggestionSource_GetSuggestions(t *testing.T) {
	source := NewFunctionSuggestionSource()

	tests := []struct {
		name          string
		prefix        string
		wantMatch     string // Expected InsertText in results
		wantMinCount  int    // Minimum number of results
		wantNotMatch  string // Should NOT be in results
	}{
		{
			name:         "av prefix matches avg",
			prefix:       "av",
			wantMatch:    "avg",
			wantMinCount: 1,
		},
		{
			name:         "mean prefix matches avg via synonym",
			prefix:       "mean",
			wantMatch:    "avg", // synonym should still insert primary name
			wantMinCount: 1,
		},
		{
			name:         "average prefix matches avg via synonym",
			prefix:       "average",
			wantMatch:    "avg",
			wantMinCount: 1,
		},
		{
			name:         "sq prefix matches sqrt",
			prefix:       "sq",
			wantMatch:    "sqrt",
			wantMinCount: 1,
		},
		{
			name:         "case insensitive matching",
			prefix:       "AV",
			wantMatch:    "avg",
			wantMinCount: 1,
		},
		{
			name:         "empty prefix returns all functions",
			prefix:       "",
			wantMinCount: 12, // All 12 builtin functions
		},
		{
			name:          "xyz prefix matches nothing",
			prefix:        "xyz",
			wantMinCount:  0,
			wantNotMatch:  "avg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := source.GetSuggestions(tt.prefix)

			if len(suggestions) < tt.wantMinCount {
				t.Errorf("got %d suggestions, want at least %d", len(suggestions), tt.wantMinCount)
			}

			if tt.wantMatch != "" {
				found := false
				for _, s := range suggestions {
					if s.InsertText == tt.wantMatch {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected to find InsertText=%q in suggestions", tt.wantMatch)
				}
			}

			if tt.wantNotMatch != "" {
				for _, s := range suggestions {
					if s.InsertText == tt.wantNotMatch {
						t.Errorf("did not expect to find InsertText=%q in suggestions", tt.wantNotMatch)
					}
				}
			}
		})
	}
}

func TestUnitSuggestionSource_GetSuggestions(t *testing.T) {
	source := NewUnitSuggestionSource()

	tests := []struct {
		name         string
		prefix       string
		wantMatch    string // Expected InsertText in results
		wantMinCount int
	}{
		{
			name:         "met prefix matches meter",
			prefix:       "met",
			wantMatch:    "meter",
			wantMinCount: 1,
		},
		{
			name:         "m prefix matches multiple units",
			prefix:       "m",
			wantMinCount: 3, // meter, millimeter, mile, milligram, milliliter, etc.
		},
		{
			name:         "kg prefix matches kilogram",
			prefix:       "kg",
			wantMatch:    "kilogram",
			wantMinCount: 1,
		},
		{
			name:         "case insensitive",
			prefix:       "MET",
			wantMatch:    "meter",
			wantMinCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := source.GetSuggestions(tt.prefix)

			if len(suggestions) < tt.wantMinCount {
				t.Errorf("got %d suggestions, want at least %d", len(suggestions), tt.wantMinCount)
			}

			if tt.wantMatch != "" {
				found := false
				for _, s := range suggestions {
					if s.InsertText == tt.wantMatch {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected to find InsertText=%q in suggestions", tt.wantMatch)
				}
			}
		})
	}
}

func TestVariableSuggestionSource_GetSuggestions(t *testing.T) {
	vars := map[string]string{
		"price":    "100",
		"quantity": "5",
		"total":    "500",
	}

	source := NewVariableSuggestionSource(func() map[string]string {
		return vars
	})

	tests := []struct {
		name         string
		prefix       string
		wantMatch    string
		wantMinCount int
	}{
		{
			name:         "pr prefix matches price",
			prefix:       "pr",
			wantMatch:    "price",
			wantMinCount: 1,
		},
		{
			name:         "empty prefix returns all variables",
			prefix:       "",
			wantMinCount: 3,
		},
		{
			name:         "case insensitive",
			prefix:       "PR",
			wantMatch:    "price",
			wantMinCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := source.GetSuggestions(tt.prefix)

			if len(suggestions) < tt.wantMinCount {
				t.Errorf("got %d suggestions, want at least %d", len(suggestions), tt.wantMinCount)
			}

			if tt.wantMatch != "" {
				found := false
				for _, s := range suggestions {
					if s.InsertText == tt.wantMatch {
						found = true
						// Verify description shows the value
						if s.Description != vars[tt.wantMatch] {
							t.Errorf("expected description=%q, got %q", vars[tt.wantMatch], s.Description)
						}
						break
					}
				}
				if !found {
					t.Errorf("expected to find InsertText=%q in suggestions", tt.wantMatch)
				}
			}
		})
	}
}

func TestCombinedSuggestionSource_GetSuggestions(t *testing.T) {
	funcSource := NewFunctionSuggestionSource()
	unitSource := NewUnitSuggestionSource()
	varSource := NewVariableSuggestionSource(func() map[string]string {
		return map[string]string{"avgPrice": "100"}
	})

	combined := NewCombinedSuggestionSource(funcSource, unitSource, varSource)

	// "av" should match both avg function and avgPrice variable
	suggestions := combined.GetSuggestions("av")

	if len(suggestions) < 2 {
		t.Errorf("expected at least 2 suggestions for 'av', got %d", len(suggestions))
	}

	// Verify ordering: functions should come before variables
	foundFunc := -1
	foundVar := -1
	for i, s := range suggestions {
		if s.InsertText == "avg" {
			foundFunc = i
		}
		if s.InsertText == "avgPrice" {
			foundVar = i
		}
	}

	if foundFunc == -1 {
		t.Error("expected to find avg function")
	}
	if foundVar == -1 {
		t.Error("expected to find avgPrice variable")
	}
	if foundFunc != -1 && foundVar != -1 && foundFunc > foundVar {
		t.Error("expected functions to appear before variables in suggestions")
	}
}

func TestFunctionSuggestionSource_SynonymDisplay(t *testing.T) {
	source := NewFunctionSuggestionSource()

	// When typing "mean", should show avg with synonyms
	suggestions := source.GetSuggestions("mean")

	if len(suggestions) == 0 {
		t.Fatal("expected at least one suggestion for 'mean'")
	}

	s := suggestions[0]
	if s.InsertText != "avg" {
		t.Errorf("expected InsertText=avg, got %q", s.InsertText)
	}

	// Display name should mention the matched synonym
	if s.Name != "avg (mean)" {
		t.Errorf("expected Name to show matched synonym, got %q", s.Name)
	}
}
