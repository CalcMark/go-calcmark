package semantic

import (
	"maps"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

// Mathematical built-in constants. Values mirror the runtime interpreter
// environment (impl/interpreter/environment.go) so static analysis and
// evaluation agree on whether a name like "PI" is bound. If the runtime
// list grows a constant, add it here too.
var (
	builtinPI = decimal.RequireFromString("3.14159265358979323846264338327950288419716939937510")
	builtinE  = decimal.RequireFromString("2.71828182845904523536028747135266249775724709369995")
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

// NewEnvironment creates a new environment pre-populated with the
// built-in mathematical constants (PI, E) so the semantic checker
// doesn't flag references to them as "undefined variable." The
// runtime interpreter environment does the equivalent via
// addConstants — this function keeps the two layers aligned.
func NewEnvironment() *Environment {
	env := &Environment{
		vars: make(map[string]*VarInfo),
	}
	env.addBuiltinConstants()
	return env
}

// addBuiltinConstants registers the mathematical constants the runtime
// interpreter also registers. The Range field is left nil — these
// names are not defined at any source location.
func (e *Environment) addBuiltinConstants() {
	e.vars["PI"] = &VarInfo{Type: types.NewNumber(builtinPI), Range: nil}
	e.vars["E"] = &VarInfo{Type: types.NewNumber(builtinE), Range: nil}
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
