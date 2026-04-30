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

func TestComputeLexicon_UnitCategoriesMirrorUnitsCategories(t *testing.T) {
	// The TS-side frontmatter form uses these to populate the
	// scale.unit_categories / convert_to.unit_categories chip list.
	// The full set must include not just the standard quantities
	// (Length, Mass, ...) but also the synthetic categories:
	// Currency, Number, Custom, DataSize. Missing any one of these
	// means the form silently filters out a category the server
	// would otherwise accept — the regression that motivated landing
	// this field in the lexicon in the first place.
	lex := computeLexicon()
	for _, cat := range []string{"Length", "Mass", "Volume", "Time", "DataSize", "Currency", "Number", "Custom"} {
		if !slices.Contains(lex.UnitCategories, cat) {
			t.Errorf("expected %q in UnitCategories, got %v", cat, lex.UnitCategories)
		}
	}
}

func TestComputeLexicon_UnitCategoriesAreSorted(t *testing.T) {
	lex := computeLexicon()
	for i := 1; i < len(lex.UnitCategories); i++ {
		if lex.UnitCategories[i-1] > lex.UnitCategories[i] {
			t.Errorf("UnitCategories not sorted: %q > %q", lex.UnitCategories[i-1], lex.UnitCategories[i])
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

// TestComputeLexicon_KeywordPhrasesIncludesMultiWord — pins the
// contract that the LSP exposes multi-word date / period operator
// phrases so the client tokenizer can render each as a single
// highlighted pill.
func TestComputeLexicon_KeywordPhrasesIncludesMultiWord(t *testing.T) {
	lex := computeLexicon()
	wantSubset := []string{
		// Multi-word date keywords
		"this year", "next month", "last quarter",
		"this fiscal quarter", "next fiscal year", "last fiscal quarter",
		// Period operators
		"end of", "start of", "length of", "days in", "between",
		// Single-word vocabulary
		"as", "in", "by", "of", "per", "and",
		// Date keyword aliases
		"fiscal quarter", "fiscal year",
		// Single-word date keywords
		"today", "tomorrow", "yesterday",
	}
	for _, kw := range wantSubset {
		if !slices.Contains(lex.KeywordPhrases, kw) {
			t.Errorf("KeywordPhrases missing expected entry %q", kw)
		}
	}
}

// TestComputeLexicon_KeywordPhrasesSortedLongestFirst — the client
// matches greedy-longest, so ordering matters: `this fiscal quarter`
// must precede `this fiscal` (and `this`, etc.) in the wire output.
func TestComputeLexicon_KeywordPhrasesSortedLongestFirst(t *testing.T) {
	lex := computeLexicon()
	for i := 1; i < len(lex.KeywordPhrases); i++ {
		prev, curr := lex.KeywordPhrases[i-1], lex.KeywordPhrases[i]
		if len(prev) < len(curr) {
			t.Errorf("KeywordPhrases not longest-first: %q (len %d) before %q (len %d)",
				prev, len(prev), curr, len(curr))
			break
		}
	}
}
