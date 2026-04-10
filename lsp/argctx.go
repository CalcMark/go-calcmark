package lsp

import "unicode"

// argumentContext describes where a cursor sits relative to an enclosing
// function call. Returned by extractArgumentContext.
//
// When the cursor is not inside any call, funcName is "" and paramIdx is -1.
// When the cursor is inside a string literal within the active argument,
// insideString is true and stringPrefix holds the characters already typed
// inside the string (for prefix-filtering enum completions).
type argumentContext struct {
	funcName     string
	paramIdx     int
	insideString bool
	stringPrefix string
}

// extractArgumentContext scans lineText from the start up to col and reports
// the innermost function call context at the cursor. Uses a forward scan with
// a paren-frame stack because forward scanning is the natural way to track
// string literal state across the line.
//
// The scan is rune-aware for UTF-8 safety. Escaped quotes (`\"`) inside a
// string do not terminate the string.
func extractArgumentContext(lineText string, col int) argumentContext {
	runes := []rune(lineText)
	if col > len(runes) {
		col = len(runes)
	}

	type frame struct {
		funcName     string
		paramIdx     int
		insideString bool
		stringStart  int // rune index just past the opening quote
	}
	var stack []frame

	i := 0
	for i < col {
		r := runes[i]

		if len(stack) > 0 && stack[len(stack)-1].insideString {
			// We're inside a string — consume escape sequences and look for closing quote.
			if r == '\\' && i+1 < col {
				i += 2
				continue
			}
			if r == '"' {
				stack[len(stack)-1].insideString = false
				i++
				continue
			}
			i++
			continue
		}

		switch r {
		case '(':
			// Extract the identifier immediately preceding this paren as the function name.
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
		case '"':
			if len(stack) > 0 {
				stack[len(stack)-1].insideString = true
				stack[len(stack)-1].stringStart = i + 1
			}
		}
		i++
	}

	if len(stack) == 0 {
		return argumentContext{funcName: "", paramIdx: -1}
	}

	top := stack[len(stack)-1]
	if top.funcName == "" {
		return argumentContext{funcName: "", paramIdx: -1}
	}

	ctx := argumentContext{
		funcName:     top.funcName,
		paramIdx:     top.paramIdx,
		insideString: top.insideString,
	}
	if top.insideString {
		ctx.stringPrefix = string(runes[top.stringStart:col])
	}
	return ctx
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
