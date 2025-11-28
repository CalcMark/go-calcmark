package display

import (
	"strings"

	"github.com/shopspring/decimal"
)

// UnitScale represents a unit with its conversion factor.
// ToBase converts a value in this unit to the base unit.
// Example: For millimeters in a meter-based family, ToBase = 0.001 (1mm = 0.001m)
type UnitScale struct {
	Unit   string          // Display symbol (e.g., "km", "MB")
	ToBase decimal.Decimal // Multiplier to convert TO base unit (e.g., mm->m: 0.001)
}

// UnitFamily represents a group of related units that can be normalized.
// Units are ordered from smallest to largest (by magnitude of what they measure).
type UnitFamily struct {
	BaseUnit string      // The reference unit (ToBase = 1)
	Units    []UnitScale // Ordered smallest to largest
}

// unitFamilies defines all known unit hierarchies for display normalization.
var unitFamilies map[string]*UnitFamily

// unitToFamily maps unit names/aliases to their family for O(1) lookup.
var unitToFamily map[string]*UnitFamily

// aliasToCanonical maps unit aliases to canonical display symbol.
var aliasToCanonical map[string]string

func init() {
	buildUnitFamilies()
}

// d is a helper to create decimals from strings for exact precision
func d(s string) decimal.Decimal {
	v, _ := decimal.NewFromString(s)
	return v
}

func buildUnitFamilies() {
	unitFamilies = make(map[string]*UnitFamily)
	unitToFamily = make(map[string]*UnitFamily)
	aliasToCanonical = make(map[string]string)

	// SI Length (base: meter)
	// ToBase: how many meters is 1 of this unit?
	siLength := &UnitFamily{
		BaseUnit: "m",
		Units: []UnitScale{
			{"mm", d("0.001")}, // 1mm = 0.001m
			{"cm", d("0.01")},  // 1cm = 0.01m
			{"m", d("1")},      // 1m = 1m
			{"km", d("1000")},  // 1km = 1000m
		},
	}
	registerFamily("si_length", siLength, map[string][]string{
		"mm": {"mm", "millimeter", "millimeters", "millimetre", "millimetres"},
		"cm": {"cm", "centimeter", "centimeters", "centimetre", "centimetres"},
		"m":  {"m", "meter", "meters", "metre", "metres"},
		"km": {"km", "kilometer", "kilometers", "kilometre", "kilometres"},
	})

	// US Customary Length (base: foot)
	usLength := &UnitFamily{
		BaseUnit: "ft",
		Units: []UnitScale{
			// Using fractional representation for exact math: 1 inch = 1/12 foot
			{"in", decimal.NewFromInt(1).Div(decimal.NewFromInt(12))}, // 1in = 1/12 ft
			{"ft", d("1")},    // 1ft = 1ft
			{"yd", d("3")},    // 1yd = 3ft
			{"mi", d("5280")}, // 1mi = 5280ft
		},
	}
	registerFamily("us_length", usLength, map[string][]string{
		"in": {"in", "inch", "inches"},
		"ft": {"ft", "foot", "feet"},
		"yd": {"yd", "yard", "yards"},
		"mi": {"mi", "mile", "miles"},
	})

	// SI Mass (base: gram)
	siMass := &UnitFamily{
		BaseUnit: "g",
		Units: []UnitScale{
			{"mg", d("0.001")},  // 1mg = 0.001g
			{"g", d("1")},       // 1g = 1g
			{"kg", d("1000")},   // 1kg = 1000g
			{"t", d("1000000")}, // 1t = 1000000g
		},
	}
	registerFamily("si_mass", siMass, map[string][]string{
		"mg": {"mg", "milligram", "milligrams"},
		"g":  {"g", "gram", "grams"},
		"kg": {"kg", "kilogram", "kilograms"},
		"t":  {"t", "tonne", "tonnes", "metric ton", "metric tons"},
	})

	// US Customary Mass (base: ounce)
	usMass := &UnitFamily{
		BaseUnit: "oz",
		Units: []UnitScale{
			{"oz", d("1")},  // 1oz = 1oz
			{"lb", d("16")}, // 1lb = 16oz
		},
	}
	registerFamily("us_mass", usMass, map[string][]string{
		"oz": {"oz", "ounce", "ounces"},
		"lb": {"lb", "lbs", "pound", "pounds"},
	})

	// SI Volume (base: liter)
	siVolume := &UnitFamily{
		BaseUnit: "l",
		Units: []UnitScale{
			{"ml", d("0.001")}, // 1ml = 0.001l
			{"l", d("1")},      // 1l = 1l
		},
	}
	registerFamily("si_volume", siVolume, map[string][]string{
		"ml": {"ml", "milliliter", "milliliters", "millilitre", "millilitres"},
		"l":  {"l", "liter", "liters", "litre", "litres"},
	})

	// US Customary Volume (base: cup)
	// Hierarchy: tsp < tbsp < cup < pt < qt < gal
	usVolume := &UnitFamily{
		BaseUnit: "cup",
		Units: []UnitScale{
			{"tsp", decimal.NewFromInt(1).Div(decimal.NewFromInt(48))},  // 1tsp = 1/48 cup
			{"tbsp", decimal.NewFromInt(1).Div(decimal.NewFromInt(16))}, // 1tbsp = 1/16 cup
			{"cup", d("1")},  // 1cup = 1cup
			{"pt", d("2")},   // 1pt = 2cups
			{"qt", d("4")},   // 1qt = 4cups
			{"gal", d("16")}, // 1gal = 16cups
		},
	}
	registerFamily("us_volume", usVolume, map[string][]string{
		"tsp":  {"tsp", "teaspoon", "teaspoons"},
		"tbsp": {"tbsp", "tablespoon", "tablespoons"},
		"cup":  {"cup", "cups"},
		"pt":   {"pt", "pint", "pints"},
		"qt":   {"qt", "quart", "quarts"},
		"gal":  {"gal", "gallon", "gallons"},
	})

	// Data Storage (binary, base: byte)
	dataStorage := &UnitFamily{
		BaseUnit: "bytes",
		Units: []UnitScale{
			{"bytes", d("1")},
			{"KB", d("1024")},
			{"MB", d("1048576")},          // 1024^2
			{"GB", d("1073741824")},       // 1024^3
			{"TB", d("1099511627776")},    // 1024^4
			{"PB", d("1125899906842624")}, // 1024^5
		},
	}
	registerFamily("data_storage", dataStorage, map[string][]string{
		"bytes": {"b", "byte", "bytes"},
		"KB":    {"kb", "KB", "kilobyte", "kilobytes"},
		"MB":    {"mb", "MB", "megabyte", "megabytes"},
		"GB":    {"gb", "GB", "gigabyte", "gigabytes"},
		"TB":    {"tb", "TB", "terabyte", "terabytes"},
		"PB":    {"pb", "PB", "petabyte", "petabytes"},
	})

	// Power (base: watt)
	power := &UnitFamily{
		BaseUnit: "W",
		Units: []UnitScale{
			{"W", d("1")},
			{"kW", d("1000")},
			{"MW", d("1000000")},
		},
	}
	registerFamily("power", power, map[string][]string{
		"W":  {"w", "W", "watt", "watts"},
		"kW": {"kw", "kW", "kilowatt", "kilowatts"},
		"MW": {"mw", "MW", "megawatt", "megawatts"},
	})

	// Energy (base: joule) - SI
	energySI := &UnitFamily{
		BaseUnit: "J",
		Units: []UnitScale{
			{"J", d("1")},
			{"kJ", d("1000")},
		},
	}
	registerFamily("energy_si", energySI, map[string][]string{
		"J":  {"j", "J", "joule", "joules"},
		"kJ": {"kj", "kJ", "kilojoule", "kilojoules"},
	})

	// Energy (base: calorie) - food/nutrition
	energyCal := &UnitFamily{
		BaseUnit: "cal",
		Units: []UnitScale{
			{"cal", d("1")},
			{"kcal", d("1000")},
		},
	}
	registerFamily("energy_cal", energyCal, map[string][]string{
		"cal":  {"cal", "calorie", "calories"},
		"kcal": {"kcal", "kilocalorie", "kilocalories"},
	})

	// Area SI (base: square meter)
	// Using "sq X" format for display (more readable than Unicode superscripts)
	areaSI := &UnitFamily{
		BaseUnit: "sq m",
		Units: []UnitScale{
			{"sq cm", d("0.0001")}, // 1 sq cm = 0.0001 sq m
			{"sq m", d("1")},
			{"ha", d("10000")},      // 1 ha = 10000 sq m
			{"sq km", d("1000000")}, // 1 sq km = 1000000 sq m
		},
	}
	registerFamily("area_si", areaSI, map[string][]string{
		"sq cm": {"cm²", "cm2", "sq cm", "square centimeter", "square centimeters"},
		"sq m":  {"m²", "m2", "sq m", "square meter", "square meters", "square metre", "square metres"},
		"ha":    {"ha", "hectare", "hectares"},
		"sq km": {"km²", "km2", "sq km", "square kilometer", "square kilometers"},
	})

	// Area US (base: square foot)
	areaUS := &UnitFamily{
		BaseUnit: "sq ft",
		Units: []UnitScale{
			{"sq in", decimal.NewFromInt(1).Div(decimal.NewFromInt(144))}, // 1 sq in = 1/144 sq ft
			{"sq ft", d("1")},
			{"sq yd", d("9")},        // 1 sq yd = 9 sq ft
			{"ac", d("43560")},       // 1 ac = 43560 sq ft
			{"sq mi", d("27878400")}, // 1 sq mi = 27878400 sq ft
		},
	}
	registerFamily("area_us", areaUS, map[string][]string{
		"sq in": {"in²", "in2", "sq in", "square inch", "square inches"},
		"sq ft": {"ft²", "ft2", "sq ft", "square foot", "square feet"},
		"sq yd": {"yd²", "yd2", "sq yd", "square yard", "square yards"},
		"ac":    {"ac", "acre", "acres"},
		"sq mi": {"mi²", "mi2", "sq mi", "square mile", "square miles"},
	})
}

// registerFamily adds a unit family and maps all aliases to it.
// aliasMap maps canonical unit symbol -> list of aliases (including the symbol itself)
func registerFamily(name string, family *UnitFamily, aliasMap map[string][]string) {
	unitFamilies[name] = family
	for canonicalUnit, aliases := range aliasMap {
		for _, alias := range aliases {
			unitToFamily[strings.ToLower(alias)] = family
			aliasToCanonical[strings.ToLower(alias)] = canonicalUnit
		}
	}
}

// epsilon for floating-point comparisons to handle decimal precision issues
var epsilon = decimal.NewFromFloat(0.0001)

// NormalizeForDisplay converts a value+unit to the most human-readable form.
// It finds the largest unit in the same family where the value is >= 1.
// Unknown units are returned unchanged.
//
// Examples:
//
//	NormalizeForDisplay(1000, "m") → (1, "km")
//	NormalizeForDisplay(23400000, "GB") → (22.89, "PB")
//	NormalizeForDisplay(100000, "users") → (100000, "users")  // unknown unit
func NormalizeForDisplay(value decimal.Decimal, unit string) (decimal.Decimal, string) {
	// Handle zero
	if value.IsZero() {
		return value, normalizeUnitSymbol(unit)
	}

	// Look up the unit family
	family, ok := unitToFamily[strings.ToLower(unit)]
	if !ok {
		// Unknown unit - return as-is
		return value, unit
	}

	// Find the current unit's scale in the family
	currentScale := findUnitScale(family, unit)
	if currentScale == nil {
		return value, unit
	}

	// Convert to base unit: value * ToBase
	// e.g., 1000 mm * 0.001 = 1 meter (base)
	baseValue := value.Mul(currentScale.ToBase)

	// Handle negative values
	isNegative := baseValue.IsNegative()
	absBaseValue := baseValue.Abs()

	// Find the best unit: the LARGEST unit where abs(value) >= 1 (with epsilon tolerance)
	// We iterate from smallest to largest, keeping track of the last valid one
	var bestUnit *UnitScale
	var bestValue decimal.Decimal
	one := decimal.NewFromInt(1)
	oneMinusEpsilon := one.Sub(epsilon) // Allow 0.9999... to be treated as >= 1

	for i := range family.Units {
		scale := &family.Units[i]
		// Convert from base to this unit: baseValue / ToBase
		// e.g., 1 meter / 1000 = 0.001 km, or 1 meter / 0.001 = 1000 mm
		testValue := absBaseValue.Div(scale.ToBase)

		// If this unit gives us a value >= 1 (with tolerance), it's a candidate
		// Keep updating to find the largest valid unit
		if testValue.GreaterThanOrEqual(oneMinusEpsilon) {
			bestUnit = scale
			bestValue = testValue
		}
	}

	// If no unit gives value >= 1, use the smallest unit
	if bestUnit == nil {
		bestUnit = &family.Units[0]
		bestValue = absBaseValue.Div(bestUnit.ToBase)
	}

	// Restore sign
	if isNegative {
		bestValue = bestValue.Neg()
	}

	// Round to reasonable precision for display
	bestValue = roundForDisplay(bestValue)

	return bestValue, bestUnit.Unit
}

// findUnitScale finds a unit's scale info in a family.
func findUnitScale(family *UnitFamily, unit string) *UnitScale {
	unitLower := strings.ToLower(unit)

	// First, try to find the canonical symbol for this alias
	canonical, hasCanonical := aliasToCanonical[unitLower]
	if hasCanonical {
		// Look for the canonical unit in the family
		for i := range family.Units {
			if family.Units[i].Unit == canonical {
				return &family.Units[i]
			}
		}
	}

	// Direct match on display symbol (case-insensitive)
	for i := range family.Units {
		if strings.ToLower(family.Units[i].Unit) == unitLower {
			return &family.Units[i]
		}
	}

	// Fallback to base unit
	for i := range family.Units {
		if family.Units[i].Unit == family.BaseUnit {
			return &family.Units[i]
		}
	}

	// Fallback to first unit
	if len(family.Units) > 0 {
		return &family.Units[0]
	}

	return nil
}

// normalizeUnitSymbol returns the canonical display symbol for a unit.
func normalizeUnitSymbol(unit string) string {
	// Map common variations to canonical symbols
	aliases := map[string]string{
		"meter": "m", "meters": "m", "metre": "m", "metres": "m",
		"kilometer": "km", "kilometers": "km", "kilometre": "km", "kilometres": "km",
		"centimeter": "cm", "centimeters": "cm", "centimetre": "cm", "centimetres": "cm",
		"millimeter": "mm", "millimeters": "mm", "millimetre": "mm", "millimetres": "mm",
		"foot": "ft", "feet": "ft",
		"inch": "in", "inches": "in",
		"yard": "yd", "yards": "yd",
		"mile": "mi", "miles": "mi",
		"gram": "g", "grams": "g",
		"kilogram": "kg", "kilograms": "kg",
		"milligram": "mg", "milligrams": "mg",
		"tonne": "t", "tonnes": "t", "metric ton": "t", "metric tons": "t",
		"ounce": "oz", "ounces": "oz",
		"pound": "lb", "pounds": "lb", "lbs": "lb",
		"liter": "l", "liters": "l", "litre": "l", "litres": "l",
		"milliliter": "ml", "milliliters": "ml", "millilitre": "ml", "millilitres": "ml",
		"teaspoon": "tsp", "teaspoons": "tsp",
		"tablespoon": "tbsp", "tablespoons": "tbsp",
		"cups": "cup",
		"pint": "pt", "pints": "pt",
		"quart": "qt", "quarts": "qt",
		"gallon": "gal", "gallons": "gal",
		"byte":     "bytes",
		"kilobyte": "KB", "kilobytes": "KB", "kb": "KB",
		"megabyte": "MB", "megabytes": "MB", "mb": "MB",
		"gigabyte": "GB", "gigabytes": "GB", "gb": "GB",
		"terabyte": "TB", "terabytes": "TB", "tb": "TB",
		"petabyte": "PB", "petabytes": "PB", "pb": "PB",
		"watt": "W", "watts": "W", "w": "W",
		"kilowatt": "kW", "kilowatts": "kW", "kw": "kW",
		"megawatt": "MW", "megawatts": "MW", "mw": "MW",
		"joule": "J", "joules": "J", "j": "J",
		"kilojoule": "kJ", "kilojoules": "kJ", "kj": "kJ",
		"calorie": "cal", "calories": "cal",
		"kilocalorie": "kcal", "kilocalories": "kcal",
		"hectare": "ha", "hectares": "ha",
		"acre": "ac", "acres": "ac",
		"square meter": "m²", "square meters": "m²", "square metre": "m²", "square metres": "m²",
		"square kilometer": "km²", "square kilometers": "km²",
		"square foot": "ft²", "square feet": "ft²",
		"square inch": "in²", "square inches": "in²",
		"square yard": "yd²", "square yards": "yd²",
		"square mile": "mi²", "square miles": "mi²",
	}

	if symbol, ok := aliases[strings.ToLower(unit)]; ok {
		return symbol
	}
	return unit
}

// roundForDisplay rounds a value to appropriate precision for human readability.
func roundForDisplay(value decimal.Decimal) decimal.Decimal {
	absValue := value.Abs()

	// For values >= 100, show no decimals
	if absValue.GreaterThanOrEqual(decimal.NewFromInt(100)) {
		return value.Round(0)
	}

	// For values >= 10, show 1 decimal
	if absValue.GreaterThanOrEqual(decimal.NewFromInt(10)) {
		return value.Round(1)
	}

	// For values >= 1, show 2 decimals
	if absValue.GreaterThanOrEqual(decimal.NewFromInt(1)) {
		return value.Round(2)
	}

	// For small values, show more precision
	return value.Round(4)
}
