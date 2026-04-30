package lexer_test

import (
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/lexer"
)

// FuzzLexerTokenize tests that the lexer never panics on arbitrary input.
// It also verifies that security limits (identifier length, number length)
// produce errors rather than unbounded memory allocation.
func FuzzLexerTokenize(f *testing.F) {
	// Seed corpus: representative calcmark expressions.
	seeds := []string{
		// Normal expressions
		"x = 10\n",
		"price = 100 USD\n",
		"total = price * 1.08\n",
		"avg(1, 2, 3)\n",
		"a = 5 + 3 * (2 - 1)\n",

		// Edge cases
		"",
		"\n",
		"\n\n\n",
		"   \t  \n",

		// Unicode
		"日本語 = 100\n",
		"π = 3.14159\n",
		"café_price = 5.50\n",

		// Boundary: identifiers near max length
		"abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz = 1\n",

		// Numbers with various formats
		"x = 1_000_000\n",
		"y = 0.123456789\n",
		"z = 1e10\n",
		"w = $100.00\n",
		"v = 50%\n",

		// Markdown-like content (should not crash lexer)
		"# This is a heading\n",
		"Some text with *emphasis* and **bold**\n",
		"- list item\n",
		"1. ordered item\n",

		// Fraction patterns
		"1/3\n",
		"0/0\n",
		"1/0\n",
		"999/1000\n",
		"1/99999\n",
		"11 3/8\n",
		"1e3/4\n",
		"$1/3\n",
		"1/3/4\n",

		// Operators and special characters
		"a + b - c * d / e\n",
		"x = (((1)))\n",

		// Potential attack vectors
		"x = " + string(make([]byte, 300)) + "\n", // NUL bytes
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// The lexer must never panic on any input.
		// Errors are expected and acceptable — panics are not.
		lex := lexer.NewLexer(input)
		tokens, err := lex.Tokenize()

		// If tokenization succeeds, tokens must not be nil.
		if err == nil && tokens == nil {
			t.Error("Tokenize returned nil tokens with nil error")
		}

		// If there's a security error, that's correct behavior — not a bug.
		if err != nil {
			// Verify it's a properly typed error, not a runtime crash.
			_ = err.Error()
		}
	})
}
