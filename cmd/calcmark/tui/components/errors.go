package components

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/charmbracelet/lipgloss"
)

// ErrorDisplayInfo contains parsed error information for display.
type ErrorDisplayInfo struct {
	ShortMessage string // e.g., "Undefined variable: My Budget"
	Hint         string // e.g., "Define it above this line: My Budget = <value>"
	Code         string // e.g., "undefined_variable"
}

// GetHintForDiagnostic returns a helpful hint based on the structured diagnostic.
func GetHintForDiagnostic(diag *document.Diagnostic) string {
	switch diag.Code {
	case "undefined_variable":
		// Extract variable name from message: `Undefined variable "varname" - ...`
		varName := ExtractQuotedString(diag.Message)
		if varName != "" {
			return fmt.Sprintf("Define it above: %s = <value>", varName)
		}
		return "Define the variable before using it"

	case "division_by_zero":
		return "Check that divisor is not zero"

	case "incompatible_units":
		return "Units must be compatible for this operation"

	case "type_mismatch":
		return "Check that values are compatible types"

	case "parse_error":
		return "Check syntax - see error message for details"

	case "invalid_currency_code":
		return "Use a valid 3-letter currency code (e.g., USD, EUR)"

	default:
		return ""
	}
}

// ParseErrorForDisplay extracts structured error information for user-friendly display.
// Used as a fallback when structured diagnostics aren't available.
func ParseErrorForDisplay(errMsg string) ErrorDisplayInfo {
	info := ErrorDisplayInfo{}

	lowerErr := strings.ToLower(errMsg)

	// Handle: "undefined_variable: Undefined variable \"varname\" - ..."
	// or: "undefined variable: \"varname\""
	if strings.Contains(lowerErr, "undefined variable") || strings.Contains(lowerErr, "undefined_variable") {
		// Extract variable name from quotes
		varName := ExtractQuotedString(errMsg)
		if varName != "" {
			info.Code = "undefined_variable"
			info.ShortMessage = fmt.Sprintf("Undefined variable: %s", varName)
			info.Hint = fmt.Sprintf("Define it above: %s = <value>", varName)
			return info
		}
	}

	// Handle: "division_by_zero" or "division by zero"
	if strings.Contains(lowerErr, "division") && strings.Contains(lowerErr, "zero") {
		info.Code = "division_by_zero"
		info.ShortMessage = "Division by zero"
		info.Hint = "Check that divisor is not zero"
		return info
	}

	// Handle: "incompatible units" or "incompatible_units"
	if strings.Contains(lowerErr, "incompatible") && strings.Contains(lowerErr, "unit") {
		info.Code = "incompatible_units"
		// Try to extract units from message
		info.ShortMessage = CleanErrorMessage(errMsg)
		info.Hint = "Units must be compatible for this operation"
		return info
	}

	// Handle: "type_mismatch" or "type mismatch"
	if strings.Contains(lowerErr, "type") && strings.Contains(lowerErr, "mismatch") {
		info.Code = "type_mismatch"
		info.ShortMessage = CleanErrorMessage(errMsg)
		info.Hint = "Check that values are compatible types"
		return info
	}

	// Default: clean up the raw error message
	info.ShortMessage = CleanErrorMessage(errMsg)
	return info
}

// ExtractQuotedString extracts the first quoted string from a message.
func ExtractQuotedString(msg string) string {
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

// CleanErrorMessage removes redundant prefixes and cleans up error messages.
func CleanErrorMessage(errMsg string) string {
	// Remove error code prefixes like "undefined_variable: "
	if idx := strings.Index(errMsg, ": "); idx > 0 && idx < 30 {
		prefix := errMsg[:idx]
		// Check if prefix looks like an error code (snake_case or single word)
		if strings.Contains(prefix, "_") || !strings.Contains(prefix, " ") {
			errMsg = errMsg[idx+2:]
		}
	}

	// Trim and clean up
	errMsg = strings.TrimSpace(errMsg)

	// Remove trailing " - " fragments
	if idx := strings.LastIndex(errMsg, " - "); idx > 0 && idx > len(errMsg)-10 {
		errMsg = errMsg[:idx]
	}

	return errMsg
}

// TruncateWithEllipsis truncates a string to fit within maxWidth, adding "..." if truncated.
func TruncateWithEllipsis(s string, maxWidth int) string {
	if lipgloss.Width(s) <= maxWidth {
		return s
	}

	// Binary search for the right truncation point
	// (accounting for variable-width characters)
	for i := len(s) - 1; i > 0; i-- {
		truncated := s[:i] + "..."
		if lipgloss.Width(truncated) <= maxWidth {
			return truncated
		}
	}

	return "..."
}
