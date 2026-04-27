package lsp

import (
	"fmt"
	"strings"
)

// buildNLExampleSnippet wraps each numeric-literal token in an NL alias
// example with LSP snippet syntax `${N:token}`. Tokens are detected with
// the same boundary rules as `findNumericLiterals` — a token is a digit
// run with optional leading `$`/`€` currency prefix and optional
// trailing `%`. The Nth token (1-indexed) becomes the Nth tab stop so
// the user can Tab through every value position.
//
// Example:
//
//	buildNLExampleSnippet("compound $1000 by 5% over 10 years")
//	→ "compound ${1:$1000} by ${2:5%} over ${3:10} years"
//
// The full token (including currency prefix and percent suffix) lives
// inside the placeholder so a type-compatible variable can replace the
// whole token without leaving stray syntax behind. This is the LSP-side
// counterpart to `nlExampleToSnippet` that previously lived in the
// calcmark-web client; centralizing it here keeps the language fact
// (where literals can appear in NL forms) in one place.
//
// If the example has no numeric tokens, returns the input unchanged
// (still safe to ship as `InsertTextFormat: Snippet`).
func buildNLExampleSnippet(example string) string {
	runes := []rune(example)
	literals := findNumericLiterals(runes, 0)
	if len(literals) == 0 {
		return example
	}

	var b strings.Builder
	b.Grow(len(example) + 6*len(literals))
	cursor := 0
	for i, lit := range literals {
		// Emit the gap before this literal verbatim.
		if lit.start > cursor {
			b.WriteString(string(runes[cursor:lit.start]))
		}
		token := string(runes[lit.start:lit.end])
		fmt.Fprintf(&b, "${%d:%s}", i+1, token)
		cursor = lit.end
	}
	// Emit any trailing tail after the last literal.
	if cursor < len(runes) {
		b.WriteString(string(runes[cursor:]))
	}
	return b.String()
}
