package parser

import (
	"testing"
)

func TestDowntimeNaturalSyntax(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "99.9% downtime per month",
			input:       "99.9% downtime per month\n",
			expectError: false,
		},
		{
			name:        "99.99% downtime per year",
			input:       "99.99% downtime per year\n",
			expectError: false,
		},
		{
			name:        "99.999% downtime per day",
			input:       "99.999% downtime per day\n",
			expectError: false,
		},
		{
			name:        "missing per",
			input:       "99.9% downtime month\n",
			expectError: true,
		},
		{
			name:        "missing time unit",
			input:       "99.9% downtime per\n",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if !tt.expectError {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}

				t.Logf("✓ Parsed: %+v", nodes[0])
			}
		})
	}
}
