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

// StandardUnits is the canonical mapping of all supported units.
// Keys are normalized (lowercase) for lookup, values contain all representations.
var StandardUnits = map[string]UnitMapping{
	// Length - SI
	"meter": {
		Canonical:   "meter",
		Symbol:      "m",
		Aliases:     []string{"meter", "meters", "metre", "metres", "m"},
		System:      "SI",
		Quantity:    "Length",
		Description: "SI base unit of length",
	},

	// Length - US Customary
	"foot": {
		Canonical:   "foot",
		Symbol:      "ft",
		Aliases:     []string{"feet", "ft", "foot"},
		System:      "US_Customary",
		Quantity:    "Length",
		Description: "12 inches, 0.3048 meters",
	},
	"mile": {
		Canonical:   "mile",
		Symbol:      "mi",
		Aliases:     []string{"mile", "miles", "mi"},
		System:      "US_Customary",
		Quantity:    "Length",
		Description: "5280 feet, 1609.344 meters",
	},

	// Mass - SI
	"kilogram": {
		Canonical:   "kilogram",
		Symbol:      "kg",
		Aliases:     []string{"kilogram", "kilograms", "kg"},
		System:      "SI",
		Quantity:    "Mass",
		Description: "1000 grams",
	},

	// Mass - US Customary
	"pound": {
		Canonical:   "pound",
		Symbol:      "lb",
		Aliases:     []string{"pound", "pounds", "lb", "lbs"},
		System:      "US_Customary",
		Quantity:    "Mass",
		Description: "Avoirdupois pound, 453.59237 grams",
	},

	// Volume - US Customary
	"cup": {
		Canonical:   "cup",
		Symbol:      "cup",
		Aliases:     []string{"cups", "cup"},
		System:      "US_Customary",
		Quantity:    "Volume",
		Description: "1/16 gallon, 236.5882365 milliliters",
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
