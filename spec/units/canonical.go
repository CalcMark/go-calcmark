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

	// ========================================
	// TEMPERATURE
	// ========================================

	"celsius": {
		Canonical:   "celsius",
		Symbol:      "C",
		Aliases:     []string{"celsius", "c", "°c", "degc"},
		System:      "SI",
		Quantity:    "Temperature",
		Description: "Celsius temperature scale, 0°C = freezing point of water",
	},
	"fahrenheit": {
		Canonical:   "fahrenheit",
		Symbol:      "F",
		Aliases:     []string{"fahrenheit", "f", "°f", "degf"},
		System:      "Imperial",
		Quantity:    "Temperature",
		Description: "Fahrenheit temperature scale, 32°F = freezing point of water",
	},
	"kelvin": {
		Canonical:   "kelvin",
		Symbol:      "K",
		Aliases:     []string{"kelvin", "k"},
		System:      "SI",
		Quantity:    "Temperature",
		Description: "Absolute temperature scale, 0 K = absolute zero",
	},

	// ========================================
	// SPEED
	// ========================================

	"m/s": {
		Canonical:   "m/s",
		Symbol:      "m/s",
		Aliases:     []string{"m/s", "mps", "meters per second", "metres per second"},
		System:      "SI",
		Quantity:    "Speed",
		Description: "Meters per second, SI unit of speed",
	},
	"meters per second": { // Alias key for multi-word lookup
		Canonical:   "m/s",
		Symbol:      "m/s",
		Aliases:     []string{"m/s", "mps", "meters per second", "metres per second"},
		System:      "SI",
		Quantity:    "Speed",
		Description: "Meters per second, SI unit of speed",
	},
	"km/h": {
		Canonical:   "km/h",
		Symbol:      "km/h",
		Aliases:     []string{"km/h", "kph", "kmh", "kilometers per hour", "kilometres per hour"},
		System:      "SI",
		Quantity:    "Speed",
		Description: "Kilometers per hour",
	},
	"kilometers per hour": { // Alias key for multi-word lookup
		Canonical:   "km/h",
		Symbol:      "km/h",
		Aliases:     []string{"km/h", "kph", "kmh", "kilometers per hour", "kilometres per hour"},
		System:      "SI",
		Quantity:    "Speed",
		Description: "Kilometers per hour",
	},
	"mph": {
		Canonical:   "mph",
		Symbol:      "mph",
		Aliases:     []string{"mph", "miles per hour"},
		System:      "Imperial",
		Quantity:    "Speed",
		Description: "Miles per hour",
	},
	"miles per hour": { // Alias key for multi-word lookup
		Canonical:   "mph",
		Symbol:      "mph",
		Aliases:     []string{"mph", "miles per hour"},
		System:      "Imperial",
		Quantity:    "Speed",
		Description: "Miles per hour",
	},
	"knot": {
		Canonical:   "knot",
		Symbol:      "knot",
		Aliases:     []string{"knot", "knots"},
		System:      "Nautical",
		Quantity:    "Speed",
		Description: "Nautical mile per hour, 1 knot = 1.852 km/h",
	},

	// ========================================
	// ENERGY
	// ========================================

	"joule": {
		Canonical:   "joule",
		Symbol:      "J",
		Aliases:     []string{"joule", "joules", "j"},
		System:      "SI",
		Quantity:    "Energy",
		Description: "SI unit of energy, 1 joule = 1 newton-meter",
	},
	"kilojoule": {
		Canonical:   "kilojoule",
		Symbol:      "kJ",
		Aliases:     []string{"kilojoule", "kilojoules", "kj"},
		System:      "SI",
		Quantity:    "Energy",
		Description: "1000 joules",
	},
	"calorie": {
		Canonical:   "calorie",
		Symbol:      "cal",
		Aliases:     []string{"calorie", "calories", "cal"},
		System:      "CGS",
		Quantity:    "Energy",
		Description: "Thermochemical calorie, 1 cal = 4.184 J",
	},
	"kilocalorie": {
		Canonical:   "kilocalorie",
		Symbol:      "kcal",
		Aliases:     []string{"kilocalorie", "kilocalories", "kcal"},
		System:      "CGS",
		Quantity:    "Energy",
		Description: "Food calorie (Calorie), 1 kcal = 4184 J",
	},
	"kwh": {
		Canonical:   "kwh",
		Symbol:      "kWh",
		Aliases:     []string{"kwh", "kilowatt-hour", "kilowatt-hours"},
		System:      "SI",
		Quantity:    "Energy",
		Description: "Kilowatt-hour, commonly used for electricity, 1 kWh = 3.6 MJ",
	},

	// ========================================
	// POWER
	// ========================================

	"watt": {
		Canonical:   "watt",
		Symbol:      "W",
		Aliases:     []string{"watt", "watts", "w"},
		System:      "SI",
		Quantity:    "Power",
		Description: "SI unit of power, 1 watt = 1 joule/second",
	},
	"kilowatt": {
		Canonical:   "kilowatt",
		Symbol:      "kW",
		Aliases:     []string{"kilowatt", "kilowatts", "kw"},
		System:      "SI",
		Quantity:    "Power",
		Description: "1000 watts",
	},
	"megawatt": {
		Canonical:   "megawatt",
		Symbol:      "MW",
		Aliases:     []string{"megawatt", "megawatts", "mw"},
		System:      "SI",
		Quantity:    "Power",
		Description: "1 million watts",
	},
	"horsepower": {
		Canonical:   "horsepower",
		Symbol:      "hp",
		Aliases:     []string{"horsepower", "hp"},
		System:      "Imperial",
		Quantity:    "Power",
		Description: "Mechanical horsepower, 1 hp = 745.7 W",
	},

	// ========== AREA UNITS ==========

	// Area - SI
	"square meter": {
		Canonical:   "square meter",
		Symbol:      "m²",
		Aliases:     []string{"square meter", "square meters", "square metre", "square metres", "m²", "m2", "sq m"},
		System:      "SI",
		Quantity:    "Area",
		Description: "SI unit of area",
	},
	"square kilometer": {
		Canonical:   "square kilometer",
		Symbol:      "km²",
		Aliases:     []string{"square kilometer", "square kilometers", "square kilometre", "square kilometres", "km²", "km2", "sq km"},
		System:      "SI",
		Quantity:    "Area",
		Description: "1 km² = 1,000,000 m²",
	},
	"square centimeter": {
		Canonical:   "square centimeter",
		Symbol:      "cm²",
		Aliases:     []string{"square centimeter", "square centimeters", "square centimetre", "square centimetres", "cm²", "cm2", "sq cm"},
		System:      "SI",
		Quantity:    "Area",
		Description: "1 cm² = 0.0001 m²",
	},
	"hectare": {
		Canonical:   "hectare",
		Symbol:      "ha",
		Aliases:     []string{"hectare", "hectares", "ha"},
		System:      "SI",
		Quantity:    "Area",
		Description: "1 ha = 10,000 m²",
	},

	// Area - Imperial/US
	"square foot": {
		Canonical:   "square foot",
		Symbol:      "ft²",
		Aliases:     []string{"square foot", "square feet", "ft²", "ft2", "sq ft"},
		System:      "Imperial",
		Quantity:    "Area",
		Description: "1 ft² = 0.09290304 m²",
	},
	"square inch": {
		Canonical:   "square inch",
		Symbol:      "in²",
		Aliases:     []string{"square inch", "square inches", "in²", "in2", "sq in"},
		System:      "Imperial",
		Quantity:    "Area",
		Description: "1 in² = 0.00064516 m²",
	},
	"square yard": {
		Canonical:   "square yard",
		Symbol:      "yd²",
		Aliases:     []string{"square yard", "square yards", "yd²", "yd2", "sq yd"},
		System:      "Imperial",
		Quantity:    "Area",
		Description: "1 yd² = 0.83612736 m²",
	},
	"square mile": {
		Canonical:   "square mile",
		Symbol:      "mi²",
		Aliases:     []string{"square mile", "square miles", "mi²", "mi2", "sq mi"},
		System:      "Imperial",
		Quantity:    "Area",
		Description: "1 mi² = 2,589,988.110336 m²",
	},
	"acre": {
		Canonical:   "acre",
		Symbol:      "ac",
		Aliases:     []string{"acre", "acres", "ac"},
		System:      "US_Customary",
		Quantity:    "Area",
		Description: "1 acre = 4,046.8564224 m²",
	},

	// ========== END OF UNITS ==========
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
