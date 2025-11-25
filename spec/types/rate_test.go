package types

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestRateCreation(t *testing.T) {
	tests := []struct {
		name     string
		amount   *Quantity
		perUnit  string
		expected string
	}{
		{
			name:     "bandwidth rate",
			amount:   &Quantity{Value: decimal.NewFromInt(100), Unit: "MB"},
			perUnit:  "second",
			expected: "100 MB/s",
		},
		{
			name:     "cost rate",
			amount:   &Quantity{Value: decimal.NewFromFloat(0.10), Unit: "USD"},
			perUnit:  "hour",
			expected: "0.1 USD/h",
		},
		{
			name:     "data rate with day",
			amount:   &Quantity{Value: decimal.NewFromInt(5), Unit: "GB"},
			perUnit:  "day",
			expected: "5 GB/day",
		},
		{
			name:     "speed rate",
			amount:   &Quantity{Value: decimal.NewFromInt(60), Unit: "meters"},
			perUnit:  "second",
			expected: "60 meters/s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rate := NewRate(tt.amount, tt.perUnit)
			result := rate.String()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestRateArithmetic(t *testing.T) {
	rate1 := NewRate(&Quantity{Value: decimal.NewFromInt(100), Unit: "MB"}, "second")
	rate2 := NewRate(&Quantity{Value: decimal.NewFromInt(50), Unit: "MB"}, "second")

	t.Run("add compatible rates", func(t *testing.T) {
		result, err := rate1.Add(rate2)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		expected := "150 MB/s"
		if result.String() != expected {
			t.Errorf("Expected %s, got %s", expected, result.String())
		}
	})

	t.Run("subtract compatible rates", func(t *testing.T) {
		result, err := rate1.Subtract(rate2)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		expected := "50 MB/s"
		if result.String() != expected {
			t.Errorf("Expected %s, got %s", expected, result.String())
		}
	})

	t.Run("incompatible time units", func(t *testing.T) {
		rate3 := NewRate(&Quantity{Value: decimal.NewFromInt(10), Unit: "MB"}, "hour")
		_, err := rate1.Add(rate3)
		if err == nil {
			t.Error("Expected error for incompatible time units")
		}
	})
}

func TestTimeUnitNormalization(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"s", "second"},
		{"sec", "second"},
		{"second", "second"},
		{"seconds", "second"},
		{"h", "hour"},
		{"hour", "hour"},
		{"hours", "hour"},
		{"day", "day"},
		{"days", "day"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeTimeUnit(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeTimeUnit(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTimeUnitToSeconds(t *testing.T) {
	tests := []struct {
		unit     string
		expected int64
	}{
		{"second", 1},
		{"minute", 60},
		{"hour", 3600},
		{"day", 86400},
		{"week", 604800},
		{"month", 2592000},
		{"year", 31536000},
	}

	for _, tt := range tests {
		t.Run(tt.unit, func(t *testing.T) {
			result, err := TimeUnitToSeconds(tt.unit)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if !result.Equal(decimal.NewFromInt(tt.expected)) {
				t.Errorf("TimeUnitToSeconds(%q) = %s, expected %d", tt.unit, result.String(), tt.expected)
			}
		})
	}
}
