package editor

import (
	"strings"
	"unicode"

	"github.com/CalcMark/go-calcmark/spec/semantic"
)

// CursorContext describes what the cursor is positioned on/in.
// This enables contextual help based on where the user is typing.
type CursorContext struct {
	// InFunctionCall is true when cursor is inside parentheses of a function call
	InFunctionCall bool

	// FunctionName is the name of the function being called (if InFunctionCall)
	FunctionName string

	// ArgIndex is the 0-based argument position (based on comma count)
	ArgIndex int

	// ParamSpec is the parameter specification for the current argument
	ParamSpec *semantic.ParamSpec

	// FunctionSpec is the full function specification
	FunctionSpec *semantic.FunctionSpec
}

// GetCursorContext analyzes the current line and cursor position to determine context.
// This is a pure function - give it the line content and cursor column, get context back.
func GetCursorContext(line string, cursorCol int) CursorContext {
	ctx := CursorContext{}

	// Ensure cursor is within bounds
	if cursorCol > len(line) {
		cursorCol = len(line)
	}

	// Only analyze text before cursor
	beforeCursor := line[:cursorCol]

	// Find the innermost unclosed function call
	// Walk backwards to find opening paren
	parenDepth := 0
	funcCallStart := -1

	for i := len(beforeCursor) - 1; i >= 0; i-- {
		ch := beforeCursor[i]
		if ch == ')' {
			parenDepth++
		} else if ch == '(' {
			if parenDepth > 0 {
				parenDepth--
			} else {
				// Found unclosed opening paren
				funcCallStart = i
				break
			}
		}
	}

	if funcCallStart < 0 {
		// Not inside a function call
		return ctx
	}

	// Extract function name (identifier before the opening paren)
	funcName := extractFunctionName(beforeCursor[:funcCallStart])
	if funcName == "" {
		return ctx
	}

	// Look up the function specification
	spec := semantic.GetFunctionSpec(funcName)
	if spec == nil {
		// Unknown function, but still inside a call
		ctx.InFunctionCall = true
		ctx.FunctionName = funcName
		return ctx
	}

	// Count commas to determine argument position
	argsText := beforeCursor[funcCallStart+1:]
	argIndex := countArgumentPosition(argsText)

	// Get parameter spec for this position
	paramSpec := spec.GetParamAtIndex(argIndex)

	ctx.InFunctionCall = true
	ctx.FunctionName = funcName
	ctx.ArgIndex = argIndex
	ctx.FunctionSpec = spec
	ctx.ParamSpec = paramSpec

	return ctx
}

// extractFunctionName extracts the identifier immediately before the text.
// For "foo(bar, " it would return "" (no identifier at end).
// For "accumulate" it would return "accumulate".
func extractFunctionName(text string) string {
	text = strings.TrimRightFunc(text, unicode.IsSpace)
	if len(text) == 0 {
		return ""
	}

	// Walk backwards to find the start of the identifier
	end := len(text)
	start := end
	for start > 0 {
		ch := text[start-1]
		if isIdentChar(ch) {
			start--
		} else {
			break
		}
	}

	if start == end {
		return ""
	}

	return text[start:end]
}

// isIdentChar returns true if the character is valid in an identifier.
func isIdentChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '_'
}

// countArgumentPosition counts the argument position based on commas.
// Handles nested parentheses correctly.
func countArgumentPosition(argsText string) int {
	count := 0
	parenDepth := 0

	for _, ch := range argsText {
		switch ch {
		case '(':
			parenDepth++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
		case ',':
			// Only count commas at depth 0 (not inside nested calls)
			if parenDepth == 0 {
				count++
			}
		}
	}

	return count
}

// FormatParamHelp formats the parameter help for display.
// Returns something like: "rate: 10 MB/s, 100 req/s, 5 GB/day"
func FormatParamHelp(param *semantic.ParamSpec) string {
	if param == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(param.Name)

	if len(param.Examples) > 0 {
		sb.WriteString(": ")
		sb.WriteString(strings.Join(param.Examples, ", "))
	} else {
		// Fall back to type examples
		typeExamples := semantic.GetExamplesForType(param.Type)
		if len(typeExamples) > 0 {
			sb.WriteString(": ")
			sb.WriteString(strings.Join(typeExamples, ", "))
		}
	}

	return sb.String()
}
