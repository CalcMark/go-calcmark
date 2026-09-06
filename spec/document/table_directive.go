package document

import (
	"regexp"
	"strings"
	"unicode"
)

// Named-table directive (go-calcmark#118):
//
//	<!-- table: rates (role, rate, hc) -->
//
// The spec layer only needs to recognise the directive and normalize its
// names — the detector treats the table name as a known name so
// `rates.rate * 2` classifies as a calculation. Extraction of the table's
// data happens in the implementation layer during evaluation.

// TableDirectivePattern captures the raw table name and the parenthesised
// column list of a directive line.
var TableDirectivePattern = regexp.MustCompile(`^\s*<!--\s*table:\s*([^()]*?)\s*\(([^)]*)\)\s*-->\s*$`)

// TableDirectiveName returns the normalized table name declared on a
// directive line, or "" and false when the line is not a directive.
func TableDirectiveName(line string) (string, bool) {
	if !strings.Contains(line, "<!--") {
		return "", false
	}
	m := TableDirectivePattern.FindStringSubmatch(line)
	if m == nil {
		return "", false
	}
	return NormalizeTableName(m[1]), true
}

// NormalizeTableName lowercases a directive name and turns runs of
// whitespace into single underscores. Nothing else is stripped: if the
// result is not a valid identifier the directive is rejected downstream.
func NormalizeTableName(raw string) string {
	fields := strings.FieldsFunc(strings.TrimSpace(raw), unicode.IsSpace)
	return strings.ToLower(strings.Join(fields, "_"))
}
