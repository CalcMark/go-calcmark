// Package calcmark provides a clean, idiomatic Go API for evaluating CalcMark expressions and documents.
//
// CalcMark is a calculation-oriented markup language that combines markdown with inline calculations.
//
// Basic usage:
//
//	result, err := calcmark.Eval("1 + 1")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(result.Value) // 2
//
// Stateful sessions (for live editors):
//
//	session := calcmark.NewSession()
//	session.Eval("x = 10")
//	result, _ := session.Eval("x + 5")
//	fmt.Println(result.Value) // 15
package calcmark

import (
	"fmt"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/parser"
	"github.com/CalcMark/go-calcmark/spec/semantic"
)

// Eval evaluates a CalcMark expression or document and returns the result.
// For single-line expressions, returns the computed value.
// For multi-line documents, returns per-line results.
//
// Example:
//
//	result, err := calcmark.Eval("100 + 20")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(result.Value) // 120
func Eval(input string) (*Result, error) {
	session := NewSession()
	return session.Eval(input)
}

// evaluate is the internal pipeline that connects parser → semantic → interpreter.
func evaluate(input string, env *interpreter.Environment) (*Result, error) {
	// Ensure input ends with newline (parser requirement)
	if len(input) > 0 && input[len(input)-1] != '\n' {
		input = input + "\n"
	}

	// 1. Parse the input
	nodes, parseErr := parser.Parse(input)
	if parseErr != nil {
		return nil, fmt.Errorf("parse error: %w", parseErr)
	}

	// 2. Perform semantic analysis
	// Create semantic environment from interpreter environment
	semEnv := semantic.NewEnvironment()
	// Copy variable names from interpreter environment to semantic environment
	// This ensures the semantic checker knows about variables defined in the session
	for name := range env.GetAllVariables() {
		semEnv.Set(name, nil) // Semantic env just tracks names, not values
	}

	checker := semantic.NewCheckerWithEnv(semEnv)
	diagnostics := checker.Check(nodes)

	// Update interpreter environment with new definitions from semantic check
	for name := range checker.GetEnvironment().GetAllVariables() {
		if !env.Has(name) && semEnv.Has(name) {
			// New variable defined during this eval - it'll get a value during interpretation
		}
	}

	// 3. Check for blocking errors
	hasError := false
	for _, d := range diagnostics {
		if d.Severity == semantic.Error {
			hasError = true
			break
		}
	}

	// Convert diagnostics to public format
	publicDiags := convertDiagnostics(diagnostics)

	// If there are semantic errors, return without interpreting
	if hasError {
		return &Result{
			Diagnostics: publicDiags,
		}, nil
	}

	// 4. Interpret the validated AST
	interp := interpreter.NewInterpreterWithEnv(env)
	values, interpErr := interp.Eval(nodes)
	if interpErr != nil {
		return nil, fmt.Errorf("evaluation error: %w", interpErr)
	}

	// 5. Build result
	result := &Result{
		Diagnostics: publicDiags,
		AllValues:   values,
	}

	// For single value, set as top-level result
	if len(values) == 1 {
		result.Value = values[0]
	} else if len(values) > 1 {
		// Multiple values (multi-line document) - use last value
		result.Value = values[len(values)-1]
	}

	return result, nil
}

// convertDiagnostics converts semantic diagnostics to public API format.
func convertDiagnostics(semDiags []semantic.Diagnostic) []Diagnostic {
	diags := make([]Diagnostic, len(semDiags))
	for i, d := range semDiags {
		diags[i] = Diagnostic{
			Severity: Severity(d.Severity),
			Code:     d.Code,
			Message:  d.Message,
		}
	}
	return diags
}
