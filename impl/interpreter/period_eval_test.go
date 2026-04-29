package interpreter

import (
	"testing"
	"time"

	"github.com/CalcMark/go-calcmark/spec/parser"
	"github.com/CalcMark/go-calcmark/spec/types"
)

// U10 — period-bearing keywords evaluate to *types.Period.
//
// Pre-v2: `Q1`, `this month`, `FY2027`, `next quarter`, etc., all
// returned *types.Date — the START of the period — and end-of /
// start-of operators synthesized the rest from kind+context.
//
// v2.0: those keywords return *types.Period with both Start and End
// populated, enabling natural Period arithmetic, Period == Period
// comparison, and variable-bound period operators (`q = Q1; e =
// end of q` works without further plumbing).
//
// True date keywords (today, tomorrow, yesterday, now) continue to
// return *types.Date — they're points, not spans.

func TestPeriodEval_QuarterNotation(t *testing.T) {
	clock := time.Date(2026, 4, 8, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		input              string
		wantStartMonth     time.Month
		wantStartDay       int
		wantEndMonth       time.Month
		wantEndDay         int
		wantQuarterIndex   int
	}{
		{"x = Q1\n", time.January, 1, time.March, 31, 1},
		{"x = Q2\n", time.April, 1, time.June, 30, 2},
		{"x = Q3\n", time.July, 1, time.September, 30, 3},
		{"x = Q4\n", time.October, 1, time.December, 31, 4},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			interp := newTestInterpreterWithClock(clock)
			nodes, err := parser.Parse(tc.input)
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
			if p.Kind != types.PeriodCalendarQuarter {
				t.Errorf("Kind = %v, want PeriodCalendarQuarter", p.Kind)
			}
			if p.QuarterIndex != tc.wantQuarterIndex {
				t.Errorf("QuarterIndex = %d, want %d", p.QuarterIndex, tc.wantQuarterIndex)
			}
			if p.Start == nil || p.End == nil {
				t.Fatalf("Start or End is nil; both must be populated")
			}
			if p.Start.Time.Month() != tc.wantStartMonth || p.Start.Time.Day() != tc.wantStartDay {
				t.Errorf("Start = %v, want %v %d", p.Start.Time, tc.wantStartMonth, tc.wantStartDay)
			}
			if p.End.Time.Month() != tc.wantEndMonth || p.End.Time.Day() != tc.wantEndDay {
				t.Errorf("End = %v, want %v %d", p.End.Time, tc.wantEndMonth, tc.wantEndDay)
			}
		})
	}
}

func TestPeriodEval_CalendarYear(t *testing.T) {
	interp := newTestInterpreterWithClock(testClock)
	nodes, err := parser.Parse("x = CY2026\n")
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
	if p.Kind != types.PeriodCalendarYear {
		t.Errorf("Kind = %v, want PeriodCalendarYear", p.Kind)
	}
	if p.Year != 2026 {
		t.Errorf("Year = %d, want 2026", p.Year)
	}
	if p.Start.Time.Month() != time.January || p.Start.Time.Day() != 1 {
		t.Errorf("Start = %v, want Jan 1", p.Start.Time)
	}
	if p.End.Time.Month() != time.December || p.End.Time.Day() != 31 {
		t.Errorf("End = %v, want Dec 31", p.End.Time)
	}
}

func TestPeriodEval_FiscalYear(t *testing.T) {
	interp := newTestInterpreterWithClock(testClock)
	interp.SetFiscalYearStarts(7, 1) // July 1
	nodes, err := parser.Parse("x = FY2027\n")
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
	// FY2027 with July start = Jul 1 2026 - Jun 30 2027 (Microsoft).
	wantStart := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2027, time.June, 30, 0, 0, 0, 0, time.UTC)
	if !p.Start.Time.Equal(wantStart) {
		t.Errorf("Start = %v, want %v", p.Start.Time, wantStart)
	}
	if !p.End.Time.Equal(wantEnd) {
		t.Errorf("End = %v, want %v", p.End.Time, wantEnd)
	}
}

func TestPeriodEval_ThisMonth(t *testing.T) {
	clock := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)
	interp := newTestInterpreterWithClock(clock)
	nodes, err := parser.Parse("x = this month\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	p, ok := results[0].(*types.Period)
	if !ok {
		t.Fatalf("expected *types.Period for `this month`, got %T", results[0])
	}
	if p.Start.Time.Month() != time.April || p.Start.Time.Day() != 1 {
		t.Errorf("Start = %v, want April 1", p.Start.Time)
	}
	if p.End.Time.Month() != time.April || p.End.Time.Day() != 30 {
		t.Errorf("End = %v, want April 30", p.End.Time)
	}
}

func TestPeriodEval_BareMonthName(t *testing.T) {
	// Bare month → RelativeDateLiteral{Keyword: "this April"} (U5)
	// → resolveMonthExpression → Period (U10).
	clock := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)
	interp := newTestInterpreterWithClock(clock)
	nodes, err := parser.Parse("x = April\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	p, ok := results[0].(*types.Period)
	if !ok {
		t.Fatalf("expected *types.Period for bare month `April`, got %T (%v)", results[0], results[0])
	}
	if p.Start.Time.Month() != time.April || p.Start.Time.Day() != 1 {
		t.Errorf("Start = %v, want April 1", p.Start.Time)
	}
	if p.End.Time.Month() != time.April || p.End.Time.Day() != 30 {
		t.Errorf("End = %v, want April 30", p.End.Time)
	}
}

// TestPeriodEval_TodayStaysDate — point-in-time keywords retain
// *types.Date semantics. A test failure here would indicate a regression
// in the Date/Period split.
func TestPeriodEval_TodayStaysDate(t *testing.T) {
	for _, input := range []string{
		"x = today\n",
		"x = tomorrow\n",
		"x = yesterday\n",
	} {
		t.Run(input, func(t *testing.T) {
			interp := newTestInterpreterWithClock(testClock)
			nodes, err := parser.Parse(input)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval: %v", err)
			}
			if _, ok := results[0].(*types.Date); !ok {
				t.Errorf("expected *types.Date for %q, got %T", input, results[0])
			}
		})
	}
}

// TestPeriodArithmetic_PlusFromEnd — v2.0 asymmetric arithmetic:
// Period + Duration extends from Period.End.
func TestPeriodArithmetic_PlusFromEnd(t *testing.T) {
	interp := newTestInterpreterWithClock(testClock)
	nodes, err := parser.Parse("x = Q1 + 30 days\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	d, ok := results[0].(*types.Date)
	if !ok {
		t.Fatalf("expected *types.Date, got %T", results[0])
	}
	// Q1.End = Mar 31; + 30 days = April 30.
	want := time.Date(2026, time.April, 30, 0, 0, 0, 0, time.UTC)
	if !d.Time.Equal(want) {
		t.Errorf("Q1 + 30 days = %v, want %v", d.Time.Format("2006-01-02"), want.Format("2006-01-02"))
	}
}

// TestPeriodArithmetic_MinusFromStart — Period - Duration extends
// from Period.Start.
func TestPeriodArithmetic_MinusFromStart(t *testing.T) {
	interp := newTestInterpreterWithClock(testClock)
	nodes, err := parser.Parse("x = Q1 - 5 days\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	d, ok := results[0].(*types.Date)
	if !ok {
		t.Fatalf("expected *types.Date, got %T", results[0])
	}
	// Q1.Start = Jan 1; - 5 days = Dec 27, 2025.
	want := time.Date(2025, time.December, 27, 0, 0, 0, 0, time.UTC)
	if !d.Time.Equal(want) {
		t.Errorf("Q1 - 5 days = %v, want %v", d.Time.Format("2006-01-02"), want.Format("2006-01-02"))
	}
}

// TestPeriodArithmetic_StructuralEquality — Period == Period uses
// structural comparison: Start.Equal && End.Equal.
func TestPeriodArithmetic_StructuralEquality(t *testing.T) {
	interp := newTestInterpreterWithClock(testClock)
	// Q1 vs Q1 — same period, same year (clock=2026).
	nodes, err := parser.Parse("x = Q1 == Q1\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	b, ok := results[0].(*types.Boolean)
	if !ok {
		t.Fatalf("expected *types.Boolean, got %T", results[0])
	}
	if !b.Value {
		t.Error("Q1 == Q1 = false, want true")
	}
}

// TestFromArith_PeriodIsNarrowed — the `from <X>` syntax preserves
// pre-v2 date-arithmetic semantics: the period is narrowed to its
// Start before the duration is added. So `2 weeks from next month`
// (clock=April) = May 1 + 14 days = May 15, NOT May 31 + 14 days.
//
// This is parser-side narrowing (StartOfExpr wrapping in
// parseFromTarget) so the semantic distinction between explicit
// `+` arithmetic (extends from end) and `from` arithmetic (extends
// from start) is preserved.
func TestFromArith_PeriodIsNarrowed(t *testing.T) {
	interp := newTestInterpreterWithClock(testClock)
	nodes, err := parser.Parse("x = 2 weeks from next month\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	d, ok := results[0].(*types.Date)
	if !ok {
		t.Fatalf("expected *types.Date, got %T", results[0])
	}
	want := time.Date(2026, time.May, 15, 0, 0, 0, 0, time.UTC)
	if !d.Time.Equal(want) {
		t.Errorf("2 weeks from next month = %v, want %v",
			d.Time.Format("2006-01-02"), want.Format("2006-01-02"))
	}
}
