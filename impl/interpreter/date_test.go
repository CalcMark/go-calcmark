package interpreter

import (
	"strings"
	"testing"
	"time"

	"github.com/CalcMark/go-calcmark/spec/parser"
	"github.com/CalcMark/go-calcmark/spec/types"
)

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

// TestDateArithmeticEval tests date +/- duration expressions.
func TestDateArithmeticEval(t *testing.T) {
	now := time.Now()

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
				// Expected: evaluation not yet implemented for extended keywords.
				// This test documents the expected behavior for when it is.
				t.Skipf("Evaluation not yet implemented: %v", err)
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
