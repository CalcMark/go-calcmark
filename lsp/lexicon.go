package lsp

import (
	"sort"

	"github.com/CalcMark/go-calcmark/v2/spec/features"
	"github.com/CalcMark/go-calcmark/v2/spec/lexer"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
	"github.com/CalcMark/go-calcmark/v2/spec/units"
)

// LexiconResult is the wire shape returned by `calcmark/lexicon`.
//
// The client uses these sets to color-code syntax in the calc-source
// preview — function names (`grow`, `throughput`, `accumulate`) and
// conversion keywords (`as`, `in`, `napkin`, `precise`) get a tertiary
// accent so the language vocabulary reads distinctly from user data.
//
// The sets are static per LSP version (the function registry and
// keyword list do not change at runtime), so the client can fetch
// once after `initialize` and cache for the session.
type LexiconResult struct {
	// Function names from the registry — both canonical (e.g. `grow`)
	// and synonyms (e.g. `average` for `avg`). Includes only function
	// suggestions reachable from FunctionSpecs / Registry, not lexer
	// pseudo-functions.
	Functions []string `json:"functions"`
	// Conversion-relevant keywords surfaced by the lexer's
	// ReservedKeywords map, plus contextual keywords used in NL forms
	// (`by`, `compounded`, `to`). Excludes control-flow words that
	// have no calcmark expression semantics yet (`if`, `else`, ...).
	ConversionKeywords []string `json:"conversionKeywords"`
	// Unit categories valid for `scale.unit_categories` and
	// `convert_to.unit_categories` in frontmatter. Mirrors
	// `units.Categories()` exactly — the canonical set the server-side
	// validator accepts. The TS frontmatter form pulls this list to
	// populate its category chip-list, so adding a category in
	// `spec/units` automatically lights it up in the UI without a
	// client-side mirror needing to follow.
	UnitCategories []string `json:"unitCategories"`
	// KeywordPhrases is the full set of keyword phrases the lexer
	// recognizes — single-word AND multi-word — so the client
	// tokenizer can render each as a single highlighted pill rather
	// than splitting `this fiscal quarter` across three segments. The
	// list is sorted longest-first so the client matches greedy
	// without ambiguity at the front of an identifier sequence.
	//
	// Sources: spec/lexer DateKeywords (today, tomorrow, weekdays),
	// RelativeDateKeywords (`this week`, `next month`, `end of`,
	// `length of`, `between`, …), ThreeWordDateKeywords
	// (`this fiscal quarter`, `last fiscal year`, …), plus the
	// hand-curated ConversionKeywords list above (the single-word
	// vocabulary like `as`, `in`, `napkin`).
	//
	// Synthetic parser phrases like `as a % of` are NOT in this list —
	// they're stitched from individual keyword tokens (`as`, `%`,
	// `of`) which each already appear here. Frontend rendering can
	// merge adjacent keyword spans visually if the styling calls for
	// it.
	KeywordPhrases []string `json:"keywordPhrases"`
}

// computeLexicon returns the static lexicon. Pure: no I/O, no document
// state.
func computeLexicon() LexiconResult {
	// Functions: every entry in FunctionSpecs plus every synonym in
	// the registry. Synonyms (`average` / `mean` for `avg`) are real
	// user-facing names — the parser accepts them — so they belong in
	// the function-color set even though they're not canonical keys.
	funcSet := make(map[string]struct{})
	for name := range types.FunctionSpecs {
		funcSet[name] = struct{}{}
	}
	registry := features.DefaultRegistry()
	for _, f := range registry.ByCategory(features.CategoryFunction) {
		for _, syn := range f.Synonyms {
			if syn != "" {
				funcSet[syn] = struct{}{}
			}
		}
	}
	functions := make([]string, 0, len(funcSet))
	for name := range funcSet {
		functions = append(functions, name)
	}
	sort.Strings(functions)

	// Conversion keywords: language-level vocabulary the user types
	// inline (`5 GB as napkin`, `100 MB/s over 1 day`, `compound X by Y`).
	// Hand-curated subset of ReservedKeywords + ContextualKeywords —
	// the full ReservedKeywords map includes control-flow tokens
	// (`if`, `else`, `for`, ...) that aren't meaningful in calcmark
	// expressions yet and shouldn't be color-coded.
	conversionKeywords := []string{
		"and",
		"as",
		"at",
		"between", // v2.0 — `between A and B`
		"by",
		"compounded",
		"from",
		"in",
		"napkin",
		"not",
		"of",
		"or",
		"over",
		"per",
		"precise",
		"to",
		"with",
	}

	// Unit categories: the canonical set from `units.Categories()`.
	// Sorted for stable wire output (consumers shouldn't depend on
	// order, but tests + diff-friendliness benefit). `units.Categories`
	// already deduplicates; we just sort the result.
	unitCategories := append([]string(nil), units.Categories()...)
	sort.Strings(unitCategories)

	// Keyword phrases: union of all lexer keyword maps + the hand-
	// curated single-word ConversionKeywords list. Sorted longest-
	// first so the client can match greedy without ambiguity (e.g.
	// `this fiscal quarter` matches before `this fiscal`).
	keywordSet := make(map[string]struct{})
	for k := range lexer.DateKeywords {
		keywordSet[k] = struct{}{}
	}
	for k := range lexer.RelativeDateKeywords {
		keywordSet[k] = struct{}{}
	}
	for k := range lexer.ThreeWordDateKeywords {
		keywordSet[k] = struct{}{}
	}
	for _, kw := range conversionKeywords {
		keywordSet[kw] = struct{}{}
	}
	keywordPhrases := make([]string, 0, len(keywordSet))
	for k := range keywordSet {
		keywordPhrases = append(keywordPhrases, k)
	}
	sort.Slice(keywordPhrases, func(i, j int) bool {
		// Longest first; tiebreak alphabetical for stable output.
		if len(keywordPhrases[i]) != len(keywordPhrases[j]) {
			return len(keywordPhrases[i]) > len(keywordPhrases[j])
		}
		return keywordPhrases[i] < keywordPhrases[j]
	})

	return LexiconResult{
		Functions:          functions,
		ConversionKeywords: conversionKeywords,
		UnitCategories:     unitCategories,
		KeywordPhrases:     keywordPhrases,
	}
}
