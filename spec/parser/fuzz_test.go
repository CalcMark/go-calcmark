package parser_test

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/parser"
)

// FuzzParserParse tests that the parser never panics on arbitrary input.
// It exercises the full pipeline: lexer → token limit check → recursive
// descent parse → AST. Security limits (nesting depth, token count)
// must produce errors rather than stack overflow or OOM.
func FuzzParserParse(f *testing.F) {
	// Seed corpus: representative calcmark expressions.
	seeds := []string{
		// Normal expressions
		"x = 10\n",
		"price = 100 USD\n",
		"total = price * 1.08\n",
		"result = avg(1, 2, 3)\n",
		"a = 5 + 3 * (2 - 1)\n",
		"tax = subtotal * 8.5%\n",

		// Multi-line documents
		"x = 10\ny = 20\ntotal = x + y\n",

		// Edge cases
		"",
		"\n",
		"\n\n\n",
		"   \n",

		// Unicode identifiers
		"café_price = 5.50\n",
		"日本語 = 100\n",

		// Functions with multiple args
		"f = min(1, max(2, 3))\n",

		// Nested expressions (within limit)
		"x = ((((1 + 2))))\n",

		// Potential attack: deep nesting (should produce security error)
		"x = (((((((((((((((((((((((((((((((((((((((((((((((((1)))))))))))))))))))))))))))))))))))))))))))))))))))))\n",

		// Potential attack: many tokens
		"x = 1+1+1+1+1+1+1+1+1+1+1+1+1+1+1+1+1+1+1+1\n",

		// Date/time expressions
		"deadline = Jan 15, 2025\n",
		"duration = 3 days\n",

		// Unit expressions
		"distance = 100 km\n",
		"weight = 5 kg to lb\n",

		// Comments and text
		"// This is a comment\n",
		"# Heading\n",

		// Percentage and rate expressions
		"growth = 15%\n",
		"speed = 60 km/hr\n",

		// Assignment with complex RHS
		"total = (price + shipping) * (1 + tax_rate)\n",

		// Malformed input (should error, not panic)
		"= = =\n",
		"(((\n",
		")))\n",
		"+++\n",
		"x = \n",
		"= 10\n",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// The parser must never panic on any input.
		// Errors are expected and acceptable — panics are not.
		nodes, err := parser.Parse(input)

		// If parsing succeeds, exercise the AST to check for latent panics.
		// Note: empty input legitimately produces (nil, nil).
		if err == nil {
			for _, n := range nodes {
				_ = n.String()
			}
		}

		// If there's a security error, that's correct behavior — not a bug.
		if err != nil {
			_ = err.Error()
		}
	})
}
