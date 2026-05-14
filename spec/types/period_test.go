package types

import (
	"strings"
	"testing"
	"time"
)

// Period is the value type that period-bearing date keywords (Q1,
// FQ1, this month, this fiscal quarter, ...) evaluate to. The
// interpreter dispatches `end of` / `start of` by Period.Kind. This
// test file pins the value-type contract — it does NOT test eval.
//
// Test design — flat struct + kind enum (per plan's Key Decision
// mirroring spec/types/duration.go's flat shape, vs. an interface
// hierarchy). Each test asserts a concrete kind's factory output.

func TestPeriodImplementsType(t *testing.T) {
	var _ Type = (*Period)(nil)
}

func TestPeriod_NewCalendarQuarter(t *testing.T) {
	cases := []struct {
		year       int
		quarter    int
		wantMonth  time.Month
		wantString string
	}{
		{2026, 1, time.January, "Calendar Q1 2026"},
		{2026, 2, time.April, "Calendar Q2 2026"},
		{2026, 3, time.July, "Calendar Q3 2026"},
		{2026, 4, time.October, "Calendar Q4 2026"},
	}
	for _, tc := range cases {
		p, err := NewCalendarQuarter(tc.year, tc.quarter)
		if err != nil {
			t.Fatalf("NewCalendarQuarter(%d, %d): %v", tc.year, tc.quarter, err)
		}
		if p.Kind != PeriodCalendarQuarter {
			t.Errorf("Kind = %v, want PeriodCalendarQuarter", p.Kind)
		}
		if p.QuarterIndex != tc.quarter {
			t.Errorf("QuarterIndex = %d, want %d", p.QuarterIndex, tc.quarter)
		}
		if p.Year != tc.year {
			t.Errorf("Year = %d, want %d", p.Year, tc.year)
		}
		if p.Start == nil {
			t.Fatalf("Start is nil")
		}
		if p.Start.Time.Year() != tc.year || p.Start.Time.Month() != tc.wantMonth || p.Start.Time.Day() != 1 {
			t.Errorf("Start = %v, want %d-%v-01", p.Start.Time, tc.year, tc.wantMonth)
		}
		if got := p.String(); got != tc.wantString {
			t.Errorf("String() = %q, want %q", got, tc.wantString)
		}
	}
}

func TestPeriod_NewCalendarQuarter_InvalidIndex(t *testing.T) {
	for _, q := range []int{0, 5, -1, 100} {
		if _, err := NewCalendarQuarter(2026, q); err == nil {
			t.Errorf("NewCalendarQuarter(2026, %d) should error; got nil", q)
		}
	}
}

func TestPeriod_NewFiscalQuarter(t *testing.T) {
	// FY starts April 1; FQ1 starts April; FQ2 = July; FQ3 = October; FQ4 = next January.
	apr1 := time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		quarter int
		want    time.Time
	}{
		{1, time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC)},
		{2, time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)},
		{3, time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)},
		{4, time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC)},
	}
	for _, tc := range cases {
		p, err := NewFiscalQuarter(apr1, tc.quarter)
		if err != nil {
			t.Fatalf("NewFiscalQuarter(Apr 1, %d): %v", tc.quarter, err)
		}
		if p.Kind != PeriodFiscalQuarter {
			t.Errorf("Kind = %v, want PeriodFiscalQuarter", p.Kind)
		}
		if !p.Start.Time.Equal(tc.want) {
			t.Errorf("FQ%d Start = %v, want %v", tc.quarter, p.Start.Time, tc.want)
		}
	}
}

func TestPeriod_NewCalendarYear(t *testing.T) {
	p := NewCalendarYear(2026)
	if p.Kind != PeriodCalendarYear {
		t.Errorf("Kind = %v, want PeriodCalendarYear", p.Kind)
	}
	if p.Year != 2026 {
		t.Errorf("Year = %d, want 2026", p.Year)
	}
	if p.Start.Time.Year() != 2026 || p.Start.Time.Month() != time.January || p.Start.Time.Day() != 1 {
		t.Errorf("Start = %v, want 2026-Jan-01", p.Start.Time)
	}
	if got := p.String(); got != "Calendar Year 2026" {
		t.Errorf("String() = %q, want %q", got, "Calendar Year 2026")
	}
}

// TestPeriod_NewFiscalQuarter_YearIsFYLabel — Period.Year carries
// the FY label (year FY ends in, default labeling), not the
// calendar year of the FQ's start. So FQ1 of FY2026 (Jul-start FY)
// starts in calendar 2025 but labels as "Fiscal Q1 2026". The
// calendar year of the start is recoverable from Period.Start when
// users need it.
func TestPeriod_NewFiscalQuarter_YearIsFYLabel(t *testing.T) {
	cases := []struct {
		name      string
		fyStart   time.Time
		quarter   int
		wantLabel int
	}{
		// Jul-start FY → FY2026 (Jul 2025 – Jun 2026).
		{"Jul-start FQ1", time.Date(2025, time.July, 1, 0, 0, 0, 0, time.UTC), 1, 2026},
		{"Jul-start FQ4", time.Date(2025, time.July, 1, 0, 0, 0, 0, time.UTC), 4, 2026},
		// Jan-start FY → FY label = start year.
		{"Jan-start FQ1", time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC), 1, 2026},
		// Apr-start FY → FY2027 (Apr 2026 – Mar 2027).
		{"Apr-start FQ1", time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC), 1, 2027},
		{"Apr-start FQ4", time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC), 4, 2027},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := NewFiscalQuarter(tc.fyStart, tc.quarter)
			if err != nil {
				t.Fatalf("NewFiscalQuarter: %v", err)
			}
			if p.Year != tc.wantLabel {
				t.Errorf("Year = %d, want %d (FY label per default end-year convention)",
					p.Year, tc.wantLabel)
			}
		})
	}
}

func TestPeriod_NewFiscalYear(t *testing.T) {
	// FY2027 with July start under default labeling = starts Jul 1, 2026.
	// Default mode (FYLabelByEndYear) labels by the year the FY ENDS in,
	// matching the Australian government year, US tax year, and most
	// publicly traded companies.
	p := NewFiscalYear(2027, time.July, 1)
	if p.Kind != PeriodFiscalYear {
		t.Errorf("Kind = %v, want PeriodFiscalYear", p.Kind)
	}
	if p.Year != 2027 {
		t.Errorf("Year = %d, want 2027", p.Year)
	}
	want := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	if !p.Start.Time.Equal(want) {
		t.Errorf("FY2027 Start = %v, want %v", p.Start.Time, want)
	}
	// String must surface the FY label, not the start year.
	if !strings.Contains(p.String(), "2027") {
		t.Errorf("String() = %q should contain FY label 2027", p.String())
	}
}

// TestPeriod_NewFiscalYearWithMode_LabelByStartYear — when a document
// declares `calendar_year_offset: after`, the FY label corresponds to
// the calendar year the FY STARTS in (some companies use this
// labeling). FY2026 with a Feb 2 start under start-year labeling
// starts Feb 2, 2026 and ends Feb 1, 2027.
func TestPeriod_NewFiscalYearWithMode_LabelByStartYear(t *testing.T) {
	p := NewFiscalYearWithMode(2026, time.February, 2, FYLabelByStartYear)
	if p.Kind != PeriodFiscalYear {
		t.Errorf("Kind = %v, want PeriodFiscalYear", p.Kind)
	}
	if p.Year != 2026 {
		t.Errorf("Year = %d, want 2026", p.Year)
	}
	wantStart := time.Date(2026, time.February, 2, 0, 0, 0, 0, time.UTC)
	if !p.Start.Time.Equal(wantStart) {
		t.Errorf("Start = %v, want %v", p.Start.Time, wantStart)
	}
	wantEnd := time.Date(2027, time.February, 1, 0, 0, 0, 0, time.UTC)
	if !p.End.Time.Equal(wantEnd) {
		t.Errorf("End = %v, want %v", p.End.Time, wantEnd)
	}
}

// TestPeriod_NewFiscalYearWithMode_LabelByEndYear — explicit
// FYLabelByEndYear matches the default factory exactly. FY2027 with
// July start = Jul 1 2026 → Jun 30 2027.
func TestPeriod_NewFiscalYearWithMode_LabelByEndYear(t *testing.T) {
	p := NewFiscalYearWithMode(2027, time.July, 1, FYLabelByEndYear)
	if p.Year != 2027 {
		t.Errorf("Year = %d, want 2027", p.Year)
	}
	wantStart := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	if !p.Start.Time.Equal(wantStart) {
		t.Errorf("Start = %v, want %v", p.Start.Time, wantStart)
	}
}

// TestPeriod_NewFiscalQuarterWithMode_LabelByStartYear — under
// start-year labeling, an Apr 2026 fiscal-year-start produces FQ1
// labeled FY2026 (not FY2027). The quarter itself spans Apr–Jun
// 2026 either way; only the year stamped on Period.Year differs.
func TestPeriod_NewFiscalQuarterWithMode_LabelByStartYear(t *testing.T) {
	apr2026 := time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC)
	p, err := NewFiscalQuarterWithMode(apr2026, 1, FYLabelByStartYear)
	if err != nil {
		t.Fatalf("NewFiscalQuarterWithMode: %v", err)
	}
	if p.Year != 2026 {
		t.Errorf("Year = %d, want 2026 (start-year labeling)", p.Year)
	}
	if p.QuarterIndex != 1 {
		t.Errorf("QuarterIndex = %d, want 1", p.QuarterIndex)
	}
}

// TestPeriod_NewFiscalQuarterWithMode_LabelByEndYear — sanity check
// that explicit end-year mode matches the existing default behavior.
func TestPeriod_NewFiscalQuarterWithMode_LabelByEndYear(t *testing.T) {
	apr2026 := time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC)
	p, err := NewFiscalQuarterWithMode(apr2026, 1, FYLabelByEndYear)
	if err != nil {
		t.Fatalf("NewFiscalQuarterWithMode: %v", err)
	}
	if p.Year != 2027 {
		t.Errorf("Year = %d, want 2027 (end-year labeling)", p.Year)
	}
}

func TestPeriod_NewCalendarMonth(t *testing.T) {
	p := NewCalendarMonth(2026, time.July)
	if p.Kind != PeriodCalendarMonth {
		t.Errorf("Kind = %v, want PeriodCalendarMonth", p.Kind)
	}
	if p.Month != int(time.July) {
		t.Errorf("Month = %d, want %d", p.Month, time.July)
	}
	want := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	if !p.Start.Time.Equal(want) {
		t.Errorf("Start = %v, want %v", p.Start.Time, want)
	}
}

func TestPeriod_NewRelativeQuarter(t *testing.T) {
	// `this quarter` / `next quarter` / `last quarter` — Direction
	// captures the offset relative to `now`.
	cases := []struct {
		name      string
		direction int
	}{
		{"this", 0},
		{"next", +1},
		{"last", -1},
	}
	for _, tc := range cases {
		now := time.Date(2026, time.May, 15, 0, 0, 0, 0, time.UTC) // Q2
		p := NewRelativeQuarter(now, tc.direction)
		if p.Kind != PeriodRelativeQuarter {
			t.Errorf("Kind = %v, want PeriodRelativeQuarter", p.Kind)
		}
		if p.Direction != tc.direction {
			t.Errorf("%s quarter Direction = %d, want %d", tc.name, p.Direction, tc.direction)
		}
		// Start must align to a quarter boundary.
		if p.Start.Time.Day() != 1 {
			t.Errorf("%s quarter Start day = %d, want 1", tc.name, p.Start.Time.Day())
		}
		switch p.Start.Time.Month() {
		case time.January, time.April, time.July, time.October:
			// Quarter boundary.
		default:
			t.Errorf("%s quarter Start month = %v, want a quarter boundary", tc.name, p.Start.Time.Month())
		}
	}
}

func TestPeriod_NewRelativeFiscalQuarter(t *testing.T) {
	// FY starts April 1. May 15 sits in FQ1 (Apr-Jun).
	now := time.Date(2026, time.May, 15, 0, 0, 0, 0, time.UTC)
	fyStartMonth := time.April
	fyStartDay := 1
	cases := []struct {
		name      string
		direction int
		wantStart time.Time
	}{
		{"this", 0, time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC)},
		{"next", +1, time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)},
		{"last", -1, time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)},
	}
	for _, tc := range cases {
		p := NewRelativeFiscalQuarter(now, fyStartMonth, fyStartDay, tc.direction)
		if p.Kind != PeriodRelativeFiscalQuarter {
			t.Errorf("Kind = %v, want PeriodRelativeFiscalQuarter", p.Kind)
		}
		if !p.Start.Time.Equal(tc.wantStart) {
			t.Errorf("%s fiscal quarter Start = %v, want %v", tc.name, p.Start.Time, tc.wantStart)
		}
	}
}

// TestPeriod_StringAlwaysNonEmpty — every kind must produce a
// readable String() output. Used in interpreter diagnostics.
func TestPeriod_StringAlwaysNonEmpty(t *testing.T) {
	periods := []*Period{
		mustCalendarQuarter(t, 2026, 1),
		mustFiscalQuarter(t, 2026, time.April, 1, 2),
		NewCalendarYear(2026),
		NewFiscalYear(2027, time.April, 1),
		NewCalendarMonth(2026, time.July),
		NewRelativeQuarter(time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC), 0),
		NewRelativeFiscalQuarter(time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC), time.April, 1, 0),
	}
	for _, p := range periods {
		if p.String() == "" {
			t.Errorf("Period{Kind: %v} produced empty String()", p.Kind)
		}
	}
}

// helpers — unwrap the err for less verbose test bodies.
func mustCalendarQuarter(t *testing.T, year, q int) *Period {
	t.Helper()
	p, err := NewCalendarQuarter(year, q)
	if err != nil {
		t.Fatalf("NewCalendarQuarter(%d, %d): %v", year, q, err)
	}
	return p
}

func mustFiscalQuarter(t *testing.T, year int, fyStartMonth time.Month, fyStartDay int, q int) *Period {
	t.Helper()
	fyStart := time.Date(year, fyStartMonth, fyStartDay, 0, 0, 0, 0, time.UTC)
	p, err := NewFiscalQuarter(fyStart, q)
	if err != nil {
		t.Fatalf("NewFiscalQuarter: %v", err)
	}
	return p
}

// --- v2.0 contract tests ---
//
// The v2.0 plan requires Period to store End as a struct field
// (rather than computing it from Kind+context at access time). End
// must be populated by every factory at construction. This pins the
// invariant: any code path producing a *Period must produce a
// fully-formed [Start, End] interval.

func TestPeriod_NewCalendarQuarter_PopulatesEnd(t *testing.T) {
	cases := []struct {
		year, quarter int
		wantEnd       time.Time
	}{
		{2026, 1, time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC)},
		{2026, 2, time.Date(2026, time.June, 30, 0, 0, 0, 0, time.UTC)},
		{2026, 3, time.Date(2026, time.September, 30, 0, 0, 0, 0, time.UTC)},
		{2026, 4, time.Date(2026, time.December, 31, 0, 0, 0, 0, time.UTC)},
	}
	for _, tc := range cases {
		p, err := NewCalendarQuarter(tc.year, tc.quarter)
		if err != nil {
			t.Fatalf("NewCalendarQuarter(%d, %d): %v", tc.year, tc.quarter, err)
		}
		if p.End == nil {
			t.Fatalf("Q%d %d: End is nil — must be populated at construction", tc.quarter, tc.year)
		}
		if !p.End.Time.Equal(tc.wantEnd) {
			t.Errorf("Q%d %d: End = %v, want %v", tc.quarter, tc.year, p.End.Time, tc.wantEnd)
		}
	}
}

func TestPeriod_NewFiscalQuarter_PopulatesEnd(t *testing.T) {
	// FY starts April 1; FQ1 = Apr 1 - Jun 30; FQ4 = Jan 1 - Mar 31 (next cal year).
	apr1 := time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		quarter int
		wantEnd time.Time
	}{
		{1, time.Date(2026, time.June, 30, 0, 0, 0, 0, time.UTC)},
		{2, time.Date(2026, time.September, 30, 0, 0, 0, 0, time.UTC)},
		{3, time.Date(2026, time.December, 31, 0, 0, 0, 0, time.UTC)},
		{4, time.Date(2027, time.March, 31, 0, 0, 0, 0, time.UTC)},
	}
	for _, tc := range cases {
		p, err := NewFiscalQuarter(apr1, tc.quarter)
		if err != nil {
			t.Fatalf("NewFiscalQuarter(Apr 1, %d): %v", tc.quarter, err)
		}
		if p.End == nil {
			t.Fatalf("FQ%d: End is nil", tc.quarter)
		}
		if !p.End.Time.Equal(tc.wantEnd) {
			t.Errorf("FQ%d: End = %v, want %v", tc.quarter, p.End.Time, tc.wantEnd)
		}
	}
}

func TestPeriod_NewCalendarYear_PopulatesEnd(t *testing.T) {
	p := NewCalendarYear(2026)
	if p.End == nil {
		t.Fatalf("End is nil")
	}
	want := time.Date(2026, time.December, 31, 0, 0, 0, 0, time.UTC)
	if !p.End.Time.Equal(want) {
		t.Errorf("End = %v, want %v", p.End.Time, want)
	}
}

func TestPeriod_NewFiscalYear_PopulatesEnd(t *testing.T) {
	// FY2027 with July start = Jul 1 2026 - Jun 30 2027 (default end-year labeling).
	p := NewFiscalYear(2027, time.July, 1)
	if p.End == nil {
		t.Fatalf("End is nil")
	}
	want := time.Date(2027, time.June, 30, 0, 0, 0, 0, time.UTC)
	if !p.End.Time.Equal(want) {
		t.Errorf("End = %v, want %v", p.End.Time, want)
	}
}

func TestPeriod_NewCalendarMonth_PopulatesEnd(t *testing.T) {
	cases := []struct {
		year    int
		month   time.Month
		wantEnd time.Time
	}{
		{2026, time.January, time.Date(2026, time.January, 31, 0, 0, 0, 0, time.UTC)},
		{2026, time.April, time.Date(2026, time.April, 30, 0, 0, 0, 0, time.UTC)},
		{2026, time.February, time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC)}, // non-leap
		{2024, time.February, time.Date(2024, time.February, 29, 0, 0, 0, 0, time.UTC)}, // leap
		{2026, time.December, time.Date(2026, time.December, 31, 0, 0, 0, 0, time.UTC)},
	}
	for _, tc := range cases {
		p := NewCalendarMonth(tc.year, tc.month)
		if p.End == nil {
			t.Fatalf("%v %d: End is nil", tc.month, tc.year)
		}
		if !p.End.Time.Equal(tc.wantEnd) {
			t.Errorf("%v %d: End = %v, want %v", tc.month, tc.year, p.End.Time, tc.wantEnd)
		}
	}
}

func TestPeriod_NewRelativeQuarter_PopulatesEnd(t *testing.T) {
	// May 15 2026 sits in Q2 (Apr-Jun). this/next/last cal-quarter ends:
	now := time.Date(2026, time.May, 15, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		direction int
		wantEnd   time.Time
	}{
		{0, time.Date(2026, time.June, 30, 0, 0, 0, 0, time.UTC)},       // this Q (Q2 2026)
		{+1, time.Date(2026, time.September, 30, 0, 0, 0, 0, time.UTC)}, // next Q (Q3 2026)
		{-1, time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC)},     // last Q (Q1 2026)
	}
	for _, tc := range cases {
		p := NewRelativeQuarter(now, tc.direction)
		if p.End == nil {
			t.Fatalf("direction=%d: End is nil", tc.direction)
		}
		if !p.End.Time.Equal(tc.wantEnd) {
			t.Errorf("direction=%d: End = %v, want %v", tc.direction, p.End.Time, tc.wantEnd)
		}
	}
}

func TestPeriod_NewRelativeFiscalQuarter_PopulatesEnd(t *testing.T) {
	// FY starts April 1; May 15 sits in FQ1 (Apr-Jun).
	now := time.Date(2026, time.May, 15, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		direction int
		wantEnd   time.Time
	}{
		{0, time.Date(2026, time.June, 30, 0, 0, 0, 0, time.UTC)},       // this FQ (FQ1 = Apr-Jun)
		{+1, time.Date(2026, time.September, 30, 0, 0, 0, 0, time.UTC)}, // next FQ (FQ2 = Jul-Sep)
		{-1, time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC)},     // last FQ (FQ4 of prior FY = Jan-Mar)
	}
	for _, tc := range cases {
		p := NewRelativeFiscalQuarter(now, time.April, 1, tc.direction)
		if p.End == nil {
			t.Fatalf("direction=%d: End is nil", tc.direction)
		}
		if !p.End.Time.Equal(tc.wantEnd) {
			t.Errorf("direction=%d: End = %v, want %v", tc.direction, p.End.Time, tc.wantEnd)
		}
	}
}

// --- NewCustomPeriod (PeriodCustom kind) ---
//
// User-defined periods via `between A and B` / `from A to B`. Plan
// U1: PeriodCustom enum constant + NewCustomPeriod factory that
// validates end >= start and stores both endpoints directly.

func TestPeriod_NewCustomPeriod_HappyPath(t *testing.T) {
	start := NewDateFromTime(time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC))
	end := NewDateFromTime(time.Date(2026, time.April, 30, 0, 0, 0, 0, time.UTC))
	p, err := NewCustomPeriod(start, end)
	if err != nil {
		t.Fatalf("NewCustomPeriod: %v", err)
	}
	if p.Kind != PeriodCustom {
		t.Errorf("Kind = %v, want PeriodCustom", p.Kind)
	}
	if !p.Start.Time.Equal(start.Time) {
		t.Errorf("Start = %v, want %v", p.Start.Time, start.Time)
	}
	if !p.End.Time.Equal(end.Time) {
		t.Errorf("End = %v, want %v", p.End.Time, end.Time)
	}
}

func TestPeriod_NewCustomPeriod_SameDay(t *testing.T) {
	// A single-day period (start == end) is the minimal valid period.
	d := NewDateFromTime(time.Date(2026, time.April, 15, 0, 0, 0, 0, time.UTC))
	p, err := NewCustomPeriod(d, d)
	if err != nil {
		t.Fatalf("same-day period should be valid: %v", err)
	}
	if !p.Start.Time.Equal(p.End.Time) {
		t.Errorf("same-day: Start (%v) != End (%v)", p.Start.Time, p.End.Time)
	}
}

func TestPeriod_NewCustomPeriod_EndBeforeStart(t *testing.T) {
	start := NewDateFromTime(time.Date(2026, time.April, 30, 0, 0, 0, 0, time.UTC))
	end := NewDateFromTime(time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC))
	if _, err := NewCustomPeriod(start, end); err == nil {
		t.Errorf("expected error for end-before-start; got nil")
	}
}

func TestPeriod_NewCustomPeriod_NilEndpoint(t *testing.T) {
	d := NewDateFromTime(time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC))
	if _, err := NewCustomPeriod(nil, d); err == nil {
		t.Errorf("expected error for nil start; got nil")
	}
	if _, err := NewCustomPeriod(d, nil); err == nil {
		t.Errorf("expected error for nil end; got nil")
	}
}

func TestPeriod_PeriodCustom_StringNonEmpty(t *testing.T) {
	start := NewDateFromTime(time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC))
	end := NewDateFromTime(time.Date(2026, time.April, 30, 0, 0, 0, 0, time.UTC))
	p, err := NewCustomPeriod(start, end)
	if err != nil {
		t.Fatalf("NewCustomPeriod: %v", err)
	}
	got := p.String()
	if got == "" {
		t.Errorf("String() must not be empty for PeriodCustom")
	}
	// Must surface both endpoints so diagnostics are debuggable.
	if !strings.Contains(got, "2026") {
		t.Errorf("String() = %q should mention the year", got)
	}
}

// TestPeriod_AllFactoriesPopulateEnd is a single-point invariant
// assertion: NO factory in this package may return a *Period with
// nil End. Locks the v2.0 contract — if a future maintainer adds a
// factory that forgets End, this test catches it.
func TestPeriod_AllFactoriesPopulateEnd(t *testing.T) {
	now := time.Date(2026, time.May, 15, 0, 0, 0, 0, time.UTC)
	periods := []*Period{
		mustCalendarQuarter(t, 2026, 1),
		mustFiscalQuarter(t, 2026, time.April, 1, 2),
		NewCalendarYear(2026),
		NewFiscalYear(2027, time.July, 1),
		NewCalendarMonth(2026, time.July),
		NewRelativeQuarter(now, 0),
		NewRelativeFiscalQuarter(now, time.April, 1, 0),
	}
	for i, p := range periods {
		if p.End == nil {
			t.Errorf("periods[%d] (Kind=%v): End is nil — every factory must populate End", i, p.Kind)
		}
		if p.Start == nil {
			t.Errorf("periods[%d] (Kind=%v): Start is nil", i, p.Kind)
		}
		if p.End != nil && p.Start != nil && p.End.Time.Before(p.Start.Time) {
			t.Errorf("periods[%d] (Kind=%v): End (%v) before Start (%v)",
				i, p.Kind, p.End.Time, p.Start.Time)
		}
	}
}
