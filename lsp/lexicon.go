package lsp

import (
	"sort"

	"github.com/CalcMark/go-calcmark/spec/features"
	"github.com/CalcMark/go-calcmark/spec/types"
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

	return LexiconResult{
		Functions:          functions,
		ConversionKeywords: conversionKeywords,
	}
}
