package calcmark

import (
	"github.com/CalcMark/go-calcmark/spec/types"
)

// Result contains the evaluation results and any diagnostics.
type Result struct {
	// Value is the final computed value (for single expressions).
	// For multi-line documents, this is the last value.
	Value types.Type

	// AllValues contains all computed values (for multi-line documents).
	AllValues []types.Type

	// Diagnostics contains any errors, warnings, or hints from semantic analysis.
	Diagnostics []Diagnostic
}

// Diagnostic represents a semantic issue (error, warning, or hint).
type Diagnostic struct {
	Severity Severity
	Code     string
	Message  string
}

// Severity indicates the severity level of a diagnostic.
type Severity int

const (
	// Error indicates a blocking error that prevents interpretation.
	Error Severity = iota
	// Warning indicates a potential issue that doesn't block interpretation.
	Warning
	// Hint indicates a suggestion for improvement.
	Hint
)

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
