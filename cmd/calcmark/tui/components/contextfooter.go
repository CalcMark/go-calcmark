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

	// Autocomplete hint (shown when autocomplete is active)
	AutocompleteActive bool
	AutocompleteName   string // Selected function/unit name
	AutocompleteSyntax string // Full signature like "avg(values...)"
	AutocompleteDesc   string // Description
}

// RenderContextFooter renders the context footer from the given state.
// Pure function: takes state, width, and background color, returns string.
// IMPORTANT: Always returns exactly ContextFooterHeight lines.
func RenderContextFooter(state ContextFooterState, width int, bg lipgloss.TerminalColor) string {
	// Helper to pad output to exactly ContextFooterHeight lines
	padToHeight := func(content string) string {
		lines := strings.Split(content, "\n")
		for len(lines) < ContextFooterHeight {
			// Add empty lines with background color to prevent terminal bleed-through
			lines = append(lines, StyledPadding(width, bg))
		}
		// Truncate if somehow more than expected (shouldn't happen)
		if len(lines) > ContextFooterHeight {
			lines = lines[:ContextFooterHeight]
		}
		// Ensure each line has background color by padding to full width
		for i, line := range lines {
			currentWidth := lipgloss.Width(line)
			if currentWidth < width {
				// Add styled padding to reach full width
				lines[i] = line + StyledPadding(width-currentWidth, bg)
			}
		}
		return strings.Join(lines, "\n")
	}

	// Empty footer for non-calc lines (unless autocomplete is showing)
	if !state.IsCalcLine && !state.AutocompleteActive {
		return padToHeight("")
	}

	// Priority 0: Show autocomplete details when active
	if state.AutocompleteActive && state.AutocompleteName != "" {
		nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true).Background(bg)   // bright blue
		syntaxStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(bg)           // white
		descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("246")).Italic(true).Background(bg) // gray italic

		var lines []string

		// Line 1: Name + Syntax (function signature)
		line1 := nameStyle.Render(state.AutocompleteName)
		if state.AutocompleteSyntax != "" {
			line1 += " " + syntaxStyle.Render(state.AutocompleteSyntax)
		}
		lines = append(lines, line1)

		// Line 2: Description
		if state.AutocompleteDesc != "" {
			line2 := "  " + descStyle.Render(state.AutocompleteDesc)
			lines = append(lines, line2)
		} else {
			lines = append(lines, StyledPadding(width, bg))
		}

		// Ensure width constraints
		for i, line := range lines {
			if lipgloss.Width(line) > width {
				lines[i] = TruncateWithEllipsis(line, width)
			}
		}

		return padToHeight(strings.Join(lines, "\n"))
	}

	// Priority 1: Show errors with helpful formatting
	if state.HasError {
		// Style based on severity - include background to prevent terminal bleed
		// Use bold red for error icon to make it highly visible
		iconStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true).Background(bg)   // bold red on themed bg
		msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true).Background(bg)    // bold white on themed bg
		hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Italic(true).Background(bg) // italic light gray on themed bg

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
			lines = append(lines, StyledPadding(width, bg))
		}

		// Ensure width constraints - lines already have background from styles above
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

	// Render variable references on first line with themed background
	line1 := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Background(bg).
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
