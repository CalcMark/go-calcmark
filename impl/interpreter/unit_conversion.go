package interpreter

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/martinlindhe/unit"
	"github.com/shopspring/decimal"
)

// evalQuantityOperation handles quantity + quantity with unit conversion
// USER REQUIREMENT: First-unit-wins rule
func evalQuantityOperation(left, right *types.Quantity, operator string) (types.Type, error) {
	if operator != "+" && operator != "-" {
		return nil, fmt.Errorf("unsupported quantity operation: %s", operator)
	}

	// First-unit-wins: convert right to left's unit
	rightConverted, err := convertQuantity(right, left.Unit)
	if err != nil {
		return nil, fmt.Errorf("cannot %s incompatible units %s and %s: %w",
			operator, left.Unit, right.Unit, err)
	}

	var result decimal.Decimal
	switch operator {
	case "+":
		result = left.Value.Add(rightConverted.Value)
	case "-":
		result = left.Value.Sub(rightConverted.Value)
	}

	// Result is in left's unit (first-unit-wins)
	return &types.Quantity{Value: result, Unit: left.Unit}, nil
}

// convertQuantity converts a quantity to the target unit using martinlindhe/unit
func convertQuantity(qty *types.Quantity, targetUnit string) (*types.Quantity, error) {
	if qty.Unit == targetUnit {
		return qty, nil // No conversion needed
	}

	// Try martinlindhe/unit conversion for known units
	fromValue, fromUnit, err := mapToMartinlindheUnit(qty.Value, qty.Unit)
	if err != nil {
		// Not a known unit - treat as arbitrary unit
		// Arbitrary units cannot be converted
		return nil, fmt.Errorf("cannot convert %s to %s (incompatible units)", qty.Unit, targetUnit)
	}

	toUnit, err := mapUnitName(targetUnit)
	if err != nil {
		// Target not a known unit either
		return nil, fmt.Errorf("cannot convert %s to %s (incompatible units)", qty.Unit, targetUnit)
	}

	// Perform conversion
	converted, err := convertValue(fromValue, fromUnit, toUnit)
	if err != nil {
		return nil, err
	}

	return &types.Quantity{
		Value: decimal.NewFromFloat(converted),
		Unit:  targetUnit,
	}, nil
}

// mapToMartinlindheUnit converts a value in the given unit to the base unit
// Returns the value in base units (meters, kilograms, liters)
func mapToMartinlindheUnit(value decimal.Decimal, unitName string) (float64, string, error) {
	f, _ := value.Float64()
	normalized := strings.ToLower(unitName)

	// Length units - convert to METERS
	switch normalized {
	case "meter", "meters", "m":
		return f, "length", nil
	case "foot", "feet", "ft":
		return f * 0.3048, "length", nil // feet to meters
	case "kilometer", "kilometers", "km":
		return f * 1000, "length", nil // km to meters
	case "mile", "miles", "mi":
		return f * 1609.344, "length", nil // miles to meters
	case "inch", "inches", "in":
		return f * 0.0254, "length", nil // inches to meters
	}

	// Mass units - convert to KILOGRAMS
	switch normalized {
	case "kilogram", "kilograms", "kg":
		return f, "mass", nil
	case "pound", "pounds", "lb", "lbs":
		return f * 0.45359237, "mass", nil // pounds to kg
	case "gram", "grams", "g":
		return f * 0.001, "mass", nil // grams to kg
	case "ounce", "ounces", "oz":
		return f * 0.028349523125, "mass", nil // ounces to kg
	}

	// Volume units - convert to LITERS
	switch normalized {
	case "liter", "liters", "l":
		return f, "volume", nil
	case "gallon", "gallons", "gal":
		return f * 3.785411784, "volume", nil // US gallons to liters
	case "milliliter", "milliliters", "ml":
		return f * 0.001, "volume", nil // ml to liters
	case "cup", "cups":
		return f * 0.24, "volume", nil // US legal cups to liters
	case "pint", "pints", "pt":
		return f * 0.473176, "volume", nil // US pints to liters
	case "quart", "quarts", "qt":
		return f * 0.946353, "volume", nil // US quarts to liters
	}

	return 0, "", fmt.Errorf("unsupported unit: %s", unitName)
}

// convertValue performs the actual unit conversion
func convertValue(value float64, fromType, toUnitName string) (float64, error) {
	normalized := strings.ToLower(toUnitName)

	switch fromType {
	case "length":
		// value is in METERS (base unit for length in martinlindhe/unit)
		length := unit.Length(value) * unit.Meter
		switch normalized {
		case "meter", "meters", "m":
			return float64(length.Meters()), nil
		case "foot", "feet", "ft":
			return float64(length.Feet()), nil
		case "kilometer", "kilometers", "km":
			return float64(length.Kilometers()), nil
		case "mile", "miles", "mi":
			return float64(length.Miles()), nil
		case "inch", "inches", "in":
			return float64(length.Inches()), nil
		}

	case "mass":
		// value is in KILOGRAMS (base unit for mass)
		mass := unit.Mass(value) * unit.Kilogram
		switch normalized {
		case "kilogram", "kilograms", "kg":
			return float64(mass.Kilograms()), nil
		case "pound", "pounds", "lb", "lbs":
			return float64(mass.AvoirdupoisPounds()), nil
		case "gram", "grams", "g":
			return float64(mass.Grams()), nil
		case "ounce", "ounces", "oz":
			return float64(mass.AvoirdupoisOunces()), nil
		}

	case "volume":
		// value is in LITERS (base unit for volume)
		vol := unit.Volume(value) * unit.Liter
		switch normalized {
		case "liter", "liters", "l":
			return float64(vol.Liters()), nil
		case "gallon", "gallons", "gal":
			return float64(vol.USLiquidGallons()), nil
		case "milliliter", "milliliters", "ml":
			return float64(vol.Milliliters()), nil
		case "cup", "cups":
			return float64(vol.USLegalCups()), nil
		}
	}

	return 0, fmt.Errorf("cannot convert to %s", toUnitName)
}

// mapUnitName normalizes unit names for comparison
func mapUnitName(unitName string) (string, error) {
	normalized := strings.ToLower(unitName)

	validUnits := map[string]bool{
		"meter": true, "meters": true, "m": true,
		"foot": true, "feet": true, "ft": true,
		"kilometer": true, "kilometers": true, "km": true,
		"mile": true, "miles": true, "mi": true,
		"inch": true, "inches": true, "in": true,
		"kilogram": true, "kilograms": true, "kg": true,
		"pound": true, "pounds": true, "lb": true, "lbs": true,
		"gram": true, "grams": true, "g": true,
		"ounce": true, "ounces": true, "oz": true,
		"liter": true, "liters": true, "l": true,
		"gallon": true, "gallons": true, "gal": true,
		"milliliter": true, "milliliters": true, "ml": true,
		"cup": true, "cups": true,
	}

	if !validUnits[normalized] {
		return "", fmt.Errorf("unknown unit: %s", unitName)
	}

	return normalized, nil
}
