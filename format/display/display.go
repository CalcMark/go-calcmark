// Package display provides human-readable formatting for CalcMark types.
//
// This package separates display concerns from the core type system:
//   - spec/types: Model layer - stores exact values, String() returns precise representation
//   - format/display: View layer - formats values for human consumption (100K instead of 100000)
//
// Usage:
//
//	import "github.com/CalcMark/go-calcmark/format/display"
//
//	result := interpreter.Eval(...)
//	fmt.Println(display.Format(result))  // "100K users" instead of "100000 users"
package display

import (
	"fmt"
	"math"
	"strings"

	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

// Format returns a human-readable string representation of any CalcMark type.
// This is the main entry point for display formatting.
func Format(t types.Type) string {
	if t == nil {
		return ""
	}

	switch v := t.(type) {
	case *types.Number:
		return FormatNumber(v.Value)
	case *types.Quantity:
		return FormatQuantity(v)
	case *types.Rate:
		return FormatRate(v)
	case *types.Currency:
		return FormatCurrency(v)
	case *types.Duration:
		return FormatDuration(v)
	case *types.Date:
		return v.String() // Dates are already human-readable
	case *types.Boolean:
		return v.String()
	case *types.Time:
		return v.String()
	default:
		return fmt.Sprintf("%v", t)
	}
}

// FormatNumber formats a decimal number in human-readable form.
// Uses K/M/B/T suffixes for large numbers, preserves small numbers as-is.
//
// Examples:
//
//	FormatNumber(100000) → "100K"
//	FormatNumber(1500000) → "1.5M"
//	FormatNumber(42) → "42"
//	FormatNumber(0.5) → "0.5"
func FormatNumber(value decimal.Decimal) string {
	return formatWithSuffix(value, "")
}

// FormatQuantity formats a quantity (value + unit) in human-readable form.
//
// Examples:
//
//	FormatQuantity(100000 users) → "100K users"
//	FormatQuantity(1500000 bytes) → "1.5M bytes"
func FormatQuantity(q *types.Quantity) string {
	if q == nil {
		return ""
	}
	return formatWithSuffix(q.Value, q.Unit)
}

// FormatRate formats a rate (quantity per time) in human-readable form.
//
// Examples:
//
//	FormatRate(100000 users/day) → "100K users/day"
//	FormatRate(1500000 bytes/s) → "1.5M bytes/s"
func FormatRate(r *types.Rate) string {
	if r == nil || r.Amount == nil {
		return "0/s"
	}
	numStr := formatWithSuffix(r.Amount.Value, r.Amount.Unit)
	timeAbbrev := abbreviateTimeUnit(r.PerUnit)
	return fmt.Sprintf("%s/%s", numStr, timeAbbrev)
}

// FormatCurrency formats a currency value in human-readable form.
// Preserves 2 decimal places for small values, uses suffixes for large values.
//
// Examples:
//
//	FormatCurrency($1500000) → "$1.5M"
//	FormatCurrency($42.50) → "$42.50"
func FormatCurrency(c *types.Currency) string {
	if c == nil {
		return ""
	}

	absValue, _ := c.Value.Abs().Float64()

	// For small values, use standard currency format
	if absValue < 10000 {
		return c.String() // Use existing precise format
	}

	// For large values, use suffix notation
	numStr := formatNumberWithSuffix(c.Value)
	return fmt.Sprintf("%s%s", c.Symbol, numStr)
}

// FormatDuration formats a duration in human-readable form.
//
// Examples:
//
//	FormatDuration(1 month) → "1 month"
//	FormatDuration(365 days) → "365 days"
func FormatDuration(d *types.Duration) string {
	if d == nil {
		return ""
	}
	// Durations are typically already human-readable
	return d.String()
}

// formatWithSuffix formats a number with optional unit suffix.
func formatWithSuffix(value decimal.Decimal, unit string) string {
	numStr := formatNumberWithSuffix(value)
	if unit == "" {
		return numStr
	}
	return fmt.Sprintf("%s %s", numStr, unit)
}

// formatNumberWithSuffix formats a number using K/M/B/T suffixes.
func formatNumberWithSuffix(value decimal.Decimal) string {
	absValue, _ := value.Abs().Float64()
	isNegative := value.IsNegative()

	// For small numbers, return as-is with reasonable precision
	if absValue < 1000 {
		return formatSmallNumber(value)
	}

	var suffix string
	var divisor float64

	switch {
	case absValue >= 1e12:
		suffix = "T"
		divisor = 1e12
	case absValue >= 1e9:
		suffix = "B"
		divisor = 1e9
	case absValue >= 1e6:
		suffix = "M"
		divisor = 1e6
	default:
		suffix = "K"
		divisor = 1e3
	}

	scaled := absValue / divisor

	// Format with appropriate precision
	var result string
	if scaled == math.Floor(scaled) {
		// Whole number
		result = fmt.Sprintf("%d%s", int(scaled), suffix)
	} else if scaled*10 == math.Floor(scaled*10) {
		// One decimal place
		result = fmt.Sprintf("%.1f%s", scaled, suffix)
	} else {
		// Two decimal places, trim trailing zeros
		result = fmt.Sprintf("%.2f%s", scaled, suffix)
		result = strings.TrimRight(strings.TrimRight(result[:len(result)-1], "0"), ".") + suffix
	}

	if isNegative {
		return "-" + result
	}
	return result
}

// formatSmallNumber formats numbers < 1000 with appropriate precision.
func formatSmallNumber(value decimal.Decimal) string {
	f, _ := value.Float64()

	// Integer values
	if f == math.Floor(f) {
		return fmt.Sprintf("%d", int(f))
	}

	// Decimal values - use up to 6 decimal places, trim trailing zeros
	str := fmt.Sprintf("%.6f", f)
	str = strings.TrimRight(strings.TrimRight(str, "0"), ".")
	return str
}

// abbreviateTimeUnit returns the short form of a time unit.
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
