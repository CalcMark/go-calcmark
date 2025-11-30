package components

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/charmbracelet/lipgloss"
)

// ContextFooterHeight is the fixed height for the context footer / helper area.
// This must be consistent to avoid bubbletea rendering artifacts.
const ContextFooterHeight = 2

// VarReference represents a referenced variable and its value.
type VarReference struct {
	Name  string
	Value string
}

// ContextFooterState holds the data needed to render a context footer.
// This decouples the rendering from the editor model.
type ContextFooterState struct {
	// HasError indicates if the current line has an error
	HasError bool

	// Error information (when HasError is true)
	ErrorMessage string
	ErrorHint    string
	Diagnostic   *document.Diagnostic

	// Variable references (when no error)
	References []VarReference

	// Whether this is a calc line (footer only shows for calc lines)
	IsCalcLine bool
}

// RenderContextFooter renders the context footer from the given state.
// Pure function: takes state and width, returns string.
// IMPORTANT: Always returns exactly ContextFooterHeight lines.
func RenderContextFooter(state ContextFooterState, width int) string {
	// Helper to pad output to exactly ContextFooterHeight lines
	padToHeight := func(content string) string {
		lines := strings.Split(content, "\n")
		for len(lines) < ContextFooterHeight {
			lines = append(lines, lipgloss.NewStyle().Width(width).Render(""))
		}
		// Truncate if somehow more than expected (shouldn't happen)
		if len(lines) > ContextFooterHeight {
			lines = lines[:ContextFooterHeight]
		}
		return strings.Join(lines, "\n")
	}

	// Empty footer for non-calc lines
	if !state.IsCalcLine {
		return padToHeight("")
	}

	// Priority 1: Show errors with helpful formatting
	if state.HasError {
		// Style based on severity
		iconStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("208")) // amber
		msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))  // light gray
		hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // dim

		// Build error display from structured diagnostic if available
		var shortMsg, hint string

		if state.Diagnostic != nil {
			// Use structured diagnostic data
			shortMsg = state.Diagnostic.Message
			hint = GetHintForDiagnostic(state.Diagnostic)
		} else if state.ErrorMessage != "" {
			// Use pre-parsed error info
			shortMsg = state.ErrorMessage
			hint = state.ErrorHint
		} else {
			return padToHeight("")
		}

		// Build error display
		var lines []string

		// Line 1: Icon + short message
		line1 := iconStyle.Render("⚠ ") + msgStyle.Render(shortMsg)
		lines = append(lines, line1)

		// Line 2: Hint/suggestion if available
		if hint != "" {
			line2 := "  " + hintStyle.Render(hint)
			lines = append(lines, line2)
		} else {
			lines = append(lines, lipgloss.NewStyle().Width(width).Render(""))
		}

		// Ensure width constraints
		for i, line := range lines {
			if lipgloss.Width(line) > width {
				lines[i] = TruncateWithEllipsis(line, width)
			}
		}

		return padToHeight(strings.Join(lines, "\n"))
	}

	// Priority 2: Show variable references
	if len(state.References) == 0 {
		return padToHeight("")
	}

	// Format as: "var1 = value │ var2 = value │ ..."
	var parts []string
	for _, ref := range state.References {
		parts = append(parts, fmt.Sprintf("%s = %s", ref.Name, ref.Value))
	}

	content := strings.Join(parts, " │ ")

	// Render variable references on first line
	line1 := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Width(width).
		MaxWidth(width).
		Render(content)

	return padToHeight(line1)
}

// FindLineReferences extracts variable references from a line.
// This is a pure function - give it the line and known variables, get references back.
func FindLineReferences(line string, knownVars map[string]string, maxRefs int) []VarReference {
	var refs []VarReference
	seen := make(map[string]bool)

	for varName, val := range knownVars {
		// Check if this variable is referenced in the line
		// Skip if it's being defined on this line (left of =)
		if strings.Contains(line, varName) && !strings.HasPrefix(strings.TrimSpace(line), varName+" =") {
			if !seen[varName] {
				seen[varName] = true
				refs = append(refs, VarReference{
					Name:  varName,
					Value: val,
				})
			}
		}
	}

	// Limit to maxRefs references to fit in footer
	if len(refs) > maxRefs {
		refs = refs[:maxRefs]
	}

	return refs
}
