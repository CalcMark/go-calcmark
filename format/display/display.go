// Package display provides human-readable formatting for CalcMark types.
//
// This package separates display concerns from the core type system:
//   - spec/types: Model layer - stores exact values, String() returns precise representation
//   - format/display: View layer - formats values for human consumption (100K instead of 100000)
//
// The Formatter type holds locale-specific configuration. Package-level functions
// (Format, FormatNumber, etc.) delegate to a default en-US formatter singleton.
//
// Usage:
//
//	import "github.com/CalcMark/go-calcmark/v2/format/display"
//
//	// Package-level (en-US default):
//	fmt.Println(display.Format(result))  // "100K users"
//
//	// Locale-aware:
//	f := display.NewFormatter(display.DefaultConfig())
//	fmt.Println(f.Format(result))
package display

import (
	"strings"

	"github.com/CalcMark/go-calcmark/v2/spec/types"
	"github.com/shopspring/decimal"
)

// defaultFormatter is the package-level singleton used by the free functions below.
// Constructed once at init time, not per-call.
var defaultFormatter Formatter

func init() {
	defaultFormatter = DefaultFormatter()
}

// Format returns a human-readable string representation of any CalcMark type.
// This is the main entry point for display formatting using en-US defaults.
func Format(t types.Type) string {
	return defaultFormatter.Format(t)
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
	return defaultFormatter.FormatNumber(value)
}

// FormatQuantity formats a quantity (value + unit) in human-readable form.
// For known units (length, mass, volume, data, etc.), it normalizes to the
// most appropriate unit scale (e.g., 1000000 GB → 976.56 TB).
// For unknown/arbitrary units, it uses K/M/B/T number suffixes.
// Napkin estimates (IsNapkin=true) are prefixed with tilde (~).
//
// Examples:
//
//	FormatQuantity(1000 m) → "1 km"
//	FormatQuantity(23400000 GB) → "22.31 PB"
//	FormatQuantity(100000 users) → "100K users"
//	FormatQuantity(400 GB, isNapkin=true) → "~400 GB"
func FormatQuantity(q *types.Quantity) string {
	return defaultFormatter.FormatQuantity(q)
}

// FormatRate formats a rate (quantity per time) in human-readable form.
// For known units, normalizes to appropriate scale (e.g., 1000000 bytes/s → 976.56 KB/s).
//
// Examples:
//
//	FormatRate(1000000 bytes/s) → "976.56 KB/s"
//	FormatRate(100000 users/day) → "100K users/day"
func FormatRate(r *types.Rate) string {
	return defaultFormatter.FormatRate(r)
}

// FormatCurrency formats a currency value in human-readable form.
// Preserves 2 decimal places for small values, uses suffixes for large values.
// Normalizes currency codes to symbols (USD -> $) and handles sign positioning.
//
// Examples:
//
//	FormatCurrency($1500000) → "$1.5M"
//	FormatCurrency($42.50) → "$42.50"
//	FormatCurrency($1500) → "$1,500.00"
//	FormatCurrency(USD100) → "$100.00"
//	FormatCurrency(CNY1000) → "CNY 1,000.00"
func FormatCurrency(c *types.Currency) string {
	return defaultFormatter.FormatCurrency(c)
}

// FormatDuration formats a duration in human-readable form.
//
// Examples:
//
//	FormatDuration(1 month) → "1 month"
//	FormatDuration(365 days) → "365 days"
func FormatDuration(d *types.Duration) string {
	return defaultFormatter.FormatDuration(d)
}

// FormatDate formats a date in locale-aware short form using en-US defaults.
//
// Examples:
//
//	FormatDate(Jan 12 2025) → "Sun, Jan 12, 2025"
func FormatDate(d *types.Date) string {
	return defaultFormatter.FormatDate(d)
}

// FormatPercentage formats a percentage in human-readable form.
// Applies magnitude-based rounding to the display value (e.g., 62.166…% → 62.2%).
//
// Examples:
//
//	FormatPercentage(0.32) → "32%"
//	FormatPercentage(0.6217) → "62.2%"
func FormatPercentage(p *types.Percentage) string {
	return defaultFormatter.FormatPercentage(p)
}

// getCurrencyDecimals returns the number of decimal places for a currency.
// Most currencies use 2 decimals, but some like JPY use 0.
func getCurrencyDecimals(code string) int {
	switch code {
	case "JPY", "KRW", "VND":
		return 0
	default:
		return 2
	}
}

// timeUnitAbbreviations maps time units to their short forms.
var timeUnitAbbreviations = map[string]string{
	"millisecond": "ms",
	"second":      "s",
	"minute":      "min",
	"hour":        "h",
	"day":         "day",
	"week":        "week",
	"month":       "month",
	"year":        "year",
}

// abbreviateTimeUnit returns the short form of a time unit.
func abbreviateTimeUnit(unit string) string {
	if abbrev, ok := timeUnitAbbreviations[unit]; ok {
		return abbrev
	}
	return unit
}

// addThousandSeparators inserts commas as thousand separators in a numeric string.
// Only operates on the integer part of a number.
//
// Examples:
//
//	addThousandSeparators("999") → "999"
//	addThousandSeparators("1000") → "1,000"
//	addThousandSeparators("1234567") → "1,234,567"
func addThousandSeparators(s string) string {
	n := len(s)
	if n <= 3 {
		return s
	}

	var result strings.Builder
	remainder := n % 3
	if remainder > 0 {
		result.WriteString(s[:remainder])
		if n > remainder {
			result.WriteString(",")
		}
	}
	for i := remainder; i < n; i += 3 {
		result.WriteString(s[i : i+3])
		if i+3 < n {
			result.WriteString(",")
		}
	}
	return result.String()
}
