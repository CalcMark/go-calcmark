package lsp

import (
	"sort"

	"github.com/CalcMark/go-calcmark/spec/features"
	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/CalcMark/go-calcmark/spec/units"
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

	return LexiconResult{
		Functions:          functions,
		ConversionKeywords: conversionKeywords,
		UnitCategories:     unitCategories,
	}
}
