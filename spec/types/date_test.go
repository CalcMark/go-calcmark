package types

import (
	"testing"
	"time"
)

func TestNewDateFromTime(t *testing.T) {
	input := time.Date(2026, 4, 10, 14, 30, 45, 0, time.UTC)
	d := NewDateFromTime(input)

	if d.HasTime {
		t.Error("NewDateFromTime should set HasTime to false")
	}
	if d.Time.Hour() != 0 || d.Time.Minute() != 0 || d.Time.Second() != 0 {
		t.Errorf("NewDateFromTime should normalize to midnight, got %s", d.Time.Format("15:04:05"))
	}
	if d.Time.Year() != 2026 || d.Time.Month() != 4 || d.Time.Day() != 10 {
		t.Errorf("NewDateFromTime should preserve date, got %s", d.Time.Format("2006-01-02"))
	}
}

func TestNewDateTime(t *testing.T) {
	input := time.Date(2026, 4, 10, 14, 30, 45, 0, time.UTC)
	d := NewDateTime(input)

	if !d.HasTime {
		t.Error("NewDateTime should set HasTime to true")
	}
	if d.Time.Hour() != 14 || d.Time.Minute() != 30 || d.Time.Second() != 45 {
		t.Errorf("NewDateTime should preserve time, got %s", d.Time.Format("15:04:05"))
	}
}

func TestNewDateTimeAtMidnight(t *testing.T) {
	input := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	d := NewDateTime(input)

	if !d.HasTime {
		t.Error("NewDateTime at midnight should still set HasTime to true")
	}
}

func TestDaysBetweenMixed(t *testing.T) {
	d1 := NewDateFromTime(time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC))
	d2 := NewDateTime(time.Date(2026, 4, 5, 14, 30, 0, 0, time.UTC))

	days := d1.DaysBetween(d2)
	// d1 (Apr 10 midnight) - d2 (Apr 5 14:30) = ~4.4 days, truncates to 4
	if days != 4 {
		t.Errorf("DaysBetween mixed HasTime dates: got %d, want 4", days)
	}
}

func TestAddDaysPreservesHasTimeFalse(t *testing.T) {
	d := NewDateFromTime(time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC))
	result := d.AddDays(5)

	if result.HasTime {
		t.Error("AddDays should preserve HasTime=false")
	}
	if result.Time.Day() != 15 {
		t.Errorf("AddDays(5) from Apr 10: got day %d, want 15", result.Time.Day())
	}
}
