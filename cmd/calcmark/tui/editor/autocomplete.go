package editor

import (
	"fmt"
	"sort"
	"strings"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/components"
	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/units"
)

// FunctionSuggestionSource provides function name suggestions from BuiltinFunctions.
type FunctionSuggestionSource struct{}

// NewFunctionSuggestionSource creates a new function suggestion source.
func NewFunctionSuggestionSource() *FunctionSuggestionSource {
	return &FunctionSuggestionSource{}
}

// GetSuggestions returns function suggestions matching the given prefix.
// Matches against both primary names and synonyms.
func (f *FunctionSuggestionSource) GetSuggestions(prefix string) []components.Suggestion {
	prefix = strings.ToLower(prefix)
	var suggestions []components.Suggestion

	for _, fn := range interpreter.BuiltinFunctions {
		// Match primary name or any synonym
		matched := strings.HasPrefix(strings.ToLower(fn.Name), prefix)
		matchedSynonym := ""
		for _, syn := range fn.Synonyms {
			if strings.HasPrefix(strings.ToLower(syn), prefix) {
				matched = true
				matchedSynonym = syn
				break
			}
		}

		if matched {
			// Format name with synonyms for display
			name := fn.Name
			if len(fn.Synonyms) > 0 {
				name = fmt.Sprintf("%s (%s)", fn.Name, strings.Join(fn.Synonyms, ", "))
			}

			// If matched via synonym, show that more prominently
			if matchedSynonym != "" && matchedSynonym != fn.Name {
				name = fmt.Sprintf("%s (%s)", fn.Name, matchedSynonym)
			}

			suggestions = append(suggestions, components.Suggestion{
				Name:        name,
				Category:    fn.Category,
				Description: fn.Description,
				Syntax:      fn.Signature,
				InsertText:  fn.Name, // Always insert the primary function name
			})
		}
	}

	return suggestions
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
type VariableSuggestionSource struct {
	getVariables func() map[string]string // Returns varName -> formattedValue
}

// NewVariableSuggestionSource creates a new variable suggestion source.
// The getVars function is called each time to get current variables.
func NewVariableSuggestionSource(getVars func() map[string]string) *VariableSuggestionSource {
	return &VariableSuggestionSource{getVariables: getVars}
}

// GetSuggestions returns variable suggestions matching the given prefix.
func (v *VariableSuggestionSource) GetSuggestions(prefix string) []components.Suggestion {
	if v.getVariables == nil {
		return nil
	}

	prefix = strings.ToLower(prefix)
	var suggestions []components.Suggestion

	for varName, value := range v.getVariables() {
		if strings.HasPrefix(strings.ToLower(varName), prefix) {
			suggestions = append(suggestions, components.Suggestion{
				Name:        varName,
				Category:    "variable",
				Description: value,
				InsertText:  varName,
			})
		}
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
func (c *CombinedSuggestionSource) GetSuggestions(prefix string) []components.Suggestion {
	var all []components.Suggestion
	for _, src := range c.sources {
		all = append(all, src.GetSuggestions(prefix)...)
	}

	// Sort: functions first (by category order), then units, then variables
	// Within each category, sort alphabetically
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

	sort.Slice(all, func(i, j int) bool {
		ci := catOrder[all[i].Category]
		cj := catOrder[all[j].Category]
		// Categories not in map get a default high value
		if _, ok := catOrder[all[i].Category]; !ok {
			ci = 50
		}
		if _, ok := catOrder[all[j].Category]; !ok {
			cj = 50
		}
		if ci != cj {
			return ci < cj
		}
		return all[i].Name < all[j].Name
	})

	return all
}
