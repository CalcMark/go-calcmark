package semantic

import (
	"fmt"

	"github.com/CalcMark/go-calcmark/spec/ast"
)

// QuantityType represents the type of a physical quantity
type QuantityType string

const (
	QuantityLength      QuantityType = "Length"
	QuantityMass        QuantityType = "Mass"
	QuantityTime        QuantityType = "Time"
	QuantityVolume      QuantityType = "Volume"
	QuantityTemperature QuantityType = "Temperature"
	QuantityEnergy      QuantityType = "Energy"
	QuantityUnknown     QuantityType = "Unknown"
)

// GetQuantityType returns the quantity type for a given unit
func GetQuantityType(unit string) QuantityType {
	// Length units
	lengthUnits := map[string]bool{
		"meter": true, "meters": true, "m": true,
		"foot": true, "feet": true, "ft": true,
		"mile": true, "miles": true, "mi": true,
		"kilometer": true, "kilometers": true, "km": true,
		"centimeter": true, "centimeters": true, "cm": true,
		"inch": true, "inches": true, "in": true,
		"yard": true, "yards": true, "yd": true,
	}

	// Mass units
	massUnits := map[string]bool{
		"kilogram": true, "kilograms": true, "kg": true,
		"pound": true, "pounds": true, "lb": true, "lbs": true,
		"gram": true, "grams": true, "g": true,
		"ounce": true, "ounces": true, "oz": true,
		"ton": true, "tons": true,
	}

	// Time units (handled separately by duration literals)
	timeUnits := map[string]bool{
		"second": true, "seconds": true, "s": true,
		"minute": true, "minutes": true, "min": true,
		"hour": true, "hours": true, "h": true, "hr": true,
		"day": true, "days": true,
		"week": true, "weeks": true,
		"month": true, "months": true,
		"year": true, "years": true,
	}

	// Volume units
	volumeUnits := map[string]bool{
		"liter": true, "liters": true, "l": true, "L": true,
		"milliliter": true, "milliliters": true, "ml": true, "mL": true,
		"gallon": true, "gallons": true, "gal": true,
		"quart": true, "quarts": true, "qt": true,
		"pint": true, "pints": true, "pt": true,
		"cup": true, "cups": true,
		"tablespoon": true, "tablespoons": true, "tbsp": true,
		"teaspoon": true, "teaspoons": true, "tsp": true,
	}

	if lengthUnits[unit] {
		return QuantityLength
	}
	if massUnits[unit] {
		return QuantityMass
	}
	if timeUnits[unit] {
		return QuantityTime
	}
	if volumeUnits[unit] {
		return QuantityVolume
	}

	return QuantityUnknown
}

// AreUnitsCompatible checks if two units are compatible for arithmetic
// USER REQUIREMENT: Used for "10 meters + 5 kg" incompatibility detection
func AreUnitsCompatible(unit1, unit2 string) bool {
	if unit1 == "" || unit2 == "" {
		return true // One is a pure number, allow it
	}

	type1 := GetQuantityType(unit1)
	type2 := GetQuantityType(unit2)

	// Same type or unknown types (user-defined units)
	return type1 == type2 || type1 == QuantityUnknown || type2 == QuantityUnknown
}

// checkUnitCompatibility validates unit compatibility in binary operations
// USER REQUIREMENT: "10 meters + 5 kg" must produce error
func (c *Checker) checkUnitCompatibility(left, right ast.Node) {
	leftUnit := getNodeUnit(left)
	rightUnit := getNodeUnit(right)

	if !AreUnitsCompatible(leftUnit, rightUnit) {
		leftType := GetQuantityType(leftUnit)
		rightType := GetQuantityType(rightUnit)

		c.addDiagnostic(Diagnostic{
			Severity: Error,
			Code:     DiagIncompatibleUnits,
			Message:  "incompatible units",
			Detailed: fmt.Sprintf(
				"Cannot add %s (%s) to %s (%s) - incompatible unit types",
				leftUnit, leftType, rightUnit, rightType),
		})
	}
}

// getNodeUnit extracts the unit from a node
func getNodeUnit(node ast.Node) string {
	switch n := node.(type) {
	case *ast.QuantityLiteral:
		return n.Unit
	case *ast.CurrencyLiteral:
		return n.Symbol // Treat currency as a unit
	default:
		return ""
	}
}
