package components

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/document"
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
				Column:     3,
				TotalLines: 100,
				CalcCount:  10,
			},
			width:    80,
			wantSubs: []string{"test.cm", "L5:3", "10 calcs"},
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
				Mode:     "EDITING", // Mode is set but should not be displayed
			},
			width:    80,
			wantSubs: []string{"test.cm"}, // Changed: mode should not appear in output
		},
		{
			name: "long status message truncated",
			state: StatusBarState{
				StatusMsg:   "Open failed: unsupported file type (.md) — this is a really long error message that should be truncated by the status bar renderer",
				StatusIsErr: true,
			},
			width:    40,
			wantSubs: []string{"Open failed", "..."},
		},
		{
			name: "short status message not truncated",
			state: StatusBarState{
				StatusMsg: "Saved: test.cm",
			},
			width:    80,
			wantSubs: []string{"Saved: test.cm"},
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

func TestCleanErrorMessage(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "double frontmatter prefix",
			input: `frontmatter: frontmatter: invalid unit category "Weight"`,
			want:  `invalid unit category "Weight"`,
		},
		{
			name:  "triple frontmatter prefix",
			input: `frontmatter: frontmatter: frontmatter: bad`,
			want:  `bad`,
		},
		{
			name:  "single frontmatter prefix",
			input: `frontmatter: invalid YAML`,
			want:  `invalid YAML`,
		},
		{
			name:  "snake_case error code",
			input: `undefined_variable: Undefined variable "x"`,
			want:  `Undefined variable "x"`,
		},
		{
			name:  "no prefix",
			input: "Division by zero",
			want:  "Division by zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanErrorMessage(tt.input)
			if got != tt.want {
				t.Errorf("CleanErrorMessage(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetHintForDiagnostic_FrontmatterValidation(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    string
	}{
		{
			name:    "invalid unit category with valid list",
			message: `frontmatter: invalid unit category "Weight" in scale.unit_categories; valid categories: All, Area, Currency, Custom`,
			want:    "valid categories: All, Area, Currency, Custom",
		},
		{
			name:    "generic frontmatter error",
			message: "frontmatter: unexpected key 'foo'",
			want:    "Check frontmatter YAML syntax",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diag := &document.Diagnostic{
				Code:    "frontmatter_validation",
				Message: tt.message,
			}
			got := GetHintForDiagnostic(diag)
			if got != tt.want {
				t.Errorf("GetHintForDiagnostic() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestGetHintForDiagnostic_PrefersDetailed verifies that when a diagnostic
// carries a Detailed field from the semantic checker, GetHintForDiagnostic
// uses it instead of computing a generic hint from the code.
func TestGetHintForDiagnostic_PrefersDetailed(t *testing.T) {
	// Diagnostic with Detailed set — should use Detailed verbatim
	diag := &document.Diagnostic{
		Code:     "undefined_variable",
		Message:  `Undefined variable "budget"`,
		Detailed: "Defined variables: income, tax_rate, expenses",
	}
	got := GetHintForDiagnostic(diag)
	if got != diag.Detailed {
		t.Errorf("Expected Detailed field %q, got %q", diag.Detailed, got)
	}

	// Same code but without Detailed — should fall back to semantic hint
	diagNoDetailed := &document.Diagnostic{
		Code:    "undefined_variable",
		Message: `Undefined variable "budget"`,
	}
	got2 := GetHintForDiagnostic(diagNoDetailed)
	if got2 == "" {
		t.Error("Expected fallback hint when Detailed is empty, got empty string")
	}
	if got2 == diag.Detailed {
		t.Error("Should not produce the same output as Detailed when Detailed is empty")
	}
}
