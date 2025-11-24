package interpreter

import (
	"strings"

	units "github.com/martinlindhe/unit"
)

// QuantityCategory represents groups of compatible units
type QuantityCategory string

const (
	CategoryLength      QuantityCategory = "length"
	CategoryMass        QuantityCategory = "mass"
	CategoryVolume      QuantityCategory = "volume"
	CategoryTemperature QuantityCategory = "temperature"
	// Speed, Energy, Power deferred to Phase 2+ (rate-based units)
	CategoryUnknown QuantityCategory = "unknown"
)

// UnitInfo contains metadata about a known unit
type UnitInfo struct {
	Category     QuantityCategory
	ToBaseUnit   func(float64) float64 // Convert to base unit (e.g., feet -> meters)
	FromBaseUnit func(float64) float64 // Convert from base unit (e.g., meters -> feet)
}

// unitRegistry maps unit names (lowercase) to conversion info
// Built at package init, no runtime reflection
var unitRegistry map[string]UnitInfo

func init() {
	unitRegistry = buildUnitRegistry()
}

// buildUnitRegistry creates the static registry
// Performance: All lookups O(1) map access, no reflection, direct function pointers
func buildUnitRegistry() map[string]UnitInfo {
	registry := make(map[string]UnitInfo)

	// Add all unit categories
	addLengthUnits(registry)
	addMassUnits(registry)
	addVolumeUnits(registry)
	// Temperature handled separately (offset-based)
	// Speed, Energy, Power deferred to Phase 2+ (rate-based units with accumulate functions)

	return registry
}

// addLengthUnits adds length unit conversions
// Base unit: meter
func addLengthUnits(registry map[string]UnitInfo) {
	// Helper to create length unit info
	makeLengthUnit := func(toMeters, fromMeters func(float64) float64) UnitInfo {
		return UnitInfo{
			Category:     CategoryLength,
			ToBaseUnit:   toMeters,
			FromBaseUnit: fromMeters,
		}
	}

	// Meter (base)
	registry["m"] = makeLengthUnit(
		func(v float64) float64 { return v },
		func(v float64) float64 { return v },
	)
	registry["meter"] = registry["m"]
	registry["meters"] = registry["m"]
	registry["metre"] = registry["m"]
	registry["metres"] = registry["m"]

	// Kilometer
	registry["km"] = makeLengthUnit(
		func(v float64) float64 { return (units.Length(v) * units.Kilometer).Meters() },
		func(v float64) float64 { return (units.Length(v) * units.Meter).Kilometers() },
	)
	registry["kilometer"] = registry["km"]
	registry["kilometers"] = registry["km"]
	registry["kilometre"] = registry["km"]
	registry["kilometres"] = registry["km"]

	// Centimeter
	registry["cm"] = makeLengthUnit(
		func(v float64) float64 { return (units.Length(v) * units.Centimeter).Meters() },
		func(v float64) float64 { return (units.Length(v) * units.Meter).Centimeters() },
	)
	registry["centimeter"] = registry["cm"]
	registry["centimeters"] = registry["cm"]
	registry["centimetre"] = registry["cm"]
	registry["centimetres"] = registry["cm"]

	// Millimeter
	registry["mm"] = makeLengthUnit(
		func(v float64) float64 { return (units.Length(v) * units.Millimeter).Meters() },
		func(v float64) float64 { return (units.Length(v) * units.Meter).Millimeters() },
	)
	registry["millimeter"] = registry["mm"]
	registry["millimeters"] = registry["mm"]
	registry["millimetre"] = registry["mm"]
	registry["millimetres"] = registry["mm"]

	// Foot
	registry["ft"] = makeLengthUnit(
		func(v float64) float64 { return (units.Length(v) * units.Foot).Meters() },
		func(v float64) float64 { return (units.Length(v) * units.Meter).Feet() },
	)
	registry["foot"] = registry["ft"]
	registry["feet"] = registry["ft"]

	// Inch
	registry["in"] = makeLengthUnit(
		func(v float64) float64 { return (units.Length(v) * units.Inch).Meters() },
		func(v float64) float64 { return (units.Length(v) * units.Meter).Inches() },
	)
	registry["inch"] = registry["in"]
	registry["inches"] = registry["in"]

	// Yard
	registry["yd"] = makeLengthUnit(
		func(v float64) float64 { return (units.Length(v) * units.Yard).Meters() },
		func(v float64) float64 { return (units.Length(v) * units.Meter).Yards() },
	)
	registry["yard"] = registry["yd"]
	registry["yards"] = registry["yd"]

	// Mile
	registry["mi"] = makeLengthUnit(
		func(v float64) float64 { return (units.Length(v) * units.Mile).Meters() },
		func(v float64) float64 { return (units.Length(v) * units.Meter).Miles() },
	)
	registry["mile"] = registry["mi"]
	registry["miles"] = registry["mi"]

	// Nautical Mile
	registry["nmi"] = makeLengthUnit(
		func(v float64) float64 { return (units.Length(v) * units.NauticalMile).Meters() },
		func(v float64) float64 { return (units.Length(v) * units.Meter).NauticalMiles() },
	)
	registry["nautical mile"] = registry["nmi"]
	registry["nautical miles"] = registry["nmi"]
}

// addMassUnits adds mass unit conversions
// Base unit: kilogram
func addMassUnits(registry map[string]UnitInfo) {
	makeMassUnit := func(toKg, fromKg func(float64) float64) UnitInfo {
		return UnitInfo{
			Category:     CategoryMass,
			ToBaseUnit:   toKg,
			FromBaseUnit: fromKg,
		}
	}

	// Kilogram (base)
	registry["kg"] = makeMassUnit(
		func(v float64) float64 { return v },
		func(v float64) float64 { return v },
	)
	registry["kilogram"] = registry["kg"]
	registry["kilograms"] = registry["kg"]

	// Gram
	registry["g"] = makeMassUnit(
		func(v float64) float64 { return (units.Mass(v) * units.Gram).Kilograms() },
		func(v float64) float64 { return (units.Mass(v) * units.Kilogram).Grams() },
	)
	registry["gram"] = registry["g"]
	registry["grams"] = registry["g"]

	// Milligram
	registry["mg"] = makeMassUnit(
		func(v float64) float64 { return (units.Mass(v) * units.Milligram).Kilograms() },
		func(v float64) float64 { return (units.Mass(v) * units.Kilogram).Milligrams() },
	)
	registry["milligram"] = registry["mg"]
	registry["milligrams"] = registry["mg"]

	// Metric Ton (Tonne)
	registry["t"] = makeMassUnit(
		func(v float64) float64 { return (units.Mass(v) * units.Tonne).Kilograms() },
		func(v float64) float64 { return (units.Mass(v) * units.Kilogram).Tonnes() },
	)
	registry["tonne"] = registry["t"]
	registry["tonnes"] = registry["t"]
	registry["metric ton"] = registry["t"]
	registry["metric tons"] = registry["t"]

	// Pound
	registry["lb"] = makeMassUnit(
		func(v float64) float64 { return (units.Mass(v) * units.AvoirdupoisPound).Kilograms() },
		func(v float64) float64 { return (units.Mass(v) * units.Kilogram).AvoirdupoisPounds() },
	)
	registry["lbs"] = registry["lb"]
	registry["pound"] = registry["lb"]
	registry["pounds"] = registry["lb"]

	// Ounce
	registry["oz"] = makeMassUnit(
		func(v float64) float64 { return (units.Mass(v) * units.AvoirdupoisOunce).Kilograms() },
		func(v float64) float64 { return (units.Mass(v) * units.Kilogram).AvoirdupoisOunces() },
	)
	registry["ounce"] = registry["oz"]
	registry["ounces"] = registry["oz"]
}

// addVolumeUnits adds volume unit conversions
// Base unit: liter
func addVolumeUnits(registry map[string]UnitInfo) {
	makeVolumeUnit := func(toLiters, fromLiters func(float64) float64) UnitInfo {
		return UnitInfo{
			Category:     CategoryVolume,
			ToBaseUnit:   toLiters,
			FromBaseUnit: fromLiters,
		}
	}

	// Liter (base)
	registry["l"] = makeVolumeUnit(
		func(v float64) float64 { return v },
		func(v float64) float64 { return v },
	)
	registry["liter"] = registry["l"]
	registry["liters"] = registry["l"]
	registry["litre"] = registry["l"]
	registry["litres"] = registry["l"]

	// Milliliter
	registry["ml"] = makeVolumeUnit(
		func(v float64) float64 { return (units.Volume(v) * units.Milliliter).Liters() },
		func(v float64) float64 { return (units.Volume(v) * units.Liter).Milliliters() },
	)
	registry["milliliter"] = registry["ml"]
	registry["milliliters"] = registry["ml"]
	registry["millilitre"] = registry["ml"]
	registry["millilitres"] = registry["ml"]

	// US Gallon
	registry["gal"] = makeVolumeUnit(
		func(v float64) float64 { return (units.Volume(v) * units.USLiquidGallon).Liters() },
		func(v float64) float64 { return (units.Volume(v) * units.Liter).USLiquidGallons() },
	)
	registry["gallon"] = registry["gal"]
	registry["gallons"] = registry["gal"]

	// US Pint
	registry["pt"] = makeVolumeUnit(
		func(v float64) float64 { return (units.Volume(v) * units.USLiquidPint).Liters() },
		func(v float64) float64 { return (units.Volume(v) * units.Liter).USLiquidPints() },
	)
	registry["pint"] = registry["pt"]
	registry["pints"] = registry["pt"]

	// US Quart
	registry["qt"] = makeVolumeUnit(
		func(v float64) float64 { return (units.Volume(v) * units.USLiquidQuart).Liters() },
		func(v float64) float64 { return (units.Volume(v) * units.Liter).USLiquidQuarts() },
	)
	registry["quart"] = registry["qt"]
	registry["quarts"] = registry["qt"]

	// US Cup
	registry["cup"] = makeVolumeUnit(
		func(v float64) float64 { return (units.Volume(v) * units.USLegalCup).Liters() },
		func(v float64) float64 { return (units.Volume(v) * units.Liter).USLegalCups() },
	)
	registry["cups"] = registry["cup"]

	// US Tablespoon
	registry["tbsp"] = makeVolumeUnit(
		func(v float64) float64 { return (units.Volume(v) * units.USTableSpoon).Liters() },
		func(v float64) float64 { return (units.Volume(v) * units.Liter).USTableSpoons() },
	)
	registry["tablespoon"] = registry["tbsp"]
	registry["tablespoons"] = registry["tbsp"]

	// US Teaspoon
	registry["tsp"] = makeVolumeUnit(
		func(v float64) float64 { return (units.Volume(v) * units.USTeaSpoon).Liters() },
		func(v float64) float64 { return (units.Volume(v) * units.Liter).USTeaSpoons() },
	)
	registry["teaspoon"] = registry["tsp"]
	registry["teaspoons"] = registry["tsp"]
}

// NOTE: Speed, Energy, and Power units are deferred to Phase 2+
// These will be integrated with the "accumulate" and rate conversion functions
// See ARCHITECURE_FUNCTIONS.md for the comprehensive design of rate-based units
// Examples: 100 KB/hour, 5 GB/day, $0.10/hour, etc.

// GetUnitInfo returns conversion info for a unit name (case-insensitive)
func GetUnitInfo(unitName string) (UnitInfo, bool) {
	info, ok := unitRegistry[strings.ToLower(unitName)]
	return info, ok
}

// IsKnownUnit checks if a unit is in the registry
func IsKnownUnit(unitName string) bool {
	_, ok := unitRegistry[strings.ToLower(unitName)]
	return ok
}

// GetCategory returns the category for a known unit
func GetCategory(unitName string) QuantityCategory {
	if info, ok := unitRegistry[strings.ToLower(unitName)]; ok {
		return info.Category
	}
	return CategoryUnknown
}
