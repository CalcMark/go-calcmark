package lsp

import (
	"strings"
	"unicode"

	"github.com/CalcMark/go-calcmark/v2/spec/features"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
)

// argumentContext describes where a cursor sits relative to an enclosing
// function call. Returned by extractArgumentContext.
//
// When the cursor is not inside any call, funcName is "" and paramIdx is -1.
//
// insideString reports whether the cursor lies inside a '"' or "'" quoted
// run within the active argument. Calcmark has no string literal syntax —
// enum-backed parameters take bare identifiers like throughput(gigabit) —
// so insideString is used only as a suppression signal: when true, the
// LSP does not offer bare-identifier completions that would produce
// invalid calcmark code.
type argumentContext struct {
	funcName     string
	paramIdx     int
	insideString bool
	// isNL reports whether the cursor sits in a natural-language form
	// (e.g., `grow 100 by 20 over 5 months`) rather than the paren-
	// based form (`grow(100, 20, 5)`). Signature help uses this to
	// surface the matching alias example as the signature label so
	// the user sees `grow X by Y over Z months` while typing the NL
	// form, instead of `grow(amount, increment, periods)`.
	isNL bool
	// requiredType, when non-empty, is the expected argument type at the
	// cursor independent of any function spec. It is set for keyword-
	// operator operands (e.g. the percentage operand of `X% of Y`) where
	// there is no enclosing function call to look up a ParamSpec from. The
	// completion handler prefers this over the funcName/paramIdx path so
	// variable suggestions are type-filtered the same way they are inside
	// a function call.
	requiredType types.ArgType
}

// extractArgumentContext scans lineText from the start up to col and reports
// the innermost function call context at the cursor. Uses a forward scan with
// a paren-frame stack because forward scanning is the natural way to track
// quote-run state across the line.
//
// The scan is rune-aware for UTF-8 safety. Both '"' and "'" act as quote
// delimiters symmetrically — whichever one opens the run closes it. There is
// no backslash-escape handling because calcmark has no string literal syntax
// and therefore no escape semantics to honor.
func extractArgumentContext(lineText string, col int) argumentContext {
	runes := []rune(lineText)
	if col > len(runes) {
		col = len(runes)
	}

	type frame struct {
		funcName     string
		paramIdx     int
		insideString bool
		openQuote    rune // '"' or '\'' while insideString, otherwise 0
	}
	var stack []frame

	for i := 0; i < col; i++ {
		r := runes[i]

		if len(stack) > 0 && stack[len(stack)-1].insideString {
			top := &stack[len(stack)-1]
			if r == top.openQuote {
				top.insideString = false
				top.openQuote = 0
			}
			// All other characters (including '(' ')' ',' '\\') are consumed
			// as string content while inside a quote run — they do NOT affect
			// paren depth or paramIdx tracking.
			continue
		}

		switch r {
		case '(':
			name := identifierEndingAt(runes, i)
			stack = append(stack, frame{funcName: name, paramIdx: 0})
		case ')':
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case ',':
			if len(stack) > 0 {
				stack[len(stack)-1].paramIdx++
			}
		case '"', '\'':
			if len(stack) > 0 {
				top := &stack[len(stack)-1]
				top.insideString = true
				top.openQuote = r
			}
		}
	}

	if len(stack) == 0 {
		// Natural-language *function* forms win first (`grow 100 by 20`,
		// `average of 1, 2, 3`), so a known function keyword like `of` in
		// `average of …` resolves to its function rather than the bare
		// percentage operator below.
		nl := extractNLArgumentContext(lineText, col)
		if nl.funcName != "" {
			nl.isNL = true
			return nl
		}
		// No enclosing function: try the keyword-operator forms, which type
		// their operands (e.g. the percentage operand of `X% of Y`).
		if kw, ok := extractKeywordArgumentContext(lineText, col); ok {
			return kw
		}
		return nl
	}

	top := stack[len(stack)-1]
	if top.funcName == "" {
		return argumentContext{funcName: "", paramIdx: -1}
	}

	return argumentContext{
		funcName:     top.funcName,
		paramIdx:     top.paramIdx,
		insideString: top.insideString,
	}
}

// identifierEndingAt returns the identifier-run ending immediately before
// position end (exclusive). Returns "" if no identifier precedes the position.
func identifierEndingAt(runes []rune, end int) string {
	start := end
	for start > 0 {
		r := runes[start-1]
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			start--
			continue
		}
		break
	}
	if start == end {
		return ""
	}
	// Reject pure-digit runs (e.g., "42(" is not a function call).
	if unicode.IsDigit(runes[start]) {
		return ""
	}
	return string(runes[start:end])
}

// extractNLArgumentContext detects natural-language function calls
// (e.g., "grow 100 by 20 over 5 months") and returns the argument context.
// It is called as a fallback when the paren-based scanner finds no call.
func extractNLArgumentContext(lineText string, col int) argumentContext {
	empty := argumentContext{funcName: "", paramIdx: -1}

	runes := []rune(lineText)
	if col > len(runes) {
		col = len(runes)
	}

	// Strip optional assignment prefix: "identifier = expr" or "identifier= expr".
	// Uses rune indexing throughout to handle multi-byte identifiers correctly.
	// Skips comparison operators (==, !=, <=, >=) which also contain '='.
	exprRunes := runes
	exprCol := col
	for i, r := range runes {
		if r == '=' {
			// Skip comparison operators: ==, !=, <=, >=
			if i+1 < len(runes) && runes[i+1] == '=' {
				continue // == at this position
			}
			if i > 0 && (runes[i-1] == '!' || runes[i-1] == '<' || runes[i-1] == '>') {
				continue // !=, <=, >= at this position
			}
			before := strings.TrimSpace(string(runes[:i]))
			if isIdentifier(before) {
				exprRunes = runes[i+1:]
				exprCol = col - (i + 1)
				if exprCol < 0 {
					return empty
				}
			}
			break
		}
	}

	// Extract the first identifier in the expression.
	exprTrimmed := []rune(strings.TrimLeft(string(exprRunes), " \t"))
	leadingSpaces := len(exprRunes) - len(exprTrimmed)
	funcName := extractLeadingIdentifier(string(exprTrimmed))
	if funcName == "" {
		return empty
	}

	// Look up the function name: first in FunctionSpecs, then synonyms.
	canonicalName := resolveNLFunctionName(funcName)
	if canonicalName == "" {
		return empty
	}

	// Scan for numeric literals in the expression to determine paramIdx.
	// A numeric literal is a sequence of digits with optional decimal point,
	// optionally preceded by $ or € and optionally followed by %.
	literals := findNumericLiterals(exprRunes, leadingSpaces+len([]rune(funcName)))

	if len(literals) == 0 {
		return argumentContext{funcName: canonicalName, paramIdx: 0}
	}

	// Find which literal the cursor is in or most recently past.
	// exprCol is relative to expr start.
	paramIdx := 0
	for i, lit := range literals {
		if exprCol >= lit.start {
			paramIdx = i
		}
	}

	return argumentContext{funcName: canonicalName, paramIdx: paramIdx}
}

// numericLiteral records the rune-index span of a numeric literal within a rune slice.
type numericLiteral struct {
	start int // inclusive, rune index
	end   int // exclusive, rune index
}

// findNumericLiterals scans runes starting from startIdx for numeric literals.
// A numeric literal is digits with optional decimal point, optionally preceded
// by $ or € and optionally followed by %.
func findNumericLiterals(runes []rune, startIdx int) []numericLiteral {
	var lits []numericLiteral
	i := startIdx
	for i < len(runes) {
		// Skip non-numeric runes.
		r := runes[i]

		// Check for currency prefix.
		hasCurrencyPrefix := r == '$' || r == '€'
		digitStart := i
		if hasCurrencyPrefix {
			if i+1 < len(runes) && (unicode.IsDigit(runes[i+1]) || runes[i+1] == '.') {
				i++ // skip currency symbol, digit follows
			} else {
				i++
				continue
			}
		}

		if !unicode.IsDigit(runes[i]) && runes[i] != '.' {
			i++
			continue
		}

		// We're at the start of a potential number. Check that a '.' is followed by digit.
		if runes[i] == '.' {
			if i+1 >= len(runes) || !unicode.IsDigit(runes[i+1]) {
				i++
				continue
			}
		}

		// Consume digits and optional decimal point.
		hasDigit := false
		for i < len(runes) && (unicode.IsDigit(runes[i]) || runes[i] == '.') {
			if unicode.IsDigit(runes[i]) {
				hasDigit = true
			}
			i++
		}

		if !hasDigit {
			continue
		}

		// Optional trailing %.
		end := i
		if i < len(runes) && runes[i] == '%' {
			end = i + 1
			i++
		}

		lits = append(lits, numericLiteral{start: digitStart, end: end})
		continue
	}
	return lits
}

// resolveNLFunctionName checks if name is a known function (in FunctionSpecs)
// or a synonym mapping to a canonical function name.
func resolveNLFunctionName(name string) string {
	lower := strings.ToLower(name)

	// Direct lookup in FunctionSpecs.
	if types.GetFunctionSpec(lower) != nil {
		return lower
	}

	// Check synonyms in the features registry.
	registry := features.DefaultRegistry()
	for _, f := range registry.ByCategory(features.CategoryFunction) {
		for _, syn := range f.Synonyms {
			if strings.EqualFold(syn, name) {
				return f.Name
			}
		}
	}

	return ""
}

// extractKeywordArgumentContext recognizes keyword-operator expressions that
// are NOT function calls and types the operand under the cursor, so variable
// completions there are filtered the same way function arguments are.
//
// Recognized forms (detection is structural — keyed on the operator phrase,
// not on a literal `%` — so it still fires while the user is mid-typing a
// variable into a placeholder, e.g. `myrate of 1000`):
//
//   - `A as % of B` / `A as a % of B` — inverse percentage. Both operands are
//     same-type amounts (currency, quantity, number), so we surface amounts
//     and exclude percentage variables. Checked first because it also
//     contains ` of `.
//   - `A of B` — percentage of a value. Operand A is a percentage; operand B
//     (the base) is left unfiltered.
//   - `A in unit` / `A as unit` — unit conversion. Operand A is a quantity.
//     The unit side is served separately by completionContextAfterUnitKeyword.
//
// Returns ok=false when the line is not one of these forms, or when the cursor
// sits in a position with no meaningful type constraint (so the caller falls
// through to unfiltered completion).
func extractKeywordArgumentContext(lineText string, col int) (argumentContext, bool) {
	runes := []rune(lineText)
	if col > len(runes) {
		col = len(runes)
	}
	lower := []rune(strings.ToLower(lineText))

	// Inverse percentage — both operands are amounts. Check before plain
	// ` of ` since this phrase also contains it.
	for _, phrase := range []string{" as a % of ", " as % of "} {
		if runeIndex(lower, phrase) >= 0 {
			return argumentContext{paramIdx: -1, requiredType: types.ArgTypeAmount}, true
		}
	}

	// Percentage of a value: the operand before ` of ` is a percentage.
	if s := runeIndex(lower, " of "); s >= 0 {
		if col <= s {
			return argumentContext{paramIdx: -1, requiredType: types.ArgTypePercentage}, true
		}
		// Operand B (the base) accepts any value — nothing to filter.
		return argumentContext{}, false
	}

	// Unit conversion: the operand before ` in `/` as ` is a quantity.
	for _, phrase := range []string{" in ", " as "} {
		if s := runeIndex(lower, phrase); s >= 0 {
			if col <= s {
				return argumentContext{paramIdx: -1, requiredType: types.ArgTypeQuantity}, true
			}
			// After the keyword the completion handler is already in the
			// after-unit-keyword context; nothing to type here.
			return argumentContext{}, false
		}
	}

	return argumentContext{}, false
}

// runeIndex returns the rune index of the first occurrence of needle in
// haystack, or -1. Rune-aware so multi-byte text before the match doesn't
// skew the offset (callers compare against a rune-indexed cursor column).
func runeIndex(haystack []rune, needle string) int {
	nr := []rune(needle)
	if len(nr) == 0 || len(nr) > len(haystack) {
		return -1
	}
	for i := 0; i+len(nr) <= len(haystack); i++ {
		match := true
		for j := range nr {
			if haystack[i+j] != nr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// extractLeadingIdentifier returns the first identifier at the start of s.
func extractLeadingIdentifier(s string) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return ""
	}
	if !unicode.IsLetter(runes[0]) && runes[0] != '_' {
		return ""
	}
	end := 1
	for end < len(runes) {
		r := runes[end]
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			end++
			continue
		}
		break
	}
	return string(runes[:end])
}
