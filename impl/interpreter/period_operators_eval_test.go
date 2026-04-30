package interpreter

import (
	"strings"
	"testing"
	"time"

	"github.com/CalcMark/go-calcmark/spec/parser"
	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

// U13 — runtime evaluation of v2.0 period operators:
//   - BetweenExpr  → *types.Period (PeriodCustom kind)
//   - LengthOfExpr → *types.Duration (length of) or *types.Number (days in)
//
// Semantic checks (U9) reject obvious type errors at parse time;
// these tests exercise the runtime against the cases that pass
// semantic and verify the evaluated values.

// --- BetweenExpr ---

func TestEvalBetween_DateLiterals(t *testing.T) {
	interp := newTestInterpreterWithClock(testClock)
	nodes, err := parser.Parse("p = between Apr 15 2026 and Jul 4 2026\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	p, ok := results[0].(*types.Period)
	if !ok {
		t.Fatalf("expected *types.Period, got %T (%v)", results[0], results[0])
	}
	if p.Kind != types.PeriodCustom {
		t.Errorf("Kind = %v, want PeriodCustom", p.Kind)
	}
	wantStart := time.Date(2026, time.April, 15, 0, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, time.July, 4, 0, 0, 0, 0, time.UTC)
	if !p.Start.Time.Equal(wantStart) {
		t.Errorf("Start = %v, want %v", p.Start.Time, wantStart)
	}
	if !p.End.Time.Equal(wantEnd) {
		t.Errorf("End = %v, want %v", p.End.Time, wantEnd)
	}
}

func TestEvalBetween_FromToSynonym(t *testing.T) {
	// `from A to B` and `between A and B` must produce structurally
	// identical Period values.
	interp1 := newTestInterpreterWithClock(testClock)
	interp2 := newTestInterpreterWithClock(testClock)

	src1 := "p = between Apr 15 2026 and Jul 4 2026\n"
	src2 := "p = from Apr 15 2026 to Jul 4 2026\n"

	nodes1, _ := parser.Parse(src1)
	nodes2, _ := parser.Parse(src2)

	r1, err := interp1.Eval(nodes1)
	if err != nil {
		t.Fatalf("between Eval: %v", err)
	}
	r2, err := interp2.Eval(nodes2)
	if err != nil {
		t.Fatalf("from-to Eval: %v", err)
	}

	p1 := r1[0].(*types.Period)
	p2 := r2[0].(*types.Period)
	if !p1.Start.Time.Equal(p2.Start.Time) {
		t.Errorf("Start mismatch: between=%v from-to=%v", p1.Start.Time, p2.Start.Time)
	}
	if !p1.End.Time.Equal(p2.End.Time) {
		t.Errorf("End mismatch: between=%v from-to=%v", p1.End.Time, p2.End.Time)
	}
}

func TestEvalBetween_RelativeDates(t *testing.T) {
	clock := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)
	interp := newTestInterpreterWithClock(clock)
	nodes, err := parser.Parse("p = between today and tomorrow\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	p, ok := results[0].(*types.Period)
	if !ok {
		t.Fatalf("expected *types.Period, got %T", results[0])
	}
	if !p.Start.Time.Equal(clock) {
		t.Errorf("Start = %v, want %v", p.Start.Time, clock)
	}
	if !p.End.Time.Equal(clock.AddDate(0, 0, 1)) {
		t.Errorf("End = %v, want %v", p.End.Time, clock.AddDate(0, 0, 1))
	}
}

func TestEvalBetween_SingleDay(t *testing.T) {
	// start == end is the minimal valid period.
	interp := newTestInterpreterWithClock(testClock)
	nodes, err := parser.Parse("p = between Apr 15 2026 and Apr 15 2026\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	p := results[0].(*types.Period)
	if !p.Start.Time.Equal(p.End.Time) {
		t.Errorf("single-day period Start (%v) != End (%v)", p.Start.Time, p.End.Time)
	}
}

func TestEvalBetween_EndBeforeStart(t *testing.T) {
	// Static-detect: end < start → runtime error from NewCustomPeriod.
	interp := newTestInterpreterWithClock(testClock)
	nodes, err := parser.Parse("p = between Jul 4 2026 and Apr 15 2026\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = interp.Eval(nodes)
	if err == nil {
		t.Fatal("expected runtime error for end-before-start; got nil")
	}
	if !strings.Contains(err.Error(), "before") && !strings.Contains(err.Error(), "end") {
		t.Errorf("error %q should mention end-before-start", err.Error())
	}
}

// TestEvalBetween_PeriodInputAcceptedAsDate — `between today and
// next month` puts a Period (next month) on the End side. The
// evaluator narrows the Period to its Start before constructing the
// custom period — so `between today and next month` = today to
// next-month-start. (Mirrors the parser-side narrowing for `from`.)
func TestEvalBetween_PeriodInputAcceptedAsDate(t *testing.T) {
	clock := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)
	interp := newTestInterpreterWithClock(clock)
	nodes, err := parser.Parse("p = between today and next month\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	p, ok := results[0].(*types.Period)
	if !ok {
		t.Fatalf("expected *types.Period, got %T", results[0])
	}
	if !p.Start.Time.Equal(clock) {
		t.Errorf("Start = %v, want %v (today)", p.Start.Time, clock)
	}
	wantEnd := time.Date(2026, time.May, 1, 0, 0, 0, 0, time.UTC)
	if !p.End.Time.Equal(wantEnd) {
		t.Errorf("End = %v, want %v (start of next month)", p.End.Time, wantEnd)
	}
}

// --- LengthOfExpr (length of) ---

func TestEvalLengthOf_Quarter(t *testing.T) {
	// length of Q1 = 90 days (Jan 1 - Mar 31 = 90 days inclusive).
	interp := newTestInterpreterWithClock(testClock)
	nodes, err := parser.Parse("d = length of Q1\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	dur, ok := results[0].(*types.Duration)
	if !ok {
		t.Fatalf("expected *types.Duration, got %T (%v)", results[0], results[0])
	}
	if !dur.Value.Equal(decimal.NewFromInt(90)) {
		t.Errorf("Q1 length = %v, want 90", dur.Value)
	}
	if dur.Unit != "day" {
		t.Errorf("unit = %q, want %q", dur.Unit, "day")
	}
}

func TestEvalLengthOf_Month(t *testing.T) {
	// length of April = 30 days.
	clock := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)
	interp := newTestInterpreterWithClock(clock)
	nodes, err := parser.Parse("d = length of April\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	dur := results[0].(*types.Duration)
	if !dur.Value.Equal(decimal.NewFromInt(30)) {
		t.Errorf("April length = %v, want 30", dur.Value)
	}
}

func TestEvalLengthOf_Custom(t *testing.T) {
	interp := newTestInterpreterWithClock(testClock)
	nodes, err := parser.Parse("d = length of (between Apr 15 2026 and Jul 4 2026)\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	dur := results[0].(*types.Duration)
	// Apr 15 to Jul 4 inclusive = 81 days (15→30 = 16, May = 31,
	// Jun = 30, Jul 1-4 = 4 → 16+31+30+4 = 81).
	if !dur.Value.Equal(decimal.NewFromInt(81)) {
		t.Errorf("custom period length = %v, want 81", dur.Value)
	}
}

// --- LengthOfExpr (days in) ---

func TestEvalDaysIn_Quarter(t *testing.T) {
	// `days in Q1` = Number(90), not Duration.
	interp := newTestInterpreterWithClock(testClock)
	nodes, err := parser.Parse("n = days in Q1\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	num, ok := results[0].(*types.Number)
	if !ok {
		t.Fatalf("expected *types.Number, got %T (%v)", results[0], results[0])
	}
	if !num.Value.Equal(decimal.NewFromInt(90)) {
		t.Errorf("days in Q1 = %v, want 90", num.Value)
	}
}

func TestEvalDaysIn_Month(t *testing.T) {
	clock := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)
	interp := newTestInterpreterWithClock(clock)
	nodes, err := parser.Parse("n = days in April\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	num := results[0].(*types.Number)
	if !num.Value.Equal(decimal.NewFromInt(30)) {
		t.Errorf("days in April = %v, want 30", num.Value)
	}
}

func TestEvalDaysIn_LeapFebruary(t *testing.T) {
	clock := time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC) // leap year
	interp := newTestInterpreterWithClock(clock)
	nodes, err := parser.Parse("n = days in this month\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	num := results[0].(*types.Number)
	if !num.Value.Equal(decimal.NewFromInt(29)) {
		t.Errorf("days in Feb 2024 = %v, want 29 (leap)", num.Value)
	}
}

func TestEvalDaysIn_NonLeapFebruary(t *testing.T) {
	clock := time.Date(2025, 2, 15, 0, 0, 0, 0, time.UTC)
	interp := newTestInterpreterWithClock(clock)
	nodes, err := parser.Parse("n = days in this month\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	num := results[0].(*types.Number)
	if !num.Value.Equal(decimal.NewFromInt(28)) {
		t.Errorf("days in Feb 2025 = %v, want 28 (non-leap)", num.Value)
	}
}

// TestEvalDaysIn_UsedInArithmetic — `days in cycle * 1000` exercises
// the Number return value flowing into ordinary arithmetic. Locks the
// brainstorm's "days in returns Number not Duration" decision.
func TestEvalDaysIn_UsedInArithmetic(t *testing.T) {
	interp := newTestInterpreterWithClock(testClock)
	src := "cycle = between Apr 15 2026 and Jul 4 2026\n" +
		"cost = days in cycle * 1000\n"
	nodes, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	num, ok := results[1].(*types.Number)
	if !ok {
		t.Fatalf("expected *types.Number for cost, got %T", results[1])
	}
	// 81 days × 1000 = 81000.
	if !num.Value.Equal(decimal.NewFromInt(81000)) {
		t.Errorf("days in cycle * 1000 = %v, want 81000", num.Value)
	}
}

// TestEvalLengthOf_VariableBound — R9 emergent capability for the
// new operators: `q = Q1; d = length of q` works because Q1
// evaluates to *types.Period and the variable holds it directly.
func TestEvalLengthOf_VariableBound(t *testing.T) {
	interp := newTestInterpreterWithClock(testClock)
	src := "q = Q1\nd = length of q\n"
	nodes, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	dur := results[1].(*types.Duration)
	if !dur.Value.Equal(decimal.NewFromInt(90)) {
		t.Errorf("length of (variable-bound Q1) = %v, want 90", dur.Value)
	}
}
