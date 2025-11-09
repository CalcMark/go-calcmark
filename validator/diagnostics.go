// Package validator provides semantic validation for CalcMark
package validator

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/ast"
)

// DiagnosticSeverity represents the severity of a diagnostic
// See DIAGNOSTIC_LEVELS.md for detailed definitions
type DiagnosticSeverity int

const (
	// Error indicates invalid syntax that prevents parsing
	Error DiagnosticSeverity = iota
	// Warning indicates valid syntax but evaluation failure (e.g., undefined variables)
	Warning
	// Hint indicates style/readability suggestions (does not affect functionality)
	Hint
)

func (s DiagnosticSeverity) String() string {
	switch s {
	case Error:
		return "error"
	case Warning:
		return "warning"
	case Hint:
		return "hint"
	default:
		return "unknown"
	}
}

// DiagnosticCode represents specific diagnostic codes
type DiagnosticCode int

const (
	UndefinedVariable DiagnosticCode = iota
	DivisionByZero
	TypeMismatch
	SyntaxError
	BlankLineIsolation // Hint: calculation should be isolated by blank lines
)

func (c DiagnosticCode) String() string {
	switch c {
	case UndefinedVariable:
		return "undefined_variable"
	case DivisionByZero:
		return "division_by_zero"
	case TypeMismatch:
		return "type_mismatch"
	case SyntaxError:
		return "syntax_error"
	case BlankLineIsolation:
		return "blank_line_isolation"
	default:
		return "unknown"
	}
}

// Diagnostic represents a diagnostic message about a calculation
type Diagnostic struct {
	Severity     DiagnosticSeverity
	Code         DiagnosticCode
	Message      string
	Range        *ast.Range
	VariableName string // For undefined variable diagnostics
}

// String returns a human-readable diagnostic string
func (d *Diagnostic) String() string {
	if d.Range != nil {
		return fmt.Sprintf("%s at %s: %s", d.Severity, d.Range, d.Message)
	}
	return fmt.Sprintf("%s: %s", d.Severity, d.Message)
}

// ToMap converts the diagnostic to a map for JSON serialization
func (d *Diagnostic) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"severity": d.Severity.String(),
		"code":     d.Code.String(),
		"message":  d.Message,
	}

	if d.Range != nil {
		result["range"] = map[string]interface{}{
			"start": map[string]int{
				"line":   d.Range.Start.Line,
				"column": d.Range.Start.Column,
			},
			"end": map[string]int{
				"line":   d.Range.End.Line,
				"column": d.Range.End.Column,
			},
		}
	}

	if d.VariableName != "" {
		result["variable_name"] = d.VariableName
	}

	return result
}

// ValidationResult represents the result of semantic validation
type ValidationResult struct {
	Diagnostics []*Diagnostic
}

// IsValid returns true if there are no errors (warnings are OK)
func (v *ValidationResult) IsValid() bool {
	return !v.HasErrors()
}

// HasErrors returns true if there are any error-level diagnostics
func (v *ValidationResult) HasErrors() bool {
	for _, d := range v.Diagnostics {
		if d.Severity == Error {
			return true
		}
	}
	return false
}

// HasWarnings returns true if there are any warning-level diagnostics
func (v *ValidationResult) HasWarnings() bool {
	for _, d := range v.Diagnostics {
		if d.Severity == Warning {
			return true
		}
	}
	return false
}

// Errors returns only error diagnostics
func (v *ValidationResult) Errors() []*Diagnostic {
	var errors []*Diagnostic
	for _, d := range v.Diagnostics {
		if d.Severity == Error {
			errors = append(errors, d)
		}
	}
	return errors
}

// Warnings returns only warning diagnostics
func (v *ValidationResult) Warnings() []*Diagnostic {
	var warnings []*Diagnostic
	for _, d := range v.Diagnostics {
		if d.Severity == Warning {
			warnings = append(warnings, d)
		}
	}
	return warnings
}

// HasHints returns true if there are any hint-level diagnostics
func (v *ValidationResult) HasHints() bool {
	for _, d := range v.Diagnostics {
		if d.Severity == Hint {
			return true
		}
	}
	return false
}

// Hints returns only hint diagnostics
func (v *ValidationResult) Hints() []*Diagnostic {
	var hints []*Diagnostic
	for _, d := range v.Diagnostics {
		if d.Severity == Hint {
			hints = append(hints, d)
		}
	}
	return hints
}

// String returns a human-readable summary
func (v *ValidationResult) String() string {
	if v.IsValid() {
		if v.HasWarnings() {
			return fmt.Sprintf("Valid with %d warning(s)", len(v.Warnings()))
		}
		return "Valid"
	}
	return fmt.Sprintf("%d error(s), %d warning(s)", len(v.Errors()), len(v.Warnings()))
}

// Bool returns true if the validation result is valid
func (v *ValidationResult) Bool() bool {
	return v.IsValid()
}

// NewValidationResult creates a new validation result
func NewValidationResult(diagnostics []*Diagnostic) *ValidationResult {
	if diagnostics == nil {
		diagnostics = []*Diagnostic{}
	}
	return &ValidationResult{Diagnostics: diagnostics}
}

// isBooleanKeyword checks if a name is a boolean keyword
func isBooleanKeyword(name string) bool {
	lower := strings.ToLower(name)
	keywords := map[string]bool{
		"true": true, "false": true,
		"yes": true, "no": true,
		"t": true, "f": true,
		"y": true, "n": true,
	}
	return keywords[lower]
}
