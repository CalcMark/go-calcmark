package interpreter

import (
	"strings"
	"testing"
	"time"

	"github.com/CalcMark/go-calcmark/v2/spec/parser"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
)

// Period basis conversion: `<period> as fiscal` / `<period> as
// calendar`. Issue #146 (upstream go-calcmark). Midpoint-rule
// implementation: returns the period of the requested basis whose
// dates contain the input period's midpoint.

// Year-level conversion: numeric label carries across. CY2026 ↔
// FY2026 (NOT FY2027), even though the underlying date ranges differ.
// Label-match preserves the user's "year 2026" mental model.

func TestBasisConversion_CalendarYearAsFiscal(t *testing.T) {
	interp := newTestInterpreterWithClock(testClock)
	interp.SetFiscalYearStarts(7, 1) // July 1
	nodes, err := parser.Parse("p = CY2026 as fiscal\n")
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
	if p.Kind != types.PeriodFiscalYear {
		t.Errorf("Kind = %v, want PeriodFiscalYear", p.Kind)
	}
	if p.Year != 2026 {
		t.Errorf("FY label = %d, want 2026 (label-match: CY2026 ↔ FY2026)", p.Year)
	}
}

func TestBasisConversion_FiscalYearAsCalendar(t *testing.T) {
	interp := newTestInterpreterWithClock(testClock)
	interp.SetFiscalYearStarts(7, 1)
	nodes, err := parser.Parse("p = FY2027 as calendar\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	p := results[0].(*types.Period)
	if p.Kind != types.PeriodCalendarYear {
		t.Errorf("Kind = %v, want PeriodCalendarYear", p.Kind)
	}
	if p.Year != 2027 {
		t.Errorf("CY label = %d, want 2027 (label-match: FY2027 ↔ CY2027)", p.Year)
	}
}

// TestBasisConversion_YearRoundTripIsIdentity — locks the
// label-match contract via two-step round-trip (chained `as` not
// supported by the existing UnitConversion parser).
func TestBasisConversion_YearRoundTripIsIdentity(t *testing.T) {
	interp := newTestInterpreterWithClock(testClock)
	interp.SetFiscalYearStarts(7, 1)
	nodes, err := parser.Parse("fy = CY2026 as fiscal\np = fy as calendar\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	p := results[1].(*types.Period)
	if p.Kind != types.PeriodCalendarYear || p.Year != 2026 {
		t.Errorf("round-trip CY2026 → FY → CY = (%v, %d), want (PeriodCalendarYear, 2026)",
			p.Kind, p.Year)
	}
}

func TestBasisConversion_QuarterAsFiscal(t *testing.T) {
	interp := newTestInterpreterWithClock(testClock)
	interp.SetFiscalYearStarts(7, 1)
	// Q1 of CY2026 = Jan-Mar 2026. With Jul-start FY, FY2026 covers
	// Jul 2025 - Jun 2026, and FQ3 of FY2026 = Jan-Mar 2026 (matches
	// Q1 dates exactly).
	nodes, err := parser.Parse("p = Q1 as fiscal\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	p := results[0].(*types.Period)
	if p.Kind != types.PeriodFiscalQuarter {
		t.Errorf("Kind = %v, want PeriodFiscalQuarter", p.Kind)
	}
	if p.QuarterIndex != 3 {
		t.Errorf("FQ index = %d, want 3 (Q1=Jan-Mar matches FQ3 of FY2026 with Jul start)",
			p.QuarterIndex)
	}
	wantStart := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC)
	if !p.Start.Time.Equal(wantStart) {
		t.Errorf("Start = %v, want %v", p.Start.Time, wantStart)
	}
	if !p.End.Time.Equal(wantEnd) {
		t.Errorf("End = %v, want %v", p.End.Time, wantEnd)
	}
}

func TestBasisConversion_FiscalQuarterAsCalendar(t *testing.T) {
	clock := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC) // Aug 2026 = FQ1 of FY2027
	interp := newTestInterpreterWithClock(clock)
	interp.SetFiscalYearStarts(7, 1)
	// FQ1 of FY2027 = Jul-Sep 2026. Midpoint = mid-Aug.
	// Calendar quarter containing mid-Aug = Q3 (Jul-Sep).
	nodes, err := parser.Parse("p = FQ1 as calendar\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	p := results[0].(*types.Period)
	if p.Kind != types.PeriodCalendarQuarter {
		t.Errorf("Kind = %v, want PeriodCalendarQuarter", p.Kind)
	}
	if p.QuarterIndex != 3 {
		t.Errorf("Q index = %d, want 3 (FQ1 with Jul start = Jul-Sep = Q3)",
			p.QuarterIndex)
	}
}

func TestBasisConversion_ThisYearAsFiscal(t *testing.T) {
	clock := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	interp := newTestInterpreterWithClock(clock)
	interp.SetFiscalYearStarts(7, 1)
	// `this year` resolves to CY2026 (Jan-Dec 2026). Label-match
	// rule: → FY2026 (NOT FY2027). The user's mental model is
	// "year 2026" — preserving the numeric year is more intuitive
	// than picking the FY whose date-range happens to contain the
	// midpoint.
	nodes, err := parser.Parse("p = this year as fiscal\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	p := results[0].(*types.Period)
	if p.Kind != types.PeriodFiscalYear {
		t.Errorf("Kind = %v, want PeriodFiscalYear", p.Kind)
	}
	if p.Year != 2026 {
		t.Errorf("FY label = %d, want 2026 (label-match)", p.Year)
	}
}

func TestBasisConversion_AsFY_ShortAlias(t *testing.T) {
	// `as fy` should behave like `as fiscal`.
	interp := newTestInterpreterWithClock(testClock)
	interp.SetFiscalYearStarts(7, 1)
	nodes, _ := parser.Parse("p = CY2026 as fy\n")
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	if _, ok := results[0].(*types.Period); !ok {
		t.Fatalf("expected *types.Period, got %T", results[0])
	}
}

func TestBasisConversion_AsCY_ShortAlias(t *testing.T) {
	interp := newTestInterpreterWithClock(testClock)
	interp.SetFiscalYearStarts(7, 1)
	nodes, _ := parser.Parse("p = FY2027 as cy\n")
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	p := results[0].(*types.Period)
	if p.Kind != types.PeriodCalendarYear {
		t.Errorf("Kind = %v, want PeriodCalendarYear", p.Kind)
	}
}

func TestBasisConversion_FiscalRequiresConfig(t *testing.T) {
	// `Q1 as fiscal` without fiscal_year_starts must error with a
	// clear message.
	interp := newTestInterpreterWithClock(testClock)
	// NOT calling SetFiscalYearStarts.
	nodes, err := parser.Parse("p = Q1 as fiscal\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = interp.Eval(nodes)
	if err == nil {
		t.Fatal("expected error for `as fiscal` without fiscal_year_starts; got nil")
	}
	if !strings.Contains(err.Error(), "fiscal_year_starts") {
		t.Errorf("error %q should mention 'fiscal_year_starts'", err.Error())
	}
}

func TestBasisConversion_CalendarSideAlwaysWorks(t *testing.T) {
	// `as calendar` must work without fiscal_year_starts when the
	// input doesn't depend on fiscal config (e.g., FY literal does
	// require it to construct, but `as calendar` reads its dates
	// from the input).
	interp := newTestInterpreterWithClock(testClock)
	// no SetFiscalYearStarts
	nodes, err := parser.Parse("p = Q1 as calendar\n")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Q1 as calendar should work without fiscal config; got: %v", err)
	}
}

