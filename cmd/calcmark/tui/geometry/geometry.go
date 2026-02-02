package geometry

import "github.com/mattn/go-runewidth"

// RowGeometry describes the visual layout of a single logical row
// when rendered in a two-column layout with text wrapping.
type RowGeometry struct {
	Height     int      // Number of visual lines needed (max of left, right)
	LeftLines  []string // Left column visual lines (padded to Height with empty strings)
	RightLines []string // Right column visual lines (padded to Height with empty strings)
}

// CalculateRowGeometry computes the visual layout for a single logical row.
// srcLine is the source text for the left column.
// resultContent is the result text for the right column (empty string if no result).
// leftWidth is the available width for the left column.
// rightWidth is the available width for the right column.
func CalculateRowGeometry(srcLine, resultContent string, leftWidth, rightWidth int) RowGeometry {
	leftWrapped := WrapText(srcLine, leftWidth)

	var rightWrapped []string
	if resultContent != "" {
		rightWrapped = WrapText(resultContent, rightWidth)
	}

	h := max(len(leftWrapped), len(rightWrapped))
	if h == 0 {
		h = 1
	}

	finalLeft := make([]string, h)
	finalRight := make([]string, h)

	for i := range h {
		if i < len(leftWrapped) {
			finalLeft[i] = leftWrapped[i]
		}
		if i < len(rightWrapped) {
			finalRight[i] = rightWrapped[i]
		}
	}

	return RowGeometry{Height: h, LeftLines: finalLeft, RightLines: finalRight}
}

// WrapText wraps text to fit within maxWidth, preferring word boundaries.
// Returns a slice of strings, each fitting within maxWidth.
// Uses runewidth.StringWidth for correct unicode width handling (CJK, emoji, etc).
// This is a pure function suitable for unit testing.
func WrapText(text string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{text}
	}

	if len(text) == 0 {
		return []string{""}
	}

	if runewidth.StringWidth(text) <= maxWidth {
		return []string{text}
	}

	var result []string
	runes := []rune(text)
	start := 0

	for start < len(runes) {
		// Find how many runes fit within maxWidth
		end := start
		currentWidth := 0
		lastSpaceIdx := -1

		for end < len(runes) {
			rw := runewidth.RuneWidth(runes[end])
			if currentWidth+rw > maxWidth {
				break
			}
			if runes[end] == ' ' {
				lastSpaceIdx = end
			}
			currentWidth += rw
			end++
		}

		// If we've consumed all remaining runes, we're done
		if end >= len(runes) {
			result = append(result, string(runes[start:]))
			break
		}

		// Prefer breaking at word boundary
		if lastSpaceIdx > start {
			// Break after the space
			result = append(result, string(runes[start:lastSpaceIdx+1]))
			start = lastSpaceIdx + 1
		} else if end > start {
			// No space found, hard break
			result = append(result, string(runes[start:end]))
			start = end
		} else {
			// Single character wider than maxWidth - include it anyway
			result = append(result, string(runes[start:start+1]))
			start++
		}
	}

	if len(result) == 0 {
		return []string{text}
	}

	return result
}

// StringWidth returns the visual width of a string, accounting for unicode
// character widths (CJK double-width, emoji, combining characters).
// This is a convenience wrapper around runewidth.StringWidth so callers
// don't need to import the runewidth package directly.
func StringWidth(s string) int {
	return runewidth.StringWidth(s)
}
