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
		{"ms", "millisecond"},
		{"millisecond", "millisecond"},
		{"milliseconds", "millisecond"},
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

func TestNormalizeTimeUnit_AdjectivalForms(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"daily", "day"},
		{"weekly", "week"},
		{"monthly", "month"},
		{"quarterly", "quarter"},
		{"yearly", "year"},
		{"Daily", "day"},
		{"MONTHLY", "month"},
		{"quarter", "quarter"},
		{"quarters", "quarter"},
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

func TestPeriodToPeriodsPerYear(t *testing.T) {
	tests := []struct {
		period   string
		expected int
		ok       bool
	}{
		{"year", 1, true},
		{"yearly", 1, true},
		{"quarter", 4, true},
		{"quarterly", 4, true},
		{"month", 12, true},
		{"monthly", 12, true},
		{"week", 52, true},
		{"weekly", 52, true},
		{"day", 365, true},
		{"daily", 365, true},
		{"second", 0, false},
		{"unknown", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.period, func(t *testing.T) {
			n, ok := PeriodToPeriodsPerYear(tt.period)
			if ok != tt.ok {
				t.Errorf("PeriodToPeriodsPerYear(%q) ok = %v, expected %v", tt.period, ok, tt.ok)
			}
			if n != tt.expected {
				t.Errorf("PeriodToPeriodsPerYear(%q) = %d, expected %d", tt.period, n, tt.expected)
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
