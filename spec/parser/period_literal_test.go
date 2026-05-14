package parser

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/ast"
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

// --- U7: BetweenExpr productions ---

func TestParser_Between_HappyPath(t *testing.T) {
	n := parseFirstStmt(t, "x = between Apr 15 2026 and Jul 4 2026")
	expr := expressionFromAssignment(t, n)
	be, ok := expr.(*ast.BetweenExpr)
	if !ok {
		t.Fatalf("expected *ast.BetweenExpr, got %T (%v)", expr, expr)
	}
	if _, ok := be.Start.(*ast.DateLiteral); !ok {
		t.Errorf("Start = %T, want *ast.DateLiteral", be.Start)
	}
	if _, ok := be.End.(*ast.DateLiteral); !ok {
		t.Errorf("End = %T, want *ast.DateLiteral", be.End)
	}
}

func TestParser_FromTo_Synonym(t *testing.T) {
	n := parseFirstStmt(t, "x = from Apr 15 2026 to Jul 4 2026")
	expr := expressionFromAssignment(t, n)
	be, ok := expr.(*ast.BetweenExpr)
	if !ok {
		t.Fatalf("expected *ast.BetweenExpr, got %T (%v)", expr, expr)
	}
	if _, ok := be.Start.(*ast.DateLiteral); !ok {
		t.Errorf("Start = %T, want *ast.DateLiteral", be.Start)
	}
	if _, ok := be.End.(*ast.DateLiteral); !ok {
		t.Errorf("End = %T, want *ast.DateLiteral", be.End)
	}
}

// TestParser_BetweenAndFromTo_ASTEquivalent — both surface forms
// must produce structurally identical AST trees. Catches subtle
// parser-side divergence (different Range, swapped fields) that
// semantic + eval would silently absorb.
func TestParser_BetweenAndFromTo_ASTEquivalent(t *testing.T) {
	a := parseFirstStmt(t, "x = between Apr 15 2026 and Jul 4 2026")
	b := parseFirstStmt(t, "x = from Apr 15 2026 to Jul 4 2026")

	beA, ok := expressionFromAssignment(t, a).(*ast.BetweenExpr)
	if !ok {
		t.Fatalf("`between ...`: expected BetweenExpr, got %T", expressionFromAssignment(t, a))
	}
	beB, ok := expressionFromAssignment(t, b).(*ast.BetweenExpr)
	if !ok {
		t.Fatalf("`from ... to ...`: expected BetweenExpr, got %T", expressionFromAssignment(t, b))
	}
	// Compare Start/End via String() — full deep-equal would fail
	// on Range positions which differ legitimately by source offset.
	if beA.Start.String() != beB.Start.String() {
		t.Errorf("Start mismatch:\n  between: %s\n  from-to: %s", beA.Start.String(), beB.Start.String())
	}
	if beA.End.String() != beB.End.String() {
		t.Errorf("End mismatch:\n  between: %s\n  from-to: %s", beA.End.String(), beB.End.String())
	}
}

func TestParser_Between_RelativeDates(t *testing.T) {
	n := parseFirstStmt(t, "x = between today and next month")
	expr := expressionFromAssignment(t, n)
	be, ok := expr.(*ast.BetweenExpr)
	if !ok {
		t.Fatalf("expected *ast.BetweenExpr, got %T", expr)
	}
	if _, ok := be.Start.(*ast.RelativeDateLiteral); !ok {
		t.Errorf("Start = %T, want *ast.RelativeDateLiteral (today)", be.Start)
	}
	if _, ok := be.End.(*ast.RelativeDateLiteral); !ok {
		t.Errorf("End = %T, want *ast.RelativeDateLiteral (next month)", be.End)
	}
}

func TestParser_Between_MissingAnd(t *testing.T) {
	_, err := Parse("x = between Apr 15 2026")
	if err == nil {
		t.Fatal("expected parse error for `between Apr 15 2026` (missing `and`)")
	}
}

func TestParser_Between_AtEOF(t *testing.T) {
	_, err := Parse("x = between")
	if err == nil {
		t.Fatal("expected parse error for `x = between` (no operands)")
	}
}

func TestParser_FromTo_MissingEnd(t *testing.T) {
	_, err := Parse("x = from Apr 15 2026 to")
	if err == nil {
		t.Fatal("expected parse error for `from Apr 15 2026 to` (missing End)")
	}
}

// TestParser_BetweenAsIdentifier_MigrationDiagnostic — `between = 50`
// is the v2.0 breaking-change scenario. The parser must surface a
// clear migration message rather than crashing or silently
// accepting it.
func TestParser_BetweenAsIdentifier_MigrationDiagnostic(t *testing.T) {
	_, err := Parse("between = 50")
	if err == nil {
		t.Fatal("expected migration diagnostic for `between = 50`")
	}
	msg := err.Error()
	if !strings.Contains(strings.ToLower(msg), "between") {
		t.Errorf("error %q should mention `between`", msg)
	}
	if !strings.Contains(strings.ToLower(msg), "reserved") &&
		!strings.Contains(strings.ToLower(msg), "keyword") {
		t.Errorf("error %q should explain that `between` is now a reserved keyword", msg)
	}
}

// TestParser_DurationFromArith_NotABetween — the existing
// `2 days from today` arithmetic must keep working. The new
// `from A to B` parser branch fires only when FROM is at the start
// of a primary expression, not after a duration.
func TestParser_DurationFromArith_NotABetween(t *testing.T) {
	n := parseFirstStmt(t, "x = 2 days from today")
	expr := expressionFromAssignment(t, n)
	if _, ok := expr.(*ast.BetweenExpr); ok {
		t.Errorf("`2 days from today` must not parse as BetweenExpr; got %T", expr)
	}
}

// --- U8: LengthOfExpr productions ---

func TestParser_LengthOf_BareMonth(t *testing.T) {
	n := parseFirstStmt(t, "x = length of April")
	expr := expressionFromAssignment(t, n)
	loe, ok := expr.(*ast.LengthOfExpr)
	if !ok {
		t.Fatalf("expected *ast.LengthOfExpr, got %T", expr)
	}
	if loe.AsNumber {
		t.Errorf("AsNumber = true, want false (`length of` form)")
	}
	rdl, ok := loe.Period.(*ast.RelativeDateLiteral)
	if !ok {
		t.Fatalf("Period inner = %T, want *ast.RelativeDateLiteral (April → this April)", loe.Period)
	}
	if rdl.Keyword != "this April" {
		t.Errorf("Period.Keyword = %q, want %q", rdl.Keyword, "this April")
	}
}

func TestParser_DaysIn_BareMonth(t *testing.T) {
	n := parseFirstStmt(t, "x = days in April")
	expr := expressionFromAssignment(t, n)
	loe, ok := expr.(*ast.LengthOfExpr)
	if !ok {
		t.Fatalf("expected *ast.LengthOfExpr, got %T", expr)
	}
	if !loe.AsNumber {
		t.Errorf("AsNumber = false, want true (`days in` form)")
	}
}

// TestParser_LengthOf_PrecedenceLockIn — `length of Q1 + 2 days`
// must parse as `(length of Q1) + 2 days`, top-level BinaryOp.
// Pinned at the AST level so the contract is locked even if the
// interpreter or formatter accidentally cancels out a precedence
// bug at runtime.
func TestParser_LengthOf_PrecedenceLockIn(t *testing.T) {
	n := parseFirstStmt(t, "x = length of Q1 + 2 days")
	expr := expressionFromAssignment(t, n)
	bop, ok := expr.(*ast.BinaryOp)
	if !ok {
		t.Fatalf("expected top-level *ast.BinaryOp (length of Q1 + 2 days), got %T", expr)
	}
	if bop.Operator != "+" {
		t.Errorf("Operator = %q, want %q", bop.Operator, "+")
	}
	if _, ok := bop.Left.(*ast.LengthOfExpr); !ok {
		t.Errorf("Left = %T, want *ast.LengthOfExpr", bop.Left)
	}
	if _, ok := bop.Right.(*ast.DurationLiteral); !ok {
		t.Errorf("Right = %T, want *ast.DurationLiteral", bop.Right)
	}
}

func TestParser_DaysIn_PrecedenceLockIn(t *testing.T) {
	n := parseFirstStmt(t, "x = days in April * 100")
	expr := expressionFromAssignment(t, n)
	bop, ok := expr.(*ast.BinaryOp)
	if !ok {
		t.Fatalf("expected top-level *ast.BinaryOp, got %T", expr)
	}
	if bop.Operator != "*" {
		t.Errorf("Operator = %q, want %q", bop.Operator, "*")
	}
	loe, ok := bop.Left.(*ast.LengthOfExpr)
	if !ok {
		t.Fatalf("Left = %T, want *ast.LengthOfExpr", bop.Left)
	}
	if !loe.AsNumber {
		t.Errorf("Left.AsNumber = false, want true (days in)")
	}
}

func TestParser_LengthOf_AtEOF(t *testing.T) {
	_, err := Parse("x = length of")
	if err == nil {
		t.Fatal("expected parse error for `length of` at EOF")
	}
}

func TestParser_DaysIn_AtEOF(t *testing.T) {
	_, err := Parse("x = days in")
	if err == nil {
		t.Fatal("expected parse error for `days in` at EOF")
	}
}

// User-asked 2026-05-07: Q1-Q4 alone resolve to the current calendar
// year, but there was no syntax to specify a quarter of an explicit
// year. `CY2026 Q3` and `Q3 CY2026` parse-errored; `2026 Q3` lexed as
// a Quantity (not a Period) because the lexer recognizes `<digits> Q<digit>`
// as a quantity-shape token. Same gap for fiscal: `FY2027 FQ1` and
// `FQ1 FY2027` parse-errored.
//
// The fix combines an adjacent year + quarter literal into a single
// RelativeDateLiteral with a year-bearing keyword. Encoding:
//
//	`Q:<n>@<year>`   — calendar quarter <n> of calendar year <year>.
//	`FQ:<n>@<year>`  — fiscal quarter <n> of fiscal year <year>.
//
// All four input forms canonicalise to the same keyword:
//
//	2026 Q3      → Q:3@2026
//	CY2026 Q3    → Q:3@2026
//	Q3 2026      → Q:3@2026
//	Q3 CY2026    → Q:3@2026
//
// Two-digit years follow the existing CY/FY rule (CY26 → 2026), so
// `CY26 Q3` keyword is `Q:3@26`; the interpreter expands at eval
// time. The explicit-year suffix coexists with the directionless
// `Q:3` (current year): the latter is unchanged.
func TestParser_PeriodYearQuarterCombinations(t *testing.T) {
	cases := []struct {
		input       string
		wantKeyword string
	}{
		// Calendar quarter + year, all four orderings + year shapes.
		{"2026 Q3", "Q:3@2026"},
		{"CY2026 Q3", "Q:3@2026"},
		{"Q3 2026", "Q:3@2026"},
		{"Q3 CY2026", "Q:3@2026"},
		{"CY26 Q3", "Q:3@26"},
		{"Q3 CY26", "Q:3@26"},
		// Fiscal quarter + fiscal year.
		{"FY2027 FQ1", "FQ:1@2027"},
		{"FQ1 FY2027", "FQ:1@2027"},
		{"FY27 FQ1", "FQ:1@27"},
		{"FQ1 FY27", "FQ:1@27"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			n := parseFirstStmt(t, "x = "+tc.input)
			expr := expressionFromAssignment(t, n)
			rdl, ok := expr.(*ast.RelativeDateLiteral)
			if !ok {
				t.Fatalf("input %q: expected *ast.RelativeDateLiteral, got %T", tc.input, expr)
			}
			if rdl.Keyword != tc.wantKeyword {
				t.Errorf("input %q: Keyword = %q, want %q", tc.input, rdl.Keyword, tc.wantKeyword)
			}
			// SourceText preserves both tokens with their original
			// spacing — used in diagnostics + AST round-tripping.
			if !strings.Contains(rdl.SourceText, "Q") &&
				!strings.Contains(rdl.SourceText, "q") {
				t.Errorf("input %q: SourceText missing quarter token: %q", tc.input, rdl.SourceText)
			}
		})
	}
}

// The unchanged path: bare `Q1`-`Q4` and `CY2026` still produce the
// pre-feature keyword shape. Documents that worked before continue
// to work without any keyword shift.
func TestParser_PeriodYearQuarterCombinations_BareFormUnchanged(t *testing.T) {
	cases := []struct {
		input       string
		wantKeyword string
	}{
		{"Q1", "Q:1"},
		{"Q4", "Q:4"},
		{"CY2026", "CY:2026"},
		{"FY27", "FY:27"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			n := parseFirstStmt(t, "x = "+tc.input)
			expr := expressionFromAssignment(t, n)
			rdl, ok := expr.(*ast.RelativeDateLiteral)
			if !ok {
				t.Fatalf("input %q: expected *ast.RelativeDateLiteral, got %T", tc.input, expr)
			}
			if rdl.Keyword != tc.wantKeyword {
				t.Errorf("input %q: Keyword = %q, want %q", tc.input, rdl.Keyword, tc.wantKeyword)
			}
		})
	}
}
