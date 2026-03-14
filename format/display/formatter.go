package display

import (
	"fmt"
	"math"
	"strings"

	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

// maxSeparatorInputLen is the maximum string length before separator insertion.
// Prevents pathological values from causing excessive allocations.
const maxSeparatorInputLen = 1000

// Formatter formats CalcMark types for human-readable display.
// It is a value type wrapping a DisplayConfig.
// Use NewFormatter to create one, or DefaultFormatter() for en-US defaults.
type Formatter struct {
	cfg DisplayConfig
}

// NewFormatter creates a Formatter from a DisplayConfig.
func NewFormatter(cfg DisplayConfig) Formatter {
	return Formatter{cfg: cfg}
}

// DefaultFormatter returns a Formatter with en-US defaults.
func DefaultFormatter() Formatter {
	return NewFormatter(DefaultConfig())
}

// Config returns the underlying DisplayConfig.
func (f Formatter) Config() DisplayConfig {
	return f.cfg
}

// Format returns a human-readable string representation of any CalcMark type.
func (f Formatter) Format(t types.Type) string {
	if t == nil {
		return ""
	}

	switch v := t.(type) {
	case *types.Number:
		return f.FormatNumber(v.Value)
	case *types.Quantity:
		return f.FormatQuantity(v)
	case *types.Rate:
		return f.FormatRate(v)
	case *types.Currency:
		return f.FormatCurrency(v)
	case *types.Duration:
		return f.FormatDuration(v)
	case *types.Date:
		return v.String()
	case *types.Boolean:
		return v.String()
	case *types.Fraction:
		// Fractions format themselves — bypass decimal rounding
		return v.String()
	case *types.Percentage:
		return v.String()
	case *types.Time:
		return v.String()
	default:
		return fmt.Sprintf("%v", t)
	}
}

// FormatNumber formats a decimal number in human-readable form.
func (f Formatter) FormatNumber(value decimal.Decimal) string {
	return f.formatWithSuffix(value, "")
}

// FormatQuantity formats a quantity (value + unit) in human-readable form.
func (f Formatter) FormatQuantity(q *types.Quantity) string {
	if q == nil {
		return ""
	}

	// Explicit conversions (via `in`/`as`) skip auto-scaling to respect user intent.
	// Use scientific notation for extreme values to keep output readable.
	if q.IsExplicit {
		return f.formatExplicitQuantity(q)
	}

	normValue, normUnit := NormalizeForDisplay(q.Value, q.Unit)

	var result string
	if normUnit != q.Unit {
		result = f.formatNormalizedQuantity(normValue, normUnit)
	} else {
		// Round for human readability unless precise display requested
		displayValue := q.Value
		if !q.IsPrecise {
			displayValue = roundForDisplay(displayValue)
		}
		result = f.formatWithSuffix(displayValue, q.Unit)
	}

	if q.IsNapkin {
		return "~" + result
	}
	return result
}

// formatExplicitQuantity formats a quantity whose unit was explicitly chosen.
// Skips NormalizeForDisplay. Uses scientific notation for extreme values.
func (f Formatter) formatExplicitQuantity(q *types.Quantity) string {
	fv, _ := q.Value.Float64()

	// Guard against values beyond float64 range
	if math.IsInf(fv, 0) || math.IsNaN(fv) {
		s := q.Value.String()
		if q.Unit == "" {
			return s
		}
		return s + " " + q.Unit
	}

	absVal := math.Abs(fv)

	// Use scientific notation for extreme values (check original magnitude)
	if absVal != 0 && (absVal < 1e-4 || absVal >= 1e12) {
		s := f.localizeDecimal(fmt.Sprintf("%g", fv))
		if q.Unit == "" {
			return s
		}
		return s + " " + q.Unit
	}

	// Apply magnitude-based rounding unless full precision requested
	value := q.Value
	if !q.IsPrecise {
		value = roundForDisplay(value)
	}

	// Format without K/M/B/T suffixes — explicit conversions show the actual number
	numStr := f.formatExplicitNumber(value)
	if q.Unit == "" {
		return numStr
	}
	return numStr + " " + q.Unit
}

// FormatRate formats a rate (quantity per time) in human-readable form.
func (f Formatter) FormatRate(r *types.Rate) string {
	if r == nil || r.Amount == nil {
		return "0/s"
	}

	normValue, normUnit := NormalizeForDisplay(r.Amount.Value, r.Amount.Unit)
	timeAbbrev := abbreviateTimeUnit(r.PerUnit)

	if normUnit != r.Amount.Unit {
		numStr := f.formatNormalizedQuantity(normValue, normUnit)
		return fmt.Sprintf("%s/%s", numStr, timeAbbrev)
	}

	numStr := f.formatWithSuffix(r.Amount.Value, r.Amount.Unit)
	return fmt.Sprintf("%s/%s", numStr, timeAbbrev)
}

// FormatCurrency formats a currency value in human-readable form.
func (f Formatter) FormatCurrency(c *types.Currency) string {
	if c == nil {
		return ""
	}

	symbol := types.GetCurrencySymbol(c.Code)

	sep := ""
	if symbol == c.Code {
		sep = " "
	}

	absValue, _ := c.Value.Abs().Float64()
	isNegative := c.Value.IsNegative()

	decimals := getCurrencyDecimals(c.Code)

	var numStr string
	switch {
	case absValue >= 10000:
		numStr = f.formatNumberWithSuffix(c.Value.Abs())
	case absValue >= 1000:
		numStr = f.formatCurrencyWithSeparators(c.Value.Abs(), decimals)
	default:
		// Small values: use decimal.StringFixed() then replace decimal separator
		numStr = f.localizeDecimal(c.Value.Abs().StringFixed(int32(decimals)))
	}

	if isNegative {
		return "-" + symbol + sep + numStr
	}
	return symbol + sep + numStr
}

// FormatDuration formats a duration in human-readable form.
func (f Formatter) FormatDuration(d *types.Duration) string {
	if d == nil {
		return ""
	}
	rounded := roundForDisplay(d.Value)
	return fmt.Sprintf("%s %s", rounded, d.Unit)
}

// formatExplicitNumber formats a number with comma separators but without K/M/B/T suffixes.
// Used for explicit conversions where the user chose a specific unit.
func (f Formatter) formatExplicitNumber(value decimal.Decimal) string {
	fv, _ := value.Float64()

	if fv == math.Floor(fv) {
		// Integer: format with thousand separators
		intStr := fmt.Sprintf("%d", int64(fv))
		return f.insertGroupSeparators(intStr)
	}

	// Decimal: format with precision, trim trailing zeros, add group separators to integer part
	str := fmt.Sprintf("%.6f", fv)
	str = strings.TrimRight(strings.TrimRight(str, "0"), ".")

	parts := strings.Split(str, ".")
	intPart := f.insertGroupSeparators(parts[0])
	if len(parts) == 2 {
		return f.localizeDecimal(intPart + "." + parts[1])
	}
	return intPart
}

// formatWithSuffix formats a number with optional unit suffix using K/M/B/T.
func (f Formatter) formatWithSuffix(value decimal.Decimal, unit string) string {
	numStr := f.formatNumberWithSuffix(value)
	if unit == "" {
		return numStr
	}
	return fmt.Sprintf("%s %s", numStr, unit)
}

// formatNormalizedQuantity formats a value+unit that has already been normalized.
func (f Formatter) formatNormalizedQuantity(value decimal.Decimal, unit string) string {
	numStr := f.formatSmallNumber(value)
	return fmt.Sprintf("%s %s", numStr, unit)
}

// formatNumberWithSuffix formats a number using K/M/B/T suffixes.
// K/M/B/T letters are always English; the decimal separator within suffix
// numbers localizes (e.g., "1,5M" in de-DE).
func (f Formatter) formatNumberWithSuffix(value decimal.Decimal) string {
	absValue, _ := value.Abs().Float64()
	isNegative := value.IsNegative()

	if absValue < 1000 {
		return f.formatSmallNumber(value)
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

	var result string
	if scaled == math.Floor(scaled) {
		result = fmt.Sprintf("%d%s", int(scaled), suffix)
	} else if scaled*10 == math.Floor(scaled*10) {
		// One decimal place — localize the decimal separator
		numPart := fmt.Sprintf("%.1f", scaled)
		numPart = f.localizeDecimal(numPart)
		result = numPart + suffix
	} else {
		// Two decimal places, trim trailing zeros — localize the decimal separator
		numPart := fmt.Sprintf("%.2f", scaled)
		numPart = strings.TrimRight(strings.TrimRight(numPart, "0"), ".")
		numPart = f.localizeDecimal(numPart)
		result = numPart + suffix
	}

	if isNegative {
		return "-" + result
	}
	return result
}

// formatSmallNumber formats numbers < 1000 with appropriate precision.
// Uses locale-aware decimal separator.
func (f Formatter) formatSmallNumber(value decimal.Decimal) string {
	fv, _ := value.Float64()

	if fv == math.Floor(fv) {
		return fmt.Sprintf("%d", int(fv))
	}

	// Format with up to 6 decimal places, trim trailing zeros
	str := fmt.Sprintf("%.6f", fv)
	str = strings.TrimRight(strings.TrimRight(str, "0"), ".")

	return f.localizeDecimal(str)
}

// formatCurrencyWithSeparators formats a value with thousand separators and fixed decimals.
// Uses locale-aware thousand and decimal separators.
func (f Formatter) formatCurrencyWithSeparators(value decimal.Decimal, decimals int) string {
	str := value.StringFixed(int32(decimals))
	parts := strings.Split(str, ".")
	intPart := f.insertGroupSeparators(parts[0])
	if len(parts) > 1 && decimals > 0 {
		return intPart + f.cfg.DecimalSep + parts[1]
	}
	return intPart
}

// insertGroupSeparators inserts locale-specific thousand separators into a digit string.
// Skips insertion for pathological strings exceeding maxSeparatorInputLen.
func (f Formatter) insertGroupSeparators(digits string) string {
	n := len(digits)
	if n <= 3 || n > maxSeparatorInputLen {
		return digits
	}

	var b strings.Builder
	b.Grow(n + (n/3)*len(f.cfg.ThousandSep))
	remainder := n % 3
	if remainder > 0 {
		b.WriteString(digits[:remainder])
		if n > remainder {
			b.WriteString(f.cfg.ThousandSep)
		}
	}
	for i := remainder; i < n; i += 3 {
		b.WriteString(digits[i : i+3])
		if i+3 < n {
			b.WriteString(f.cfg.ThousandSep)
		}
	}
	return b.String()
}

// localizeDecimal replaces "." with the locale's decimal separator.
// This is a simple string replacement suitable for numbers already formatted
// by fmt.Sprintf or decimal.StringFixed (which always use ".").
func (f Formatter) localizeDecimal(s string) string {
	if f.cfg.DecimalSep == "." {
		return s
	}
	return strings.Replace(s, ".", f.cfg.DecimalSep, 1)
}
