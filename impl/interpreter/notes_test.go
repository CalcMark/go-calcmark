package interpreter

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/parser"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
	"github.com/shopspring/decimal"
)

// notes_test.go — informational notes attached to successful results.
// A note tells the user what the evaluator assumed on their behalf
// (a duration converted into the rate's time unit, a year counted as
// 52.14 weeks) so a surprising number is explained rather than hidden.

func evalNotes(t *testing.T, src string) []Note {
	t.Helper()
	interp := NewInterpreter()
	nodes, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if _, err := interp.Eval(nodes); err != nil {
		t.Fatalf("eval: %v", err)
	}
	return interp.TakeNotes()
}

func TestNotes_RateTimesConvertedDuration(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{"days into hours", "rate = $20/hour\nx = rate * 3 days\n", "3 days = 72 hours (rate is per hour)"},
		{"commuted", "rate = $20/hour\nx = 3 days * rate\n", "3 days = 72 hours (rate is per hour)"},
		{"weeks into hours", "rate = $100/hour\nx = rate * 1 week\n", "1 week = 168 hours (rate is per hour)"},
		{"minutes into hours", "rate = $60/hour\nx = rate * 90 minutes\n", "90 minutes = 1.5 hours (rate is per hour)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notes := evalNotes(t, tt.src)
			if len(notes) != 1 {
				t.Fatalf("want 1 note, got %d: %v", len(notes), notes)
			}
			if notes[0].Code != NoteUnitConversion {
				t.Errorf("want code %q, got %q", NoteUnitConversion, notes[0].Code)
			}
			if !strings.HasPrefix(notes[0].Message, tt.want) {
				t.Errorf("want message starting %q, got %q", tt.want, notes[0].Message)
			}
		})
	}
}

func TestNotes_HourlyRateTimesWholeDaysSuggestsWorkingHours(t *testing.T) {
	// The bug-report shape: an hourly wage times whole days almost
	// always means working hours, not 24-hour days. Say so.
	notes := evalNotes(t, "rate = $20/hour\ndpw = 3 days\nweekly = 8 * rate * dpw\n")
	if len(notes) != 1 {
		t.Fatalf("want 1 note, got %d: %v", len(notes), notes)
	}
	if !strings.Contains(notes[0].Message, "hours/day") {
		t.Errorf("want a hours/day suggestion, got %q", notes[0].Message)
	}
}

func TestNotes_NoneWhenUnitsAlreadyMatch(t *testing.T) {
	for _, src := range []string{
		"rate = $20/hour\nx = rate * 2 hours\n",
		"rate = $20/hour\nx = rate * 8\n",
		"rate = $20/hour\nx = rate * 8 hours/day\n",
		"x = 2 * 3 days\n",
		"w = $480/week\nx = w over 2 weeks\n",
	} {
		if notes := evalNotes(t, src); len(notes) != 0 {
			t.Errorf("%q: want no notes, got %v", src, notes)
		}
	}
}

func TestNotes_AccumulateOverConvertedPeriod(t *testing.T) {
	notes := evalNotes(t, "w = $480/week\nx = w over 1 year\n")
	if len(notes) != 1 {
		t.Fatalf("want 1 note, got %d: %v", len(notes), notes)
	}
	if notes[0].Message != "1 year = 52.14 weeks (rate is per week)" {
		t.Errorf("got %q", notes[0].Message)
	}
}

func TestNotes_TakeNotesClearsAndEvalResets(t *testing.T) {
	interp := NewInterpreter()
	nodes, err := parser.Parse("rate = $20/hour\nx = rate * 3 days\n")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := interp.Eval(nodes); err != nil {
		t.Fatal(err)
	}
	if got := interp.TakeNotes(); len(got) != 1 {
		t.Fatalf("want 1 note, got %d", len(got))
	}
	if got := interp.TakeNotes(); len(got) != 0 {
		t.Errorf("second take should be empty, got %v", got)
	}
	// A fresh Eval starts from a clean slate even if notes were never taken.
	if _, err := interp.Eval(nodes); err != nil {
		t.Fatal(err)
	}
	if _, err := interp.Eval(nodes[1:]); err != nil {
		t.Fatal(err)
	}
	if got := interp.TakeNotes(); len(got) != 1 {
		t.Errorf("want 1 note from the last Eval only, got %d", len(got))
	}
}

func TestFormatTimeSpan(t *testing.T) {
	tests := []struct {
		value string
		unit  string
		want  string
	}{
		{"1", "year", "1 year"},
		{"3", "day", "3 days"},
		{"72", "hour", "72 hours"},
		{"52.142857", "week", "52.14 weeks"},
		{"1.5", "hour", "1.5 hours"},
		{"0.5", "day", "0.5 days"},
	}
	for _, tt := range tests {
		got := formatTimeSpan(decimal.RequireFromString(tt.value), tt.unit)
		if got != tt.want {
			t.Errorf("formatTimeSpan(%s, %s) = %q, want %q", tt.value, tt.unit, got, tt.want)
		}
	}
}

func TestConversionNote_IgnoresNonTimeRates(t *testing.T) {
	rate := types.NewRate(&types.Quantity{Value: decimal.NewFromInt(100), Unit: "cakes"}, "box")
	qty := &types.Quantity{Value: decimal.NewFromInt(5), Unit: "box"}
	if _, ok := conversionNote(rate, qty, "*"); ok {
		t.Error("custom-unit rates have nothing to convert; want no note")
	}
}
