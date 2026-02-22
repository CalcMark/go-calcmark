// Package document provides document structure and parsing for CalcMark.
package document

import (
	"fmt"
	"sort"
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

	// exchangeKeys preserves insertion order of exchange rate keys.
	// Go maps have non-deterministic iteration order; frontmatter variables
	// must be processed in document order (they are *front* matter).
	exchangeKeys []string

	// globalKeys preserves insertion order of global variable keys.
	globalKeys []string

	// rawSource preserves the exact frontmatter text (including --- delimiters)
	// as it appeared in the original document. This allows Serialize() to return
	// the user's formatting rather than reconstructed YAML, enabling natural
	// editing of frontmatter lines (Enter, whitespace, etc.) in the TUI editor.
	// Cleared on programmatic modification (SetGlobal, SetExchangeRate).
	rawSource string
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
	normalized := strings.ToUpper(key)
	if _, exists := f.Exchange[normalized]; !exists {
		f.exchangeKeys = append(f.exchangeKeys, normalized)
	}
	f.Exchange[normalized] = rate
	f.rawSource = "" // Invalidate raw source — Serialize() will reconstruct
}

// SetGlobal sets a global variable value. The valueExpr is stored as the
// raw expression string for serialization.
func (f *Frontmatter) SetGlobal(name, valueExpr string) {
	if f.Globals == nil {
		f.Globals = make(map[string]string)
	}
	if _, exists := f.Globals[name]; !exists {
		f.globalKeys = append(f.globalKeys, name)
	}
	f.Globals[name] = valueExpr
	f.rawSource = "" // Invalidate raw source — Serialize() will reconstruct
}

// GlobalKeys returns global variable names in insertion order.
// Falls back to sorted keys for backward compatibility when frontmatter
// is created via struct literals rather than ParseFrontmatter or SetGlobal.
func (f *Frontmatter) GlobalKeys() []string {
	if f == nil {
		return nil
	}
	if len(f.globalKeys) > 0 {
		return f.globalKeys
	}
	// Fallback: sorted order for determinism
	keys := make([]string, 0, len(f.Globals))
	for k := range f.Globals {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ExchangeKeys returns exchange rate keys in insertion order.
// Falls back to sorted keys for backward compatibility when frontmatter
// is created via struct literals rather than ParseFrontmatter or SetExchangeRate.
func (f *Frontmatter) ExchangeKeys() []string {
	if f == nil {
		return nil
	}
	if len(f.exchangeKeys) > 0 {
		return f.exchangeKeys
	}
	// Fallback: sorted order for determinism
	keys := make([]string, 0, len(f.Exchange))
	for k := range f.Exchange {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
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
		return nil, "", formatYAMLError(err)
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
		return nil, "", formatYAMLError(err)
	}

	// Convert to Frontmatter with decimal values
	fm := &Frontmatter{
		Exchange:  make(map[string]decimal.Decimal),
		Globals:   make(map[string]string),
		rawSource: strings.Join(lines[0:closeIdx+1], "\n") + "\n",
	}

	// Process exchange rates — use YAML node order for deterministic ordering.
	// yaml.v3 preserves key order when unmarshaling into a map, but Go map
	// iteration is non-deterministic. We extract order from the YAML AST.
	exchangeOrder := extractYAMLKeyOrder(yamlContent, "exchange")
	for _, key := range exchangeOrder {
		rate, ok := raw.Exchange[key]
		if !ok {
			continue
		}
		from, to, err := ParseExchangeRateKey(key)
		if err != nil {
			return nil, "", err
		}
		normalizedKey := ExchangeRateKey(from, to)
		fm.Exchange[normalizedKey] = decimal.NewFromFloat(rate)
		fm.exchangeKeys = append(fm.exchangeKeys, normalizedKey)
	}
	// Handle any keys not in YAML order (defensive)
	for key, rate := range raw.Exchange {
		from, to, err := ParseExchangeRateKey(key)
		if err != nil {
			return nil, "", err
		}
		normalizedKey := ExchangeRateKey(from, to)
		if _, exists := fm.Exchange[normalizedKey]; !exists {
			fm.Exchange[normalizedKey] = decimal.NewFromFloat(rate)
			fm.exchangeKeys = append(fm.exchangeKeys, normalizedKey)
		}
	}

	// Copy globals — use YAML node order for deterministic ordering.
	globalsOrder := extractYAMLKeyOrder(yamlContent, "globals")
	for _, name := range globalsOrder {
		expr, ok := raw.Globals[name]
		if !ok {
			continue
		}
		if !isValidIdentifier(name) {
			return nil, "", fmt.Errorf("invalid global variable name '%s': must be a valid identifier", name)
		}
		fm.Globals[name] = expr
		fm.globalKeys = append(fm.globalKeys, name)
	}
	// Handle any keys not in YAML order (defensive)
	for name, expr := range raw.Globals {
		if _, exists := fm.Globals[name]; !exists {
			if !isValidIdentifier(name) {
				return nil, "", fmt.Errorf("invalid global variable name '%s': must be a valid identifier", name)
			}
			fm.Globals[name] = expr
			fm.globalKeys = append(fm.globalKeys, name)
		}
	}

	// Calculate remaining source (after closing delimiter)
	remaining := ""
	if closeIdx+1 < len(lines) {
		remaining = strings.Join(lines[closeIdx+1:], "\n")
	}

	return fm, remaining, nil
}

// extractYAMLKeyOrder extracts the key ordering for a nested map under a
// top-level key from YAML content. Uses yaml.v3's Node API which preserves
// document order. Returns keys in the order they appear in the YAML source.
func extractYAMLKeyOrder(yamlContent string, topKey string) []string {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(yamlContent), &root); err != nil {
		return nil
	}

	// root is a Document node containing a Mapping node
	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		return nil
	}
	mapping := root.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return nil
	}

	// Find the top-level key (e.g., "exchange" or "globals")
	for i := 0; i < len(mapping.Content)-1; i += 2 {
		keyNode := mapping.Content[i]
		valueNode := mapping.Content[i+1]
		if keyNode.Value == topKey && valueNode.Kind == yaml.MappingNode {
			// Extract keys in document order
			var keys []string
			for j := 0; j < len(valueNode.Content)-1; j += 2 {
				keys = append(keys, valueNode.Content[j].Value)
			}
			return keys
		}
	}
	return nil
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
// When rawSource is available (from parsing), the original text is preserved
// to support natural editing (Enter, whitespace, etc.) in the TUI editor.
func (f *Frontmatter) Serialize() string {
	if f == nil {
		return ""
	}
	if len(f.Exchange) == 0 && len(f.Globals) == 0 {
		return ""
	}

	// Use raw source if available (preserves user formatting)
	if f.rawSource != "" {
		return f.rawSource + "\n" // Add CommonMark blank line
	}

	// Fall back to reconstruction (programmatically created frontmatter)
	var sb strings.Builder
	sb.WriteString("---\n")

	// Serialize exchange rates in insertion order (falls back to sorted)
	if len(f.Exchange) > 0 {
		sb.WriteString("exchange:\n")
		for _, key := range f.ExchangeKeys() {
			if rate, ok := f.Exchange[key]; ok {
				sb.WriteString(fmt.Sprintf("  %s: %s\n", key, rate.String()))
			}
		}
	}

	// Serialize globals in insertion order (falls back to sorted)
	if len(f.Globals) > 0 {
		sb.WriteString("globals:\n")
		for _, name := range f.GlobalKeys() {
			if expr, ok := f.Globals[name]; ok {
				sb.WriteString(fmt.Sprintf("  %s: %s\n", name, expr))
			}
		}
	}

	sb.WriteString("---\n\n") // Blank line after frontmatter for CommonMark compatibility
	return sb.String()
}

// formatYAMLError extracts line numbers from yaml.v3 errors and formats them clearly.
// yaml.v3 provides two error types:
// - *yaml.TypeError: structured errors with line info in Errors slice
// - Other errors: contain "line N:" in the message string
func formatYAMLError(err error) error {
	if err == nil {
		return nil
	}

	// Check for yaml.TypeError (structured errors from type mismatches)
	if typeErr, ok := err.(*yaml.TypeError); ok {
		// TypeError.Errors contains strings like "line 2: cannot unmarshal..."
		// Join them into a single error message
		return fmt.Errorf("frontmatter YAML error:\n  %s", strings.Join(typeErr.Errors, "\n  "))
	}

	// For other yaml errors, the message already includes "yaml: line N: ..."
	// Extract and reformat for consistency
	errMsg := err.Error()

	// Check if it contains "yaml: line N:" pattern
	if strings.Contains(errMsg, "yaml: line ") {
		// The error already has line info, just reformat prefix
		return fmt.Errorf("frontmatter YAML error: %s", strings.TrimPrefix(errMsg, "yaml: "))
	}

	// Generic fallback
	return fmt.Errorf("invalid frontmatter YAML: %w", err)
}
