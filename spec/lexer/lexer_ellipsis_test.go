package lexer

import (
	"testing"
)

// TestEllipsisHandling verifies that ellipsis characters are properly rejected in calculations
// TDD: Ellipsis (... or …) should error in calc blocks but be allowed in markdown
func TestEllipsisInCalculation(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{
			name:      "three dots in expression",
			input:     "x = 10 ...",
			wantError: true,
		},
		{
			name:      "unicode ellipsis in expression",
			input:     "x = 10 …",
			wantError: true,
		},
		{
			name:      "ellipsis in middle",
			input:     "x = 10 ... 20",
			wantError: true,
		},
		{
			name:      "valid expression no ellipsis",
			input:     "x = 10 + 20",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lex := NewLexer(tt.input)
			_, err := lex.Tokenize()

			if tt.wantError && err == nil {
				t.Errorf("Expected error for %q, got none", tt.input)
			}
			if !tt.wantError && err != nil {
				t.Errorf("Unexpected error for %q: %v", tt.input, err)
			}
		})
	}
}
