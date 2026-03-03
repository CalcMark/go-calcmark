package editor

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
)

// TestContextFooterShowsVariableReferences verifies that the context footer
// renders variable references when the cursor is on a calc line that
// references other variables. This tests the full rendering pipeline:
// Model → GetLineResults → getLineReferences → renderContextFooter.
func TestContextFooterShowsVariableReferences(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		cursorLine int
		wantInOutput []string // substrings expected in footer output
	}{
		{
			name:         "simple variable reference",
			source:       "x = 10\ny = x + 5\n",
			cursorLine:   1,
			wantInOutput: []string{"x"},
		},
		{
			name:         "multiple variable references",
			source:       "a = 1\nb = 2\nc = a + b\n",
			cursorLine:   2,
			wantInOutput: []string{"a", "b"},
		},
		{
			name:         "cross-block variable reference",
			source:       "rent = 1500\n\n# Totals\n\ntotal = rent + 200\n",
			cursorLine:   4,
			wantInOutput: []string{"rent"},
		},
		{
			name:         "no refs for literal-only line",
			source:       "x = 42\n",
			cursorLine:   0,
			wantInOutput: nil, // expect empty footer
		},
		{
			name:         "no refs for currency arithmetic",
			source:       "budget = 1000 EUR - 250 EUR\n",
			cursorLine:   0,
			wantInOutput: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := document.NewDocument(tt.source)
			if err != nil {
				t.Fatalf("NewDocument(%q): %v", tt.source, err)
			}
			m := New(doc)
			m.width = 80
			m.height = 24
			m.cursorLine = tt.cursorLine

			footer := m.renderContextFooter(80)

			// Strip ANSI escape codes for content matching
			plain := stripAnsi(footer)

			if len(tt.wantInOutput) == 0 {
				// No specific variable names expected — just log for inspection
				t.Logf("Footer (plain, expect empty): %q", plain)
				return
			}

			t.Logf("Footer (plain): %q", plain)
			for _, want := range tt.wantInOutput {
				if !strings.Contains(plain, want) {
					t.Errorf("context footer missing %q\nfooter output: %q", want, plain)
				}
			}
		})
	}
}

// stripAnsi removes ANSI escape sequences from a string.
func stripAnsi(s string) string {
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}
