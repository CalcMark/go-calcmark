package types

import (
	"fmt"
	"time"
)

// Time represents a time of day with optional timezone offset.
// Uses Go's time.Time internally.
type Time struct {
	Time time.Time
}

// NewTime creates a Time from hour, minute, second (optional), and UTC offset (optional).
// hour: 0-23 for 24-hour format, or 1-12 for 12-hour with isPM
// minute: 0-59
// second: 0-59 (use -1 if not specified)
// isPM: true for PM in 12-hour format, false for AM or 24-hour format
// utcOffsetMinutes: offset from UTC in minutes (0 for UTC/local time)
func NewTime(hour, minute, second int, isPM bool, utcOffsetMinutes int) (*Time, error) {
	// Convert 12-hour to 24-hour if needed
	if isPM && hour != 12 {
		hour += 12
	} else if !isPM && hour == 12 {
		hour = 0
	}

	// Validate ranges
	if hour < 0 || hour > 23 {
		return nil, fmt.Errorf("invalid hour: %d (must be 0-23)", hour)
	}
	if minute < 0 || minute > 59 {
		return nil, fmt.Errorf("invalid minute: %d (must be 0-59)", minute)
	}
	if second < -1 || second > 59 {
		return nil, fmt.Errorf("invalid second: %d (must be 0-59 or -1 for unspecified)", second)
	}

	if second == -1 {
		second = 0
	}

	// Create timezone location
	location := time.FixedZone("", utcOffsetMinutes*60)

	// Use a reference date (Jan 1, 2000) since we only care about time
	t := time.Date(2000, 1, 1, hour, minute, second, 0, location)

	return &Time{Time: t}, nil
}

// String returns a human-readable time representation.
func (t *Time) String() string {
	// Format based on whether it has timezone info
	if t.Time.Location() == time.UTC {
		return t.Time.Format("15:04:05")
	}
	return t.Time.Format("15:04:05 MST")
}
