package parser_test

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/lexer"
)

// TestNaturalLanguageTokenization checks how "average of" is tokenized
func TestNaturalLanguageTokenization(t *testing.T) {
	input := "average of 1, 2, 3\n"

	lex := lexer.NewLexer(input)
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatalf("Tokenization failed: %v", err)
	}

	t.Logf("Input: %q", input)
	t.Logf("Tokens (%d total):", len(tokens))
	for i, tok := range tokens {
		t.Logf("  [%d] Type=%-20v Value=%q", i, tok.Type, tok.Value)
	}
}
