package units

import "strings"

// SpeedDecomposition maps a Speed unit to its Rate components.
type SpeedDecomposition struct {
	NumeratorUnit string // Length unit (e.g., "km", "mi", "m", "nmi")
	TimeUnit      string // Time unit (e.g., "hour", "second")
}

// speedDecompositions maps lowercase speed unit aliases to their Rate components.
var speedDecompositions = map[string]SpeedDecomposition{
	"kph":                 {NumeratorUnit: "km", TimeUnit: "hour"},
	"kmh":                 {NumeratorUnit: "km", TimeUnit: "hour"},
	"km/h":                {NumeratorUnit: "km", TimeUnit: "hour"},
	"kilometers per hour": {NumeratorUnit: "km", TimeUnit: "hour"},
	"mph":                 {NumeratorUnit: "mi", TimeUnit: "hour"},
	"miles per hour":      {NumeratorUnit: "mi", TimeUnit: "hour"},
	"mps":                 {NumeratorUnit: "m", TimeUnit: "second"},
	"m/s":                 {NumeratorUnit: "m", TimeUnit: "second"},
	"meters per second":   {NumeratorUnit: "m", TimeUnit: "second"},
	"knot":                {NumeratorUnit: "nmi", TimeUnit: "hour"},
	"knots":               {NumeratorUnit: "nmi", TimeUnit: "hour"},
}

// DecomposeSpeedUnit returns the Rate components for a Speed unit.
// Returns the numerator unit, time unit, and whether the decomposition was found.
func DecomposeSpeedUnit(unit string) (numeratorUnit, timeUnit string, ok bool) {
	d, found := speedDecompositions[strings.ToLower(unit)]
	if !found {
		return "", "", false
	}
	return d.NumeratorUnit, d.TimeUnit, true
}

// IsSpeedUnit checks if a unit belongs to the Speed category.
func IsSpeedUnit(unit string) bool {
	return CategoryForUnit(unit) == "Speed"
}
