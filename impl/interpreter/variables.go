package interpreter

import (
	"fmt"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/types"
)

// Variable and identifier evaluation.

func (interp *Interpreter) evalAssignment(a *ast.Assignment) (types.Type, error) {
	value, err := interp.evalNode(a.Value)
	if err != nil {
		return nil, err
	}

	interp.env.Set(a.Name, value)
	return value, nil
}

func (interp *Interpreter) evalIdentifier(id *ast.Identifier) (types.Type, error) {
	// Check for errored variables FIRST (error recovery: variable failed in prior statement)
	if cause, errored := interp.env.GetError(id.Name); errored {
		return nil, &CascadingError{VarName: id.Name, Cause: cause}
	}

	// Check for defined variables (variables take precedence over keywords)
	if value, ok := interp.env.Get(id.Name); ok {
		return value, nil
	}

	// Then check for boolean keywords
	if isBooleanKeyword(id.Name) {
		value, _ := parseBooleanValue(id.Name)
		return types.NewBoolean(value), nil
	}

	// Undefined variable
	return nil, fmt.Errorf("undefined variable: %q", id.Name)
}
