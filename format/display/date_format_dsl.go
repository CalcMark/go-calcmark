package display

import (
	"strings"
	"time"

	"github.com/goodsign/monday"
)

// User-facing date format DSL. Translates a small set of common
// tokens to Go's reference layout so users don't have to memorize
// `2006-01-02 15:04:05`.
//
// Tokens (longest match first; literal characters pass through):
//
//   YYYY  4-digit year                2006
//   YY    2-digit year                06
//   MMMM  Full month name (locale)    January / janvier / Januar
//   MMM   Short month name (locale)   Jan / janv. / Jan.
//   MON   Short month, UPPERCASE      JAN / JANV. / JAN.
//   mon   Short month, lowercase      jan / janv. / jan.
//   MM    2-digit month number        01
//   M     Month number (no pad)       1
//   dd    2-digit day                 02
//   d     Day (no pad)                2
//   EEEE  Full weekday (locale)       Monday / lundi / Montag
//   EEE   Short weekday (locale)      Mon / lun. / Mo.
//
// Month-name case transforms (MON / mon) are applied after monday
// formats the locale name — so "MON" with locale fr_FR for April
// returns "AVR." (locale's abbreviation, uppercased).
//
// Anything else passes through literally. Quote any letter you
// want preserved as-is by surrounding it with single quotes — e.g.
// "'T'YYYY" emits "T2026".
//
// Returns the Go layout (suitable for monday.Format) and a post-
// processor that handles MON / mon case transforms.

// upperMonthMarker / lowerMonthMarker bracket month-name spans in
// the layout that need post-format case conversion. They're ASCII
// control characters that won't appear in normal date output.
const (
	upperMonthOpen  = "\x02"
	upperMonthClose = "\x03"
	lowerMonthOpen  = "\x04"
	lowerMonthClose = "\x05"
)

// translateUserDateFormat converts a DSL string to a Go time
// layout + post-processor. Returns the original string unchanged
// (and an identity post-processor) if the input is empty.
func translateUserDateFormat(dsl string) (layout string, postProcess func(string) string) {
	if dsl == "" {
		return "", func(s string) string { return s }
	}

	var b strings.Builder
	hasUpper := false
	hasLower := false

	i := 0
	for i < len(dsl) {
		// Quoted literal: 'X' -> X (skip the quotes)
		if dsl[i] == '\'' {
			j := i + 1
			for j < len(dsl) && dsl[j] != '\'' {
				b.WriteByte(dsl[j])
				j++
			}
			if j < len(dsl) {
				j++ // skip closing quote
			}
			i = j
			continue
		}

		// Token table — longest match first.
		switch {
		case hasPrefix(dsl, i, "YYYY"):
			b.WriteString("2006")
			i += 4
		case hasPrefix(dsl, i, "YY"):
			b.WriteString("06")
			i += 2
		case hasPrefix(dsl, i, "MMMM"):
			b.WriteString("January")
			i += 4
		case hasPrefix(dsl, i, "MMM"):
			b.WriteString("Jan")
			i += 3
		case hasPrefix(dsl, i, "MON"):
			b.WriteString(upperMonthOpen + "Jan" + upperMonthClose)
			hasUpper = true
			i += 3
		case hasPrefix(dsl, i, "mon"):
			b.WriteString(lowerMonthOpen + "Jan" + lowerMonthClose)
			hasLower = true
			i += 3
		case hasPrefix(dsl, i, "MM"):
			b.WriteString("01")
			i += 2
		case hasPrefix(dsl, i, "M"):
			// Don't match if the next char is one we'd otherwise
			// pair with (already-handled longer token caught above).
			b.WriteString("1")
			i++
		case hasPrefix(dsl, i, "dd"):
			b.WriteString("02")
			i += 2
		case hasPrefix(dsl, i, "d"):
			b.WriteString("2")
			i++
		case hasPrefix(dsl, i, "EEEE"):
			b.WriteString("Monday")
			i += 4
		case hasPrefix(dsl, i, "EEE"):
			b.WriteString("Mon")
			i += 3
		default:
			b.WriteByte(dsl[i])
			i++
		}
	}

	layout = b.String()
	if !hasUpper && !hasLower {
		return layout, func(s string) string { return s }
	}
	return layout, func(s string) string {
		if hasUpper {
			s = applyCaseBetweenMarkers(s, upperMonthOpen, upperMonthClose, strings.ToUpper)
		}
		if hasLower {
			s = applyCaseBetweenMarkers(s, lowerMonthOpen, lowerMonthClose, strings.ToLower)
		}
		return s
	}
}

func hasPrefix(s string, i int, prefix string) bool {
	return i+len(prefix) <= len(s) && s[i:i+len(prefix)] == prefix
}

// applyCaseBetweenMarkers walks s, finds open/close marker pairs,
// applies transform to the substring between them, and strips the
// markers. Pairs that span the entire string or are unpaired are
// left as-is (markers stripped).
func applyCaseBetweenMarkers(s, open, close string, transform func(string) string) string {
	var b strings.Builder
	for {
		i := strings.Index(s, open)
		if i < 0 {
			b.WriteString(s)
			return b.String()
		}
		b.WriteString(s[:i])
		rest := s[i+len(open):]
		j := strings.Index(rest, close)
		if j < 0 {
			b.WriteString(rest)
			return b.String()
		}
		b.WriteString(transform(rest[:j]))
		s = rest[j+len(close):]
	}
}

// formatDateWithDSL formats t using a user DSL string. Empty DSL
// returns "" so callers can detect "no override" and fall back to
// locale defaults. Locale flows through monday for month / weekday
// names.
func formatDateWithDSL(t time.Time, dsl string, locale monday.Locale) string {
	if dsl == "" {
		return ""
	}
	layout, postProcess := translateUserDateFormat(dsl)
	return postProcess(monday.Format(t, layout, locale))
}
