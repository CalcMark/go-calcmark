package features

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/CalcMark/go-calcmark/spec/units"
)

// FunctionSuggestions returns function suggestions matching prefix.
// implementedNames filters to only functions that have interpreter implementations.
// Pass nil to include all registered functions.
func FunctionSuggestions(prefix string, implementedNames map[string]bool) []Suggestion {
	prefix = strings.ToLower(prefix)
	var suggestions []Suggestion

	registry := DefaultRegistry()

	// Pre-build NL lookup from parseable aliases and NLExample fallbacks.
	type nlInfo struct {
		aliasName string // Cleaned alias name for display (e.g., "average of")
		example   string // Concrete example for insertion (e.g., "average of 1, 2, 3")
		matchWord string // First word for prefix matching (e.g., "average")
	}
	nlByFunction := make(map[string][]nlInfo)
	for _, f := range registry.ByCategory(CategoryFunction) {
		for _, alias := range f.Aliases {
			if alias.Parseable && alias.Example != "" {
				nlByFunction[f.Name] = append(nlByFunction[f.Name], nlInfo{
					aliasName: cleanAliasName(alias.Name),
					example:   alias.Example,
					matchWord: firstWord(alias.Name),
				})
			}
		}
		if f.NLExample != "" && len(nlByFunction[f.Name]) == 0 {
			nlByFunction[f.Name] = append(nlByFunction[f.Name], nlInfo{
				aliasName: f.Name,
				example:   f.NLExample,
				matchWord: f.Name,
			})
		}
	}

	for _, f := range registry.ByCategory(CategoryFunction) {
		// Filter to implemented functions when the set is provided
		if implementedNames != nil && !implementedNames[f.Name] {
			continue
		}

		// Match primary name or any synonym
		fnMatched := MatchesPrefix(f.Name, prefix)
		matchedSynonym := ""
		for _, syn := range f.Synonyms {
			if MatchesPrefix(syn, prefix) {
				fnMatched = true
				matchedSynonym = syn
				break
			}
		}

		// Check NL alias match (even if function name didn't match)
		nlMatched := false
		var matchedNL []nlInfo
		if nls, ok := nlByFunction[f.Name]; ok {
			for _, nl := range nls {
				if MatchesPrefix(nl.matchWord, prefix) {
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
			name := f.Name
			if len(f.Synonyms) > 0 {
				name = fmt.Sprintf("%s (%s)", f.Name, strings.Join(f.Synonyms, ", "))
			}
			if matchedSynonym != "" && matchedSynonym != f.Name {
				name = fmt.Sprintf("%s (%s)", f.Name, matchedSynonym)
			}

			suggestions = append(suggestions, Suggestion{
				Name:        name,
				Category:    f.Subcategory,
				Description: f.Description,
				Syntax:      f.Syntax,
				InsertText:  f.Name,
			})
		}

		// Emit NL row(s) immediately after the fn row
		for _, nl := range matchedNL {
			suggestions = append(suggestions, Suggestion{
				Name:         nl.aliasName,
				Category:     "example",
				Syntax:       nl.example,
				InsertText:   nl.example,
				SortCategory: f.Subcategory, // Sort alongside parent function
			})
		}
	}

	return suggestions
}

// UnitSuggestions returns unit suggestions matching prefix.
// Deduplicates by canonical name.
func UnitSuggestions(prefix string) []Suggestion {
	prefix = strings.ToLower(prefix)
	var suggestions []Suggestion
	seen := make(map[string]bool)

	for _, unit := range units.StandardUnits {
		if seen[unit.Canonical] {
			continue
		}

		matched := MatchesPrefix(unit.Canonical, prefix)
		if !matched {
			matched = MatchesPrefix(unit.Symbol, prefix)
		}
		if !matched {
			for _, alias := range unit.Aliases {
				if MatchesPrefix(alias, prefix) {
					matched = true
					break
				}
			}
		}

		if matched {
			seen[unit.Canonical] = true
			suggestions = append(suggestions, Suggestion{
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

// VariableSuggestions returns variable suggestions matching prefix.
// Filters out variables defined at or after cursorLine when definedLines is provided.
// vars maps variable names to their formatted display values.
func VariableSuggestions(vars map[string]string, prefix string, cursorLine int, definedLines map[string]int) []Suggestion {
	if vars == nil {
		return nil
	}

	prefix = strings.ToLower(prefix)
	var suggestions []Suggestion

	for varName, value := range vars {
		if !MatchesPrefix(varName, prefix) {
			continue
		}
		// Position filtering: exclude variables defined at or after cursor
		if definedLines != nil {
			if lineNum, hasLine := definedLines[varName]; hasLine && lineNum >= cursorLine {
				continue
			}
		}
		suggestions = append(suggestions, Suggestion{
			Name:        varName,
			Category:    "variable",
			Description: value,
			InsertText:  varName,
		})
	}

	return suggestions
}

// DirectiveSuggestions returns @scale and @globals.field suggestions.
// scaleFactor is the string representation of the scale factor (empty if no scale).
// globals maps global names to their values (nil or empty if no globals).
func DirectiveSuggestions(prefix string, scaleFactor string, globals map[string]string) []Suggestion {
	// Only respond to prefixes starting with '@'
	if !strings.HasPrefix(prefix, "@") {
		return nil
	}

	var suggestions []Suggestion

	// @scale -- offered when a scale factor is configured
	if scaleFactor != "" {
		name := "@scale"
		if MatchesPrefix(name, strings.ToLower(prefix)) {
			suggestions = append(suggestions, Suggestion{
				Name:        "@scale",
				Category:    "directive",
				Description: fmt.Sprintf("Scale factor (%s)", scaleFactor),
				InsertText:  "@scale",
			})
		}
	}

	// @globals.field -- offered when globals are defined
	if len(globals) > 0 {
		if strings.HasPrefix("@globals", strings.ToLower(prefix)) && !strings.Contains(prefix, ".") {
			suggestions = append(suggestions, Suggestion{
				Name:        "@globals",
				Category:    "directive",
				Description: fmt.Sprintf("%d global(s) defined", len(globals)),
				InsertText:  "@globals.",
			})
		} else if strings.HasPrefix(strings.ToLower(prefix), "@globals.") {
			fieldPrefix := strings.ToLower(prefix[len("@globals."):])
			for name, value := range globals {
				if MatchesPrefix(name, fieldPrefix) {
					suggestions = append(suggestions, Suggestion{
						Name:        "@globals." + name,
						Category:    "directive",
						Description: value,
						InsertText:  "@globals." + name,
					})
				}
			}
		}
	}

	return suggestions
}

// ExtractPrefix extracts the completion prefix from a line at the given column.
// Handles @globals.field patterns and word boundaries.
// Uses rune-aware indexing for UTF-8 safety.
func ExtractPrefix(line string, col int) string {
	runes := []rune(line)
	if col > len(runes) {
		col = len(runes)
	}

	isWord := func(ch rune) bool {
		return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_'
	}

	// Walk backward from cursor to find start of identifier
	start := col
	for start > 0 && isWord(runes[start-1]) {
		start--
	}

	// Extend to include '@' and dot-separated @globals.field patterns.
	// Only include '@' when preceded by non-word char (not email@example).
	if start > 0 && runes[start-1] == '.' {
		// @globals.field or @globals. (no field yet)
		dotPos := start - 1
		wordStart := dotPos
		for wordStart > 0 && isWord(runes[wordStart-1]) {
			wordStart--
		}
		if wordStart > 0 && runes[wordStart-1] == '@' && (wordStart <= 1 || !isWord(runes[wordStart-2])) {
			start = wordStart - 1
		}
	} else if start > 0 && runes[start-1] == '@' && (start <= 1 || !isWord(runes[start-2])) {
		start--
	} else if start >= col && col > 0 && runes[col-1] == '.' {
		// Cursor right after dot: @globals.
		dotPos := col - 1
		wordStart := dotPos
		for wordStart > 0 && isWord(runes[wordStart-1]) {
			wordStart--
		}
		if wordStart > 0 && runes[wordStart-1] == '@' && (wordStart <= 1 || !isWord(runes[wordStart-2])) {
			start = wordStart - 1
		}
	}

	return string(runes[start:col])
}

// cleanAliasName strips "..." template notation from alias names.
// e.g., "transfer...across" -> "transfer across"
func cleanAliasName(name string) string {
	return strings.ReplaceAll(name, "...", " ")
}

// firstWord returns the text before the first space or "..." in a name.
// e.g., "average of" -> "average", "transfer...across" -> "transfer"
func firstWord(name string) string {
	if before, _, ok := strings.Cut(name, "..."); ok {
		return before
	}
	if before, _, ok := strings.Cut(name, " "); ok {
		return before
	}
	return name
}
