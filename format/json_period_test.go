package format

import (
	"testing"
	"time"

	"github.com/CalcMark/go-calcmark/spec/types"
)

// U14 — JSON formatter populates period_start, period_end, and
// period_kind fields for *types.Period values. cmw and other
// downstream consumers match against the canonical kind strings.

func TestPopulateResult_PeriodCalendarQuarter(t *testing.T) {
	q1, err := types.NewCalendarQuarter(2026, 1)
	if err != nil {
		t.Fatalf("NewCalendarQuarter: %v", err)
	}
	jr := JSONResult{}
	populateResult(&jr, q1)

	if jr.Type != "period" {
		t.Errorf("Type = %q, want %q", jr.Type, "period")
	}
	if jr.PeriodKind != "calendar_quarter" {
		t.Errorf("PeriodKind = %q, want %q", jr.PeriodKind, "calendar_quarter")
	}
	if jr.PeriodStart != "2026-01-01" {
		t.Errorf("PeriodStart = %q, want %q", jr.PeriodStart, "2026-01-01")
	}
	if jr.PeriodEnd != "2026-03-31" {
		t.Errorf("PeriodEnd = %q, want %q", jr.PeriodEnd, "2026-03-31")
	}
}

func TestPopulateResult_PeriodFiscalYear(t *testing.T) {
	fy := types.NewFiscalYear(2027, time.July, 1)
	jr := JSONResult{}
	populateResult(&jr, fy)

	if jr.PeriodKind != "fiscal_year" {
		t.Errorf("PeriodKind = %q, want %q", jr.PeriodKind, "fiscal_year")
	}
	// FY2027 (Jul start) = Jul 1 2026 to Jun 30 2027.
	if jr.PeriodStart != "2026-07-01" {
		t.Errorf("PeriodStart = %q, want %q", jr.PeriodStart, "2026-07-01")
	}
	if jr.PeriodEnd != "2027-06-30" {
		t.Errorf("PeriodEnd = %q, want %q", jr.PeriodEnd, "2027-06-30")
	}
}

func TestPopulateResult_PeriodCustom(t *testing.T) {
	start := types.NewDateFromTime(time.Date(2026, time.April, 15, 0, 0, 0, 0, time.UTC))
	end := types.NewDateFromTime(time.Date(2026, time.July, 4, 0, 0, 0, 0, time.UTC))
	custom, err := types.NewCustomPeriod(start, end)
	if err != nil {
		t.Fatalf("NewCustomPeriod: %v", err)
	}
	jr := JSONResult{}
	populateResult(&jr, custom)

	if jr.PeriodKind != "custom" {
		t.Errorf("PeriodKind = %q, want %q", jr.PeriodKind, "custom")
	}
	if jr.PeriodStart != "2026-04-15" {
		t.Errorf("PeriodStart = %q, want %q", jr.PeriodStart, "2026-04-15")
	}
	if jr.PeriodEnd != "2026-07-04" {
		t.Errorf("PeriodEnd = %q, want %q", jr.PeriodEnd, "2026-07-04")
	}
}

// TestPeriodKindName_AllKindsCovered guards against drift between
// types.PeriodKind and the JSON-name registry. If a new PeriodKind
// is added to spec/types/ but periodKindName isn't extended, this
// test surfaces the gap by failing on "unknown".
func TestPeriodKindName_AllKindsCovered(t *testing.T) {
	// Iterate from PeriodCalendarQuarter (iota=0) through
	// PeriodCustom (iota=12). If types.PeriodCustom changes ordinal,
	// update this loop.
	last := types.PeriodCustom
	for k := types.PeriodCalendarQuarter; k <= last; k++ {
		got := periodKindName(k)
		if got == "unknown" {
			t.Errorf("PeriodKind ordinal %d → %q (no registered name)", k, got)
		}
	}
}
