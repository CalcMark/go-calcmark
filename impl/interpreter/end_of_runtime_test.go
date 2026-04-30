package interpreter

import (
	"testing"
	"time"

	"github.com/CalcMark/go-calcmark/v2/spec/parser"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
)

// TestEndOfExpr_VariableBoundWorks — v2.0 R9 (variables-as-periods).
// Pre-v2: `q = Q1; e = end of q` errored because the variable held
// a Date (Q1's start) and end-of couldn't recover the period kind.
// v2: Q1 evaluates to *types.Period; the variable holds the Period
// directly; `end of q` reads Period.End cleanly. The test now asserts
// the working behavior — Q1 ends March 31 — and pins the v2.0 R9
// emergent capability so it can't regress.
func TestEndOfExpr_VariableBoundWorks(t *testing.T) {
	interp := newTestInterpreterWithClock(time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC))
	src := "q = Q1\ne = end of q"
	nodes, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("expected `end of q` to evaluate cleanly with v2 Period plumbing; got error: %v", err)
	}
	// Two results: the assignment of q, then end of q.
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	d, ok := results[1].(*types.Date)
	if !ok {
		t.Fatalf("expected *types.Date for `end of q`, got %T", results[1])
	}
	if d.Time.Year() != 2026 || d.Time.Month() != time.March || d.Time.Day() != 31 {
		t.Errorf("end of Q1 = %v, want 2026-03-31", d.Time.Format("2006-01-02"))
	}
}

// TestEndOfExpr_NumberInnerRuntimeError — `end of 5` is rejected at
// runtime even when the type-checker is bypassed. The interpreter
// guard ensures correctness.
func TestEndOfExpr_NumberInnerRuntimeError(t *testing.T) {
	interp := newTestInterpreterWithClock(time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC))
	nodes, err := parser.Parse("x = end of 5")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, err := interp.Eval(nodes); err == nil {
		t.Fatal("expected runtime error from `end of 5`; got nil")
	}
}

// TestEndOfExpr_BareMonthName — regression for PR-1b. Pre-PR-1b,
// `end of April` worked through the old string-prefix dispatch. The
// new AST-based dispatch sees `*ast.DateLiteral` (parser produces
// that for bare month names with implicit Day=1, Year=nil) and
// initially rejected it. The fix accepts bare month names (no
// digits in SourceText) and routes them through the existing
// keyword-keyed evalEndOf path.
func TestEndOfExpr_BareMonthName(t *testing.T) {
	cases := []struct {
		name, input    string
		wantMonth, wantDay int
	}{
		{"end of April", "x = end of April", 4, 30},
		{"end of January", "x = end of January", 1, 31},
		{"end of February (non-leap)", "x = end of February", 2, 28},
		{"end of December", "x = end of December", 12, 31},
		{"end of Apr (abbrev)", "x = end of Apr", 4, 30},
		{"start of April", "x = start of April", 4, 1},
		{"start of February", "x = start of February", 2, 1},
	}
	// Use a non-leap year for predictable Feb behavior.
	clock := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			interp := newTestInterpreterWithClock(clock)
			nodes, err := parser.Parse(tc.input)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval(%q): %v", tc.input, err)
			}
			if len(results) != 1 {
				t.Fatalf("expected 1 result, got %d", len(results))
			}
			date, ok := results[0].(*types.Date)
			if !ok {
				t.Fatalf("expected *types.Date, got %T", results[0])
			}
			if int(date.Time.Month()) != tc.wantMonth || date.Time.Day() != tc.wantDay {
				t.Errorf("got %s, want month=%d day=%d",
					date.Time.Format("2006-01-02"), tc.wantMonth, tc.wantDay)
			}
		})
	}
}
