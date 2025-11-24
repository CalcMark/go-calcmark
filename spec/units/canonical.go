// Package units provides canonical unit name mappings based on UCUM and NIST standards.
// This ensures CalcMark supports both SI units and US customary units with standard
// abbreviations and full names.
package units

import "strings"

// Canonical unit names and abbreviations based on UCUM (Unified Code for Units of Measure)
// and NIST SP 811 / Handbook 44 standards.

// UnitMapping defines a canonical unit with its various representations.
type UnitMapping struct {
	Canonical   string   // Canonical name (e.g., "meter")
	Symbol      string   // Standard symbol (e.g., "m")
	Aliases     []string // Alternative names (e.g., "meters", "metre", "metres")
	System      string   // "SI", "US_Customary", "Imperial"
	Quantity    string   // Type: "Length", "Mass", "Volume", "Temperature", etc.
	Description string   // Human-readable description
}

// StandardUnits is the canonical mappingof all supported units.
// Keys are normalized (lowercase) for lookup, values contain all representations.
var StandardUnits = map[string]UnitMapping{
	// ========== LENGTH UNITS ==========

	// Length - SI Base
	"meter": {
		Canonical:   "meter",
		Symbol:      "m",
		Aliases:     []string{"meter", "meters", "metre", "metres", "m"},
		System:      "SI",
		Quantity:    "Length",
		Description: "SI base unit of length",
	},
	"millimeter": {
		Canonical:   "millimeter",
		Symbol:      "mm",
		Aliases:     []string{"millimeter", "millimeters", "millimetre", "millimetres", "mm"},
		System:      "SI",
		Quantity:    "Length",
		Description: "0.001 meters",
	},
	"centimeter": {
		Canonical:   "centimeter",
		Symbol:      "cm",
		Aliases:     []string{"centimeter", "centimeters", "centimetre", "centimetres", "cm"},
		System:      "SI",
		Quantity:    "Length",
		Description: "0.01 meters",
	},
	"kilometer": {
		Canonical:   "kilometer",
		Symbol:      "km",
		Aliases:     []string{"kilometer", "kilometers", "kilometre", "kilometres", "km"},
		System:      "SI",
		Quantity:    "Length",
		Description: "1000 meters",
	},

	// Length - US Customary / Imperial
	"inch": {
		Canonical:   "inch",
		Symbol:      "in",
		Aliases:     []string{"inch", "inches", "in"},
		System:      "US_Customary",
		Quantity:    "Length",
		Description: "1/12 foot, 0.0254 meters",
	},
	"foot": {
		Canonical:   "foot",
		Symbol:      "ft",
		Aliases:     []string{"foot", "feet", "ft"},
		System:      "US_Customary",
		Quantity:    "Length",
		Description: "12 inches, 0.3048 meters",
	},
	"yard": {
		Canonical:   "yard",
		Symbol:      "yd",
		Aliases:     []string{"yard", "yards", "yd"},
		System:      "US_Customary",
		Quantity:    "Length",
		Description: "3 feet, 0.9144 meters",
	},
	"mile": {
		Canonical:   "mile",
		Symbol:      "mi",
		Aliases:     []string{"mile", "miles", "mi"},
		System:      "US_Customary",
		Quantity:    "Length",
		Description: "5280 feet, 1609.344 meters",
	},
	"nautical mile": {
		Canonical:   "nautical mile",
		Symbol:      "nmi",
		Aliases:     []string{"nautical mile", "nautical miles", "nmi"},
		System:      "International",
		Quantity:    "Length",
		Description: "1852 meters (international standard)",
	},

	// ========== MASS UNITS ==========

	// Mass - SI
	"gram": {
		Canonical:   "gram",
		Symbol:      "g",
		Aliases:     []string{"gram", "grams", "g"},
		System:      "SI",
		Quantity:    "Mass",
		Description: "0.001 kilograms",
	},
	"milligram": {
		Canonical:   "milligram",
		Symbol:      "mg",
		Aliases:     []string{"milligram", "milligrams", "mg"},
		System:      "SI",
		Quantity:    "Mass",
		Description: "0.000001 kilograms",
	},
	"kilogram": {
		Canonical:   "kilogram",
		Symbol:      "kg",
		Aliases:     []string{"kilogram", "kilograms", "kg"},
		System:      "SI",
		Quantity:    "Mass",
		Description: "SI base unit of mass, 1000 grams",
	},
	"metric ton": {
		Canonical:   "metric ton",
		Symbol:      "t",
		Aliases:     []string{"metric ton", "metric tons", "tonne", "tonnes", "t"},
		System:      "SI",
		Quantity:    "Mass",
		Description: "1000 kilograms",
	},

	// Mass - US Customary
	"ounce": {
		Canonical:   "ounce",
		Symbol:      "oz",
		Aliases:     []string{"ounce", "ounces", "oz"},
		System:      "US_Customary",
		Quantity:    "Mass",
		Description: "Avoirdupois ounce, 1/16 pound, 28.349523125 grams",
	},
	"pound": {
		Canonical:   "pound",
		Symbol:      "lb",
		Aliases:     []string{"pound", "pounds", "lb", "lbs"},
		System:      "US_Customary",
		Quantity:    "Mass",
		Description: "Avoirdupois pound, 453.59237 grams",
	},

	// ========== VOLUME UNITS ==========

	// Volume - SI
	"milliliter": {
		Canonical:   "milliliter",
		Symbol:      "ml",
		Aliases:     []string{"milliliter", "milliliters", "millilitre", "millilitres", "ml"},
		System:      "SI",
		Quantity:    "Volume",
		Description: "0.001 liters, 1 cubic centimeter",
	},
	"liter": {
		Canonical:   "liter",
		Symbol:      "l",
		Aliases:     []string{"liter", "liters", "litre", "litres", "l"},
		System:      "SI",
		Quantity:    "Volume",
		Description: "SI base unit of volume, 1 cubic decimeter",
	},

	// Volume - US Customary
	"teaspoon": {
		Canonical:   "teaspoon",
		Symbol:      "tsp",
		Aliases:     []string{"teaspoon", "teaspoons", "tsp"},
		System:      "US_Customary",
		Quantity:    "Volume",
		Description: "1/3 tablespoon, approximately 4.929 milliliters",
	},
	"tablespoon": {
		Canonical:   "tablespoon",
		Symbol:      "tbsp",
		Aliases:     []string{"tablespoon", "tablespoons", "tbsp"},
		System:      "US_Customary",
		Quantity:    "Volume",
		Description: "3 teaspoons, approximately 14.787 milliliters",
	},
	"cup": {
		Canonical:   "cup",
		Symbol:      "cup",
		Aliases:     []string{"cup", "cups"},
		System:      "US_Customary",
		Quantity:    "Volume",
		Description: "US legal cup, 240 milliliters",
	},
	"pint": {
		Canonical:   "pint",
		Symbol:      "pt",
		Aliases:     []string{"pint", "pints", "pt"},
		System:      "US_Customary",
		Quantity:    "Volume",
		Description: "US liquid pint, 2 cups, 473.176 milliliters",
	},
	"quart": {
		Canonical:   "quart",
		Symbol:      "qt",
		Aliases:     []string{"quart", "quarts", "qt"},
		System:      "US_Customary",
		Quantity:    "Volume",
		Description: "US liquid quart, 2 pints, 946.353 milliliters",
	},
	"gallon": {
		Canonical:   "gallon",
		Symbol:      "gal",
		Aliases:     []string{"gallon", "gallons", "gal"},
		System:      "US_Customary",
		Quantity:    "Volume",
		Description: "US liquid gallon, 4 quarts, 3.785411784 liters",
	},
}

// NormalizeUnitName converts any unit alias to its canonical form.
// Returns the canonical name and true if found, empty string and false otherwise.
func NormalizeUnitName(input string) (canonical string, found bool) {
	// Normalize input (lowercase, trim spaces)
	normalized := strings.ToLower(strings.TrimSpace(input))

	// Direct lookup
	if unit, ok := StandardUnits[normalized]; ok {
		return unit.Canonical, true
	}

	// Check aliases
	for _, unit := range StandardUnits {
		for _, alias := range unit.Aliases {
			if strings.ToLower(alias) == normalized {
				return unit.Canonical, true
			}
		}
	}

	return "", false
}

// GetUnitSymbol returns the standard symbol for a unit name.
func GetUnitSymbol(unitName string) (string, bool) {
	canonical, found := NormalizeUnitName(unitName)
	if !found {
		return "", false
	}

	if unit, ok := StandardUnits[canonical]; ok {
		return unit.Symbol, true
	}

	return "", false
}
