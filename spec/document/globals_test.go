package document

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

func TestParseGlobals_Numbers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected decimal.Decimal
	}{
		{"integer", "42", decimal.NewFromInt(42)},
		{"decimal", "3.14", decimal.NewFromFloat(3.14)},
		{"with suffix K", "1.5K", decimal.NewFromInt(1500)},
		{"with suffix M", "10M", decimal.NewFromInt(10_000_000)},
		{"percentage", "25%", decimal.NewFromFloat(0.25)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			globals := map[string]string{"x": tt.input}
			parsed, err := ParseGlobals(globals)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			val, ok := parsed.Values["x"]
			if !ok {
				t.Fatal("expected 'x' in parsed values")
			}

			num, ok := val.(*types.Number)
			if !ok {
				t.Fatalf("expected Number, got %T", val)
			}

			if !num.Value.Equal(tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, num.Value)
			}
		})
	}
}

func TestParseGlobals_Quantities(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedVal  decimal.Decimal
		expectedUnit string
	}{
		{"meters", "10 meters", decimal.NewFromInt(10), "meters"},
		{"kg", "5 kg", decimal.NewFromInt(5), "kg"},
		{"MB", "100 MB", decimal.NewFromInt(100), "MB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			globals := map[string]string{"q": tt.input}
			parsed, err := ParseGlobals(globals)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			val, ok := parsed.Values["q"]
			if !ok {
				t.Fatal("expected 'q' in parsed values")
			}

			qty, ok := val.(*types.Quantity)
			if !ok {
				t.Fatalf("expected Quantity, got %T", val)
			}

			if !qty.Value.Equal(tt.expectedVal) {
				t.Errorf("value: expected %v, got %v", tt.expectedVal, qty.Value)
			}
			if qty.Unit != tt.expectedUnit {
				t.Errorf("unit: expected %q, got %q", tt.expectedUnit, qty.Unit)
			}
		})
	}
}

func TestParseGlobals_Currencies(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedVal    decimal.Decimal
		expectedSymbol string
	}{
		{"USD with dollar", "$100", decimal.NewFromInt(100), "$"},
		{"EUR", "50 EUR", decimal.NewFromInt(50), "EUR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			globals := map[string]string{"c": tt.input}
			parsed, err := ParseGlobals(globals)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			val, ok := parsed.Values["c"]
			if !ok {
				t.Fatal("expected 'c' in parsed values")
			}

			cur, ok := val.(*types.Currency)
			if !ok {
				t.Fatalf("expected Currency, got %T", val)
			}

			if !cur.Value.Equal(tt.expectedVal) {
				t.Errorf("value: expected %v, got %v", tt.expectedVal, cur.Value)
			}
			if cur.Symbol != tt.expectedSymbol {
				t.Errorf("symbol: expected %q, got %q", tt.expectedSymbol, cur.Symbol)
			}
		})
	}
}

func TestParseGlobals_Durations(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedVal  decimal.Decimal
		expectedUnit string
	}{
		{"days", "5 days", decimal.NewFromInt(5), "day"},
		{"weeks", "2 weeks", decimal.NewFromInt(2), "week"},
		{"hours", "24 hours", decimal.NewFromInt(24), "hour"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			globals := map[string]string{"d": tt.input}
			parsed, err := ParseGlobals(globals)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			val, ok := parsed.Values["d"]
			if !ok {
				t.Fatal("expected 'd' in parsed values")
			}

			dur, ok := val.(*types.Duration)
			if !ok {
				t.Fatalf("expected Duration, got %T", val)
			}

			if !dur.Value.Equal(tt.expectedVal) {
				t.Errorf("value: expected %v, got %v", tt.expectedVal, dur.Value)
			}
			if dur.Unit != tt.expectedUnit {
				t.Errorf("unit: expected %q, got %q", tt.expectedUnit, dur.Unit)
			}
		})
	}
}

func TestParseGlobals_Dates(t *testing.T) {
	globals := map[string]string{"date": "Jan 15 2025"}
	parsed, err := ParseGlobals(globals)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, ok := parsed.Values["date"]
	if !ok {
		t.Fatal("expected 'date' in parsed values")
	}

	date, ok := val.(*types.Date)
	if !ok {
		t.Fatalf("expected Date, got %T", val)
	}

	if date.Time.Year() != 2025 {
		t.Errorf("year: expected 2025, got %d", date.Time.Year())
	}
	if date.Time.Month() != 1 {
		t.Errorf("month: expected 1, got %d", date.Time.Month())
	}
	if date.Time.Day() != 15 {
		t.Errorf("day: expected 15, got %d", date.Time.Day())
	}
}

func TestParseGlobals_Booleans(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"true", "true", true},
		{"false", "false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			globals := map[string]string{"b": tt.input}
			parsed, err := ParseGlobals(globals)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			val, ok := parsed.Values["b"]
			if !ok {
				t.Fatal("expected 'b' in parsed values")
			}

			b, ok := val.(*types.Boolean)
			if !ok {
				t.Fatalf("expected Boolean, got %T", val)
			}

			if b.Value != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b.Value)
			}
		})
	}
}

func TestParseGlobals_Rates(t *testing.T) {
	globals := map[string]string{"rate": "100 MB/s"}
	parsed, err := ParseGlobals(globals)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, ok := parsed.Values["rate"]
	if !ok {
		t.Fatal("expected 'rate' in parsed values")
	}

	rate, ok := val.(*types.Rate)
	if !ok {
		t.Fatalf("expected Rate, got %T", val)
	}

	if !rate.Amount.Value.Equal(decimal.NewFromInt(100)) {
		t.Errorf("amount value: expected 100, got %v", rate.Amount.Value)
	}
	if rate.Amount.Unit != "MB" {
		t.Errorf("amount unit: expected 'MB', got %q", rate.Amount.Unit)
	}
	if rate.PerUnit != "second" {
		t.Errorf("per unit: expected 'second', got %q", rate.PerUnit)
	}
}

func TestParseGlobals_ExpressionNotAllowed(t *testing.T) {
	// Expressions like "1 + 1" should not be allowed
	globals := map[string]string{"x": "1 + 1"}
	_, err := ParseGlobals(globals)
	if err == nil {
		t.Error("expected error for expression, got nil")
	}
}

func TestParseGlobals_MultipleValues(t *testing.T) {
	globals := map[string]string{
		"price":    "$100",
		"quantity": "5 kg",
		"days":     "7 days",
	}

	parsed, err := ParseGlobals(globals)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(parsed.Values) != 3 {
		t.Errorf("expected 3 values, got %d", len(parsed.Values))
	}

	// Check price
	if cur, ok := parsed.Values["price"].(*types.Currency); ok {
		if !cur.Value.Equal(decimal.NewFromInt(100)) {
			t.Errorf("price: expected 100, got %v", cur.Value)
		}
	} else {
		t.Errorf("price: expected Currency, got %T", parsed.Values["price"])
	}

	// Check quantity
	if qty, ok := parsed.Values["quantity"].(*types.Quantity); ok {
		if qty.Unit != "kg" {
			t.Errorf("quantity unit: expected 'kg', got %q", qty.Unit)
		}
	} else {
		t.Errorf("quantity: expected Quantity, got %T", parsed.Values["quantity"])
	}

	// Check days
	if dur, ok := parsed.Values["days"].(*types.Duration); ok {
		if dur.Unit != "day" {
			t.Errorf("days unit: expected 'day', got %q", dur.Unit)
		}
	} else {
		t.Errorf("days: expected Duration, got %T", parsed.Values["days"])
	}
}
