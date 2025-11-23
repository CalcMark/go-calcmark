package parser_test

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/parser"
)

// TestValidSyntax is the DEFINITIVE SPECIFICATION of all valid CalcMark syntax.
// Every entry here MUST parse successfully.
// If a test fails, either:
// 1. The parser has a bug, OR
// 2. The spec has changed (update this test)
func TestValidSyntax(t *testing.T) {
	tests := []struct {
		category string
		input    string
		desc     string
	}{
		// Numbers
		{"numbers", "42", "integer literal"},
		{"numbers", "3.14", "decimal literal"},
		{"numbers", "1.2e10", "scientific notation (lowercase e)"},
		{"numbers", "1.2E10", "scientific notation (uppercase E)"},
		{"numbers", "12k", "thousands multiplier (lowercase)"},
		{"numbers", "12K", "thousands multiplier (uppercase)"},
		{"numbers", "1.5M", "millions multiplier"},
		{"numbers", "2B", "billions multiplier"},
		{"numbers", "1T", "trillions multiplier"},
		{"numbers", "25%", "percentage"},

		// Arithmetic operators
		{"arithmetic", "1 + 2", "addition"},
		{"arithmetic", "10 - 5", "subtraction"},
		{"arithmetic", "3 * 4", "multiplication"},
		{"arithmetic", "20 / 5", "division"},
		{"arithmetic", "17 % 5", "modulus"},
		{"arithmetic", "2 ^ 3", "exponentiation"},

		// Unary operators
		{"arithmetic", "+10", "unary plus"},
		{"arithmetic", "-5", "unary minus"},
		{"arithmetic", "+(10)", "unary plus with parens"},
		{"arithmetic", "-(10)", "unary minus with parens"},

		// Precedence & parentheses
		{"precedence", "2 + 3 * 4", "multiplication before addition"},
		{"precedence", "(2 + 3) * 4", "parentheses override precedence"},
		{"precedence", "2 ^ 3 * 4", "exponent before multiplication"},
		{"precedence", "((1 + 2) * 3)", "nested parentheses"},

		// Variables & assignments
		{"variables", "x = 10", "simple assignment"},
		{"variables", "total = price + tax", "assignment with expression"},
		{"variables", "result = (a + b) * c", "assignment with complex expression"},
		{"variables", "x", "variable reference"},

		// Comparisons
		{"comparisons", "x == 10", "equality"},
		{"comparisons", "x != 10", "inequality"},
		{"comparisons", "x > 10", "greater than"},
		{"comparisons", "x < 10", "less than"},
		{"comparisons", "x >= 10", "greater or equal"},
		{"comparisons", "x <= 10", "less or equal"},

		// Functions - traditional
		{"functions_trad", "avg(1, 2, 3)", "avg with 3 args"},
		{"functions_trad", "avg(10)", "avg with 1 arg"},
		{"functions_trad", "sqrt(16)", "sqrt with 1 arg"},
		{"functions_trad", "avg(x, y, z)", "avg with variables"},

		// Functions - natural language
		{"functions_nl", "average of 1, 2, 3", "average of 3 numbers"},
		{"functions_nl", "average of 10, 20", "average of 2 numbers"},
		{"functions_nl", "square root of 16", "square root of number"},
		{"functions_nl", "square root of x", "square root of variable"},

		// Currency
		{"currency", "$100", "dollar without decimals"},
		{"currency", "$100.50", "dollar with decimals"},
		{"currency", "€50.25", "euro"},
		{"currency", "£30", "pound"},
		{"currency", "100 USD", "currency code"},
		{"currency", "50.25 EUR", "currency code with decimals"},

		// Units & quantities
		{"units", "10 meters", "quantity with unit"},
		{"units", "5.5 kg", "decimal quantity"},
		{"units", "100 km/h", "compound unit"},
		{"units", "x meters", "variable with unit"},

		// Dates - relative
		{"dates", "December 12", "month + day (full name)"},
		{"dates", "Dec 12", "month + day (abbreviated)"},
		{"dates", "January 1", "first of month"},

		// Dates - absolute
		{"dates", "December 12 2025", "full date"},
		{"dates", "Dec 12 2025", "abbreviated full date"},
		{"dates", "December 2025", "month + year (1st of month)"},

		// Date keywords
		{"dates", "today", "today keyword"},
		{"dates", "tomorrow", "tomorrow keyword"},
		{"dates", "yesterday", "yesterday keyword"},

		// Durations - simple
		{"durations", "1 day", "singular day"},
		{"durations", "2 days", "plural days"},
		{"durations", "3 weeks", "plural weeks"},
		{"durations", "1 month", "singular month"},
		{"durations", "2 years", "plural years"},
		{"durations", "5 hours", "hours"},
		{"durations", "30 minutes", "minutes"},
		{"durations", "45 seconds", "seconds"},

		// Durations - compound with 'and'
		{"durations", "2 weeks and 3 days", "weeks and days"},
		{"durations", "1 year and 6 months", "year and months"},
		{"durations", "3 months and 2 weeks and 5 days", "multi-term"},

		// Date arithmetic
		{"date_math", "December 12 + 30 days", "date plus duration"},
		{"date_math", "today + 2 weeks", "today plus weeks"},
		{"date_math", "Dec 25 2024 - 1 week", "date minus duration"},

		// Natural language date expressions
		{"date_nl", "2 weeks from today", "duration from today"},
		{"date_nl", "3 days from December 12", "duration from date"},
		{"date_nl", "1 month from Dec 25 2025", "duration from full date"},

		// Booleans (only true/false)
		{"booleans", "true", "boolean true"},
		{"booleans", "false", "boolean false"},

		// Complex expressions
		{"complex", "total = subtotal + (subtotal * 0.08)", "nested arithmetic"},
		{"complex", "avg(10, 20, 30) + 5", "function in expression"},
		{"complex", "x = average of 1, 2, 3", "natural language in assignment"},
		{"complex", "(December 12 + 1 week) + 2 days", "nested date arithmetic"},
	}

	for _, tc := range tests {
		t.Run(tc.category+"/"+tc.desc, func(t *testing.T) {
			input := tc.input + "\n" // Parser expects newline

			nodes, err := parser.Parse(input)
			if err != nil {
				t.Errorf("SPEC VIOLATION: %q should be VALID\n"+
					"  Category: %s\n"+
					"  Description: %s\n"+
					"  Error: %v",
					tc.input, tc.category, tc.desc, err)
				return
			}

			if len(nodes) == 0 {
				t.Errorf("SPEC VIOLATION: %q parsed but returned no nodes\n"+
					"  Category: %s\n"+
					"  Description: %s",
					tc.input, tc.category, tc.desc)
			}
		})
	}
}
