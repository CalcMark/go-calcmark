package interpreter

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/parser"
)

// TestNapkinWithNegativeNumbers tests that negative numbers work correctly with napkin conversion
func TestNapkinWithNegativeNumbers(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "negative number with napkin",
			input:   "x = -1234567 as napkin\n",
			wantErr: false, // Should work - napkin applies to the negative number
		},
		{
			name:    "positive number with napkin",
			input:   "x = 1234567 as napkin\n",
			wantErr: false,
		},
		{
			name:    "small negative with napkin",
			input:   "x = -47 as napkin\n",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			// Interpret
			interp := NewInterpreter()
			results, err := interp.Eval(nodes)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for %q", tt.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("Eval(%q) unexpected error: %v", tt.input, err)
			}

			if len(results) == 0 {
				t.Fatalf("Eval(%q) returned no results", tt.input)
			}

			// Result should be a Number (napkin converts to Number)
			if results[0] == nil {
				t.Errorf("Eval(%q) returned nil result", tt.input)
			}
		})
	}
}
