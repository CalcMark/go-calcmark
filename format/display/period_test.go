package display

import (
	"testing"
	"time"

	"github.com/CalcMark/go-calcmark/spec/types"
)

// U14 — display formatter renders *types.Period via Period.String().
// The String() method is the single source of truth for human-
// readable Period output; this test pins what each kind renders to
// so accidental drift surfaces.

func TestFormatPeriod(t *testing.T) {
	f := NewFormatter(DefaultConfig())

	// Calendar quarter.
	q1, err := types.NewCalendarQuarter(2026, 1)
	if err != nil {
		t.Fatalf("NewCalendarQuarter: %v", err)
	}
	if got := f.Format(q1); got != "Calendar Q1 2026" {
		t.Errorf("Format(Q1 2026) = %q, want %q", got, "Calendar Q1 2026")
	}

	// Calendar year.
	cy := types.NewCalendarYear(2026)
	if got := f.Format(cy); got != "Calendar Year 2026" {
		t.Errorf("Format(CY2026) = %q, want %q", got, "Calendar Year 2026")
	}

	// Fiscal year.
	fy := types.NewFiscalYear(2027, time.July, 1)
	if got := f.Format(fy); got != "Fiscal Year 2027" {
		t.Errorf("Format(FY2027) = %q, want %q", got, "Fiscal Year 2027")
	}

	// Calendar month.
	apr := types.NewCalendarMonth(2026, time.April)
	if got := f.Format(apr); got != "April 2026" {
		t.Errorf("Format(April 2026) = %q, want %q", got, "April 2026")
	}

	// Custom period — `between Apr 15 2026 and Jul 4 2026`.
	start := types.NewDateFromTime(time.Date(2026, time.April, 15, 0, 0, 0, 0, time.UTC))
	end := types.NewDateFromTime(time.Date(2026, time.July, 4, 0, 0, 0, 0, time.UTC))
	custom, err := types.NewCustomPeriod(start, end)
	if err != nil {
		t.Fatalf("NewCustomPeriod: %v", err)
	}
	got := f.Format(custom)
	want := "2026-04-15 to 2026-07-04"
	if got != want {
		t.Errorf("Format(custom period) = %q, want %q", got, want)
	}
}
