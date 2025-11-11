package validator

import (
	"testing"

	"github.com/CalcMark/go-calcmark/impl/evaluator"
)

// TestPercentageOnLeftSideOfAddition tests that percentage literals on left side of + trigger diagnostic
func TestPercentageOnLeftSideOfAddition(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantError   bool
		description string
	}{
		{
			name:        "percentage plus number",
			input:       "20% + 5",
			wantError:   true,
			description: "Should produce diagnostic for percentage on left of +",
		},
		{
			name:        "percentage plus currency",
			input:       "20% + $10",
			wantError:   true,
			description: "Should produce diagnostic for percentage on left of +",
		},
		{
			name:        "number plus percentage (valid)",
			input:       "100 + 20%",
			wantError:   false,
			description: "Valid: percentage on right side",
		},
		{
			name:        "currency plus percentage (valid)",
			input:       "$100 + 20%",
			wantError:   false,
			description: "Valid: percentage on right side",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := evaluator.NewContext()
			result := ValidateCalculation(tt.input, ctx)

			hasPercentageError := false
			for _, diag := range result.Diagnostics {
				if diag.Code == PercentageOnLeftOfAdditionSubtraction {
					hasPercentageError = true
					t.Logf("✓ Found expected diagnostic: %s", diag.Message)
				}
			}

			if tt.wantError && !hasPercentageError {
				t.Errorf("Expected percentage diagnostic for %q, but got none", tt.input)
			}

			if !tt.wantError && hasPercentageError {
				t.Errorf("Did not expect percentage diagnostic for %q, but got one", tt.input)
			}
		})
	}
}

// TestPercentageOnLeftSideOfSubtraction tests that percentage literals on left side of - trigger diagnostic
func TestPercentageOnLeftSideOfSubtraction(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantError   bool
		description string
	}{
		{
			name:        "percentage minus number",
			input:       "20% - 2",
			wantError:   true,
			description: "Should produce diagnostic for percentage on left of -",
		},
		{
			name:        "percentage minus currency",
			input:       "20% - $10",
			wantError:   true,
			description: "Should produce diagnostic for percentage on left of -",
		},
		{
			name:        "number minus percentage (valid)",
			input:       "120 - 20%",
			wantError:   false,
			description: "Valid: percentage on right side",
		},
		{
			name:        "currency minus percentage (valid)",
			input:       "$120 - 20%",
			wantError:   false,
			description: "Valid: percentage on right side",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := evaluator.NewContext()
			result := ValidateCalculation(tt.input, ctx)

			hasPercentageError := false
			for _, diag := range result.Diagnostics {
				if diag.Code == PercentageOnLeftOfAdditionSubtraction {
					hasPercentageError = true
					t.Logf("✓ Found expected diagnostic: %s", diag.Message)
				}
			}

			if tt.wantError && !hasPercentageError {
				t.Errorf("Expected percentage diagnostic for %q, but got none", tt.input)
			}

			if !tt.wantError && hasPercentageError {
				t.Errorf("Did not expect percentage diagnostic for %q, but got one", tt.input)
			}
		})
	}
}
