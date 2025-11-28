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

// DurationToSeconds provides conversion factors to seconds (approximate for months/years).
// Uses int64 for whole-second units. Milliseconds are handled separately in conversion functions.
var DurationToSeconds = map[string]int64{
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

// durationToSecondsDecimal provides decimal conversion factors including sub-second units.
var durationToSecondsDecimal = map[string]decimal.Decimal{
	"millisecond":  decimal.NewFromFloat(0.001),
	"milliseconds": decimal.NewFromFloat(0.001),
	"second":       decimal.NewFromInt(1),
	"seconds":      decimal.NewFromInt(1),
	"minute":       decimal.NewFromInt(60),
	"minutes":      decimal.NewFromInt(60),
	"hour":         decimal.NewFromInt(3600),
	"hours":        decimal.NewFromInt(3600),
	"day":          decimal.NewFromInt(86400),
	"days":         decimal.NewFromInt(86400),
	"week":         decimal.NewFromInt(604800),
	"weeks":        decimal.NewFromInt(604800),
	"month":        decimal.NewFromInt(2592000),
	"months":       decimal.NewFromInt(2592000),
	"year":         decimal.NewFromInt(31536000),
	"years":        decimal.NewFromInt(31536000),
}

// isValidDurationUnit checks if the unit is a valid duration unit.
func isValidDurationUnit(unit string) bool {
	_, ok := durationToSecondsDecimal[unit]
	return ok
}

// NewDuration creates a new Duration with the given value and unit.
func NewDuration(value decimal.Decimal, unit string) (*Duration, error) {
	// Validate unit using the decimal map (includes milliseconds)
	if !isValidDurationUnit(unit) {
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
	factor := durationToSecondsDecimal[d.Unit]
	return d.Value.Mul(factor)
}

// Convert converts the duration to a different unit.
func (d *Duration) Convert(targetUnit string) (*Duration, error) {
	// Validate target unit
	if !isValidDurationUnit(targetUnit) {
		return nil, fmt.Errorf("invalid duration unit: %s", targetUnit)
	}

	// Convert to seconds, then to target unit
	seconds := d.ToSeconds()
	targetFactor := durationToSecondsDecimal[targetUnit]
	newValue := seconds.Div(targetFactor)

	return &Duration{
		Value: newValue,
		Unit:  targetUnit,
	}, nil
}
