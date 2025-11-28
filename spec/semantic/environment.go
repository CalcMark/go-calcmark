package semantic

import (
	"maps"

	"github.com/CalcMark/go-calcmark/spec/types"
)

// Environment tracks variable bindings during semantic analysis.
// This is separate from Go's context.Context - it's simply variable storage.
type Environment struct {
	vars map[string]types.Type
}

// NewEnvironment creates a new empty environment.
func NewEnvironment() *Environment {
	return &Environment{
		vars: make(map[string]types.Type),
	}
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
// Useful for creating scoped environments.
func (e *Environment) Clone() *Environment {
	newEnv := &Environment{
		vars: make(map[string]types.Type),
	}
	maps.Copy(newEnv.vars, e.vars)
	return newEnv
}

// GetAllVariables returns the map of all variables.
func (e *Environment) GetAllVariables() map[string]types.Type {
	return e.vars
}
