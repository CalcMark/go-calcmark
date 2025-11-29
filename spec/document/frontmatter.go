// Package document provides document structure and parsing for CalcMark.
package document

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
	"gopkg.in/yaml.v3"
)

// Frontmatter represents structured metadata at the start of a CalcMark document.
// It is delimited by --- markers and contains YAML content.
//
// Example:
//
//	---
//	exchange:
//	  USD/EUR: 0.92
//	  EUR/GBP: 0.86
//	---
type Frontmatter struct {
	// Exchange contains currency exchange rates as "FROM/TO" -> rate.
	// Example: "USD/EUR" -> 0.92 means 1 USD = 0.92 EUR
	Exchange map[string]decimal.Decimal
}

// ExchangeRateKey creates a normalized key for looking up exchange rates.
// Format: "FROM/TO" (e.g., "USD/EUR").
func ExchangeRateKey(from, to string) string {
	return strings.ToUpper(from) + "/" + strings.ToUpper(to)
}

// ParseExchangeRateKey splits a key like "USD/EUR" into (from, to) parts.
// Returns an error if the key format is invalid.
func ParseExchangeRateKey(key string) (from, to string, err error) {
	parts := strings.Split(key, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid exchange rate key '%s': expected format 'FROM/TO' (e.g., 'USD/EUR')", key)
	}
	from = strings.TrimSpace(strings.ToUpper(parts[0]))
	to = strings.TrimSpace(strings.ToUpper(parts[1]))
	if from == "" || to == "" {
		return "", "", fmt.Errorf("invalid exchange rate key '%s': currency codes cannot be empty", key)
	}
	return from, to, nil
}

// GetExchangeRate looks up the rate to convert from one currency to another.
// Returns the rate and true if found, or zero and false if not defined.
func (f *Frontmatter) GetExchangeRate(from, to string) (decimal.Decimal, bool) {
	if f == nil || f.Exchange == nil {
		return decimal.Zero, false
	}
	key := ExchangeRateKey(from, to)
	rate, ok := f.Exchange[key]
	return rate, ok
}

// frontmatterYAML is the intermediate struct for YAML unmarshaling.
// This keeps the YAML structure separate from the normalized Frontmatter type.
type frontmatterYAML struct {
	Exchange map[string]float64 `yaml:"exchange"`
}

// ParseFrontmatter extracts YAML frontmatter from the beginning of a document.
// Returns the parsed frontmatter, the remaining source (without frontmatter), and any error.
//
// Frontmatter must:
//   - Start at line 1 with exactly "---"
//   - End with a line containing exactly "---"
//   - Contain valid YAML between the delimiters
//
// If no frontmatter is present, returns (nil, source, nil).
func ParseFrontmatter(source string) (*Frontmatter, string, error) {
	lines := strings.Split(source, "\n")
	if len(lines) == 0 {
		return nil, source, nil
	}

	// Must start with exactly "---"
	if strings.TrimSpace(lines[0]) != "---" {
		return nil, source, nil
	}

	// Find closing delimiter
	closeIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			closeIdx = i
			break
		}
	}

	if closeIdx == -1 {
		return nil, "", fmt.Errorf("frontmatter not closed: missing closing '---' delimiter")
	}

	// Extract YAML content (between the delimiters)
	yamlContent := strings.Join(lines[1:closeIdx], "\n")

	// Parse YAML
	var raw frontmatterYAML
	if err := yaml.Unmarshal([]byte(yamlContent), &raw); err != nil {
		return nil, "", fmt.Errorf("invalid frontmatter YAML: %w", err)
	}

	// Convert to Frontmatter with decimal values
	fm := &Frontmatter{
		Exchange: make(map[string]decimal.Decimal),
	}

	for key, rate := range raw.Exchange {
		// Validate key format
		from, to, err := ParseExchangeRateKey(key)
		if err != nil {
			return nil, "", err
		}
		// Normalize key and store rate
		normalizedKey := ExchangeRateKey(from, to)
		fm.Exchange[normalizedKey] = decimal.NewFromFloat(rate)
	}

	// Calculate remaining source (after closing delimiter)
	remaining := ""
	if closeIdx+1 < len(lines) {
		remaining = strings.Join(lines[closeIdx+1:], "\n")
	}

	return fm, remaining, nil
}
