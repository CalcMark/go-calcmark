// Package constants provides common constants used throughout go-calcmark
package constants

import "strings"

// Character literals used in parsing and classification
const (
	// Whitespace characters (per ENCODING_SPEC.md Section 2)
	// Only these 4 Unicode code points are considered whitespace:
	// - U+0020 (Space)
	// - U+0009 (Tab)
	// - U+000D (Carriage Return)
	// - U+000A (Line Feed)
	Tab     = "\t"
	Newline = "\n"
	Space   = " "

	// Common whitespace string for Trim operations
	// Includes all CalcMark whitespace characters
	Whitespace = " \t\r\n"
)

// TrimSpace removes CalcMark whitespace from both ends of a string.
// Unlike strings.TrimSpace(), this only removes the 4 whitespace characters
// defined in ENCODING_SPEC.md, not all Unicode whitespace (category Z).
func TrimSpace(s string) string {
	return strings.Trim(s, Whitespace)
}

// IsBlankLine returns true if the line contains only CalcMark whitespace
// or is empty.
func IsBlankLine(line string) bool {
	return line == "" || TrimSpace(line) == ""
}
