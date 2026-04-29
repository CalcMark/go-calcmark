package display

import (
	"strings"
	"testing"
	"time"

	"github.com/CalcMark/go-calcmark/spec/types"
	"golang.org/x/text/language"
)

// U14 — display formatter renders *types.Period via FormatPeriod.
// Default form includes both the kind label AND the date range so
// users see concrete boundaries instead of an opaque label like
// "last fiscal quarter".
//
// Format:
//   Named/calendar:   "<label> (<start> – <end>)"
//   Custom:           "<start> – <end>"  (no label; the dates are the value)

func TestFormatPeriod(t *testing.T) {
	f := NewFormatter(DefaultConfig())

	// Calendar quarter — label + range.
	q1, err := types.NewCalendarQuarter(2026, 1)
	if err != nil {
		t.Fatalf("NewCalendarQuarter: %v", err)
	}
	got := f.Format(q1)
	if !strings.HasPrefix(got, "Calendar Q1 2026 (") {
		t.Errorf("Format(Q1 2026) = %q, should start with %q", got, "Calendar Q1 2026 (")
	}
	if !strings.Contains(got, "Jan") || !strings.Contains(got, "Mar") {
		t.Errorf("Format(Q1 2026) = %q, should contain start/end month names", got)
	}

	// Calendar year.
	cy := types.NewCalendarYear(2026)
	got = f.Format(cy)
	if !strings.HasPrefix(got, "Calendar Year 2026 (") {
		t.Errorf("Format(CY2026) = %q, should start with %q", got, "Calendar Year 2026 (")
	}

	// Fiscal year.
	fy := types.NewFiscalYear(2027, time.July, 1)
	got = f.Format(fy)
	if !strings.HasPrefix(got, "Fiscal Year 2027 (") {
		t.Errorf("Format(FY2027) = %q, should start with %q", got, "Fiscal Year 2027 (")
	}
	if !strings.Contains(got, "Jul") || !strings.Contains(got, "Jun") {
		t.Errorf("Format(FY2027) = %q, should mention Jul/Jun boundaries", got)
	}

	// Calendar month.
	apr := types.NewCalendarMonth(2026, time.April)
	got = f.Format(apr)
	if !strings.HasPrefix(got, "April 2026 (") {
		t.Errorf("Format(April 2026) = %q, should start with %q", got, "April 2026 (")
	}

	// Custom period — dates only, no label.
	start := types.NewDateFromTime(time.Date(2026, time.April, 15, 0, 0, 0, 0, time.UTC))
	end := types.NewDateFromTime(time.Date(2026, time.July, 4, 0, 0, 0, 0, time.UTC))
	custom, err := types.NewCustomPeriod(start, end)
	if err != nil {
		t.Fatalf("NewCustomPeriod: %v", err)
	}
	got = f.Format(custom)
	if strings.Contains(got, "Period") || strings.Contains(got, "(") {
		t.Errorf("Format(custom period) = %q, should be dates only without label or parens", got)
	}
	if !strings.Contains(got, "Apr") || !strings.Contains(got, "Jul") {
		t.Errorf("Format(custom period) = %q, should reference Apr and Jul", got)
	}
}

// TestFormatPeriod_RelativeIncludesDates — regression for the
// reported bug: `last fiscal quarter` (and other relative kinds)
// rendered as just the bare kind label, giving no indication of
// which dates the period covered. The display formatter must
// surface concrete boundaries for every relative kind.
func TestFormatPeriod_RelativeIncludesDates(t *testing.T) {
	f := NewFormatter(DefaultConfig())
	now := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC) // Apr 15, 2026
	// fiscal_year_starts: June 1 → FQ4 of FY2026 = Mar 1 - May 31 2026.
	// last fiscal quarter (FQ3) = Dec 1 2025 - Feb 28 2026.
	p := types.NewRelativeFiscalQuarter(now, time.June, 1, -1)
	got := f.Format(p)
	if got == "last fiscal quarter" {
		t.Fatalf("Format returned bare label %q with no date range — regression of the user-reported bug", got)
	}
	if !strings.Contains(got, "last fiscal quarter") {
		t.Errorf("Format(last fiscal quarter) = %q, should contain the kind label", got)
	}
	if !strings.Contains(got, "(") || !strings.Contains(got, ")") {
		t.Errorf("Format(last fiscal quarter) = %q, should include date range in parens", got)
	}
}

// TestFormatPeriod_DatesAreLocalized — the date range in
// FormatPeriod's output goes through FormatDate, which respects
// the formatter's locale (Tag) via getDateFormat. en-US uses
// month-first ("Apr 15"); de-DE uses day-first ("15. Apr."); etc.
//
// This locks the contract that period output IS locale-aware so a
// future refactor can't accidentally hardcode an ASCII format that
// regresses non-English users.
func TestFormatPeriod_DatesAreLocalized(t *testing.T) {
	q1, err := types.NewCalendarQuarter(2026, 1)
	if err != nil {
		t.Fatalf("NewCalendarQuarter: %v", err)
	}

	usCfg := DefaultConfig()
	usCfg.Tag = language.AmericanEnglish
	usOut := NewFormatter(usCfg).Format(q1)

	deCfg := DefaultConfig()
	deCfg.Tag = language.German
	deOut := NewFormatter(deCfg).Format(q1)

	// en-US uses "Jan" (English short month).
	if !strings.Contains(usOut, "Jan") {
		t.Errorf("en-US output %q should contain English `Jan`", usOut)
	}
	// de-DE uses "Jan." (German short month with period; monday lib's
	// LocaleDeDE convention).
	if !strings.Contains(deOut, "Jan.") {
		t.Errorf("de-DE output %q should contain German short-month form `Jan.`", deOut)
	}

	// Outputs must differ — if they're identical the locale path
	// isn't being threaded through.
	if usOut == deOut {
		t.Errorf("en-US and de-DE outputs are identical (%q); locale isn't being applied", usOut)
	}
}
