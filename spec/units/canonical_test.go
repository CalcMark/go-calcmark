package units_test

import (
	"slices"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/units"
)

// TestNormalizeUnitName_Comprehensive tests all unit mappings
func TestNormalizeUnitName_Comprehensive(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		canonical string
		found     bool
	}{
		// Length - SI
		{"meter canonical", "meter", "meter", true},
		{"meter plural", "meters", "meter", true},
		{"meter symbol", "m", "meter", true},
		{"metre british", "metre", "meter", true},
		{"metres plural", "metres", "meter", true},

		{"kilometer canonical", "kilometer", "kilometer", true},
		{"km symbol", "km", "kilometer", true},

		// Length - US Customary
		{"foot canonical", "foot", "foot", true},
		{"feet plural", "feet", "foot", true},
		{"ft symbol", "ft", "foot", true},

		{"mile canonical", "mile", "mile", true},
		{"miles plural", "miles", "mile", true},
		{"mi symbol", "mi", "mile", true},

		// Mass - SI
		{"kilogram canonical", "kilogram", "kilogram", true},
		{"kg symbol", "kg", "kilogram", true},
		{"kilograms plural", "kilograms", "kilogram", true},

		// Mass - US
		{"pound canonical", "pound", "pound", true},
		{"pounds plural", "pounds", "pound", true},
		{"lb symbol", "lb", "pound", true},
		{"lbs common", "lbs", "pound", true},

		// Volume - US
		{"cup canonical", "cup", "cup", true},
		{"cups plural", "cups", "cup", true},

		// Case insensitive
		{"METER uppercase", "METER", "meter", true},
		{"MeTeR mixed", "MeTeR", "meter", true},

		// Whitespace handling
		{"meter with spaces", "  meter  ", "meter", true},

		// Not found
		{"invalid unit", "foobar", "", false},
		{"empty string", "", "", false},
		{"unknown abbr", "xyz", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canonical, found := units.NormalizeUnitName(tt.input)

			if found != tt.found {
				t.Errorf("NormalizeUnitName(%q) found = %v, want %v",
					tt.input, found, tt.found)
			}

			if canonical != tt.canonical {
				t.Errorf("NormalizeUnitName(%q) = %q, want %q",
					tt.input, canonical, tt.canonical)
			}
		})
	}
}

// TestGetUnitSymbol_AllUnits ensures all units have valid symbols
func TestGetUnitSymbol_AllUnits(t *testing.T) {
	tests := []struct {
		unitName string
		symbol   string
	}{
		{"meter", "m"},
		{"meters", "m"},
		{"foot", "ft"},
		{"feet", "ft"},
		{"mile", "mi"},
		{"kilogram", "kg"},
		{"pound", "lb"},
		{"cup", "cup"},
	}

	for _, tt := range tests {
		t.Run(tt.unitName, func(t *testing.T) {
			symbol, found := units.GetUnitSymbol(tt.unitName)

			if !found {
				t.Errorf("GetUnitSymbol(%q) not found", tt.unitName)
			}

			if symbol != tt.symbol {
				t.Errorf("GetUnitSymbol(%q) = %q, want %q",
					tt.unitName, symbol, tt.symbol)
			}
		})
	}
}

// TestUnitSystem verifies units are categorized correctly
func TestUnitSystem(t *testing.T) {
	siUnits := []string{"meter", "kilogram"}
	usUnits := []string{"foot", "pound", "mile", "cup"}

	for _, unit := range siUnits {
		mapping, ok := units.StandardUnits[unit]
		if !ok {
			t.Errorf("Unit %q not found in StandardUnits", unit)
			continue
		}
		if mapping.System != "SI" {
			t.Errorf("Unit %q has system %q, want SI", unit, mapping.System)
		}
	}

	for _, unit := range usUnits {
		mapping, ok := units.StandardUnits[unit]
		if !ok {
			t.Errorf("Unit %q not found in StandardUnits", unit)
			continue
		}
		if mapping.System != "US_Customary" {
			t.Errorf("Unit %q has system %q, want US_Customary", unit, mapping.System)
		}
	}
}

// TestUnitQuantityTypes ensures units have correct quantity types
func TestUnitQuantityTypes(t *testing.T) {
	lengthUnits := []string{"meter", "foot", "mile"}
	massUnits := []string{"kilogram", "pound"}
	volumeUnits := []string{"cup"}

	for _, unit := range lengthUnits {
		mapping := units.StandardUnits[unit]
		if mapping.Quantity != "Length" {
			t.Errorf("Unit %q has quantity %q, want Length", unit, mapping.Quantity)
		}
	}

	for _, unit := range massUnits {
		mapping := units.StandardUnits[unit]
		if mapping.Quantity != "Mass" {
			t.Errorf("Unit %q has quantity %q, want Mass", unit, mapping.Quantity)
		}
	}

	for _, unit := range volumeUnits {
		mapping := units.StandardUnits[unit]
		if mapping.Quantity != "Volume" {
			t.Errorf("Unit %q has quantity %q, want Volume", unit, mapping.Quantity)
		}
	}
}

// TestAllUnitsHaveAliases ensures every unit has at least one alias
func TestAllUnitsHaveAliases(t *testing.T) {
	for canonical, mapping := range units.StandardUnits {
		if len(mapping.Aliases) == 0 {
			t.Errorf("Unit %q has no aliases", canonical)
		}

		// Canonical name should be in aliases
		if !slices.Contains(mapping.Aliases, canonical) {
			t.Errorf("Unit %q canonical name not in aliases: %v", canonical, mapping.Aliases)
		}
	}
}

// NOTE: Unit conversion tests would go in spec/interpreter or spec/semantic
// since conversion is semantic, not lexical/syntactic
