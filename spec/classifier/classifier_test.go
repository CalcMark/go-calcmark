package classifier

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/v2/constants"
	"github.com/CalcMark/go-calcmark/v2/impl/interpreter"
	"github.com/CalcMark/go-calcmark/v2/impl/types"
)

// Helper function to wrap ClassifyLine calls in tests
func classifyLineTest(t *testing.T, line string, env *interpreter.Environment) LineType {
	t.Helper()
	lineType, err := ClassifyLine(line, env)
	if err != nil {
		t.Fatalf("ClassifyLine(%q) error: %v", line, err)
	}
	return lineType
}

// TestBlankLines tests blank line classification
func TestEmptyString(t *testing.T) {
	if classifyLineTest(t, "", nil) != Blank {
		t.Error("expected BLANK")
	}
}

func TestWhitespaceOnly(t *testing.T) {
	tests := []string{"   ", "\t\t", "  \t  "}
	for _, test := range tests {
		if classifyLineTest(t, test, nil) != Blank {
			t.Errorf("expected BLANK for %q", test)
		}
	}
}

// TestMarkdownPrefixes tests markdown prefix detection
func TestHeader(t *testing.T) {
	tests := []string{"# Header", "## Subheader"}
	for _, test := range tests {
		if classifyLineTest(t, test, nil) != Markdown {
			t.Errorf("expected MARKDOWN for %q", test)
		}
	}
}

func TestQuote(t *testing.T) {
	tests := []string{
		"> Quote",
		"> This should be markdown",
		">Blockquote",
	}
	for _, test := range tests {
		if classifyLineTest(t, test, nil) != Markdown {
			t.Errorf("expected MARKDOWN for %q", test)
		}
	}
}

func TestList(t *testing.T) {
	tests := []string{"- List item", "* Bullet"}
	for _, test := range tests {
		if classifyLineTest(t, test, nil) != Markdown {
			t.Errorf("expected MARKDOWN for %q", test)
		}
	}
}

func TestNumberedList(t *testing.T) {
	tests := []string{"1. First", "2. Second"}
	for _, test := range tests {
		if classifyLineTest(t, test, nil) != Markdown {
			t.Errorf("expected MARKDOWN for %q", test)
		}
	}
}

// TestLiterals tests literal classification
func TestNumberLiteral(t *testing.T) {
	tests := []string{"42", "3.14", "1,000"}
	for _, test := range tests {
		if classifyLineTest(t, test, nil) != Calculation {
			t.Errorf("expected CALCULATION for %q", test)
		}
	}
}

func TestCurrencyLiteral(t *testing.T) {
	// Standalone currency literals are not calculations - they're just data
	// They need to be in expressions or assignments to be calculations
	tests := []string{"$100", "$1,000.50"}
	for _, test := range tests {
		if classifyLineTest(t, test, nil) != Markdown {
			t.Errorf("expected MARKDOWN for %q (standalone currency is not a calculation)", test)
		}
	}
}

func TestBooleanLiteral(t *testing.T) {
	tests := []string{"true", "false"}
	for _, test := range tests {
		if classifyLineTest(t, test, nil) != Calculation {
			t.Errorf("expected CALCULATION for %q", test)
		}
	}
}

// TestAssignments tests assignment classification
func TestSimpleAssignment(t *testing.T) {
	tests := []string{"x = 5", "salary = $50000"}
	for _, test := range tests {
		if classifyLineTest(t, test, nil) != Calculation {
			t.Errorf("expected CALCULATION for %q", test)
		}
	}
}

func TestUnicodeAssignment(t *testing.T) {
	tests := []string{"💰 = $1000", "給料 = $5000"}
	for _, test := range tests {
		if classifyLineTest(t, test, nil) != Calculation {
			t.Errorf("expected CALCULATION for %q", test)
		}
	}
}

func TestAssignmentWithUnderscores(t *testing.T) {
	// BREAKING CHANGE: Spaces no longer allowed in identifiers, use underscores
	tests := []string{"my_budget = 1000", "weeks_in_year = 52"}
	for _, test := range tests {
		if classifyLineTest(t, test, nil) != Calculation {
			t.Errorf("expected CALCULATION for %q", test)
		}
	}
}

func TestMalformedAssignment(t *testing.T) {
	tests := []string{"x =", "= 5"}
	for _, test := range tests {
		if classifyLineTest(t, test, nil) != Markdown {
			t.Errorf("expected MARKDOWN for %q", test)
		}
	}
}

// TestArithmeticExpressions tests arithmetic expression classification
func TestSimpleArithmetic(t *testing.T) {
	tests := []string{"3 + 5", "10 - 3", "4 * 5", "20 / 4"}
	for _, test := range tests {
		if classifyLineTest(t, test, nil) != Calculation {
			t.Errorf("expected CALCULATION for %q", test)
		}
	}
}

func TestNewOperators(t *testing.T) {
	tests := []string{"2 ^ 3", "2 ** 3", "10 % 3"}
	for _, test := range tests {
		if classifyLineTest(t, test, nil) != Calculation {
			t.Errorf("expected CALCULATION for %q", test)
		}
	}
}

func TestCurrencyArithmetic(t *testing.T) {
	tests := []string{"$100 * 52", "$1000 + $500"}
	for _, test := range tests {
		if classifyLineTest(t, test, nil) != Calculation {
			t.Errorf("expected CALCULATION for %q", test)
		}
	}
}

// TestComparisonExpressions tests comparison expression classification
func TestComparisons(t *testing.T) {
	tests := []string{
		"1 > 0",
		"5 < 10",
		"5 >= 5",
		"3 <= 10",
		"5 == 5",
		"5 != 3",
	}
	for _, test := range tests {
		if classifyLineTest(t, test, nil) != Calculation {
			t.Errorf("expected CALCULATION for %q", test)
		}
	}
}

// TestContextAwareness tests context-aware classification
func TestKnownVariableReference(t *testing.T) {
	ctx := interpreter.NewEnvironment()
	num, _ := types.NewNumber(5)
	ctx.Set("x", num)

	if classifyLineTest(t, "x", ctx) != Calculation {
		t.Error("expected CALCULATION for 'x'")
	}
	if classifyLineTest(t, "x * 2", ctx) != Calculation {
		t.Error("expected CALCULATION for 'x * 2'")
	}
}

func TestUnknownVariableReference(t *testing.T) {
	ctx := interpreter.NewEnvironment()

	if classifyLineTest(t, "unknown_var", ctx) != Markdown {
		t.Error("expected MARKDOWN for 'unknown_var'")
	}
	if classifyLineTest(t, "emoji * 2", ctx) != Markdown {
		t.Error("expected MARKDOWN for 'emoji * 2'")
	}
}

func TestMixedKnownUnknown(t *testing.T) {
	ctx := interpreter.NewEnvironment()
	num, _ := types.NewNumber(5)
	ctx.Set("x", num)

	if classifyLineTest(t, "x + unknown", ctx) != Markdown {
		t.Error("expected MARKDOWN for 'x + unknown'")
	}
	if classifyLineTest(t, "unknown + x", ctx) != Markdown {
		t.Error("expected MARKDOWN for 'unknown + x'")
	}
}

func TestBooleanKeywordsAlwaysKnown(t *testing.T) {
	ctx := interpreter.NewEnvironment()

	// 'true' and 'false' are boolean keywords, so they're always available
	if classifyLineTest(t, "true", ctx) != Calculation {
		t.Error("expected CALCULATION for 'true'")
	}
	if classifyLineTest(t, "false", ctx) != Calculation {
		t.Error("expected CALCULATION for 'false'")
	}
}

// TestEdgeCases tests edge cases and special scenarios
func TestTrailingText(t *testing.T) {
	tests := []string{"$100 budget", "5 + 3 equals eight"}
	for _, test := range tests {
		if classifyLineTest(t, test, nil) != Markdown {
			t.Errorf("expected MARKDOWN for %q", test)
		}
	}
}

func TestIncompleteExpressions(t *testing.T) {
	// Note: "+ 5" and "- 5" are now valid (unary operators)
	// Only truly incomplete expressions should be markdown
	tests := []string{"x *", "5 +", "5 -"}
	for _, test := range tests {
		if classifyLineTest(t, test, nil) != Markdown {
			t.Errorf("expected MARKDOWN for %q", test)
		}
	}
}

func TestNaturalLanguage(t *testing.T) {
	tests := []string{
		"This is a sentence",
		"Let's calculate something",
		"The answer is 42",
	}
	for _, test := range tests {
		if classifyLineTest(t, test, nil) != Markdown {
			t.Errorf("expected MARKDOWN for %q", test)
		}
	}
}

func TestURLs(t *testing.T) {
	tests := []string{
		"https://example.com",
		"http://test.org?foo=bar",
	}
	for _, test := range tests {
		if classifyLineTest(t, test, nil) != Markdown {
			t.Errorf("expected MARKDOWN for %q", test)
		}
	}
}

func TestSpecialCharacters(t *testing.T) {
	if classifyLineTest(t, "@#$%^&*()", nil) != Markdown {
		t.Error("expected MARKDOWN")
	}
}

// TestDirectiveReferences tests classification of @scale and @globals directive references
func TestDirectiveReferences(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected LineType
	}{
		// Standalone directives are calculations
		{"standalone @scale", "@scale", Calculation},
		{"standalone @globals.tax_rate", "@globals.tax_rate", Calculation},

		// Directives in assignment expressions
		{"assignment with @scale", "a = @scale", Calculation},
		{"assignment with @scale arithmetic", "a = @scale * 3", Calculation},
		{"assignment with @globals", "a = @globals.tax_rate * 100", Calculation},

		// Directives with operators (no assignment)
		{"@scale in operator expression", "@scale + 1", Calculation},
		{"@scale * literal", "@scale * 2", Calculation},

		// Directives with units
		{"@scale with unit", "@scale meters", Calculation},

		// Directives with rate units
		{"@scale with rate", "@scale meters / second", Calculation},

		// Self-referential: @scale used twice in one expression
		{"@scale ^ @scale", "@scale ^ @scale", Calculation},

		// Complex expression with parentheses
		{"(@scale + 1) * @scale", "(@scale + 1) * @scale", Calculation},

		// @globals with unit annotation
		{"@globals.rate with unit", "@globals.rate kg", Calculation},

		// Invalid directive syntax stays markdown
		{"bare @ with garbage", "@#$%^&*()", Markdown},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := classifyLineTest(t, tc.line, nil)
			if result != tc.expected {
				t.Errorf("ClassifyLine(%q) = %s, want %s", tc.line, result, tc.expected)
			}
		})
	}
}

// TestDocumentExample tests classification of a full document
func TestBudgetDocument(t *testing.T) {
	document := `# My Monthly Budget

Income:
salary = $5000
bonus = $500

Expenses:
rent = $1500
food = $800
utilities = $200

Total expenses:
expenses = rent + food + utilities

Savings:
savings = salary + bonus - expenses`

	expected := []LineType{
		Markdown,    // # My Monthly Budget
		Blank,       // (empty)
		Markdown,    // Income:
		Calculation, // salary = $5000
		Calculation, // bonus = $500
		Blank,       // (empty)
		Markdown,    // Expenses:
		Calculation, // rent = $1500
		Calculation, // food = $800
		Calculation, // utilities = $200
		Blank,       // (empty)
		Markdown,    // Total expenses:
		Calculation, // expenses = rent + food + utilities
		Blank,       // (empty)
		Markdown,    // Savings:
		Calculation, // savings = salary + bonus - expenses
	}

	ctx := interpreter.NewEnvironment()
	lines := strings.Split(document, constants.Newline)

	for i, line := range lines {
		result := classifyLineTest(t, line, ctx)
		if result != expected[i] {
			t.Errorf("Line %d (%q): expected %s, got %s", i+1, line, expected[i], result)
		}

		// Evaluate calculations to update context
		if result == Calculation {
			interpreter.Evaluate(line, ctx)
		}
	}
}

// TestDateExpressionClassification verifies date expressions classify as calculations.
func TestDateExpressionClassification(t *testing.T) {
	tests := []struct {
		name string
		line string
		want LineType
	}{
		{"today", "d = today", Calculation},
		{"next Friday", "d = next Friday", Calculation},
		{"this quarter", "d = this quarter", Calculation},
		{"2 weeks ago", "d = 2 weeks ago", Calculation},
		{"end of this month", "d = end of this month", Calculation},
		{"this fiscal quarter", "d = this fiscal quarter", Calculation},
		{"next April", "d = next April", Calculation},
		{"last year", "d = last year", Calculation},
		{"next Monday + 2 weeks", "d = next Monday + 2 weeks", Calculation},
		{"3 days from next Friday", "d = 3 days from next Friday", Calculation},
		// Bare weekday without assignment — should still be calculation
		{"bare next Friday", "next Friday", Calculation},
		// Markdown should still be markdown
		{"heading", "# Next Friday Meeting", Markdown},
		{"prose", "The meeting is next Friday", Markdown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyLineTest(t, tt.line, nil)
			if got != tt.want {
				t.Errorf("ClassifyLine(%q) = %s, want %s", tt.line, got, tt.want)
			}
		})
	}
}
