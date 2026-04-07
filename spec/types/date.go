package types

import (
	"fmt"
	"time"
)

// Date represents a calendar date, optionally with time-of-day precision.
// Uses Go's time.Time internally for proper date handling (leap years, etc.).
// When HasTime is false, the time component is midnight and should not be displayed.
// When HasTime is true, the time was explicitly set and should be displayed.
type Date struct {
	Time    time.Time
	HasTime bool
}

// NewDate creates a Date from year, month, and day.
// If year is 0, uses the current year.
func NewDate(year, month, day int) (*Date, error) {
	if year == 0 {
		year = time.Now().Year()
	}

	// Validate the date
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)

	// Check if the date is valid by verifying it matches what we asked for
	if t.Year() != year || int(t.Month()) != month || t.Day() != day {
		return nil, fmt.Errorf("invalid date: year=%d, month=%d, day=%d", year, month, day)
	}

	return &Date{Time: t}, nil
}

// NewDateFromTime creates a date-only Date from a time.Time value.
// The time component is normalized to midnight UTC and HasTime is false.
func NewDateFromTime(t time.Time) *Date {
	// Normalize to midnight UTC
	normalized := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	return &Date{Time: normalized, HasTime: false}
}

// NewDateTime creates a Date with full time precision from a time.Time value.
// The time component is preserved and HasTime is true.
func NewDateTime(t time.Time) *Date {
	return &Date{Time: t, HasTime: true}
}

// String returns a human-readable date representation with day of week.
// USER REQUIREMENT: Show actual computed date, not just "today"
// Example: "Friday, November 22, 2024" not "today"
func (d *Date) String() string {
	return d.Time.Format("Monday, January 2, 2006")
}

// ShortString returns a shorter date format
func (d *Date) ShortString() string {
	return d.Time.Format("Jan 2, 2006")
}

// Format formats the date using a Go time layout string
func (d *Date) Format(layout string) string {
	return d.Time.Format(layout)
}

// AddDays adds the given number of days to the date.
func (d *Date) AddDays(days int) *Date {
	newTime := d.Time.AddDate(0, 0, days)
	return &Date{Time: newTime}
}

// DaysBetween returns the number of days from other to this date (d - other).
func (d *Date) DaysBetween(other *Date) int {
	duration := d.Time.Sub(other.Time)
	return int(duration.Hours() / 24)
}
