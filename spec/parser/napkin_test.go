package parser

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/ast"
)

// TestNapkinFormat tests the napkin conversion syntax parsing
func TestNapkinFormat(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		// Positive numbers
		{
			name:    "small positive number",
			input:   "x = 47 as napkin\n",
			wantErr: false,
		},
		{
			name:    "large positive number",
			input:   "x = 1234567 as napkin\n",
			wantErr: false,
		},

		// Negative numbers
		{
			name:    "small negative number",
			input:   "x = -47 as napkin\n",
			wantErr: false,
		},
		{
			name:    "large negative number",
			input:   "x = -1234567 as napkin\n",
			wantErr: false,
		},

		// With quantities - napkin works with any numeric type
		{
			name:    "quantity with napkin",
			input:   "x = 100 MB as napkin\n",
			wantErr: false,
		},

		// Parenthesized expressions
		{
			name:    "expression in parentheses",
			input:   "x = (100 + 50) as napkin\n",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for %q, got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tt.input, err)
			}

			if len(nodes) == 0 {
				t.Fatalf("Parse(%q) returned no nodes", tt.input)
			}

			// Verify it's an assignment with napkin conversion
			assign, ok := nodes[0].(*ast.Assignment)
			if !ok {
				t.Fatalf("Parse(%q) expected Assignment, got %T", tt.input, nodes[0])
			}

			// The value should be a NapkinConversion
			napkin, ok := assign.Value.(*ast.NapkinConversion)
			if !ok {
				t.Fatalf("Parse(%q) expected NapkinConversion, got %T", tt.input, assign.Value)
			}

			// Should have an expression to convert
			if napkin.Expression == nil {
				t.Errorf("Parse(%q) napkin conversion has nil Expression", tt.input)
			}
		})
	}
}
