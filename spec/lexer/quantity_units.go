package lexer

import "strings"

// isQuantityUnit checks if a string is a known unit for quantities (mass, length, volume, currency)
func isQuantityUnit(unit string) bool {
	normalized := strings.ToLower(unit)

	// Length units
	lengthUnits := map[string]bool{
		"mm": true, "millimeter": true, "millimeters": true, "millimetre": true, "millimetres": true,
		"cm": true, "centimeter": true, "centimeters": true, "centimetre": true, "centimetres": true,
		"m": true, "meter": true, "meters": true, "metre": true, "metres": true,
		"km": true, "kilometer": true, "kilometers": true, "kilometre": true, "kilometres": true,
		"in": true, "inch": true, "inches": true,
		"ft": true, "foot": true, "feet": true,
		"yd": true, "yard": true, "yards": true,
		"mi": true, "mile": true, "miles": true,
		"nmi": true, "nautical mile": true, "nautical miles": true,
	}

	// Mass units
	massUnits := map[string]bool{
		"mg": true, "milligram": true, "milligrams": true,
		"g": true, "gram": true, "grams": true,
		"kg": true, "kilogram": true, "kilograms": true,
		"t": true, "tonne": true, "tonnes": true, "metric ton": true, "metric tons": true,
		"oz": true, "ounce": true, "ounces": true,
		"lb": true, "lbs": true, "pound": true, "pounds": true,
	}

	// Volume units
	volumeUnits := map[string]bool{
		"ml": true, "milliliter": true, "milliliters": true, "millilitre": true, "millilitres": true,
		"l": true, "liter": true, "liters": true, "litre": true, "litres": true,
		"tsp": true, "teaspoon": true, "teaspoons": true,
		"tbsp": true, "tablespoon": true, "tablespoons": true,
		"cup": true, "cups": true,
		"pt": true, "pint": true, "pints": true,
		"qt": true, "quart": true, "quarts": true,
		"gal": true, "gallon": true, "gallons": true,
	}

	// Currency codes (ISO 4217) - 3 uppercase letters
	if len(unit) == 3 {
		allUpper := true
		for _, r := range unit {
			if r < 'A' || r > 'Z' {
				allUpper = false
				break
			}
		}
		if allUpper {
			return true // Currency code
		}
	}

	// Currency symbols
	if unit == "$" || unit == "€" || unit == "£" || unit == "¥" {
		return true
	}

	return lengthUnits[normalized] || massUnits[normalized] || volumeUnits[normalized]
}
