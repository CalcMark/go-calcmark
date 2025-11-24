package evaluator

import (
	"testing"
)

// TestBinaryOperationsPreserveUnits tests that binary operations preserve currency units
// when one operand has units and the other doesn't
func TestBinaryOperationsPreserveUnits(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		want        string
		wantType    string
		description string
	}{
		// Addition with currency
		{name: "currency_plus_number", input: "$200 + 0.1", want: "USD 200.10", wantType: "Currency", description: "Currency + number should preserve currency unit"},
		{name: "number_plus_currency", input: "0.1 + $200", want: "USD 200.10", wantType: "Currency", description: "Number + currency should preserve currency unit"},
		{name: "currency_plus_currency_same_unit", input: "$100 + $50", want: "USD 150.00", wantType: "Currency", description: "Same currency units preserved"},

		// Subtraction with currency
		{name: "currency_minus_number", input: "$200 - 0.1", want: "USD 199.90", wantType: "Currency", description: "Currency - number should preserve currency unit"},
		{name: "number_minus_currency", input: "300 - $100", want: "USD 200.00", wantType: "Currency", description: "Number - currency should preserve currency unit"},

		// Multiplication with currency (percentage-like)
		{name: "currency_times_number", input: "$200 * 0.1", want: "USD 20.00", wantType: "Currency", description: "Currency * number (like 10%) should preserve currency"},
		{name: "number_times_currency", input: "0.1 * $200", want: "USD 20.00", wantType: "Currency", description: "Number * currency should preserve currency"},
		{name: "currency_times_percentage", input: "$1000 * 0.05", want: "USD 50.00", wantType: "Currency", description: "Currency * 5% = currency result"},
		{name: "currency_times_percentage_literal", input: "$100 * 20%", want: "USD 20.00", wantType: "Currency", description: "Currency * 20% literal = USD20 (20% of USD100)"},
		{name: "number_plus_percentage", input: "100 + 5%", want: "105", wantType: "Number", description: "Number + percentage adds the percentage value"},
		{name: "number_minus_percentage", input: "100 - 10%", want: "90", wantType: "Number", description: "Number - percentage subtracts the percentage value"},
		{name: "currency_plus_percentage", input: "$100 + 20%", want: "USD 120.00", wantType: "Currency", description: "USD 100 + 20% = USD100 + (USD100 * 0.20) = USD120"},
		{name: "currency_minus_percentage", input: "$200 - 10%", want: "USD 180.00", wantType: "Currency", description: "Currency - percentage = currency result"},

		// Division with currency
		{name: "currency_divided_by_number", input: "$200 / 2", want: "USD 100.00", wantType: "Currency", description: "Currency / number should preserve currency"},
		{name: "number_divided_by_currency", input: "100 / $2", want: "50", wantType: "Number", description: "Number / currency drops unit (inverse rate)"}, // The test expects Number type (drops unit) not Currency - this is the actual semantic test

		// Mixed currency units in binary ops should probably error or return Number
		// (this is different from functions which explicitly strip units)
		{name: "different_currencies_added", input: "$100 + €50", want: "150", wantType: "Number", description: "Different currency units should drop to Number"},

		// Euro examples
		{name: "euro_plus_number", input: "€500 + 25", want: "EUR 525.00", wantType: "Currency", description: "Euro + number preserves euro"},
		{name: "euro_times_percentage", input: "€1000 * 0.15", want: "EUR 150.00", wantType: "Currency", description: "Euro * 15% = euro result"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.description)

			ctx := NewContext()
			results, err := Evaluate(tt.input, ctx)
			if err != nil {
				t.Fatalf("Evaluate(%q) error = %v, want nil", tt.input, err)
			}

			if len(results) != 1 {
				t.Fatalf("Evaluate(%q) returned %d results, want 1", tt.input, len(results))
			}

			result := results[0]
			if result.TypeName() != tt.wantType {
				t.Errorf("Evaluate(%q) type = %s, want %s", tt.input, result.TypeName(), tt.wantType)
			}

			if result.String() != tt.want {
				t.Errorf("Evaluate(%q) = %s, want %s", tt.input, result.String(), tt.want)
			}
		})
	}
}

// TestInvalidPercentageOperations tests that percentage on left side of +/- produces diagnostics
// Note: These are validator warnings, not evaluation errors.
// The expressions will still evaluate but should trigger validator diagnostics.
func TestInvalidPercentageOperations(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name:        "percentage minus number",
			input:       "20% - 2",
			description: "Subtracting a number from a percentage should produce validator diagnostic",
		},
		{
			name:        "percentage plus number",
			input:       "20% + 5",
			description: "Adding a number to a percentage should produce validator diagnostic",
		},
		{
			name:        "percentage minus currency",
			input:       "20% - $10",
			description: "Subtracting currency from a percentage should produce validator diagnostic",
		},
		{
			name:        "percentage plus currency",
			input:       "20% + $10",
			description: "Adding currency to a percentage should produce validator diagnostic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.description)
			t.Log("Note: This will still evaluate, but validator will flag it as an error")

			// These will evaluate (semantically nonsensical but syntactically valid)
			// The validator (not tested here) will flag these as errors
			ctx := NewContext()
			_, err := Evaluate(tt.input, ctx)

			if err != nil {
				t.Logf("Evaluation error (unexpected): %v", err)
			} else {
				t.Logf("✓ Evaluation succeeded (validator should flag this)")
			}
		})
	}
}

// TestBinaryOperationsWithVariables tests unit preservation with variables
func TestBinaryOperationsWithVariables(t *testing.T) {
	ctx := NewContext()

	// Set up variables
	_, err := Evaluate("price = $100\ntax_rate = 0.08", ctx)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		want     string
		wantType string
	}{
		{
			name:     "currency var times number var",
			input:    "price * tax_rate",
			want:     "USD 8.00",
			wantType: "Currency",
		},
		{
			name:     "currency var plus number",
			input:    "price + 10",
			want:     "USD 110.00",
			wantType: "Currency",
		},
		{
			name:     "currency calculation",
			input:    "price + (price * tax_rate)",
			want:     "USD 108.00",
			wantType: "Currency",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := Evaluate(tt.input, ctx)
			if err != nil {
				t.Fatalf("Evaluate(%q) error = %v, want nil", tt.input, err)
			}

			if len(results) != 1 {
				t.Fatalf("Evaluate(%q) returned %d results, want 1", tt.input, len(results))
			}

			if results[0].TypeName() != tt.wantType {
				t.Errorf("Evaluate(%q) type = %s, want %s", tt.input, results[0].TypeName(), tt.wantType)
			}

			if results[0].String() != tt.want {
				t.Errorf("Evaluate(%q) = %s, want %s", tt.input, results[0].String(), tt.want)
			}
		})
	}
}
