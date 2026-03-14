package units

import (
	"math"
	"testing"

	"github.com/shopspring/decimal"
)

func TestConvert_MassGramsToOunces(t *testing.T) {
	result, err := Convert(decimal.NewFromInt(280), "grams", "ounces")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f, _ := result.Float64()
	if math.Abs(f-9.88) > 0.1 {
		t.Errorf("expected ~9.88, got %f", f)
	}
}

func TestConvert_TemperatureCelsiusToFahrenheit(t *testing.T) {
	result, err := Convert(decimal.NewFromInt(220), "celsius", "fahrenheit")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f, _ := result.Float64()
	if math.Abs(f-428) > 0.5 {
		t.Errorf("expected ~428, got %f", f)
	}
}

func TestConvert_SameUnit(t *testing.T) {
	result, err := Convert(decimal.NewFromInt(100), "grams", "grams")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Equal(decimal.NewFromInt(100)) {
		t.Errorf("expected 100, got %s", result.String())
	}
}

func TestConvert_IncompatibleUnits(t *testing.T) {
	_, err := Convert(decimal.NewFromInt(100), "grams", "meters")
	if err == nil {
		t.Fatal("expected error for incompatible units")
	}
}

func TestConvert_UnknownUnit(t *testing.T) {
	_, err := Convert(decimal.NewFromInt(100), "grams", "widgets")
	if err == nil {
		t.Fatal("expected error for unknown unit")
	}
}

func TestGetSystemForUnit(t *testing.T) {
	tests := []struct {
		unit     string
		expected string
	}{
		{"grams", "SI"},
		{"ounces", "US_Customary"},
		{"celsius", "SI"},
		{"fahrenheit", "Imperial"},
		{"meters", "SI"},
		{"feet", "US_Customary"},
		{"eggs", ""},
	}

	for _, tt := range tests {
		t.Run(tt.unit, func(t *testing.T) {
			got := GetSystemForUnit(tt.unit)
			if got != tt.expected {
				t.Errorf("GetSystemForUnit(%q) = %q, want %q", tt.unit, got, tt.expected)
			}
		})
	}
}

func TestCategoryForUnit(t *testing.T) {
	tests := []struct {
		unit     string
		expected string
	}{
		{"grams", "Mass"},
		{"ounces", "Mass"},
		{"meters", "Length"},
		{"fahrenheit", "Temperature"},
		{"km/h", "Speed"},
		{"eggs", CategoryCustom},
	}

	for _, tt := range tests {
		t.Run(tt.unit, func(t *testing.T) {
			got := CategoryForUnit(tt.unit)
			if got != tt.expected {
				t.Errorf("CategoryForUnit(%q) = %q, want %q", tt.unit, got, tt.expected)
			}
		})
	}
}

func TestGetDefaultTargetUnit(t *testing.T) {
	tests := []struct {
		unit         string
		targetSystem string
		expected     string
	}{
		{"grams", "imperial", "ounce"},
		{"ounces", "si", "gram"},
		{"celsius", "imperial", "fahrenheit"},
		{"fahrenheit", "si", "celsius"},
		{"meters", "imperial", "foot"},
		{"eggs", "imperial", ""},
	}

	for _, tt := range tests {
		t.Run(tt.unit+"->"+tt.targetSystem, func(t *testing.T) {
			got := GetDefaultTargetUnit(tt.unit, tt.targetSystem)
			if got != tt.expected {
				t.Errorf("GetDefaultTargetUnit(%q, %q) = %q, want %q", tt.unit, tt.targetSystem, got, tt.expected)
			}
		})
	}
}

func TestConvert_LengthMetersToFeet(t *testing.T) {
	result, err := Convert(decimal.NewFromInt(10), "meters", "feet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f, _ := result.Float64()
	if math.Abs(f-32.8084) > 0.01 {
		t.Errorf("expected ~32.81, got %f", f)
	}
}

func TestConvert_VolumeMLToCups(t *testing.T) {
	result, err := Convert(decimal.NewFromInt(240), "ml", "cups")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f, _ := result.Float64()
	if math.Abs(f-1.0) > 0.01 {
		t.Errorf("expected ~1.0, got %f", f)
	}
}

// ========== MEASUREMENT CONVENTIONS: Imperial volume variants ==========

func TestConvert_ImperialGallonToLiters(t *testing.T) {
	result, err := Convert(decimal.NewFromInt(1), "imp gal", "liters")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f, _ := result.Float64()
	if math.Abs(f-4.546) > 0.01 {
		t.Errorf("expected ~4.546, got %f", f)
	}
}

func TestConvert_ImperialPintToML(t *testing.T) {
	result, err := Convert(decimal.NewFromInt(1), "imp pt", "ml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f, _ := result.Float64()
	if math.Abs(f-568.26) > 0.5 {
		t.Errorf("expected ~568.26, got %f", f)
	}
}

func TestConvert_ImperialQuartToML(t *testing.T) {
	result, err := Convert(decimal.NewFromInt(1), "imp qt", "ml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f, _ := result.Float64()
	if math.Abs(f-1136.52) > 1.0 {
		t.Errorf("expected ~1136.52, got %f", f)
	}
}

func TestConvert_ImperialFlOzToML(t *testing.T) {
	result, err := Convert(decimal.NewFromInt(1), "imp fl oz", "ml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f, _ := result.Float64()
	if math.Abs(f-28.41) > 0.1 {
		t.Errorf("expected ~28.41, got %f", f)
	}
}

func TestConvert_ImperialCupToML(t *testing.T) {
	result, err := Convert(decimal.NewFromInt(1), "imp cup", "ml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f, _ := result.Float64()
	if math.Abs(f-284.13) > 0.5 {
		t.Errorf("expected ~284.13, got %f", f)
	}
}

// ========== MEASUREMENT CONVENTIONS: US-qualified volume aliases ==========

func TestConvert_USGallonToLiters(t *testing.T) {
	result, err := Convert(decimal.NewFromInt(1), "us gal", "liters")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f, _ := result.Float64()
	if math.Abs(f-3.785) > 0.01 {
		t.Errorf("expected ~3.785, got %f", f)
	}
}

func TestConvert_USFlOzToML(t *testing.T) {
	result, err := Convert(decimal.NewFromInt(1), "us fl oz", "ml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f, _ := result.Float64()
	if math.Abs(f-29.57) > 0.1 {
		t.Errorf("expected ~29.57, got %f", f)
	}
}

// ========== MEASUREMENT CONVENTIONS: Troy mass variants ==========

func TestConvert_TroyOzToGrams(t *testing.T) {
	result, err := Convert(decimal.NewFromInt(1), "troy oz", "grams")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f, _ := result.Float64()
	if math.Abs(f-31.1035) > 0.01 {
		t.Errorf("expected ~31.1035, got %f", f)
	}
}

func TestConvert_TroyLbToGrams(t *testing.T) {
	result, err := Convert(decimal.NewFromInt(1), "troy lb", "grams")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f, _ := result.Float64()
	if math.Abs(f-373.24) > 0.1 {
		t.Errorf("expected ~373.24, got %f", f)
	}
}

// ========== MEASUREMENT CONVENTIONS: US-qualified mass aliases ==========

func TestConvert_USOzToGrams(t *testing.T) {
	// "us oz" should be identical to bare "oz" (avoirdupois)
	result, err := Convert(decimal.NewFromInt(1), "us oz", "grams")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f, _ := result.Float64()
	if math.Abs(f-28.35) > 0.1 {
		t.Errorf("expected ~28.35, got %f", f)
	}
}

// ========== MEASUREMENT CONVENTIONS: Ton variants ==========

func TestConvert_ShortTonToKg(t *testing.T) {
	result, err := Convert(decimal.NewFromInt(1), "short ton", "kg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f, _ := result.Float64()
	if math.Abs(f-907.185) > 0.5 {
		t.Errorf("expected ~907.185, got %f", f)
	}
}

func TestConvert_LongTonToKg(t *testing.T) {
	result, err := Convert(decimal.NewFromInt(1), "long ton", "kg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f, _ := result.Float64()
	if math.Abs(f-1016.047) > 0.5 {
		t.Errorf("expected ~1016.047, got %f", f)
	}
}

// ========== MEASUREMENT CONVENTIONS: Cross-system conversions ==========

func TestConvert_ImperialGallonToUSGallon(t *testing.T) {
	result, err := Convert(decimal.NewFromInt(1), "imp gal", "us gal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f, _ := result.Float64()
	// 1 imp gal = 4.546 L, 1 us gal = 3.785 L, so 1 imp gal ≈ 1.201 us gal
	if math.Abs(f-1.201) > 0.01 {
		t.Errorf("expected ~1.201, got %f", f)
	}
}

func TestConvert_TroyOzToStandardOz(t *testing.T) {
	result, err := Convert(decimal.NewFromInt(1), "troy oz", "us oz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	f, _ := result.Float64()
	// 1 troy oz = 31.1035g, 1 std oz = 28.3495g, so 1 troy oz ≈ 1.097 std oz
	if math.Abs(f-1.097) > 0.01 {
		t.Errorf("expected ~1.097, got %f", f)
	}
}
