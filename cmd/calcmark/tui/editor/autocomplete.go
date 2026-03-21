package editor

import (
	"sort"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/components"
	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/CalcMark/go-calcmark/spec/features"
)

// implementedFunctionNames returns a set of function names that have interpreter
// implementations. Used to filter FunctionSuggestions to only show functions
// the interpreter can actually evaluate.
func implementedFunctionNames() map[string]bool {
	names := make(map[string]bool, len(interpreter.BuiltinFunctions))
	for _, fn := range interpreter.BuiltinFunctions {
		names[fn.Name] = true
	}
	return names
}

// CombinedSuggestionSource queries shared completion functions and merges results.
// Wraps calls to features.FunctionSuggestions, features.UnitSuggestions,
// features.VariableSuggestions, and features.DirectiveSuggestions.
type CombinedSuggestionSource struct {
	implementedNames map[string]bool
	getVariables     func() map[string]string
	getDefinedLines  func() map[string]int
	getFrontmatter   func() *document.Frontmatter
	CursorLine       int // Current cursor line (0-indexed), set before GetSuggestions
}

// NewCombinedSuggestionSource creates a combined source that delegates to shared
// completion functions in spec/features.
func NewCombinedSuggestionSource(
	getVars func() map[string]string,
	getLines func() map[string]int,
	getFm func() *document.Frontmatter,
) *CombinedSuggestionSource {
	return &CombinedSuggestionSource{
		implementedNames: implementedFunctionNames(),
		getVariables:     getVars,
		getDefinedLines:  getLines,
		getFrontmatter:   getFm,
	}
}

// GetSuggestions returns suggestions from all completion sources, sorted by
// category then name. Uses stable sort to preserve fn/nl pair ordering.
func (c *CombinedSuggestionSource) GetSuggestions(prefix string) []components.Suggestion {
	var all []components.Suggestion

	// Functions (with NL rows)
	all = append(all, features.FunctionSuggestions(prefix, c.implementedNames)...)

	// Units
	all = append(all, features.UnitSuggestions(prefix)...)

	// Variables (position-aware)
	if c.getVariables != nil {
		vars := c.getVariables()
		var definedLines map[string]int
		if c.getDefinedLines != nil {
			definedLines = c.getDefinedLines()
		}
		all = append(all, features.VariableSuggestions(vars, prefix, c.CursorLine, definedLines)...)
	}

	// Directives (@scale, @globals.field)
	if c.getFrontmatter != nil {
		fm := c.getFrontmatter()
		if fm != nil {
			scaleFactor := ""
			if fm.Scale != nil {
				scaleFactor = fm.Scale.Factor.String()
			}
			all = append(all, features.DirectiveSuggestions(prefix, scaleFactor, fm.Globals)...)
		}
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
		"directive":   -1,  // Directives first (they're context-specific)
		"variable":    100, // Variables last
	}

	sort.SliceStable(all, func(i, j int) bool {
		ci := suggestionSortOrder(all[i], catOrder)
		cj := suggestionSortOrder(all[j], catOrder)
		if ci != cj {
			return ci < cj
		}
		// Within the same sort order, don't re-sort NL "example" rows --
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
