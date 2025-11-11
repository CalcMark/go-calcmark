package validator

import (
	"testing"

	"github.com/CalcMark/go-calcmark/impl/evaluator"
)

// TestInvalidCurrencyFormatDiagnostics tests that users get helpful feedback
// when they use invalid currency formats like "100USD" or "100 USD"
func TestInvalidCurrencyFormatDiagnostics(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectDiagnostic bool
		diagnosticType   DiagnosticSeverity
		description      string
	}{
		{
			name:             "100USD - no space",
			input:            "100USD",
			expectDiagnostic: true,
			diagnosticType:   Error,
			description:      "Should show syntax error - NUMBER followed by IDENTIFIER is invalid",
		},
		{
			name:             "100 USD - with space",
			input:            "100 USD",
			expectDiagnostic: true,
			diagnosticType:   Error,
			description:      "Should show syntax error - postfix not yet supported",
		},
		{
			name:             "USD100 - prefix format",
			input:            "USD100",
			expectDiagnostic: false,
			description:      "Valid prefix format - should work",
		},
		{
			name:             "bonus = 100USD + 20%",
			input:            "bonus = 100USD + 20%",
			expectDiagnostic: true,
			diagnosticType:   Error,
			description:      "Assignment with invalid format should error",
		},
		{
			name:             "bonus = USD100 + 20%",
			input:            "bonus = USD100 + 20%",
			expectDiagnostic: false,
			description:      "Assignment with valid prefix format should work",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := evaluator.NewContext()
			result := ValidateCalculation(tt.input, ctx)

			hasDiagnostic := len(result.Diagnostics) > 0
			hasError := result.HasErrors()

			if tt.expectDiagnostic && !hasDiagnostic {
				t.Errorf("Expected diagnostic for %q but got none", tt.input)
				t.Logf("This means users won't get feedback about invalid currency format")
			}

			if !tt.expectDiagnostic && hasDiagnostic {
				t.Errorf("Did not expect diagnostic for %q but got:", tt.input)
				for _, diag := range result.Diagnostics {
					t.Logf("  %s: %s", diag.Severity, diag.Message)
				}
			}

			if tt.expectDiagnostic && hasDiagnostic {
				// Log what diagnostic we got
				t.Logf("✓ Got diagnostic: %s", result.Diagnostics[0].Message)

				if tt.diagnosticType == Error && !hasError {
					t.Errorf("Expected ERROR severity but got %s", result.Diagnostics[0].Severity)
				}
			}

			t.Logf("✓ %s", tt.description)
		})
	}
}

// TestCurrencyFormatHintOpportunity tests whether we should add a HINT
// to help users understand valid currency formats
func TestCurrencyFormatHintOpportunity(t *testing.T) {
	// This test documents the current behavior and identifies where
	// we might want to add helpful hints

	tests := []struct {
		input       string
		currentlyAs string
		idealHint   string
	}{
		{
			input:       "100USD",
			currentlyAs: "syntax error",
			idealHint:   "Currency codes must be prefix format (e.g., USD100) or use symbols (e.g., $100)",
		},
		{
			input:       "100 USD",
			currentlyAs: "syntax error",
			idealHint:   "Postfix currency format not yet supported. Use USD100 instead of 100 USD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			ctx := evaluator.NewContext()
			result := ValidateCalculation(tt.input, ctx)

			t.Logf("Input: %q", tt.input)
			t.Logf("Currently treated as: %s", tt.currentlyAs)
			t.Logf("Ideal hint: %s", tt.idealHint)

			if len(result.Diagnostics) > 0 {
				t.Logf("Current diagnostic: %s", result.Diagnostics[0].Message)
			} else {
				t.Logf("⚠️  No diagnostic shown - user gets no feedback!")
			}
		})
	}
}
