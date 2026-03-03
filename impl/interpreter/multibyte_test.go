package interpreter_test

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/parser"
)

// TestMultibyteVariableAssignment validates that multi-byte UTF-8 identifiers
// can be assigned values and used in subsequent expressions.
func TestMultibyteVariableAssignment(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// CJK variable names
		{
			name:     "CJK single char assignment",
			input:    "手 = 5\n",
			expected: "5",
		},
		{
			name:     "CJK multi-char assignment",
			input:    "給料 = 5000\n",
			expected: "5000",
		},
		{
			name:     "CJK variable in expression",
			input:    "a = 3\n手 = a * 5\n",
			expected: "15",
		},

		// Latin extended
		{
			name:     "Latin with accent",
			input:    "café = 100\n",
			expected: "100",
		},
		{
			name:     "Latin extended in expression",
			input:    "café = 100\nrésumé = café + 50\n",
			expected: "150",
		},

		// Cyrillic
		{
			name:     "Cyrillic assignment",
			input:    "Москва = 200\n",
			expected: "200",
		},
		{
			name:     "Cyrillic in expression",
			input:    "Москва = 200\nдоход = Москва * 3\n",
			expected: "600",
		},

		// Emoji (supported ranges)
		{
			name:     "Emoji money bag assignment",
			input:    "💰 = 1000\n",
			expected: "1000",
		},
		{
			name:     "Emoji in expression",
			input:    "💰 = 1000\n🎯 = 💰 + 500\n",
			expected: "1500",
		},
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

			// Check the last result (for multi-line inputs)
			actual := results[len(results)-1].String()
			if !strings.HasPrefix(actual, tt.expected) {
				t.Errorf("Result = %q, expected to start with %q", actual, tt.expected)
			}
		})
	}
}

// TestMultibyteCrossScriptReferences validates that variables assigned with
// one script can be referenced from expressions using a different script.
func TestMultibyteCrossScriptReferences(t *testing.T) {
	input := "a = 10\n手 = 20\ncafé = 30\nМосква = 40\n💰 = 50\ntotal = a + 手 + café + Москва + 💰\n"

	nodes, err := parser.Parse(input)
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

	// total = 10 + 20 + 30 + 40 + 50 = 150
	last := results[len(results)-1].String()
	if !strings.HasPrefix(last, "150") {
		t.Errorf("Cross-script total = %q, expected 150", last)
	}
}

// TestMultibyteWithFunctions validates that multi-byte variables work with
// built-in functions like avg() and sqrt().
func TestMultibyteWithFunctions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "avg with CJK variables",
			input:    "甲 = 10\n乙 = 20\n丙 = 30\navg(甲, 乙, 丙)\n",
			expected: "20",
		},
		{
			name:     "sqrt with Latin extended",
			input:    "carré = 16\nsqrt(carré)\n",
			expected: "4",
		},
		{
			name:     "avg with emoji variables",
			input:    "💰 = 100\n🎯 = 200\n📊 = 300\navg(💰, 🎯, 📊)\n",
			expected: "200",
		},
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

			last := results[len(results)-1].String()
			if !strings.HasPrefix(last, tt.expected) {
				t.Errorf("Result = %q, expected to start with %q", last, tt.expected)
			}
		})
	}
}

// TestMultibyteWithMultipliers validates that multi-byte identifiers work
// correctly with K/M/B/T number multipliers.
func TestMultibyteWithMultipliers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "CJK with K multiplier",
			input:    "價格 = 5K\n",
			expected: "5000",
		},
		{
			name:     "Cyrillic with M multiplier",
			input:    "бюджет = 2M\n",
			expected: "2000000",
		},
		{
			name:     "Emoji with multiplier expression",
			input:    "💰 = 1K + 500\n",
			expected: "1500",
		},
		{
			name:     "CJK variable with multiplier in reference",
			input:    "a = 3\n手 = a * 5\ntest = 手 * 1K\n",
			expected: "15000",
		},
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

			last := results[len(results)-1].String()
			if !strings.HasPrefix(last, tt.expected) {
				t.Errorf("Result = %q, expected to start with %q", last, tt.expected)
			}
		})
	}
}

// TestMultibyteWithArbitraryUnits validates that multi-byte identifiers work
// as both variable names and alongside arbitrary unit names.
func TestMultibyteWithArbitraryUnits(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "CJK variable with arbitrary units",
			input:    "個数 = 10 apples + 5 apples\n",
			expected: "15 apples",
		},
		{
			name:     "Cyrillic variable with units",
			input:    "количество = 20 items * 3\n",
			expected: "60 items",
		},
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

			last := results[len(results)-1].String()
			if !strings.HasPrefix(last, tt.expected) {
				t.Errorf("Result = %q, expected to start with %q", last, tt.expected)
			}
		})
	}
}
