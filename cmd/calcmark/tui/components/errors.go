package components

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/CalcMark/go-calcmark/v2/spec/document"
	"github.com/CalcMark/go-calcmark/v2/spec/semantic"
)

// ErrorDisplayInfo contains parsed error information for display.
type ErrorDisplayInfo struct {
	ShortMessage string // e.g., "Undefined variable: My Budget"
	Hint         string // e.g., "Define it above this line: My Budget = <value>"
	Code         string // e.g., "undefined_variable"
}

// GetHintForDiagnostic returns a helpful hint based on the structured diagnostic.
// Prefers the Detailed field populated by the semantic checker. Falls back to
// semantic.HintForDiagnostic for diagnostics that lack Detailed (e.g. synthetic
// frontmatter diagnostics created in the TUI layer).
func GetHintForDiagnostic(diag *document.Diagnostic) string {
	if diag.Detailed != "" {
		return diag.Detailed
	}
	return semantic.HintForDiagnostic(diag.Code, diag.Message)
}

// ParseErrorForDisplay extracts structured error information for user-friendly display.
// Used as a fallback when structured diagnostics aren't available.
// Infers the diagnostic code from the error message text, then delegates hint
// generation to semantic.HintForDiagnostic.
func ParseErrorForDisplay(errMsg string) ErrorDisplayInfo {
	info := ErrorDisplayInfo{}

	lowerErr := strings.ToLower(errMsg)

	// Infer diagnostic code from error message text
	switch {
	case strings.Contains(lowerErr, "undefined variable") || strings.Contains(lowerErr, "undefined_variable"):
		varName := ExtractQuotedString(errMsg)
		if varName != "" {
			info.Code = semantic.DiagUndefinedVariable
			info.ShortMessage = fmt.Sprintf("Undefined variable: %s", varName)
		}

	case strings.Contains(lowerErr, "division") && strings.Contains(lowerErr, "zero"):
		info.Code = semantic.DiagDivisionByZero
		info.ShortMessage = "Division by zero"

	case strings.Contains(lowerErr, "incompatible") && strings.Contains(lowerErr, "unit"):
		info.Code = semantic.DiagIncompatibleUnits
		info.ShortMessage = CleanErrorMessage(errMsg)

	case strings.Contains(lowerErr, "type") && strings.Contains(lowerErr, "mismatch"):
		info.Code = semantic.DiagTypeMismatch
		info.ShortMessage = CleanErrorMessage(errMsg)

	case strings.Contains(lowerErr, "frontmatter") && strings.Contains(lowerErr, "invalid"):
		info.Code = semantic.DiagFrontmatterValidation
		info.ShortMessage = CleanErrorMessage(errMsg)
	}

	// Get hint from semantic layer
	if info.Code != "" {
		info.Hint = semantic.HintForDiagnostic(info.Code, errMsg)
	}

	// Default short message if not set above
	if info.ShortMessage == "" {
		info.ShortMessage = CleanErrorMessage(errMsg)
	}

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
	// Remove repeated "frontmatter: " prefixes (caused by double-wrapping in
	// ParseFrontmatter and NewDocument).
	for strings.HasPrefix(errMsg, "frontmatter: frontmatter: ") {
		errMsg = errMsg[len("frontmatter: "):]
	}

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

	// Scan backwards for the right truncation point
	// (accounting for variable-width characters)
	for i := len(s) - 1; i > 0; i-- {
		truncated := s[:i] + "..."
		if lipgloss.Width(truncated) <= maxWidth {
			return truncated
		}
	}

	return "..."
}
