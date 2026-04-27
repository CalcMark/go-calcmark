package lsp

import (
	"slices"
	"testing"
)

func TestComputeLexicon_FunctionsIncludeRegistryNames(t *testing.T) {
	lex := computeLexicon()
	for _, name := range []string{"grow", "compound", "depreciate", "throughput", "accumulate", "avg", "sum", "sqrt", "number"} {
		if !slices.Contains(lex.Functions, name) {
			t.Errorf("expected %q in functions, got %v", name, lex.Functions)
		}
	}
}

func TestComputeLexicon_FunctionsIncludeSynonyms(t *testing.T) {
	// avg's synonyms are `average` and `mean` per the registry.
	lex := computeLexicon()
	for _, syn := range []string{"average", "mean"} {
		if !slices.Contains(lex.Functions, syn) {
			t.Errorf("expected synonym %q in functions, got %v", syn, lex.Functions)
		}
	}
}

func TestComputeLexicon_FunctionsAreSorted(t *testing.T) {
	lex := computeLexicon()
	for i := 1; i < len(lex.Functions); i++ {
		if lex.Functions[i-1] > lex.Functions[i] {
			t.Errorf("functions not sorted: %q > %q", lex.Functions[i-1], lex.Functions[i])
		}
	}
}

func TestComputeLexicon_ConversionKeywordsCoverDocumentedExamples(t *testing.T) {
	lex := computeLexicon()
	// The user-flagged examples in the styling request: `as napkin`,
	// `over`, `by`. These must appear so the calc-source tokenizer
	// can color them.
	for _, kw := range []string{"as", "napkin", "precise", "over", "by", "in"} {
		if !slices.Contains(lex.ConversionKeywords, kw) {
			t.Errorf("expected %q in conversionKeywords, got %v", kw, lex.ConversionKeywords)
		}
	}
}

func TestComputeLexicon_ExcludesControlFlowKeywords(t *testing.T) {
	// Reserved control-flow words (`if`, `for`, `while`, ...) are not
	// meaningful in calcmark expressions yet — they should NOT be
	// color-coded as language vocabulary because the user doesn't
	// type them.
	lex := computeLexicon()
	for _, kw := range []string{"if", "else", "for", "while", "then"} {
		if slices.Contains(lex.ConversionKeywords, kw) {
			t.Errorf("did not expect control-flow %q in conversionKeywords", kw)
		}
	}
}
