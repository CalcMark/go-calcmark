package components

import (
	"strings"
	"testing"
)

func TestRenderStatusBar(t *testing.T) {
	style := DefaultStatusBarStyle()

	tests := []struct {
		name     string
		state    StatusBarState
		width    int
		wantSubs []string // Substrings that should appear
	}{
		{
			name: "basic with filename",
			state: StatusBarState{
				Filename:   "test.cm",
				Line:       5,
				TotalLines: 100,
				CalcCount:  10,
			},
			width:    80,
			wantSubs: []string{"test.cm", "L5/100", "10 calcs"},
		},
		{
			name: "new file",
			state: StatusBarState{
				Filename:   "",
				Line:       1,
				TotalLines: 1,
			},
			width:    80,
			wantSubs: []string{"[New]"},
		},
		{
			name: "modified file",
			state: StatusBarState{
				Filename: "modified.cm",
				Modified: true,
			},
			width:    80,
			wantSubs: []string{"modified.cm", "[+]"},
		},
		{
			name: "with mode",
			state: StatusBarState{
				Filename: "test.cm",
				Mode:     "EDITING",
			},
			width:    80,
			wantSubs: []string{"EDITING"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderStatusBar(tt.state, tt.width, style)
			for _, sub := range tt.wantSubs {
				if !strings.Contains(result, sub) {
					t.Errorf("Expected %q in output, got: %s", sub, result)
				}
			}
		})
	}
}

func TestRenderMinimalStatusBar(t *testing.T) {
	style := DefaultStatusBarStyle()
	state := StatusBarState{
		Filename: "very-long-filename-that-should-be-truncated.cm",
		Modified: true,
	}

	result := RenderMinimalStatusBar(state, 30, style)

	// Should be truncated and show modified indicator
	if !strings.Contains(result, "[+]") {
		t.Error("Expected modified indicator")
	}
	if !strings.Contains(result, "...") {
		t.Error("Expected truncation")
	}
}

func TestRenderSuggestions(t *testing.T) {
	style := DefaultAutosuggestStyle()

	tests := []struct {
		name     string
		state    AutosuggestState
		width    int
		wantSub  string
		wantNone bool
	}{
		{
			name: "visible with suggestions",
			state: AutosuggestState{
				Suggestions: []Suggestion{
					{Name: "avg", Syntax: "avg(list)"},
					{Name: "sum", Syntax: "sum(list)"},
				},
				Visible: true,
			},
			width:   80,
			wantSub: "avg",
		},
		{
			name: "not visible",
			state: AutosuggestState{
				Suggestions: []Suggestion{{Name: "test"}},
				Visible:     false,
			},
			width:    80,
			wantNone: true,
		},
		{
			name: "empty suggestions",
			state: AutosuggestState{
				Suggestions: []Suggestion{},
				Visible:     true,
			},
			width:    80,
			wantNone: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderSuggestions(tt.state, tt.width, style)
			if tt.wantNone {
				if result != "" {
					t.Errorf("Expected empty result, got: %s", result)
				}
			} else if !strings.Contains(result, tt.wantSub) {
				t.Errorf("Expected %q in output, got: %s", tt.wantSub, result)
			}
		})
	}
}

func TestFilterSuggestions(t *testing.T) {
	suggestions := []Suggestion{
		{Name: "avg"},
		{Name: "absolute"},
		{Name: "sum"},
		{Name: "sqrt"},
	}

	tests := []struct {
		prefix    string
		wantCount int
	}{
		{"", 4},  // Empty returns all
		{"a", 2}, // "avg", "absolute"
		{"av", 1},
		{"s", 2}, // "sum", "sqrt"
		{"x", 0}, // No match
	}

	for _, tt := range tests {
		result := FilterSuggestions(suggestions, tt.prefix)
		if len(result) != tt.wantCount {
			t.Errorf("FilterSuggestions(%q): got %d, want %d", tt.prefix, len(result), tt.wantCount)
		}
	}
}

func TestRenderGlobalsPanel(t *testing.T) {
	style := DefaultGlobalsPanelStyle()

	tests := []struct {
		name     string
		state    GlobalsPanelState
		width    int
		wantSubs []string
	}{
		{
			name: "expanded with globals",
			state: GlobalsPanelState{
				Globals: []GlobalVar{
					{Name: "tax_rate", Value: "0.32"},
					{Name: "USD_EUR", Value: "0.92", IsExchange: true},
				},
				Expanded: true,
			},
			width:    40,
			wantSubs: []string{"Globals", "tax_rate", "0.32", "USD_EUR", "exchange"},
		},
		{
			name: "collapsed",
			state: GlobalsPanelState{
				Globals:  []GlobalVar{{Name: "test"}},
				Expanded: false,
			},
			width:    40,
			wantSubs: []string{"Globals", "1 items"},
		},
		{
			name: "empty expanded",
			state: GlobalsPanelState{
				Globals:  []GlobalVar{},
				Expanded: true,
			},
			width:    40,
			wantSubs: []string{"no globals"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderGlobalsPanel(tt.state, tt.width, style)
			for _, sub := range tt.wantSubs {
				if !strings.Contains(result, sub) {
					t.Errorf("Expected %q in output, got: %s", sub, result)
				}
			}
		})
	}
}

func TestRenderPinnedPanel(t *testing.T) {
	style := DefaultPinnedPanelStyle()

	tests := []struct {
		name     string
		state    PinnedPanelState
		width    int
		wantSubs []string
	}{
		{
			name: "with variables",
			state: PinnedPanelState{
				Variables: []PinnedVar{
					{Name: "x", Value: "10"},
					{Name: "y", Value: "20", Changed: true},
				},
				Height: 10,
			},
			width:    30,
			wantSubs: []string{"Pinned", "x", "10", "y", "20", "*"},
		},
		{
			name: "empty",
			state: PinnedPanelState{
				Variables: []PinnedVar{},
				Height:    10,
			},
			width:    30,
			wantSubs: []string{"no variables pinned"},
		},
		{
			name: "with frontmatter",
			state: PinnedPanelState{
				Variables: []PinnedVar{
					{Name: "rate", Value: "0.1", IsFrontmatter: true},
				},
				Height: 10,
			},
			width:    30,
			wantSubs: []string{"@", "rate"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderPinnedPanel(tt.state, tt.width, style)
			for _, sub := range tt.wantSubs {
				if !strings.Contains(result, sub) {
					t.Errorf("Expected %q in output, got: %s", sub, result)
				}
			}
		})
	}
}

func TestRenderPinnedPanelScrolling(t *testing.T) {
	style := DefaultPinnedPanelStyle()

	// Create more variables than can fit
	vars := make([]PinnedVar, 20)
	for i := range 20 {
		vars[i] = PinnedVar{Name: string(rune('a' + i)), Value: "1"}
	}

	state := PinnedPanelState{
		Variables: vars,
		ScrollY:   0,
		Height:    5,
	}

	result := RenderPinnedPanel(state, 30, style)

	// Should show scroll indicator
	if !strings.Contains(result, "↑↓") {
		t.Error("Expected scroll indicator for overflowing content")
	}

	// Should show first variable (a)
	if !strings.Contains(result, "a") {
		t.Error("Expected first variable 'a' to be visible")
	}
}

func TestVariableSuggestionSource(t *testing.T) {
	source := &VariableSuggestionSource{
		Variables: map[string]string{
			"alpha":   "1",
			"beta":    "2",
			"average": "50",
		},
	}

	// All suggestions
	all := source.GetSuggestions("")
	if len(all) != 3 {
		t.Errorf("Expected 3 suggestions, got %d", len(all))
	}

	// Filtered
	filtered := source.GetSuggestions("a")
	if len(filtered) != 2 { // alpha, average
		t.Errorf("Expected 2 suggestions for 'a', got %d", len(filtered))
	}
}
