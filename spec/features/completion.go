package features

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/CalcMark/go-calcmark/v2/spec/units"
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
				Name:         name,
				Category:     f.Subcategory,
				Description:  f.Description,
				Syntax:       f.Syntax,
				InsertText:   f.Name,
				FunctionName: f.Name,
			})
		}

		// Emit NL row(s) immediately after the fn row
		for _, nl := range matchedNL {
			suggestions = append(suggestions, Suggestion{
				Name:         nl.aliasName,
				Category:     "example",
				Syntax:       nl.example,
				InsertText:   nl.example,
				FunctionName: f.Name,
				SortCategory: f.Subcategory, // Sort alongside parent function
			})
		}
	}

	return suggestions
}

// KeywordSuggestions returns insertable suggestions for the calc keyword-
// operator forms (`X% of Y`, `X as a % of Y`, `X in unit`). It mirrors
// FunctionSuggestions but is far simpler: keywords have no synonyms and no
// NL/paren split. Only keyword features that carry a snippet `InsertText`
// surface — the rest (`per`, `over`, `at`, napkin/precise, bare `as`) are
// language keywords without a slash-command form yet.
//
// The Name field carries the keyword identity (e.g. "of", "as % of", "in") so
// clients can attach their own palette label/aliases; InsertText carries the
// canonical `${N:default}` snippet so the `%`/unit placeholders live in one
// authoritative place.
func KeywordSuggestions(prefix string) []Suggestion {
	registry := DefaultRegistry()
	var suggestions []Suggestion

	for _, f := range registry.ByCategory(CategoryKeyword) {
		if f.InsertText == "" {
			continue
		}
		if !MatchesPrefix(f.Name, prefix) {
			continue
		}
		suggestions = append(suggestions, Suggestion{
			Name:        f.Name,
			Category:    string(CategoryKeyword),
			Description: f.Description,
			Syntax:      f.Syntax,
			InsertText:  f.InsertText,
		})
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

// templateDateNames lists registry entries whose name is a TEMPLATE
// placeholder form, not a literal calcmark expression. The parser
// cannot consume `this weekday` — the user is expected to type `this
// Friday` (specific weekday). Surfacing them as completions inserts
// invalid source. These names exist in the registry for documentation
// purposes (syntax, example) but must not appear in DateSuggestions.
var templateDateNames = map[string]bool{
	"next weekday":    true,
	"this weekday":    true,
	"last weekday":    true,
	"next month name": true,
	"this month name": true,
	"last month name": true,
}

// nonPeriodDateNames identifies CategoryDate features that are NOT
// period-bearing -- they resolve to a single date or a duration, not
// to a period the user can take the start/end of. The
// `end of <period>` / `start of <period>` synthesis handler skips
// these so it does not offer nonsensical combinations like
// `end of today` or `end of days`.
//
// Adding a new CategoryDate feature: if it produces a Period (a
// span the user can ask for the start or end of), do nothing -- it
// flows through synthesis automatically. If it produces a Date
// (point in time) or Duration (length), add it here.
var nonPeriodDateNames = map[string]bool{
	"today":     true,
	"tomorrow":  true,
	"yesterday": true,
	"now":       true,
	"days":      true,
	"weeks":     true,
	"months":    true,
	"years":     true,
	"from":      true,
	"ago":       true,
}

// DateSuggestions returns date keyword suggestions matching prefix.
// Surfaces date-related features like "today", "next Friday", "this quarter", "ago", "end of".
func DateSuggestions(prefix string) []Suggestion {
	prefix = strings.ToLower(prefix)
	var suggestions []Suggestion

	// `end[ of]?` / `start[ of]?` prefix triggers the synthesis
	// handler, which prefixes the registered period set with
	// `end of ` / `start of `. Stays in sync with the registry --
	// new period kinds light up automatically.
	if op, isOp, opPrefix := detectEndStartOfPrefix(prefix); isOp {
		return synthesizeEndStartOfPeriods(op, opPrefix)
	}

	registry := DefaultRegistry()

	for _, f := range registry.ByCategory(CategoryDate) {
		if !templateDateNames[f.Name] && MatchesPrefix(f.Name, prefix) {
			// Snippet-form entries (e.g., year-bearing literals like
			// `FY${1:NNNN}`) carry their own InsertText. Plain literals
			// fall back to Name. Downstream LSP layers detect the
			// `${...}` placeholder and flag InsertTextFormat: Snippet.
			insertText := f.InsertText
			if insertText == "" {
				insertText = f.Name
			}
			suggestions = append(suggestions, Suggestion{
				Name:        f.Name,
				Category:    "Date",
				Description: f.Description,
				Syntax:      f.Syntax,
				InsertText:  insertText,
			})
		}
		// Also check aliases for parseable date expressions.
		//
		// Suffix matching for relative-period aliases: when an alias
		// starts with `this `, `next `, or `last ` followed by an
		// abbreviation like `FQ` / `CQ` / `FY` / `CY`, also match the
		// abbreviation prefix on its own. The lexer rejects bare `FQ`
		// (it collides with `FQ1` notation parsing — see
		// spec/lexer/date_keywords.go), so the user types `FQ` to
		// search but the dropdown surfaces `this FQ` (which lexes
		// correctly). Inserting `this FQ` rather than `FQ` keeps the
		// completion always parseable.
		for _, alias := range f.Aliases {
			if templateDateNames[alias.Name] {
				continue
			}
			matched := alias.Parseable && MatchesPrefix(alias.Name, prefix)
			if !matched && alias.Parseable {
				// Try matching the period-abbreviation suffix.
				if suffix, ok := relativePeriodSuffix(alias.Name); ok && MatchesPrefix(suffix, prefix) {
					matched = true
				}
			}
			if matched {
				suggestions = append(suggestions, Suggestion{
					Name:        alias.Name,
					Category:    "Date",
					Description: f.Description,
					Syntax:      f.Syntax,
					InsertText:  alias.Name,
				})
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

// detectEndStartOfPrefix recognizes the `end[ of]?` / `start[ of]?`
// prefix shapes that trigger synthesis. Returns:
//   - op: the operator string ("end of " or "start of ")
//   - isOp: true when the prefix matches one of these shapes
//   - opPrefix: the period-name prefix the user has typed beyond
//     the operator (e.g., "Q" from "end of Q", or "" from "end")
//
// Word-boundary check: the prefix must be exactly `end`, `end `,
// `end of`, `end of `, `end of <text>`, or the `start` equivalents.
// `ending`, `endorse`, `started` etc. don't match -- the input
// has to *be* the operator, not just begin with it.
func detectEndStartOfPrefix(prefix string) (op string, isOp bool, opPrefix string) {
	for _, candidate := range []string{"end", "start"} {
		full := candidate + " of"
		switch {
		case prefix == candidate:
			return candidate + " of ", true, ""
		case strings.HasPrefix(prefix, candidate+" ") && (prefix == candidate+" " || strings.HasPrefix(prefix, full)):
			// Either "end " (still typing) or "end of..." (typing the period).
			rest := strings.TrimPrefix(prefix, candidate+" ")
			rest = strings.TrimPrefix(rest, "of")
			rest = strings.TrimPrefix(rest, " ")
			return candidate + " of ", true, rest
		}
	}
	return "", false, ""
}

// synthesizeEndStartOfPeriods emits `op + <period-name>` items for
// every registered period-bearing CategoryDate feature whose name
// matches `opPrefix`. The `nonPeriodDateNames` skip-list filters out
// non-period entries (today, tomorrow, days, etc.).
//
// Snippet propagation: when a period entry has its own InsertText
// (e.g., FY/CY year-bearing snippets `FY${1:2026}`), the synthesized
// item carries `op + that-snippet` so editors still treat the year
// as a placeholder.
func synthesizeEndStartOfPeriods(op string, opPrefix string) []Suggestion {
	var out []Suggestion
	registry := DefaultRegistry()

	opPrefixLower := strings.ToLower(opPrefix)

	for _, f := range registry.ByCategory(CategoryDate) {
		if templateDateNames[f.Name] || nonPeriodDateNames[f.Name] {
			continue
		}
		if opPrefix != "" && !MatchesPrefix(f.Name, opPrefixLower) {
			continue
		}
		insertBase := f.InsertText
		if insertBase == "" {
			insertBase = f.Name
		}
		out = append(out, Suggestion{
			Name:        op + f.Name,
			Category:    "Date",
			Description: f.Description,
			Syntax:      op + f.Syntax,
			InsertText:  op + insertBase,
		})
	}
	return out
}

// relativePeriodSuffix returns the trailing abbreviation of a
// relative-period alias name. Given `"this FQ"` returns `"FQ"`,
// `true`. Given anything that doesn't start with a relative
// modifier OR whose suffix is a multi-word phrase rather than a
// short abbreviation, returns `"", false`.
//
// Used by DateSuggestions to surface aliases like `this FQ` when
// the user types just `FQ`. Only matches the recognized
// abbreviations (FQ / CQ / FY / CY) so we don't accidentally let
// `this fiscal quarter` match the prefix `fiscal` (that already
// matches via the alias name itself).
func relativePeriodSuffix(aliasName string) (string, bool) {
	for _, modifier := range []string{"this ", "next ", "last "} {
		if !strings.HasPrefix(aliasName, modifier) {
			continue
		}
		rest := aliasName[len(modifier):]
		switch rest {
		case "FQ", "CQ", "FY", "CY":
			return rest, true
		}
	}
	return "", false
}
