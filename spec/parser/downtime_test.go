package parser

import (
	"testing"
)

func TestDowntimeNaturalSyntax(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		description string
	}{
		// Valid percentage literals
		{
			name:        "99.9% downtime per month",
			input:       "99.9% downtime per month\n",
			expectError: false,
			description: "Standard availability percentage",
		},
		{
			name:        "99.99% downtime per year",
			input:       "99.99% downtime per year\n",
			expectError: false,
			description: "High availability",
		},
		{
			name:        "99.999% downtime per day",
			input:       "99.999% downtime per day\n",
			expectError: false,
			description: "Very high availability",
		},
		{
			name:        "1.2222252% downtime per year",
			input:       "1.2222252% downtime per year\n",
			expectError: false,
			description: "Arbitrary precision percentage",
		},
		{
			name:        "0.5% downtime per week",
			input:       "0.5% downtime per week\n",
			expectError: false,
			description: "Sub-one percent",
		},
		{
			name:        "100% downtime per hour",
			input:       "100% downtime per hour\n",
			expectError: false,
			description: "Full downtime (edge case)",
		},

		// Valid with different time units
		{
			name:        "downtime per second",
			input:       "95% downtime per second\n",
			expectError: false,
			description: "Second time unit",
		},
		{
			name:        "downtime per minute",
			input:       "98% downtime per minute\n",
			expectError: false,
			description: "Minute time unit",
		},
		{
			name:        "downtime per hour",
			input:       "99% downtime per hour\n",
			expectError: false,
			description: "Hour time unit",
		},

		// Error cases
		{
			name:        "missing per keyword",
			input:       "99.9% downtime month\n",
			expectError: true,
			description: "Should require 'per'",
		},
		{
			name:        "missing time unit",
			input:       "99.9% downtime per\n",
			expectError: true,
			description: "Should require time unit after 'per'",
		},
		{
			name:        "invalid time unit",
			input:       "99.9% downtime per invalidunit\n",
			expectError: true,
			description: "Should validate time unit",
		},

		// Regression tests: ensure rate parsing still works
		{
			name:        "REGRESSION: simple rate with per",
			input:       "100 MB per second\n",
			expectError: false,
			description: "Rate parsing should still work",
		},
		{
			name:        "REGRESSION: quantity rate with per",
			input:       "5 GB per day\n",
			expectError: false,
			description: "Quantity + per should create rate",
		},
		{
			name:        "REGRESSION: number rate with per",
			input:       "60 per minute\n",
			expectError: false,
			description: "Plain number + per should create rate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none. Description: %s", tt.description)
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v. Description: %s", err, tt.description)
				return
			}

			if !tt.expectError {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}

				t.Logf("✓ %s - %s", tt.name, tt.description)
			}
		})
	}
}
