package types

import (
	"testing"

	"github.com/shopspring/decimal"
)

// TestNumber tests

func TestNumberFromInt(t *testing.T) {
	num, err := NewNumber(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := decimal.NewFromInt(42)
	if !num.Value.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, num.Value)
	}
}

func TestNumberFromFloat(t *testing.T) {
	num, err := NewNumber(3.14)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := decimal.NewFromFloat(3.14)
	if !num.Value.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, num.Value)
	}
}

func TestNumberFromString(t *testing.T) {
	num, err := NewNumber("123.45")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected, _ := decimal.NewFromString("123.45")
	if !num.Value.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, num.Value)
	}
}

func TestNumberToDecimal(t *testing.T) {
	num, _ := NewNumber(42)
	result := num.ToDecimal()

	expected := decimal.NewFromInt(42)
	if !result.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestNumberString(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"integer", 42, "42"},
		{"decimal", 3.14, "3.14"},
		{"trailing zeros", "3.00", "3"},
		{"zero", 0, "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			num, err := NewNumber(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			result := num.String()
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestNumberEquality(t *testing.T) {
	tests := []struct {
		name     string
		a        interface{}
		b        interface{}
		expected bool
	}{
		{"equal integers", 42, 42, true},
		{"equal decimals from different types", 3.14, "3.14", true},
		{"not equal", 42, 43, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			numA, _ := NewNumber(tt.a)
			numB, _ := NewNumber(tt.b)

			result := numA.Equal(numB)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestCurrency tests

func TestCurrencyFromInt(t *testing.T) {
	curr, err := NewCurrency(1000, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := decimal.NewFromInt(1000)
	if !curr.Value.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, curr.Value)
	}

	if curr.Symbol != "$" {
		t.Errorf("expected symbol '$', got '%s'", curr.Symbol)
	}
}

func TestCurrencyFromString(t *testing.T) {
	curr, err := NewCurrency("1500.50", "$")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected, _ := decimal.NewFromString("1500.50")
	if !curr.Value.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, curr.Value)
	}
}

func TestCurrencyCustomSymbol(t *testing.T) {
	curr, err := NewCurrency(100, "€")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if curr.Symbol != "€" {
		t.Errorf("expected symbol '€', got '%s'", curr.Symbol)
	}
}

func TestCurrencyToDecimal(t *testing.T) {
	curr, _ := NewCurrency(1000, "$")
	result := curr.ToDecimal()

	expected := decimal.NewFromInt(1000)
	if !result.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestCurrencyString(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		symbol   string
		expected string
	}{
		{"dollars 1000", 1000, "$", "$1,000.00"},
		{"dollars with cents", 1500.5, "$", "$1,500.50"},
		{"euros", 100, "€", "€100.00"},
		{"large number", 1000000, "$", "$1,000,000.00"},
		{"small number", 5, "$", "$5.00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			curr, err := NewCurrency(tt.value, tt.symbol)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			result := curr.String()
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestCurrencyEquality(t *testing.T) {
	tests := []struct {
		name     string
		a        *Currency
		b        *Currency
		expected bool
	}{
		{
			"equal same symbol",
			&Currency{Value: decimal.NewFromInt(1000), Symbol: "$"},
			&Currency{Value: decimal.NewFromInt(1000), Symbol: "$"},
			true,
		},
		{
			"equal different input types",
			&Currency{Value: decimal.NewFromInt(1000), Symbol: "$"},
			func() *Currency { c, _ := NewCurrency("1000", "$"); return c }(),
			true,
		},
		{
			"not equal different values",
			&Currency{Value: decimal.NewFromInt(1000), Symbol: "$"},
			&Currency{Value: decimal.NewFromInt(2000), Symbol: "$"},
			false,
		},
		{
			"not equal different symbols",
			&Currency{Value: decimal.NewFromInt(1000), Symbol: "$"},
			&Currency{Value: decimal.NewFromInt(1000), Symbol: "€"},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.a.Equal(tt.b)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestBoolean tests

func TestBooleanFromBool(t *testing.T) {
	tests := []struct {
		name     string
		input    bool
		expected bool
	}{
		{"true", true, true},
		{"false", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := NewBoolean(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if b.Value != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b.Value)
			}
		})
	}
}

func TestBooleanFromString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
		hasError bool
	}{
		{"true", "true", true, false},
		{"yes", "yes", true, false},
		{"t", "t", true, false},
		{"y", "y", true, false},
		{"1", "1", true, false},
		{"false", "false", false, false},
		{"no", "no", false, false},
		{"f", "f", false, false},
		{"n", "n", false, false},
		{"0", "0", false, false},
		{"invalid", "invalid", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := NewBoolean(tt.input)

			if tt.hasError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if b.Value != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b.Value)
			}
		})
	}
}

func TestBooleanFromInt(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected bool
	}{
		{"zero", 0, false},
		{"one", 1, true},
		{"non-zero", 42, true},
		{"negative", -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := NewBoolean(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if b.Value != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, b.Value)
			}
		})
	}
}

func TestBooleanString(t *testing.T) {
	tests := []struct {
		name     string
		input    bool
		expected string
	}{
		{"true", true, "true"},
		{"false", false, "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, _ := NewBoolean(tt.input)
			result := b.String()

			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestBooleanEquality(t *testing.T) {
	tests := []struct {
		name     string
		a        bool
		b        bool
		expected bool
	}{
		{"both true", true, true, true},
		{"both false", false, false, true},
		{"different", true, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			boolA, _ := NewBoolean(tt.a)
			boolB, _ := NewBoolean(tt.b)

			result := boolA.Equal(boolB)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestBooleanToBool(t *testing.T) {
	tests := []struct {
		name     string
		input    bool
		expected bool
	}{
		{"true", true, true},
		{"false", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, _ := NewBoolean(tt.input)
			result := b.ToBool()

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Test type interface compliance

func TestTypeInterface(t *testing.T) {
	var _ Type = (*Number)(nil)
	var _ Type = (*Currency)(nil)
	var _ Type = (*Boolean)(nil)
}

func TestTypeNames(t *testing.T) {
	num, _ := NewNumber(42)
	curr, _ := NewCurrency(100, "$")
	b, _ := NewBoolean(true)

	if num.TypeName() != "Number" {
		t.Errorf("expected 'Number', got '%s'", num.TypeName())
	}

	if curr.TypeName() != "Currency" {
		t.Errorf("expected 'Currency', got '%s'", curr.TypeName())
	}

	if b.TypeName() != "Boolean" {
		t.Errorf("expected 'Boolean', got '%s'", b.TypeName())
	}
}

// TestCurrencyFormatting tests currency string formatting with symbols
func TestCurrencyFormatting(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		symbol   string
		expected string
	}{
		{"USD whole", 100, "$", "$100.00"},
		{"USD decimal", 100.50, "$", "$100.50"},
		{"EUR", 50, "€", "€50.00"},
		{"GBP", 25.99, "£", "£25.99"},
		{"JPY", 1000, "¥", "¥1,000.00"},
		{"large amount", 1000000, "$", "$1,000,000.00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			curr, err := NewCurrency(tt.value, tt.symbol)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			result := curr.String()
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}

			// Verify symbol is preserved
			if curr.Symbol != tt.symbol {
				t.Errorf("expected symbol %s, got %s", tt.symbol, curr.Symbol)
			}
		})
	}
}

// TestBooleanValues tests creation and string representation of boolean values
func TestBooleanValues(t *testing.T) {
	tests := []struct {
		name     string
		value    bool
		expected string
	}{
		{"true", true, "true"},
		{"false", false, "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := NewBoolean(tt.value)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if b.Value != tt.value {
				t.Errorf("expected value %v, got %v", tt.value, b.Value)
			}

			if b.String() != tt.expected {
				t.Errorf("expected string %s, got %s", tt.expected, b.String())
			}

			if b.TypeName() != "Boolean" {
				t.Errorf("expected type 'Boolean', got '%s'", b.TypeName())
			}
		})
	}

	// Test boolean keywords (Title case supported) via NewBoolean
	keywords := []struct {
		keyword  string
		expected bool
	}{
		{"true", true},
		{"True", true},
		{"yes", true},
		{"Yes", true},
		{"y", true},
		{"t", true},
		{"false", false},
		{"False", false},
		{"no", false},
		{"No", false},
		{"n", false},
		{"f", false},
	}

	for _, kw := range keywords {
		t.Run("keyword_"+kw.keyword, func(t *testing.T) {
			b, err := NewBoolean(kw.keyword)
			if err != nil {
				t.Fatalf("unexpected error for keyword '%s': %v", kw.keyword, err)
			}

			if b.Value != kw.expected {
				t.Errorf("keyword '%s': expected %v, got %v", kw.keyword, kw.expected, b.Value)
			}
		})
	}
}
