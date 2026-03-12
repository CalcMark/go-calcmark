package editor

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/components"
	"github.com/CalcMark/go-calcmark/spec/semantic"
)

// renderContextFooter renders the context footer showing errors or referenced variables.
// Delegates to components.RenderContextFooter with prepared state.
// maxHeight controls the footer height (2 normally, up to 4 on error lines).
func (m Model) renderContextFooter(width int, results []LineResult, maxHeight int) string {

	// Build state for the pure render function
	state := components.ContextFooterState{}

	// Check bounds
	if m.cursorLine < len(results) {
		currentResult := results[m.cursorLine]
		state.IsCalcLine = currentResult.IsCalc || currentResult.IsFrontmatter

		// Find the effective error for this line. For frontmatter lines the
		// error lives on the closing --- but applies to the whole block.
		// Uses the same helper as contextFooterHeight to keep height and
		// content in sync (pure-functional-layout-calculations pattern).
		if errResult := m.effectiveErrorForLine(results); errResult != nil {
			state.HasError = true

			if errResult.IsBlocked {
				// Blocked errors are caused by an undefined variable from a prior
				// error. Show a brief message instead of the full diagnostic so the
				// user focuses on the root cause. Footer stays compact (2 lines).
				state.ErrorMessage = components.CleanErrorMessage(errResult.Error)
				state.ErrorHint = "Caused by error above — fix it first"
			} else {
				state.Diagnostic = errResult.Diagnostic

				// If no structured diagnostic, parse the error string for display
				if state.Diagnostic == nil {
					errInfo := components.ParseErrorForDisplay(errResult.Error)
					state.ErrorMessage = errInfo.ShortMessage
					state.ErrorHint = errInfo.Hint
				}
			}
		}

		// Get variable references if no error
		if !state.HasError && currentResult.IsCalc {
			state.References = m.getLineReferences(m.cursorLine, results)

			// When the current line is scaled, append @scale = N as a synthetic reference
			if currentResult.IsScaled {
				if fm := m.doc.GetFrontmatter(); fm != nil && fm.Scale != nil {
					state.References = append(state.References, components.VarReference{
						Name:  "@scale",
						Value: fm.Scale.Factor.String(),
					})
				}
			}
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

	return components.RenderContextFooter(state, width, contextFooterBg, maxHeight)
}

// getLineReferences returns variables referenced in the given line.
// Uses AST-derived ReferencedVars from LineResult instead of text matching.
// Accepts pre-computed results to avoid redundant GetLineResults() calls.
// O(v) where v is the number of referenced variables in the statement.
func (m Model) getLineReferences(lineNum int, results []LineResult) []components.VarReference {
	if lineNum >= len(results) || len(results[lineNum].ReferencedVars) == 0 {
		return nil
	}

	env := m.eval.GetEnvironment()
	allVars := env.GetAllVariables()

	// Filter out self-references: if this line defines variable X, don't show
	// X in the footer. This prevents circular display for cases like
	// "hundred_gig = throughput(hundred_gig)" where the function argument
	// shares the variable name.
	definedVar := results[lineNum].VarName

	const maxRefs = 4
	refs := results[lineNum].ReferencedVars
	out := make([]components.VarReference, 0, min(len(refs), maxRefs))
	for _, varName := range refs {
		if varName == definedVar {
			continue
		}
		if val, ok := allVars[varName]; ok {
			out = append(out, components.VarReference{
				Name:  varName,
				Value: m.displayFormat(val),
			})
			if len(out) >= maxRefs {
				break
			}
		}
	}
	return out
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
