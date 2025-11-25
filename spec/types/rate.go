package types

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// Rate represents a rate type: quantity per time period.
// Examples: 100 MB/s, $0.10/hour, 5 GB/day, 60 meters per second
//
// Rate syntax:
//   - Slash (no spaces): "100 MB/s", "5 GB/day"
//   - Word "per" (with spaces): "100 MB per second", "5 GB per day"
//
// Rates are first-class types that enable:
//   - Accumulation: rate * time = quantity
//   - Conversion: rate in different time units
//   - Arithmetic: adding/subtracting compatible rates
type Rate struct {
	// Amount is the numerator (quantity per time unit)
	// Can be any Quantity type (GB, requests, dollars, etc.)
	Amount *Quantity

	// PerUnit is the time unit denominator
	// Valid units: "second", "minute", "hour", "day", "week", "month", "year"
	PerUnit string
}

// NewRate creates a new Rate from a quantity and time unit.
func NewRate(amount *Quantity, perUnit string) *Rate {
	return &Rate{
		Amount:  amount,
		PerUnit: NormalizeTimeUnit(perUnit),
	}
}

// String returns the string representation of the rate.
// Examples: "100 MB/s", "$0.10/hour", "5 GB/day"
func (r *Rate) String() string {
	if r == nil || r.Amount == nil {
		return "0/s"
	}

	// Format: "amount/timeunit"
	timeAbbrev := abbreviateTimeUnit(r.PerUnit)
	return fmt.Sprintf("%s/%s", r.Amount.String(), timeAbbrev)
}

// IsCompatible checks if two rates can be added/subtracted.
// Rates are compatible if their amounts have compatible units and same time periods.
func (r *Rate) IsCompatible(other *Rate) bool {
	if r == nil || other == nil {
		return false
	}
	if r.PerUnit != other.PerUnit {
		return false
	}
	// Check if amount units are compatible (via quantity compatibility)
	return r.Amount.Unit == other.Amount.Unit
}

// Add adds two compatible rates.
func (r *Rate) Add(other *Rate) (*Rate, error) {
	if !r.IsCompatible(other) {
		return nil, fmt.Errorf("incompatible rates: %s and %s", r.String(), other.String())
	}

	newAmount := &Quantity{
		Value: r.Amount.Value.Add(other.Amount.Value),
		Unit:  r.Amount.Unit,
	}

	return &Rate{
		Amount:  newAmount,
		PerUnit: r.PerUnit,
	}, nil
}

// Subtract subtracts two compatible rates.
func (r *Rate) Subtract(other *Rate) (*Rate, error) {
	if !r.IsCompatible(other) {
		return nil, fmt.Errorf("incompatible rates: %s and %s", r.String(), other.String())
	}

	newAmount := &Quantity{
		Value: r.Amount.Value.Sub(other.Amount.Value),
		Unit:  r.Amount.Unit,
	}

	return &Rate{
		Amount:  newAmount,
		PerUnit: r.PerUnit,
	}, nil
}

// Multiply multiplies a rate by a scalar.
func (r *Rate) Multiply(scalar decimal.Decimal) *Rate {
	if r == nil || r.Amount == nil {
		return NewRate(&Quantity{Value: decimal.Zero, Unit: ""}, "second")
	}

	newAmount := &Quantity{
		Value: r.Amount.Value.Mul(scalar),
		Unit:  r.Amount.Unit,
	}

	return &Rate{
		Amount:  newAmount,
		PerUnit: r.PerUnit,
	}
}

// NormalizeTimeUnit converts various time unit formats to canonical form.
// Examples: "s" → "second", "seconds" → "second", "sec" → "second"
func NormalizeTimeUnit(unit string) string {
	lower := strings.ToLower(strings.TrimSpace(unit))

	// Map of aliases to canonical forms
	timeUnits := map[string]string{
		"s":       "second",
		"sec":     "second",
		"second":  "second",
		"seconds": "second",

		"m":       "minute",
		"min":     "minute",
		"minute":  "minute",
		"minutes": "minute",

		"h":     "hour",
		"hr":    "hour",
		"hour":  "hour",
		"hours": "hour",

		"d":    "day",
		"day":  "day",
		"days": "day",

		"w":     "week",
		"week":  "week",
		"weeks": "week",

		"month":  "month",
		"months": "month",
		"mo":     "month",

		"y":     "year",
		"yr":    "year",
		"year":  "year",
		"years": "year",
	}

	if canonical, ok := timeUnits[lower]; ok {
		return canonical
	}

	// Unknown unit, return as-is
	return lower
}

// abbreviateTimeUnit returns short form for display.
func abbreviateTimeUnit(unit string) string {
	abbrevs := map[string]string{
		"second": "s",
		"minute": "min",
		"hour":   "h",
		"day":    "day",
		"week":   "week",
		"month":  "month",
		"year":   "year",
	}

	if abbrev, ok := abbrevs[unit]; ok {
		return abbrev
	}
	return unit
}

// TimeUnitToSeconds returns the number of seconds in a time unit.
// Used for accumulation and conversion calculations.
func TimeUnitToSeconds(unit string) (decimal.Decimal, error) {
	normalized := NormalizeTimeUnit(unit)

	conversions := map[string]int64{
		"second": 1,
		"minute": 60,
		"hour":   3600,
		"day":    86400,
		"week":   604800,
		"month":  2592000,  // 30 days
		"year":   31536000, // 365 days
	}

	if seconds, ok := conversions[normalized]; ok {
		return decimal.NewFromInt(seconds), nil
	}

	return decimal.Zero, fmt.Errorf("unknown time unit: %s", unit)
}
