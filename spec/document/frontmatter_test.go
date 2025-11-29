package document

import (
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
  USD/EUR: 0.92
  EUR/GBP: 0.86
  GBP/USD: 1.27
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
		"USD/EUR": 0.92,
		"EUR/GBP": 0.86,
		"GBP/USD": 1.27,
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
  USD/EUR: 0.92
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
  USD/EUR: not_a_number
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
			name: "missing slash",
			source: `---
exchange:
  USDEUR: 0.92
---`,
			errMsg: "invalid exchange rate key 'USDEUR': expected format 'FROM/TO'",
		},
		{
			name: "empty from",
			source: `---
exchange:
  /EUR: 0.92
---`,
			errMsg: "invalid exchange rate key '/EUR': currency codes cannot be empty",
		},
		{
			name: "empty to",
			source: `---
exchange:
  USD/: 0.92
---`,
			errMsg: "invalid exchange rate key 'USD/': currency codes cannot be empty",
		},
		{
			name: "too many slashes",
			source: `---
exchange:
  USD/EUR/GBP: 0.92
---`,
			errMsg: "invalid exchange rate key 'USD/EUR/GBP': expected format 'FROM/TO'",
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
  usd/eur: 0.92
  Eur/Gbp: 0.86
---`
	fm, _, err := ParseFrontmatter(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Keys should be normalized to uppercase
	if _, ok := fm.Exchange["USD/EUR"]; !ok {
		t.Error("expected USD/EUR key (normalized from usd/eur)")
	}
	if _, ok := fm.Exchange["EUR/GBP"]; !ok {
		t.Error("expected EUR/GBP key (normalized from Eur/Gbp)")
	}
}

func TestParseFrontmatter_NoRemainingContent(t *testing.T) {
	source := `---
exchange:
  USD/EUR: 0.92
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
			"USD/EUR": decimal.NewFromFloat(0.92),
			"EUR/GBP": decimal.NewFromFloat(0.86),
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
		{"USD", "GBP", 0, false},      // Not defined
		{"EUR", "USD", 0, false},      // Inverse not auto-computed
		{"usd", "eur", 0.92, true},    // Case insensitive lookup
		{"JPY", "USD", 0, false},      // Not defined
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
		{"USD", "EUR", "USD/EUR"},
		{"usd", "eur", "USD/EUR"},
		{"Usd", "Eur", "USD/EUR"},
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
		{"USD/EUR", "USD", "EUR", false},
		{"usd/eur", "USD", "EUR", false},
		{"USDEUR", "", "", true},
		{"/EUR", "", "", true},
		{"USD/", "", "", true},
		{"A/B/C", "", "", true},
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
