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
x = @globals.price * @globals.tax_rate
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
	expectedRemaining := `x = @globals.price * @globals.tax_rate
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
price_eur = @globals.base_price in EUR
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

func TestParseFrontmatter_UnknownKeysIgnored(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{
			name: "unknown top-level key silently ignored",
			source: `---
base_date: Jan 15 2025
---`,
		},
		{
			name: "standard markdown frontmatter keys",
			source: `---
title: My Document
date: 2026-03-09
tags: [finance, planning]
description: A sample document
---`,
		},
		{
			name: "mixed known and unknown keys",
			source: `---
title: Budget
exchange:
  USD_EUR: 0.92
tags: [budget]
globals:
  tax: "0.08"
---`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, _, err := ParseFrontmatter(tt.source)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if fm == nil {
				t.Error("expected non-nil frontmatter")
			}
		})
	}
}

func TestParseFrontmatter_UnknownKeysWithCalcMarkKeys(t *testing.T) {
	source := `---
title: Budget Report
exchange:
  USD_EUR: 0.92
date: 2026-03-09
globals:
  tax: "0.08"
description: Monthly budget
---`

	fm, _, err := ParseFrontmatter(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// CalcMark keys should be parsed correctly
	if len(fm.Exchange) != 1 {
		t.Errorf("expected 1 exchange rate, got %d", len(fm.Exchange))
	}
	if len(fm.Globals) != 1 {
		t.Errorf("expected 1 global, got %d", len(fm.Globals))
	}
	if fm.Globals["tax"] != "0.08" {
		t.Errorf("expected global tax=0.08, got %q", fm.Globals["tax"])
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
	source := "---\nglobals:\n  a: 1\n\n  b: 2\n---\nx = @globals.a + @globals.b\n"

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

// ============================================================
// Scale and ConvertTo frontmatter tests
// ============================================================

func TestParseFrontmatter_ScaleScalar(t *testing.T) {
	source := "---\nscale: 4\n---\nflour = 280 grams\n"
	fm, remaining, err := ParseFrontmatter(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm == nil {
		t.Fatal("expected frontmatter")
	}
	if fm.Scale == nil {
		t.Fatal("expected Scale to be set")
	}
	if !fm.Scale.Factor.Equal(decimal.NewFromInt(4)) {
		t.Errorf("expected factor 4, got %s", fm.Scale.Factor.String())
	}
	if len(fm.Scale.UnitCategories) != 0 {
		t.Errorf("expected no unit categories, got %v", fm.Scale.UnitCategories)
	}
	if remaining != "flour = 280 grams\n" {
		t.Errorf("unexpected remaining: %q", remaining)
	}
}

func TestParseFrontmatter_ScaleFloat(t *testing.T) {
	source := "---\nscale: 2.5\n---\n"
	fm, _, err := ParseFrontmatter(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm.Scale == nil {
		t.Fatal("expected Scale")
	}
	if !fm.Scale.Factor.Equal(decimal.NewFromFloat(2.5)) {
		t.Errorf("expected factor 2.5, got %s", fm.Scale.Factor.String())
	}
}

func TestParseFrontmatter_ScaleMap(t *testing.T) {
	source := "---\nscale:\n  factor: 4\n  unit_categories: [Mass, Volume]\n---\n"
	fm, _, err := ParseFrontmatter(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm.Scale == nil {
		t.Fatal("expected Scale")
	}
	if !fm.Scale.Factor.Equal(decimal.NewFromInt(4)) {
		t.Errorf("expected factor 4, got %s", fm.Scale.Factor.String())
	}
	if len(fm.Scale.UnitCategories) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(fm.Scale.UnitCategories))
	}
	if fm.Scale.UnitCategories[0] != "Mass" || fm.Scale.UnitCategories[1] != "Volume" {
		t.Errorf("unexpected categories: %v", fm.Scale.UnitCategories)
	}
}

func TestParseFrontmatter_ScaleAllCategory(t *testing.T) {
	source := "---\nscale:\n  factor: 3\n  unit_categories: [All]\n---\n"
	fm, _, err := ParseFrontmatter(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm.Scale == nil {
		t.Fatal("expected Scale")
	}
	if len(fm.Scale.UnitCategories) != 1 || fm.Scale.UnitCategories[0] != "All" {
		t.Errorf("expected [All], got %v", fm.Scale.UnitCategories)
	}
}

func TestParseFrontmatter_ScaleCustomCategory(t *testing.T) {
	source := "---\nscale:\n  factor: 2\n  unit_categories: [Mass, Custom]\n---\n"
	fm, _, err := ParseFrontmatter(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm.Scale == nil {
		t.Fatal("expected Scale")
	}
	if len(fm.Scale.UnitCategories) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(fm.Scale.UnitCategories))
	}
	if fm.Scale.UnitCategories[0] != "Mass" || fm.Scale.UnitCategories[1] != "Custom" {
		t.Errorf("expected [Mass, Custom], got %v", fm.Scale.UnitCategories)
	}
}

func TestParseFrontmatter_ScaleValidation(t *testing.T) {
	tests := []struct {
		name   string
		source string
		errMsg string
	}{
		{
			name:   "zero",
			source: "---\nscale: 0\n---\n",
			errMsg: "scale factor must be positive",
		},
		{
			name:   "negative",
			source: "---\nscale: -1\n---\n",
			errMsg: "scale factor must be positive",
		},
		{
			name:   "invalid type",
			source: "---\nscale: big\n---\n",
			errMsg: "scale must be a number",
		},
		{
			name:   "map missing factor",
			source: "---\nscale:\n  unit_categories: [Mass]\n---\n",
			errMsg: "scale map form requires 'factor' key",
		},
		{
			name:   "invalid category",
			source: "---\nscale:\n  factor: 4\n  unit_categories: [Flavor]\n---\n",
			errMsg: "invalid unit category",
		},
		{
			name:   "NaN",
			source: "---\nscale: .nan\n---\n",
			errMsg: "scale factor must be a finite number",
		},
		{
			name:   "Inf",
			source: "---\nscale: .inf\n---\n",
			errMsg: "scale factor must be a finite number",
		},
		{
			name:   "map factor NaN",
			source: "---\nscale:\n  factor: .nan\n---\n",
			errMsg: "scale.factor must be a finite number",
		},
		{
			name:   "unknown sub-key",
			source: "---\nscale:\n  factor: 4\n  bogus: true\n---\n",
			errMsg: "unknown key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseFrontmatter(tt.source)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

func TestParseFrontmatter_ConvertToScalar(t *testing.T) {
	source := "---\nconvert_to: imperial\n---\nflour = 280 grams\n"
	fm, _, err := ParseFrontmatter(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm.ConvertTo == nil {
		t.Fatal("expected ConvertTo to be set")
	}
	if fm.ConvertTo.System != "imperial" {
		t.Errorf("expected system 'imperial', got %q", fm.ConvertTo.System)
	}
	if len(fm.ConvertTo.UnitCategories) != 0 {
		t.Errorf("expected no unit categories, got %v", fm.ConvertTo.UnitCategories)
	}
}

func TestParseFrontmatter_ConvertToSI(t *testing.T) {
	source := "---\nconvert_to: si\n---\n"
	fm, _, err := ParseFrontmatter(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm.ConvertTo == nil || fm.ConvertTo.System != "si" {
		t.Fatalf("expected ConvertTo with system 'si', got %+v", fm.ConvertTo)
	}
}

func TestParseFrontmatter_ConvertToMap(t *testing.T) {
	source := "---\nconvert_to:\n  system: si\n  unit_categories: [Mass]\n---\n"
	fm, _, err := ParseFrontmatter(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm.ConvertTo == nil {
		t.Fatal("expected ConvertTo")
	}
	if fm.ConvertTo.System != "si" {
		t.Errorf("expected system 'si', got %q", fm.ConvertTo.System)
	}
	if len(fm.ConvertTo.UnitCategories) != 1 || fm.ConvertTo.UnitCategories[0] != "Mass" {
		t.Errorf("unexpected categories: %v", fm.ConvertTo.UnitCategories)
	}
}

func TestParseFrontmatter_ConvertToValidation(t *testing.T) {
	tests := []struct {
		name   string
		source string
		errMsg string
	}{
		{
			name:   "invalid system",
			source: "---\nconvert_to: metric\n---\n",
			errMsg: "convert_to system must be 'si' or 'imperial'",
		},
		{
			name:   "map missing system",
			source: "---\nconvert_to:\n  unit_categories: [Mass]\n---\n",
			errMsg: "convert_to map form requires 'system' key",
		},
		{
			name:   "invalid category",
			source: "---\nconvert_to:\n  system: imperial\n  unit_categories: [Flavor]\n---\n",
			errMsg: "invalid unit category",
		},
		{
			name:   "unknown sub-key",
			source: "---\nconvert_to:\n  system: imperial\n  bogus: true\n---\n",
			errMsg: "unknown key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseFrontmatter(tt.source)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
			}
		})
	}
}

func TestParseFrontmatter_UnitCategoryCaseInsensitive(t *testing.T) {
	source := "---\nscale:\n  factor: 2\n  unit_categories: [mass, VOLUME, Temperature]\n---\n"
	fm, _, err := ParseFrontmatter(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm.Scale == nil {
		t.Fatal("expected Scale")
	}
	expected := []string{"Mass", "Volume", "Temperature"}
	if len(fm.Scale.UnitCategories) != len(expected) {
		t.Fatalf("expected %d categories, got %d", len(expected), len(fm.Scale.UnitCategories))
	}
	for i, want := range expected {
		if fm.Scale.UnitCategories[i] != want {
			t.Errorf("category %d: got %q, want %q", i, fm.Scale.UnitCategories[i], want)
		}
	}
}

func TestParseFrontmatter_ScaleConvertToCoexist(t *testing.T) {
	source := "---\nscale: 4\nconvert_to: imperial\nexchange:\n  USD_EUR: 0.92\nglobals:\n  tax: 0.1\n---\n"
	fm, _, err := ParseFrontmatter(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm.Scale == nil || fm.ConvertTo == nil {
		t.Fatal("expected both Scale and ConvertTo")
	}
	if !fm.Scale.Factor.Equal(decimal.NewFromInt(4)) {
		t.Errorf("unexpected scale factor: %s", fm.Scale.Factor.String())
	}
	if fm.ConvertTo.System != "imperial" {
		t.Errorf("unexpected convert_to system: %s", fm.ConvertTo.System)
	}
	if len(fm.Exchange) != 1 {
		t.Errorf("expected 1 exchange rate, got %d", len(fm.Exchange))
	}
	if fm.Globals["tax"] != "0.1" {
		t.Errorf("expected global tax=0.1, got %q", fm.Globals["tax"])
	}
}

func TestFrontmatter_Serialize_ScaleScalar(t *testing.T) {
	fm := &Frontmatter{
		Exchange: make(map[string]decimal.Decimal),
		Globals:  make(map[string]string),
		Scale:    &ScaleConfig{Factor: decimal.NewFromInt(4)},
	}
	result := fm.Serialize()
	if !strings.Contains(result, "scale: 4") {
		t.Errorf("expected 'scale: 4' in output, got %q", result)
	}
}

func TestFrontmatter_Serialize_ScaleMap(t *testing.T) {
	fm := &Frontmatter{
		Exchange: make(map[string]decimal.Decimal),
		Globals:  make(map[string]string),
		Scale: &ScaleConfig{
			Factor:         decimal.NewFromInt(4),
			UnitCategories: []string{"Mass", "Volume"},
		},
	}
	result := fm.Serialize()
	if !strings.Contains(result, "factor: 4") {
		t.Errorf("expected 'factor: 4' in output, got %q", result)
	}
	if !strings.Contains(result, "unit_categories: [Mass, Volume]") {
		t.Errorf("expected categories in output, got %q", result)
	}
}

func TestFrontmatter_Serialize_ConvertTo(t *testing.T) {
	fm := &Frontmatter{
		Exchange:  make(map[string]decimal.Decimal),
		Globals:   make(map[string]string),
		ConvertTo: &ConvertToConfig{System: "imperial"},
	}
	result := fm.Serialize()
	if !strings.Contains(result, "convert_to: imperial") {
		t.Errorf("expected 'convert_to: imperial' in output, got %q", result)
	}
}

func TestFrontmatter_Serialize_ConvertToMap(t *testing.T) {
	fm := &Frontmatter{
		Exchange: make(map[string]decimal.Decimal),
		Globals:  make(map[string]string),
		ConvertTo: &ConvertToConfig{
			System:         "si",
			UnitCategories: []string{"Mass"},
		},
	}
	result := fm.Serialize()
	if !strings.Contains(result, "system: si") {
		t.Errorf("expected 'system: si' in output, got %q", result)
	}
	if !strings.Contains(result, "unit_categories: [Mass]") {
		t.Errorf("expected categories in output, got %q", result)
	}
}

func TestFrontmatter_ScaleConvertTo_RoundTrip(t *testing.T) {
	fm := &Frontmatter{
		Exchange:  make(map[string]decimal.Decimal),
		Globals:   make(map[string]string),
		Scale:     &ScaleConfig{Factor: decimal.NewFromInt(4)},
		ConvertTo: &ConvertToConfig{System: "imperial"},
	}

	serialized := fm.Serialize()
	parsed, _, err := ParseFrontmatter(serialized)
	if err != nil {
		t.Fatalf("round-trip parse failed: %v", err)
	}

	if parsed.Scale == nil || !parsed.Scale.Factor.Equal(decimal.NewFromInt(4)) {
		t.Errorf("scale lost in round-trip: %+v", parsed.Scale)
	}
	if parsed.ConvertTo == nil || parsed.ConvertTo.System != "imperial" {
		t.Errorf("convert_to lost in round-trip: %+v", parsed.ConvertTo)
	}
}

// ========== Measurement Convention Tests ==========

func TestParseFrontmatter_MeasurementFullConfig(t *testing.T) {
	source := `---
measurement:
  volume: imperial
  mass: troy
  ton: long
  strict: false
---
gold = 10 oz
`
	fm, _, err := ParseFrontmatter(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm.Measurement == nil {
		t.Fatal("expected measurement config, got nil")
	}
	if fm.Measurement.Volume != "imperial" {
		t.Errorf("volume = %q, want %q", fm.Measurement.Volume, "imperial")
	}
	if fm.Measurement.Mass != "troy" {
		t.Errorf("mass = %q, want %q", fm.Measurement.Mass, "troy")
	}
	if fm.Measurement.Ton != "long" {
		t.Errorf("ton = %q, want %q", fm.Measurement.Ton, "long")
	}
	if fm.Measurement.Strict == nil || *fm.Measurement.Strict != false {
		t.Errorf("strict = %v, want false", fm.Measurement.Strict)
	}
}

func TestParseFrontmatter_MeasurementPartialConfig(t *testing.T) {
	source := `---
measurement:
  volume: imperial
---
milk = 1 gallon
`
	fm, _, err := ParseFrontmatter(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm.Measurement == nil {
		t.Fatal("expected measurement config, got nil")
	}
	if fm.Measurement.Volume != "imperial" {
		t.Errorf("volume = %q, want %q", fm.Measurement.Volume, "imperial")
	}
	// Defaults should be filled in
	if fm.Measurement.Mass != "standard" {
		t.Errorf("mass default = %q, want %q", fm.Measurement.Mass, "standard")
	}
	if fm.Measurement.Ton != "short" {
		t.Errorf("ton default = %q, want %q", fm.Measurement.Ton, "short")
	}
	if fm.Measurement.Strict != nil {
		t.Errorf("strict should be nil (unset), got %v", *fm.Measurement.Strict)
	}
}

func TestParseFrontmatter_MeasurementInvalidVolume(t *testing.T) {
	source := `---
measurement:
  volume: metric
---
`
	_, _, err := ParseFrontmatter(source)
	if err == nil {
		t.Fatal("expected error for invalid volume convention")
	}
	if !strings.Contains(err.Error(), "unknown volume convention") {
		t.Errorf("error should mention volume convention, got: %v", err)
	}
}

func TestParseFrontmatter_MeasurementInvalidMass(t *testing.T) {
	source := `---
measurement:
  mass: avoidupois
---
`
	_, _, err := ParseFrontmatter(source)
	if err == nil {
		t.Fatal("expected error for invalid mass convention")
	}
	if !strings.Contains(err.Error(), "unknown mass convention") {
		t.Errorf("error should mention mass convention, got: %v", err)
	}
	// Error should explain what "standard" means
	if !strings.Contains(err.Error(), "everyday weight") {
		t.Errorf("error should explain standard = everyday weight, got: %v", err)
	}
}

func TestParseFrontmatter_MeasurementInvalidTon(t *testing.T) {
	source := `---
measurement:
  ton: imperial
---
`
	_, _, err := ParseFrontmatter(source)
	if err == nil {
		t.Fatal("expected error for invalid ton convention")
	}
	if !strings.Contains(err.Error(), "unknown ton convention") {
		t.Errorf("error should mention ton convention, got: %v", err)
	}
}

func TestParseFrontmatter_MeasurementUnknownKey(t *testing.T) {
	source := `---
measurement:
  volume: us
  cups: metric
---
`
	_, _, err := ParseFrontmatter(source)
	if err == nil {
		t.Fatal("expected error for unknown measurement key")
	}
	if !strings.Contains(err.Error(), "unknown key") {
		t.Errorf("error should mention unknown key, got: %v", err)
	}
}

func TestParseFrontmatter_MeasurementNotMap(t *testing.T) {
	source := `---
measurement: imperial
---
`
	_, _, err := ParseFrontmatter(source)
	if err == nil {
		t.Fatal("expected error for non-map measurement value")
	}
	if !strings.Contains(err.Error(), "must be a map") {
		t.Errorf("error should mention map requirement, got: %v", err)
	}
}

func TestParseFrontmatter_MeasurementAllTonOptions(t *testing.T) {
	for _, ton := range []string{"short", "long", "metric"} {
		t.Run(ton, func(t *testing.T) {
			source := "---\nmeasurement:\n  ton: " + ton + "\n---\n"
			fm, _, err := ParseFrontmatter(source)
			if err != nil {
				t.Fatalf("unexpected error for ton=%q: %v", ton, err)
			}
			if fm.Measurement.Ton != ton {
				t.Errorf("ton = %q, want %q", fm.Measurement.Ton, ton)
			}
		})
	}
}

func TestParseFrontmatter_MeasurementNoConfig(t *testing.T) {
	source := `---
exchange:
  USD_EUR: 0.92
---
price = 100 USD
`
	fm, _, err := ParseFrontmatter(source)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fm.Measurement != nil {
		t.Errorf("expected nil measurement config when not specified, got %+v", fm.Measurement)
	}
}
