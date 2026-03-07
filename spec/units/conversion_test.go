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
		{"eggs", ""},
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
