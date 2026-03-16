package units

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/shopspring/decimal"

	martinlindhe "github.com/martinlindhe/unit"
)

// CategoryDataSize is the category name for data size units (bytes, bits, etc.).
const CategoryDataSize = "DataSize"

// ConversionInfo contains metadata about a unit for conversion purposes.
type ConversionInfo struct {
	Category     string                // "Length", "Mass", etc. (matches canonical.go Quantity)
	ToBaseUnit   func(float64) float64 // Convert to base unit (e.g., feet -> meters)
	FromBaseUnit func(float64) float64 // Convert from base unit (e.g., meters -> feet)
}

// conversionRegistry maps lowercase unit names to conversion info.
// Built at package init.
var conversionRegistry map[string]ConversionInfo

func init() {
	conversionRegistry = buildConversionRegistry()
}

// Convert converts a value from one unit to another.
// Returns an error if units are unknown or incompatible.
func Convert(value decimal.Decimal, fromUnit, toUnit string) (decimal.Decimal, error) {
	if strings.EqualFold(fromUnit, toUnit) {
		return value, nil
	}

	sourceNorm := strings.ToLower(fromUnit)
	targetNorm := strings.ToLower(toUnit)

	sourceInfo, sourceOk := conversionRegistry[sourceNorm]
	targetInfo, targetOk := conversionRegistry[targetNorm]

	if !sourceOk || !targetOk {
		return decimal.Zero, fmt.Errorf("cannot convert %s to %s (unknown unit)", fromUnit, toUnit)
	}

	if sourceInfo.Category != targetInfo.Category {
		return decimal.Zero, fmt.Errorf("cannot convert %s to %s (different categories: %s vs %s)",
			fromUnit, toUnit, sourceInfo.Category, targetInfo.Category)
	}

	v, _ := value.Float64()
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return decimal.Zero, fmt.Errorf("cannot convert %s to %s (non-finite value)", fromUnit, toUnit)
	}
	baseValue := sourceInfo.ToBaseUnit(v)
	targetValue := targetInfo.FromBaseUnit(baseValue)
	if math.IsNaN(targetValue) || math.IsInf(targetValue, 0) {
		return decimal.Zero, fmt.Errorf("cannot convert %s to %s (conversion produced non-finite result)", fromUnit, toUnit)
	}
	return decimal.NewFromFloat(targetValue), nil
}

// CategoryForUnit returns the canonical category name for a unit.
// Returns CategoryCustom for units not in the standard library.
func CategoryForUnit(unitName string) string {
	info, ok := conversionRegistry[strings.ToLower(unitName)]
	if !ok {
		return CategoryCustom
	}
	return info.Category
}

// GetSystemForUnit returns the measurement system for a unit from canonical.go.
// Returns "" if the unit is not in StandardUnits.
func GetSystemForUnit(unitName string) string {
	canonical, found := NormalizeUnitName(unitName)
	if !found {
		return ""
	}
	if mapping, ok := StandardUnits[canonical]; ok {
		return mapping.System
	}
	return ""
}

// GetDefaultTargetUnit returns the preferred unit to convert to within a target system.
// For example, GetDefaultTargetUnit("grams", "imperial") returns "ounces".
// Returns "" if no suitable target exists.
func GetDefaultTargetUnit(unitName string, targetSystem string) string {
	category := CategoryForUnit(unitName)
	if category == "" {
		return ""
	}

	// Map target system request to canonical System values
	var matchSystems []string
	switch strings.ToLower(targetSystem) {
	case "imperial":
		matchSystems = []string{"US_Customary", "Imperial"}
	case "si":
		matchSystems = []string{"SI"}
	default:
		return ""
	}

	// Look for a default unit in the target system for this category.
	// Use the defaultTargetUnits table for deterministic selection.
	key := category + ":" + strings.ToLower(targetSystem)
	if target, ok := defaultTargetUnits[key]; ok {
		return target
	}

	// Fallback: scan StandardUnits for any unit in the target system and category
	for _, mapping := range StandardUnits {
		if mapping.Quantity != category {
			continue
		}
		if slices.Contains(matchSystems, mapping.System) {
			return mapping.Canonical
		}
	}

	return ""
}

// defaultTargetUnits maps "Category:system" to the preferred target unit.
// This ensures deterministic, natural conversions (e.g., grams -> ounces, not pounds).
var defaultTargetUnits = map[string]string{
	// SI targets (for convert_to: si)
	"Length:si":      "meter",
	"Mass:si":        "gram",
	"Volume:si":      "liter",
	"Temperature:si": "celsius",
	"Speed:si":       "km/h",
	"Energy:si":      "joule",
	"Power:si":       "watt",
	"Area:si":        "square meter",

	// New physical categories
	"Force:si":              "newton",
	"Force:imperial":        "pound-force",
	"Impulse:si":            "newton-second",
	"Impulse:imperial":      "pound-force-second",
	"Pressure:si":           "pascal",
	"Pressure:imperial":     "psi",
	"Acceleration:si":       "m/s^2",
	"Acceleration:imperial": "ft/s^2",
	"Frequency:si":          "hertz",
	// No Frequency:imperial — frequency is universal (hertz everywhere)

	// Imperial/US targets (for convert_to: imperial)
	"Length:imperial":      "foot",
	"Mass:imperial":        "ounce",
	"Volume:imperial":      "cup",
	"Temperature:imperial": "fahrenheit",
	"Speed:imperial":       "mph",
	"Power:imperial":       "horsepower",
	"Area:imperial":        "square foot",
}

// GetUnitInfo returns conversion info for a unit name (case-insensitive).
// This is the primary lookup for raw conversion functions (ToBaseUnit/FromBaseUnit).
func GetUnitInfo(unitName string) (ConversionInfo, bool) {
	info, ok := conversionRegistry[strings.ToLower(unitName)]
	return info, ok
}

// buildConversionRegistry creates the unit conversion registry.
// Uses the martinlindhe/unit library for all conversions.
func buildConversionRegistry() map[string]ConversionInfo {
	r := make(map[string]ConversionInfo)

	addLengthConversions(r)
	addMassConversions(r)
	addVolumeConversions(r)
	addTemperatureConversions(r)
	addSpeedConversions(r)
	addForceConversions(r)
	addImpulseConversions(r)
	addPressureConversions(r)
	addAccelerationConversions(r)
	addFrequencyConversions(r)
	addEnergyConversions(r)
	addPowerConversions(r)
	addAreaConversions(r)
	addDataSizeConversions(r)

	return r
}

// Helper to register a unit with all its aliases from canonical.go
func registerUnit(r map[string]ConversionInfo, info ConversionInfo, aliases ...string) {
	for _, alias := range aliases {
		r[strings.ToLower(alias)] = info
	}
}

func addLengthConversions(r map[string]ConversionInfo) {
	cat := "Length"

	// Meter (base)
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return v },
		func(v float64) float64 { return v },
	}, "m", "meter", "meters", "metre", "metres")

	// Kilometer
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return (martinlindhe.Length(v) * martinlindhe.Kilometer).Meters() },
		func(v float64) float64 { return (martinlindhe.Length(v) * martinlindhe.Meter).Kilometers() },
	}, "km", "kilometer", "kilometers", "kilometre", "kilometres")

	// Centimeter
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return (martinlindhe.Length(v) * martinlindhe.Centimeter).Meters() },
		func(v float64) float64 { return (martinlindhe.Length(v) * martinlindhe.Meter).Centimeters() },
	}, "cm", "centimeter", "centimeters", "centimetre", "centimetres")

	// Millimeter
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return (martinlindhe.Length(v) * martinlindhe.Millimeter).Meters() },
		func(v float64) float64 { return (martinlindhe.Length(v) * martinlindhe.Meter).Millimeters() },
	}, "mm", "millimeter", "millimeters", "millimetre", "millimetres")

	// Foot
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return (martinlindhe.Length(v) * martinlindhe.Foot).Meters() },
		func(v float64) float64 { return (martinlindhe.Length(v) * martinlindhe.Meter).Feet() },
	}, "ft", "foot", "feet")

	// Inch
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return (martinlindhe.Length(v) * martinlindhe.Inch).Meters() },
		func(v float64) float64 { return (martinlindhe.Length(v) * martinlindhe.Meter).Inches() },
	}, "in", "inch", "inches")

	// Yard
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return (martinlindhe.Length(v) * martinlindhe.Yard).Meters() },
		func(v float64) float64 { return (martinlindhe.Length(v) * martinlindhe.Meter).Yards() },
	}, "yd", "yard", "yards")

	// Mile
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return (martinlindhe.Length(v) * martinlindhe.Mile).Meters() },
		func(v float64) float64 { return (martinlindhe.Length(v) * martinlindhe.Meter).Miles() },
	}, "mi", "mile", "miles")

	// Nautical Mile
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return (martinlindhe.Length(v) * martinlindhe.NauticalMile).Meters() },
		func(v float64) float64 { return (martinlindhe.Length(v) * martinlindhe.Meter).NauticalMiles() },
	}, "nmi", "nautical mile", "nautical miles")
}

func addMassConversions(r map[string]ConversionInfo) {
	cat := "Mass"

	// Kilogram (base)
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return v },
		func(v float64) float64 { return v },
	}, "kg", "kilogram", "kilograms")

	// Gram
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return (martinlindhe.Mass(v) * martinlindhe.Gram).Kilograms() },
		func(v float64) float64 { return (martinlindhe.Mass(v) * martinlindhe.Kilogram).Grams() },
	}, "g", "gram", "grams")

	// Milligram
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return (martinlindhe.Mass(v) * martinlindhe.Milligram).Kilograms() },
		func(v float64) float64 { return (martinlindhe.Mass(v) * martinlindhe.Kilogram).Milligrams() },
	}, "mg", "milligram", "milligrams")

	// Metric Ton
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return (martinlindhe.Mass(v) * martinlindhe.Tonne).Kilograms() },
		func(v float64) float64 { return (martinlindhe.Mass(v) * martinlindhe.Kilogram).Tonnes() },
	}, "t", "tonne", "tonnes", "metric ton", "metric tons")

	// Pound
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return (martinlindhe.Mass(v) * martinlindhe.AvoirdupoisPound).Kilograms() },
		func(v float64) float64 { return (martinlindhe.Mass(v) * martinlindhe.Kilogram).AvoirdupoisPounds() },
	}, "lb", "lbs", "pound", "pounds")

	// Ounce (standard/avoirdupois — everyday weight: 1 oz = 28.35g)
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return (martinlindhe.Mass(v) * martinlindhe.AvoirdupoisOunce).Kilograms() },
		func(v float64) float64 { return (martinlindhe.Mass(v) * martinlindhe.Kilogram).AvoirdupoisOunces() },
	}, "oz", "ounce", "ounces", "us oz")

	// Troy Ounce (precious metals: 1 troy oz = 31.1035g)
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return (martinlindhe.Mass(v) * martinlindhe.TroyOunce).Kilograms() },
		func(v float64) float64 { return (martinlindhe.Mass(v) * martinlindhe.Kilogram).TroyOunces() },
	}, "troy oz", "troy ounce", "troy ounces")

	// Troy Pound (12 troy ounces, 373.24g)
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return (martinlindhe.Mass(v) * martinlindhe.TroyPound).Kilograms() },
		func(v float64) float64 { return (martinlindhe.Mass(v) * martinlindhe.Kilogram).TroyPounds() },
	}, "troy lb", "troy pound", "troy pounds")

	// US-qualified pound alias
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return (martinlindhe.Mass(v) * martinlindhe.AvoirdupoisPound).Kilograms() },
		func(v float64) float64 { return (martinlindhe.Mass(v) * martinlindhe.Kilogram).AvoirdupoisPounds() },
	}, "us lb")

	// Short Ton (US: 2000 lbs = 907.185 kg)
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 {
			return (martinlindhe.Mass(v) * martinlindhe.AvoirdupoisPound * 2000).Kilograms()
		},
		func(v float64) float64 {
			return (martinlindhe.Mass(v) * martinlindhe.Kilogram).AvoirdupoisPounds() / 2000
		},
	}, "short ton", "short tons")

	// Long Ton (Imperial: 2240 lbs = 1016.047 kg)
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 {
			return (martinlindhe.Mass(v) * martinlindhe.AvoirdupoisPound * 2240).Kilograms()
		},
		func(v float64) float64 {
			return (martinlindhe.Mass(v) * martinlindhe.Kilogram).AvoirdupoisPounds() / 2240
		},
	}, "long ton", "long tons")
}

func addVolumeConversions(r map[string]ConversionInfo) {
	cat := "Volume"

	// Liter (base)
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return v },
		func(v float64) float64 { return v },
	}, "l", "liter", "liters", "litre", "litres")

	// Milliliter
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return (martinlindhe.Volume(v) * martinlindhe.Milliliter).Liters() },
		func(v float64) float64 { return (martinlindhe.Volume(v) * martinlindhe.Liter).Milliliters() },
	}, "ml", "milliliter", "milliliters", "millilitre", "millilitres")

	// US Gallon
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return (martinlindhe.Volume(v) * martinlindhe.USLiquidGallon).Liters() },
		func(v float64) float64 { return (martinlindhe.Volume(v) * martinlindhe.Liter).USLiquidGallons() },
	}, "gal", "gallon", "gallons", "us gal")

	// US Fluid Ounce
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return (martinlindhe.Volume(v) * martinlindhe.USFluidOunce).Liters() },
		func(v float64) float64 { return (martinlindhe.Volume(v) * martinlindhe.Liter).USFluidOunces() },
	}, "fl oz", "fluid ounce", "fluid ounces", "us fl oz")

	// US Pint
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return (martinlindhe.Volume(v) * martinlindhe.USLiquidPint).Liters() },
		func(v float64) float64 { return (martinlindhe.Volume(v) * martinlindhe.Liter).USLiquidPints() },
	}, "pt", "pint", "pints", "us pt")

	// US Quart
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return (martinlindhe.Volume(v) * martinlindhe.USLiquidQuart).Liters() },
		func(v float64) float64 { return (martinlindhe.Volume(v) * martinlindhe.Liter).USLiquidQuarts() },
	}, "qt", "quart", "quarts", "us qt")

	// US Cup
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return (martinlindhe.Volume(v) * martinlindhe.USLegalCup).Liters() },
		func(v float64) float64 { return (martinlindhe.Volume(v) * martinlindhe.Liter).USLegalCups() },
	}, "cup", "cups", "us cup")

	// Imperial Gallon (4.54609 liters)
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return (martinlindhe.Volume(v) * martinlindhe.ImperialGallon).Liters() },
		func(v float64) float64 { return (martinlindhe.Volume(v) * martinlindhe.Liter).ImperialGallons() },
	}, "imp gal", "imperial gallon", "imperial gallons")

	// Imperial Quart
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return (martinlindhe.Volume(v) * martinlindhe.ImperialQuart).Liters() },
		func(v float64) float64 { return (martinlindhe.Volume(v) * martinlindhe.Liter).ImperialQuarts() },
	}, "imp qt", "imperial quart", "imperial quarts")

	// Imperial Pint
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return (martinlindhe.Volume(v) * martinlindhe.ImperialPint).Liters() },
		func(v float64) float64 { return (martinlindhe.Volume(v) * martinlindhe.Liter).ImperialPints() },
	}, "imp pt", "imperial pint", "imperial pints")

	// Imperial Cup (half an imperial pint)
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return (martinlindhe.Volume(v) * martinlindhe.ImperialCup).Liters() },
		func(v float64) float64 { return (martinlindhe.Volume(v) * martinlindhe.Liter).ImperialCups() },
	}, "imp cup", "imperial cup", "imperial cups")

	// Imperial Fluid Ounce (28.4131 mL)
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return (martinlindhe.Volume(v) * martinlindhe.ImperialFluidOunce).Liters() },
		func(v float64) float64 { return (martinlindhe.Volume(v) * martinlindhe.Liter).ImperialFluidOunces() },
	}, "imp fl oz", "imperial fluid ounce", "imperial fluid ounces")

	// US Tablespoon
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return (martinlindhe.Volume(v) * martinlindhe.USTableSpoon).Liters() },
		func(v float64) float64 { return (martinlindhe.Volume(v) * martinlindhe.Liter).USTableSpoons() },
	}, "tbsp", "tablespoon", "tablespoons")

	// US Teaspoon
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return (martinlindhe.Volume(v) * martinlindhe.USTeaSpoon).Liters() },
		func(v float64) float64 { return (martinlindhe.Volume(v) * martinlindhe.Liter).USTeaSpoons() },
	}, "tsp", "teaspoon", "teaspoons")
}

func addTemperatureConversions(r map[string]ConversionInfo) {
	cat := "Temperature"

	// Celsius (base)
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return v },
		func(v float64) float64 { return v },
	}, "c", "celsius", "\u00b0c", "degc")

	// Fahrenheit (affine, not linear)
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return martinlindhe.FromFahrenheit(v).Celsius() },
		func(v float64) float64 { return martinlindhe.FromCelsius(v).Fahrenheit() },
	}, "f", "fahrenheit", "\u00b0f", "degf")

	// Kelvin
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return martinlindhe.FromKelvin(v).Celsius() },
		func(v float64) float64 { return martinlindhe.FromCelsius(v).Kelvin() },
	}, "k", "kelvin")
}

func addSpeedConversions(r map[string]ConversionInfo) {
	cat := "Speed"

	// m/s (base)
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return v },
		func(v float64) float64 { return v },
	}, "m/s", "mps", "meters per second")

	// km/h
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 {
			return (martinlindhe.Speed(v) * martinlindhe.KilometersPerHour).MetersPerSecond()
		},
		func(v float64) float64 {
			return (martinlindhe.Speed(v) * martinlindhe.MetersPerSecond).KilometersPerHour()
		},
	}, "km/h", "kph", "kmh", "kilometers per hour")

	// mph
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 {
			return (martinlindhe.Speed(v) * martinlindhe.MilesPerHour).MetersPerSecond()
		},
		func(v float64) float64 {
			return (martinlindhe.Speed(v) * martinlindhe.MetersPerSecond).MilesPerHour()
		},
	}, "mph", "miles per hour")

	// Knots
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 {
			return (martinlindhe.Speed(v) * martinlindhe.Knot).MetersPerSecond()
		},
		func(v float64) float64 {
			return (martinlindhe.Speed(v) * martinlindhe.MetersPerSecond).Knots()
		},
	}, "knot", "knots")
}

func addEnergyConversions(r map[string]ConversionInfo) {
	cat := "Energy"

	// Joule (base)
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return v },
		func(v float64) float64 { return v },
	}, "j", "joule", "joules")

	// Kilojoule
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return (martinlindhe.Energy(v) * martinlindhe.Kilojoule).Joules() },
		func(v float64) float64 { return (martinlindhe.Energy(v) * martinlindhe.Joule).Kilojoules() },
	}, "kj", "kilojoule", "kilojoules")

	// Calorie
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return v * 4.184 },
		func(v float64) float64 { return v / 4.184 },
	}, "cal", "calorie", "calories")

	// Kilocalorie
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return v * 4184 },
		func(v float64) float64 { return v / 4184 },
	}, "kcal", "kilocalorie", "kilocalories")

	// kWh
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return (martinlindhe.Energy(v) * martinlindhe.KilowattHour).Joules() },
		func(v float64) float64 { return (martinlindhe.Energy(v) * martinlindhe.Joule).KilowattHours() },
	}, "kwh", "kilowatt-hour", "kilowatt-hours")
}

func addPowerConversions(r map[string]ConversionInfo) {
	cat := "Power"

	// Watt (base)
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return v },
		func(v float64) float64 { return v },
	}, "w", "watt", "watts")

	// Kilowatt
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return (martinlindhe.Power(v) * martinlindhe.Kilowatt).Watts() },
		func(v float64) float64 { return (martinlindhe.Power(v) * martinlindhe.Watt).Kilowatts() },
	}, "kw", "kilowatt", "kilowatts")

	// Megawatt
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return (martinlindhe.Power(v) * martinlindhe.Megawatt).Watts() },
		func(v float64) float64 { return (martinlindhe.Power(v) * martinlindhe.Watt).Megawatts() },
	}, "mw", "megawatt", "megawatts")

	// Horsepower
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return v * 745.7 },
		func(v float64) float64 { return v / 745.7 },
	}, "hp", "horsepower")
}

func addAreaConversions(r map[string]ConversionInfo) {
	cat := "Area"

	// Square meter (base)
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return v },
		func(v float64) float64 { return v },
	}, "m\u00b2", "m2", "sq m", "square meter", "square meters", "square metre", "square metres")

	// Square kilometer
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return v * 1_000_000 },
		func(v float64) float64 { return v / 1_000_000 },
	}, "km\u00b2", "km2", "sq km", "square kilometer", "square kilometers", "square kilometre", "square kilometres")

	// Square centimeter
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return v * 0.0001 },
		func(v float64) float64 { return v / 0.0001 },
	}, "cm\u00b2", "cm2", "sq cm")

	// Square foot
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return v * 0.09290304 },
		func(v float64) float64 { return v / 0.09290304 },
	}, "ft\u00b2", "ft2", "sq ft", "square foot", "square feet")

	// Square inch
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return v * 0.00064516 },
		func(v float64) float64 { return v / 0.00064516 },
	}, "in\u00b2", "in2", "sq in")

	// Square yard
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return v * 0.83612736 },
		func(v float64) float64 { return v / 0.83612736 },
	}, "yd\u00b2", "yd2", "sq yd")

	// Square mile
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return v * 2_589_988.110336 },
		func(v float64) float64 { return v / 2_589_988.110336 },
	}, "mi\u00b2", "mi2", "sq mi")

	// Acre
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return v * 4046.8564224 },
		func(v float64) float64 { return v / 4046.8564224 },
	}, "acre", "acres")

	// Hectare
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return v * 10000 },
		func(v float64) float64 { return v / 10000 },
	}, "ha", "hectare", "hectares")
}

func addDataSizeConversions(r map[string]ConversionInfo) {
	cat := CategoryDataSize

	// Bit (base unit for data size category)
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return float64(martinlindhe.Datasize(v) * martinlindhe.Bit) },
		func(v float64) float64 { return martinlindhe.Datasize(v).Bits() },
	}, "bit", "bits")

	// Byte (8 bits)
	byteInfo := ConversionInfo{cat,
		func(v float64) float64 { return float64(martinlindhe.Datasize(v) * martinlindhe.Byte) },
		func(v float64) float64 { return martinlindhe.Datasize(v).Bytes() },
	}
	registerUnit(r, byteInfo, "byte", "bytes", "b")

	// ========== BINARY PREFIXES (IEC) - 1024-based ==========

	kibInfo := ConversionInfo{cat,
		func(v float64) float64 { return float64(martinlindhe.Datasize(v) * martinlindhe.Kibibyte) },
		func(v float64) float64 { return martinlindhe.Datasize(v).Kibibytes() },
	}
	registerUnit(r, kibInfo, "kib", "kibibyte", "kibibytes",
		"kb", "kilobyte", "kilobytes") // KB aliases to binary

	mibInfo := ConversionInfo{cat,
		func(v float64) float64 { return float64(martinlindhe.Datasize(v) * martinlindhe.Mebibyte) },
		func(v float64) float64 { return martinlindhe.Datasize(v).Mebibytes() },
	}
	registerUnit(r, mibInfo, "mib", "mebibyte", "mebibytes",
		"mb", "megabyte", "megabytes") // MB aliases to binary

	gibInfo := ConversionInfo{cat,
		func(v float64) float64 { return float64(martinlindhe.Datasize(v) * martinlindhe.Gibibyte) },
		func(v float64) float64 { return martinlindhe.Datasize(v).Gibibytes() },
	}
	registerUnit(r, gibInfo, "gib", "gibibyte", "gibibytes",
		"gb", "gigabyte", "gigabytes") // GB aliases to binary

	tibInfo := ConversionInfo{cat,
		func(v float64) float64 { return float64(martinlindhe.Datasize(v) * martinlindhe.Tebibyte) },
		func(v float64) float64 { return martinlindhe.Datasize(v).Tebibytes() },
	}
	registerUnit(r, tibInfo, "tib", "tebibyte", "tebibytes",
		"tb", "terabyte", "terabytes") // TB aliases to binary

	pibInfo := ConversionInfo{cat,
		func(v float64) float64 { return float64(martinlindhe.Datasize(v) * martinlindhe.Pebibyte) },
		func(v float64) float64 { return martinlindhe.Datasize(v).Pebibytes() },
	}
	registerUnit(r, pibInfo, "pib", "pebibyte", "pebibytes",
		"pb", "petabyte", "petabytes") // PB aliases to binary

	eibInfo := ConversionInfo{cat,
		func(v float64) float64 { return float64(martinlindhe.Datasize(v) * martinlindhe.Exbibyte) },
		func(v float64) float64 { return martinlindhe.Datasize(v).Exbibytes() },
	}
	registerUnit(r, eibInfo, "eib", "exbibyte", "exbibytes",
		"eb", "exabyte", "exabytes") // EB aliases to binary

	// ========== BIT UNITS (NETWORK) - 1000-based ==========

	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return float64(martinlindhe.Datasize(v) * martinlindhe.Kilobit) },
		func(v float64) float64 { return martinlindhe.Datasize(v).Kilobits() },
	}, "kbit", "kilobit", "kilobits", "kbps")

	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return float64(martinlindhe.Datasize(v) * martinlindhe.Megabit) },
		func(v float64) float64 { return martinlindhe.Datasize(v).Megabits() },
	}, "mbit", "megabit", "megabits", "mbps")

	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return float64(martinlindhe.Datasize(v) * martinlindhe.Gigabit) },
		func(v float64) float64 { return martinlindhe.Datasize(v).Gigabits() },
	}, "gbit", "gigabit", "gigabits", "gbps")

	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return float64(martinlindhe.Datasize(v) * martinlindhe.Terabit) },
		func(v float64) float64 { return martinlindhe.Datasize(v).Terabits() },
	}, "tbit", "terabit", "terabits", "tbps")

	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return float64(martinlindhe.Datasize(v) * martinlindhe.Petabit) },
		func(v float64) float64 { return martinlindhe.Datasize(v).Petabits() },
	}, "pbit", "petabit", "petabits", "pbps")

	// ========== BINARY BIT UNITS (IEC) - 1024-based ==========

	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return float64(martinlindhe.Datasize(v) * martinlindhe.Kibibit) },
		func(v float64) float64 { return martinlindhe.Datasize(v).Kibibits() },
	}, "kibit", "kibibit", "kibibits")

	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return float64(martinlindhe.Datasize(v) * martinlindhe.Mebibit) },
		func(v float64) float64 { return martinlindhe.Datasize(v).Mebibits() },
	}, "mibit", "mebibit", "mebibits")

	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return float64(martinlindhe.Datasize(v) * martinlindhe.Gibibit) },
		func(v float64) float64 { return martinlindhe.Datasize(v).Gibibits() },
	}, "gibit", "gibibit", "gibibits")

	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return float64(martinlindhe.Datasize(v) * martinlindhe.Tebibit) },
		func(v float64) float64 { return martinlindhe.Datasize(v).Tebibits() },
	}, "tibit", "tebibit", "tebibits")
}

func addForceConversions(r map[string]ConversionInfo) {
	cat := "Force"

	// Newton (base)
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return v },
		func(v float64) float64 { return v },
	}, "newton", "newtons")

	// Kilonewton
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return v * 1000 },
		func(v float64) float64 { return v / 1000 },
	}, "kilonewton", "kilonewtons")

	// Dyne
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 {
			return (martinlindhe.Force(v) * martinlindhe.Dyne).Newtons()
		},
		func(v float64) float64 {
			return (martinlindhe.Force(v) * martinlindhe.Newton).Dynes()
		},
	}, "dyne", "dynes")

	// Kilogram-force (kilopond)
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 {
			return (martinlindhe.Force(v) * martinlindhe.KilogramForce).Newtons()
		},
		func(v float64) float64 {
			return (martinlindhe.Force(v) * martinlindhe.Newton).KilogramForce()
		},
	}, "kilogram-force", "kilopond", "kiloponds", "kgf")

	// Pound-force
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 {
			return (martinlindhe.Force(v) * martinlindhe.PoundForce).Newtons()
		},
		func(v float64) float64 {
			return (martinlindhe.Force(v) * martinlindhe.Newton).PoundForce()
		},
	}, "pound-force", "pound-forces", "lbf")

	// Poundal
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 {
			return (martinlindhe.Force(v) * martinlindhe.Poundal).Newtons()
		},
		func(v float64) float64 {
			return (martinlindhe.Force(v) * martinlindhe.Newton).Poundals()
		},
	}, "poundal", "poundals", "pdl")
}

func addImpulseConversions(r map[string]ConversionInfo) {
	cat := "Impulse"

	// Newton-second (base) — no martinlindhe Impulse type, manual conversion
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return v },
		func(v float64) float64 { return v },
	}, "newton-second", "newton-seconds")

	// Pound-force-second (1 lbf·s = 4.448222 N·s, same factor as pound-force to newton)
	lbfToNewton := (martinlindhe.Force(1) * martinlindhe.PoundForce).Newtons()
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return v * lbfToNewton },
		func(v float64) float64 { return v / lbfToNewton },
	}, "pound-force-second", "pound-force-seconds")
}

func addPressureConversions(r map[string]ConversionInfo) {
	cat := "Pressure"

	// Pascal (base)
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return v },
		func(v float64) float64 { return v },
	}, "pascal", "pascals", "pa")

	// Kilopascal
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 {
			return (martinlindhe.Pressure(v) * martinlindhe.Kilopascal).Pascals()
		},
		func(v float64) float64 {
			return (martinlindhe.Pressure(v) * martinlindhe.Pascal).Kilopascals()
		},
	}, "kilopascal", "kilopascals", "kpa")

	// Megapascal
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 {
			return (martinlindhe.Pressure(v) * martinlindhe.Megapascal).Pascals()
		},
		func(v float64) float64 {
			return (martinlindhe.Pressure(v) * martinlindhe.Pascal).Megapascals()
		},
	}, "megapascal", "megapascals", "mpa")

	// Bar
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 {
			return (martinlindhe.Pressure(v) * martinlindhe.Bar).Pascals()
		},
		func(v float64) float64 {
			return (martinlindhe.Pressure(v) * martinlindhe.Pascal).Bars()
		},
	}, "bar", "bars")

	// Millibar
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 {
			return (martinlindhe.Pressure(v) * martinlindhe.Millibar).Pascals()
		},
		func(v float64) float64 {
			return (martinlindhe.Pressure(v) * martinlindhe.Pascal).Millibars()
		},
	}, "millibar", "millibars", "mbar")

	// Atmosphere
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 {
			return (martinlindhe.Pressure(v) * martinlindhe.Atmosphere).Pascals()
		},
		func(v float64) float64 {
			return (martinlindhe.Pressure(v) * martinlindhe.Pascal).Atmospheres()
		},
	}, "atmosphere", "atmospheres", "atm")

	// Torr
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 {
			return (martinlindhe.Pressure(v) * martinlindhe.Torr).Pascals()
		},
		func(v float64) float64 {
			return (martinlindhe.Pressure(v) * martinlindhe.Pascal).Torrs()
		},
	}, "torr", "torrs")

	// PSI (Pounds per Square Inch)
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 {
			return (martinlindhe.Pressure(v) * martinlindhe.PoundsPerSquareInch).Pascals()
		},
		func(v float64) float64 {
			return (martinlindhe.Pressure(v) * martinlindhe.Pascal).PoundsPerSquareInch()
		},
	}, "psi", "pounds per square inch")

	// Inch of Mercury
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 {
			return (martinlindhe.Pressure(v) * martinlindhe.InchOfMercury).Pascals()
		},
		func(v float64) float64 {
			return (martinlindhe.Pressure(v) * martinlindhe.Pascal).InchOfMercury()
		},
	}, "inch of mercury", "inches of mercury", "inhg")
}

func addAccelerationConversions(r map[string]ConversionInfo) {
	cat := "Acceleration"

	// m/s^2 (base)
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return v },
		func(v float64) float64 { return v },
	}, "m/s^2", "meters per second squared")

	// cm/s^2
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 {
			return (martinlindhe.Acceleration(v) * martinlindhe.CentimeterPerSecondSquared).MetersPerSecondSquared()
		},
		func(v float64) float64 {
			return (martinlindhe.Acceleration(v) * martinlindhe.MeterPerSecondSquared).CentimetersPerSecondSquared()
		},
	}, "cm/s^2", "centimeters per second squared")

	// ft/s^2
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 {
			return (martinlindhe.Acceleration(v) * martinlindhe.FootPerSecondSquared).MetersPerSecondSquared()
		},
		func(v float64) float64 {
			return (martinlindhe.Acceleration(v) * martinlindhe.MeterPerSecondSquared).FeetPerSecondSquared()
		},
	}, "ft/s^2", "feet per second squared")

	// Standard gravity
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 {
			return (martinlindhe.Acceleration(v) * martinlindhe.StandardGravity).MetersPerSecondSquared()
		},
		func(v float64) float64 {
			return (martinlindhe.Acceleration(v) * martinlindhe.MeterPerSecondSquared).StandardGravity()
		},
	}, "standard-gravity", "standard gravity", "standard gravities")
}

func addFrequencyConversions(r map[string]ConversionInfo) {
	cat := "Frequency"

	// Hertz (base)
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 { return v },
		func(v float64) float64 { return v },
	}, "hertz", "hz")

	// Kilohertz
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 {
			return (martinlindhe.Frequency(v) * martinlindhe.Kilohertz).Hertz()
		},
		func(v float64) float64 {
			return (martinlindhe.Frequency(v) * martinlindhe.Hertz).Kilohertz()
		},
	}, "kilohertz", "khz")

	// Megahertz
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 {
			return (martinlindhe.Frequency(v) * martinlindhe.Megahertz).Hertz()
		},
		func(v float64) float64 {
			return (martinlindhe.Frequency(v) * martinlindhe.Hertz).Megahertz()
		},
	}, "megahertz", "mhz")

	// Gigahertz
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 {
			return (martinlindhe.Frequency(v) * martinlindhe.Gigahertz).Hertz()
		},
		func(v float64) float64 {
			return (martinlindhe.Frequency(v) * martinlindhe.Hertz).Gigahertz()
		},
	}, "gigahertz", "ghz")

	// Terahertz
	registerUnit(r, ConversionInfo{cat,
		func(v float64) float64 {
			return (martinlindhe.Frequency(v) * martinlindhe.Terahertz).Hertz()
		},
		func(v float64) float64 {
			return (martinlindhe.Frequency(v) * martinlindhe.Hertz).Terahertz()
		},
	}, "terahertz", "thz")
}
