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
// Reserved keys (CalcMark grammar):
//   - exchange: Currency conversion rates
//   - (future: precision, locale, etc.)
//
// User-defined variables go under 'globals':
//
//	---
//	exchange:
//	  USD_EUR: 0.92
//	globals:
//	  base_date: Jan 15 2025
//	  tax_rate: 0.32
//	---
type Frontmatter struct {
	// Exchange contains currency exchange rates as "FROM_TO" -> rate.
	// Example: "USD_EUR" -> 0.92 means 1 USD = 0.92 EUR
	Exchange map[string]decimal.Decimal

	// Globals contains user-defined variables as name -> expression string.
	// Values are CalcMark expressions that will be parsed and evaluated.
	// Example: "base_date" -> "Jan 15 2025", "tax_rate" -> "0.32"
	Globals map[string]string
}

// reservedKeys lists all top-level frontmatter keys reserved for CalcMark grammar.
// Unknown keys at the top level are rejected to ensure forward compatibility.
var reservedKeys = map[string]bool{
	"exchange": true,
	"globals":  true,
}

// ExchangeRateKey creates a normalized key for looking up exchange rates.
// Format: "FROM_TO" (e.g., "USD_EUR").
func ExchangeRateKey(from, to string) string {
	return strings.ToUpper(from) + "_" + strings.ToUpper(to)
}

// ParseExchangeRateKey splits a key like "USD_EUR" into (from, to) parts.
// Returns an error if the key format is invalid.
func ParseExchangeRateKey(key string) (from, to string, err error) {
	parts := strings.Split(key, "_")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid exchange rate key '%s': expected format 'FROM_TO' (e.g., 'USD_EUR')", key)
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

// SetExchangeRate sets an exchange rate. The key should be in FROM_TO format.
func (f *Frontmatter) SetExchangeRate(key string, rate decimal.Decimal) {
	if f.Exchange == nil {
		f.Exchange = make(map[string]decimal.Decimal)
	}
	// Normalize key to uppercase
	f.Exchange[strings.ToUpper(key)] = rate
}

// SetGlobal sets a global variable value. The valueExpr is stored as the
// raw expression string for serialization.
func (f *Frontmatter) SetGlobal(name, valueExpr string) {
	if f.Globals == nil {
		f.Globals = make(map[string]string)
	}
	f.Globals[name] = valueExpr
}

// HasGlobal returns true if the global variable is defined in frontmatter.
func (f *Frontmatter) HasGlobal(name string) bool {
	if f == nil || f.Globals == nil {
		return false
	}
	_, ok := f.Globals[name]
	return ok
}

// HasExchangeRate returns true if the exchange rate is defined in frontmatter.
func (f *Frontmatter) HasExchangeRate(key string) bool {
	if f == nil || f.Exchange == nil {
		return false
	}
	_, ok := f.Exchange[strings.ToUpper(key)]
	return ok
}

// frontmatterYAML is the intermediate struct for YAML unmarshaling.
// This keeps the YAML structure separate from the normalized Frontmatter type.
type frontmatterYAML struct {
	Exchange map[string]float64 `yaml:"exchange"`
	Globals  map[string]string  `yaml:"globals"`
}

// ParseFrontmatter extracts YAML frontmatter from the beginning of a document.
// Returns the parsed frontmatter, the remaining source (without frontmatter), and any error.
//
// Frontmatter must:
//   - Start at line 1 with exactly "---"
//   - End with a line containing exactly "---"
//   - Contain valid YAML between the delimiters
//   - Only use reserved keys at top level (exchange, globals)
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

	// First, parse into a generic map to check for unknown keys
	var rawMap map[string]any
	if err := yaml.Unmarshal([]byte(yamlContent), &rawMap); err != nil {
		return nil, "", fmt.Errorf("invalid frontmatter YAML: %w", err)
	}

	// Validate that all top-level keys are reserved
	for key := range rawMap {
		if !reservedKeys[key] {
			return nil, "", fmt.Errorf("unknown frontmatter key '%s'; user variables must go under 'globals:'", key)
		}
	}

	// Now parse into typed struct
	var raw frontmatterYAML
	if err := yaml.Unmarshal([]byte(yamlContent), &raw); err != nil {
		return nil, "", fmt.Errorf("invalid frontmatter YAML: %w", err)
	}

	// Convert to Frontmatter with decimal values
	fm := &Frontmatter{
		Exchange: make(map[string]decimal.Decimal),
		Globals:  make(map[string]string),
	}

	// Process exchange rates
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

	// Copy globals (values are raw strings to be parsed as CalcMark expressions)
	for name, expr := range raw.Globals {
		// Validate variable name (must be valid identifier)
		if !isValidIdentifier(name) {
			return nil, "", fmt.Errorf("invalid global variable name '%s': must be a valid identifier", name)
		}
		fm.Globals[name] = expr
	}

	// Calculate remaining source (after closing delimiter)
	remaining := ""
	if closeIdx+1 < len(lines) {
		remaining = strings.Join(lines[closeIdx+1:], "\n")
	}

	return fm, remaining, nil
}

// isValidIdentifier checks if a string is a valid CalcMark identifier.
// Identifiers must start with a letter or underscore and contain only
// letters, digits, and underscores.
func isValidIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if !isLetter(r) && r != '_' {
				return false
			}
		} else {
			if !isLetter(r) && !isDigit(r) && r != '_' {
				return false
			}
		}
	}
	return true
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

// Serialize returns the frontmatter as a YAML string with --- delimiters.
// If the frontmatter has no content (no exchange rates, no globals), returns "".
func (f *Frontmatter) Serialize() string {
	if f == nil {
		return ""
	}
	if len(f.Exchange) == 0 && len(f.Globals) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("---\n")

	// Serialize exchange rates
	if len(f.Exchange) > 0 {
		sb.WriteString("exchange:\n")
		for key, rate := range f.Exchange {
			// Use String() for decimal to preserve precision
			sb.WriteString(fmt.Sprintf("  %s: %s\n", key, rate.String()))
		}
	}

	// Serialize globals
	if len(f.Globals) > 0 {
		sb.WriteString("globals:\n")
		for name, expr := range f.Globals {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", name, expr))
		}
	}

	sb.WriteString("---\n\n") // Blank line after frontmatter for CommonMark compatibility
	return sb.String()
}
