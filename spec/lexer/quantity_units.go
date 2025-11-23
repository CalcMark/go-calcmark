package lexer

import "strings"

// isQuantityUnit checks if a string is a known unit for quantities (mass, length, volume, currency)
func isQuantityUnit(unit string) bool {
	normalized := strings.ToLower(unit)

	// Length units
	lengthUnits := map[string]bool{
		"m": true, "meter": true, "meters": true,
		"km": true, "kilometer": true, "kilometers": true,
		"cm": true, "centimeter": true, "centimeters": true,
		"mm": true, "millimeter": true, "millimeters": true,
		"ft": true, "foot": true, "feet": true,
		"yd": true, "yard": true, "yards": true,
		"in": true, "inch": true, "inches": true,
		"mi": true, "mile": true, "miles": true,
	}

	// Mass units
	massUnits := map[string]bool{
		"kg": true, "kilogram": true, "kilograms": true,
		"g": true, "gram": true, "grams": true,
		"lb": true, "lbs": true, "pound": true, "pounds": true,
		"oz": true, "ounce": true, "ounces": true,
	}

	// Volume units
	volumeUnits := map[string]bool{
		"l": true, "liter": true, "liters": true,
		"ml": true, "milliliter": true, "milliliters": true,
		"gal": true, "gallon": true, "gallons": true,
		"cup": true, "cups": true,
		"pt": true, "pint": true, "pints": true,
		"qt": true, "quart": true, "quarts": true,
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
