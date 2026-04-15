package semantic

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/shopspring/decimal"
)

// TestCheckFrontmatter_Empty: an empty Frontmatter produces no diagnostics.
func TestCheckFrontmatter_Empty(t *testing.T) {
	fm := document.Frontmatter{}
	diags := CheckFrontmatter(fm)
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics for empty frontmatter, got %d: %+v", len(diags), diags)
	}
}

// TestCheckFrontmatter_ValidConvertToSI: convert_to: si is accepted.
func TestCheckFrontmatter_ValidConvertToSI(t *testing.T) {
	fm := document.Frontmatter{
		ConvertTo: &document.ConvertToConfig{System: "si"},
	}
	diags := CheckFrontmatter(fm)
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics, got %d: %+v", len(diags), diags)
	}
}

// TestCheckFrontmatter_ValidConvertToImperial: convert_to: imperial is accepted.
func TestCheckFrontmatter_ValidConvertToImperial(t *testing.T) {
	fm := document.Frontmatter{
		ConvertTo: &document.ConvertToConfig{System: "imperial"},
	}
	diags := CheckFrontmatter(fm)
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics, got %d: %+v", len(diags), diags)
	}
}

// TestCheckFrontmatter_ValidExchange: positive exchange rates produce no diagnostics.
func TestCheckFrontmatter_ValidExchange(t *testing.T) {
	fm := document.Frontmatter{
		Exchange: map[string]decimal.Decimal{
			"USD_EUR": decimal.NewFromFloat(1.1),
		},
	}
	diags := CheckFrontmatter(fm)
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics, got %d: %+v", len(diags), diags)
	}
}

// TestCheckFrontmatter_InvalidConvertToValue: convert_to with unknown system value
// produces one Error diagnostic, anchored at KeyRanges["convert_to"], whose message
// names both the offending value and the valid set.
func TestCheckFrontmatter_InvalidConvertToValue(t *testing.T) {
	keyRange := ast.Range{
		Start: ast.Position{Line: 2, Column: 1},
		End:   ast.Position{Line: 2, Column: 22},
	}
	fm := document.Frontmatter{
		ConvertTo: &document.ConvertToConfig{System: "xyz_not_a_value"},
		KeyRanges: map[string]ast.Range{
			"convert_to": keyRange,
		},
	}
	diags := CheckFrontmatter(fm)
	if len(diags) != 1 {
		t.Fatalf("expected exactly 1 diagnostic, got %d: %+v", len(diags), diags)
	}
	d := diags[0]
	if d.Severity != Error {
		t.Errorf("expected Severity=Error, got %v", d.Severity)
	}
	if !strings.Contains(d.Message, "xyz_not_a_value") {
		t.Errorf("expected message to mention offending value 'xyz_not_a_value', got %q", d.Message)
	}
	if !strings.Contains(d.Message, "si") || !strings.Contains(d.Message, "imperial") {
		t.Errorf("expected message to list valid set 'si'/'imperial', got %q", d.Message)
	}
	if d.Range == nil {
		t.Fatalf("expected Range to be populated, got nil")
	}
	if *d.Range != keyRange {
		t.Errorf("expected Range %+v, got %+v", keyRange, *d.Range)
	}
}

// TestCheckFrontmatter_InvalidConvertTo_NoKeyRanges: when KeyRanges is missing
// the entry, the diagnostic still emits but with a zero-value Range (fallback).
func TestCheckFrontmatter_InvalidConvertTo_NoKeyRanges(t *testing.T) {
	fm := document.Frontmatter{
		ConvertTo: &document.ConvertToConfig{System: "xyz"},
	}
	diags := CheckFrontmatter(fm)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
	if diags[0].Range == nil {
		t.Fatalf("expected non-nil Range (zero-value fallback), got nil")
	}
	if (*diags[0].Range != ast.Range{}) {
		t.Errorf("expected zero-value Range fallback, got %+v", *diags[0].Range)
	}
}

// TestCheckFrontmatter_ExchangeZeroRate: a zero exchange rate is an Error
// (zero rates would yield zero-valued conversions silently).
func TestCheckFrontmatter_ExchangeZeroRate(t *testing.T) {
	keyRange := ast.Range{
		Start: ast.Position{Line: 3, Column: 1},
		End:   ast.Position{Line: 4, Column: 12},
	}
	fm := document.Frontmatter{
		Exchange: map[string]decimal.Decimal{
			"USD_EUR": decimal.Zero,
		},
		KeyRanges: map[string]ast.Range{
			"exchange": keyRange,
		},
	}
	diags := CheckFrontmatter(fm)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d: %+v", len(diags), diags)
	}
	if diags[0].Severity != Error {
		t.Errorf("expected Error, got %v", diags[0].Severity)
	}
	if !strings.Contains(diags[0].Message, "USD_EUR") {
		t.Errorf("expected message to name the rate key 'USD_EUR', got %q", diags[0].Message)
	}
	if diags[0].Range == nil || *diags[0].Range != keyRange {
		t.Errorf("expected Range from KeyRanges[exchange], got %+v", diags[0].Range)
	}
}

// TestCheckFrontmatter_ExchangeNegativeRate: negative rates are nonsense (Error).
func TestCheckFrontmatter_ExchangeNegativeRate(t *testing.T) {
	fm := document.Frontmatter{
		Exchange: map[string]decimal.Decimal{
			"USD_EUR": decimal.NewFromFloat(-1),
		},
	}
	diags := CheckFrontmatter(fm)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
	if diags[0].Severity != Error {
		t.Errorf("expected Error, got %v", diags[0].Severity)
	}
	if !strings.Contains(diags[0].Message, "USD_EUR") {
		t.Errorf("expected USD_EUR in message, got %q", diags[0].Message)
	}
}

// TestCheckFrontmatter_ExtraKeysIgnored_Title: an Extra "title" key produces
// no diagnostics (the passthrough invariant).
func TestCheckFrontmatter_ExtraKeysIgnored_Title(t *testing.T) {
	fm := document.Frontmatter{
		Extra: []document.ExtraField{{Key: "title", Value: "Hello"}},
	}
	diags := CheckFrontmatter(fm)
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics for Extra passthrough, got %d: %+v", len(diags), diags)
	}
}

// TestCheckFrontmatter_ExtraKeysIgnored_Jekyll: another Jekyll-style Extra key.
func TestCheckFrontmatter_ExtraKeysIgnored_Jekyll(t *testing.T) {
	fm := document.Frontmatter{
		Extra: []document.ExtraField{{Key: "jekyll_layout", Value: "post"}},
	}
	diags := CheckFrontmatter(fm)
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics for Extra passthrough, got %d: %+v", len(diags), diags)
	}
}

// TestCheckFrontmatter_CaseSensitiveEnum: convert_to: SI (uppercase) is rejected.
// Decision: enum comparison is case-sensitive at the semantic layer. The parser
// already lowercases input, so any uppercase value reaching CheckFrontmatter was
// constructed programmatically and should be flagged. Documented in the impl.
func TestCheckFrontmatter_CaseSensitiveEnum(t *testing.T) {
	fm := document.Frontmatter{
		ConvertTo: &document.ConvertToConfig{System: "SI"},
	}
	diags := CheckFrontmatter(fm)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic for uppercase 'SI', got %d", len(diags))
	}
}

// TestCheckFrontmatter_MultipleInvalid_OrderedByLine: multiple invalid keys
// produce diagnostics ordered by KeyRanges.Start.Line (ascending).
func TestCheckFrontmatter_MultipleInvalid_OrderedByLine(t *testing.T) {
	fm := document.Frontmatter{
		ConvertTo: &document.ConvertToConfig{System: "xyz"},
		Exchange: map[string]decimal.Decimal{
			"USD_EUR": decimal.Zero,
		},
		KeyRanges: map[string]ast.Range{
			// exchange appears earlier in source
			"exchange":   {Start: ast.Position{Line: 2, Column: 1}, End: ast.Position{Line: 3, Column: 14}},
			"convert_to": {Start: ast.Position{Line: 4, Column: 1}, End: ast.Position{Line: 4, Column: 18}},
		},
	}
	diags := CheckFrontmatter(fm)
	if len(diags) != 2 {
		t.Fatalf("expected 2 diagnostics, got %d: %+v", len(diags), diags)
	}
	if diags[0].Range == nil || diags[1].Range == nil {
		t.Fatalf("expected both diagnostics to have ranges")
	}
	if diags[0].Range.Start.Line >= diags[1].Range.Start.Line {
		t.Errorf("expected diagnostics ordered by Start.Line ascending; got line %d before line %d",
			diags[0].Range.Start.Line, diags[1].Range.Start.Line)
	}
	// The earlier-line diagnostic should be the exchange one
	if !strings.Contains(diags[0].Message, "USD_EUR") {
		t.Errorf("expected first diagnostic to be the exchange one (line 2), got message %q", diags[0].Message)
	}
}

// TestCheckFrontmatter_AllSixKeysValid: a populated Frontmatter with all six
// registered keys set to valid values produces no diagnostics.
func TestCheckFrontmatter_AllSixKeysValid(t *testing.T) {
	strict := true
	fm := document.Frontmatter{
		Exchange: map[string]decimal.Decimal{
			"USD_EUR": decimal.NewFromFloat(0.92),
		},
		Globals: map[string]string{
			"tax_rate": "0.32",
		},
		Scale:     &document.ScaleConfig{Factor: decimal.NewFromInt(2)},
		ConvertTo: &document.ConvertToConfig{System: "si"},
		Measurement: &document.MeasurementFrontmatter{
			Strict: &strict,
		},
		FiscalYearStarts: &document.FiscalYearConfig{Month: 1, Day: 1},
	}
	diags := CheckFrontmatter(fm)
	if len(diags) != 0 {
		t.Errorf("expected no diagnostics for fully valid frontmatter, got %d: %+v", len(diags), diags)
	}
}

// TestCheckFrontmatter_ScaleNonPositive: a non-positive scale factor is an Error.
// Programmatic construction can bypass the parser's positivity check.
func TestCheckFrontmatter_ScaleNonPositive(t *testing.T) {
	fm := document.Frontmatter{
		Scale: &document.ScaleConfig{Factor: decimal.Zero},
	}
	diags := CheckFrontmatter(fm)
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
	if diags[0].Severity != Error {
		t.Errorf("expected Error severity, got %v", diags[0].Severity)
	}
}
