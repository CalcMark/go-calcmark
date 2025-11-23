package parser_test

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/parser"
)

// TestInvalidSyntax is the DEFINITIVE SPECIFICATION of all invalid CalcMark syntax.
// Every entry here MUST produce a parse error.
// If a test passes (no error), the spec has been violated.
func TestInvalidSyntax(t *testing.T) {
	tests := []struct {
		category    string
		input       string
		desc        string
		expectedErr string // Optional: substring that should appear in error
	}{
		// Incomplete expressions
		{"syntax", "1 +", "missing right operand after plus", ""},
		{"syntax", "10 -", "missing right operand after minus", ""},
		{"syntax", "5 *", "missing right operand after multiply", ""},
		{"syntax", "20 /", "missing right operand after divide", ""},
		{"syntax", "+ 1", "prefix plus without left context", ""},
		{"syntax", "* 5", "prefix multiply (invalid)", ""},

		// Missing operators
		{"syntax", "1 2", "adjacent numbers without operator", ""},
		{"syntax", "10 20", "missing operator between numbers", ""},
		{"syntax", "x y", "missing operator between identifiers", ""},

		// Parentheses
		{"syntax", "(1 + 2", "unclosed left parenthesis", ""},
		{"syntax", "1 + 2)", "unmatched right parenthesis", ""},
		{"syntax", "((1 + 2)", "missing closing parenthesis", ""},

		// Invalid functions - traditional
		{"functions", "avg()", "avg with no arguments", ""},
		{"functions", "sqrt()", "sqrt with no arguments", ""},
		{"functions", "sqrt(1, 2)", "sqrt with too many arguments", ""},
		{"functions", "unknown(1)", "undefined function name", ""},

		// Invalid functions - natural language
		{"functions", "average", "incomplete natural language (missing 'of')", ""},
		{"functions", "average of", "natural language without arguments", ""},
		{"functions", "square root", "incomplete natural language (missing 'of')", ""},
		{"functions", "average 1, 2, 3", "missing 'of' in natural language", ""},

		// Invalid assignments
		{"assignments", "= 10", "assignment without variable", ""},
		{"assignments", "123 = x", "cannot assign to number literal", ""},
		{"assignments", "(x + y) = 10", "cannot assign to expression", ""},

		// Invalid operators
		{"operators", "1 ++ 2", "double plus operator", ""},
		{"operators", "x === y", "triple equals (not supported)", ""},
		{"operators", "1 ** 2", "double asterisk (use ^ for power)", ""},

		// Invalid dates
		{"dates", "February 30", "invalid day for month (Feb has max 29)", ""},
		{"dates", "Dec 32", "day out of range (max 31)", ""},
		{"dates", "Month 12", "invalid month name", ""},
		{"dates", "13 days", "number before time unit (should be 'X days')", ""},

		// Invalid durations
		{"durations", "2 days and", "incomplete 'and' expression", ""},
		{"durations", "and 3 days", "'and' without left operand", ""},
		{"durations", "2 unknownunit", "invalid time unit", ""},

		// Invalid date expressions
		{"date_expr", "from today", "missing duration before 'from'", ""},
		{"date_expr", "2 weeks from", "missing date after 'from'", ""},
		{"date_expr", "+ 2 weeks", "duration without date in arithmetic", ""},
	}

	for _, tc := range tests {
		t.Run(tc.category+"/"+tc.desc, func(t *testing.T) {
			input := tc.input + "\n"

			_, err := parser.Parse(input)
			if err == nil {
				t.Errorf("SPEC VIOLATION: %q should be INVALID but parsed successfully\n"+
					"  Category: %s\n"+
					"  Description: %s",
					tc.input, tc.category, tc.desc)
				return
			}

			// Optionally check error message contains expected substring
			if tc.expectedErr != "" && !strings.Contains(err.Error(), tc.expectedErr) {
				t.Logf("Warning: Error message doesn't contain expected substring\n"+
					"  Input: %q\n"+
					"  Expected substring: %q\n"+
					"  Got: %v",
					tc.input, tc.expectedErr, err)
			}
		})
	}
}
