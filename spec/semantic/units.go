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
	QuantitySpeed       QuantityType = "Speed"
	QuantityEnergy      QuantityType = "Energy"
	QuantityPower       QuantityType = "Power"
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

	// Temperature units
	temperatureUnits := map[string]bool{
		"celsius": true, "c": true, "°c": true, "degc": true,
		"fahrenheit": true, "f": true, "°f": true, "degf": true,
		"kelvin": true, "k": true,
	}

	// Speed units
	speedUnits := map[string]bool{
		"m/s": true, "mps": true, "meters per second": true,
		"km/h": true, "kph": true, "kmh": true, "kilometers per hour": true,
		"mph": true, "miles per hour": true,
		"knot": true, "knots": true,
	}

	// Energy units
	energyUnits := map[string]bool{
		"joule": true, "joules": true, "j": true,
		"kilojoule": true, "kilojoules": true, "kj": true,
		"calorie": true, "calories": true, "cal": true,
		"kilocalorie": true, "kilocalories": true, "kcal": true,
		"kwh": true, "kilowatt-hour": true, "kilowatt-hours": true,
	}

	// Power units
	powerUnits := map[string]bool{
		"watt": true, "watts": true, "w": true,
		"kilowatt": true, "kilowatts": true, "kw": true,
		"megawatt": true, "megawatts": true, "mw": true,
		"horsepower": true, "hp": true,
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
	if temperatureUnits[unit] {
		return QuantityTemperature
	}
	if speedUnits[unit] {
		return QuantitySpeed
	}
	if energyUnits[unit] {
		return QuantityEnergy
	}
	if powerUnits[unit] {
		return QuantityPower
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
