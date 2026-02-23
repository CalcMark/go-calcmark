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
		// Currency code format validation (must be exactly 3 letters)
		{"AB_EUR", "", "", true},   // Too short
		{"ABCD_EUR", "", "", true}, // Too long
		{"USD_AB", "", "", true},   // Too short on right
		{"USD_ABCD", "", "", true}, // Too long on right
		{"12_EUR", "", "", true},   // Digits, not letters
		{"USD_34", "", "", true},   // Digits on right
		{"U1D_EUR", "", "", true},  // Mixed digits and letters
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

// TestFrontmatter_OrderPreservation verifies that globals and exchange rates
// are iterated in document order (YAML source order), not random map order.
// This is critical because frontmatter variables must be processed deterministically.
func TestFrontmatter_OrderPreservation(t *testing.T) {
	source := `---
globals:
  zebra: 1
  alpha: 2
  middle: 3
exchange:
  JPY_USD: 0.0067
  EUR_USD: 1.08
  GBP_USD: 1.27
---
x = alpha`

	fm, _, err := ParseFrontmatter(source)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify GlobalKeys returns YAML document order, not sorted order
	expectedGlobalKeys := []string{"zebra", "alpha", "middle"}
	gotGlobalKeys := fm.GlobalKeys()
	if len(gotGlobalKeys) != len(expectedGlobalKeys) {
		t.Fatalf("GlobalKeys length: got %d, want %d", len(gotGlobalKeys), len(expectedGlobalKeys))
	}
	for i, want := range expectedGlobalKeys {
		if gotGlobalKeys[i] != want {
			t.Errorf("GlobalKeys[%d]: got %q, want %q", i, gotGlobalKeys[i], want)
		}
	}

	// Verify ExchangeKeys returns YAML document order
	expectedExchangeKeys := []string{"JPY_USD", "EUR_USD", "GBP_USD"}
	gotExchangeKeys := fm.ExchangeKeys()
	if len(gotExchangeKeys) != len(expectedExchangeKeys) {
		t.Fatalf("ExchangeKeys length: got %d, want %d", len(gotExchangeKeys), len(expectedExchangeKeys))
	}
	for i, want := range expectedExchangeKeys {
		if gotExchangeKeys[i] != want {
			t.Errorf("ExchangeKeys[%d]: got %q, want %q", i, gotExchangeKeys[i], want)
		}
	}

	// Verify SetGlobal preserves insertion order for new keys
	fm.SetGlobal("new_var", "99")
	gotGlobalKeys = fm.GlobalKeys()
	expectedGlobalKeys = append(expectedGlobalKeys, "new_var")
	if len(gotGlobalKeys) != len(expectedGlobalKeys) {
		t.Fatalf("GlobalKeys after SetGlobal: got %d, want %d", len(gotGlobalKeys), len(expectedGlobalKeys))
	}
	for i, want := range expectedGlobalKeys {
		if gotGlobalKeys[i] != want {
			t.Errorf("GlobalKeys[%d] after SetGlobal: got %q, want %q", i, gotGlobalKeys[i], want)
		}
	}

	// Verify SetGlobal for existing key doesn't duplicate
	fm.SetGlobal("alpha", "999")
	gotGlobalKeys = fm.GlobalKeys()
	if len(gotGlobalKeys) != len(expectedGlobalKeys) {
		t.Fatalf("GlobalKeys after update: got %d, want %d (should not add duplicate)", len(gotGlobalKeys), len(expectedGlobalKeys))
	}

	// Run 100 times to confirm determinism (Go maps randomize order per-run)
	for i := range 100 {
		fm2, _, err := ParseFrontmatter(source)
		if err != nil {
			t.Fatalf("Parse failed on iteration %d: %v", i, err)
		}
		keys := fm2.GlobalKeys()
		for j, want := range []string{"zebra", "alpha", "middle"} {
			if keys[j] != want {
				t.Fatalf("Iteration %d: GlobalKeys[%d] = %q, want %q", i, j, keys[j], want)
			}
		}
	}
}

// TestFrontmatter_RawSourcePreservation verifies that Serialize() returns the
// exact raw text from parsing, preserving user formatting like empty lines.
func TestFrontmatter_RawSourcePreservation(t *testing.T) {
	tests := []struct {
		name   string
		source string
		// wantLines are the frontmatter lines (after TrimRight + Split), excluding block content.
		wantLines []string
	}{
		{
			name: "standard formatting",
			source: `---
globals:
  my_var: 42
---
x = 10
`,
			wantLines: []string{"---", "globals:", "  my_var: 42", "---"},
		},
		{
			name: "extra whitespace in values",
			source: `---
globals:
  my_var:    42
---
x = 10
`,
			wantLines: []string{"---", "globals:", "  my_var:    42", "---"},
		},
		{
			name: "empty line between entries",
			source: `---
globals:
  a: 1

  b: 2
---
x = 10
`,
			wantLines: []string{"---", "globals:", "  a: 1", "", "  b: 2", "---"},
		},
		{
			name: "empty line before closing delimiter",
			source: `---
globals:
  my_var: 42

---
x = 10
`,
			wantLines: []string{"---", "globals:", "  my_var: 42", "", "---"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, _, err := ParseFrontmatter(tt.source)
			if err != nil {
				t.Fatalf("ParseFrontmatter failed: %v", err)
			}
			if fm == nil {
				t.Fatal("expected non-nil frontmatter")
			}

			serialized := fm.Serialize()
			gotLines := strings.Split(strings.TrimRight(serialized, "\n"), "\n")

			if len(gotLines) != len(tt.wantLines) {
				t.Errorf("line count: got %d, want %d\ngot:  %q\nwant: %q",
					len(gotLines), len(tt.wantLines), gotLines, tt.wantLines)
				return
			}
			for i := range gotLines {
				if gotLines[i] != tt.wantLines[i] {
					t.Errorf("line %d: got %q, want %q", i, gotLines[i], tt.wantLines[i])
				}
			}
		})
	}
}

// TestFrontmatter_RawSourceClearedOnModification verifies that programmatic
// changes (SetGlobal, SetExchangeRate) clear rawSource so Serialize reconstructs.
func TestFrontmatter_RawSourceClearedOnModification(t *testing.T) {
	source := "---\nglobals:\n  my_var:    42\n---\nx = 10\n"
	fm, _, err := ParseFrontmatter(source)
	if err != nil {
		t.Fatalf("ParseFrontmatter failed: %v", err)
	}

	// Before modification: raw formatting preserved (extra spaces)
	serialized := fm.Serialize()
	if !strings.Contains(serialized, "my_var:    42") {
		t.Errorf("before SetGlobal, expected raw formatting preserved, got %q", serialized)
	}

	// After SetGlobal: rawSource cleared, Serialize reconstructs with normalized formatting
	fm.SetGlobal("new_var", "99")
	serialized = fm.Serialize()
	if strings.Contains(serialized, "my_var:    42") {
		t.Errorf("after SetGlobal, expected reconstructed formatting, got %q", serialized)
	}
	if !strings.Contains(serialized, "my_var: 42") {
		t.Errorf("after SetGlobal, expected 'my_var: 42' in output, got %q", serialized)
	}
	if !strings.Contains(serialized, "new_var: 99") {
		t.Errorf("after SetGlobal, expected 'new_var: 99' in output, got %q", serialized)
	}
}

// TestFrontmatter_RawSourceRoundTrip verifies that parse→serialize→parse
// produces identical frontmatter even with non-standard formatting.
func TestFrontmatter_RawSourceRoundTrip(t *testing.T) {
	source := "---\nglobals:\n  a: 1\n\n  b: 2\n---\nx = a + b\n"

	// First parse
	fm1, _, err := ParseFrontmatter(source)
	if err != nil {
		t.Fatalf("first parse failed: %v", err)
	}

	// Serialize (should use rawSource)
	serialized1 := fm1.Serialize()

	// Second parse (from serialized output only)
	fm2, _, err := ParseFrontmatter(serialized1)
	if err != nil {
		t.Fatalf("second parse failed: %v", err)
	}

	// Globals should match
	if fm1.Globals["a"] != fm2.Globals["a"] || fm1.Globals["b"] != fm2.Globals["b"] {
		t.Errorf("globals mismatch:\n  first:  %v\n  second: %v", fm1.Globals, fm2.Globals)
	}

	// Serialized output should be identical (raw text preserved through round-trip)
	serialized2 := fm2.Serialize()
	if serialized1 != serialized2 {
		t.Errorf("serialized output changed through round-trip:\n  first:  %q\n  second: %q", serialized1, serialized2)
	}
}

func TestParseFrontmatter_ErrorWithLineNumber(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		wantContain string // Error message should contain this
	}{
		{
			name: "unclosed bracket syntax error",
			source: `---
globals:
  tax_rate: [unclosed
---
`,
			wantContain: "line", // yaml error includes line number
		},
		{
			name: "type mismatch",
			source: `---
globals:
  tax_rate:
    nested: value
---
`,
			wantContain: "line", // TypeError includes line number
		},
		{
			name: "invalid yaml colon",
			source: `---
globals:
  bad key: value: extra
---
`,
			wantContain: "line", // syntax error includes line number
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseFrontmatter(tt.source)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			errMsg := err.Error()
			t.Logf("Error message: %s", errMsg)

			if !strings.Contains(errMsg, tt.wantContain) {
				t.Errorf("error message should contain %q, got %q", tt.wantContain, errMsg)
			}

			// Verify the error message starts with our prefix
			if !strings.HasPrefix(errMsg, "frontmatter YAML error") {
				t.Errorf("error message should start with 'frontmatter YAML error', got %q", errMsg)
			}
		})
	}
}
