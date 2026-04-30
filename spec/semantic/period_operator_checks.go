// Period-operator inner checks for v2.0 forms:
//   - `length of <P>` / `days in <P>` (LengthOfExpr): inner must
//     type as Period.
//   - `between A and B` (BetweenExpr): both endpoints must type as
//     Date.
//
// Like checkEndOfStartOfInner, these are AST-shape checks (not full
// type inference). Variable-bound inputs (Identifier) defer to
// runtime — the R9-deferred path. NumberLiteral / BooleanLiteral /
// QuantityLiteral / etc. are rejected up-front because their type
// is statically obvious.
package semantic

import (
	"fmt"

	"github.com/CalcMark/go-calcmark/v2/spec/ast"
)

// checkLengthOfInner validates the inner expression of `length of
// <P>` / `days in <P>`. Mirrors checkEndOfStartOfInner. The op
// argument is "length of" or "days in" so the diagnostic message
// names the surface form the user typed.
func (c *Checker) checkLengthOfInner(inner ast.Node, op string, r *ast.Range) {
	if inner == nil {
		c.addDiagnostic(Diagnostic{
			Severity: Error,
			Code:     DiagInvalidLengthOfPeriod,
			Message:  fmt.Sprintf("%s: missing inner expression", op),
			Range:    r,
		})
		return
	}

	switch v := inner.(type) {
	case *ast.RelativeDateLiteral:
		// Period keywords (Q1, this month, this April, FY2027, etc.)
		// → accept. True date keywords (today, tomorrow, yesterday,
		// now) → reject.
		if !isPeriodKeyword(v.Keyword) {
			c.addDiagnostic(Diagnostic{
				Severity: Error,
				Code:     DiagInvalidLengthOfPeriod,
				Message:  fmt.Sprintf("%s requires a period; got Date (%s)", op, v.Keyword),
				Range:    r,
			})
		}
	case *ast.DateLiteral:
		// Specific date (April 15, Apr 1 2026, January 2026 with
		// year). Bare months no longer reach this arm — U5 routes
		// them to RelativeDateLiteral.
		c.addDiagnostic(Diagnostic{
			Severity: Error,
			Code:     DiagInvalidLengthOfPeriod,
			Message:  fmt.Sprintf("%s requires a period; got Date", op),
			Range:    r,
		})
	case *ast.Identifier:
		// Variable-bound: type-checker permits, runtime catches
		// non-Period bindings. Same R9-deferred path as end-of.
	case *ast.NumberLiteral:
		c.addDiagnostic(Diagnostic{
			Severity: Error,
			Code:     DiagInvalidLengthOfPeriod,
			Message:  fmt.Sprintf("%s requires a period; got Number", op),
			Range:    r,
		})
	case *ast.BooleanLiteral:
		c.addDiagnostic(Diagnostic{
			Severity: Error,
			Code:     DiagInvalidLengthOfPeriod,
			Message:  fmt.Sprintf("%s requires a period; got Boolean", op),
			Range:    r,
		})
	case *ast.CurrencyLiteral:
		c.addDiagnostic(Diagnostic{
			Severity: Error,
			Code:     DiagInvalidLengthOfPeriod,
			Message:  fmt.Sprintf("%s requires a period; got Currency", op),
			Range:    r,
		})
	case *ast.QuantityLiteral:
		c.addDiagnostic(Diagnostic{
			Severity: Error,
			Code:     DiagInvalidLengthOfPeriod,
			Message:  fmt.Sprintf("%s requires a period; got Quantity", op),
			Range:    r,
		})
	case *ast.DurationLiteral:
		c.addDiagnostic(Diagnostic{
			Severity: Error,
			Code:     DiagInvalidLengthOfPeriod,
			Message:  fmt.Sprintf("%s requires a period; got Duration", op),
			Range:    r,
		})
	default:
		c.addDiagnostic(Diagnostic{
			Severity: Error,
			Code:     DiagInvalidLengthOfPeriod,
			Message:  fmt.Sprintf("%s requires a period; inner expression is not a period", op),
			Range:    r,
		})
	}
	// Recurse so nested issues still surface (undefined identifiers,
	// invalid date components, etc.).
	c.checkNode(inner)
}

// checkBetweenEndpoint validates one endpoint of a `between A and
// B` / `from A to B` form. Both Start and End must type as Date.
// Variable-bound endpoints (Identifier) defer to runtime; obvious
// non-Date literals are rejected.
//
// labelKind is "start" or "end" so the diagnostic message tells the
// user which endpoint is wrong.
func (c *Checker) checkBetweenEndpoint(node ast.Node, labelKind string, r *ast.Range) {
	if node == nil {
		c.addDiagnostic(Diagnostic{
			Severity: Error,
			Code:     DiagInvalidBetweenEndpoint,
			Message:  fmt.Sprintf("between: missing %s endpoint", labelKind),
			Range:    r,
		})
		return
	}
	switch node.(type) {
	case *ast.DateLiteral, *ast.RelativeDateLiteral, *ast.Identifier, *ast.BinaryOp:
		// Date literals (`Apr 15 2026`), date keywords (`today`,
		// `tomorrow`, `next month` — Period inputs are also accepted
		// here for now since runtime will narrow them to Date),
		// variable-bound (R9-deferred path), and BinaryOp expressions
		// like `today + 30 days` (must reduce to Date at runtime).
		// Runtime catches non-Date results.
	case *ast.NumberLiteral:
		c.addDiagnostic(Diagnostic{
			Severity: Error,
			Code:     DiagInvalidBetweenEndpoint,
			Message:  fmt.Sprintf("between requires Date inputs; got Number for %s endpoint", labelKind),
			Range:    r,
		})
	case *ast.BooleanLiteral:
		c.addDiagnostic(Diagnostic{
			Severity: Error,
			Code:     DiagInvalidBetweenEndpoint,
			Message:  fmt.Sprintf("between requires Date inputs; got Boolean for %s endpoint", labelKind),
			Range:    r,
		})
	case *ast.CurrencyLiteral:
		c.addDiagnostic(Diagnostic{
			Severity: Error,
			Code:     DiagInvalidBetweenEndpoint,
			Message:  fmt.Sprintf("between requires Date inputs; got Currency for %s endpoint", labelKind),
			Range:    r,
		})
	case *ast.QuantityLiteral:
		c.addDiagnostic(Diagnostic{
			Severity: Error,
			Code:     DiagInvalidBetweenEndpoint,
			Message:  fmt.Sprintf("between requires Date inputs; got Quantity for %s endpoint", labelKind),
			Range:    r,
		})
	case *ast.DurationLiteral:
		c.addDiagnostic(Diagnostic{
			Severity: Error,
			Code:     DiagInvalidBetweenEndpoint,
			Message:  fmt.Sprintf("between requires Date inputs; got Duration for %s endpoint", labelKind),
			Range:    r,
		})
	}
	// Recurse so nested issues surface.
	c.checkNode(node)
}
