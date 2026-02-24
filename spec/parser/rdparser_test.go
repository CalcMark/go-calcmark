package parser_test

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/parser"
)

// TestRecursiveDescentBasics tests the basic functionality of the new parser.
func TestRecursiveDescentBasics(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "simple number",
			input:   "42\n",
			wantErr: false,
		},
		{
			name:    "simple addition",
			input:   "1 + 2\n",
			wantErr: false,
		},
		{
			name:    "simple assignment",
			input:   "x = 10\n",
			wantErr: false,
		},
		{
			name:    "parentheses",
			input:   "(1 + 2) * 3\n",
			wantErr: false,
		},
		{
			name:    "incomplete expression",
			input:   "1 +\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := parser.NewRecursiveDescentParser(tt.input)
			nodes, err := p.Parse()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(nodes) == 0 {
					t.Errorf("expected nodes but got none")
				}
			}
		})
	}
}
