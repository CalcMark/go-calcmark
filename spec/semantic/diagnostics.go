package semantic

import (
	"strings"

	"github.com/CalcMark/go-calcmark/spec/ast"
)

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
	DiagInvalidDate        = "invalid_date"
	DiagInvalidMonth       = "invalid_month"
	DiagInvalidDay         = "invalid_day"
	DiagInvalidYear        = "invalid_year"
	DiagInvalidLeapYear    = "invalid_leap_year"
	DiagInvalidEndOfPeriod = "invalid_end_of_period"

	// Period operator diagnostics (v2.0)
	DiagInvalidLengthOfPeriod  = "invalid_length_of_period"  // length of / days in inner not Period
	DiagInvalidBetweenEndpoint = "invalid_between_endpoint" // between A and B endpoint not Date
	DiagInvalidPeriodRange     = "invalid_period_range"     // between A and B where end < start (statically detectable)

	// Variable diagnostics
	DiagUndefinedVariable    = "undefined_variable"
	DiagVariableRedefinition = "variable_redefinition"

	// Arithmetic diagnostics
	DiagDivisionByZero = "division_by_zero"

	// Data size unit hints
	DiagMixedBaseUnits = "mixed_base_units"

	// Directive diagnostics
	DiagInvalidDirective   = "invalid_directive"
	DiagUndefinedGlobal    = "undefined_global"
	DiagMissingFrontmatter = "missing_frontmatter"

	// Frontmatter diagnostics
	DiagFrontmatterValidation = "frontmatter_validation"

	// Parse diagnostics
	DiagParseError = "parse_error"
)

// HintForDiagnostic returns a user-facing hint for a diagnostic based on its
// code and message. This is the canonical source of hint definitions — the TUI
// layer delegates here rather than defining hints itself.
func HintForDiagnostic(code, message string) string {
	switch code {
	case DiagUndefinedVariable:
		varName := extractQuoted(message)
		if varName != "" {
			return "Define it above: " + varName + " = <value>"
		}
		return "Define the variable before using it"

	case DiagDivisionByZero:
		return "Check that divisor is not zero"

	case DiagIncompatibleUnits:
		return "Units must be compatible for this operation"

	case DiagTypeMismatch:
		return "Check that values are compatible types"

	case DiagParseError:
		return "Check syntax - see error message for details"

	case DiagInvalidCurrencyCode:
		return "Use a valid 3-letter currency code (e.g., USD, EUR)"

	case DiagFrontmatterValidation:
		// Extract valid options list from the error message if present.
		// Error format: "... valid categories: All, Area, Currency, ..."
		if idx := strings.Index(message, "valid categories: "); idx >= 0 {
			return message[idx:]
		}
		return "Check frontmatter YAML syntax"

	default:
		return ""
	}
}

// extractQuoted extracts the first double-quoted string from a message.
func extractQuoted(msg string) string {
	start := strings.Index(msg, "\"")
	if start < 0 {
		return ""
	}
	end := strings.Index(msg[start+1:], "\"")
	if end < 0 {
		return ""
	}
	return msg[start+1 : start+1+end]
}
