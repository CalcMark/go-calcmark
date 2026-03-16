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

	// Mass - US Customary (standard = avoirdupois, the everyday weight system)
	"ounce": {
		Canonical:   "ounce",
		Symbol:      "oz",
		Aliases:     []string{"ounce", "ounces", "oz", "us oz"},
		System:      "US_Customary",
		Quantity:    "Mass",
		Description: "Standard (avoirdupois) ounce, 1/16 pound, 28.349523125 grams",
	},
	"pound": {
		Canonical:   "pound",
		Symbol:      "lb",
		Aliases:     []string{"pound", "pounds", "lb", "lbs", "us lb"},
		System:      "US_Customary",
		Quantity:    "Mass",
		Description: "Standard (avoirdupois) pound, 453.59237 grams",
	},

	// Mass - Troy (precious metals: 1 troy oz = 31.1035g)
	"troy ounce": {
		Canonical:   "troy ounce",
		Symbol:      "troy oz",
		Aliases:     []string{"troy ounce", "troy ounces", "troy oz"},
		System:      "Troy",
		Quantity:    "Mass",
		Description: "Troy ounce, used for precious metals, 31.1035 grams",
	},
	"troy pound": {
		Canonical:   "troy pound",
		Symbol:      "troy lb",
		Aliases:     []string{"troy pound", "troy pounds", "troy lb"},
		System:      "Troy",
		Quantity:    "Mass",
		Description: "Troy pound, 12 troy ounces, 373.2417 grams",
	},

	// Mass - Ton variants
	"short ton": {
		Canonical:   "short ton",
		Symbol:      "short ton",
		Aliases:     []string{"short ton", "short tons"},
		System:      "US_Customary",
		Quantity:    "Mass",
		Description: "US short ton, 2000 pounds, 907.185 kg",
	},
	"long ton": {
		Canonical:   "long ton",
		Symbol:      "long ton",
		Aliases:     []string{"long ton", "long tons"},
		System:      "Imperial",
		Quantity:    "Mass",
		Description: "Imperial long ton, 2240 pounds, 1016.047 kg",
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

	// Volume - US Customary (with "us" prefix aliases for explicit qualification)
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
		Aliases:     []string{"cup", "cups", "us cup"},
		System:      "US_Customary",
		Quantity:    "Volume",
		Description: "US legal cup, 240 milliliters",
	},
	"fluid ounce": {
		Canonical:   "fluid ounce",
		Symbol:      "fl oz",
		Aliases:     []string{"fluid ounce", "fluid ounces", "fl oz", "us fl oz"},
		System:      "US_Customary",
		Quantity:    "Volume",
		Description: "US fluid ounce, 29.5735 milliliters",
	},
	"pint": {
		Canonical:   "pint",
		Symbol:      "pt",
		Aliases:     []string{"pint", "pints", "pt", "us pt"},
		System:      "US_Customary",
		Quantity:    "Volume",
		Description: "US liquid pint, 2 cups, 473.176 milliliters",
	},
	"quart": {
		Canonical:   "quart",
		Symbol:      "qt",
		Aliases:     []string{"quart", "quarts", "qt", "us qt"},
		System:      "US_Customary",
		Quantity:    "Volume",
		Description: "US liquid quart, 2 pints, 946.353 milliliters",
	},
	"gallon": {
		Canonical:   "gallon",
		Symbol:      "gal",
		Aliases:     []string{"gallon", "gallons", "gal", "us gal"},
		System:      "US_Customary",
		Quantity:    "Volume",
		Description: "US liquid gallon, 4 quarts, 3.785411784 liters",
	},

	// Volume - Imperial
	"imperial gallon": {
		Canonical:   "imperial gallon",
		Symbol:      "imp gal",
		Aliases:     []string{"imperial gallon", "imperial gallons", "imp gal"},
		System:      "Imperial",
		Quantity:    "Volume",
		Description: "Imperial gallon, 4.54609 liters",
	},
	"imperial quart": {
		Canonical:   "imperial quart",
		Symbol:      "imp qt",
		Aliases:     []string{"imperial quart", "imperial quarts", "imp qt"},
		System:      "Imperial",
		Quantity:    "Volume",
		Description: "Imperial quart, 1.13652 liters",
	},
	"imperial pint": {
		Canonical:   "imperial pint",
		Symbol:      "imp pt",
		Aliases:     []string{"imperial pint", "imperial pints", "imp pt"},
		System:      "Imperial",
		Quantity:    "Volume",
		Description: "Imperial pint, 568.261 milliliters",
	},
	"imperial cup": {
		Canonical:   "imperial cup",
		Symbol:      "imp cup",
		Aliases:     []string{"imperial cup", "imperial cups", "imp cup"},
		System:      "Imperial",
		Quantity:    "Volume",
		Description: "Imperial cup, 284.131 milliliters",
	},
	"imperial fluid ounce": {
		Canonical:   "imperial fluid ounce",
		Symbol:      "imp fl oz",
		Aliases:     []string{"imperial fluid ounce", "imperial fluid ounces", "imp fl oz"},
		System:      "Imperial",
		Quantity:    "Volume",
		Description: "Imperial fluid ounce, 28.4131 milliliters",
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
		Symbol:      "m2",
		Aliases:     []string{"square meter", "square meters", "square metre", "square metres", "m²", "m2", "sq m"},
		System:      "SI",
		Quantity:    "Area",
		Description: "SI unit of area",
	},
	"square kilometer": {
		Canonical:   "square kilometer",
		Symbol:      "km2",
		Aliases:     []string{"square kilometer", "square kilometers", "square kilometre", "square kilometres", "km²", "km2", "sq km"},
		System:      "SI",
		Quantity:    "Area",
		Description: "1,000,000 square meters",
	},
	"square centimeter": {
		Canonical:   "square centimeter",
		Symbol:      "cm2",
		Aliases:     []string{"square centimeter", "square centimeters", "square centimetre", "square centimetres", "cm²", "cm2", "sq cm"},
		System:      "SI",
		Quantity:    "Area",
		Description: "0.0001 square meters",
	},
	"hectare": {
		Canonical:   "hectare",
		Symbol:      "ha",
		Aliases:     []string{"hectare", "hectares", "ha"},
		System:      "SI",
		Quantity:    "Area",
		Description: "10,000 square meters",
	},

	// Area - Imperial/US
	"square foot": {
		Canonical:   "square foot",
		Symbol:      "ft2",
		Aliases:     []string{"square foot", "square feet", "ft²", "ft2", "sq ft"},
		System:      "Imperial",
		Quantity:    "Area",
		Description: "0.0929 square meters",
	},
	"square inch": {
		Canonical:   "square inch",
		Symbol:      "in2",
		Aliases:     []string{"square inch", "square inches", "in²", "in2", "sq in"},
		System:      "Imperial",
		Quantity:    "Area",
		Description: "0.000645 square meters",
	},
	"square yard": {
		Canonical:   "square yard",
		Symbol:      "yd2",
		Aliases:     []string{"square yard", "square yards", "yd²", "yd2", "sq yd"},
		System:      "Imperial",
		Quantity:    "Area",
		Description: "0.836 square meters",
	},
	"square mile": {
		Canonical:   "square mile",
		Symbol:      "mi2",
		Aliases:     []string{"square mile", "square miles", "mi²", "mi2", "sq mi"},
		System:      "Imperial",
		Quantity:    "Area",
		Description: "2.59 million square meters",
	},
	"acre": {
		Canonical:   "acre",
		Symbol:      "ac",
		Aliases:     []string{"acre", "acres", "ac"},
		System:      "US_Customary",
		Quantity:    "Area",
		Description: "4,047 square meters",
	},

	// ========== FORCE UNITS ==========

	"newton": {
		Canonical:   "newton",
		Symbol:      "N",
		Aliases:     []string{"newton", "newtons"},
		System:      "SI",
		Quantity:    "Force",
		Description: "SI unit of force, 1 N = 1 kg·m/s²",
	},
	"kilonewton": {
		Canonical:   "kilonewton",
		Symbol:      "kN",
		Aliases:     []string{"kilonewton", "kilonewtons"},
		System:      "SI",
		Quantity:    "Force",
		Description: "1000 newtons",
	},
	"dyne": {
		Canonical:   "dyne",
		Symbol:      "dyn",
		Aliases:     []string{"dyne", "dynes"},
		System:      "CGS",
		Quantity:    "Force",
		Description: "CGS unit of force, 1 dyne = 10⁻⁵ N",
	},
	"kilogram-force": {
		Canonical:   "kilogram-force",
		Symbol:      "kgf",
		Aliases:     []string{"kilogram-force", "kilopond", "kiloponds", "kgf"},
		System:      "SI",
		Quantity:    "Force",
		Description: "Force exerted by 1 kg under standard gravity, 9.80665 N",
	},
	"pound-force": {
		Canonical:   "pound-force",
		Symbol:      "lbf",
		Aliases:     []string{"pound-force", "pound-forces", "lbf"},
		System:      "Imperial",
		Quantity:    "Force",
		Description: "Force exerted by 1 avoirdupois pound under standard gravity, 4.448222 N",
	},
	"poundal": {
		Canonical:   "poundal",
		Symbol:      "pdl",
		Aliases:     []string{"poundal", "poundals", "pdl"},
		System:      "Imperial",
		Quantity:    "Force",
		Description: "Imperial absolute unit of force, 0.138255 N",
	},

	// ========== IMPULSE UNITS ==========

	"newton-second": {
		Canonical:   "newton-second",
		Symbol:      "N·s",
		Aliases:     []string{"newton-second", "newton-seconds"},
		System:      "SI",
		Quantity:    "Impulse",
		Description: "SI unit of impulse (force × time)",
	},
	"pound-force-second": {
		Canonical:   "pound-force-second",
		Symbol:      "lbf·s",
		Aliases:     []string{"pound-force-second", "pound-force-seconds"},
		System:      "Imperial",
		Quantity:    "Impulse",
		Description: "Imperial unit of impulse, 4.448222 N·s",
	},

	// ========== PRESSURE UNITS ==========

	"pascal": {
		Canonical:   "pascal",
		Symbol:      "Pa",
		Aliases:     []string{"pascal", "pascals", "pa"},
		System:      "SI",
		Quantity:    "Pressure",
		Description: "SI unit of pressure, 1 Pa = 1 N/m²",
	},
	"kilopascal": {
		Canonical:   "kilopascal",
		Symbol:      "kPa",
		Aliases:     []string{"kilopascal", "kilopascals", "kpa"},
		System:      "SI",
		Quantity:    "Pressure",
		Description: "1000 pascals",
	},
	"megapascal": {
		Canonical:   "megapascal",
		Symbol:      "MPa",
		Aliases:     []string{"megapascal", "megapascals", "mpa"},
		System:      "SI",
		Quantity:    "Pressure",
		Description: "1 million pascals",
	},
	"bar": {
		Canonical:   "bar",
		Symbol:      "bar",
		Aliases:     []string{"bar", "bars"},
		System:      "SI",
		Quantity:    "Pressure",
		Description: "100,000 pascals",
	},
	"millibar": {
		Canonical:   "millibar",
		Symbol:      "mbar",
		Aliases:     []string{"millibar", "millibars", "mbar"},
		System:      "SI",
		Quantity:    "Pressure",
		Description: "100 pascals, commonly used in meteorology",
	},
	"atmosphere": {
		Canonical:   "atmosphere",
		Symbol:      "atm",
		Aliases:     []string{"atmosphere", "atmospheres", "atm"},
		System:      "SI",
		Quantity:    "Pressure",
		Description: "Standard atmosphere, 101,325 pascals",
	},
	"torr": {
		Canonical:   "torr",
		Symbol:      "Torr",
		Aliases:     []string{"torr", "torrs"},
		System:      "SI",
		Quantity:    "Pressure",
		Description: "1/760 of an atmosphere, approximately 133.322 Pa",
	},
	"psi": {
		Canonical:   "psi",
		Symbol:      "psi",
		Aliases:     []string{"psi", "pounds per square inch"},
		System:      "Imperial",
		Quantity:    "Pressure",
		Description: "Pounds per square inch, approximately 6894.76 Pa",
	},
	"pounds per square inch": {
		Canonical:   "psi",
		Symbol:      "psi",
		Aliases:     []string{"psi", "pounds per square inch"},
		System:      "Imperial",
		Quantity:    "Pressure",
		Description: "Pounds per square inch, approximately 6894.76 Pa",
	},
	"inch of mercury": {
		Canonical:   "inch of mercury",
		Symbol:      "inHg",
		Aliases:     []string{"inch of mercury", "inches of mercury", "inhg"},
		System:      "Imperial",
		Quantity:    "Pressure",
		Description: "Commonly used in aviation and weather, approximately 3386.39 Pa",
	},
	"inches of mercury": {
		Canonical:   "inch of mercury",
		Symbol:      "inHg",
		Aliases:     []string{"inch of mercury", "inches of mercury", "inhg"},
		System:      "Imperial",
		Quantity:    "Pressure",
		Description: "Commonly used in aviation and weather, approximately 3386.39 Pa",
	},

	// ========== ACCELERATION UNITS ==========

	"m/s^2": {
		Canonical:   "m/s^2",
		Symbol:      "m/s²",
		Aliases:     []string{"m/s^2", "meters per second squared"},
		System:      "SI",
		Quantity:    "Acceleration",
		Description: "SI unit of acceleration",
	},
	"meters per second squared": {
		Canonical:   "m/s^2",
		Symbol:      "m/s²",
		Aliases:     []string{"m/s^2", "meters per second squared"},
		System:      "SI",
		Quantity:    "Acceleration",
		Description: "SI unit of acceleration",
	},
	"cm/s^2": {
		Canonical:   "cm/s^2",
		Symbol:      "cm/s²",
		Aliases:     []string{"cm/s^2", "centimeters per second squared"},
		System:      "CGS",
		Quantity:    "Acceleration",
		Description: "CGS unit of acceleration",
	},
	"centimeters per second squared": {
		Canonical:   "cm/s^2",
		Symbol:      "cm/s²",
		Aliases:     []string{"cm/s^2", "centimeters per second squared"},
		System:      "CGS",
		Quantity:    "Acceleration",
		Description: "CGS unit of acceleration",
	},
	"ft/s^2": {
		Canonical:   "ft/s^2",
		Symbol:      "ft/s²",
		Aliases:     []string{"ft/s^2", "feet per second squared"},
		System:      "Imperial",
		Quantity:    "Acceleration",
		Description: "Imperial unit of acceleration",
	},
	"feet per second squared": {
		Canonical:   "ft/s^2",
		Symbol:      "ft/s²",
		Aliases:     []string{"ft/s^2", "feet per second squared"},
		System:      "Imperial",
		Quantity:    "Acceleration",
		Description: "Imperial unit of acceleration",
	},
	"standard-gravity": {
		Canonical:   "standard-gravity",
		Symbol:      "g₀",
		Aliases:     []string{"standard-gravity", "standard gravity", "standard gravities"},
		System:      "SI",
		Quantity:    "Acceleration",
		Description: "Standard acceleration due to gravity, 9.80665 m/s²",
	},
	"standard gravity": {
		Canonical:   "standard-gravity",
		Symbol:      "g₀",
		Aliases:     []string{"standard-gravity", "standard gravity", "standard gravities"},
		System:      "SI",
		Quantity:    "Acceleration",
		Description: "Standard acceleration due to gravity, 9.80665 m/s²",
	},

	// ========== FREQUENCY UNITS ==========

	"hertz": {
		Canonical:   "hertz",
		Symbol:      "Hz",
		Aliases:     []string{"hertz", "hz"},
		System:      "SI",
		Quantity:    "Frequency",
		Description: "SI unit of frequency, 1 Hz = 1 cycle per second",
	},
	"kilohertz": {
		Canonical:   "kilohertz",
		Symbol:      "kHz",
		Aliases:     []string{"kilohertz", "khz"},
		System:      "SI",
		Quantity:    "Frequency",
		Description: "1000 hertz",
	},
	"megahertz": {
		Canonical:   "megahertz",
		Symbol:      "MHz",
		Aliases:     []string{"megahertz", "mhz"},
		System:      "SI",
		Quantity:    "Frequency",
		Description: "1 million hertz",
	},
	"gigahertz": {
		Canonical:   "gigahertz",
		Symbol:      "GHz",
		Aliases:     []string{"gigahertz", "ghz"},
		System:      "SI",
		Quantity:    "Frequency",
		Description: "1 billion hertz",
	},
	"terahertz": {
		Canonical:   "terahertz",
		Symbol:      "THz",
		Aliases:     []string{"terahertz", "thz"},
		System:      "SI",
		Quantity:    "Frequency",
		Description: "1 trillion hertz",
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

// GetUnitQuantity returns the quantity type for a unit name (e.g., "Length", "Mass", "Volume").
// Returns the quantity and true if found, empty string and false otherwise.
func GetUnitQuantity(unitName string) (string, bool) {
	canonical, found := NormalizeUnitName(unitName)
	if !found {
		return "", false
	}
	if unit, ok := StandardUnits[canonical]; ok {
		return unit.Quantity, true
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
