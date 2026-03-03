package components

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/config/theme"
	"github.com/CalcMark/go-calcmark/spec/document"
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

	// Function argument context (shown when typing inside function call)
	InFunctionCall bool
	FunctionName   string // Name of function being called
	ParamName      string // Current parameter name
	ParamExamples  string // Example values for this parameter
	ArgIndex       int    // Which argument position (0-based)
}

// RenderContextFooter renders the context footer from the given state.
// Pure function: takes state, width, and background color, returns string.
// IMPORTANT: Always returns exactly ContextFooterHeight lines.
func RenderContextFooter(state ContextFooterState, width int, bg color.Color) string {
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

	// Empty footer for non-calc lines (unless autocomplete or function help is showing).
	// Incomplete function calls like "accumulate(" aren't parsed as calc lines,
	// but should still show parameter help.
	if !state.IsCalcLine && !state.AutocompleteActive && !state.InFunctionCall {
		return padToHeight("")
	}

	// Styled space/text helper: renders text with bg to prevent terminal bleed-through.
	// ALL raw text between styled segments must use this.
	bgText := func(s string) string {
		return lipgloss.NewStyle().Background(bg).Render(s)
	}

	// Priority 0: Show autocomplete details when active
	if state.AutocompleteActive && state.AutocompleteName != "" {
		nameStyle := lipgloss.NewStyle().Foreground(theme.FooterFuncName).Bold(true).Background(bg)
		descStyle := lipgloss.NewStyle().Foreground(theme.TextMuted).Italic(true).Background(bg)

		var lines []string

		// Line 1: Syntax already includes the function name (e.g. "capacity(demand, ...)"),
		// so render it directly to avoid showing the name twice.
		var line1 string
		if state.AutocompleteSyntax != "" {
			line1 = nameStyle.Render(state.AutocompleteSyntax)
		} else {
			line1 = nameStyle.Render(state.AutocompleteName)
		}
		lines = append(lines, line1)

		// Line 2: Description
		if state.AutocompleteDesc != "" {
			line2 := bgText("  ") + descStyle.Render(state.AutocompleteDesc)
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

	// Priority 0.5: Show function argument help when inside function call
	if state.InFunctionCall && state.ParamName != "" {
		funcStyle := lipgloss.NewStyle().Foreground(theme.FooterFuncName).Bold(true).Background(bg)
		paramStyle := lipgloss.NewStyle().Foreground(theme.FooterParamHighlight).Bold(true).Background(bg)
		exampleStyle := lipgloss.NewStyle().Foreground(theme.Text).Background(bg)

		var lines []string

		// Line 1: function(arg1, arg2, ►current_arg◄, ...)
		line1 := funcStyle.Render(state.FunctionName) + bgText("(") + paramStyle.Render("►"+state.ParamName+"◄") + bgText(")")
		lines = append(lines, line1)

		// Line 2: Examples for this argument type
		if state.ParamExamples != "" {
			line2 := bgText("  ") + exampleStyle.Render(state.ParamExamples)
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
		iconStyle := lipgloss.NewStyle().Foreground(theme.ErrorIcon).Bold(true).Background(bg)
		msgStyle := lipgloss.NewStyle().Foreground(theme.TextBright).Bold(true).Background(bg)
		hintStyle := lipgloss.NewStyle().Foreground(theme.Hint).Italic(true).Background(bg)

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
			line2 := bgText("  ") + hintStyle.Render(hint)
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
		Foreground(theme.FooterVarRef).
		Background(bg).
		Width(width).
		MaxWidth(width).
		Render(content)

	return padToHeight(line1)
}

