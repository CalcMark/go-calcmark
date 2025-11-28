package interpreter

import (
	"maps"

	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

// Environment tracks variable bindings during interpretation.
// This is separate from Go's context.Context - it's simply variable storage for CalcMark variables.
type Environment struct {
	vars map[string]types.Type
}

// NewEnvironment creates a new empty environment with built-in constants.
func NewEnvironment() *Environment {
	env := &Environment{
		vars: make(map[string]types.Type),
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
		vars: make(map[string]types.Type),
	}
	maps.Copy(newEnv.vars, e.vars)
	return newEnv
}

// GetAllVariables returns the map of all variables (for sync with semantic checker).
func (e *Environment) GetAllVariables() map[string]types.Type {
	return e.vars
}
