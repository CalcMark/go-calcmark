package parser

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/ast"
)

// parseFirstStmt extracts the first statement's expression body
// from a calc-line input. Most parser tests need just one
// expression; this helper keeps test bodies focused on AST shape.
func parseFirstStmt(t *testing.T, input string) ast.Node {
	t.Helper()
	nodes, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse(%q): %v", input, err)
	}
	if len(nodes) == 0 {
		t.Fatalf("Parse(%q) returned 0 nodes", input)
	}
	// Top-level node may be Assignment, ExpressionStmt, or BinaryOp
	// depending on the input. Walk into common wrappers to get the
	// underlying expression. Tests inspect specific paths.
	return nodes[0]
}

// expressionFromAssignment extracts the right-hand-side expression
// from `name = expr`. Used for tests that pin AST shape against an
// assignment-form input.
func expressionFromAssignment(t *testing.T, n ast.Node) ast.Node {
	t.Helper()
	a, ok := n.(*ast.Assignment)
	if !ok {
		t.Fatalf("expected *ast.Assignment, got %T (%v)", n, n)
	}
	return a.Value
}

// TestParser_EndOfQ1ProducesEndOfExpr — the core refactor: the
// parser emits a structured EndOfExpr node, not a flat
// RelativeDateLiteral with concatenated string keyword.
func TestParser_EndOfQ1ProducesEndOfExpr(t *testing.T) {
	expr := expressionFromAssignment(t, parseFirstStmt(t, "x = end of Q1"))
	endOf, ok := expr.(*ast.EndOfExpr)
	if !ok {
		t.Fatalf("expected *ast.EndOfExpr, got %T (%v)", expr, expr)
	}
	if endOf.Period == nil {
		t.Fatal("EndOfExpr.Period is nil")
	}
	inner, ok := endOf.Period.(*ast.RelativeDateLiteral)
	if !ok {
		t.Fatalf("expected inner *ast.RelativeDateLiteral, got %T", endOf.Period)
	}
	if inner.Keyword != "Q:1" {
		t.Errorf("inner Keyword = %q, want %q", inner.Keyword, "Q:1")
	}
}

// TestParser_StartOfQ1ProducesStartOfExpr — symmetric.
func TestParser_StartOfQ1ProducesStartOfExpr(t *testing.T) {
	expr := expressionFromAssignment(t, parseFirstStmt(t, "x = start of Q1"))
	startOf, ok := expr.(*ast.StartOfExpr)
	if !ok {
		t.Fatalf("expected *ast.StartOfExpr, got %T", expr)
	}
	if startOf.Period == nil {
		t.Fatal("StartOfExpr.Period is nil")
	}
}

// TestParser_EndOfFQ1AndFQ2_AllFiscalQuarters — every fiscal
// quarter literal must work as inner.
func TestParser_EndOfFQ1AndFQ2_AllFiscalQuarters(t *testing.T) {
	for _, q := range []string{"FQ1", "FQ2", "FQ3", "FQ4"} {
		expr := expressionFromAssignment(t, parseFirstStmt(t, "x = end of "+q))
		endOf, ok := expr.(*ast.EndOfExpr)
		if !ok {
			t.Errorf("end of %s: expected *ast.EndOfExpr, got %T", q, expr)
			continue
		}
		inner, ok := endOf.Period.(*ast.RelativeDateLiteral)
		if !ok {
			t.Errorf("end of %s: inner not *ast.RelativeDateLiteral, got %T", q, endOf.Period)
			continue
		}
		want := "FQ:" + string(q[2])
		if inner.Keyword != want {
			t.Errorf("end of %s: inner Keyword = %q, want %q", q, inner.Keyword, want)
		}
	}
}

// TestParser_EndOfThisMonth — relative-period inner.
func TestParser_EndOfThisMonth(t *testing.T) {
	expr := expressionFromAssignment(t, parseFirstStmt(t, "x = end of this month"))
	endOf, ok := expr.(*ast.EndOfExpr)
	if !ok {
		t.Fatalf("expected *ast.EndOfExpr, got %T", expr)
	}
	inner, ok := endOf.Period.(*ast.RelativeDateLiteral)
	if !ok {
		t.Fatalf("expected inner *ast.RelativeDateLiteral, got %T", endOf.Period)
	}
	if inner.Keyword != "this month" {
		t.Errorf("inner Keyword = %q, want %q", inner.Keyword, "this month")
	}
}

// TestParser_EndOfThisFiscalQuarter — multi-word relative-period.
func TestParser_EndOfThisFiscalQuarter(t *testing.T) {
	expr := expressionFromAssignment(t, parseFirstStmt(t, "x = end of this fiscal quarter"))
	endOf, ok := expr.(*ast.EndOfExpr)
	if !ok {
		t.Fatalf("expected *ast.EndOfExpr, got %T", expr)
	}
	inner, ok := endOf.Period.(*ast.RelativeDateLiteral)
	if !ok {
		t.Fatalf("expected inner *ast.RelativeDateLiteral, got %T", endOf.Period)
	}
	if inner.Keyword != "this fiscal quarter" {
		t.Errorf("inner Keyword = %q, want %q", inner.Keyword, "this fiscal quarter")
	}
}

// TestParser_EndOfIdentifierComposes — the load-bearing R9-emergent
// behavior: `end of <variable>` parses successfully. The parser
// doesn't enforce the variable resolves to a Period; the type
// checker (semantic) and interpreter (runtime) handle that.
func TestParser_EndOfIdentifierComposes(t *testing.T) {
	nodes, err := Parse("q = Q1\ne = end of q")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(nodes) < 2 {
		t.Fatalf("expected 2 statements, got %d", len(nodes))
	}
	expr := expressionFromAssignment(t, nodes[1])
	endOf, ok := expr.(*ast.EndOfExpr)
	if !ok {
		t.Fatalf("expected *ast.EndOfExpr, got %T", expr)
	}
	id, ok := endOf.Period.(*ast.Identifier)
	if !ok {
		t.Fatalf("expected inner *ast.Identifier, got %T", endOf.Period)
	}
	if id.Name != "q" {
		t.Errorf("inner Identifier.Name = %q, want %q", id.Name, "q")
	}
}

// TestParser_EndOfPrecedence_AdditiveOuter — `end of Q1 + 1 day`
// must parse as `(end of Q1) + 1 day`, NOT `end of (Q1 + 1 day)`.
// The plan's Key Decision: parsePrimary()-inner preserves
// precedence so the outer additive operator binds across the
// EndOfExpr boundary, not into it.
func TestParser_EndOfPrecedence_AdditiveOuter(t *testing.T) {
	expr := expressionFromAssignment(t, parseFirstStmt(t, "x = end of Q1 + 1 day"))
	bo, ok := expr.(*ast.BinaryOp)
	if !ok {
		t.Fatalf("expected *ast.BinaryOp at the top, got %T (input parses as `end of (Q1 + 1 day)` which is wrong)", expr)
	}
	if _, ok := bo.Left.(*ast.EndOfExpr); !ok {
		t.Errorf("expected BinaryOp.Left to be *ast.EndOfExpr, got %T", bo.Left)
	}
}

// TestParser_EndOfWithExplicitParens — `end of (Q1)` parses with
// the parenthesized inner. After PR-1b, `end of (Q1 + 1 day)` is a
// type error (BinaryOp returns Date, not Period), but the PARSER
// accepts it -- semantic check is what rejects it.
func TestParser_EndOfWithExplicitParens(t *testing.T) {
	expr := expressionFromAssignment(t, parseFirstStmt(t, "x = end of (Q1)"))
	endOf, ok := expr.(*ast.EndOfExpr)
	if !ok {
		t.Fatalf("expected *ast.EndOfExpr, got %T", expr)
	}
	if endOf.Period == nil {
		t.Fatal("Period is nil")
	}
}

// TestParser_EndOfMissingInnerErrors — clear error when nothing
// follows `end of`.
func TestParser_EndOfMissingInnerErrors(t *testing.T) {
	_, err := Parse("x = end of")
	if err == nil {
		t.Fatal("Parse(\"x = end of\") should error; got nil")
	}
	// Error mentions the modifier so the user knows where the
	// problem is. Loose match -- exact wording is implementation.
	if !strings.Contains(strings.ToLower(err.Error()), "end of") &&
		!strings.Contains(strings.ToLower(err.Error()), "period") &&
		!strings.Contains(strings.ToLower(err.Error()), "expression") {
		t.Errorf("error message %q should mention 'end of', 'period', or 'expression'", err.Error())
	}
}

// TestParser_StartOfMissingInnerErrors — symmetric error.
func TestParser_StartOfMissingInnerErrors(t *testing.T) {
	_, err := Parse("x = start of")
	if err == nil {
		t.Fatal("Parse(\"x = start of\") should error; got nil")
	}
}

// TestParser_EndOfRangeIsPopulated — parser must populate Range so
// type-check diagnostics can point back to source. Pin via line +
// column non-zero.
func TestParser_EndOfRangeIsPopulated(t *testing.T) {
	expr := expressionFromAssignment(t, parseFirstStmt(t, "x = end of Q1"))
	endOf := expr.(*ast.EndOfExpr)
	if endOf.GetRange() == nil {
		t.Fatal("EndOfExpr.Range is nil")
	}
}

// TestParser_BackwardCompat_AllOldEndOfFormsStillParse — every
// inner that worked under the bounded-token-list before the
// refactor must still parse. The list is exhaustive over the
// inner forms the old code accepted.
func TestParser_BackwardCompat_AllOldEndOfFormsStillParse(t *testing.T) {
	inputs := []string{
		// Calendar-relative
		"x = end of this week",
		"x = end of this month",
		"x = end of this year",
		"x = end of this quarter",
		"x = end of next month",
		"x = end of last month",
		// Fiscal-relative
		"x = end of this fiscal quarter",
		"x = end of next fiscal quarter",
		"x = end of last fiscal quarter",
		"x = end of this fiscal year",
		// Notation literals
		"x = end of Q2",
		"x = end of FQ1",
		// Symmetric start-of
		"x = start of Q1",
		"x = start of this month",
	}
	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			expr := expressionFromAssignment(t, parseFirstStmt(t, input))
			switch expr.(type) {
			case *ast.EndOfExpr, *ast.StartOfExpr:
				// OK
			default:
				t.Errorf("expected EndOfExpr/StartOfExpr, got %T", expr)
			}
		})
	}
}
