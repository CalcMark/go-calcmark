package document

import (
	"strings"
	"unicode"
)

// splitLines splits text into lines, handling all Unicode line terminators:
// - LF (\n) - Unix
// - CRLF (\r\n) - Windows
// - CR (\r) - Old Mac
// - U+2028 (Line Separator)
// - U+2029 (Paragraph Separator)
//
// This is critical for WASM/JS interop where code-point handling differs.
func splitLines(text string) []string {
	if text == "" {
		return []string{}
	}

	lines := []string{}
	currentLine := strings.Builder{}
	runes := []rune(text) // Work with runes, not bytes

	for i := 0; i < len(runes); i++ {
		r := runes[i]

		switch r {
		case '\r':
			// Check for CRLF
			if i+1 < len(runes) && runes[i+1] == '\n' {
				// CRLF - skip the \r, \n will be handled next iteration
				lines = append(lines, currentLine.String())
				currentLine.Reset()
				i++ // Skip the \n
			} else {
				// CR alone
				lines = append(lines, currentLine.String())
				currentLine.Reset()
			}

		case '\n':
			// LF
			lines = append(lines, currentLine.String())
			currentLine.Reset()

		case '\u2028', '\u2029':
			// Unicode line/paragraph separators
			lines = append(lines, currentLine.String())
			currentLine.Reset()

		default:
			currentLine.WriteRune(r)
		}
	}

	// Don't forget last line if no trailing newline
	if currentLine.Len() > 0 || len(runes) > 0 {
		lines = append(lines, currentLine.String())
	}

	return lines
}

// isEmptyLine checks if a line is empty (only whitespace).
// Unicode-aware: handles all Unicode whitespace.
func isEmptyLine(line string) bool {
	for _, r := range line {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}
