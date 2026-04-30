package interpreter_test

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/v2/impl/interpreter"
	"github.com/CalcMark/go-calcmark/v2/spec/parser"
)

// TestNaturalLanguageForms verifies that natural language function forms
// produce exactly the same results as their standard function equivalents.
//
// From spec/lexer/token.go lines 93-94:
// - FUNC_AVERAGE_OF: "average of" -> maps to "avg"
// - FUNC_SQUARE_ROOT_OF: "square root of" -> maps to "sqrt"
//
// These are the ONLY natural language forms in the lexer.
func TestNaturalLanguageForms(t *testing.T) {
	tests := []struct {
		name     string
		nlForm   string // Natural language form
		stdForm  string // Standard function form
		expected string // Expected result value
	}{
		// ========== "average of" -> avg() ==========
		{
			name:     "average of 3 numbers",
			nlForm:   "average of 1, 2, 3\n",
			stdForm:  "avg(1, 2, 3)\n",
			expected: "2",
		},
		{
			name:     "average of 5 numbers",
			nlForm:   "average of 10, 20, 30, 40, 50\n",
			stdForm:  "avg(10, 20, 30, 40, 50)\n",
			expected: "30",
		},
		{
			name:     "average of 2 numbers",
			nlForm:   "average of 100, 200\n",
			stdForm:  "avg(100, 200)\n",
			expected: "150",
		},
		{
			name:     "average of with decimals",
			nlForm:   "average of 1.5, 2.5, 3.0\n",
			stdForm:  "avg(1.5, 2.5, 3.0)\n",
			expected: "2.333", // Partial match - actual is repeating decimal
		},
		{
			name:     "average of single number",
			nlForm:   "average of 42\n",
			stdForm:  "avg(42)\n",
			expected: "42",
		},
		{
			name:     "average of negative numbers",
			nlForm:   "average of -10, 10\n",
			stdForm:  "avg(-10, 10)\n",
			expected: "0",
		},
		{
			name:     "average of large numbers",
			nlForm:   "average of 1000000, 2000000, 3000000\n",
			stdForm:  "avg(1000000, 2000000, 3000000)\n",
			expected: "2000000",
		},

		// ========== "square root of" -> sqrt() ==========
		{
			name:     "square root of 25",
			nlForm:   "square root of 25\n",
			stdForm:  "sqrt(25)\n",
			expected: "5",
		},
		{
			name:     "square root of 100",
			nlForm:   "square root of 100\n",
			stdForm:  "sqrt(100)\n",
			expected: "10",
		},
		{
			name:     "square root of 2",
			nlForm:   "square root of 2\n",
			stdForm:  "sqrt(2)\n",
			expected: "1.414", // Partial match - irrational number
		},
		{
			name:     "square root of 0",
			nlForm:   "square root of 0\n",
			stdForm:  "sqrt(0)\n",
			expected: "0",
		},
		{
			name:     "square root of 1",
			nlForm:   "square root of 1\n",
			stdForm:  "sqrt(1)\n",
			expected: "1",
		},
		{
			name:     "square root of 4",
			nlForm:   "square root of 4\n",
			stdForm:  "sqrt(4)\n",
			expected: "2",
		},
		{
			name:     "square root of 16",
			nlForm:   "square root of 16\n",
			stdForm:  "sqrt(16)\n",
			expected: "4",
		},
		{
			name:     "square root of 81",
			nlForm:   "square root of 81\n",
			stdForm:  "sqrt(81)\n",
			expected: "9",
		},
		{
			name:     "square root of decimal",
			nlForm:   "square root of 2.25\n",
			stdForm:  "sqrt(2.25)\n",
			expected: "1.5",
		},
		{
			name:     "square root of large number",
			nlForm:   "square root of 1000000\n",
			stdForm:  "sqrt(1000000)\n",
			expected: "1000",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Evaluate natural language form
			nlResult := evaluateExpression(t, tc.nlForm)

			// Evaluate standard form
			stdResult := evaluateExpression(t, tc.stdForm)

			// Both should produce same result (exact match)
			if nlResult != stdResult {
				t.Errorf("NL form %q produced %s, but std form %q produced %s",
					strings.TrimSpace(tc.nlForm), nlResult,
					strings.TrimSpace(tc.stdForm), stdResult)
			}

			// Result should contain expected prefix (handles repeating decimals)
			if !strings.HasPrefix(nlResult, tc.expected) {
				t.Errorf("Expected result to start with %q, got %q", tc.expected, nlResult)
			}

			t.Logf("NL: %s -> %s (std: %s)", strings.TrimSpace(tc.nlForm), nlResult, stdResult)
		})
	}
}

// TestNaturalLanguageFormEquivalence tests that NL forms are semantically
// equivalent to standard forms in complex expressions.
func TestNaturalLanguageFormEquivalence(t *testing.T) {
	tests := []struct {
		name    string
		nlForm  string
		stdForm string
	}{
		{
			name:    "average with variable assignment",
			nlForm:  "x = average of 10, 20, 30\nx * 2\n",
			stdForm: "x = avg(10, 20, 30)\nx * 2\n",
		},
		{
			name:    "sqrt in arithmetic expression",
			nlForm:  "(square root of 16) + 6\n",
			stdForm: "sqrt(16) + 6\n",
		},
		{
			name:    "combined with parentheses",
			nlForm:  "(square root of 4) + (average of 2, 2)\n",
			stdForm: "sqrt(4) + avg(2, 2)\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			nlResult := evaluateExpression(t, tc.nlForm)
			stdResult := evaluateExpression(t, tc.stdForm)

			if nlResult != stdResult {
				t.Errorf("NL form produced %s, std form produced %s", nlResult, stdResult)
			}

			t.Logf("Both forms -> %s", nlResult)
		})
	}
}

// evaluateExpression parses and evaluates a CalcMark expression, returning
// the string representation of the result.
func evaluateExpression(t *testing.T, input string) string {
	t.Helper()

	nodes, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse error for %q: %v", input, err)
	}

	interp := interpreter.NewInterpreter()
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval error for %q: %v", input, err)
	}

	if len(results) == 0 {
		t.Fatalf("No results returned for %q", input)
	}

	// Return the last result (handles multi-line expressions)
	return results[len(results)-1].String()
}
