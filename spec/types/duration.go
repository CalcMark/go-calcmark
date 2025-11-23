package types

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// Duration represents a time duration with a specific unit.
// Supports: days, hours, minutes, seconds, weeks, months, years.
type Duration struct {
	Value decimal.Decimal
	Unit  string // "days", "hours", "minutes", "seconds", "weeks", "months", "years"
}

// Conversion factors to seconds (approximate for months/years)
var durationToSeconds = map[string]int64{
	"second":  1,
	"seconds": 1,
	"minute":  60,
	"minutes": 60,
	"hour":    3600,
	"hours":   3600,
	"day":     86400,
	"days":    86400,
	"week":    604800,
	"weeks":   604800,
	"month":   2592000,  // 30 days
	"months":  2592000,  // 30 days
	"year":    31536000, // 365 days
	"years":   31536000, // 365 days
}

// NewDuration creates a new Duration with the given value and unit.
func NewDuration(value decimal.Decimal, unit string) (*Duration, error) {
	// Validate unit
	if _, ok := durationToSeconds[unit]; !ok {
		return nil, fmt.Errorf("invalid duration unit: %s", unit)
	}

	return &Duration{
		Value: value,
		Unit:  unit,
	}, nil
}

// NewDurationFromString creates a Duration from string value and unit.
func NewDurationFromString(s string, unit string) (*Duration, error) {
	value, err := decimal.NewFromString(s)
	if err != nil {
		return nil, err
	}
	return NewDuration(value, unit)
}

// String returns the string representation.
func (d *Duration) String() string {
	return fmt.Sprintf("%s %s", d.Value.String(), d.Unit)
}

// ToSeconds converts the duration to seconds.
// Note: Uses approximate conversions for months (30 days) and years (365 days).
func (d *Duration) ToSeconds() decimal.Decimal {
	factor := durationToSeconds[d.Unit]
	return d.Value.Mul(decimal.NewFromInt(factor))
}

// Convert converts the duration to a different unit.
func (d *Duration) Convert(targetUnit string) (*Duration, error) {
	// Validate target unit
	if _, ok := durationToSeconds[targetUnit]; !ok {
		return nil, fmt.Errorf("invalid duration unit: %s", targetUnit)
	}

	// Convert to seconds, then to target unit
	seconds := d.ToSeconds()
	targetFactor := decimal.NewFromInt(durationToSeconds[targetUnit])
	newValue := seconds.Div(targetFactor)

	return &Duration{
		Value: newValue,
		Unit:  targetUnit,
	}, nil
}
