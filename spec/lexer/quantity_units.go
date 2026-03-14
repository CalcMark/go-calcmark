package lexer

import "strings"

// IsQuantityUnit checks if a string is a known unit for quantities (mass, length, volume, currency).
// Exported for use by document detector.
func IsQuantityUnit(unit string) bool {
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

	// Mass units (including troy, US-qualified, and ton variants)
	massUnits := map[string]bool{
		"mg": true, "milligram": true, "milligrams": true,
		"g": true, "gram": true, "grams": true,
		"kg": true, "kilogram": true, "kilograms": true,
		"t": true, "tonne": true, "tonnes": true, "metric ton": true, "metric tons": true,
		"oz": true, "ounce": true, "ounces": true, "us oz": true,
		"lb": true, "lbs": true, "pound": true, "pounds": true, "us lb": true,
		"troy oz": true, "troy ounce": true, "troy ounces": true,
		"troy lb": true, "troy pound": true, "troy pounds": true,
		"short ton": true, "short tons": true,
		"long ton": true, "long tons": true,
	}

	// Volume units (including imperial and US-qualified variants)
	volumeUnits := map[string]bool{
		"ml": true, "milliliter": true, "milliliters": true, "millilitre": true, "millilitres": true,
		"l": true, "liter": true, "liters": true, "litre": true, "litres": true,
		"tsp": true, "teaspoon": true, "teaspoons": true,
		"tbsp": true, "tablespoon": true, "tablespoons": true,
		"cup": true, "cups": true, "us cup": true,
		"fl oz": true, "fluid ounce": true, "fluid ounces": true, "us fl oz": true,
		"pt": true, "pint": true, "pints": true, "us pt": true,
		"qt": true, "quart": true, "quarts": true, "us qt": true,
		"gal": true, "gallon": true, "gallons": true, "us gal": true,
		"imp gal": true, "imperial gallon": true, "imperial gallons": true,
		"imp qt": true, "imperial quart": true, "imperial quarts": true,
		"imp pt": true, "imperial pint": true, "imperial pints": true,
		"imp cup": true, "imperial cup": true, "imperial cups": true,
		"imp fl oz": true, "imperial fluid ounce": true, "imperial fluid ounces": true,
	}

	// Temperature units
	temperatureUnits := map[string]bool{
		"c": true, "celsius": true, "°c": true, "degc": true,
		"f": true, "fahrenheit": true, "°f": true, "degf": true,
		"k": true, "kelvin": true,
	}

	// Speed units (include '/' variants)
	speedUnits := map[string]bool{
		"m/s": true, "mps": true, "meters per second": true,
		"km/h": true, "kph": true, "kmh": true, "kilometers per hour": true,
		"mph": true, "miles per hour": true,
		"knot": true, "knots": true,
	}

	// Energy units
	energyUnits := map[string]bool{
		"j": true, "joule": true, "joules": true,
		"kj": true, "kilojoule": true, "kilojoules": true,
		"cal": true, "calorie": true, "calories": true,
		"kcal": true, "kilocalorie": true, "kilocalories": true,
		"kwh": true, "kilowatt-hour": true, "kilowatt-hours": true,
	}

	// Power units
	powerUnits := map[string]bool{
		"w": true, "watt": true, "watts": true,
		"kw": true, "kilowatt": true, "kilowatts": true,
		"mw": true, "megawatt": true, "megawatts": true,
		"hp": true, "horsepower": true,
	}

	// Check all unit categories
	if lengthUnits[unit] || massUnits[unit] || volumeUnits[unit] ||
		temperatureUnits[unit] || speedUnits[unit] ||
		energyUnits[unit] || powerUnits[unit] {
		return true
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
