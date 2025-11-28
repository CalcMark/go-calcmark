package interpreter

import (
	"fmt"
	"math"
	"strings"

	"github.com/shopspring/decimal"
)

// formatNapkin formats a number in human-readable "napkin math" style.
// Rounds to specified precision (default 2 significant figures) and uses
// K/M/B/T suffixes for thousands/millions/billions/trillions.
//
// Parameters:
//   - value: The number to format
//   - precision: Number of significant figures (typically 2, but adaptable)
//
// Examples:
//   - formatNapkin(1234567, 2) → "~1.2M"
//   - formatNapkin(47, 2) → "~47"
//   - formatNapkin(8734, 2) → "~8700"
func formatNapkin(value decimal.Decimal, precision int) string {
	// Handle zero and very small numbers
	absValue := value.Abs()
	if absValue.LessThan(decimal.NewFromFloat(0.01)) {
		return "~0"
	}

	// Determine the scale (K, M, B, T)
	floatVal, _ := absValue.Float64()

	var scale string
	var divisor float64

	if floatVal >= 1e12 {
		scale = "T"
		divisor = 1e12
	} else if floatVal >= 1e9 {
		scale = "B"
		divisor = 1e9
	} else if floatVal >= 1e6 {
		scale = "M"
		divisor = 1e6
	} else if floatVal >= 1e3 {
		scale = "K"
		divisor = 1e3
	} else {
		// For numbers < 1000, round to precision significant figures
		rounded := roundToSignificantFigures(floatVal, precision)
		if value.IsNegative() {
			return fmt.Sprintf("~-%v", rounded)
		}
		return fmt.Sprintf("~%v", rounded)
	}

	// Scale the number
	scaled := floatVal / divisor

	// Round to precision significant figures
	rounded := roundToSignificantFigures(scaled, precision)

	// Format the result
	result := fmt.Sprintf("~%v%s", rounded, scale)
	if value.IsNegative() {
		result = "~-" + result[1:] // Add negative sign after ~
	}

	return result
}

// roundToSignificantFigures rounds a number to the specified number of significant figures.
// This function is kept separate and adaptable so precision can be easily changed.
//
// Examples:
//   - roundToSignificantFigures(1.234567, 2) → 1.2
//   - roundToSignificantFigures(8734, 2) → 8700
//   - roundToSignificantFigures(0.0123, 2) → 0.012
func roundToSignificantFigures(value float64, sigFigs int) any {
	if value == 0 {
		return 0
	}

	// Calculate the order of magnitude
	magnitude := math.Floor(math.Log10(math.Abs(value)))

	// Calculate the multiplier for the desired precision
	multiplier := math.Pow(10, float64(sigFigs-1)-magnitude)

	// Round the value
	rounded := math.Round(value*multiplier) / multiplier

	// Determine if we should return an int or float
	// If the rounded value is a whole number, return as int
	if rounded == math.Floor(rounded) && magnitude >= 0 {
		return int(rounded)
	}

	// Otherwise return as float, but clean up trailing zeros
	str := fmt.Sprintf("%.*f", int(math.Max(0, float64(sigFigs-1)-magnitude)), rounded)
	str = strings.TrimRight(strings.TrimRight(str, "0"), ".")

	// Try to parse back to see if it's actually an integer
	if !strings.Contains(str, ".") {
		return str
	}

	return str
}
