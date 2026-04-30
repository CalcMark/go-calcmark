package display

import (
	"strings"
	"testing"
	"time"

	"github.com/CalcMark/go-calcmark/spec/types"
	"golang.org/x/text/language"
)

// U14 — display formatter renders *types.Period via FormatPeriod
// as a compact dd-MON-YYYY range. No kind label: the source
// already names the period (`this fiscal quarter`, `Q1`), and
// echoing the label is noise — same reason `today` doesn't echo
// "today" in its formatted Date output.

func TestFormatPeriod(t *testing.T) {
	f := NewFormatter(DefaultConfig())

	// Calendar quarter Q1 2026 = Jan 1 – Mar 31.
	q1, err := types.NewCalendarQuarter(2026, 1)
	if err != nil {
		t.Fatalf("NewCalendarQuarter: %v", err)
	}
	if got, want := f.Format(q1), "01-Jan-2026 – 31-Mar-2026"; got != want {
		t.Errorf("Format(Q1 2026) = %q, want %q", got, want)
	}

	// Calendar year 2026.
	cy := types.NewCalendarYear(2026)
	if got, want := f.Format(cy), "01-Jan-2026 – 31-Dec-2026"; got != want {
		t.Errorf("Format(CY2026) = %q, want %q", got, want)
	}

	// Fiscal year 2027 (July start) = Jul 1 2026 – Jun 30 2027.
	fy := types.NewFiscalYear(2027, time.July, 1)
	if got, want := f.Format(fy), "01-Jul-2026 – 30-Jun-2027"; got != want {
		t.Errorf("Format(FY2027) = %q, want %q", got, want)
	}

	// Calendar month April 2026.
	apr := types.NewCalendarMonth(2026, time.April)
	if got, want := f.Format(apr), "01-Apr-2026 – 30-Apr-2026"; got != want {
		t.Errorf("Format(April 2026) = %q, want %q", got, want)
	}

	// Custom period.
	start := types.NewDateFromTime(time.Date(2026, time.April, 15, 0, 0, 0, 0, time.UTC))
	end := types.NewDateFromTime(time.Date(2026, time.July, 4, 0, 0, 0, 0, time.UTC))
	custom, err := types.NewCustomPeriod(start, end)
	if err != nil {
		t.Fatalf("NewCustomPeriod: %v", err)
	}
	if got, want := f.Format(custom), "15-Apr-2026 – 04-Jul-2026"; got != want {
		t.Errorf("Format(custom) = %q, want %q", got, want)
	}
}

// TestFormatPeriod_RelativeShowsOnlyDates — regression for the
// reported bug: `this fiscal quarter` rendered as
// "this fiscal quarter (Wed, Apr 1, 2026 – Tue, Jun 30, 2026)" —
// echoing the source line and using the verbose Date format.
// FormatPeriod must surface ONLY the resolved dates in the compact
// dd-MON-YYYY layout.
func TestFormatPeriod_RelativeShowsOnlyDates(t *testing.T) {
	f := NewFormatter(DefaultConfig())
	now := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)
	p := types.NewRelativeFiscalQuarter(now, time.July, 1, 0) // FY starts July
	got := f.Format(p)

	if strings.Contains(got, "fiscal quarter") {
		t.Errorf("Format(this fiscal quarter) = %q, must not echo the kind label", got)
	}
	if strings.Contains(got, "(") || strings.Contains(got, ")") {
		t.Errorf("Format(this fiscal quarter) = %q, no parens — just dates", got)
	}
	if strings.Contains(got, "Wed") || strings.Contains(got, "Tue") {
		t.Errorf("Format(this fiscal quarter) = %q, must use compact dd-MON-YYYY (no day-of-week)", got)
	}
	// FY starts July; April 15 sits in FQ4 (Apr-Jun) of FY2026.
	want := "01-Apr-2026 – 30-Jun-2026"
	if got != want {
		t.Errorf("Format(this fiscal quarter) = %q, want %q", got, want)
	}
}

// TestFormatPeriod_MonthAbbrevIsLocalized — the dd-MON-YYYY layout
// still uses the locale's month-name abbreviation via the monday
// library, so non-English users see e.g. "01-mars-2026" (fr) /
// "01-Mär-2026" (de) / "01-Jan-2026" (en) for the same dates.
// Locks the locale contract so a future refactor can't accidentally
// hardcode English month names.
func TestFormatPeriod_MonthAbbrevIsLocalized(t *testing.T) {
	q3, err := types.NewCalendarQuarter(2026, 3) // Jul–Sep, distinctive across locales
	if err != nil {
		t.Fatalf("NewCalendarQuarter: %v", err)
	}

	usCfg := DefaultConfig()
	usCfg.Tag = language.AmericanEnglish
	usOut := NewFormatter(usCfg).Format(q3)

	frCfg := DefaultConfig()
	frCfg.Tag = language.French
	frOut := NewFormatter(frCfg).Format(q3)

	// en-US: "Jul" for July.
	if !strings.Contains(usOut, "Jul") {
		t.Errorf("en-US output %q should contain English `Jul`", usOut)
	}
	// fr-FR: "juil." for juillet (monday LocaleFrFR convention).
	if !strings.Contains(frOut, "juil") {
		t.Errorf("fr-FR output %q should contain French `juil` (juillet abbrev)", frOut)
	}

	if usOut == frOut {
		t.Errorf("en-US and fr-FR outputs are identical (%q); locale isn't being applied", usOut)
	}
}
