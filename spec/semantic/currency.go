package semantic

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"golang.org/x/text/currency"
)

// Known currency symbols that map to ISO codes
var knownSymbols = map[string]bool{
	"$": true,
	"€": true,
	"£": true,
	"¥": true,
}

// ValidateCurrencyCodeWithDiagnostic validates a currency code and returns enhanced diagnostic
// USER REQUIREMENT: Provide both short and detailed messages with documentation link
func ValidateCurrencyCodeWithDiagnostic(code string) (valid bool, diag *Diagnostic) {
	if ValidateCurrencyCode(code) {
		return true, nil
	}

	// USER REQUIREMENT: Enhanced diagnostic with short + detailed + link to source
	sourceLink := "https://github.com/CalcMark/go-calcmark/blob/main/spec/semantic/currency.go"
	return false, &Diagnostic{
		Severity: Warning, // Warning not error - can still use as user-defined unit
		Code:     DiagInvalidCurrencyCode,
		Message:  "unknown currency",
		Detailed: fmt.Sprintf(
			"%s is not a known currency. You can still use %s as a "+
				"user-defined unit but you should not expect currency math to "+
				"work with it. See the currency validation code at: %s",
			code, code, sourceLink),
		Link: sourceLink,
	}
}

// ValidateCurrencyCode checks if a currency code is valid (ISO 4217 or common symbol)
func ValidateCurrencyCode(code string) bool {
	// Known symbols are always valid (they map to real currencies)
	if knownSymbols[code] {
		return true
	}

	// Must be exactly 3 uppercase letters
	if len(code) != 3 {
		return false
	}

	// Check if it's all uppercase letters
	for _, r := range code {
		if r < 'A' || r > 'Z' {
			return false
		}
	}

	// Validate against ISO 4217 using golang.org/x/text/currency
	unit, err := currency.ParseISO(code)
	if err != nil {
		return false
	}

	// Check if it's a recognized currency (not just syntactically valid)
	// XXX is a test/special code that shouldn't be used as real currency
	if code == "XXX" || code == "XTS" || code == "XUA" || code == "XAG" || code == "XAU" {
		return false // Test codes and special drawing rights
	}

	return unit.String() == code
}

// CreateInvalidCurrencyDiagnostic creates a HINT diagnostic for an invalid currency code.
// This suggests that the identifier might have been intended as a unit instead.
func CreateInvalidCurrencyDiagnostic(code string, node ast.Node) *Diagnostic {
	var rng *ast.Range
	if node != nil {
		rng = node.GetRange()
	}

	return &Diagnostic{
		Severity: Hint,
		Code:     DiagInvalidCurrencyCode,
		Message: fmt.Sprintf(
			`"%s" is not a valid ISO 4217 currency code. `+
				`If you meant to use a unit of measurement, this is fine. `+
				`Otherwise, check the currency code.`,
			code,
		),
		Range: rng,
	}
}

// CreateIncompatibleCurrenciesDiagnostic creates an ERROR diagnostic for incompatible currency operations.
func CreateIncompatibleCurrenciesDiagnostic(code1, code2 string, operation string, node ast.Node) *Diagnostic {
	var rng *ast.Range
	if node != nil {
		rng = node.GetRange()
	}

	return &Diagnostic{
		Severity: Error,
		Code:     DiagIncompatibleCurrencies,
		Message: fmt.Sprintf(
			`Cannot %s %s and %s directly - convert to the same currency first`,
			operation, code1, code2,
		),
		Range: rng,
	}
}

// NormalizeCurrencySymbol converts a currency symbol to its ISO code.
// Returns the normalized code and true if it's a known symbol, or the original value and false otherwise.
func NormalizeCurrencySymbol(symbolOrCode string) (string, bool) {
	// Check known symbols first
	switch symbolOrCode {
	case "$":
		return "USD", true
	case "€":
		return "EUR", true
	case "£":
		return "GBP", true
	case "¥":
		return "JPY", true
	}

	// If it's uppercase 3 letters, it's likely already a code
	if len(symbolOrCode) == 3 && strings.ToUpper(symbolOrCode) == symbolOrCode {
		return symbolOrCode, false
	}

	// Otherwise return as-is
	return symbolOrCode, false
}
