package lexer

import "unicode"

// tryParseFraction checks if the current position in the rune slice starts a fraction literal.
// A fraction is `digits/digits` where:
//   - The numerator has already been consumed (integerPart)
//   - The current position is at '/'
//   - '/' is immediately followed by one or more digits (no space)
//   - The denominator is not followed by an identifier char (prevents matching "100/3cup")
//
// This is a pure function with no lexer state dependency (Rob Pike pattern).
// Returns the denominator digits, total runes consumed (including '/'), and whether a fraction was found.
func tryParseFraction(text []rune, pos int, integerPart string) (denomStr string, consumed int, ok bool) {
	// Must have a '/' at current position
	if pos >= len(text) || text[pos] != '/' {
		return "", 0, false
	}

	// The integer part must not contain a decimal point — "1.5/3" is not a fraction
	for _, r := range integerPart {
		if r == '.' {
			return "", 0, false
		}
	}

	// '/' must be immediately followed by at least one digit (no space)
	nextPos := pos + 1
	if nextPos >= len(text) || !unicode.IsDigit(text[nextPos]) {
		return "", 0, false
	}

	// Read denominator digits
	denomStart := nextPos
	for nextPos < len(text) && unicode.IsDigit(text[nextPos]) {
		nextPos++
	}

	// Denominator must not be immediately followed by an identifier char
	// This prevents "1/3cup" from being a fraction (requires space: "1/3 cup")
	if nextPos < len(text) && isIdentStartChar(text[nextPos]) {
		return "", 0, false
	}

	// Also reject if followed by '/' — "1/3/4" should not parse first part as fraction
	if nextPos < len(text) && text[nextPos] == '/' {
		return "", 0, false
	}

	denomStr = string(text[denomStart:nextPos])
	consumed = nextPos - pos // includes '/' and denominator digits
	return denomStr, consumed, true
}

// isIdentStartChar checks if a rune can start an identifier.
func isIdentStartChar(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}
