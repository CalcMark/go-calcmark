package parser

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/ast"
)

// U5 — bare month names parse to a Period-bearing AST.
//
// Pre-v2.0, `April` lexes as DATE_LITERAL("April:1:") and the parser
// emits ast.DateLiteral{Month: "April", Day: "1", Year: nil}. PR-1b's
// `end of <Period>` check then needed a transitional shim to accept
// this DateLiteral as period-bearing.
//
// v2.0 fixes the discrimination at the parser: the lexer threads
// HasExplicitDay (whether a day number was scanned) into the token,
// and the parser routes:
//
//   - HasExplicitDay = false (bare month: `April`, `Apr`, `january`)
//     → RelativeDateLiteral{Keyword: "this <Month>"}.
//     This is the same AST shape as `this April`, so the existing
//     period-keyword evaluation path handles it directly.
//
//   - HasExplicitDay = true (specific date: `April 1`, `April 15
//     2026`) → DateLiteral{HasExplicitDay: true, ...}, unchanged.
//
//   - Year-only (`April 2026`, no explicit day) → DateLiteral
//     (status quo). Period-of-year-month is a future feature; for
//     now it remains a Date.
//
// Discrimination must be lexer-driven, not source-text substring
// matching — the latter drifts with whitespace, comments, and
// future lexer changes (reviewer F1).

func TestParser_BareMonthName_ProducesRelativeDateLiteral(t *testing.T) {
	// The lexer canonicalizes month names ("Apr" → "April", "sept"
	// → "September"), so the parser-emitted Keyword is always the
	// canonical form regardless of how the user typed it. This is
	// the same canonicalization that `this April` / `this Sep`
	// already go through; the parser routes both inputs to the same
	// keyword string.
	cases := []struct {
		input       string
		wantKeyword string
	}{
		{"April", "this April"},
		{"Apr", "this April"},
		{"january", "this January"},
		{"Dec", "this December"},
		{"sept", "this September"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			n := parseFirstStmt(t, "x = "+tc.input)
			expr := expressionFromAssignment(t, n)
			rdl, ok := expr.(*ast.RelativeDateLiteral)
			if !ok {
				t.Fatalf("input %q: expected *ast.RelativeDateLiteral, got %T (%v)", tc.input, expr, expr)
			}
			if rdl.Keyword != tc.wantKeyword {
				t.Errorf("input %q: Keyword = %q, want %q", tc.input, rdl.Keyword, tc.wantKeyword)
			}
		})
	}
}

func TestParser_SpecificDate_ProducesDateLiteralWithExplicitDay(t *testing.T) {
	cases := []struct {
		input    string
		wantDay  string
		wantYear *string
	}{
		{"April 1", "1", nil},
		{"April 15", "15", nil},
		{"Apr 1 2026", "1", strptr("2026")},
		{"April 15 2026", "15", strptr("2026")},
		{"December 25 2026", "25", strptr("2026")},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			n := parseFirstStmt(t, "x = "+tc.input)
			expr := expressionFromAssignment(t, n)
			dl, ok := expr.(*ast.DateLiteral)
			if !ok {
				t.Fatalf("input %q: expected *ast.DateLiteral, got %T (%v)", tc.input, expr, expr)
			}
			if !dl.HasExplicitDay {
				t.Errorf("input %q: HasExplicitDay = false, want true (specific date had a day number)", tc.input)
			}
			if dl.Day != tc.wantDay {
				t.Errorf("input %q: Day = %q, want %q", tc.input, dl.Day, tc.wantDay)
			}
			if (tc.wantYear == nil) != (dl.Year == nil) {
				t.Errorf("input %q: Year nil mismatch: got %v, want %v", tc.input, dl.Year, tc.wantYear)
			}
			if tc.wantYear != nil && dl.Year != nil && *dl.Year != *tc.wantYear {
				t.Errorf("input %q: Year = %q, want %q", tc.input, *dl.Year, *tc.wantYear)
			}
		})
	}
}

// TestParser_YearOnlyMonth_StaysAsDateLiteral — `January 2026`
// (year scanned, no explicit day) preserves status-quo as
// DateLiteral. Period-of-year-month is out of scope for U5 — a
// later PR can route this to a MonthOfYear period node if the
// product calls for it.
func TestParser_YearOnlyMonth_StaysAsDateLiteral(t *testing.T) {
	cases := []string{"January 2026", "April 2026", "Dec 2025"}
	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			n := parseFirstStmt(t, "x = "+input)
			expr := expressionFromAssignment(t, n)
			dl, ok := expr.(*ast.DateLiteral)
			if !ok {
				t.Fatalf("input %q: expected *ast.DateLiteral, got %T (%v)", input, expr, expr)
			}
			if dl.HasExplicitDay {
				t.Errorf("input %q: HasExplicitDay = true, want false (no day number was scanned)", input)
			}
			if dl.Year == nil {
				t.Errorf("input %q: Year nil, expected the year to be captured", input)
			}
		})
	}
}

// TestParser_BareMonthFlowsThroughEndOf — integration: with U5,
// `end of April` parses as EndOfExpr{Period: RelativeDateLiteral},
// which is the same shape as `end of this April`. This eliminates
// the transitional *ast.DateLiteral arm in spec/semantic (deferred
// to U9 to actually remove the shim).
func TestParser_BareMonthFlowsThroughEndOf(t *testing.T) {
	n := parseFirstStmt(t, "x = end of April")
	expr := expressionFromAssignment(t, n)
	endOf, ok := expr.(*ast.EndOfExpr)
	if !ok {
		t.Fatalf("expected *ast.EndOfExpr, got %T (%v)", expr, expr)
	}
	rdl, ok := endOf.Period.(*ast.RelativeDateLiteral)
	if !ok {
		t.Fatalf("end of April: inner expected *ast.RelativeDateLiteral, got %T (%v)", endOf.Period, endOf.Period)
	}
	if rdl.Keyword != "this April" {
		t.Errorf("end of April: inner Keyword = %q, want %q", rdl.Keyword, "this April")
	}
}

func strptr(s string) *string { return &s }
