package evaluator

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/impl/types"
	"github.com/CalcMark/go-calcmark/spec/parser"
	"github.com/shopspring/decimal"
)

// Helper function for tests
func mustEvaluate(t *testing.T, text string, context *Context) []types.Type {
	t.Helper()
	results, err := Evaluate(text, context)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return results
}

func TestEvalSimpleNumber(t *testing.T) {
	results := mustEvaluate(t, "42", nil)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	num, ok := results[0].(*types.Number)
	if !ok {
		t.Fatalf("expected Number, got %T", results[0])
	}

	expected := decimal.NewFromInt(42)
	if !num.Value.Equal(expected) {
		t.Errorf("expected %s, got %s", expected, num.Value)
	}
}

func TestEvalCurrency(t *testing.T) {
	results := mustEvaluate(t, "$1000", nil)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	curr, ok := results[0].(*types.Currency)
	if !ok {
		t.Fatalf("expected Currency, got %T", results[0])
	}

	if curr.Symbol != "$" {
		t.Errorf("expected symbol '$', got '%s'", curr.Symbol)
	}

	expected := decimal.NewFromInt(1000)
	if !curr.Value.Equal(expected) {
		t.Errorf("expected %s, got %s", expected, curr.Value)
	}
}

func TestEvalMultiplication(t *testing.T) {
	results := mustEvaluate(t, "3 * 3", nil)
	num := results[0].(*types.Number)

	expected := decimal.NewFromInt(9)
	if !num.Value.Equal(expected) {
		t.Errorf("expected %s, got %s", expected, num.Value)
	}
}

func TestEvalAddition(t *testing.T) {
	results := mustEvaluate(t, "5 + 3", nil)
	num := results[0].(*types.Number)

	expected := decimal.NewFromInt(8)
	if !num.Value.Equal(expected) {
		t.Errorf("expected %s, got %s", expected, num.Value)
	}
}

func TestEvalSubtraction(t *testing.T) {
	results := mustEvaluate(t, "10 - 3", nil)
	num := results[0].(*types.Number)

	expected := decimal.NewFromInt(7)
	if !num.Value.Equal(expected) {
		t.Errorf("expected %s, got %s", expected, num.Value)
	}
}

func TestEvalDivision(t *testing.T) {
	results := mustEvaluate(t, "20 / 4", nil)
	num := results[0].(*types.Number)

	expected := decimal.NewFromInt(5)
	if !num.Value.Equal(expected) {
		t.Errorf("expected %s, got %s", expected, num.Value)
	}
}

func TestEvalDivisionByZero(t *testing.T) {
	_, err := Evaluate("10 / 0", nil)
	if err == nil {
		t.Error("expected error for division by zero")
	}
	// Just check that error message contains "Division by zero"
	if err != nil {
		errMsg := err.Error()
		if !strings.Contains(errMsg, "Division by zero") {
			t.Errorf("expected error containing 'Division by zero', got '%s'", errMsg)
		}
	}
}

func TestCurrencyMultiplication(t *testing.T) {
	results := mustEvaluate(t, "$100 * 5", nil)
	curr, ok := results[0].(*types.Currency)
	if !ok {
		t.Fatalf("expected Currency, got %T", results[0])
	}

	expected := decimal.NewFromInt(500)
	if !curr.Value.Equal(expected) {
		t.Errorf("expected %s, got %s", expected, curr.Value)
	}
}

func TestNumberTimesCurrency(t *testing.T) {
	results := mustEvaluate(t, "5 * $100", nil)
	curr, ok := results[0].(*types.Currency)
	if !ok {
		t.Fatalf("expected Currency, got %T", results[0])
	}

	expected := decimal.NewFromInt(500)
	if !curr.Value.Equal(expected) {
		t.Errorf("expected %s, got %s", expected, curr.Value)
	}
}

func TestCurrencyDivision(t *testing.T) {
	results := mustEvaluate(t, "$100 / 4", nil)
	curr, ok := results[0].(*types.Currency)
	if !ok {
		t.Fatalf("expected Currency, got %T", results[0])
	}

	expected := decimal.NewFromInt(25)
	if !curr.Value.Equal(expected) {
		t.Errorf("expected %s, got %s", expected, curr.Value)
	}
}

func TestSimpleAssignment(t *testing.T) {
	results := mustEvaluate(t, "x = 5", nil)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	num, ok := results[0].(*types.Number)
	if !ok {
		t.Fatalf("expected Number, got %T", results[0])
	}

	expected := decimal.NewFromInt(5)
	if !num.Value.Equal(expected) {
		t.Errorf("expected %s, got %s", expected, num.Value)
	}
}

func TestVariableReference(t *testing.T) {
	context := NewContext()
	results := mustEvaluate(t, "x = 5\nx", context)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	num := results[1].(*types.Number)
	expected := decimal.NewFromInt(5)
	if !num.Value.Equal(expected) {
		t.Errorf("expected %s, got %s", expected, num.Value)
	}
}

func TestVariableInExpression(t *testing.T) {
	results := mustEvaluate(t, "x = 5\ny = x * 2", nil)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	num := results[1].(*types.Number)
	expected := decimal.NewFromInt(10)
	if !num.Value.Equal(expected) {
		t.Errorf("expected %s, got %s", expected, num.Value)
	}
}

func TestUndefinedVariable(t *testing.T) {
	_, err := Evaluate("undefined_var", nil)
	if err == nil {
		t.Error("expected error for undefined variable")
	}
	if err != nil && err.Error() != "Undefined variable 'undefined_var'" {
		errMsg := err.Error()
		if len(errMsg) < 19 || errMsg[:19] != "Undefined variable " {
			t.Errorf("expected 'Undefined variable' error, got '%s'", errMsg)
		}
	}
}

func TestMultiplicationBeforeAddition(t *testing.T) {
	results := mustEvaluate(t, "2 + 3 * 4", nil)
	num := results[0].(*types.Number)

	expected := decimal.NewFromInt(14)
	if !num.Value.Equal(expected) {
		t.Errorf("expected %s, got %s", expected, num.Value)
	}
}

func TestComplexExpression(t *testing.T) {
	results := mustEvaluate(t, "10 + 5 * 2 - 3", nil)
	num := results[0].(*types.Number)

	expected := decimal.NewFromInt(17)
	if !num.Value.Equal(expected) {
		t.Errorf("expected %s, got %s", expected, num.Value)
	}
}

func TestContextPersists(t *testing.T) {
	context := NewContext()
	mustEvaluate(t, "x = 10", context)

	results := mustEvaluate(t, "y = x * 2", context)
	num := results[0].(*types.Number)

	expected := decimal.NewFromInt(20)
	if !num.Value.Equal(expected) {
		t.Errorf("expected %s, got %s", expected, num.Value)
	}
}

func TestVariableShadowing(t *testing.T) {
	context := NewContext()
	evaluator := NewEvaluator(context)

	nodes, err := parser.Parse("x = 5\nx = 10")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	_, err = evaluator.Eval(nodes)
	if err != nil {
		t.Fatalf("unexpected eval error: %v", err)
	}

	value, err := context.Get("x")
	if err != nil {
		t.Fatalf("unexpected error getting x: %v", err)
	}

	num := value.(*types.Number)
	expected := decimal.NewFromInt(10)
	if !num.Value.Equal(expected) {
		t.Errorf("expected %s, got %s", expected, num.Value)
	}
}

func TestBooleanKeywordResolution(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"true", "true", true},
		{"false", "false", false},
		{"yes", "yes", true},
		{"no", "no", false},
		{"TRUE", "TRUE", true},
		{"FALSE", "FALSE", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := mustEvaluate(t, tt.input, nil)
			boolean, ok := results[0].(*types.Boolean)
			if !ok {
				t.Fatalf("expected Boolean, got %T", results[0])
			}

			if boolean.Value != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, boolean.Value)
			}
		})
	}
}

func TestComparisons(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"greater than true", "5 > 3", true},
		{"greater than false", "3 > 5", false},
		{"less than true", "3 < 5", true},
		{"less than false", "5 < 3", false},
		{"greater equal true", "5 >= 5", true},
		{"greater equal false", "3 >= 5", false},
		{"less equal true", "5 <= 5", true},
		{"less equal false", "5 <= 3", false},
		{"equal true", "5 == 5", true},
		{"equal false", "5 == 3", false},
		{"not equal true", "5 != 3", true},
		{"not equal false", "5 != 5", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := mustEvaluate(t, tt.input, nil)
			boolean, ok := results[0].(*types.Boolean)
			if !ok {
				t.Fatalf("expected Boolean, got %T", results[0])
			}

			if boolean.Value != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, boolean.Value)
			}
		})
	}
}

func TestExponentiation(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"2^3", "2 ^ 3", "8"},
		{"2**3", "2 ** 3", "8"},
		{"3^2", "3 ^ 2", "9"},
		{"10^2", "10 ^ 2", "100"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := mustEvaluate(t, tt.input, nil)
			num := results[0].(*types.Number)

			expected, _ := decimal.NewFromString(tt.expected)
			if !num.Value.Equal(expected) {
				t.Errorf("expected %s, got %s", expected, num.Value)
			}
		})
	}
}

func TestModulus(t *testing.T) {
	results := mustEvaluate(t, "10 % 3", nil)
	num := results[0].(*types.Number)

	expected := decimal.NewFromInt(1)
	if !num.Value.Equal(expected) {
		t.Errorf("expected %s, got %s", expected, num.Value)
	}
}

func TestCurrencyAddition(t *testing.T) {
	results := mustEvaluate(t, "$100 + $50", nil)
	curr, ok := results[0].(*types.Currency)
	if !ok {
		t.Fatalf("expected Currency, got %T", results[0])
	}

	expected := decimal.NewFromInt(150)
	if !curr.Value.Equal(expected) {
		t.Errorf("expected %s, got %s", expected, curr.Value)
	}
}

func TestCurrencySubtraction(t *testing.T) {
	results := mustEvaluate(t, "$100 - $30", nil)
	curr, ok := results[0].(*types.Currency)
	if !ok {
		t.Fatalf("expected Currency, got %T", results[0])
	}

	expected := decimal.NewFromInt(70)
	if !curr.Value.Equal(expected) {
		t.Errorf("expected %s, got %s", expected, curr.Value)
	}
}

func TestNumberPlusCurrency(t *testing.T) {
	results := mustEvaluate(t, "50 + $100", nil)
	curr, ok := results[0].(*types.Currency)
	if !ok {
		t.Fatalf("expected Currency, got %T", results[0])
	}

	expected := decimal.NewFromInt(150)
	if !curr.Value.Equal(expected) {
		t.Errorf("expected %s, got %s", expected, curr.Value)
	}
}

func TestCurrencyPlusNumber(t *testing.T) {
	results := mustEvaluate(t, "$100 + 50", nil)
	curr, ok := results[0].(*types.Currency)
	if !ok {
		t.Fatalf("expected Currency, got %T", results[0])
	}

	expected := decimal.NewFromInt(150)
	if !curr.Value.Equal(expected) {
		t.Errorf("expected %s, got %s", expected, curr.Value)
	}
}

func TestNegativeNumbers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"negative number no space addition", "-50 + 100", "50"},
		{"negative number no space multiplication", "-10 * 5", "-50"},
		{"negative number no space subtraction", "-5 - 3", "-8"},
		{"double negative no space", "-10 - -5", "-5"},
		{"negative no space in expression", "100 + -50", "50"},
		{"negative number literal", "-42", "-42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := Evaluate(tt.input, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			num := results[0].(*types.Number)
			expected, _ := decimal.NewFromString(tt.expected)
			if !num.Value.Equal(expected) {
				t.Errorf("expected %s, got %s", expected, num.Value)
			}
		})
	}
}

func TestMarkdownBulletNotParsed(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"dash with space", "- 50"},
		{"asterisk with space", "* 50"},
		{"indented dash", "  - item"},
		{"indented asterisk", "    * item"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := Evaluate(tt.input, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Should return no results since it's markdown content
			if len(results) != 0 {
				t.Errorf("expected 0 results for markdown bullet %q, got %d", tt.input, len(results))
			}
		})
	}
}

func TestNegativeVsMarkdown(t *testing.T) {
	// Verify that negative numbers work but markdown bullets don't
	tests := []struct {
		name          string
		input         string
		expectResults int
		isMarkdown    bool
		expectedValue string
	}{
		{"negative at start no space", "-50", 1, false, "-50"},
		{"negative in expression", "100 + -50", 1, false, "50"},
		{"markdown bullet dash", "- 50", 0, true, ""},
		{"markdown bullet asterisk", "* 50", 0, true, ""},
		{"indented markdown", "  - item", 0, true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := Evaluate(tt.input, nil)
			if err != nil {
				if !tt.isMarkdown {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}

			if len(results) != tt.expectResults {
				t.Errorf("expected %d results, got %d", tt.expectResults, len(results))
			}

			if tt.expectResults > 0 && tt.expectedValue != "" {
				num := results[0].(*types.Number)
				if num.Value.String() != tt.expectedValue {
					t.Errorf("expected value %s, got %s", tt.expectedValue, num.Value.String())
				}
			}
		})
	}
}

// TestUndefinedVariableError tests that evaluating undefined variables returns errors
func TestUndefinedVariableError(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"single undefined variable", "x"},
		{"undefined in expression", "x + 2"},
		{"undefined in assignment RHS", "y = x + 2"},
		{"multiple undefined", "a + b + c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewContext()
			_, err := Evaluate(tt.input, ctx)
			if err == nil {
				t.Errorf("expected error for undefined variable in '%s', got none", tt.input)
			}
			// Check that error is EvaluationError
			if _, ok := err.(*EvaluationError); !ok {
				t.Errorf("expected EvaluationError, got %T", err)
			}
		})
	}
}

// TestUnicodeCalculations tests end-to-end calculations with Unicode identifiers
func TestUnicodeCalculations(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Japanese variable", "給料 = 5000\n給料", "5000"},
		{"Emoji variable", "💰 = 1000\n💰 * 2", "2000"},
		{"Mixed Unicode", "時給 = 25\n時間 = 40\n週給 = 時給 * 時間", "1000"},
		{"Emoji in expression", "💵 = 100\n💶 = 50\n💵 + 💶", "150"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewContext()
			results, err := Evaluate(tt.input, ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(results) == 0 {
				t.Fatal("expected at least one result")
			}

			lastResult := results[len(results)-1]
			if lastResult.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, lastResult.String())
			}
		})
	}
}

// TestUnaryPrecedence tests operator precedence with unary operators
func TestUnaryPrecedence(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"unary before multiply", "-5 * 3", "-15"},
		{"unary before divide", "-10 / 2", "-5"},
		{"unary before exponent", "-2 ^ 3", "-8"},
		{"multiply before unary RHS", "3 * -5", "-15"},
		{"unary with parentheses", "-(5 + 3)", "-8"},
		{"double unary", "--5", "5"},
		{"unary plus before multiply", "+5 * 2", "10"},
		{"parentheses precedence", "(2 + 3) * 4", "20"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewContext()
			results, err := Evaluate(tt.input, ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(results) != 1 {
				t.Fatalf("expected 1 result, got %d", len(results))
			}

			num := results[0].(*types.Number)
			if num.Value.String() != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, num.Value.String())
			}
		})
	}
}
