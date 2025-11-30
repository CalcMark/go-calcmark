package interpreter

import (
	"maps"
	"strings"

	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

// Environment tracks variable bindings during interpretation.
// This is separate from Go's context.Context - it's simply variable storage for CalcMark variables.
type Environment struct {
	vars          map[string]types.Type
	exchangeRates map[string]decimal.Decimal // "USD_EUR" -> rate
}

// NewEnvironment creates a new empty environment with built-in constants.
func NewEnvironment() *Environment {
	env := &Environment{
		vars:          make(map[string]types.Type),
		exchangeRates: make(map[string]decimal.Decimal),
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
	}
	maps.Copy(newEnv.vars, e.vars)
	maps.Copy(newEnv.exchangeRates, e.exchangeRates)
	return newEnv
}

// GetAllVariables returns the map of all variables (for sync with semantic checker).
func (e *Environment) GetAllVariables() map[string]types.Type {
	return e.vars
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
