package classifier

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/evaluator"
	"github.com/CalcMark/go-calcmark/types"
)

// TestBlankLines tests blank line classification
func TestEmptyString(t *testing.T) {
	if ClassifyLine("", nil) != Blank {
		t.Error("expected BLANK")
	}
}

func TestWhitespaceOnly(t *testing.T) {
	tests := []string{"   ", "\t\t", "  \t  "}
	for _, test := range tests {
		if ClassifyLine(test, nil) != Blank {
			t.Errorf("expected BLANK for %q", test)
		}
	}
}

// TestMarkdownPrefixes tests markdown prefix detection
func TestHeader(t *testing.T) {
	tests := []string{"# Header", "## Subheader"}
	for _, test := range tests {
		if ClassifyLine(test, nil) != Markdown {
			t.Errorf("expected MARKDOWN for %q", test)
		}
	}
}

func TestQuote(t *testing.T) {
	if ClassifyLine("> Quote", nil) != Markdown {
		t.Error("expected MARKDOWN")
	}
}

func TestList(t *testing.T) {
	tests := []string{"- List item", "* Bullet"}
	for _, test := range tests {
		if ClassifyLine(test, nil) != Markdown {
			t.Errorf("expected MARKDOWN for %q", test)
		}
	}
}

func TestNumberedList(t *testing.T) {
	tests := []string{"1. First", "2. Second"}
	for _, test := range tests {
		if ClassifyLine(test, nil) != Markdown {
			t.Errorf("expected MARKDOWN for %q", test)
		}
	}
}

// TestLiterals tests literal classification
func TestNumberLiteral(t *testing.T) {
	tests := []string{"42", "3.14", "1,000"}
	for _, test := range tests {
		if ClassifyLine(test, nil) != Calculation {
			t.Errorf("expected CALCULATION for %q", test)
		}
	}
}

func TestCurrencyLiteral(t *testing.T) {
	tests := []string{"$100", "$1,000.50"}
	for _, test := range tests {
		if ClassifyLine(test, nil) != Calculation {
			t.Errorf("expected CALCULATION for %q", test)
		}
	}
}

func TestBooleanLiteral(t *testing.T) {
	tests := []string{"true", "false", "yes", "no", "t", "f", "y", "n"}
	for _, test := range tests {
		if ClassifyLine(test, nil) != Calculation {
			t.Errorf("expected CALCULATION for %q", test)
		}
	}
}

// TestAssignments tests assignment classification
func TestSimpleAssignment(t *testing.T) {
	tests := []string{"x = 5", "salary = $50000"}
	for _, test := range tests {
		if ClassifyLine(test, nil) != Calculation {
			t.Errorf("expected CALCULATION for %q", test)
		}
	}
}

func TestUnicodeAssignment(t *testing.T) {
	tests := []string{"💰 = $1000", "給料 = $5000"}
	for _, test := range tests {
		if ClassifyLine(test, nil) != Calculation {
			t.Errorf("expected CALCULATION for %q", test)
		}
	}
}

func TestAssignmentWithSpaces(t *testing.T) {
	tests := []string{"my budget = 1000", "weeks in year = 52"}
	for _, test := range tests {
		if ClassifyLine(test, nil) != Calculation {
			t.Errorf("expected CALCULATION for %q", test)
		}
	}
}

func TestMalformedAssignment(t *testing.T) {
	tests := []string{"x =", "= 5"}
	for _, test := range tests {
		if ClassifyLine(test, nil) != Markdown {
			t.Errorf("expected MARKDOWN for %q", test)
		}
	}
}

// TestArithmeticExpressions tests arithmetic expression classification
func TestSimpleArithmetic(t *testing.T) {
	tests := []string{"3 + 5", "10 - 3", "4 * 5", "20 / 4"}
	for _, test := range tests {
		if ClassifyLine(test, nil) != Calculation {
			t.Errorf("expected CALCULATION for %q", test)
		}
	}
}

func TestNewOperators(t *testing.T) {
	tests := []string{"2 ^ 3", "2 ** 3", "10 % 3"}
	for _, test := range tests {
		if ClassifyLine(test, nil) != Calculation {
			t.Errorf("expected CALCULATION for %q", test)
		}
	}
}

func TestCurrencyArithmetic(t *testing.T) {
	tests := []string{"$100 * 52", "$1000 + $500"}
	for _, test := range tests {
		if ClassifyLine(test, nil) != Calculation {
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
		if ClassifyLine(test, nil) != Calculation {
			t.Errorf("expected CALCULATION for %q", test)
		}
	}
}

// TestContextAwareness tests context-aware classification
func TestKnownVariableReference(t *testing.T) {
	ctx := evaluator.NewContext()
	num, _ := types.NewNumber(5)
	ctx.Set("x", num)

	if ClassifyLine("x", ctx) != Calculation {
		t.Error("expected CALCULATION for 'x'")
	}
	if ClassifyLine("x * 2", ctx) != Calculation {
		t.Error("expected CALCULATION for 'x * 2'")
	}
}

func TestUnknownVariableReference(t *testing.T) {
	ctx := evaluator.NewContext()

	if ClassifyLine("unknown_var", ctx) != Markdown {
		t.Error("expected MARKDOWN for 'unknown_var'")
	}
	if ClassifyLine("emoji * 2", ctx) != Markdown {
		t.Error("expected MARKDOWN for 'emoji * 2'")
	}
}

func TestMixedKnownUnknown(t *testing.T) {
	ctx := evaluator.NewContext()
	num, _ := types.NewNumber(5)
	ctx.Set("x", num)

	if ClassifyLine("x + unknown", ctx) != Markdown {
		t.Error("expected MARKDOWN for 'x + unknown'")
	}
	if ClassifyLine("unknown + x", ctx) != Markdown {
		t.Error("expected MARKDOWN for 'unknown + x'")
	}
}

func TestBooleanKeywordsAlwaysKnown(t *testing.T) {
	ctx := evaluator.NewContext()

	// 'y' is a boolean keyword, so it's always available
	if ClassifyLine("y", ctx) != Calculation {
		t.Error("expected CALCULATION for 'y'")
	}
	if ClassifyLine("true", ctx) != Calculation {
		t.Error("expected CALCULATION for 'true'")
	}
}

// TestEdgeCases tests edge cases and special scenarios
func TestTrailingText(t *testing.T) {
	tests := []string{"$100 budget", "5 + 3 equals eight"}
	for _, test := range tests {
		if ClassifyLine(test, nil) != Markdown {
			t.Errorf("expected MARKDOWN for %q", test)
		}
	}
}

func TestIncompleteExpressions(t *testing.T) {
	tests := []string{"x *", "+ 5", "5 +"}
	for _, test := range tests {
		if ClassifyLine(test, nil) != Markdown {
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
		if ClassifyLine(test, nil) != Markdown {
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
		if ClassifyLine(test, nil) != Markdown {
			t.Errorf("expected MARKDOWN for %q", test)
		}
	}
}

func TestSpecialCharacters(t *testing.T) {
	if ClassifyLine("@#$%^&*()", nil) != Markdown {
		t.Error("expected MARKDOWN")
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

	ctx := evaluator.NewContext()
	lines := strings.Split(document, "\n")

	for i, line := range lines {
		result := ClassifyLine(line, ctx)
		if result != expected[i] {
			t.Errorf("Line %d (%q): expected %s, got %s", i+1, line, expected[i], result)
		}

		// Evaluate calculations to update context
		if result == Calculation {
			evaluator.Evaluate(line, ctx)
		}
	}
}
