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

func TestPeriod_NewFiscalYear(t *testing.T) {
	// FY2027 with July start = starts Jul 1, 2026. (Microsoft labeling.)
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
