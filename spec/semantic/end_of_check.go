// Period-bearing inner-expression check for `end of` / `start of`
// operators. Inline-AST only -- does NOT extend the TypeKind enum.
// See docs/plans (downstream calcmark-web) for the rationale: a new
// TypeKind value would be a public API break for downstream
// consumers that switch on TypeKind without `default`.
package semantic

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/spec/ast"
)

// isPeriodKeyword reports whether the given RelativeDateLiteral
// keyword names a period-producing form (something `end of` /
// `start of` can take). Mirrors evalRelativeDateLiteral's
// period-bearing arms in impl/interpreter/datetime.go -- when a new
// period kind is added there, add it here too.
//
// Non-period date keywords (today, tomorrow, yesterday, now) return
// false. They evaluate to a single Date, not a Period.
func isPeriodKeyword(keyword string) bool {
	kw := strings.ToLower(strings.TrimSpace(keyword))

	// Known relative-period keywords (this/next/last + period unit).
	switch kw {
	case "this week", "next week", "last week",
		"this month", "next month", "last month",
		"this year", "next year", "last year",
		"this quarter", "next quarter", "last quarter",
		"this fiscal quarter", "next fiscal quarter", "last fiscal quarter",
		"this fiscal year", "next fiscal year", "last fiscal year":
		return true
	}

	// Notation forms: Q:N, FQ:N, FY:NNNN, CY:NNNN.
	if strings.HasPrefix(kw, "q:") || strings.HasPrefix(kw, "fq:") ||
		strings.HasPrefix(kw, "fy:") || strings.HasPrefix(kw, "cy:") {
		return true
	}

	// Named months (`this january`, `next april`, etc.). Bare month
	// names without a modifier are not period literals on their own
	// in the lexer; the parser wraps them with `this`. So we only
	// need to recognize the modifier-prefixed forms here.
	for _, prefix := range []string{"this ", "next ", "last "} {
		if strings.HasPrefix(kw, prefix) {
			rest := strings.TrimPrefix(kw, prefix)
			if isMonthName(rest) {
				return true
			}
		}
	}

	return false
}

// isMonthName reports whether the given lowercase string is a
// recognized month name or 3-letter abbreviation. Mirrors the
// `canonicalMonths` map in impl/interpreter/datetime.go.
func isMonthName(s string) bool {
	switch s {
	case "january", "jan",
		"february", "feb",
		"march", "mar",
		"april", "apr",
		"may",
		"june", "jun",
		"july", "jul",
		"august", "aug",
		"september", "sep", "sept",
		"october", "oct",
		"november", "nov",
		"december", "dec":
		return true
	}
	return false
}

// checkEndOfStartOfInner validates that the inner expression of an
// `end of` / `start of` operator is a period-bearing AST shape.
//
// Inline-AST check (no TypeKind enum extension):
//   - *ast.RelativeDateLiteral with isPeriodKeyword(.Keyword) -> OK.
//     Includes Q1-Q4, FQ1-FQ4, FY/CY year literals (which become
//     "FY:NNNN" / "CY:NNNN" keywords from the parser), this/next/last
//     <period>, named-month forms.
//   - *ast.Identifier -> OK at type-check; runtime catches non-Period
//     bindings. Documented as the R9-demoted path -- introducing
//     forward type-inference through assignment is a separate PR.
//   - Anything else -> Diagnostic with the actual node type named
//     in the message.
func (c *Checker) checkEndOfStartOfInner(inner ast.Node, op string, r *ast.Range) {
	if inner == nil {
		c.addDiagnostic(Diagnostic{
			Severity: Error,
			Code:     DiagInvalidEndOfPeriod,
			Message:  fmt.Sprintf("%s: missing inner expression", op),
			Range:    r,
		})
		return
	}

	switch v := inner.(type) {
	case *ast.RelativeDateLiteral:
		if !isPeriodKeyword(v.Keyword) {
			c.addDiagnostic(Diagnostic{
				Severity: Error,
				Code:     DiagInvalidEndOfPeriod,
				Message:  fmt.Sprintf("%s requires a period; got Date (%s)", op, v.Keyword),
				Range:    r,
			})
		}
	case *ast.Identifier:
		// Variable-bound: type-checker permits, runtime catches
		// non-Period values. R9 emergent capability is deferred to
		// a future PR with full *types.Period plumbing.
	case *ast.NumberLiteral:
		c.addDiagnostic(Diagnostic{
			Severity: Error,
			Code:     DiagInvalidEndOfPeriod,
			Message:  fmt.Sprintf("%s requires a period; got Number", op),
			Range:    r,
		})
	case *ast.CurrencyLiteral:
		c.addDiagnostic(Diagnostic{
			Severity: Error,
			Code:     DiagInvalidEndOfPeriod,
			Message:  fmt.Sprintf("%s requires a period; got Currency", op),
			Range:    r,
		})
	case *ast.DateLiteral:
		c.addDiagnostic(Diagnostic{
			Severity: Error,
			Code:     DiagInvalidEndOfPeriod,
			Message:  fmt.Sprintf("%s requires a period; got Date", op),
			Range:    r,
		})
	case *ast.QuantityLiteral:
		c.addDiagnostic(Diagnostic{
			Severity: Error,
			Code:     DiagInvalidEndOfPeriod,
			Message:  fmt.Sprintf("%s requires a period; got Quantity", op),
			Range:    r,
		})
	default:
		c.addDiagnostic(Diagnostic{
			Severity: Error,
			Code:     DiagInvalidEndOfPeriod,
			Message:  fmt.Sprintf("%s requires a period; inner expression is not a period", op),
			Range:    r,
		})
	}
	// Recurse so nested issues still surface (e.g., inner uses an
	// undefined identifier — the existing identifier-check handles
	// that).
	c.checkNode(inner)
}
