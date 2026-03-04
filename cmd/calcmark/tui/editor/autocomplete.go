package editor

import (
	"fmt"
	"sort"
	"strings"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/components"
	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/features"
	"github.com/CalcMark/go-calcmark/spec/units"
)

// nlInfo holds NL example data for a function, sourced from the feature registry.
type nlInfo struct {
	aliasName string // Cleaned alias name for display (e.g., "average of")
	example   string // Concrete example for insertion (e.g., "average of 1, 2, 3")
	matchWord string // First word for prefix matching (e.g., "average")
}

// FunctionSuggestionSource provides function name suggestions from BuiltinFunctions,
// plus NL example rows sourced from the feature registry.
type FunctionSuggestionSource struct {
	nlByFunction map[string][]nlInfo // function name -> NL examples
}

// NewFunctionSuggestionSource creates a new function suggestion source.
// Builds an NL lookup from the feature registry for NL example rows.
func NewFunctionSuggestionSource() *FunctionSuggestionSource {
	src := &FunctionSuggestionSource{
		nlByFunction: make(map[string][]nlInfo),
	}

	registry := features.NewRegistry()
	for _, f := range registry.ByCategory(features.CategoryFunction) {
		// Parseable aliases with examples
		for _, alias := range f.Aliases {
			if alias.Parseable && alias.Example != "" {
				src.nlByFunction[f.Name] = append(src.nlByFunction[f.Name], nlInfo{
					aliasName: cleanAliasName(alias.Name),
					example:   alias.Example,
					matchWord: firstWord(alias.Name),
				})
			}
		}
		// Functions with NLExample but no parseable alias examples
		if f.NLExample != "" && len(src.nlByFunction[f.Name]) == 0 {
			src.nlByFunction[f.Name] = append(src.nlByFunction[f.Name], nlInfo{
				aliasName: f.Name,
				example:   f.NLExample,
				matchWord: f.Name, // Match by function name only
			})
		}
	}

	return src
}

// GetSuggestions returns function suggestions matching the given prefix.
// Matches against primary names, synonyms, and NL alias keywords.
// Emits fn/nl pairs: function row followed by its NL example row(s).
func (f *FunctionSuggestionSource) GetSuggestions(prefix string) []components.Suggestion {
	prefix = strings.ToLower(prefix)
	var suggestions []components.Suggestion

	for _, fn := range interpreter.BuiltinFunctions {
		// Match primary name or any synonym
		fnMatched := strings.HasPrefix(strings.ToLower(fn.Name), prefix)
		matchedSynonym := ""
		for _, syn := range fn.Synonyms {
			if strings.HasPrefix(strings.ToLower(syn), prefix) {
				fnMatched = true
				matchedSynonym = syn
				break
			}
		}

		// Check NL alias match (even if function name didn't match)
		nlMatched := false
		var matchedNL []nlInfo
		if nls, ok := f.nlByFunction[fn.Name]; ok {
			for _, nl := range nls {
				if strings.HasPrefix(strings.ToLower(nl.matchWord), prefix) {
					nlMatched = true
					matchedNL = append(matchedNL, nl)
				}
			}
			// If fn matched but NL didn't match by keyword, still include all NL rows
			if fnMatched && !nlMatched {
				matchedNL = nls
			}
		}

		// Emit fn row if function name or synonym matched
		if fnMatched {
			name := fn.Name
			if len(fn.Synonyms) > 0 {
				name = fmt.Sprintf("%s (%s)", fn.Name, strings.Join(fn.Synonyms, ", "))
			}
			if matchedSynonym != "" && matchedSynonym != fn.Name {
				name = fmt.Sprintf("%s (%s)", fn.Name, matchedSynonym)
			}

			suggestions = append(suggestions, components.Suggestion{
				Name:        name,
				Category:    fn.Category,
				Description: fn.Description,
				Syntax:      fn.Signature,
				InsertText:  fn.Name,
			})
		}

		// Emit NL row(s) immediately after the fn row
		for _, nl := range matchedNL {
			suggestions = append(suggestions, components.Suggestion{
				Name:         nl.aliasName,
				Category:     "example",
				Syntax:       nl.example,
				InsertText:   nl.example,
				SortCategory: fn.Category, // Sort alongside parent function
			})
		}
	}

	return suggestions
}

// cleanAliasName strips "..." template notation from alias names.
// e.g., "transfer...across" → "transfer across"
func cleanAliasName(name string) string {
	return strings.ReplaceAll(name, "...", " ")
}

// firstWord returns the text before the first space or "..." in a name.
// e.g., "average of" → "average", "transfer...across" → "transfer"
func firstWord(name string) string {
	if before, _, ok := strings.Cut(name, "..."); ok {
		return before
	}
	if before, _, ok := strings.Cut(name, " "); ok {
		return before
	}
	return name
}

// UnitSuggestionSource provides unit name suggestions from StandardUnits.
type UnitSuggestionSource struct{}

// NewUnitSuggestionSource creates a new unit suggestion source.
func NewUnitSuggestionSource() *UnitSuggestionSource {
	return &UnitSuggestionSource{}
}

// GetSuggestions returns unit suggestions matching the given prefix.
// Matches against canonical names, symbols, and aliases.
func (u *UnitSuggestionSource) GetSuggestions(prefix string) []components.Suggestion {
	prefix = strings.ToLower(prefix)
	var suggestions []components.Suggestion
	seen := make(map[string]bool) // Avoid duplicate canonical names

	for _, unit := range units.StandardUnits {
		// Skip if we've already added this canonical unit
		if seen[unit.Canonical] {
			continue
		}

		matched := strings.HasPrefix(strings.ToLower(unit.Canonical), prefix)
		if !matched {
			matched = strings.HasPrefix(strings.ToLower(unit.Symbol), prefix)
		}
		// Also check aliases
		if !matched {
			for _, alias := range unit.Aliases {
				if strings.HasPrefix(strings.ToLower(alias), prefix) {
					matched = true
					break
				}
			}
		}

		if matched {
			seen[unit.Canonical] = true
			suggestions = append(suggestions, components.Suggestion{
				Name:        unit.Canonical,
				Category:    unit.Quantity,
				Description: unit.Description,
				Syntax:      unit.Symbol,
				InsertText:  unit.Canonical,
			})
		}
	}

	return suggestions
}

// VariableSuggestionSource provides variable name suggestions from the document environment.
// Position-aware: only suggests variables defined above CursorLine.
type VariableSuggestionSource struct {
	getVariables    func() map[string]string // Returns varName -> formattedValue
	getDefinedLines func() map[string]int    // Returns varName -> definition line (0-indexed)
	CursorLine      int                      // Current cursor line (0-indexed), set before GetSuggestions
}

// NewVariableSuggestionSource creates a new variable suggestion source.
// getVars returns all variables. getLines returns definition line numbers.
// Set CursorLine before calling GetSuggestions to filter by position.
func NewVariableSuggestionSource(getVars func() map[string]string, getLines func() map[string]int) *VariableSuggestionSource {
	return &VariableSuggestionSource{getVariables: getVars, getDefinedLines: getLines}
}

// GetSuggestions returns variable suggestions matching the given prefix.
// Only includes variables defined above CursorLine (or built-ins/globals with no line).
func (v *VariableSuggestionSource) GetSuggestions(prefix string) []components.Suggestion {
	if v.getVariables == nil {
		return nil
	}

	prefix = strings.ToLower(prefix)
	var suggestions []components.Suggestion

	vars := v.getVariables()
	var definedLines map[string]int
	if v.getDefinedLines != nil {
		definedLines = v.getDefinedLines()
	}

	for varName, value := range vars {
		if !strings.HasPrefix(strings.ToLower(varName), prefix) {
			continue
		}
		// Position filtering: exclude variables defined at or after cursor
		if definedLines != nil {
			if lineNum, hasLine := definedLines[varName]; hasLine && lineNum >= v.CursorLine {
				continue
			}
		}
		suggestions = append(suggestions, components.Suggestion{
			Name:        varName,
			Category:    "variable",
			Description: value,
			InsertText:  varName,
		})
	}

	return suggestions
}

// CombinedSuggestionSource queries multiple sources and merges results.
type CombinedSuggestionSource struct {
	sources []components.SuggestionSource
}

// NewCombinedSuggestionSource creates a combined source from multiple sources.
func NewCombinedSuggestionSource(sources ...components.SuggestionSource) *CombinedSuggestionSource {
	return &CombinedSuggestionSource{sources: sources}
}

// GetSuggestions returns suggestions from all sources, sorted by category then name.
// Uses stable sort to preserve fn/nl pair ordering from FunctionSuggestionSource.
func (c *CombinedSuggestionSource) GetSuggestions(prefix string) []components.Suggestion {
	var all []components.Suggestion
	for _, src := range c.sources {
		all = append(all, src.GetSuggestions(prefix)...)
	}

	// Sort: functions first (by category order), then units, then variables.
	// NL "example" rows use SortCategory to sort alongside their parent function.
	// SliceStable preserves insertion order for equal keys, keeping fn/nl pairs adjacent.
	catOrder := map[string]int{
		"Math":        0,
		"Conversion":  1,
		"Network":     2,
		"Storage":     3,
		"Capacity":    4,
		"Length":      10,
		"Mass":        11,
		"Volume":      12,
		"Temperature": 13,
		"Speed":       14,
		"Energy":      15,
		"Power":       16,
		"Area":        17,
		"variable":    100, // Variables last
	}

	sort.SliceStable(all, func(i, j int) bool {
		ci := suggestionSortOrder(all[i], catOrder)
		cj := suggestionSortOrder(all[j], catOrder)
		if ci != cj {
			return ci < cj
		}
		// Within the same sort order, don't re-sort NL "example" rows —
		// they should stay in insertion order (immediately after their fn row).
		if all[i].Category == "example" || all[j].Category == "example" {
			return false // preserve insertion order
		}
		return all[i].Name < all[j].Name
	})

	return all
}

// suggestionSortOrder returns the sort order for a suggestion.
// Uses SortCategory if set (for NL rows), otherwise falls back to Category.
func suggestionSortOrder(s components.Suggestion, catOrder map[string]int) int {
	cat := s.Category
	if s.SortCategory != "" {
		cat = s.SortCategory
	}
	if order, ok := catOrder[cat]; ok {
		return order
	}
	return 50
}
