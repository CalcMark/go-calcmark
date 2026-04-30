package interpreter

import (
	"maps"
	"strings"

	"github.com/CalcMark/go-calcmark/v2/spec/types"
	"github.com/shopspring/decimal"
)

// Environment tracks variable bindings during interpretation.
// This is separate from Go's context.Context - it's simply variable storage for CalcMark variables.
type Environment struct {
	vars          map[string]types.Type
	exchangeRates map[string]decimal.Decimal // "USD_EUR" -> rate
	erroredVars   map[string]error           // variables that failed evaluation
	// definedLines records the doc-absolute, 0-indexed line where each
	// user-defined variable was assigned. Built-in constants (PI, E)
	// and frontmatter globals applied before block evaluation are
	// intentionally absent — completions filter by line only when an
	// entry is present, so absent entries (constants, globals) are
	// always visible. Populated by `Interpreter.evalAssignment` once
	// `SetLineOffset` has wired up the block's doc-line offset.
	definedLines map[string]int
}

// NewEnvironment creates a new empty environment with built-in constants.
func NewEnvironment() *Environment {
	env := &Environment{
		vars:          make(map[string]types.Type),
		exchangeRates: make(map[string]decimal.Decimal),
		erroredVars:   make(map[string]error),
		definedLines:  make(map[string]int),
	}

	// Add built-in constants
	env.addConstants()

	return env
}

// Mathematical constants with high precision (50 decimal places).
// These are sufficient for any practical calculation.
var (
	piValue = decimal.RequireFromString("3.14159265358979323846264338327950288419716939937510")
	eValue  = decimal.RequireFromString("2.71828182845904523536028747135266249775724709369995")
)

// addConstants adds built-in mathematical constants (PI, E).
func (e *Environment) addConstants() {
	e.vars["PI"] = types.NewNumber(piValue)
	e.vars["E"] = types.NewNumber(eValue)
}

// Set stores a variable binding.
func (e *Environment) Set(name string, value types.Type) {
	e.vars[name] = value
}

// Get retrieves a variable binding.
// Returns the value and true if found, nil and false if not found.
func (e *Environment) Get(name string) (types.Type, bool) {
	val, ok := e.vars[name]
	return val, ok
}

// Has checks if a variable is defined.
func (e *Environment) Has(name string) bool {
	_, ok := e.vars[name]
	return ok
}

// Clone creates a shallow copy of the environment.
func (e *Environment) Clone() *Environment {
	newEnv := &Environment{
		vars:          make(map[string]types.Type),
		exchangeRates: make(map[string]decimal.Decimal),
		erroredVars:   make(map[string]error),
		definedLines:  make(map[string]int),
	}
	maps.Copy(newEnv.vars, e.vars)
	maps.Copy(newEnv.exchangeRates, e.exchangeRates)
	maps.Copy(newEnv.erroredVars, e.erroredVars)
	maps.Copy(newEnv.definedLines, e.definedLines)
	return newEnv
}

// SetDefinedLine records the doc-absolute, 0-indexed line where a
// variable was assigned. Used by the LSP to filter completion items
// to variables defined ABOVE the cursor — calcmark is strictly ordered.
func (e *Environment) SetDefinedLine(name string, line int) {
	e.definedLines[name] = line
}

// GetAllDefinedLines returns a snapshot of variable → defined-line.
// Variables without a recorded line (built-in constants, frontmatter
// globals) are absent from the map, which signals "no position
// constraint" to the consumer.
func (e *Environment) GetAllDefinedLines() map[string]int {
	result := make(map[string]int, len(e.definedLines))
	maps.Copy(result, e.definedLines)
	return result
}

// GetAllVariables returns a snapshot copy of all variables.
// The returned map is safe to iterate without concern for concurrent mutation.
func (e *Environment) GetAllVariables() map[string]types.Type {
	result := make(map[string]types.Type, len(e.vars))
	maps.Copy(result, e.vars)
	return result
}

// SetError marks a variable as having failed evaluation.
func (e *Environment) SetError(name string, err error) {
	e.erroredVars[name] = err
}

// GetError checks if a variable has a recorded evaluation error.
// Returns the error and true if found, nil and false otherwise.
func (e *Environment) GetError(name string) (error, bool) {
	err, ok := e.erroredVars[name]
	return err, ok
}

// ClearError removes a single variable's error state.
func (e *Environment) ClearError(name string) {
	delete(e.erroredVars, name)
}

// ClearErrors removes all tracked variable errors.
func (e *Environment) ClearErrors() {
	clear(e.erroredVars)
}

// GetAllErroredVars returns a copy of the errored variables map.
func (e *Environment) GetAllErroredVars() map[string]error {
	result := make(map[string]error, len(e.erroredVars))
	maps.Copy(result, e.erroredVars)
	return result
}

// SetExchangeRate sets an exchange rate for currency conversion.
// Key format: "FROM_TO" (e.g., "USD_EUR").
func (e *Environment) SetExchangeRate(from, to string, rate decimal.Decimal) {
	key := strings.ToUpper(from) + "_" + strings.ToUpper(to)
	e.exchangeRates[key] = rate
}

// GetExchangeRate retrieves an exchange rate for currency conversion.
// Returns the rate and true if found, zero and false if not defined.
func (e *Environment) GetExchangeRate(from, to string) (decimal.Decimal, bool) {
	key := strings.ToUpper(from) + "_" + strings.ToUpper(to)
	rate, ok := e.exchangeRates[key]
	return rate, ok
}

// HasExchangeRates returns true if any exchange rates are defined.
func (e *Environment) HasExchangeRates() bool {
	return len(e.exchangeRates) > 0
}
