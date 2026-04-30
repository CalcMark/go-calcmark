package interpreter_test

import (
	"testing"

	"github.com/CalcMark/go-calcmark/v2/impl/interpreter"
	"github.com/CalcMark/go-calcmark/v2/spec/parser"
)

// TestCurrencyNumberOperations verifies currency +/-/* /÷ number operations.
// Issue #16: Adding a number to a currency value should work.
func TestCurrencyNumberOperations(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Currency + Number → Currency (keeps currency unit)
		{"EUR addition", "a = 10 EUR\nb = a + 2\n", "EUR12.00"},
		{"USD addition", "$100 + 5\n", "$105.00"},
		{"GBP addition", "£50 + 10\n", "£60.00"},

		// Currency - Number → Currency
		{"EUR subtraction", "a = 10 EUR\nb = a - 3\n", "EUR7.00"},
		{"USD subtraction", "$100 - 25\n", "$75.00"},

		// Currency / Number → Currency
		{"EUR division", "a = 10 EUR\nb = a / 2\n", "EUR5.00"},
		{"USD division", "$100 / 4\n", "$25.00"},

		// Currency * Number → Currency (already works, verify it still does)
		{"EUR multiplication", "a = 10 EUR\nb = a * 3\n", "EUR30.00"},
		{"USD multiplication", "$100 * 2\n", "$200.00"},

		// Number + Currency → Currency
		{"number + EUR", "a = 10 EUR\nb = 2 + a\n", "EUR12.00"},
		{"number + USD", "5 + $100\n", "$105.00"},

		// Number - Currency → Currency
		{"number - EUR", "a = 10 EUR\nb = 20 - a\n", "EUR10.00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			interp := interpreter.NewInterpreter()
			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error: %v", err)
			}

			if len(results) == 0 {
				t.Fatal("No results returned")
			}

			// Use the last result (for multi-line inputs with assignments)
			actual := results[len(results)-1].String()
			if actual != tt.expected {
				t.Errorf("Result = %s, expected %s", actual, tt.expected)
			}
		})
	}
}

// TestCurrencyNumberErrors verifies that mixing different currencies still fails.
func TestCurrencyNumberErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"USD + EUR", "$100 + 50 EUR\n"},
		{"EUR - GBP", "10 EUR - 5 GBP\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				return // Parse error is fine
			}

			interp := interpreter.NewInterpreter()
			_, err = interp.Eval(nodes)
			if err == nil {
				t.Errorf("Expected error for %q but got none", tt.input)
			}
		})
	}
}
