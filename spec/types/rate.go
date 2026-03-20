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

// CompoundUnit returns the compound unit string for this rate (e.g., "MB/s", "USD/h").
// The numerator uses the amount's unit and the denominator uses the abbreviated time unit.
func (r *Rate) CompoundUnit() string {
	if r == nil || r.Amount == nil {
		return ""
	}
	return r.Amount.Unit + "/" + abbreviateTimeUnit(r.PerUnit)
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

// timeUnitAliases maps all time unit variants to their canonical form.
// This is a package-level var to avoid per-call allocation.
var timeUnitAliases = map[string]string{
	"ns":          "nanosecond",
	"nanosecond":  "nanosecond",
	"nanoseconds": "nanosecond",

	"μs":           "microsecond",
	"us":           "microsecond",
	"microsecond":  "microsecond",
	"microseconds": "microsecond",

	"ms":           "millisecond",
	"millisecond":  "millisecond",
	"milliseconds": "millisecond",

	"s":       "second",
	"sec":     "second",
	"second":  "second",
	"seconds": "second",

	"m":       "minute",
	"min":     "minute",
	"mins":    "minute",
	"minute":  "minute",
	"minutes": "minute",

	"h":     "hour",
	"hr":    "hour",
	"hour":  "hour",
	"hours": "hour",

	"d":     "day",
	"day":   "day",
	"days":  "day",
	"daily": "day",

	"w":      "week",
	"wk":     "week",
	"week":   "week",
	"weeks":  "week",
	"weekly": "week",

	"month":   "month",
	"months":  "month",
	"mo":      "month",
	"monthly": "month",

	"quarter":   "quarter",
	"quarters":  "quarter",
	"quarterly": "quarter",

	"y":      "year",
	"yr":     "year",
	"year":   "year",
	"years":  "year",
	"yearly": "year",
}

// periodsPerYear maps canonical time units to periods per year.
var periodsPerYear = map[string]int{
	"year":    1,
	"quarter": 4,
	"month":   12,
	"week":    52,
	"day":     365,
}

// timeUnitAbbrevs maps canonical time units to short display forms.
var timeUnitAbbrevs = map[string]string{
	"nanosecond":  "ns",
	"microsecond": "μs",
	"millisecond": "ms",
	"second":      "s",
	"minute":      "min",
	"hour":        "h",
	"day":         "day",
	"week":        "week",
	"month":       "month",
	"year":        "year",
}

// NormalizeTimeUnit converts various time unit formats to canonical form.
// Handles abbreviations, plurals, and adjectival forms (e.g., "monthly" → "month").
// Examples: "s" → "second", "seconds" → "second", "sec" → "second", "daily" → "day"
func NormalizeTimeUnit(unit string) string {
	lower := strings.ToLower(strings.TrimSpace(unit))

	if canonical, ok := timeUnitAliases[lower]; ok {
		return canonical
	}

	// Unknown unit, return as-is
	return lower
}

// PeriodToPeriodsPerYear returns how many of the given period fit in one year.
// Used by growth functions to convert between annual rates and per-period rates.
// Returns 0 and false for unrecognized periods.
func PeriodToPeriodsPerYear(period string) (int, bool) {
	normalized := NormalizeTimeUnit(period)
	n, ok := periodsPerYear[normalized]
	return n, ok
}

// abbreviateTimeUnit returns short form for display.
func abbreviateTimeUnit(unit string) string {
	if abbrev, ok := timeUnitAbbrevs[unit]; ok {
		return abbrev
	}
	return unit
}

// TimeUnitToSeconds returns the number of seconds in a time unit.
// Used for accumulation and conversion calculations.
// Uses the shared DurationToSeconds map from duration.go
func TimeUnitToSeconds(unit string) (decimal.Decimal, error) {
	normalized := NormalizeTimeUnit(unit)

	// Use the decimal map which includes sub-second units (ns, μs, ms).
	if seconds, ok := durationToSecondsDecimal[normalized]; ok {
		return seconds, nil
	}

	return decimal.Zero, fmt.Errorf("unknown time unit: %s", unit)
}

// IsTimeUnit returns true if the given unit is a valid time unit.
func IsTimeUnit(unit string) bool {
	_, err := TimeUnitToSeconds(unit)
	return err == nil
}
