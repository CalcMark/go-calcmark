package display

import (
	"fmt"
	"math"
	"strings"

	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

// Formatter formats CalcMark types for human-readable display.
// It is a value type (~56 bytes) wrapping a DisplayConfig.
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

	normValue, normUnit := NormalizeForDisplay(q.Value, q.Unit)

	var result string
	if normUnit != q.Unit {
		result = f.formatNormalizedQuantity(normValue, normUnit)
	} else {
		result = f.formatWithSuffix(q.Value, q.Unit)
	}

	if q.IsNapkin {
		return "~" + result
	}
	return result
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
		numStr = c.Value.Abs().StringFixed(int32(decimals))
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
	return d.String()
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
		result = fmt.Sprintf("%.1f%s", scaled, suffix)
	} else {
		result = fmt.Sprintf("%.2f%s", scaled, suffix)
		result = strings.TrimRight(strings.TrimRight(result[:len(result)-1], "0"), ".") + suffix
	}

	if isNegative {
		return "-" + result
	}
	return result
}

// formatSmallNumber formats numbers < 1000 with appropriate precision.
func (f Formatter) formatSmallNumber(value decimal.Decimal) string {
	fv, _ := value.Float64()

	if fv == math.Floor(fv) {
		return fmt.Sprintf("%d", int(fv))
	}

	str := fmt.Sprintf("%.6f", fv)
	str = strings.TrimRight(strings.TrimRight(str, "0"), ".")
	return str
}

// formatCurrencyWithSeparators formats a value with thousand separators and fixed decimals.
func (f Formatter) formatCurrencyWithSeparators(value decimal.Decimal, decimals int) string {
	str := value.StringFixed(int32(decimals))
	parts := strings.Split(str, ".")
	intPart := addThousandSeparators(parts[0])
	if len(parts) > 1 && decimals > 0 {
		return intPart + "." + parts[1]
	}
	return intPart
}
