package types

import (
	"fmt"
	"time"
)

// Date represents a calendar date.
// Uses Go's time.Time internally for proper date handling (leap years, etc.).
type Date struct {
	Time time.Time
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

// NewDateFromTime creates a Date from a time.Time value.
func NewDateFromTime(t time.Time) *Date {
	// Normalize to midnight UTC
	normalized := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	return &Date{Time: normalized}
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

// DaysBetween returns the number of days between this date and another.
func (d *Date) DaysBetween(other *Date) int {
	duration := other.Time.Sub(d.Time)
	return int(duration.Hours() / 24)
}
