package types

import (
	"testing"

	"github.com/shopspring/decimal"
)

// TestNumber tests the Number type
func TestNumber(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"integer", "42", "42"},
		{"decimal", "3.14", "3.14"},
		{"negative", "-10.5", "-10.5"},
		{"zero", "0", "0"},
		{"large number", "1234567890.123456789", "1234567890.123456789"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := NewNumberFromString(tt.input)
			if err != nil {
				t.Fatalf("NewNumberFromString(%q) error = %v", tt.input, err)
			}
			if got := n.String(); got != tt.want {
				t.Errorf("Number.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestCurrency tests the Currency type
func TestCurrency(t *testing.T) {
	tests := []struct {
		name         string
		value        string
		symbolOrCode string
		wantString   string
		wantCode     string
	}{
		{"dollar symbol", "100", "$", "$100.00", "USD"},
		{"euro symbol", "50.50", "€", "€50.50", "EUR"},
		{"pound symbol", "1000", "£", "£1000.00", "GBP"},
		{"yen symbol", "5000", "¥", "¥5000.00", "JPY"},
		{"USD code", "100", "USD", "USD100.00", "USD"},
		{"GBP code", "50", "GBP", "GBP50.00", "GBP"},
		{"custom code", "25", "CAD", "CAD25.00", "CAD"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewCurrencyFromString(tt.value, tt.symbolOrCode)
			if err != nil {
				t.Fatalf("NewCurrencyFromString(%q, %q) error = %v", tt.value, tt.symbolOrCode, err)
			}
			if got := c.String(); got != tt.wantString {
				t.Errorf("Currency.String() = %v, want %v", got, tt.wantString)
			}
			if c.Code != tt.wantCode {
				t.Errorf("Currency.Code = %v, want %v", c.Code, tt.wantCode)
			}
		})
	}
}

// TestCurrencyComparison tests IsSameCurrency
func TestCurrencyComparison(t *testing.T) {
	usd1 := NewCurrency(decimal.NewFromInt(100), "$")
	usd2 := NewCurrency(decimal.NewFromInt(200), "USD")
	eur := NewCurrency(decimal.NewFromInt(100), "€")

	if !usd1.IsSameCurrency(usd2) {
		t.Error("$ and USD should be the same currency")
	}

	if usd1.IsSameCurrency(eur) {
		t.Error("USD and EUR should be different currencies")
	}
}

// TestBoolean tests the Boolean type
func TestBoolean(t *testing.T) {
	trueVal := NewBoolean(true)
	if trueVal.String() != "true" {
		t.Errorf("Boolean(true).String() = %v, want 'true'", trueVal.String())
	}

	falseVal := NewBoolean(false)
	if falseVal.String() != "false" {
		t.Errorf("Boolean(false).String() = %v, want 'false'", falseVal.String())
	}
}

// TestDate tests the Date type
func TestDate(t *testing.T) {
	tests := []struct {
		name    string
		year    int
		month   int
		day     int
		wantErr bool
	}{
		{"valid date", 2024, 12, 25, false},
		{"leap year Feb 29", 2024, 2, 29, false},
		{"non-leap year Feb 29", 2023, 2, 29, true},
		{"invalid month", 2024, 13, 1, true},
		{"invalid day", 2024, 12, 32, true},
		{"current year", 0, 12, 25, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := NewDate(tt.year, tt.month, tt.day)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && d == nil {
				t.Error("NewDate() returned nil for valid date")
			}
		})
	}
}

// TestDateArithmetic tests date arithmetic
func TestDateArithmetic(t *testing.T) {
	d1, _ := NewDate(2024, 12, 25)
	d2 := d1.AddDays(5)

	if d2.Time.Day() != 30 {
		t.Errorf("AddDays(5) from Dec 25 should be Dec 30, got day %d", d2.Time.Day())
	}

	// Test days between
	d3, _ := NewDate(2024, 12, 31)
	days := d1.DaysBetween(d3)
	if days != 6 {
		t.Errorf("Days between Dec 25 and Dec 31 should be 6, got %d", days)
	}
}

// TestTime tests the Time type
func TestTime(t *testing.T) {
	tests := []struct {
		name             string
		hour             int
		minute           int
		second           int
		isPM             bool
		utcOffsetMinutes int
		wantErr          bool
	}{
		{"24-hour format", 14, 30, 0, false, 0, false},
		{"12-hour AM", 10, 30, 0, false, 0, false},
		{"12-hour PM", 10, 30, 0, true, 0, false},
		{"midnight 12 AM", 12, 0, 0, false, 0, false},
		{"noon 12 PM", 12, 0, 0, true, 0, false},
		{"with seconds", 14, 30, 45, false, 0, false},
		{"with UTC offset", 10, 30, 0, false, -420, false}, // UTC-7
		{"invalid hour", 25, 0, 0, false, 0, true},
		{"invalid minute", 10, 60, 0, false, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm, err := NewTime(tt.hour, tt.minute, tt.second, tt.isPM, tt.utcOffsetMinutes)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTime() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && tm == nil {
				t.Error("NewTime() returned nil for valid time")
			}
		})
	}
}

// TestDuration tests the Duration type
func TestDuration(t *testing.T) {
	tests := []struct {
		name  string
		value string
		unit  string
		want  string
	}{
		{"5 days", "5", "days", "5 days"},
		{"1.5 hours", "1.5", "hours", "1.5 hours"},
		{"30 minutes", "30", "minutes", "30 minutes"},
		{"2 weeks", "2", "weeks", "2 weeks"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := NewDurationFromString(tt.value, tt.unit)
			if err != nil {
				t.Fatalf("NewDurationFromString(%q, %q) error = %v", tt.value, tt.unit, err)
			}
			if got := d.String(); got != tt.want {
				t.Errorf("Duration.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDurationConversion tests duration unit conversion
func TestDurationConversion(t *testing.T) {
	// 2 hours should equal 120 minutes
	hours, _ := NewDurationFromString("2", "hours")
	minutes, err := hours.Convert("minutes")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	if !minutes.Value.Equal(decimal.NewFromInt(120)) {
		t.Errorf("2 hours to minutes = %v, want 120", minutes.Value)
	}

	// 1 day should equal 24 hours
	days, _ := NewDurationFromString("1", "days")
	hrs, err := days.Convert("hours")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	if !hrs.Value.Equal(decimal.NewFromInt(24)) {
		t.Errorf("1 day to hours = %v, want 24", hrs.Value)
	}
}

// TestQuantity tests the Quantity type
func TestQuantity(t *testing.T) {
	q, err := NewQuantityFromString("10", "meters")
	if err != nil {
		t.Fatalf("NewQuantityFromString() error = %v", err)
	}

	want := "10 meters"
	if got := q.String(); got != want {
		t.Errorf("Quantity.String() = %v, want %v", got, want)
	}
}

// TestTypeInterface ensures all types implement the Type interface
func TestTypeInterface(t *testing.T) {
	var _ Type = (*Number)(nil)
	var _ Type = (*Currency)(nil)
	var _ Type = (*Boolean)(nil)
	var _ Type = (*Date)(nil)
	var _ Type = (*Time)(nil)
	var _ Type = (*Duration)(nil)
	var _ Type = (*Quantity)(nil)
}
