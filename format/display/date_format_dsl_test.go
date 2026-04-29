package display

import (
	"testing"
	"time"

	"github.com/CalcMark/go-calcmark/spec/types"
	"golang.org/x/text/language"
)

// Tests for the user-facing date format DSL — verifies the full
// DateFormat / PeriodDateFormat config flow from a DSL string
// through to formatted output, plus that locale month names still
// flow through the custom layout.

func TestDateFormatDSL_TokenTranslation(t *testing.T) {
	// April 4, 2026 (a Saturday) — distinctive across all token slots.
	tt := time.Date(2026, time.April, 4, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		dsl, want string
	}{
		// User's example.
		{"MON dd, YYYY", "APR 04, 2026"},
		// Common ISO-ish.
		{"YYYY-MM-dd", "2026-04-04"},
		// Compact dd-MON-YYYY (matches Period default).
		{"dd-MON-YYYY", "04-APR-2026"},
		// Lowercase month.
		{"dd mon YYYY", "04 apr 2026"},
		// Full names.
		{"EEEE, MMMM d, YYYY", "Saturday, April 4, 2026"},
		// 2-digit year.
		{"d/M/YY", "4/4/26"},
		// Short weekday + month.
		{"EEE MMM dd", "Sat Apr 04"},
		// Quoted literal letters.
		{"'T'YYYY-MM-dd", "T2026-04-04"},
		// Mix of MON and other tokens.
		{"YYYY-MON-dd", "2026-APR-04"},
	}
	for _, tc := range cases {
		t.Run(tc.dsl, func(t *testing.T) {
			got := formatDateWithDSL(tt, tc.dsl, defaultDateFormat.locale)
			if got != tc.want {
				t.Errorf("formatDateWithDSL(%q) = %q, want %q", tc.dsl, got, tc.want)
			}
		})
	}
}

func TestDateFormatDSL_LocalePassthrough(t *testing.T) {
	tt := time.Date(2026, time.April, 4, 0, 0, 0, 0, time.UTC)
	dsl := "MON dd, YYYY"

	usLocale := dateFormats["en_US"].locale
	frLocale := dateFormats["fr_FR"].locale

	usOut := formatDateWithDSL(tt, dsl, usLocale)
	frOut := formatDateWithDSL(tt, dsl, frLocale)

	// en-US: "APR 04, 2026"
	if usOut != "APR 04, 2026" {
		t.Errorf("en-US output = %q, want %q", usOut, "APR 04, 2026")
	}
	// fr-FR: locale's short month for April is "avr." → "AVR. 04, 2026"
	// (uppercase preserves the dot).
	if frOut == usOut {
		t.Errorf("fr-FR output should differ from en-US; both are %q", usOut)
	}
}

func TestDisplayConfig_DateFormatOverride(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DateFormat = "MON dd, YYYY"
	f := NewFormatter(cfg)

	d := types.NewDateFromTime(time.Date(2026, time.April, 4, 0, 0, 0, 0, time.UTC))
	got := f.FormatDate(d)
	if got != "APR 04, 2026" {
		t.Errorf("FormatDate with DateFormat=%q = %q, want %q",
			cfg.DateFormat, got, "APR 04, 2026")
	}
}

func TestDisplayConfig_DateFormatOverrideAppliesToPeriod(t *testing.T) {
	// When DateFormat is set but PeriodDateFormat is NOT, the period
	// output uses DateFormat too. Mirrors the user's expectation
	// that "set the date format in config" affects all date-bearing
	// output uniformly.
	cfg := DefaultConfig()
	cfg.DateFormat = "MON dd, YYYY"
	f := NewFormatter(cfg)

	q1, err := types.NewCalendarQuarter(2026, 1)
	if err != nil {
		t.Fatalf("NewCalendarQuarter: %v", err)
	}
	got := f.Format(q1)
	want := "JAN 01, 2026 – MAR 31, 2026"
	if got != want {
		t.Errorf("Format(Q1) with DateFormat override = %q, want %q", got, want)
	}
}

func TestDisplayConfig_PeriodDateFormatOverride(t *testing.T) {
	// PeriodDateFormat takes precedence over DateFormat for periods.
	cfg := DefaultConfig()
	cfg.DateFormat = "MMMM d, YYYY"   // verbose for single dates
	cfg.PeriodDateFormat = "YYYY-MM-dd" // compact for periods
	f := NewFormatter(cfg)

	d := types.NewDateFromTime(time.Date(2026, time.April, 4, 0, 0, 0, 0, time.UTC))
	if got, want := f.FormatDate(d), "April 4, 2026"; got != want {
		t.Errorf("FormatDate = %q, want %q", got, want)
	}

	q1, _ := types.NewCalendarQuarter(2026, 1)
	if got, want := f.Format(q1), "2026-01-01 – 2026-03-31"; got != want {
		t.Errorf("Format(Q1) = %q, want %q", got, want)
	}
}

func TestDisplayConfig_NoOverride_UsesLocaleDefault(t *testing.T) {
	// Empty DateFormat / PeriodDateFormat → existing locale behavior.
	cfg := DefaultConfig()
	cfg.Tag = language.AmericanEnglish
	f := NewFormatter(cfg)

	d := types.NewDateFromTime(time.Date(2026, time.April, 4, 0, 0, 0, 0, time.UTC))
	if got := f.FormatDate(d); got == "" {
		t.Error("FormatDate with empty DateFormat returned empty string")
	}

	q1, _ := types.NewCalendarQuarter(2026, 1)
	got := f.Format(q1)
	// Default Period format is dd-MON-YYYY.
	if got != "01-Jan-2026 – 31-Mar-2026" {
		t.Errorf("Format(Q1) with no override = %q, want default %q",
			got, "01-Jan-2026 – 31-Mar-2026")
	}
}
