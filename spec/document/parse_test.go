package document

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/parser"
)

// TestParseability checks what the parser considers valid
func TestParseability(t *testing.T) {
	tests := []struct {
		input       string
		shouldParse bool
	}{
		{"# Header", false},
		{"x = 10", true},
		{"Some text", false},
		{"1 + 1", true},
		{"**bold**", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, err := parser.Parse(tt.input + "\n")
			parseable := (err == nil)

			if parseable != tt.shouldParse {
				t.Errorf("%q: expected parseable=%v, got %v (err=%v)",
					tt.input, tt.shouldParse, parseable, err)
			}
		})
	}
}
