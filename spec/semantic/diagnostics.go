package semantic

import "github.com/CalcMark/go-calcmark/spec/ast"

// Severity represents the severity level of a diagnostic.
type Severity int

const (
	// Error indicates a critical error that prevents execution.
	Error Severity = iota
	// Warning indicates a semantic issue that should be addressed.
	Warning
	// Hint indicates a suggestion or style recommendation.
	Hint
)

// String returns the string representation of the severity.
func (s Severity) String() string {
	switch s {
	case Error:
		return "ERROR"
	case Warning:
		return "WARNING"
	case Hint:
		return "HINT"
	default:
		return "UNKNOWN"
	}
}

// Diagnostic represents a semantic validation issue.
// USER REQUIREMENT: Both short and detailed messages for better UX
type Diagnostic struct {
	Severity Severity
	Code     string     // Diagnostic code: "invalid_currency_code", "type_mismatch", etc.
	Message  string     // Short, human-readable error message (e.g., "unknown currency")
	Detailed string     // Detailed explanation with context and guidance
	Link     string     // Optional documentation link for more information
	Range    *ast.Range // Location in source code
}

// DiagnosticCode constants for all diagnostic types
const (
	// Currency diagnostics
	DiagInvalidCurrencyCode    = "invalid_currency_code"
	DiagIncompatibleCurrencies = "incompatible_currencies"

	// Type diagnostics
	DiagTypeMismatch         = "type_mismatch"
	DiagInvalidDateOperation = "invalid_date_operation"
	DiagUnsupportedUnit      = "unsupported_unit"
	DiagIncompatibleUnits    = "incompatible_units"

	// Date diagnostics (USER REQUIREMENT)
	DiagInvalidDate     = "invalid_date"
	DiagInvalidMonth    = "invalid_month"
	DiagInvalidDay      = "invalid_day"
	DiagInvalidYear     = "invalid_year"
	DiagInvalidLeapYear = "invalid_leap_year"

	// Variable diagnostics
	DiagUndefinedVariable = "undefined_variable"

	// Arithmetic diagnostics
	DiagDivisionByZero = "division_by_zero"

	// Data size unit hints
	DiagMixedBaseUnits = "mixed_base_units"
)
