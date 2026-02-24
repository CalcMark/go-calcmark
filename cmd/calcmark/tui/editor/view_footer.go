package editor

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/components"
	"github.com/CalcMark/go-calcmark/spec/semantic"
	"github.com/charmbracelet/lipgloss"
)

// renderContextFooter renders the context footer showing errors or referenced variables.
// Delegates to components.RenderContextFooter with prepared state.
func (m Model) renderContextFooter(width int) string {
	results := m.GetLineResults()

	// Build state for the pure render function
	state := components.ContextFooterState{}

	// Check bounds
	if m.cursorLine < len(results) {
		currentResult := results[m.cursorLine]
		state.IsCalcLine = currentResult.IsCalc

		if currentResult.IsCalc && currentResult.Error != "" {
			state.HasError = true
			state.Diagnostic = currentResult.Diagnostic

			// If no structured diagnostic, parse the error string for display
			if state.Diagnostic == nil {
				errInfo := components.ParseErrorForDisplay(currentResult.Error)
				state.ErrorMessage = errInfo.ShortMessage
				state.ErrorHint = errInfo.Hint
			}
		}

		// Get variable references if no error
		if !state.HasError && state.IsCalcLine {
			state.References = m.getLineReferences(m.cursorLine)
		}
	}

	// Add autocomplete details when popup is active
	if m.mode == StateAutocomplete && m.autocompleteState.Visible {
		if len(m.autocompleteState.Suggestions) > 0 {
			selected := m.autocompleteState.Suggestions[m.autocompleteState.Selected]
			state.AutocompleteActive = true
			state.AutocompleteName = selected.InsertText
			if state.AutocompleteName == "" {
				state.AutocompleteName = selected.Name
			}
			state.AutocompleteSyntax = selected.Syntax

			// For functions, show parameter examples instead of/in addition to description
			// This helps users understand what format to use for each parameter
			funcName := selected.InsertText
			if funcName == "" {
				funcName = selected.Name
			}
			paramHint := formatFunctionParamHint(funcName)
			if paramHint != "" {
				state.AutocompleteDesc = paramHint
			} else {
				state.AutocompleteDesc = selected.Description
			}
		}
	}

	// Check for function argument context (when typing inside function call)
	// NOTE: We check this even when there's an error because incomplete function
	// calls (like "accumulate(") will have parse errors but should still show
	// parameter help. The function context takes priority over error display
	// in RenderContextFooter (Priority 0.5 vs Priority 1).
	if !state.AutocompleteActive {
		m.loadCurrentLineIntoEditBuffer()
		cursorCtx := GetCursorContext(m.editBuf, m.cursorCol)
		if cursorCtx.InFunctionCall && cursorCtx.ParamSpec != nil {
			state.InFunctionCall = true
			state.FunctionName = cursorCtx.FunctionName
			state.ParamName = cursorCtx.ParamSpec.Name
			state.ParamExamples = FormatParamHelp(cursorCtx.ParamSpec)
			state.ArgIndex = cursorCtx.ArgIndex
		}
	}

	// Get themed context footer background
	contextFooterBg := m.styles.ContextFooter.GetBackground()
	if _, ok := contextFooterBg.(lipgloss.NoColor); ok {
		contextFooterBg = m.sourcePaneBg() // Fallback to source pane background
	}

	return components.RenderContextFooter(state, width, contextFooterBg)
}

// getLineReferences returns variables referenced in the given line.
// Delegates to components.FindLineReferences with model's known variables.
func (m Model) getLineReferences(lineNum int) []components.VarReference {
	lines := m.GetLines()
	if lineNum >= len(lines) {
		return nil
	}

	line := lines[lineNum]

	// Build map of known variables from environment
	env := m.eval.GetEnvironment()
	allVars := env.GetAllVariables()

	knownVars := make(map[string]string)
	for varName, val := range allVars {
		knownVars[varName] = fmt.Sprintf("%v", val)
	}

	return components.FindLineReferences(line, knownVars, 4)
}

// formatFunctionParamHint looks up a function's parameter specs and formats
// a helpful hint showing examples for each parameter type.
// Returns empty string if the function has no parameter specs.
func formatFunctionParamHint(funcName string) string {
	spec := semantic.GetFunctionSpec(funcName)
	if spec == nil || len(spec.Params) == 0 {
		return ""
	}

	// Format: "param1: example | param2: example"
	var parts []string
	for _, param := range spec.Params {
		example := ""
		if len(param.Examples) > 0 {
			// Show first example as it's usually most representative
			example = param.Examples[0]
		} else {
			// Fall back to type examples
			typeExamples := semantic.GetExamplesForType(param.Type)
			if len(typeExamples) > 0 {
				example = typeExamples[0]
			}
		}

		if example != "" {
			parts = append(parts, param.Name+": "+example)
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, " | ")
}
