package document

import (
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

func TestParseFrontmatter_NoFrontmatter(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{"empty", ""},
		{"plain text", "Hello world"},
		{"calculation", "x = 10"},
		{"markdown header", "# Title\n\nSome content"},
		{"dashes in middle", "Some text\n---\nMore text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, remaining, err := ParseFrontmatter(tt.source)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if fm != nil {
				t.Errorf("expected nil frontmatter, got %+v", fm)
			}
			if remaining != tt.source {
				t.Errorf("expected source unchanged, got %q", remaining)
			}
		})
	}
}

func TestParseFrontmatter_ValidExchangeRates(t *testing.T) {
	source := `---
exchange:
  USD_EUR: 0.92
  EUR_GBP: 0.86
  GBP_USD: 1.27
---
# My Budget

price = 100 USD
`
	fm, remaining, err := ParseFrontmatter(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm == nil {
		t.Fatal("expected frontmatter, got nil")
	}

	// Check exchange rates
	expectedRates := map[string]float64{
		"USD_EUR": 0.92,
		"EUR_GBP": 0.86,
		"GBP_USD": 1.27,
	}
	if len(fm.Exchange) != len(expectedRates) {
		t.Errorf("expected %d exchange rates, got %d", len(expectedRates), len(fm.Exchange))
	}
	for key, expected := range expectedRates {
		got, ok := fm.Exchange[key]
		if !ok {
			t.Errorf("missing exchange rate for %s", key)
			continue
		}
		if !got.Equal(decimal.NewFromFloat(expected)) {
			t.Errorf("exchange rate %s: expected %v, got %v", key, expected, got)
		}
	}

	// Check remaining content
	expectedRemaining := `# My Budget

price = 100 USD
`
	if remaining != expectedRemaining {
		t.Errorf("remaining content mismatch:\nexpected: %q\ngot: %q", expectedRemaining, remaining)
	}
}

func TestParseFrontmatter_EmptyExchange(t *testing.T) {
	source := `---
exchange:
---
x = 10`
	fm, remaining, err := ParseFrontmatter(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm == nil {
		t.Fatal("expected frontmatter, got nil")
	}
	if len(fm.Exchange) != 0 {
		t.Errorf("expected empty exchange map, got %d entries", len(fm.Exchange))
	}
	if remaining != "x = 10" {
		t.Errorf("unexpected remaining: %q", remaining)
	}
}

func TestParseFrontmatter_UnclosedDelimiter(t *testing.T) {
	source := `---
exchange:
  USD_EUR: 0.92
x = 10`
	_, _, err := ParseFrontmatter(source)
	if err == nil {
		t.Error("expected error for unclosed frontmatter")
	}
	if err != nil && err.Error() != "frontmatter not closed: missing closing '---' delimiter" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestParseFrontmatter_InvalidYAML(t *testing.T) {
	source := `---
exchange:
  USD_EUR: not_a_number
---
x = 10`
	_, _, err := ParseFrontmatter(source)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestParseFrontmatter_InvalidExchangeKey(t *testing.T) {
	tests := []struct {
		name   string
		source string
		errMsg string
	}{
		{
			name: "missing underscore",
			source: `---
exchange:
  USDEUR: 0.92
---`,
			errMsg: "invalid exchange rate key 'USDEUR': expected format 'FROM_TO'",
		},
		{
			name: "empty from",
			source: `---
exchange:
  _EUR: 0.92
---`,
			errMsg: "invalid exchange rate key '_EUR': currency codes cannot be empty",
		},
		{
			name: "empty to",
			source: `---
exchange:
  USD_: 0.92
---`,
			errMsg: "invalid exchange rate key 'USD_': currency codes cannot be empty",
		},
		{
			name: "too many underscores",
			source: `---
exchange:
  USD_EUR_GBP: 0.92
---`,
			errMsg: "invalid exchange rate key 'USD_EUR_GBP': expected format 'FROM_TO'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseFrontmatter(tt.source)
			if err == nil {
				t.Error("expected error")
				return
			}
			if !contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestParseFrontmatter_CaseInsensitiveKeys(t *testing.T) {
	source := `---
exchange:
  usd_eur: 0.92
  Eur_Gbp: 0.86
---`
	fm, _, err := ParseFrontmatter(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Keys should be normalized to uppercase
	if _, ok := fm.Exchange["USD_EUR"]; !ok {
		t.Error("expected USD_EUR key (normalized from usd_eur)")
	}
	if _, ok := fm.Exchange["EUR_GBP"]; !ok {
		t.Error("expected EUR_GBP key (normalized from Eur_Gbp)")
	}
}

func TestParseFrontmatter_NoRemainingContent(t *testing.T) {
	source := `---
exchange:
  USD_EUR: 0.92
---`
	fm, remaining, err := ParseFrontmatter(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm == nil {
		t.Fatal("expected frontmatter")
	}
	if remaining != "" {
		t.Errorf("expected empty remaining, got %q", remaining)
	}
}

func TestGetExchangeRate(t *testing.T) {
	fm := &Frontmatter{
		Exchange: map[string]decimal.Decimal{
			"USD_EUR": decimal.NewFromFloat(0.92),
			"EUR_GBP": decimal.NewFromFloat(0.86),
		},
	}

	tests := []struct {
		from     string
		to       string
		expected float64
		found    bool
	}{
		{"USD", "EUR", 0.92, true},
		{"EUR", "GBP", 0.86, true},
		{"USD", "GBP", 0, false},   // Not defined
		{"EUR", "USD", 0, false},   // Inverse not auto-computed
		{"usd", "eur", 0.92, true}, // Case insensitive lookup
		{"JPY", "USD", 0, false},   // Not defined
	}

	for _, tt := range tests {
		t.Run(tt.from+"/"+tt.to, func(t *testing.T) {
			rate, found := fm.GetExchangeRate(tt.from, tt.to)
			if found != tt.found {
				t.Errorf("found: expected %v, got %v", tt.found, found)
			}
			if tt.found && !rate.Equal(decimal.NewFromFloat(tt.expected)) {
				t.Errorf("rate: expected %v, got %v", tt.expected, rate)
			}
		})
	}
}

func TestGetExchangeRate_NilFrontmatter(t *testing.T) {
	var fm *Frontmatter
	rate, found := fm.GetExchangeRate("USD", "EUR")
	if found {
		t.Error("expected not found for nil frontmatter")
	}
	if !rate.IsZero() {
		t.Error("expected zero rate")
	}
}

func TestExchangeRateKey(t *testing.T) {
	tests := []struct {
		from     string
		to       string
		expected string
	}{
		{"USD", "EUR", "USD_EUR"},
		{"usd", "eur", "USD_EUR"},
		{"Usd", "Eur", "USD_EUR"},
	}

	for _, tt := range tests {
		got := ExchangeRateKey(tt.from, tt.to)
		if got != tt.expected {
			t.Errorf("ExchangeRateKey(%q, %q) = %q, want %q", tt.from, tt.to, got, tt.expected)
		}
	}
}

func TestParseExchangeRateKey(t *testing.T) {
	tests := []struct {
		key      string
		from     string
		to       string
		hasError bool
	}{
		{"USD_EUR", "USD", "EUR", false},
		{"usd_eur", "USD", "EUR", false},
		{"USDEUR", "", "", true},
		{"_EUR", "", "", true},
		{"USD_", "", "", true},
		{"A_B_C", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			from, to, err := ParseExchangeRateKey(tt.key)
			if tt.hasError {
				if err == nil {
					t.Error("expected error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if from != tt.from {
					t.Errorf("from: expected %q, got %q", tt.from, from)
				}
				if to != tt.to {
					t.Errorf("to: expected %q, got %q", tt.to, to)
				}
			}
		})
	}
}

func TestParseFrontmatter_ValidGlobals(t *testing.T) {
	source := `---
globals:
  base_date: Jan 15 2025
  tax_rate: 0.32
  price: $100
---
x = price * tax_rate
`
	fm, remaining, err := ParseFrontmatter(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm == nil {
		t.Fatal("expected frontmatter, got nil")
	}

	// Check globals (raw strings)
	expectedGlobals := map[string]string{
		"base_date": "Jan 15 2025",
		"tax_rate":  "0.32",
		"price":     "$100",
	}
	if len(fm.Globals) != len(expectedGlobals) {
		t.Errorf("expected %d globals, got %d", len(expectedGlobals), len(fm.Globals))
	}
	for key, expected := range expectedGlobals {
		got, ok := fm.Globals[key]
		if !ok {
			t.Errorf("missing global %q", key)
			continue
		}
		if got != expected {
			t.Errorf("global %q: expected %q, got %q", key, expected, got)
		}
	}

	// Check remaining content
	expectedRemaining := `x = price * tax_rate
`
	if remaining != expectedRemaining {
		t.Errorf("remaining content mismatch:\nexpected: %q\ngot: %q", expectedRemaining, remaining)
	}
}

func TestParseFrontmatter_ExchangeAndGlobals(t *testing.T) {
	source := `---
exchange:
  USD_EUR: 0.92
globals:
  base_price: 100 USD
---
price_eur = base_price in EUR
`
	fm, _, err := ParseFrontmatter(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm == nil {
		t.Fatal("expected frontmatter, got nil")
	}

	// Check exchange rate
	if len(fm.Exchange) != 1 {
		t.Errorf("expected 1 exchange rate, got %d", len(fm.Exchange))
	}

	// Check globals
	if len(fm.Globals) != 1 {
		t.Errorf("expected 1 global, got %d", len(fm.Globals))
	}
	if fm.Globals["base_price"] != "100 USD" {
		t.Errorf("unexpected global value: %q", fm.Globals["base_price"])
	}
}

func TestParseFrontmatter_UnknownKey(t *testing.T) {
	tests := []struct {
		name   string
		source string
		errMsg string
	}{
		{
			name: "unknown top-level key",
			source: `---
base_date: Jan 15 2025
---`,
			errMsg: "unknown frontmatter key 'base_date'; user variables must go under 'globals:'",
		},
		{
			name: "mixed known and unknown",
			source: `---
exchange:
  USD_EUR: 0.92
my_var: 42
---`,
			errMsg: "unknown frontmatter key 'my_var'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseFrontmatter(tt.source)
			if err == nil {
				t.Error("expected error for unknown key")
				return
			}
			if !contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

func TestParseFrontmatter_InvalidGlobalName(t *testing.T) {
	tests := []struct {
		name   string
		source string
		errMsg string
	}{
		{
			name: "starts with digit",
			source: `---
globals:
  1invalid: 42
---`,
			errMsg: "invalid global variable name '1invalid'",
		},
		{
			name: "contains special char",
			source: `---
globals:
  my-var: 42
---`,
			errMsg: "invalid global variable name 'my-var'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseFrontmatter(tt.source)
			if err == nil {
				t.Error("expected error for invalid global name")
				return
			}
			if !contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

func TestFrontmatter_Serialize_Nil(t *testing.T) {
	var fm *Frontmatter
	result := fm.Serialize()
	if result != "" {
		t.Errorf("expected empty string for nil frontmatter, got %q", result)
	}
}

func TestFrontmatter_Serialize_Empty(t *testing.T) {
	fm := &Frontmatter{
		Exchange: make(map[string]decimal.Decimal),
		Globals:  make(map[string]string),
	}
	result := fm.Serialize()
	if result != "" {
		t.Errorf("expected empty string for empty frontmatter, got %q", result)
	}
}

func TestFrontmatter_Serialize_ExchangeOnly(t *testing.T) {
	fm := &Frontmatter{
		Exchange: map[string]decimal.Decimal{
			"USD_EUR": decimal.NewFromFloat(0.92),
		},
		Globals: make(map[string]string),
	}
	result := fm.Serialize()

	// Verify structure
	if !contains(result, "---\n") {
		t.Error("missing opening delimiter")
	}
	if !contains(result, "exchange:\n") {
		t.Error("missing exchange section")
	}
	if !contains(result, "USD_EUR: 0.92") {
		t.Error("missing exchange rate")
	}
	// Should not have globals section
	if contains(result, "globals:") {
		t.Error("unexpected globals section")
	}
}

func TestFrontmatter_Serialize_GlobalsOnly(t *testing.T) {
	fm := &Frontmatter{
		Exchange: make(map[string]decimal.Decimal),
		Globals: map[string]string{
			"tax_rate": "0.32",
		},
	}
	result := fm.Serialize()

	if !contains(result, "---\n") {
		t.Error("missing opening delimiter")
	}
	if !contains(result, "globals:\n") {
		t.Error("missing globals section")
	}
	if !contains(result, "tax_rate: 0.32") {
		t.Error("missing global variable")
	}
	// Should not have exchange section
	if contains(result, "exchange:") {
		t.Error("unexpected exchange section")
	}
}

func TestFrontmatter_Serialize_Full(t *testing.T) {
	fm := &Frontmatter{
		Exchange: map[string]decimal.Decimal{
			"USD_EUR": decimal.NewFromFloat(0.92),
		},
		Globals: map[string]string{
			"tax_rate": "0.32",
		},
	}
	result := fm.Serialize()

	if !contains(result, "exchange:\n") {
		t.Error("missing exchange section")
	}
	if !contains(result, "USD_EUR: 0.92") {
		t.Error("missing exchange rate")
	}
	if !contains(result, "globals:\n") {
		t.Error("missing globals section")
	}
	if !contains(result, "tax_rate: 0.32") {
		t.Error("missing global variable")
	}
}

func TestFrontmatter_Serialize_RoundTrip(t *testing.T) {
	// Create frontmatter
	fm := &Frontmatter{
		Exchange: map[string]decimal.Decimal{
			"USD_EUR": decimal.NewFromFloat(0.92),
		},
		Globals: map[string]string{
			"tax_rate": "0.32",
		},
	}

	// Serialize it
	serialized := fm.Serialize()

	// Parse it back
	parsed, remaining, err := ParseFrontmatter(serialized)
	if err != nil {
		t.Fatalf("failed to parse serialized frontmatter: %v", err)
	}
	// Remaining should be empty or just whitespace (blank line for CommonMark compatibility)
	if strings.TrimSpace(remaining) != "" {
		t.Errorf("expected no remaining content (besides whitespace), got %q", remaining)
	}

	// Verify exchange rates
	rate, ok := parsed.Exchange["USD_EUR"]
	if !ok {
		t.Error("missing USD_EUR in parsed frontmatter")
	} else if !rate.Equal(decimal.NewFromFloat(0.92)) {
		t.Errorf("USD_EUR rate: expected 0.92, got %v", rate)
	}

	// Verify globals
	if parsed.Globals["tax_rate"] != "0.32" {
		t.Errorf("tax_rate: expected 0.32, got %q", parsed.Globals["tax_rate"])
	}
}
