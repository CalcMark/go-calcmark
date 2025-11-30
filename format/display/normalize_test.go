package display

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestNormalizeForDisplay(t *testing.T) {
	tests := []struct {
		name         string
		value        string
		unit         string
		expectedVal  string
		expectedUnit string
	}{
		// ========== SI LENGTH ==========
		// Millimeters scaling up
		{"1000mm to m", "1000", "mm", "1", "m"},
		{"1500mm to m", "1500", "mm", "1.5", "m"},
		{"10000mm to m", "10000", "mm", "10", "m"},
		{"1000000mm to km", "1000000", "mm", "1", "km"},

		// Centimeters scaling up
		{"100cm to m", "100", "cm", "1", "m"},
		{"1000cm to m", "1000", "cm", "10", "m"},
		{"100000cm to km", "100000", "cm", "1", "km"},

		// Meters scaling up
		{"1000m to km", "1000", "m", "1", "km"},
		{"1500m to km", "1500", "m", "1.5", "km"},
		{"10000m to km", "10000", "m", "10", "km"},

		// Small values: algorithm picks largest unit where value >= 1
		// 0.5m = 50cm, so cm is chosen (50 >= 1)
		{"0.5m to cm", "0.5", "m", "50", "cm"},
		// 0.01cm = 0.1mm, mm is largest where value >= 0.1, so mm
		{"0.01cm to mm", "0.01", "cm", "0.1", "mm"},

		// Values that stay in their unit
		{"500m stays m", "500", "m", "500", "m"},
		{"50cm stays cm", "50", "cm", "50", "cm"},

		// ========== US CUSTOMARY LENGTH ==========
		// 12 inches = 1 foot; algorithm picks ft (largest where value >= 1)
		{"12in to ft", "12", "inch", "1", "ft"},
		// 36 inches = 3 feet = 1 yard; algorithm picks yd
		{"36in to yd", "36", "inch", "1", "yd"},
		// 24 inches = 2 feet; ft is chosen
		{"24in to ft", "24", "inch", "2", "ft"},

		// Feet scaling up
		{"3ft to yd", "3", "feet", "1", "yd"},
		{"5280ft to mi", "5280", "feet", "1", "mi"},
		{"10560ft to mi", "10560", "feet", "2", "mi"},

		// Yards scaling up
		{"1760yd to mi", "1760", "yard", "1", "mi"},

		// Small US length stays as-is (value already >= 1 in original unit)
		{"6in stays in", "6", "inch", "6", "in"},
		{"2ft stays ft", "2", "feet", "2", "ft"},

		// ========== SI MASS ==========
		// Milligrams scaling up
		{"1000mg to g", "1000", "mg", "1", "g"},
		{"1000000mg to kg", "1000000", "mg", "1", "kg"},

		// Grams scaling up
		{"1000g to kg", "1000", "g", "1", "kg"},
		{"1500g to kg", "1500", "g", "1.5", "kg"},
		{"1000000g to t", "1000000", "g", "1", "t"},

		// Kilograms scaling up
		{"1000kg to t", "1000", "kg", "1", "t"},
		{"2500kg to t", "2500", "kg", "2.5", "t"},

		// Small mass stays as-is
		{"500g stays g", "500", "g", "500", "g"},
		{"50kg stays kg", "50", "kg", "50", "kg"},

		// ========== US CUSTOMARY MASS ==========
		// Ounces scaling up
		{"16oz to lb", "16", "oz", "1", "lb"},
		{"32oz to lb", "32", "oz", "2", "lb"},

		// Small US mass stays as-is
		{"8oz stays oz", "8", "oz", "8", "oz"},
		{"5lb stays lb", "5", "lb", "5", "lb"},

		// ========== SI VOLUME ==========
		// Milliliters scaling up
		{"1000ml to l", "1000", "ml", "1", "l"},
		{"1500ml to l", "1500", "ml", "1.5", "l"},

		// Small volume stays as-is
		{"500ml stays ml", "500", "ml", "500", "ml"},
		{"5l stays l", "5", "l", "5", "l"},

		// ========== US CUSTOMARY VOLUME ==========
		// 3 tsp = 3 tsp (stays, since 3 >= 1 and tbsp would be < 1)
		// Actually 3 tsp = 1 tbsp (3 tsp / 3 = 1 tbsp), so tbsp
		// Wait: 1 tbsp = 3 tsp, so 3 tsp = 1 tbsp. Algorithm picks tbsp.
		{"3tsp to tbsp", "3", "tsp", "1", "tbsp"},
		{"6tsp to tbsp", "6", "tsp", "2", "tbsp"},

		// Tablespoons scaling up
		{"16tbsp to cup", "16", "tbsp", "1", "cup"},

		// Cups scaling up: 2 cups = 1 pint
		{"2cups to pt", "2", "cup", "1", "pt"},
		{"4cups to qt", "4", "cup", "1", "qt"},
		{"16cups to gal", "16", "cup", "1", "gal"},

		// Pints scaling up
		{"2pt to qt", "2", "pint", "1", "qt"},
		{"8pt to gal", "8", "pint", "1", "gal"}, // 8 pints = 1 gallon (2 pints/qt * 4 qt/gal = 8 pt/gal)

		// Quarts scaling up
		{"4qt to gal", "4", "quart", "1", "gal"},

		// Small US volume: algorithm picks largest where value >= 1
		{"1cup stays cup", "1", "cup", "1", "cup"},
		// 3 cups = 1.5 pints; pint is chosen since 1.5 >= 1
		{"3cups to pt", "3", "cup", "1.5", "pt"},
		{"1qt stays qt", "1", "quart", "1", "qt"},

		// ========== DATA/STORAGE (Binary) ==========
		// Bytes scaling up
		{"1024B to KB", "1024", "bytes", "1", "KB"},
		{"1048576B to MB", "1048576", "bytes", "1", "MB"},
		{"1073741824B to GB", "1073741824", "bytes", "1", "GB"},

		// Kilobytes scaling up
		{"1024KB to MB", "1024", "KB", "1", "MB"},
		{"1048576KB to GB", "1048576", "KB", "1", "GB"},

		// Megabytes scaling up
		{"1024MB to GB", "1024", "MB", "1", "GB"},
		{"1048576MB to TB", "1048576", "MB", "1", "TB"},

		// Gigabytes scaling up
		{"1024GB to TB", "1024", "GB", "1", "TB"},
		{"1048576GB to PB", "1048576", "GB", "1", "PB"},

		// Large values with decimal
		{"1536MB to GB", "1536", "MB", "1.5", "GB"},
		// 23400000 GB = 23400000 / 1024 / 1024 PB ≈ 22.31 PB (rounding)
		{"23400000GB to PB", "23400000", "GB", "22.31", "PB"},

		// Small data stays as-is
		{"500MB stays MB", "500", "MB", "500", "MB"},
		{"100GB stays GB", "100", "GB", "100", "GB"},

		// ========== POWER ==========
		// Watts scaling up
		{"1000W to kW", "1000", "watt", "1", "kW"},
		{"1000000W to MW", "1000000", "watt", "1", "MW"},

		// Kilowatts scaling up
		{"1000kW to MW", "1000", "kW", "1", "MW"},

		// Small power stays as-is
		{"500W stays W", "500", "watt", "500", "W"},
		{"50kW stays kW", "50", "kW", "50", "kW"},

		// ========== ENERGY ==========
		// Joules scaling up
		{"1000J to kJ", "1000", "joule", "1", "kJ"},

		// Calories scaling up
		{"1000cal to kcal", "1000", "calorie", "1", "kcal"},

		// ========== AREA ==========
		// Square meters scaling up (using "sq X" display format)
		{"10000m² to ha", "10000", "m²", "1", "ha"},
		{"1000000m² to sq km", "1000000", "m²", "1", "sq km"},

		// Square feet scaling up
		{"43560ft² to acre", "43560", "ft²", "1", "ac"},

		// Acres scaling up
		{"640acres to sq mi", "640", "acre", "1", "sq mi"},

		// ========== UNKNOWN/ARBITRARY UNITS ==========
		// Units not in our registry should pass through unchanged
		{"arbitrary unit", "1000000", "widgets", "1000000", "widgets"},
		{"users", "100000", "users", "100000", "users"},
		{"requests", "5000000", "requests", "5000000", "requests"},

		// ========== EDGE CASES ==========
		// Zero
		{"zero meters", "0", "m", "0", "m"},
		{"zero bytes", "0", "bytes", "0", "bytes"},

		// Negative values
		{"negative meters", "-1000", "m", "-1", "km"},
		{"negative GB", "-1024", "GB", "-1", "TB"},

		// Very small values (stay in smallest reasonable unit)
		{"tiny meters", "0.001", "m", "1", "mm"},
		{"very tiny meters", "0.0001", "m", "0.1", "mm"},

		// Exact boundary values
		{"exactly 1km in m", "1000", "m", "1", "km"},
		{"just under 1km", "999", "m", "999", "m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := decimal.NewFromString(tt.value)
			if err != nil {
				t.Fatalf("invalid test value %q: %v", tt.value, err)
			}

			gotVal, gotUnit := NormalizeForDisplay(value, tt.unit)

			// Compare values with reasonable precision
			expectedVal, _ := decimal.NewFromString(tt.expectedVal)

			// Allow small floating-point differences
			diff := gotVal.Sub(expectedVal).Abs()
			tolerance := decimal.NewFromFloat(0.01)

			if diff.GreaterThan(tolerance) {
				t.Errorf("NormalizeForDisplay(%s %s) value = %s, want %s",
					tt.value, tt.unit, gotVal.String(), tt.expectedVal)
			}

			if gotUnit != tt.expectedUnit {
				t.Errorf("NormalizeForDisplay(%s %s) unit = %q, want %q",
					tt.value, tt.unit, gotUnit, tt.expectedUnit)
			}
		})
	}
}

func TestNormalizeForDisplay_PreservesSystem(t *testing.T) {
	// Test that SI units stay SI and US Customary stays US Customary
	tests := []struct {
		name       string
		value      string
		unit       string
		wantSystem string // "SI" or "US"
	}{
		// SI stays SI
		{"meters stay metric", "1000", "m", "SI"},
		{"millimeters stay metric", "1000000", "mm", "SI"},
		{"grams stay metric", "1000000", "g", "SI"},
		{"liters stay metric", "1000", "l", "SI"},

		// US stays US
		{"feet stay US", "5280", "feet", "US"},
		{"inches stay US", "36", "inch", "US"},
		{"cups stay US", "16", "cup", "US"},
		{"ounces stay US", "32", "oz", "US"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, _ := decimal.NewFromString(tt.value)
			_, gotUnit := NormalizeForDisplay(value, tt.unit)

			// Verify the result unit is in the expected system
			gotSystem := getUnitSystem(gotUnit)
			if gotSystem != tt.wantSystem {
				t.Errorf("NormalizeForDisplay(%s %s) resulted in %q (system: %s), want system: %s",
					tt.value, tt.unit, gotUnit, gotSystem, tt.wantSystem)
			}
		})
	}
}

// Helper to determine unit system (simplified for tests)
func getUnitSystem(unit string) string {
	siUnits := map[string]bool{
		"mm": true, "cm": true, "m": true, "km": true,
		"mg": true, "g": true, "kg": true, "t": true,
		"ml": true, "l": true,
		"W": true, "kW": true, "MW": true,
		"J": true, "kJ": true,
		"m²": true, "km²": true, "ha": true,
	}
	usUnits := map[string]bool{
		"in": true, "ft": true, "yd": true, "mi": true,
		"oz": true, "lb": true,
		"tsp": true, "tbsp": true, "cup": true, "pt": true, "qt": true, "gal": true,
		"ft²": true, "ac": true, "mi²": true,
	}

	if siUnits[unit] {
		return "SI"
	}
	if usUnits[unit] {
		return "US"
	}
	return "unknown"
}

func TestFormatQuantity_WithNormalization(t *testing.T) {
	// Integration tests: verify the full Format pipeline uses normalization
	tests := []struct {
		name     string
		value    string
		unit     string
		expected string
	}{
		// The original problem: 23.4M GB should become ~22.89 PB
		{"23.4M GB normalized", "23400000", "GB", "22.89 PB"},

		// Other examples
		{"1000 meters", "1000", "m", "1 km"},
		{"5280 feet", "5280", "feet", "1 mi"},
		{"1024 MB", "1024", "MB", "1 GB"},
		{"16 cups", "16", "cup", "1 gal"},

		// Values that should stay as-is
		{"100 users (arbitrary)", "100", "users", "100 users"},
		{"500 meters (under threshold)", "500", "m", "500 m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test will verify FormatQuantity uses normalization
			// Implementation will make this pass
			_ = tt // placeholder until integration
		})
	}
}

func BenchmarkNormalizeForDisplay(b *testing.B) {
	value := decimal.NewFromInt(23400000)
	unit := "GB"

	for b.Loop() {
		NormalizeForDisplay(value, unit)
	}
}

func BenchmarkNormalizeForDisplay_SmallValue(b *testing.B) {
	value := decimal.NewFromInt(500)
	unit := "m"

	for b.Loop() {
		NormalizeForDisplay(value, unit)
	}
}

func BenchmarkNormalizeForDisplay_UnknownUnit(b *testing.B) {
	value := decimal.NewFromInt(1000000)
	unit := "widgets"

	for b.Loop() {
		NormalizeForDisplay(value, unit)
	}
}
