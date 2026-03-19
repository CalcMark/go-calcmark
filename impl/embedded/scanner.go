// Package embedded scans standard markdown documents for cm/calcmark fenced
// code blocks and splits the input into passthrough and CalcMark segments.
package embedded

import "strings"

// SegmentKind distinguishes passthrough text from CalcMark blocks.
type SegmentKind int

const (
	// Passthrough represents markdown text that is not a CalcMark block.
	Passthrough SegmentKind = iota
	// CalcMarkBlock represents an extracted CalcMark fenced code block.
	CalcMarkBlock
)

// Segment represents either passthrough markdown text or an extracted CalcMark block.
type Segment struct {
	Kind     SegmentKind
	Text     string // For Passthrough: the raw text (including newlines). For CalcMarkBlock: the block content (without fence lines).
	OpenLine int    // 1-based line number of the opening fence in the host document (0 for Passthrough).
}

// Scan finds cm/calcmark fenced code blocks in the input and returns segments.
// It performs a single-pass O(n) scan over lines.
func Scan(input string) []Segment {
	if input == "" {
		return nil
	}

	lines := splitLines(input)
	var segments []Segment
	var passthroughBuf strings.Builder

	flushPassthrough := func() {
		if passthroughBuf.Len() > 0 {
			segments = append(segments, Segment{
				Kind: Passthrough,
				Text: passthroughBuf.String(),
			})
			passthroughBuf.Reset()
		}
	}

	i := 0
	for i < len(lines) {
		fenceChar, fenceLen, isCM := parseOpeningFence(lines[i])
		if !isCM {
			passthroughBuf.WriteString(lines[i])
			i++
			continue
		}

		// Found a candidate opening fence at line i.
		openLineNum := i + 1 // 1-based
		openIdx := i
		i++ // move past opening fence

		// Collect content lines until matching close.
		var contentBuf strings.Builder
		closed := false
		for i < len(lines) {
			if isMatchingClose(lines[i], fenceChar, fenceLen) {
				closed = true
				i++ // consume the closing fence
				break
			}
			contentBuf.WriteString(lines[i])
			i++
		}

		if !closed {
			// Unclosed fence: treat opening fence + all collected lines as passthrough.
			passthroughBuf.WriteString(lines[openIdx])
			passthroughBuf.WriteString(contentBuf.String())
			continue
		}

		// Emit the CalcMark block.
		flushPassthrough()
		segments = append(segments, Segment{
			Kind:     CalcMarkBlock,
			Text:     contentBuf.String(),
			OpenLine: openLineNum,
		})
	}

	flushPassthrough()
	return segments
}

// splitLines splits input into lines, each including its trailing newline.
// A final line without a newline is included as-is.
func splitLines(input string) []string {
	var lines []string
	for len(input) > 0 {
		idx := strings.IndexByte(input, '\n')
		if idx < 0 {
			lines = append(lines, input)
			break
		}
		lines = append(lines, input[:idx+1])
		input = input[idx+1:]
	}
	return lines
}

// parseOpeningFence checks if a line is a valid cm/calcmark opening fence.
// Returns the fence character, fence length, and whether it's a cm/calcmark fence.
func parseOpeningFence(line string) (byte, int, bool) {
	// Strip trailing newline for parsing.
	trimmed := strings.TrimRight(line, "\n")

	// Count leading spaces (0-3 allowed).
	spaces := 0
	for spaces < len(trimmed) && trimmed[spaces] == ' ' {
		spaces++
	}
	if spaces > 3 {
		return 0, 0, false
	}
	rest := trimmed[spaces:]

	if len(rest) < 3 {
		return 0, 0, false
	}

	fenceChar := rest[0]
	if fenceChar != '`' && fenceChar != '~' {
		return 0, 0, false
	}

	// Count fence characters.
	fenceLen := 0
	for fenceLen < len(rest) && rest[fenceLen] == fenceChar {
		fenceLen++
	}
	if fenceLen < 3 {
		return 0, 0, false
	}

	// Extract info string (everything after fence chars).
	infoString := rest[fenceLen:]
	infoString = strings.TrimLeft(infoString, " \t")

	// First whitespace-delimited token must be exactly "cm" or "calcmark".
	token := infoString
	if idx := strings.IndexAny(token, " \t"); idx >= 0 {
		token = token[:idx]
	}
	if token != "cm" && token != "calcmark" {
		return 0, 0, false
	}

	return fenceChar, fenceLen, true
}

// isMatchingClose checks if a line is a valid closing fence matching the given
// fence character and minimum length.
func isMatchingClose(line string, fenceChar byte, minLen int) bool {
	trimmed := strings.TrimRight(line, "\n")

	// Count leading spaces (0-3 allowed).
	spaces := 0
	for spaces < len(trimmed) && trimmed[spaces] == ' ' {
		spaces++
	}
	if spaces > 3 {
		return false
	}
	rest := trimmed[spaces:]

	if len(rest) < minLen {
		return false
	}
	if len(rest) == 0 || rest[0] != fenceChar {
		return false
	}

	// Count fence characters and ensure only spaces follow.
	count := 0
	for j := 0; j < len(rest); j++ {
		if rest[j] == fenceChar {
			count++
		} else if rest[j] == ' ' {
			// trailing spaces are OK, but only after fence chars
		} else {
			return false
		}
	}
	return count >= minLen
}
