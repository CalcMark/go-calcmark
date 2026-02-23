package document_test

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
)

// FuzzNewDocument tests that the document constructor never panics on
// arbitrary input. This exercises the full pipeline: frontmatter parsing,
// block detection, lexer, parser, and dependency graph construction.
// This is the most important fuzz target because NewDocument is called
// on every file read and every TUI document rebuild.
func FuzzNewDocument(f *testing.F) {
	// Seed corpus: representative calcmark documents.
	seeds := []string{
		// Simple documents
		"x = 10\n",
		"x = 10\ny = 20\ntotal = x + y\n",
		"# Heading\n\nSome text\n\nx = 42\n",

		// Empty / whitespace
		"",
		"\n",
		"\n\n\n",
		"   \t  \n",

		// Frontmatter
		"---\nexchange:\n  USD_EUR: 0.85\n---\nprice = 100 USD\nresult = price in EUR\n",
		"---\nglobals:\n  tax: 0.08\n---\ntotal = 100 * (1 + tax)\n",

		// Malformed frontmatter (should error, not panic)
		"---\n---\n",
		"---\nbad: yaml: here\n---\n",
		"---\nunknown_key: value\n---\n",
		"---\nexchange:\n  INVALID: 0.85\n---\n",
		"---\nexchange:\n  USD_EUR: -1\n---\n",
		"---\nexchange:\n  USD_EUR: 0\n---\n",

		// Block boundaries (double blank lines)
		"a = 1\n\n\nb = 2\n",
		"a = 1\n\nb = 2\n",
		"# Header\n\ntext\n\n\ncalc = 42\n",

		// Mixed calc and text blocks
		"# Budget\n\nincome = 5000\ntax = income * 0.3\n\n\n# Summary\n\nnet = income - tax\n",

		// Unicode content
		"日本語 = 100\n",
		"café = 5.50 EUR\n",

		// Ordered lists mixed with calc
		"1. First item\n2. Second item\n",
		"x = 10\n\n1. Item\n2. Item\n\ny = 20\n",

		// Potential attack: very long line
		strings.Repeat("x", 500) + " = 1\n",

		// Potential attack: many lines
		strings.Repeat("x = 1\n", 200),

		// Potential attack: deeply nested expressions in a document
		"result = " + strings.Repeat("(", 50) + "1" + strings.Repeat(")", 50) + "\n",

		// Potential attack: many blocks
		strings.Repeat("a = 1\n\n\n", 50),

		// Edge: frontmatter without closing delimiter
		"---\nexchange:\n  USD_EUR: 0.85\n",

		// Edge: only frontmatter, no content
		"---\nglobals:\n  x: 10\n---\n",

		// Edge: content that looks like frontmatter delimiter
		"---\n",
		"--- text\n",

		// Malformed expressions (should error gracefully)
		"= = =\n",
		"(((\n",
		")))\n",
		"x = \n",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// NewDocument must never panic on any input.
		// Errors are expected and acceptable — panics are not.
		doc, err := document.NewDocument(input)

		if err != nil {
			// Verify the error is properly formatted.
			_ = err.Error()
			return
		}

		// If construction succeeds, the document must be usable.
		if doc == nil {
			t.Fatal("NewDocument returned nil doc with nil error")
		}

		// Basic invariant: blocks list must not be nil.
		blocks := doc.GetBlocks()
		if blocks == nil {
			t.Fatal("GetBlocks returned nil on valid document")
		}

		// Exercise block accessors to check for latent panics.
		for _, bn := range blocks {
			_ = bn.ID
			if bn.Block != nil {
				_ = bn.Block.Source()
			}
		}
	})
}
