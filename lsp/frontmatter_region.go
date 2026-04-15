package lsp

import (
	"strings"

	protocol "github.com/tliron/glsp/protocol_3_16"
)

// CursorPosition classifies where in the frontmatter region a cursor sits.
// String-typed for diagnostic readability per D1 (pragmatism).
type CursorPosition string

const (
	CursorOutside CursorPosition = "outside"
	CursorFence   CursorPosition = "fence"
	CursorKey     CursorPosition = "key"
	CursorValue   CursorPosition = "value"
)

// FrontmatterRegion describes the YAML frontmatter region delimited by `---`
// fences at the top of a CalcMark source. Lines are 0-based to match LSP.
type FrontmatterRegion struct {
	StartLine int            // line of opening `---`
	EndLine   int            // line of closing `---`
	KeyLines  map[int]string // line index → top-level key name
	// lines is the verbatim content of the region (StartLine..EndLine inclusive)
	// indexed by absolute line number. Used by ClassifyCursor; unexported because
	// callers should treat the region as opaque.
	lines map[int]string
}

// CursorContext is the result of classifying a cursor against a region.
type CursorContext struct {
	InRegion bool
	Position CursorPosition
	Key      string // name of the key the cursor relates to ("" when unknown)
}

// DetectRegion finds the YAML frontmatter region in source. It is mid-edit
// robust: it does NOT parse YAML — it only locates the fences and extracts
// top-level key names from `key:` patterns. Returns ok=false when the source
// does not begin with a `---\n` fence or when no closing fence is present.
//
// Complexity: O(n) over the bytes up to (and including) the closing fence.
func DetectRegion(source string) (FrontmatterRegion, bool) {
	if !strings.HasPrefix(source, "---\n") && source != "---\r\n" && !strings.HasPrefix(source, "---\r\n") {
		return FrontmatterRegion{}, false
	}

	lines := strings.Split(source, "\n")
	if len(lines) < 2 {
		return FrontmatterRegion{}, false
	}
	// Confirm line 0 is exactly `---` (allow trailing CR).
	if strings.TrimRight(lines[0], "\r") != "---" {
		return FrontmatterRegion{}, false
	}

	endLine := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], "\r") == "---" {
			endLine = i
			break
		}
	}
	if endLine == -1 {
		return FrontmatterRegion{}, false
	}

	region := FrontmatterRegion{
		StartLine: 0,
		EndLine:   endLine,
		KeyLines:  make(map[int]string),
		lines:     make(map[int]string, endLine+1),
	}
	for i := 0; i <= endLine; i++ {
		region.lines[i] = lines[i]
	}
	for i := 1; i < endLine; i++ {
		if name, ok := topLevelKey(lines[i]); ok {
			region.KeyLines[i] = name
		}
	}
	return region, true
}

// topLevelKey extracts the key name from a `key: ...` line at zero indentation.
// Returns ok=false for indented lines, comments, blank lines, or lines without
// a `:` separator.
func topLevelKey(line string) (string, bool) {
	trimmedRight := strings.TrimRight(line, "\r")
	if trimmedRight == "" {
		return "", false
	}
	// Top-level keys have NO leading whitespace.
	if trimmedRight[0] == ' ' || trimmedRight[0] == '\t' {
		return "", false
	}
	if trimmedRight[0] == '#' {
		return "", false
	}
	colon := strings.IndexByte(trimmedRight, ':')
	if colon <= 0 {
		return "", false
	}
	name := trimmedRight[:colon]
	// Reject names with internal whitespace — not a valid YAML mapping key shape we recognize.
	if strings.ContainsAny(name, " \t") {
		return "", false
	}
	return name, true
}

// ClassifyCursor determines where the cursor at position sits relative to the
// region: outside, on a fence, on a key, or in a value (including indented
// continuations of the most recent top-level key).
//
// Edge case: if the cursor sits exactly on the `:` separator column, it is
// classified as "key" — the user has not yet stepped into the value.
//
// Complexity: O(EndLine) worst case when scanning back for a parent key; for
// well-formed frontmatter this is bounded by the region size.
func ClassifyCursor(region FrontmatterRegion, position protocol.Position) CursorContext {
	line := int(position.Line)
	if line < region.StartLine || line > region.EndLine {
		return CursorContext{InRegion: false, Position: CursorOutside}
	}
	if line == region.StartLine || line == region.EndLine {
		return CursorContext{InRegion: true, Position: CursorFence}
	}

	col := int(position.Character)
	text := region.lines[line]

	// Top-level key line: cursor at-or-before colon → key; after colon → value.
	if name, ok := region.KeyLines[line]; ok {
		colon := strings.IndexByte(strings.TrimRight(text, "\r"), ':')
		if colon < 0 || col <= colon {
			return CursorContext{InRegion: true, Position: CursorKey, Key: name}
		}
		return CursorContext{InRegion: true, Position: CursorValue, Key: name}
	}

	// Blank line → key-position with no name (author about to type a key).
	if strings.TrimSpace(text) == "" {
		return CursorContext{InRegion: true, Position: CursorKey, Key: ""}
	}

	// Indented continuation → value of the most recent top-level key above.
	if len(text) > 0 && (text[0] == ' ' || text[0] == '\t') {
		return CursorContext{InRegion: true, Position: CursorValue, Key: parentKey(region, line)}
	}

	// Fallback: a non-blank, non-indented line we didn't classify as a top-level
	// key (e.g., garbled mid-edit input). Treat as key-position with no name.
	return CursorContext{InRegion: true, Position: CursorKey, Key: ""}
}

// parentKey scans backward from line-1 to find the nearest top-level key in
// region.KeyLines. Returns "" if none found.
func parentKey(region FrontmatterRegion, line int) string {
	for i := line - 1; i > region.StartLine; i-- {
		if name, ok := region.KeyLines[i]; ok {
			return name
		}
	}
	return ""
}
