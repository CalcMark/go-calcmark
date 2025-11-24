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
		"millimeter": true, "millimeters": true, "millimetre": true, "millimetres": true, "mm": true,
		"centimeter": true, "centimeters": true, "centimetre": true, "centimetres": true, "cm": true,
		"meter": true, "meters": true, "metre": true, "metres": true, "m": true,
		"kilometer": true, "kilometers": true, "kilometre": true, "kilometres": true, "km": true,
		"inch": true, "inches": true, "in": true,
		"foot": true, "feet": true, "ft": true,
		"yard": true, "yards": true, "yd": true,
		"mile": true, "miles": true, "mi": true,
		"nautical mile": true, "nautical miles": true, "nmi": true,
	}

	// Mass units
	massUnits := map[string]bool{
		"milligram": true, "milligrams": true, "mg": true,
		"gram": true, "grams": true, "g": true,
		"kilogram": true, "kilograms": true, "kg": true,
		"metric ton": true, "metric tons": true, "tonne": true, "tonnes": true, "t": true,
		"ounce": true, "ounces": true, "oz": true,
		"pound": true, "pounds": true, "lb": true, "lbs": true,
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
		"milliliter": true, "milliliters": true, "millilitre": true, "millilitres": true, "ml": true, "mL": true,
		"liter": true, "liters": true, "litre": true, "litres": true, "l": true, "L": true,
		"teaspoon": true, "teaspoons": true, "tsp": true,
		"tablespoon": true, "tablespoons": true, "tbsp": true,
		"cup": true, "cups": true,
		"pint": true, "pints": true, "pt": true,
		"quart": true, "quarts": true, "qt": true,
		"gallon": true, "gallons": true, "gal": true,
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
