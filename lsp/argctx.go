package lsp

import "unicode"

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
		return argumentContext{funcName: "", paramIdx: -1}
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
