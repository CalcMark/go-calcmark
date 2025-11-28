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
	CategorySpeed       QuantityCategory = "speed"
	CategoryEnergy      QuantityCategory = "energy"
	CategoryPower       QuantityCategory = "power"
	CategoryArea        QuantityCategory = "area"
	CategoryDataSize    QuantityCategory = "datasize"
	CategoryUnknown     QuantityCategory = "unknown"
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
	addTemperatureUnits(registry)
	addSpeedUnits(registry)
	addEnergyUnits(registry)
	addPowerUnits(registry)
	addAreaUnits(registry)
	addDataSizeUnits(registry)

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

// addTemperatureUnits adds temperature unit conversions
// NOTE: Temperature uses OFFSET-BASED conversion (32°F = 0°C, not 0°F)
// Base unit: celsius
func addTemperatureUnits(registry map[string]UnitInfo) {
	makeTemperatureUnit := func(toCelsius, fromCelsius func(float64) float64) UnitInfo {
		return UnitInfo{
			Category:     CategoryTemperature,
			ToBaseUnit:   toCelsius,
			FromBaseUnit: fromCelsius,
		}
	}

	// Celsius (base)
	registry["c"] = makeTemperatureUnit(
		func(v float64) float64 { return v },
		func(v float64) float64 { return v },
	)
	registry["celsius"] = registry["c"]
	registry["°c"] = registry["c"]
	registry["degc"] = registry["c"]

	// Fahrenheit
	registry["f"] = makeTemperatureUnit(
		func(v float64) float64 { return units.FromFahrenheit(v).Celsius() },
		func(v float64) float64 { return units.FromCelsius(v).Fahrenheit() },
	)
	registry["fahrenheit"] = registry["f"]
	registry["°f"] = registry["f"]
	registry["degf"] = registry["f"]

	// Kelvin
	registry["k"] = makeTemperatureUnit(
		func(v float64) float64 { return units.FromKelvin(v).Celsius() },
		func(v float64) float64 { return units.FromCelsius(v).Kelvin() },
	)
	registry["kelvin"] = registry["k"]
}

// addSpeedUnits adds speed unit conversions
// Base unit: meters_per_second (m/s)
func addSpeedUnits(registry map[string]UnitInfo) {
	makeSpeedUnit := func(toMps, fromMps func(float64) float64) UnitInfo {
		return UnitInfo{
			Category:     CategorySpeed,
			ToBaseUnit:   toMps,
			FromBaseUnit: fromMps,
		}
	}

	// Meters per second (base)
	registry["m/s"] = makeSpeedUnit(
		func(v float64) float64 { return v },
		func(v float64) float64 { return v },
	)
	registry["mps"] = registry["m/s"]
	registry["meters per second"] = registry["m/s"]

	// Kilometers per hour - 1 km/h = 0.277778 m/s
	registry["km/h"] = makeSpeedUnit(
		func(v float64) float64 { return v * 0.277778 },
		func(v float64) float64 { return v / 0.277778 },
	)
	registry["kph"] = registry["km/h"]
	registry["kmh"] = registry["km/h"]
	registry["kilometers per hour"] = registry["km/h"]

	// Miles per hour - 1 mph = 0.44704 m/s
	registry["mph"] = makeSpeedUnit(
		func(v float64) float64 { return v * 0.44704 },
		func(v float64) float64 { return v / 0.44704 },
	)
	registry["miles per hour"] = registry["mph"]

	// Knots - 1 knot = 0.514444 m/s
	registry["knot"] = makeSpeedUnit(
		func(v float64) float64 { return v * 0.514444 },
		func(v float64) float64 { return v / 0.514444 },
	)
	registry["knots"] = registry["knot"]
}

// addEnergyUnits adds energy unit conversions
// Base unit: joule (J)
func addEnergyUnits(registry map[string]UnitInfo) {
	makeEnergyUnit := func(toJoules, fromJoules func(float64) float64) UnitInfo {
		return UnitInfo{
			Category:     CategoryEnergy,
			ToBaseUnit:   toJoules,
			FromBaseUnit: fromJoules,
		}
	}

	// Joule (base)
	registry["j"] = makeEnergyUnit(
		func(v float64) float64 { return v },
		func(v float64) float64 { return v },
	)
	registry["joule"] = registry["j"]
	registry["joules"] = registry["j"]

	// Kilojoule
	registry["kj"] = makeEnergyUnit(
		func(v float64) float64 { return (units.Energy(v) * units.Kilojoule).Joules() },
		func(v float64) float64 { return (units.Energy(v) * units.Joule).Kilojoules() },
	)
	registry["kilojoule"] = registry["kj"]
	registry["kilojoules"] = registry["kj"]

	// Calorie (thermochemical) - 1 cal = 4.184 J
	registry["cal"] = makeEnergyUnit(
		func(v float64) float64 { return v * 4.184 },
		func(v float64) float64 { return v / 4.184 },
	)
	registry["calorie"] = registry["cal"]
	registry["calories"] = registry["cal"]

	// Kilocalorie (food Calorie) - 1 kcal = 4184 J
	registry["kcal"] = makeEnergyUnit(
		func(v float64) float64 { return v * 4184 },
		func(v float64) float64 { return v / 4184 },
	)
	registry["kilocalorie"] = registry["kcal"]
	registry["kilocalories"] = registry["kcal"]

	// Kilowatt-hour
	registry["kwh"] = makeEnergyUnit(
		func(v float64) float64 { return (units.Energy(v) * units.KilowattHour).Joules() },
		func(v float64) float64 { return (units.Energy(v) * units.Joule).KilowattHours() },
	)
	registry["kilowatt-hour"] = registry["kwh"]
	registry["kilowatt-hours"] = registry["kwh"]
}

// addPowerUnits adds power unit conversions
// Base unit: watt (W)
func addPowerUnits(registry map[string]UnitInfo) {
	makePowerUnit := func(toWatts, fromWatts func(float64) float64) UnitInfo {
		return UnitInfo{
			Category:     CategoryPower,
			ToBaseUnit:   toWatts,
			FromBaseUnit: fromWatts,
		}
	}

	// Watt (base)
	registry["w"] = makePowerUnit(
		func(v float64) float64 { return v },
		func(v float64) float64 { return v },
	)
	registry["watt"] = registry["w"]
	registry["watts"] = registry["w"]

	// Kilowatt
	registry["kw"] = makePowerUnit(
		func(v float64) float64 { return (units.Power(v) * units.Kilowatt).Watts() },
		func(v float64) float64 { return (units.Power(v) * units.Watt).Kilowatts() },
	)
	registry["kilowatt"] = registry["kw"]
	registry["kilowatts"] = registry["kw"]

	// Megawatt
	registry["mw"] = makePowerUnit(
		func(v float64) float64 { return (units.Power(v) * units.Megawatt).Watts() },
		func(v float64) float64 { return (units.Power(v) * units.Watt).Megawatts() },
	)
	registry["megawatt"] = registry["mw"]
	registry["megawatts"] = registry["mw"]

	// Horsepower (mechanical) - 1 hp = 745.7 W
	registry["hp"] = makePowerUnit(
		func(v float64) float64 { return v * 745.7 },
		func(v float64) float64 { return v / 745.7 },
	)
	registry["horsepower"] = registry["hp"]
}

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

// addAreaUnits adds area unit conversions
// Base unit: square meter (m²)
func addAreaUnits(registry map[string]UnitInfo) {
	makeAreaUnit := func(toSqMeters, fromSqMeters func(float64) float64) UnitInfo {
		return UnitInfo{
			Category:     CategoryArea,
			ToBaseUnit:   toSqMeters,
			FromBaseUnit: fromSqMeters,
		}
	}

	// Square meter (base)
	registry["m²"] = makeAreaUnit(
		func(v float64) float64 { return v },
		func(v float64) float64 { return v },
	)
	registry["m2"] = registry["m²"]
	registry["sq m"] = registry["m²"]
	registry["square meter"] = registry["m²"]
	registry["square meters"] = registry["m²"]
	registry["square metre"] = registry["m²"]  // British
	registry["square metres"] = registry["m²"] // British

	// Square kilometer
	registry["km²"] = makeAreaUnit(
		func(v float64) float64 { return v * 1_000_000 }, // 1 km² = 1,000,000 m²
		func(v float64) float64 { return v / 1_000_000 },
	)
	registry["km2"] = registry["km²"]
	registry["sq km"] = registry["km²"]
	registry["square kilometer"] = registry["km²"]
	registry["square kilometers"] = registry["km²"]
	registry["square kilometre"] = registry["km²"]  // British
	registry["square kilometres"] = registry["km²"] // British

	// Square centimeter
	registry["cm²"] = makeAreaUnit(
		func(v float64) float64 { return v * 0.0001 }, // 1 cm² = 0.0001 m²
		func(v float64) float64 { return v / 0.0001 },
	)
	registry["cm2"] = registry["cm²"]
	registry["sq cm"] = registry["cm²"]

	// Square foot
	registry["ft²"] = makeAreaUnit(
		func(v float64) float64 { return v * 0.09290304 }, // 1 ft² = 0.09290304 m²
		func(v float64) float64 { return v / 0.09290304 },
	)
	registry["ft2"] = registry["ft²"]
	registry["sq ft"] = registry["ft²"]
	registry["square foot"] = registry["ft²"]
	registry["square feet"] = registry["ft²"]

	// Square inch
	registry["in²"] = makeAreaUnit(
		func(v float64) float64 { return v * 0.00064516 }, // 1 in² = 0.00064516 m²
		func(v float64) float64 { return v / 0.00064516 },
	)
	registry["in2"] = registry["in²"]
	registry["sq in"] = registry["in²"]

	// Square yard
	registry["yd²"] = makeAreaUnit(
		func(v float64) float64 { return v * 0.83612736 }, // 1 yd² = 0.83612736 m²
		func(v float64) float64 { return v / 0.83612736 },
	)
	registry["yd2"] = registry["yd²"]
	registry["sq yd"] = registry["yd²"]

	// Square mile
	registry["mi²"] = makeAreaUnit(
		func(v float64) float64 { return v * 2_589_988.110336 }, // 1 mi² = 2,589,988.110336 m²
		func(v float64) float64 { return v / 2_589_988.110336 },
	)
	registry["mi2"] = registry["mi²"]
	registry["sq mi"] = registry["mi²"]

	// Acre
	registry["acre"] = makeAreaUnit(
		func(v float64) float64 { return v * 4046.8564224 }, // 1 acre = 4,046.8564224 m²
		func(v float64) float64 { return v / 4046.8564224 },
	)
	registry["acres"] = registry["acre"]

	// Hectare
	registry["ha"] = makeAreaUnit(
		func(v float64) float64 { return v * 10000 }, // 1 ha = 10,000 m²
		func(v float64) float64 { return v / 10000 },
	)
	registry["hectare"] = registry["ha"]
	registry["hectares"] = registry["ha"]
}

// addDataSizeUnits adds data size unit conversions using martinlindhe/unit library.
// Base unit: bit (to allow conversion between bytes and bits)
//
// Supports three conventions:
// 1. Binary (IEC): KiB, MiB, GiB - 1024-based, traditional computing
// 2. Decimal (SI): KB, MB, GB - 1000-based, storage marketing & some contexts
// 3. Bits: Kbit, Mbit, Gbit, Mbps, Gbps - 1000-based, networking
//
// Note: KB/MB/GB use binary (1024) by default as this matches user expectations
// in most computing contexts. Use KiB/MiB/GiB or explicit decimal units if needed.
func addDataSizeUnits(registry map[string]UnitInfo) {
	makeDataSizeUnit := func(toBits, fromBits func(float64) float64) UnitInfo {
		return UnitInfo{
			Category:     CategoryDataSize,
			ToBaseUnit:   toBits,
			FromBaseUnit: fromBits,
		}
	}

	// ========== BASE UNITS ==========

	// Bit (base unit for data size category)
	registry["bit"] = makeDataSizeUnit(
		func(v float64) float64 { return float64(units.Datasize(v) * units.Bit) },
		func(v float64) float64 { return units.Datasize(v).Bits() },
	)
	registry["bits"] = registry["bit"]

	// Byte (8 bits)
	registry["byte"] = makeDataSizeUnit(
		func(v float64) float64 { return float64(units.Datasize(v) * units.Byte) },
		func(v float64) float64 { return units.Datasize(v).Bytes() },
	)
	registry["bytes"] = registry["byte"]
	registry["b"] = registry["byte"] // Common abbreviation

	// ========== BINARY PREFIXES (IEC) - 1024-based ==========
	// These are the traditional computing units

	// Kibibyte (KiB = 1024 bytes)
	registry["kib"] = makeDataSizeUnit(
		func(v float64) float64 { return float64(units.Datasize(v) * units.Kibibyte) },
		func(v float64) float64 { return units.Datasize(v).Kibibytes() },
	)
	registry["kibibyte"] = registry["kib"]
	registry["kibibytes"] = registry["kib"]

	// Mebibyte (MiB = 1024 KiB)
	registry["mib"] = makeDataSizeUnit(
		func(v float64) float64 { return float64(units.Datasize(v) * units.Mebibyte) },
		func(v float64) float64 { return units.Datasize(v).Mebibytes() },
	)
	registry["mebibyte"] = registry["mib"]
	registry["mebibytes"] = registry["mib"]

	// Gibibyte (GiB = 1024 MiB)
	registry["gib"] = makeDataSizeUnit(
		func(v float64) float64 { return float64(units.Datasize(v) * units.Gibibyte) },
		func(v float64) float64 { return units.Datasize(v).Gibibytes() },
	)
	registry["gibibyte"] = registry["gib"]
	registry["gibibytes"] = registry["gib"]

	// Tebibyte (TiB = 1024 GiB)
	registry["tib"] = makeDataSizeUnit(
		func(v float64) float64 { return float64(units.Datasize(v) * units.Tebibyte) },
		func(v float64) float64 { return units.Datasize(v).Tebibytes() },
	)
	registry["tebibyte"] = registry["tib"]
	registry["tebibytes"] = registry["tib"]

	// Pebibyte (PiB = 1024 TiB)
	registry["pib"] = makeDataSizeUnit(
		func(v float64) float64 { return float64(units.Datasize(v) * units.Pebibyte) },
		func(v float64) float64 { return units.Datasize(v).Pebibytes() },
	)
	registry["pebibyte"] = registry["pib"]
	registry["pebibytes"] = registry["pib"]

	// ========== DECIMAL PREFIXES (SI) - 1000-based ==========
	// Used in storage marketing and networking contexts
	// Note: KB/MB/GB are aliased to binary (KiB/MiB/GiB) for user convenience

	// Kilobyte - use binary (1024) as default for KB (matches most user expectations)
	registry["kb"] = registry["kib"]
	registry["kilobyte"] = registry["kib"]
	registry["kilobytes"] = registry["kib"]

	// Megabyte - use binary (1024²) as default for MB
	registry["mb"] = registry["mib"]
	registry["megabyte"] = registry["mib"]
	registry["megabytes"] = registry["mib"]

	// Gigabyte - use binary (1024³) as default for GB
	registry["gb"] = registry["gib"]
	registry["gigabyte"] = registry["gib"]
	registry["gigabytes"] = registry["gib"]

	// Terabyte - use binary (1024⁴) as default for TB
	registry["tb"] = registry["tib"]
	registry["terabyte"] = registry["tib"]
	registry["terabytes"] = registry["tib"]

	// Petabyte - use binary (1024⁵) as default for PB
	registry["pb"] = registry["pib"]
	registry["petabyte"] = registry["pib"]
	registry["petabytes"] = registry["pib"]

	// Exabyte - use binary (1024⁶) as default for EB
	registry["eb"] = makeDataSizeUnit(
		func(v float64) float64 { return float64(units.Datasize(v) * units.Exbibyte) },
		func(v float64) float64 { return units.Datasize(v).Exbibytes() },
	)
	registry["exabyte"] = registry["eb"]
	registry["exabytes"] = registry["eb"]
	registry["eib"] = registry["eb"]
	registry["exbibyte"] = registry["eb"]
	registry["exbibytes"] = registry["eb"]

	// ========== BIT UNITS (NETWORK) - 1000-based ==========
	// Standard for network throughput (Mbps, Gbps, etc.)

	// Kilobit (1000 bits)
	registry["kbit"] = makeDataSizeUnit(
		func(v float64) float64 { return float64(units.Datasize(v) * units.Kilobit) },
		func(v float64) float64 { return units.Datasize(v).Kilobits() },
	)
	registry["kilobit"] = registry["kbit"]
	registry["kilobits"] = registry["kbit"]
	registry["kbps"] = registry["kbit"] // Kbps when used as rate numerator

	// Megabit (1,000,000 bits)
	registry["mbit"] = makeDataSizeUnit(
		func(v float64) float64 { return float64(units.Datasize(v) * units.Megabit) },
		func(v float64) float64 { return units.Datasize(v).Megabits() },
	)
	registry["megabit"] = registry["mbit"]
	registry["megabits"] = registry["mbit"]
	registry["mbps"] = registry["mbit"] // Mbps when used as rate numerator

	// Gigabit (1,000,000,000 bits)
	registry["gbit"] = makeDataSizeUnit(
		func(v float64) float64 { return float64(units.Datasize(v) * units.Gigabit) },
		func(v float64) float64 { return units.Datasize(v).Gigabits() },
	)
	registry["gigabit"] = registry["gbit"]
	registry["gigabits"] = registry["gbit"]
	registry["gbps"] = registry["gbit"] // Gbps when used as rate numerator

	// Terabit (1,000,000,000,000 bits)
	registry["tbit"] = makeDataSizeUnit(
		func(v float64) float64 { return float64(units.Datasize(v) * units.Terabit) },
		func(v float64) float64 { return units.Datasize(v).Terabits() },
	)
	registry["terabit"] = registry["tbit"]
	registry["terabits"] = registry["tbit"]
	registry["tbps"] = registry["tbit"] // Tbps when used as rate numerator

	// Petabit (1,000,000,000,000,000 bits)
	registry["pbit"] = makeDataSizeUnit(
		func(v float64) float64 { return float64(units.Datasize(v) * units.Petabit) },
		func(v float64) float64 { return units.Datasize(v).Petabits() },
	)
	registry["petabit"] = registry["pbit"]
	registry["petabits"] = registry["pbit"]
	registry["pbps"] = registry["pbit"] // Pbps when used as rate numerator

	// ========== BINARY BIT UNITS (IEC) - 1024-based ==========
	// Less common but included for completeness

	// Kibibit (1024 bits)
	registry["kibit"] = makeDataSizeUnit(
		func(v float64) float64 { return float64(units.Datasize(v) * units.Kibibit) },
		func(v float64) float64 { return units.Datasize(v).Kibibits() },
	)
	registry["kibibit"] = registry["kibit"]
	registry["kibibits"] = registry["kibit"]

	// Mebibit (1024 Kibibits)
	registry["mibit"] = makeDataSizeUnit(
		func(v float64) float64 { return float64(units.Datasize(v) * units.Mebibit) },
		func(v float64) float64 { return units.Datasize(v).Mebibits() },
	)
	registry["mebibit"] = registry["mibit"]
	registry["mebibits"] = registry["mibit"]

	// Gibibit (1024 Mebibits)
	registry["gibit"] = makeDataSizeUnit(
		func(v float64) float64 { return float64(units.Datasize(v) * units.Gibibit) },
		func(v float64) float64 { return units.Datasize(v).Gibibits() },
	)
	registry["gibibit"] = registry["gibit"]
	registry["gibibits"] = registry["gibit"]

	// Tebibit (1024 Gibibits)
	registry["tibit"] = makeDataSizeUnit(
		func(v float64) float64 { return float64(units.Datasize(v) * units.Tebibit) },
		func(v float64) float64 { return units.Datasize(v).Tebibits() },
	)
	registry["tebibit"] = registry["tibit"]
	registry["tebibits"] = registry["tibit"]
}
