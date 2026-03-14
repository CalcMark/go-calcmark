package units

import "strings"

// MeasurementConfig configures how ambiguous unit names are interpreted.
// Each axis controls an independent dimension of unit ambiguity.
// Only axes that differ from US Customary defaults need to be specified.
type MeasurementConfig struct {
	// Volume: "us" (default) or "imperial"
	Volume string

	// Mass: "standard" (default, avoirdupois — everyday weight: 1 oz = 28.35g)
	// or "troy" (precious metals: 1 troy oz = 31.10g)
	Mass string

	// Ton: "short" (default, US 2000 lb), "long" (Imperial 2240 lb), or "metric" (1000 kg)
	Ton string
}

// DefaultMeasurement returns the default measurement config (US Customary).
func DefaultMeasurement() *MeasurementConfig {
	return &MeasurementConfig{
		Volume: "us",
		Mass:   "standard",
		Ton:    "short",
	}
}

// ambiguousVolumeUnits maps bare unit names to their qualified forms per convention.
var ambiguousVolumeUnits = map[string]map[string]string{
	"us": {
		"gallon": "gallon", "gallons": "gallon", "gal": "gallon",
		"quart": "quart", "quarts": "quart", "qt": "quart",
		"pint": "pint", "pints": "pint", "pt": "pint",
		"cup": "cup", "cups": "cup",
		"fl oz": "fluid ounce", "fluid ounce": "fluid ounce", "fluid ounces": "fluid ounce",
	},
	"imperial": {
		"gallon": "imperial gallon", "gallons": "imperial gallon", "gal": "imperial gallon",
		"quart": "imperial quart", "quarts": "imperial quart", "qt": "imperial quart",
		"pint": "imperial pint", "pints": "imperial pint", "pt": "imperial pint",
		"cup": "imperial cup", "cups": "imperial cup",
		"fl oz": "imperial fluid ounce", "fluid ounce": "imperial fluid ounce", "fluid ounces": "imperial fluid ounce",
	},
}

// ambiguousMassUnits maps bare unit names to their qualified forms per convention.
var ambiguousMassUnits = map[string]map[string]string{
	"standard": {
		"oz": "ounce", "ounce": "ounce", "ounces": "ounce",
		"lb": "pound", "lbs": "pound", "pound": "pound", "pounds": "pound",
	},
	"troy": {
		"oz": "troy ounce", "ounce": "troy ounce", "ounces": "troy ounce",
		"lb": "troy pound", "lbs": "troy pound", "pound": "troy pound", "pounds": "troy pound",
	},
}

// ambiguousTonUnits maps bare "ton" names to their qualified forms per convention.
var ambiguousTonUnits = map[string]map[string]string{
	"short": {
		"ton": "short ton", "tons": "short ton",
	},
	"long": {
		"ton": "long ton", "tons": "long ton",
	},
	"metric": {
		"ton": "metric ton", "tons": "metric ton",
	},
}

// ResolveUnit maps a bare ambiguous unit name to its qualified canonical name
// using the active measurement conventions. Non-ambiguous units and already-qualified
// units (like "troy oz", "imp gal") pass through unchanged.
//
// Examples with default config (US Customary):
//
//	ResolveUnit("oz", nil)   → "oz"     (unchanged — US is default)
//	ResolveUnit("gallon", nil) → "gallon" (unchanged — US is default)
//
// With imperial volume:
//
//	ResolveUnit("gallon", &MeasurementConfig{Volume: "imperial"}) → "imperial gallon"
//	ResolveUnit("imp gal", ...) → "imp gal" (already qualified, unchanged)
func ResolveUnit(unitName string, mc *MeasurementConfig) string {
	if mc == nil {
		return unitName // No measurement config = no resolution needed
	}

	normalized := strings.ToLower(strings.TrimSpace(unitName))

	// Skip already-qualified units (prefixed with us/imp/troy/short/long/metric)
	if isQualifiedUnit(normalized) {
		return unitName
	}

	// Check volume axis
	if mc.Volume != "us" {
		if m, ok := ambiguousVolumeUnits[mc.Volume]; ok {
			if resolved, ok := m[normalized]; ok {
				return resolved
			}
		}
	}

	// Check mass axis
	if mc.Mass != "standard" {
		if m, ok := ambiguousMassUnits[mc.Mass]; ok {
			if resolved, ok := m[normalized]; ok {
				return resolved
			}
		}
	}

	// Check ton axis
	if mc.Ton != "short" {
		if m, ok := ambiguousTonUnits[mc.Ton]; ok {
			if resolved, ok := m[normalized]; ok {
				return resolved
			}
		}
	}

	return unitName
}

// isQualifiedUnit returns true if the unit name already has an explicit
// measurement system qualifier prefix. These are never remapped.
func isQualifiedUnit(normalized string) bool {
	prefixes := []string{"us ", "imp ", "imperial ", "troy ", "short ", "long ", "metric "}
	for _, p := range prefixes {
		if strings.HasPrefix(normalized, p) {
			return true
		}
	}
	return false
}

// IsAmbiguousUnit returns true if the given unit name is ambiguous — i.e., it
// has different definitions depending on measurement convention.
func IsAmbiguousUnit(unitName string) bool {
	normalized := strings.ToLower(strings.TrimSpace(unitName))
	if isQualifiedUnit(normalized) {
		return false
	}
	for _, m := range ambiguousVolumeUnits["us"] {
		_ = m // just checking keys exist
	}
	// Check all axes
	if _, ok := ambiguousVolumeUnits["us"][normalized]; ok {
		return true
	}
	if _, ok := ambiguousMassUnits["standard"][normalized]; ok {
		return true
	}
	if _, ok := ambiguousTonUnits["short"][normalized]; ok {
		return true
	}
	return false
}

// ConventionPrefix returns the prefix that should be shown for a bare ambiguous
// unit under the given measurement config (e.g., "us", "troy", "imp").
// Returns "" if the unit is not ambiguous.
func ConventionPrefix(unitName string, mc *MeasurementConfig) string {
	if mc == nil {
		mc = DefaultMeasurement()
	}

	normalized := strings.ToLower(strings.TrimSpace(unitName))
	if isQualifiedUnit(normalized) {
		return ""
	}

	if _, ok := ambiguousVolumeUnits["us"][normalized]; ok {
		switch mc.Volume {
		case "imperial":
			return "imp"
		default:
			return "us"
		}
	}
	if _, ok := ambiguousMassUnits["standard"][normalized]; ok {
		switch mc.Mass {
		case "troy":
			return "troy"
		default:
			return "us"
		}
	}
	if _, ok := ambiguousTonUnits["short"][normalized]; ok {
		switch mc.Ton {
		case "long":
			return "long"
		case "metric":
			return "metric"
		default:
			return "short"
		}
	}
	return ""
}
