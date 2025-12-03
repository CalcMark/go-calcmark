package semantic

import (
	"maps"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/types"
)

// VarInfo tracks information about a variable definition
type VarInfo struct {
	Type  types.Type
	Range *ast.Range
}

// Environment tracks variable bindings during semantic analysis.
// This is separate from Go's context.Context - it's simply variable storage.
type Environment struct {
	vars map[string]*VarInfo
}

// NewEnvironment creates a new empty environment.
func NewEnvironment() *Environment {
	return &Environment{
		vars: make(map[string]*VarInfo),
	}
}

// Set stores a variable binding with optional range information.
func (e *Environment) Set(name string, value types.Type) {
	e.SetWithRange(name, value, nil)
}

// SetWithRange stores a variable binding with range information.
func (e *Environment) SetWithRange(name string, value types.Type, r *ast.Range) {
	e.vars[name] = &VarInfo{
		Type:  value,
		Range: r,
	}
}

// Get retrieves a variable binding.
// Returns the value and true if found, nil and false if not found.
func (e *Environment) Get(name string) (types.Type, bool) {
	info, ok := e.vars[name]
	if !ok {
		return nil, false
	}
	return info.Type, ok
}

// GetInfo retrieves full variable information including range.
func (e *Environment) GetInfo(name string) (*VarInfo, bool) {
	info, ok := e.vars[name]
	return info, ok
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
		vars: make(map[string]*VarInfo),
	}
	maps.Copy(newEnv.vars, e.vars)
	return newEnv
}

// GetAllVariables returns the map of all variables.
func (e *Environment) GetAllVariables() map[string]types.Type {
	result := make(map[string]types.Type)
	for name, info := range e.vars {
		result[name] = info.Type
	}
	return result
}
