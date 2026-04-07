package interpreter

import (
	"strings"
	"testing"
	"time"

	"github.com/CalcMark/go-calcmark/spec/parser"
	"github.com/CalcMark/go-calcmark/spec/types"
)

// ---------------------------------------------------------------------------
// Date creation rules: what IS a date and what is NOT.
// These tests document and enforce the type boundary contract.
// ---------------------------------------------------------------------------

// TestValidDateCreation verifies every way a user can create a date value.
func TestValidDateCreation(t *testing.T) {
	clock := testClock // Wednesday, April 8, 2026

	tests := []struct {
		name  string
		input string
	}{
		// Date literals with month names
		{"full month + day + year", "d = January 15 2026\n"},
		{"abbrev month + day + year", "d = Jan 15 2026\n"},
		{"month + day (no year)", "d = Dec 25\n"},
		{"month + day only", "d = Jul 4\n"},

		// Keywords
		{"today", "d = today\n"},
		{"tomorrow", "d = tomorrow\n"},
		{"yesterday", "d = yesterday\n"},
		{"now", "d = now\n"},

		// Relative weekday
		{"next Friday", "d = next Friday\n"},
		{"last Tuesday", "d = last Tuesday\n"},
		{"this Monday", "d = this Monday\n"},
		{"bare Wednesday", "d = Wednesday\n"},

		// Relative periods
		{"this week", "d = this week\n"},
		{"next month", "d = next month\n"},
		{"last year", "d = last year\n"},

		// Relative month names
		{"next April", "d = next April\n"},
		{"last Dec", "d = last Dec\n"},
		{"this September", "d = this September\n"},

		// Calendar quarters
		{"this quarter", "d = this quarter\n"},
		{"next quarter", "d = next quarter\n"},

		// End/start of
		{"end of this month", "d = end of this month\n"},
		{"end of this quarter", "d = end of this quarter\n"},
		{"start of this year", "d = start of this year\n"},

		// Ago
		{"2 weeks ago", "d = 2 weeks ago\n"},
		{"3 months ago", "d = 3 months ago\n"},

		// From
		{"3 days from today", "d = 3 days from today\n"},
		{"2 weeks from now", "d = 2 weeks from now\n"},
		{"1 month from next Friday", "d = 1 month from next Friday\n"},

		// Arithmetic producing date
		{"date + duration", "d = Jan 1 2026 + 90 days\n"},
		{"date - duration", "d = Jun 1 2026 - 2 weeks\n"},
		{"date + months", "d = Jan 31 2026 + 1 month\n"},
		{"date + years", "d = Feb 29 2024 + 1 year\n"},
		{"date + hours", "d = Apr 1 2026 + 2 hours\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interp := newTestInterpreterWithClock(clock)

			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval(%q) error = %v", tt.input, err)
			}

			if len(results) != 1 {
				t.Fatalf("Expected 1 result, got %d", len(results))
			}

			if _, ok := results[0].(*types.Date); !ok {
				t.Errorf("Expected *types.Date, got %T (%v)", results[0], results[0])
			}
		})
	}
}

// TestNotADate verifies expressions that should NOT produce date values.
func TestNotADate(t *testing.T) {
	clock := testClock

	tests := []struct {
		name     string
		input    string
		wantType string // "number", "duration", "error", "markdown"
	}{
		// Bare numbers are never dates
		{"bare year 2019", "d = 2019\n", "number"},
		{"bare year 2025", "d = 2025\n", "number"},
		{"bare number 1990", "d = 1990\n", "number"},

		// Duration literals are durations, not dates
		{"5 days literal", "d = 5 days\n", "duration"},
		{"3 months literal", "d = 3 months\n", "duration"},
		{"1 year literal", "d = 1 year\n", "duration"},

		// Date subtraction produces duration
		{"date minus date", "d = Jan 1 2026 - Dec 1 2025\n", "duration"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interp := newTestInterpreterWithClock(clock)

			nodes, err := parser.Parse(tt.input)
			if err != nil {
				if tt.wantType == "error" {
					return // expected parse error
				}
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			results, err := interp.Eval(nodes)
			if err != nil {
				if tt.wantType == "error" {
					return // expected eval error
				}
				t.Fatalf("Eval(%q) error = %v", tt.input, err)
			}

			if len(results) != 1 {
				t.Fatalf("Expected 1 result, got %d", len(results))
			}

			result := results[0]
			if _, isDate := result.(*types.Date); isDate {
				t.Errorf("%q produced a *types.Date — bare numbers and durations must NOT be dates", tt.input)
			}

			switch tt.wantType {
			case "number":
				if _, ok := result.(*types.Number); !ok {
					t.Errorf("Expected *types.Number, got %T", result)
				}
			case "duration":
				if _, ok := result.(*types.Duration); !ok {
					t.Errorf("Expected *types.Duration, got %T", result)
				}
			}
		})
	}
}

// TestDateCreationErrors verifies expressions that should produce errors.
func TestDateCreationErrors(t *testing.T) {
	clock := testClock

	tests := []struct {
		name      string
		input     string
		wantInErr string // substring expected in error message
	}{
		// Fiscal without frontmatter
		{"fiscal quarter no config", "d = this fiscal quarter\n", "fiscal_year_starts"},
		{"fiscal year no config", "d = this fiscal year\n", "fiscal_year_starts"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interp := newTestInterpreterWithClock(clock)

			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			_, err = interp.Eval(nodes)
			if err == nil {
				t.Fatalf("Expected error for %q, got nil", tt.input)
			}

			if tt.wantInErr != "" && !strings.Contains(err.Error(), tt.wantInErr) {
				t.Errorf("Error %q should contain %q", err.Error(), tt.wantInErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Clock injection and date evaluation tests
// ---------------------------------------------------------------------------

// pinnedClock returns a time function that always returns the given time.
func pinnedClock(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

// newTestInterpreterWithClock creates an interpreter with a fixed clock for deterministic testing.
func newTestInterpreterWithClock(clock time.Time) *Interpreter {
	interp := NewInterpreter()
	interp.SetTimeFunc(pinnedClock(clock))
	return interp
}

// Reference date for pinned-clock tests: Wednesday, April 8, 2026 14:30:00 UTC.
var testClock = time.Date(2026, 4, 8, 14, 30, 0, 0, time.UTC)

// TestDateKeywords tests relative date keyword evaluation with a pinned clock.
func TestDateKeywords(t *testing.T) {
	now := testClock

	tests := []struct {
		name      string
		input     string
		wantYear  int
		wantMonth int
		wantDay   int
	}{
		{
			name:      "today",
			input:     "d = today\n",
			wantYear:  now.Year(),
			wantMonth: int(now.Month()),
			wantDay:   now.Day(),
		},
		{
			name:      "tomorrow",
			input:     "d = tomorrow\n",
			wantYear:  now.AddDate(0, 0, 1).Year(),
			wantMonth: int(now.AddDate(0, 0, 1).Month()),
			wantDay:   now.AddDate(0, 0, 1).Day(),
		},
		{
			name:      "yesterday",
			input:     "d = yesterday\n",
			wantYear:  now.AddDate(0, 0, -1).Year(),
			wantMonth: int(now.AddDate(0, 0, -1).Month()),
			wantDay:   now.AddDate(0, 0, -1).Day(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interp := newTestInterpreterWithClock(now)

			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error = %v", err)
			}

			if len(results) != 1 {
				t.Fatalf("Expected 1 result, got %d", len(results))
			}

			date, ok := results[0].(*types.Date)
			if !ok {
				t.Fatalf("Expected *types.Date, got %T", results[0])
			}

			gotYear := date.Time.Year()
			gotMonth := int(date.Time.Month())
			gotDay := date.Time.Day()
			if gotYear != tt.wantYear || gotMonth != tt.wantMonth || gotDay != tt.wantDay {
				t.Errorf("Got date %d-%02d-%02d, want %d-%02d-%02d",
					gotYear, gotMonth, gotDay,
					tt.wantYear, tt.wantMonth, tt.wantDay)
			}
		})
	}
}

// TestPinnedClockIsolation verifies that the clock injection works across year boundaries.
func TestPinnedClockIsolation(t *testing.T) {
	tests := []struct {
		name      string
		clock     time.Time
		input     string
		wantYear  int
		wantMonth int
		wantDay   int
	}{
		{
			name:      "tomorrow across year boundary",
			clock:     time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
			input:     "d = tomorrow\n",
			wantYear:  2027,
			wantMonth: 1,
			wantDay:   1,
		},
		{
			name:      "yesterday across year boundary",
			clock:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			input:     "d = yesterday\n",
			wantYear:  2025,
			wantMonth: 12,
			wantDay:   31,
		},
		{
			name:      "date literal without year uses clock year",
			clock:     time.Date(2030, 6, 15, 0, 0, 0, 0, time.UTC),
			input:     "d = Dec 25\n",
			wantYear:  2030,
			wantMonth: 12,
			wantDay:   25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interp := newTestInterpreterWithClock(tt.clock)

			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error = %v", err)
			}

			if len(results) != 1 {
				t.Fatalf("Expected 1 result, got %d", len(results))
			}

			date, ok := results[0].(*types.Date)
			if !ok {
				t.Fatalf("Expected *types.Date, got %T", results[0])
			}

			gotYear := date.Time.Year()
			gotMonth := int(date.Time.Month())
			gotDay := date.Time.Day()
			if gotYear != tt.wantYear || gotMonth != tt.wantMonth || gotDay != tt.wantDay {
				t.Errorf("Got date %d-%02d-%02d, want %d-%02d-%02d",
					gotYear, gotMonth, gotDay,
					tt.wantYear, tt.wantMonth, tt.wantDay)
			}
		})
	}
}

// TestDateLiterals tests Month Day [Year] date literals.
func TestDateLiterals(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantYear  int // 0 means current year
		wantMonth int
		wantDay   int
	}{
		{"Dec 25", "d = Dec 25\n", 0, 12, 25},
		{"December 25", "d = December 25\n", 0, 12, 25},
		{"Jan 1", "d = Jan 1\n", 0, 1, 1},
		{"February 28", "d = February 28\n", 0, 2, 28},
		{"Dec 25 2025", "d = Dec 25 2025\n", 2025, 12, 25},
		{"January 1 2026", "d = January 1 2026\n", 2026, 1, 1},
		{"Jul 4 2024", "d = Jul 4 2024\n", 2024, 7, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interp := NewInterpreter()

			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error = %v", err)
			}

			if len(results) != 1 {
				t.Fatalf("Expected 1 result, got %d", len(results))
			}

			date, ok := results[0].(*types.Date)
			if !ok {
				t.Fatalf("Expected *types.Date, got %T", results[0])
			}

			wantYear := tt.wantYear
			if wantYear == 0 {
				wantYear = time.Now().Year()
			}

			gotYear := date.Time.Year()
			gotMonth := int(date.Time.Month())
			gotDay := date.Time.Day()
			if gotYear != wantYear || gotMonth != tt.wantMonth || gotDay != tt.wantDay {
				t.Errorf("Got date %d-%02d-%02d, want %d-%02d-%02d",
					gotYear, gotMonth, gotDay,
					wantYear, tt.wantMonth, tt.wantDay)
			}
		})
	}
}

// TestDateArithmeticEval tests date +/- duration expressions with pinned clock.
func TestDateArithmeticEval(t *testing.T) {
	now := testClock // Wednesday, April 8, 2026

	tests := []struct {
		name      string
		input     string
		wantYear  int
		wantMonth int
		wantDay   int
	}{
		{
			name:      "today + 2 days",
			input:     "d = today + 2 days\n",
			wantYear:  now.AddDate(0, 0, 2).Year(),
			wantMonth: int(now.AddDate(0, 0, 2).Month()),
			wantDay:   now.AddDate(0, 0, 2).Day(),
		},
		{
			name:      "today - 3 days",
			input:     "d = today - 3 days\n",
			wantYear:  now.AddDate(0, 0, -3).Year(),
			wantMonth: int(now.AddDate(0, 0, -3).Month()),
			wantDay:   now.AddDate(0, 0, -3).Day(),
		},
		{
			name:      "today + 1 week",
			input:     "d = today + 1 week\n",
			wantYear:  now.AddDate(0, 0, 7).Year(),
			wantMonth: int(now.AddDate(0, 0, 7).Month()),
			wantDay:   now.AddDate(0, 0, 7).Day(),
		},
		{
			name:      "today + 2 weeks",
			input:     "d = today + 2 weeks\n",
			wantYear:  now.AddDate(0, 0, 14).Year(),
			wantMonth: int(now.AddDate(0, 0, 14).Month()),
			wantDay:   now.AddDate(0, 0, 14).Day(),
		},
		{
			name:      "tomorrow + 1 day",
			input:     "d = tomorrow + 1 day\n",
			wantYear:  now.AddDate(0, 0, 2).Year(),
			wantMonth: int(now.AddDate(0, 0, 2).Month()),
			wantDay:   now.AddDate(0, 0, 2).Day(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interp := newTestInterpreterWithClock(now)

			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error = %v", err)
			}

			if len(results) != 1 {
				t.Fatalf("Expected 1 result, got %d", len(results))
			}

			date, ok := results[0].(*types.Date)
			if !ok {
				t.Fatalf("Expected *types.Date, got %T", results[0])
			}

			gotYear := date.Time.Year()
			gotMonth := int(date.Time.Month())
			gotDay := date.Time.Day()
			if gotYear != tt.wantYear || gotMonth != tt.wantMonth || gotDay != tt.wantDay {
				t.Errorf("Got date %d-%02d-%02d, want %d-%02d-%02d",
					gotYear, gotMonth, gotDay,
					tt.wantYear, tt.wantMonth, tt.wantDay)
			}
		})
	}
}

// TestCalendarCorrectMonthArithmetic tests unit-aware month/year arithmetic.
// This verifies calendar-correct behavior (not 30-day approximation).
func TestCalendarCorrectMonthArithmetic(t *testing.T) {
	tests := []struct {
		name      string
		clock     time.Time
		input     string
		wantYear  int
		wantMonth int
		wantDay   int
	}{
		{
			name:      "Jan 31 + 1 month = Feb 28 (non-leap)",
			clock:     testClock,
			input:     "d = Jan 31 2026 + 1 month\n",
			wantYear:  2026,
			wantMonth: 2,
			wantDay:   28,
		},
		{
			name:      "Jan 31 + 1 month = Feb 29 (leap year)",
			clock:     testClock,
			input:     "d = Jan 31 2024 + 1 month\n",
			wantYear:  2024,
			wantMonth: 2,
			wantDay:   29,
		},
		{
			name:      "Apr 6 - 3 months = Jan 6 (calendar correct)",
			clock:     time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC),
			input:     "d = today - 3 months\n",
			wantYear:  2026,
			wantMonth: 1,
			wantDay:   6,
		},
		{
			name:      "today + 1 year across leap year",
			clock:     time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC),
			input:     "d = today + 1 year\n",
			wantYear:  2025,
			wantMonth: 2,
			wantDay:   28,
		},
		{
			name:      "Dec 15 + 1 month = Jan 15 next year",
			clock:     testClock,
			input:     "d = Dec 15 2026 + 1 month\n",
			wantYear:  2027,
			wantMonth: 1,
			wantDay:   15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interp := newTestInterpreterWithClock(tt.clock)

			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error = %v", err)
			}

			if len(results) != 1 {
				t.Fatalf("Expected 1 result, got %d", len(results))
			}

			date, ok := results[0].(*types.Date)
			if !ok {
				t.Fatalf("Expected *types.Date, got %T", results[0])
			}

			gotYear := date.Time.Year()
			gotMonth := int(date.Time.Month())
			gotDay := date.Time.Day()
			if gotYear != tt.wantYear || gotMonth != tt.wantMonth || gotDay != tt.wantDay {
				t.Errorf("Got date %d-%02d-%02d, want %d-%02d-%02d",
					gotYear, gotMonth, gotDay,
					tt.wantYear, tt.wantMonth, tt.wantDay)
			}
		})
	}
}

// TestDurationLiterals tests duration literal evaluation.
func TestDurationLiterals(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantSeconds int64
	}{
		{"2 days", "d = 2 days\n", 2 * 24 * 60 * 60},
		{"3 weeks", "d = 3 weeks\n", 3 * 7 * 24 * 60 * 60},
		{"1 hour", "d = 1 hour\n", 60 * 60},
		{"30 minutes", "d = 30 minutes\n", 30 * 60},
		{"1 year", "d = 1 year\n", 365 * 24 * 60 * 60},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interp := NewInterpreter()

			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error = %v", err)
			}

			if len(results) != 1 {
				t.Fatalf("Expected 1 result, got %d", len(results))
			}

			dur, ok := results[0].(*types.Duration)
			if !ok {
				t.Fatalf("Expected *types.Duration, got %T", results[0])
			}

			gotSeconds := dur.ToSeconds().IntPart()
			if gotSeconds != tt.wantSeconds {
				t.Errorf("Got %d seconds, want %d seconds", gotSeconds, tt.wantSeconds)
			}
		})
	}
}

// TestDateLiteralArithmetic tests date literal + duration expressions.
func TestDateLiteralArithmetic(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantYear  int
		wantMonth int
		wantDay   int
	}{
		{
			name:      "Dec 25 + 7 days (year rollover)",
			input:     "d = Dec 25 + 7 days\n",
			wantYear:  time.Now().Year() + 1, // Rolls over to next year
			wantMonth: 1,                     // January 1
			wantDay:   1,
		},
		{
			name:      "Dec 25 2025 + 7 days",
			input:     "d = Dec 25 2025 + 7 days\n",
			wantYear:  2026,
			wantMonth: 1,
			wantDay:   1,
		},
		{
			name:      "Jan 1 2026 - 1 day",
			input:     "d = Jan 1 2026 - 1 day\n",
			wantYear:  2025,
			wantMonth: 12,
			wantDay:   31,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interp := NewInterpreter()

			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error = %v", err)
			}

			if len(results) != 1 {
				t.Fatalf("Expected 1 result, got %d", len(results))
			}

			date, ok := results[0].(*types.Date)
			if !ok {
				t.Fatalf("Expected *types.Date, got %T", results[0])
			}

			gotYear := date.Time.Year()
			gotMonth := int(date.Time.Month())
			gotDay := date.Time.Day()
			if gotYear != tt.wantYear || gotMonth != tt.wantMonth || gotDay != tt.wantDay {
				t.Errorf("Got date %d-%02d-%02d, want %d-%02d-%02d",
					gotYear, gotMonth, gotDay,
					tt.wantYear, tt.wantMonth, tt.wantDay)
			}
		})
	}
}

// TestDateDifference tests date - date expressions.
// left - right should produce (left - right) days with correct sign.
func TestDateDifference(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantDays int64
	}{
		{
			name:     "Jan 2 2025 - Jan 1 2025 = 1 day",
			input:    "d = Jan 2 2025 - Jan 1 2025\n",
			wantDays: 1, // Later date - earlier date = positive
		},
		{
			name:     "Jan 1 2025 - Jan 8 2025 = -7 days",
			input:    "d = Jan 1 2025 - Jan 8 2025\n",
			wantDays: -7, // Earlier date - later date = negative
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interp := NewInterpreter()

			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error = %v", err)
			}

			if len(results) != 1 {
				t.Fatalf("Expected 1 result, got %d", len(results))
			}

			dur, ok := results[0].(*types.Duration)
			if !ok {
				t.Fatalf("Expected *types.Duration, got %T", results[0])
			}

			gotDays := dur.Value.IntPart()
			if gotDays != tt.wantDays {
				t.Errorf("Got %d days, want %d days", gotDays, tt.wantDays)
			}
		})
	}
}

// TestCurrencyDividedByDurationVariable tests that dividing a currency by a
// duration variable produces a rate: $1453.84 / 4 days = $363.46/day.
// Using number(days) gives plain currency division instead.
func TestCurrencyDividedByDurationVariable(t *testing.T) {
	input := `arrive = Apr 22
depart = April 26
days = depart - arrive
total = $1,453.84
daily = total / days
`
	interp := NewInterpreter()

	nodes, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}

	// Find the "daily" result (last one)
	daily := results[len(results)-1]

	rate, ok := daily.(*types.Rate)
	if !ok {
		t.Fatalf("Expected daily to be *types.Rate, got %T (%v)", daily, daily)
	}

	// $1,453.84 / 4 days = $363.46/day
	expected := "363.46"
	got := rate.Amount.Value.StringFixed(2)
	if got != expected {
		t.Errorf("Expected rate amount = %s, got %s", expected, got)
	}
	if rate.Amount.Unit != "$" {
		t.Errorf("Expected rate unit '$', got '%s'", rate.Amount.Unit)
	}
	if rate.PerUnit != "day" {
		t.Errorf("Expected per unit 'day', got '%s'", rate.PerUnit)
	}
}

// TestTimeLiterals tests time literal parsing and evaluation.
// NOTE: Standalone time expressions like "10:30" don't produce results
// because they're parsed as ratio expressions (10 divided by 30).
// Time literals work in context (e.g., "meeting at 10:30AM").
func TestTimeLiterals(t *testing.T) {
	t.Skip("Standalone time literals parse as division; need context-aware testing")

	tests := []struct {
		name       string
		input      string
		wantHour   int
		wantMinute int
	}{
		{"10:30", "t = 10:30\n", 10, 30},
		{"14:00", "t = 14:00\n", 14, 0},
		{"10:30AM", "t = 10:30AM\n", 10, 30},
		{"10:30PM", "t = 10:30PM\n", 22, 30},
		{"12:00PM", "t = 12:00PM\n", 12, 0}, // Noon
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interp := NewInterpreter()

			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error = %v", err)
			}

			if len(results) != 1 {
				t.Fatalf("Expected 1 result, got %d", len(results))
			}

			tm, ok := results[0].(*types.Time)
			if !ok {
				t.Fatalf("Expected *types.Time, got %T", results[0])
			}

			gotHour := tm.Time.Hour()
			gotMinute := tm.Time.Minute()
			if gotHour != tt.wantHour || gotMinute != tt.wantMinute {
				t.Errorf("Got %02d:%02d, want %02d:%02d",
					gotHour, gotMinute, tt.wantHour, tt.wantMinute)
			}
		})
	}
}

// TestExtendedRelativeDates tests this/next/last week/month/year keywords.
// Clock pinned to Wednesday, April 8, 2026. Evaluation not yet implemented —
// these tests confirm parsing works and will fail at evaluation until Unit 5.
func TestExtendedRelativeDates(t *testing.T) {
	clock := testClock // Wednesday, April 8, 2026

	tests := []struct {
		name      string
		input     string
		wantYear  int
		wantMonth int
		wantDay   int
	}{
		// this week = Monday of current week = April 6, 2026
		{"this week", "d = this week\n", 2026, 4, 6},
		// this month = 1st of current month = April 1, 2026
		{"this month", "d = this month\n", 2026, 4, 1},
		// this year = Jan 1 of current year
		{"this year", "d = this year\n", 2026, 1, 1},
		// next week = Monday of next week = April 13, 2026
		{"next week", "d = next week\n", 2026, 4, 13},
		// next month = 1st of next month = May 1, 2026
		{"next month", "d = next month\n", 2026, 5, 1},
		// next year = Jan 1 of next year
		{"next year", "d = next year\n", 2027, 1, 1},
		// last week = Monday of last week = March 30, 2026
		{"last week", "d = last week\n", 2026, 3, 30},
		// last month = 1st of last month = March 1, 2026
		{"last month", "d = last month\n", 2026, 3, 1},
		// last year = Jan 1 of last year
		{"last year", "d = last year\n", 2025, 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interp := newTestInterpreterWithClock(clock)

			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error = %v", err)
			}

			if len(results) != 1 {
				t.Fatalf("Expected 1 result, got %d", len(results))
			}

			date, ok := results[0].(*types.Date)
			if !ok {
				t.Fatalf("Expected *types.Date, got %T", results[0])
			}

			gotYear := date.Time.Year()
			gotMonth := int(date.Time.Month())
			gotDay := date.Time.Day()
			if gotYear != tt.wantYear || gotMonth != tt.wantMonth || gotDay != tt.wantDay {
				t.Errorf("Got date %d-%02d-%02d, want %d-%02d-%02d",
					gotYear, gotMonth, gotDay,
					tt.wantYear, tt.wantMonth, tt.wantDay)
			}
		})
	}
}

// TestXFromYSyntax tests "X from Y" date expressions with pinned clock.
func TestXFromYSyntax(t *testing.T) {
	now := testClock // Wednesday, April 8, 2026

	tests := []struct {
		name      string
		input     string
		wantYear  int
		wantMonth int
		wantDay   int
	}{
		{
			name:      "2 days from today",
			input:     "d = 2 days from today\n",
			wantYear:  now.AddDate(0, 0, 2).Year(),
			wantMonth: int(now.AddDate(0, 0, 2).Month()),
			wantDay:   now.AddDate(0, 0, 2).Day(),
		},
		{
			name:      "1 week from tomorrow",
			input:     "d = 1 week from tomorrow\n",
			wantYear:  now.AddDate(0, 0, 8).Year(),
			wantMonth: int(now.AddDate(0, 0, 8).Month()),
			wantDay:   now.AddDate(0, 0, 8).Day(),
		},
		{
			name:      "7 days from Dec 25 2025",
			input:     "d = 7 days from Dec 25 2025\n",
			wantYear:  2026,
			wantMonth: 1,
			wantDay:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interp := newTestInterpreterWithClock(now)

			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error = %v", err)
			}

			if len(results) != 1 {
				t.Fatalf("Expected 1 result, got %d", len(results))
			}

			date, ok := results[0].(*types.Date)
			if !ok {
				t.Fatalf("Expected *types.Date, got %T", results[0])
			}

			gotYear := date.Time.Year()
			gotMonth := int(date.Time.Month())
			gotDay := date.Time.Day()
			if gotYear != tt.wantYear || gotMonth != tt.wantMonth || gotDay != tt.wantDay {
				t.Errorf("Got date %d-%02d-%02d, want %d-%02d-%02d",
					gotYear, gotMonth, gotDay,
					tt.wantYear, tt.wantMonth, tt.wantDay)
			}
		})
	}
}

// TestWeekdayExpressions tests this/next/last <weekday> and bare weekday.
// Clock pinned to Wednesday, April 8, 2026.
func TestWeekdayExpressions(t *testing.T) {
	clock := testClock // Wednesday, April 8, 2026

	tests := []struct {
		name      string
		input     string
		wantYear  int
		wantMonth int
		wantDay   int
	}{
		// Bare weekday = this <weekday>
		{"bare Friday", "d = Friday\n", 2026, 4, 10},   // this week's Friday
		{"bare Monday", "d = Monday\n", 2026, 4, 6},    // this week's Monday (past)
		{"bare Wednesday", "d = Wednesday\n", 2026, 4, 8}, // today

		// This <weekday> = occurrence in current calendar week (Mon-Sun)
		{"this Friday", "d = this Friday\n", 2026, 4, 10},
		{"this Monday", "d = this Monday\n", 2026, 4, 6},    // past within week
		{"this Sunday", "d = this Sunday\n", 2026, 4, 12},   // future within week
		{"this Wednesday", "d = this Wednesday\n", 2026, 4, 8}, // today

		// Next <weekday> = soonest future occurrence (skip today if same day)
		{"next Friday on Wed", "d = next Friday\n", 2026, 4, 10},   // 2 days ahead
		{"next Wednesday on Wed", "d = next Wednesday\n", 2026, 4, 15}, // skip today, next week
		{"next Monday on Wed", "d = next Monday\n", 2026, 4, 13},   // next week

		// Last <weekday> = most recent past (skip today if same day)
		{"last Monday on Wed", "d = last Monday\n", 2026, 4, 6},     // 2 days ago
		{"last Wednesday on Wed", "d = last Wednesday\n", 2026, 4, 1}, // skip today, last week
		{"last Friday on Wed", "d = last Friday\n", 2026, 4, 3},     // last week

		// Year boundary
		{"next Monday on Dec 31 Wed", "d = next Monday\n", 2027, 1, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := clock
			if strings.Contains(tt.name, "Dec 31") {
				c = time.Date(2026, 12, 30, 0, 0, 0, 0, time.UTC) // Wednesday Dec 30
			}
			interp := newTestInterpreterWithClock(c)

			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error = %v", err)
			}

			if len(results) != 1 {
				t.Fatalf("Expected 1 result, got %d", len(results))
			}

			date, ok := results[0].(*types.Date)
			if !ok {
				t.Fatalf("Expected *types.Date, got %T", results[0])
			}

			gotYear := date.Time.Year()
			gotMonth := int(date.Time.Month())
			gotDay := date.Time.Day()
			if gotYear != tt.wantYear || gotMonth != tt.wantMonth || gotDay != tt.wantDay {
				t.Errorf("Got date %d-%02d-%02d, want %d-%02d-%02d",
					gotYear, gotMonth, gotDay,
					tt.wantYear, tt.wantMonth, tt.wantDay)
			}
		})
	}
}

// TestRelativeMonthExpressions tests this/next/last <month>.
// Clock pinned to Wednesday, April 8, 2026.
func TestRelativeMonthExpressions(t *testing.T) {
	clock := testClock // April 8, 2026

	tests := []struct {
		name      string
		input     string
		wantYear  int
		wantMonth int
		wantDay   int
	}{
		// this <month> = 1st of that month, current year
		{"this April", "d = this April\n", 2026, 4, 1},
		{"this January", "d = this January\n", 2026, 1, 1},
		{"this December", "d = this December\n", 2026, 12, 1},

		// next <month> = 1st of that month, next occurrence after current month
		{"next April on Apr", "d = next April\n", 2027, 4, 1},     // April is current month → next year
		{"next March on Apr", "d = next March\n", 2027, 3, 1},     // March is past → next year
		{"next May on Apr", "d = next May\n", 2026, 5, 1},         // May is future → this year
		{"next Jan on Apr", "d = next Jan\n", 2027, 1, 1},         // Jan is past → next year
		{"next Dec on Apr", "d = next Dec\n", 2026, 12, 1},        // Dec is future → this year

		// last <month> = 1st of that month, most recent past occurrence
		{"last April on Apr", "d = last April\n", 2025, 4, 1},     // April is current month → last year
		{"last March on Apr", "d = last March\n", 2026, 3, 1},     // March is past → this year
		{"last May on Apr", "d = last May\n", 2025, 5, 1},         // May is future → last year
		{"last Dec on Apr", "d = last Dec\n", 2025, 12, 1},        // Dec is future → last year

		// Abbreviations work
		{"next Sept", "d = next Sept\n", 2026, 9, 1},
		{"last Feb", "d = last Feb\n", 2026, 2, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interp := newTestInterpreterWithClock(clock)

			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error = %v", err)
			}

			if len(results) != 1 {
				t.Fatalf("Expected 1 result, got %d", len(results))
			}

			date, ok := results[0].(*types.Date)
			if !ok {
				t.Fatalf("Expected *types.Date, got %T", results[0])
			}

			gotYear := date.Time.Year()
			gotMonth := int(date.Time.Month())
			gotDay := date.Time.Day()
			if gotYear != tt.wantYear || gotMonth != tt.wantMonth || gotDay != tt.wantDay {
				t.Errorf("Got date %d-%02d-%02d, want %d-%02d-%02d",
					gotYear, gotMonth, gotDay,
					tt.wantYear, tt.wantMonth, tt.wantDay)
			}
		})
	}
}

// TestFiscalWithDayOffset tests fiscal_year_starts with a day offset (e.g., "July 15").
// A fiscal year starting July 15 means FQ1 runs Jul 15 - Oct 14, not Jul 1 - Sep 30.
func TestFiscalWithDayOffset(t *testing.T) {
	tests := []struct {
		name             string
		clock            time.Time
		fiscalStartMonth int
		fiscalStartDay   int
		input            string
		wantYear         int
		wantMonth        int
		wantDay          int
	}{
		// FY starts July 15. On Aug 1, "this fiscal year" = Jul 15 of current year
		{"this fiscal year Jul 15 start", time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), 7, 15,
			"d = this fiscal year\n", 2026, 7, 15},
		// On Jul 10 (before Jul 15), "this fiscal year" = Jul 15 of PREVIOUS year
		{"this fiscal year before start", time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC), 7, 15,
			"d = this fiscal year\n", 2025, 7, 15},
		// "end of this fiscal year" from Aug = Jul 14 of next year (day before next FY start)
		{"end of this fiscal year Jul 15", time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), 7, 15,
			"d = end of this fiscal year\n", 2027, 7, 14},
		// Month-only (day=1) still works — backward compatible
		{"backward compat month only", time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), 7, 1,
			"d = this fiscal year\n", 2026, 7, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interp := newTestInterpreterWithClock(tt.clock)
			interp.SetFiscalYearStarts(tt.fiscalStartMonth, tt.fiscalStartDay)

			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error = %v", err)
			}

			date, ok := results[0].(*types.Date)
			if !ok {
				t.Fatalf("Expected *types.Date, got %T", results[0])
			}

			gotYear := date.Time.Year()
			gotMonth := int(date.Time.Month())
			gotDay := date.Time.Day()
			if gotYear != tt.wantYear || gotMonth != tt.wantMonth || gotDay != tt.wantDay {
				t.Errorf("Got date %d-%02d-%02d, want %d-%02d-%02d",
					gotYear, gotMonth, gotDay,
					tt.wantYear, tt.wantMonth, tt.wantDay)
			}
		})
	}
}

// TestAgoExpressions tests "<N> <unit> ago" syntax.
func TestAgoExpressions(t *testing.T) {
	clock := testClock // Wednesday, April 8, 2026 14:30

	tests := []struct {
		name      string
		input     string
		wantYear  int
		wantMonth int
		wantDay   int
	}{
		{"2 weeks ago", "d = 2 weeks ago\n", 2026, 3, 25},
		{"3 months ago", "d = 3 months ago\n", 2026, 1, 8},
		{"1 year ago", "d = 1 year ago\n", 2025, 4, 8},
		{"7 days ago", "d = 7 days ago\n", 2026, 4, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interp := newTestInterpreterWithClock(clock)

			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error = %v", err)
			}

			if len(results) != 1 {
				t.Fatalf("Expected 1 result, got %d", len(results))
			}

			date, ok := results[0].(*types.Date)
			if !ok {
				t.Fatalf("Expected *types.Date, got %T", results[0])
			}

			gotYear := date.Time.Year()
			gotMonth := int(date.Time.Month())
			gotDay := date.Time.Day()
			if gotYear != tt.wantYear || gotMonth != tt.wantMonth || gotDay != tt.wantDay {
				t.Errorf("Got date %d-%02d-%02d, want %d-%02d-%02d",
					gotYear, gotMonth, gotDay,
					tt.wantYear, tt.wantMonth, tt.wantDay)
			}
		})
	}
}

// TestSubDayAgoExpressions tests that "N hours/minutes ago" uses wall clock time, not midnight.
func TestSubDayAgoExpressions(t *testing.T) {
	// Clock at 14:30 — "10 minutes ago" should be 14:20, NOT 23:50 yesterday
	clock := time.Date(2026, 4, 8, 14, 30, 0, 0, time.UTC)

	tests := []struct {
		name     string
		input    string
		wantDay  int
		wantHour int
		wantMin  int
		wantHas  bool
	}{
		{"10 minutes ago", "d = 10 minutes ago\n", 8, 14, 20, true},
		{"2 hours ago", "d = 2 hours ago\n", 8, 12, 30, true},
		{"7 days ago (preserves time from now)", "d = 7 days ago\n", 1, 14, 30, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interp := newTestInterpreterWithClock(clock)

			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error = %v", err)
			}

			date, ok := results[0].(*types.Date)
			if !ok {
				t.Fatalf("Expected *types.Date, got %T", results[0])
			}

			if date.Time.Day() != tt.wantDay {
				t.Errorf("Day = %d, want %d", date.Time.Day(), tt.wantDay)
			}
			if date.Time.Hour() != tt.wantHour {
				t.Errorf("Hour = %d, want %d", date.Time.Hour(), tt.wantHour)
			}
			if date.Time.Minute() != tt.wantMin {
				t.Errorf("Minute = %d, want %d", date.Time.Minute(), tt.wantMin)
			}
			if date.HasTime != tt.wantHas {
				t.Errorf("HasTime = %v, want %v", date.HasTime, tt.wantHas)
			}
		})
	}
}

// TestEndOfExpressions tests "end of" period expressions.
func TestEndOfExpressions(t *testing.T) {
	tests := []struct {
		name      string
		clock     time.Time
		input     string
		wantYear  int
		wantMonth int
		wantDay   int
	}{
		// End of month
		{"end of this month (Apr)", testClock, "d = end of this month\n", 2026, 4, 30},
		{"end of this month (Feb non-leap)", time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC),
			"d = end of this month\n", 2026, 2, 28},
		{"end of this month (Feb leap)", time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC),
			"d = end of this month\n", 2024, 2, 29},
		{"end of next month (Apr→May)", testClock, "d = end of next month\n", 2026, 5, 31},
		{"end of last month (Apr→Mar)", testClock, "d = end of last month\n", 2026, 3, 31},

		// End of quarter
		{"end of this quarter (Q2 Apr)", testClock, "d = end of this quarter\n", 2026, 6, 30},
		{"end of this quarter (Q1 Jan)", time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
			"d = end of this quarter\n", 2026, 3, 31},
		{"end of this quarter (Q4 Dec)", time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC),
			"d = end of this quarter\n", 2026, 12, 31},
		{"end of next quarter from Q2", testClock, "d = end of next quarter\n", 2026, 9, 30},
		{"end of last quarter from Q2", testClock, "d = end of last quarter\n", 2026, 3, 31},

		// End of year
		{"end of this year", testClock, "d = end of this year\n", 2026, 12, 31},
		{"end of next year", testClock, "d = end of next year\n", 2027, 12, 31},
		{"end of last year", testClock, "d = end of last year\n", 2025, 12, 31},

		// End of week (Sunday)
		{"end of this week (Wed)", testClock, "d = end of this week\n", 2026, 4, 12}, // Sunday

		// End of named month
		{"end of January", testClock, "d = end of this January\n", 2026, 1, 31},
		{"end of next February (non-leap)", testClock, "d = end of next February\n", 2027, 2, 28},

		// Start of (explicit, equivalent to bare period)
		{"start of this month", testClock, "d = start of this month\n", 2026, 4, 1},
		{"start of this quarter", testClock, "d = start of this quarter\n", 2026, 4, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interp := newTestInterpreterWithClock(tt.clock)

			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error = %v", err)
			}

			if len(results) != 1 {
				t.Fatalf("Expected 1 result, got %d", len(results))
			}

			date, ok := results[0].(*types.Date)
			if !ok {
				t.Fatalf("Expected *types.Date, got %T", results[0])
			}

			gotYear := date.Time.Year()
			gotMonth := int(date.Time.Month())
			gotDay := date.Time.Day()
			if gotYear != tt.wantYear || gotMonth != tt.wantMonth || gotDay != tt.wantDay {
				t.Errorf("Got date %d-%02d-%02d, want %d-%02d-%02d",
					gotYear, gotMonth, gotDay,
					tt.wantYear, tt.wantMonth, tt.wantDay)
			}
		})
	}
}

// TestEndOfComposition tests "end of" composes with arithmetic.
func TestEndOfComposition(t *testing.T) {
	interp := newTestInterpreterWithClock(testClock) // April 8, 2026

	// "end of this quarter - 2 weeks" = Jun 30 - 14 days = Jun 16
	nodes, err := parser.Parse("d = end of this quarter - 2 weeks\n")
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}

	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval error = %v", err)
	}

	date := results[0].(*types.Date)
	if date.Time.Month() != 6 || date.Time.Day() != 16 {
		t.Errorf("end of this quarter - 2 weeks: got %s, want Jun 16",
			date.Time.Format("2006-01-02"))
	}
}

// TestFromNow tests "N units from now" syntax.
func TestFromNow(t *testing.T) {
	clock := time.Date(2026, 4, 8, 14, 30, 0, 0, time.UTC)
	interp := newTestInterpreterWithClock(clock)

	nodes, err := parser.Parse("d = 2 weeks from now\n")
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}

	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval error = %v", err)
	}

	date := results[0].(*types.Date)
	// "now" preserves time, so 2 weeks from now at 14:30 = Apr 22 14:30
	if date.Time.Day() != 22 || date.Time.Month() != 4 {
		t.Errorf("2 weeks from now: got %s, want Apr 22",
			date.Time.Format("2006-01-02"))
	}
	if !date.HasTime {
		t.Error("'from now' should produce HasTime=true (now preserves time)")
	}
	if date.Time.Hour() != 14 || date.Time.Minute() != 30 {
		t.Errorf("Time should be 14:30, got %s", date.Time.Format("15:04"))
	}
}

// TestExtendedFromTargets tests "from" with new relative date targets.
func TestExtendedFromTargets(t *testing.T) {
	clock := testClock // Wednesday, April 8, 2026

	tests := []struct {
		name      string
		input     string
		wantYear  int
		wantMonth int
		wantDay   int
	}{
		{
			name:      "3 days from next Friday",
			input:     "d = 3 days from next Friday\n",
			wantYear:  2026,
			wantMonth: 4,
			wantDay:   13, // next Friday = Apr 10, + 3 = Apr 13
		},
		{
			name:      "2 weeks from next month",
			input:     "d = 2 weeks from next month\n",
			wantYear:  2026,
			wantMonth: 5,
			wantDay:   15, // next month = May 1, + 14 = May 15
		},
		{
			name:      "1 month from this year",
			input:     "d = 1 month from this year\n",
			wantYear:  2026,
			wantMonth: 2,
			wantDay:   1, // this year = Jan 1, + 1 month = Feb 1
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interp := newTestInterpreterWithClock(clock)

			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error = %v", err)
			}

			if len(results) != 1 {
				t.Fatalf("Expected 1 result, got %d", len(results))
			}

			date, ok := results[0].(*types.Date)
			if !ok {
				t.Fatalf("Expected *types.Date, got %T", results[0])
			}

			gotYear := date.Time.Year()
			gotMonth := int(date.Time.Month())
			gotDay := date.Time.Day()
			if gotYear != tt.wantYear || gotMonth != tt.wantMonth || gotDay != tt.wantDay {
				t.Errorf("Got date %d-%02d-%02d, want %d-%02d-%02d",
					gotYear, gotMonth, gotDay,
					tt.wantYear, tt.wantMonth, tt.wantDay)
			}
		})
	}
}

// TestCalendarQuarterExpressions tests this/next/last quarter.
func TestCalendarQuarterExpressions(t *testing.T) {
	tests := []struct {
		name      string
		clock     time.Time
		input     string
		wantYear  int
		wantMonth int
		wantDay   int
	}{
		// Q1 (Jan-Mar): this quarter = Jan 1
		{"this quarter in Feb", time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC), "d = this quarter\n", 2026, 1, 1},
		// Q2 (Apr-Jun): this quarter = Apr 1
		{"this quarter in Apr", time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), "d = this quarter\n", 2026, 4, 1},
		// Q3 (Jul-Sep): this quarter = Jul 1
		{"this quarter in Jul", time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC), "d = this quarter\n", 2026, 7, 1},
		// Q4 (Oct-Dec): this quarter = Oct 1
		{"this quarter in Oct", time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC), "d = this quarter\n", 2026, 10, 1},

		// next quarter from Q4 = Q1 next year
		{"next quarter from Nov", time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC), "d = next quarter\n", 2027, 1, 1},
		// next quarter from Q1 = Q2
		{"next quarter from Feb", time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC), "d = next quarter\n", 2026, 4, 1},

		// last quarter from Q1 = Q4 of previous year
		{"last quarter from Jan", time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), "d = last quarter\n", 2025, 10, 1},
		// last quarter from Q3 = Q2
		{"last quarter from Aug", time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC), "d = last quarter\n", 2026, 4, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interp := newTestInterpreterWithClock(tt.clock)

			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error = %v", err)
			}

			if len(results) != 1 {
				t.Fatalf("Expected 1 result, got %d", len(results))
			}

			date, ok := results[0].(*types.Date)
			if !ok {
				t.Fatalf("Expected *types.Date, got %T", results[0])
			}

			gotYear := date.Time.Year()
			gotMonth := int(date.Time.Month())
			gotDay := date.Time.Day()
			if gotYear != tt.wantYear || gotMonth != tt.wantMonth || gotDay != tt.wantDay {
				t.Errorf("Got date %d-%02d-%02d, want %d-%02d-%02d",
					gotYear, gotMonth, gotDay,
					tt.wantYear, tt.wantMonth, tt.wantDay)
			}
		})
	}
}

// TestFiscalExpressions tests fiscal quarter and fiscal year with configured start month.
func TestFiscalExpressions(t *testing.T) {
	tests := []struct {
		name             string
		clock            time.Time
		fiscalStartMonth int
		input            string
		wantYear         int
		wantMonth        int
		wantDay          int
	}{
		// Microsoft FY starts July. Aug 2026 = FQ1 of FY starting Jul 2026
		{"this fiscal quarter Aug+July", time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC), 7,
			"d = this fiscal quarter\n", 2026, 7, 1},
		// Oct 2026 = FQ2 (Oct-Dec)
		{"this fiscal quarter Oct+July", time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC), 7,
			"d = this fiscal quarter\n", 2026, 10, 1},
		// next fiscal quarter from Aug (FQ1) = Oct 1 (FQ2)
		{"next fiscal quarter Aug+July", time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC), 7,
			"d = next fiscal quarter\n", 2026, 10, 1},
		// this fiscal year in Aug with July start = Jul 1 2026
		{"this fiscal year Aug+July", time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC), 7,
			"d = this fiscal year\n", 2026, 7, 1},
		// this fiscal year in May with July start = Jul 1 2025 (previous FY)
		{"this fiscal year May+July", time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC), 7,
			"d = this fiscal year\n", 2025, 7, 1},
		// fiscal_year_starts: january → fiscal = calendar
		{"fiscal=calendar Jan start", time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC), 1,
			"d = this fiscal quarter\n", 2026, 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interp := newTestInterpreterWithClock(tt.clock)
			interp.SetFiscalYearStarts(tt.fiscalStartMonth, 1)

			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error = %v", err)
			}

			if len(results) != 1 {
				t.Fatalf("Expected 1 result, got %d", len(results))
			}

			date, ok := results[0].(*types.Date)
			if !ok {
				t.Fatalf("Expected *types.Date, got %T", results[0])
			}

			gotYear := date.Time.Year()
			gotMonth := int(date.Time.Month())
			gotDay := date.Time.Day()
			if gotYear != tt.wantYear || gotMonth != tt.wantMonth || gotDay != tt.wantDay {
				t.Errorf("Got date %d-%02d-%02d, want %d-%02d-%02d",
					gotYear, gotMonth, gotDay,
					tt.wantYear, tt.wantMonth, tt.wantDay)
			}
		})
	}
}

// TestFiscalMissingConfig tests error when fiscal expressions lack frontmatter.
func TestFiscalMissingConfig(t *testing.T) {
	interp := newTestInterpreterWithClock(testClock)
	// No SetFiscalYearStarts call

	nodes, err := parser.Parse("d = this fiscal quarter\n")
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}

	_, err = interp.Eval(nodes)
	if err == nil {
		t.Fatal("Expected error for fiscal expression without frontmatter, got nil")
	}
	if !strings.Contains(err.Error(), "fiscal_year_starts") {
		t.Errorf("Error should mention fiscal_year_starts, got: %v", err)
	}
}

// TestWeekdayComposition tests weekday expressions compose with duration arithmetic.
func TestWeekdayComposition(t *testing.T) {
	clock := testClock // Wednesday, April 8, 2026

	interp := newTestInterpreterWithClock(clock)

	// next Friday + 2 weeks = April 10 + 14 = April 24
	nodes, err := parser.Parse("d = next Friday + 2 weeks\n")
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}

	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval error = %v", err)
	}

	date := results[0].(*types.Date)
	if date.Time.Year() != 2026 || date.Time.Month() != 4 || date.Time.Day() != 24 {
		t.Errorf("next Friday + 2 weeks: got %s, want 2026-04-24",
			date.Time.Format("2006-01-02"))
	}
}

// TestDateStringFormat tests date output formatting with pinned clock.
// Date.String() returns long format: "Monday, January 2, 2006"
func TestDateStringFormat(t *testing.T) {
	clock := testClock // Wednesday, April 8, 2026

	tests := []struct {
		name        string
		input       string
		wantContain string
	}{
		{"Dec 25 2025 format", "d = Dec 25 2025\n", "December 25, 2025"},
		{"today format", "d = today\n", clock.Format("January 2, 2006")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interp := newTestInterpreterWithClock(clock)

			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error = %v", err)
			}

			if len(results) != 1 {
				t.Fatalf("Expected 1 result, got %d", len(results))
			}

			str := results[0].String()
			if !strings.Contains(str, tt.wantContain) {
				t.Errorf("String() = %q, want to contain %q", str, tt.wantContain)
			}
		})
	}
}

// TestAddMonthsClipped tests the month-clipping helper directly.
func TestAddMonthsClipped(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		months   int
		wantDate string // "2006-01-02"
	}{
		{"Jan 31 + 1 month = Feb 28 (non-leap)", time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC), 1, "2026-02-28"},
		{"Jan 31 + 1 month = Feb 29 (leap)", time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC), 1, "2024-02-29"},
		{"Mar 31 - 1 month = Feb 28", time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC), -1, "2026-02-28"},
		{"Jan 15 + 12 months = Jan 15 next year", time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), 12, "2027-01-15"},
		{"Feb 29 + 12 months = Feb 28 (leap to non-leap)", time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC), 12, "2025-02-28"},
		{"Jan 31 - 14 months = Nov 30 (large negative)", time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC), -14, "2024-11-30"},
		{"Jul 31 + 25 months = Aug 31 (multi-year)", time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC), 25, "2028-08-31"},
		{"Dec 31 + 2 months = Feb 28 (year rollover + clip)", time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC), 2, "2026-02-28"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := addMonthsClipped(tt.input, tt.months)
			got := result.Format("2006-01-02")
			if got != tt.wantDate {
				t.Errorf("addMonthsClipped(%s, %d) = %s, want %s",
					tt.input.Format("2006-01-02"), tt.months, got, tt.wantDate)
			}
		})
	}
}

// TestSubDayDurationArithmetic tests date + sub-day duration produces HasTime=true.
func TestSubDayDurationArithmetic(t *testing.T) {
	clock := testClock // Wednesday, April 8, 2026 14:30 UTC

	tests := []struct {
		name     string
		input    string
		wantHour int
		wantDay  int
		wantHas  bool
	}{
		{
			name:     "today + 2 hours",
			input:    "d = today + 2 hours\n",
			wantHour: 2,
			wantDay:  8,
			wantHas:  true,
		},
		{
			name:     "today + 30 minutes",
			input:    "d = today + 30 minutes\n",
			wantHour: 0,
			wantDay:  8,
			wantHas:  true,
		},
		{
			name:     "today + 1 day (remains date-only)",
			input:    "d = today + 1 day\n",
			wantHour: 0,
			wantDay:  9,
			wantHas:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interp := newTestInterpreterWithClock(clock)

			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error = %v", err)
			}

			date, ok := results[0].(*types.Date)
			if !ok {
				t.Fatalf("Expected *types.Date, got %T", results[0])
			}

			if date.HasTime != tt.wantHas {
				t.Errorf("HasTime = %v, want %v", date.HasTime, tt.wantHas)
			}
			if date.Time.Hour() != tt.wantHour {
				t.Errorf("Hour = %d, want %d", date.Time.Hour(), tt.wantHour)
			}
			if date.Time.Day() != tt.wantDay {
				t.Errorf("Day = %d, want %d", date.Time.Day(), tt.wantDay)
			}
		})
	}
}

// TestFiscalYearAndQuarterCompleteCoverage tests all fiscal modifier combinations.
func TestFiscalYearAndQuarterCompleteCoverage(t *testing.T) {
	tests := []struct {
		name             string
		clock            time.Time
		fiscalStartMonth int
		input            string
		wantYear         int
		wantMonth        int
		wantDay          int
	}{
		// next fiscal year from Aug 2026 with July start = Jul 1, 2027
		{"next fiscal year", time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC), 7,
			"d = next fiscal year\n", 2027, 7, 1},
		// last fiscal year from Aug 2026 with July start = Jul 1, 2025
		{"last fiscal year", time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC), 7,
			"d = last fiscal year\n", 2025, 7, 1},
		// last fiscal quarter from Aug (FQ1=Jul-Sep) = Apr 1 (FQ4 of previous FY)
		{"last fiscal quarter from Aug", time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC), 7,
			"d = last fiscal quarter\n", 2026, 4, 1},
		// next fiscal quarter from Jan (FQ3=Jan-Mar) with July start = Apr 1 (FQ4)
		{"next fiscal quarter from Jan", time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC), 7,
			"d = next fiscal quarter\n", 2027, 4, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interp := newTestInterpreterWithClock(tt.clock)
			interp.SetFiscalYearStarts(tt.fiscalStartMonth, 1)

			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error = %v", err)
			}

			date, ok := results[0].(*types.Date)
			if !ok {
				t.Fatalf("Expected *types.Date, got %T", results[0])
			}

			gotYear := date.Time.Year()
			gotMonth := int(date.Time.Month())
			gotDay := date.Time.Day()
			if gotYear != tt.wantYear || gotMonth != tt.wantMonth || gotDay != tt.wantDay {
				t.Errorf("Got date %d-%02d-%02d, want %d-%02d-%02d",
					gotYear, gotMonth, gotDay,
					tt.wantYear, tt.wantMonth, tt.wantDay)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FY/FQ/Q/CY notation tests
// ---------------------------------------------------------------------------

// TestCalendarQuarterNotation tests Q1-Q4 shorthand.
func TestCalendarQuarterNotation(t *testing.T) {
	clock := testClock // April 8, 2026

	tests := []struct {
		name      string
		input     string
		wantYear  int
		wantMonth int
		wantDay   int
	}{
		{"Q1", "d = Q1\n", 2026, 1, 1},
		{"Q2", "d = Q2\n", 2026, 4, 1},
		{"Q3", "d = Q3\n", 2026, 7, 1},
		{"Q4", "d = Q4\n", 2026, 10, 1},
		{"q1 lowercase", "d = q1\n", 2026, 1, 1},
		{"q3 lowercase", "d = q3\n", 2026, 7, 1},
		{"Q1 + 30 days", "d = Q1 + 30 days\n", 2026, 1, 31},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interp := newTestInterpreterWithClock(clock)
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}
			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval(%q) error = %v", tt.input, err)
			}
			if len(results) != 1 {
				t.Fatalf("Expected 1 result, got %d", len(results))
			}
			date, ok := results[0].(*types.Date)
			if !ok {
				t.Fatalf("Expected *types.Date, got %T (%v)", results[0], results[0])
			}
			if date.Time.Year() != tt.wantYear || int(date.Time.Month()) != tt.wantMonth || date.Time.Day() != tt.wantDay {
				t.Errorf("Got %d-%02d-%02d, want %d-%02d-%02d",
					date.Time.Year(), int(date.Time.Month()), date.Time.Day(),
					tt.wantYear, tt.wantMonth, tt.wantDay)
			}
		})
	}
}

// TestFiscalQuarterNotation tests FQ1-FQ4 shorthand.
func TestFiscalQuarterNotation(t *testing.T) {
	clock := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		input     string
		wantYear  int
		wantMonth int
		wantDay   int
	}{
		// fiscal_year_starts: july → FQ1=Jul, FQ2=Oct, FQ3=Jan, FQ4=Apr
		{"FQ1", "d = FQ1\n", 2026, 7, 1},
		{"FQ2", "d = FQ2\n", 2026, 10, 1},
		{"FQ3", "d = FQ3\n", 2027, 1, 1},
		{"FQ4", "d = FQ4\n", 2027, 4, 1},
		{"fq1 lowercase", "d = fq1\n", 2026, 7, 1},
		{"fq4 lowercase", "d = fq4\n", 2027, 4, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interp := newTestInterpreterWithClock(clock)
			interp.SetFiscalYearStarts(7, 1)
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}
			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval(%q) error = %v", tt.input, err)
			}
			date, ok := results[0].(*types.Date)
			if !ok {
				t.Fatalf("Expected *types.Date, got %T (%v)", results[0], results[0])
			}
			if date.Time.Year() != tt.wantYear || int(date.Time.Month()) != tt.wantMonth || date.Time.Day() != tt.wantDay {
				t.Errorf("Got %d-%02d-%02d, want %d-%02d-%02d",
					date.Time.Year(), int(date.Time.Month()), date.Time.Day(),
					tt.wantYear, tt.wantMonth, tt.wantDay)
			}
		})
	}
}

// TestFiscalQuarterNotationMissingConfig tests FQ without frontmatter.
func TestFiscalQuarterNotationMissingConfig(t *testing.T) {
	interp := newTestInterpreterWithClock(testClock)
	nodes, err := parser.Parse("d = FQ1\n")
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}
	_, err = interp.Eval(nodes)
	if err == nil {
		t.Fatal("Expected error for FQ1 without fiscal config")
	}
	if !strings.Contains(err.Error(), "fiscal_year_starts") {
		t.Errorf("Error should mention fiscal_year_starts, got: %v", err)
	}
}

// TestFiscalYearNotation tests FY26, FY2026 shorthand.
// Convention: FY<N> = fiscal year starting in calendar year N at the configured start date.
func TestFiscalYearNotation(t *testing.T) {
	clock := time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		input     string
		wantYear  int
		wantMonth int
		wantDay   int
	}{
		{"FY2026 full", "d = FY2026\n", 2026, 7, 1},
		{"FY2025 full", "d = FY2025\n", 2025, 7, 1},
		{"FY26 short", "d = FY26\n", 2026, 7, 1},
		{"FY25 short", "d = FY25\n", 2025, 7, 1},
		{"fy2026 lowercase", "d = fy2026\n", 2026, 7, 1},
		{"fy26 lowercase", "d = fy26\n", 2026, 7, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interp := newTestInterpreterWithClock(clock)
			interp.SetFiscalYearStarts(7, 1)
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}
			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval(%q) error = %v", tt.input, err)
			}
			date, ok := results[0].(*types.Date)
			if !ok {
				t.Fatalf("Expected *types.Date, got %T (%v)", results[0], results[0])
			}
			if date.Time.Year() != tt.wantYear || int(date.Time.Month()) != tt.wantMonth || date.Time.Day() != tt.wantDay {
				t.Errorf("Got %d-%02d-%02d, want %d-%02d-%02d",
					date.Time.Year(), int(date.Time.Month()), date.Time.Day(),
					tt.wantYear, tt.wantMonth, tt.wantDay)
			}
		})
	}
}

// TestCalendarYearNotation tests CY2026, CY26 shorthand.
func TestCalendarYearNotation(t *testing.T) {
	clock := testClock
	tests := []struct {
		name      string
		input     string
		wantYear  int
		wantMonth int
		wantDay   int
	}{
		{"CY2026", "d = CY2026\n", 2026, 1, 1},
		{"CY2001", "d = CY2001\n", 2001, 1, 1},
		{"CY26 short", "d = CY26\n", 2026, 1, 1},
		{"CY01 short", "d = CY01\n", 2001, 1, 1},
		{"cy2026 lowercase", "d = cy2026\n", 2026, 1, 1},
		{"CY2026 + 6 months", "d = CY2026 + 6 months\n", 2026, 7, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interp := newTestInterpreterWithClock(clock)
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}
			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval(%q) error = %v", tt.input, err)
			}
			date, ok := results[0].(*types.Date)
			if !ok {
				t.Fatalf("Expected *types.Date, got %T (%v)", results[0], results[0])
			}
			if date.Time.Year() != tt.wantYear || int(date.Time.Month()) != tt.wantMonth || date.Time.Day() != tt.wantDay {
				t.Errorf("Got %d-%02d-%02d, want %d-%02d-%02d",
					date.Time.Year(), int(date.Time.Month()), date.Time.Day(),
					tt.wantYear, tt.wantMonth, tt.wantDay)
			}
		})
	}
}
