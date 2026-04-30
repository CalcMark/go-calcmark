// Package document provides document structure and parsing for CalcMark.
package document

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/CalcMark/go-calcmark/v2/spec/ast"
	"github.com/CalcMark/go-calcmark/v2/spec/units"
	"github.com/shopspring/decimal"
	"gopkg.in/yaml.v3"
)

// ScaleConfig configures the document-level scale transform.
// When present, eligible quantities are multiplied by Factor.
// UnitCategories lists the categories to scale (empty = nothing scales).
type ScaleConfig struct {
	Factor         decimal.Decimal
	UnitCategories []string // empty = nothing scales
}

// ConvertToConfig configures the document-level unit system conversion.
// When present, eligible quantities are converted to the target measurement system.
// UnitCategories restricts conversion to specific categories (empty = all).
type ConvertToConfig struct {
	System         string   // "si" or "imperial"
	UnitCategories []string // empty = all
}

// MeasurementFrontmatter wraps units.MeasurementConfig with an optional Strict flag
// that controls formatter annotation of bare ambiguous units.
type MeasurementFrontmatter struct {
	units.MeasurementConfig

	// Strict controls whether the formatter annotates bare ambiguous units in output.
	// nil = use config default (true), true = annotate, false = pass through.
	Strict *bool
}

// validVolumeConventions lists the valid options for the volume axis.
var validVolumeConventions = map[string]bool{
	"us":       true,
	"imperial": true,
}

// validMassConventions lists the valid options for the mass axis.
var validMassConventions = map[string]bool{
	"standard": true, // avoirdupois — everyday weight (1 oz = 28.35g)
	"troy":     true, // precious metals (1 troy oz = 31.10g)
}

// validTonConventions lists the valid options for the ton axis.
var validTonConventions = map[string]bool{
	"short":  true, // US short ton (2000 lb)
	"long":   true, // Imperial long ton (2240 lb)
	"metric": true, // metric tonne (1000 kg)
}

// Frontmatter represents structured metadata at the start of a CalcMark document.
// It is delimited by --- markers and contains YAML content.
//
// Reserved keys (CalcMark grammar):
//   - exchange: Currency conversion rates
//   - scale: Document-level quantity scaling
//   - convert_to: Document-level unit system conversion
//
// User-defined variables go under 'globals':
//
//	---
//	exchange:
//	  USD_EUR: 0.92
//	scale: 4
//	convert_to: imperial
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

	// Scale is the document-level scale transform (nil if not present).
	Scale *ScaleConfig

	// ConvertTo is the document-level unit system conversion (nil if not present).
	ConvertTo *ConvertToConfig

	// Measurement configures how ambiguous unit names are interpreted (nil if not present).
	Measurement *MeasurementFrontmatter

	// FiscalYearStarts configures the start of the fiscal year (nil if not set).
	// When set, fiscal expressions (FQ1, FY26, "this fiscal quarter") resolve relative
	// to this configuration. When nil, fiscal expressions produce an error diagnostic.
	FiscalYearStarts *FiscalYearConfig

	// CalendarYearOffset selects which calendar year an FY label refers
	// to. CalendarYearOffsetBefore (default) labels by the calendar
	// year the FY ENDS in (matches the Australian government year,
	// the US tax year, and most companies). CalendarYearOffsetAfter
	// labels by the calendar year the FY STARTS in. Only has effect
	// when FiscalYearStarts is non-nil and the start month is not
	// January.
	CalendarYearOffset CalendarYearOffset

	// exchangeKeys preserves insertion order of exchange rate keys.
	// Go maps have non-deterministic iteration order; frontmatter variables
	// must be processed in document order (they are *front* matter).
	exchangeKeys []string

	// globalKeys preserves insertion order of global variable keys.
	globalKeys []string

	// Extra contains non-CalcMark frontmatter keys (e.g., title, author)
	// preserved in document order. Values are the raw YAML-parsed types
	// (string, int, float64, []any, map[string]any).
	Extra []ExtraField

	// rawSource preserves the exact frontmatter text (including --- delimiters)
	// as it appeared in the original document. This allows Serialize() to return
	// the user's formatting rather than reconstructed YAML, enabling natural
	// editing of frontmatter lines (Enter, whitespace, etc.) in the TUI editor.
	// Cleared on programmatic modification (SetGlobal, SetExchangeRate).
	rawSource string

	// KeyRanges maps each top-level frontmatter key (registered or Extra) to
	// the source range covering its key+value YAML block. Lines and columns are
	// 1-based, matching ast.Position convention (see spec/ast/position.go).
	// The range starts at the key identifier and ends at the last byte of its
	// value (for mappings/sequences, the last byte of the deepest child).
	//
	// Absent keys are simply not present in the map. Use the
	// `r, ok := fm.KeyRanges[key]` idiom; a returned zero-value ast.Range from a
	// missing key is indistinguishable from one explicitly stored as zero.
	//
	// Lines are document-relative: line 1 is the opening "---" fence and the
	// first key sits on line 2 in a typical frontmatter.
	KeyRanges map[string]ast.Range
}

// ExtraField holds a non-CalcMark frontmatter key-value pair.
type ExtraField struct {
	Key   string
	Value any // string, int, float64, []any, map[string]any
}

// validUnitCategories maps lowercase category names to their canonical form.
// Derived from units.Categories() at init time so it stays in sync with the
// actual unit definitions and never drifts.
var validUnitCategories map[string]string

func init() {
	validUnitCategories = make(map[string]string)
	for _, cat := range units.Categories() {
		validUnitCategories[strings.ToLower(cat)] = cat
	}
	validUnitCategories["all"] = "All"
}

// validConvertToSystems lists the valid target systems for convert_to.
var validConvertToSystems = map[string]bool{
	"si":       true,
	"imperial": true,
}

// ExchangeRateKey creates a normalized key for looking up exchange rates.
// Format: "FROM_TO" (e.g., "USD_EUR").
func ExchangeRateKey(from, to string) string {
	return strings.ToUpper(from) + "_" + strings.ToUpper(to)
}

// ParseExchangeRateKey splits a key like "USD_EUR" into (from, to) parts.
// Returns an error if the key format is invalid. Currency codes must be
// exactly 3 ASCII letters (ISO 4217 format, e.g., USD, EUR, GBP).
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
	if !isValidCurrencyCode(from) {
		return "", "", fmt.Errorf("invalid currency code '%s' in exchange rate key '%s': must be exactly 3 letters (e.g., USD, EUR)", from, key)
	}
	if !isValidCurrencyCode(to) {
		return "", "", fmt.Errorf("invalid currency code '%s' in exchange rate key '%s': must be exactly 3 letters (e.g., USD, EUR)", to, key)
	}
	return from, to, nil
}

// isValidCurrencyCode checks if a string looks like a valid currency code:
// exactly 3 ASCII letters (already expected to be uppercase after normalization).
func isValidCurrencyCode(code string) bool {
	if len(code) != 3 {
		return false
	}
	for _, r := range code {
		if !isLetter(r) {
			return false
		}
	}
	return true
}

// validateExchangeRate rejects NaN, Inf, zero, and negative exchange rate values.
// These are all invalid in a financial context and would produce silently wrong results.
func validateExchangeRate(key string, rate float64) error {
	if math.IsNaN(rate) || math.IsInf(rate, 0) {
		return fmt.Errorf("exchange rate for '%s' is not a finite number", key)
	}
	if rate <= 0 {
		return fmt.Errorf("exchange rate for '%s' must be positive, got %g", key, rate)
	}
	return nil
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

// LineCount returns the number of source lines occupied by the frontmatter
// (including both --- delimiters). Returns 0 if no frontmatter is present.
func (f *Frontmatter) LineCount() int {
	if f == nil || f.rawSource == "" {
		return 0
	}
	return strings.Count(f.rawSource, "\n")
}

// SetRawSource replaces the serialized frontmatter text (including --- delimiters
// and trailing newline) without re-parsing YAML. This allows the TUI editor to
// preserve user-typed text even when intermediate edits produce invalid YAML.
// Parsed data (Globals, Exchange) remains unchanged until a valid parse succeeds.
func (f *Frontmatter) SetRawSource(raw string) {
	f.rawSource = raw
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

// HasScale returns true if a scale directive is defined in frontmatter.
func (f *Frontmatter) HasScale() bool {
	return f != nil && f.Scale != nil
}

// HasGlobals returns true if any globals are defined in frontmatter.
func (f *Frontmatter) HasGlobals() bool {
	return f != nil && len(f.Globals) > 0
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
	Exchange           map[string]float64 `yaml:"exchange"`
	Globals            map[string]string  `yaml:"globals"`
	Scale              any                `yaml:"scale"`
	ConvertTo          any                `yaml:"convert_to"`
	Measurement        any                `yaml:"measurement"`
	FiscalYearStarts   string             `yaml:"fiscal_year_starts"`
	CalendarYearOffset string             `yaml:"calendar_year_offset"`
}

// CalendarYearOffset enumerates the FY-label conventions a document
// can declare via the `calendar_year_offset` frontmatter key.
type CalendarYearOffset int

const (
	// CalendarYearOffsetBefore is the default. The FY label corresponds
	// to the calendar year the FY ENDS in.
	CalendarYearOffsetBefore CalendarYearOffset = iota
	// CalendarYearOffsetAfter labels by the calendar year the FY
	// STARTS in.
	CalendarYearOffsetAfter
)

// parseScaleConfig parses the scale field which can be a scalar number or a map
// with factor and unit_categories keys.
func parseScaleConfig(raw any) (*ScaleConfig, error) {
	if raw == nil {
		return nil, nil
	}

	switch v := raw.(type) {
	case int:
		return validateScaleConfig(decimal.NewFromInt(int64(v)), nil)
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, fmt.Errorf("scale factor must be a finite number")
		}
		return validateScaleConfig(decimal.NewFromFloat(v), nil)
	case map[string]any:
		factorRaw, hasFactor := v["factor"]
		if !hasFactor {
			return nil, fmt.Errorf("scale map form requires 'factor' key")
		}
		var factor decimal.Decimal
		switch f := factorRaw.(type) {
		case int:
			factor = decimal.NewFromInt(int64(f))
		case float64:
			if math.IsNaN(f) || math.IsInf(f, 0) {
				return nil, fmt.Errorf("scale.factor must be a finite number")
			}
			factor = decimal.NewFromFloat(f)
		default:
			return nil, fmt.Errorf("scale.factor must be a number, got %T", factorRaw)
		}
		// Reject unknown sub-keys
		for key := range v {
			if key != "factor" && key != "unit_categories" {
				return nil, fmt.Errorf("unknown key %q in scale; valid keys: factor, unit_categories", key)
			}
		}
		cats, err := parseUnitCategories(v, "scale")
		if err != nil {
			return nil, err
		}
		return validateScaleConfig(factor, cats)
	default:
		return nil, fmt.Errorf("scale must be a number or a map with 'factor' key, got %T", raw)
	}
}

// validateScaleConfig validates and returns a ScaleConfig.
func validateScaleConfig(factor decimal.Decimal, categories []string) (*ScaleConfig, error) {
	if !factor.IsPositive() {
		return nil, fmt.Errorf("scale factor must be positive, got %s", factor.String())
	}
	return &ScaleConfig{Factor: factor, UnitCategories: categories}, nil
}

// parseConvertToConfig parses the convert_to field which can be a string ("si"/"imperial")
// or a map with system and unit_categories keys.
func parseConvertToConfig(raw any) (*ConvertToConfig, error) {
	if raw == nil {
		return nil, nil
	}

	switch v := raw.(type) {
	case string:
		return validateConvertToConfig(v, nil)
	case map[string]any:
		systemRaw, hasSystem := v["system"]
		if !hasSystem {
			return nil, fmt.Errorf("convert_to map form requires 'system' key")
		}
		system, ok := systemRaw.(string)
		if !ok {
			return nil, fmt.Errorf("convert_to.system must be a string, got %T", systemRaw)
		}
		// Reject unknown sub-keys
		for key := range v {
			if key != "system" && key != "unit_categories" {
				return nil, fmt.Errorf("unknown key %q in convert_to; valid keys: system, unit_categories", key)
			}
		}
		cats, err := parseUnitCategories(v, "convert_to")
		if err != nil {
			return nil, err
		}
		return validateConvertToConfig(system, cats)
	default:
		return nil, fmt.Errorf("convert_to must be a string or a map with 'system' key, got %T", raw)
	}
}

// validateConvertToConfig validates and returns a ConvertToConfig.
func validateConvertToConfig(system string, categories []string) (*ConvertToConfig, error) {
	normalized := strings.ToLower(strings.TrimSpace(system))
	if !validConvertToSystems[normalized] {
		return nil, fmt.Errorf("convert_to system must be 'si' or 'imperial', got %q", system)
	}
	return &ConvertToConfig{System: normalized, UnitCategories: categories}, nil
}

// parseMeasurementConfig parses the measurement field which must be a map
// with axis keys (volume, mass, ton) and an optional strict boolean.
func parseMeasurementConfig(raw any) (*MeasurementFrontmatter, error) {
	if raw == nil {
		return nil, nil
	}

	m, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("measurement must be a map with axis keys (volume, mass, ton), got %T", raw)
	}

	mc := &MeasurementFrontmatter{
		MeasurementConfig: *units.DefaultMeasurement(),
	}

	// Validate keys
	for key := range m {
		switch key {
		case "volume", "mass", "ton", "strict":
			// valid
		default:
			return nil, fmt.Errorf("unknown key %q in measurement; valid keys: volume, mass, ton, strict", key)
		}
	}

	// Parse volume axis
	if v, ok := m["volume"]; ok {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("measurement.volume must be a string, got %T", v)
		}
		normalized := strings.ToLower(strings.TrimSpace(s))
		if !validVolumeConventions[normalized] {
			return nil, fmt.Errorf("unknown volume convention %q — valid options: us, imperial", s)
		}
		mc.Volume = normalized
	}

	// Parse mass axis
	if v, ok := m["mass"]; ok {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("measurement.mass must be a string, got %T", v)
		}
		normalized := strings.ToLower(strings.TrimSpace(s))
		if !validMassConventions[normalized] {
			return nil, fmt.Errorf(
				"unknown mass convention %q — valid options: standard (everyday weight: 1 oz = 28.35g), troy (precious metals: 1 troy oz = 31.10g)",
				s,
			)
		}
		mc.Mass = normalized
	}

	// Parse ton axis
	if v, ok := m["ton"]; ok {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("measurement.ton must be a string, got %T", v)
		}
		normalized := strings.ToLower(strings.TrimSpace(s))
		if !validTonConventions[normalized] {
			return nil, fmt.Errorf("unknown ton convention %q — valid options: short (US, 2000 lb), long (Imperial, 2240 lb), metric (1000 kg)", s)
		}
		mc.Ton = normalized
	}

	// Parse strict flag
	if v, ok := m["strict"]; ok {
		b, ok := v.(bool)
		if !ok {
			return nil, fmt.Errorf("measurement.strict must be true or false, got %T", v)
		}
		mc.Strict = &b
	}

	return mc, nil
}

// parseUnitCategories extracts and validates unit_categories from a map form.
func parseUnitCategories(m map[string]any, directive string) ([]string, error) {
	catsRaw, hasCats := m["unit_categories"]
	if !hasCats {
		return nil, nil
	}

	catSlice, ok := catsRaw.([]any)
	if !ok {
		return nil, fmt.Errorf("%s.unit_categories must be a list", directive)
	}

	var categories []string
	seen := make(map[string]bool)
	for _, c := range catSlice {
		name, ok := c.(string)
		if !ok {
			return nil, fmt.Errorf("%s.unit_categories entries must be strings", directive)
		}
		canonical, valid := validUnitCategories[strings.ToLower(name)]
		if !valid {
			validNames := make([]string, 0, len(validUnitCategories))
			for _, v := range validUnitCategories {
				validNames = append(validNames, v)
			}
			sort.Strings(validNames)
			return nil, fmt.Errorf("invalid unit category %q in %s.unit_categories; valid categories: %s",
				name, directive, strings.Join(validNames, ", "))
		}
		if !seen[canonical] {
			categories = append(categories, canonical)
			seen[canonical] = true
		}
	}
	return categories, nil
}

// ErrFrontmatterUnclosed is returned by ParseFrontmatter when a document
// starts with an opening `---` fence but no matching closing fence is
// present. Along with this error, ParseFrontmatter returns the full
// source as the remaining body so interactive callers (editors, LSPs)
// can render in-progress content instead of erroring out.
//
// Strict callers (like NewDocument) may continue to treat this as fatal.
// Lenient callers should check `errors.Is(err, ErrFrontmatterUnclosed)`
// and use the returned source.
var ErrFrontmatterUnclosed = fmt.Errorf("frontmatter not closed: missing closing '---' delimiter")

// ParseFrontmatter extracts YAML frontmatter from the beginning of a document.
// Returns the parsed frontmatter, the remaining source (without frontmatter), and any error.
//
// Frontmatter must:
//   - Start at line 1 with exactly "---"
//   - End with a line containing exactly "---"
//   - Contain valid YAML between the delimiters
//   - Unknown top-level keys are silently ignored; only CalcMark keys are processed
//     (exchange, globals, scale, convert_to)
//
// If no frontmatter is present, returns (nil, source, nil).
//
// If the opening fence is found but the closing fence is missing,
// returns (nil, source, ErrFrontmatterUnclosed). The full source is
// returned as body so lenient callers can render in-progress content.
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
		return nil, source, ErrFrontmatterUnclosed
	}

	// Extract YAML content (between the delimiters)
	yamlContent := strings.Join(lines[1:closeIdx], "\n")

	// Parse into typed struct (unknown keys are silently ignored by yaml.Unmarshal)
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
		if err := validateExchangeRate(key, rate); err != nil {
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
			if err := validateExchangeRate(key, rate); err != nil {
				return nil, "", err
			}
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

	// Parse scale directive
	if raw.Scale != nil {
		sc, err := parseScaleConfig(raw.Scale)
		if err != nil {
			return nil, "", fmt.Errorf("frontmatter: %w", err)
		}
		fm.Scale = sc
	}

	// Parse convert_to directive
	if raw.ConvertTo != nil {
		ct, err := parseConvertToConfig(raw.ConvertTo)
		if err != nil {
			return nil, "", fmt.Errorf("frontmatter: %w", err)
		}
		fm.ConvertTo = ct
	}

	// Parse measurement directive
	if raw.Measurement != nil {
		mc, err := parseMeasurementConfig(raw.Measurement)
		if err != nil {
			return nil, "", fmt.Errorf("frontmatter: %w", err)
		}
		fm.Measurement = mc
	}

	// Parse fiscal_year_starts directive
	if raw.FiscalYearStarts != "" {
		fc, err := parseFiscalYearStarts(raw.FiscalYearStarts)
		if err != nil {
			return nil, "", fmt.Errorf("frontmatter: %w", err)
		}
		fm.FiscalYearStarts = fc
	}

	// Parse calendar_year_offset directive (defaults to "before").
	if raw.CalendarYearOffset != "" {
		off, err := parseCalendarYearOffset(raw.CalendarYearOffset)
		if err != nil {
			return nil, "", fmt.Errorf("frontmatter: %w", err)
		}
		fm.CalendarYearOffset = off
	}

	// Capture per-key source ranges for every top-level key (registered + Extra).
	// Lines are document-relative (1-based); the YAML body starts on line 2
	// because line 1 is the opening "---" fence, hence yamlLineOffset = 1.
	fm.KeyRanges = collectKeyRanges(yamlContent, 1)

	// Capture non-CalcMark frontmatter keys in document order. The Registry
	// (frontmatter_registry.go) is the source of truth for which keys carry
	// CalcMark semantics; everything else is preserved verbatim in Extra.
	extraOrder := extractYAMLTopLevelKeyOrder(yamlContent)
	var rawMap map[string]any
	_ = yaml.Unmarshal([]byte(yamlContent), &rawMap)
	for _, key := range extraOrder {
		if IsRegisteredKey(key) {
			continue
		}
		if val, ok := rawMap[key]; ok {
			fm.Extra = append(fm.Extra, ExtraField{Key: key, Value: val})
		}
	}

	// Calculate remaining source (after closing delimiter)
	remaining := ""
	if closeIdx+1 < len(lines) {
		remaining = strings.Join(lines[closeIdx+1:], "\n")
	}

	return fm, remaining, nil
}

// FiscalYearConfig holds the fiscal year start month and optional day.
type FiscalYearConfig struct {
	Month int // 1-12 (January-December)
	Day   int // 1-31 (defaults to 1 if not specified)
}

// parseFiscalYearStarts parses the fiscal_year_starts value.
// Accepts: "july", "jul", "July 15", "october 1", etc.
// When only a month is given, day defaults to 1.
func parseFiscalYearStarts(value string) (*FiscalYearConfig, error) {
	monthNames := map[string]int{
		"january": 1, "jan": 1, "february": 2, "feb": 2,
		"march": 3, "mar": 3, "april": 4, "apr": 4,
		"may": 5, "june": 6, "jun": 6, "july": 7, "jul": 7,
		"august": 8, "aug": 8, "september": 9, "sep": 9, "sept": 9,
		"october": 10, "oct": 10, "november": 11, "nov": 11,
		"december": 12, "dec": 12,
	}

	parts := strings.Fields(strings.TrimSpace(value))
	if len(parts) == 0 {
		return nil, fmt.Errorf("fiscal_year_starts: empty value")
	}

	month, ok := monthNames[strings.ToLower(parts[0])]
	if !ok {
		return nil, fmt.Errorf("fiscal_year_starts: invalid month name %q", parts[0])
	}

	if len(parts) > 2 {
		return nil, fmt.Errorf("fiscal_year_starts: expected 'Month' or 'Month Day' (e.g., 'july' or 'July 15'), got %q", value)
	}

	day := 1
	if len(parts) == 2 {
		d, err := strconv.Atoi(parts[1])
		if err != nil || d < 1 || d > 31 {
			return nil, fmt.Errorf("fiscal_year_starts: invalid day %q", parts[1])
		}
		day = d
	}

	return &FiscalYearConfig{Month: month, Day: day}, nil
}

// parseCalendarYearOffset parses the `calendar_year_offset` value.
// Accepts the strings "before" (default — FY label = year FY ends in)
// and "after" (FY label = year FY starts in). Comparison is
// case-insensitive; surrounding whitespace is tolerated. Any other
// value returns an error.
func parseCalendarYearOffset(value string) (CalendarYearOffset, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "before":
		return CalendarYearOffsetBefore, nil
	case "after":
		return CalendarYearOffsetAfter, nil
	default:
		return CalendarYearOffsetBefore, fmt.Errorf(
			"calendar_year_offset: invalid value %q (expected 'before' or 'after')", value)
	}
}

// parseYAMLMapping parses YAML content and returns the root mapping node.
// Returns nil if the content is empty, invalid, or not a mapping.
func parseYAMLMapping(yamlContent string) *yaml.Node {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(yamlContent), &root); err != nil {
		return nil
	}
	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		return nil
	}
	m := root.Content[0]
	if m.Kind != yaml.MappingNode {
		return nil
	}
	return m
}

// extractYAMLKeyOrder extracts the key ordering for a nested map under a
// top-level key from YAML content. Uses yaml.v3's Node API which preserves
// document order. Returns keys in the order they appear in the YAML source.
func extractYAMLKeyOrder(yamlContent string, topKey string) []string {
	mapping := parseYAMLMapping(yamlContent)
	if mapping == nil {
		return nil
	}

	for i := 0; i < len(mapping.Content)-1; i += 2 {
		keyNode := mapping.Content[i]
		valueNode := mapping.Content[i+1]
		if keyNode.Value == topKey && valueNode.Kind == yaml.MappingNode {
			var keys []string
			for j := 0; j < len(valueNode.Content)-1; j += 2 {
				keys = append(keys, valueNode.Content[j].Value)
			}
			return keys
		}
	}
	return nil
}

// collectKeyRanges walks the top-level keys of a YAML mapping and returns each
// key's source range covering the key identifier through the last byte of its
// value. Line/column numbers are 1-based per ast.Position; lineOffset is added
// to every yaml.Node line so callers can translate yaml-content-relative lines
// into document-relative lines (typical caller passes 1 to skip the "---"
// fence on document line 1).
//
// Returns an empty map (non-nil) when the YAML is empty or not a mapping, so
// callers can use len() and ranged-for safely without nil-checks.
//
// Complexity: O(N) in total YAML node count — single pass over top-level keys
// plus one descent per value to find its deepest end position. Lookup is O(1).
func collectKeyRanges(yamlContent string, lineOffset int) map[string]ast.Range {
	out := make(map[string]ast.Range)
	mapping := parseYAMLMapping(yamlContent)
	if mapping == nil {
		return out
	}
	for i := 0; i < len(mapping.Content)-1; i += 2 {
		keyNode := mapping.Content[i]
		valueNode := mapping.Content[i+1]
		start := ast.Position{
			Line:   keyNode.Line + lineOffset,
			Column: keyNode.Column,
		}
		end := nodeEndPosition(valueNode, lineOffset)
		out[keyNode.Value] = ast.Range{Start: start, End: end}
	}
	return out
}

// nodeEndPosition returns a Position pointing to the last byte of the YAML
// node's textual span. yaml.v3 only exposes a node's start; the end is derived
// by walking to the deepest trailing child for mappings/sequences and by
// adding the scalar's length for leaf scalars. lineOffset is added to convert
// yaml-content lines into document-relative lines.
func nodeEndPosition(n *yaml.Node, lineOffset int) ast.Position {
	if n == nil {
		return ast.Position{}
	}
	switch n.Kind {
	case yaml.MappingNode, yaml.SequenceNode:
		if len(n.Content) == 0 {
			return ast.Position{Line: n.Line + lineOffset, Column: n.Column}
		}
		// For mappings, the trailing element is a value (odd-indexed); for
		// sequences, it's the last item. In both cases the last Content entry
		// is the deepest trailing node — recurse into it.
		return nodeEndPosition(n.Content[len(n.Content)-1], lineOffset)
	case yaml.AliasNode:
		return ast.Position{Line: n.Line + lineOffset, Column: n.Column}
	default: // ScalarNode and DocumentNode (DocumentNode shouldn't appear here)
		// Approximate the scalar's end by adding its rendered length to the
		// start column. yaml.v3 does not expose folded/literal block scalar end
		// positions; for `|` and `>` block scalars this leaves End.Line on the
		// KEY line (the line of the `|` / `>` indicator), not the last physical
		// line of the block content. Consumers (currently only the LSP) that
		// anchor UI to the key location are unaffected; anything needing the
		// full byte span of a block scalar would be wrong. See
		// spec/document/frontmatter_yaml_shapes_test.go
		// (TestParseFrontmatter_YAMLShape_LiteralBlockScalar) — the test
		// asserts the observed (imperfect) value, so a future fix is a clean
		// test-flip.
		col := n.Column + len(n.Value)
		if col < n.Column {
			col = n.Column
		}
		return ast.Position{Line: n.Line + lineOffset, Column: col}
	}
}

// extractYAMLTopLevelKeyOrder returns all top-level YAML keys in document order.
func extractYAMLTopLevelKeyOrder(yamlContent string) []string {
	mapping := parseYAMLMapping(yamlContent)
	if mapping == nil {
		return nil
	}
	var keys []string
	for i := 0; i < len(mapping.Content)-1; i += 2 {
		keys = append(keys, mapping.Content[i].Value)
	}
	return keys
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

	// Use raw source if available (preserves user formatting during editing).
	// This takes priority over the empty-data check below because raw source
	// is set by ParseFrontmatter and SetRawSource — when present, the user
	// explicitly wrote frontmatter delimiters that should be preserved even
	// if the YAML content between them is empty.
	if f.rawSource != "" {
		return f.rawSource + "\n" // Add CommonMark blank line
	}

	if len(f.Exchange) == 0 && len(f.Globals) == 0 && f.Scale == nil && f.ConvertTo == nil {
		return ""
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

	// Serialize scale directive
	if f.Scale != nil {
		if len(f.Scale.UnitCategories) == 0 {
			sb.WriteString(fmt.Sprintf("scale: %s\n", f.Scale.Factor.String()))
		} else {
			sb.WriteString(fmt.Sprintf("scale:\n  factor: %s\n  unit_categories: [%s]\n",
				f.Scale.Factor.String(), strings.Join(f.Scale.UnitCategories, ", ")))
		}
	}

	// Serialize convert_to directive
	if f.ConvertTo != nil {
		if len(f.ConvertTo.UnitCategories) == 0 {
			sb.WriteString(fmt.Sprintf("convert_to: %s\n", f.ConvertTo.System))
		} else {
			sb.WriteString(fmt.Sprintf("convert_to:\n  system: %s\n  unit_categories: [%s]\n",
				f.ConvertTo.System, strings.Join(f.ConvertTo.UnitCategories, ", ")))
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
