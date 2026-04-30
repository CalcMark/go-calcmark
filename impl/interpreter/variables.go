package interpreter

import (
	"fmt"

	"github.com/CalcMark/go-calcmark/v2/spec/ast"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
)

// Variable and identifier evaluation.

func (interp *Interpreter) evalAssignment(a *ast.Assignment) (types.Type, error) {
	value, err := interp.evalNode(a.Value)
	if err != nil {
		return nil, err
	}

	interp.env.Set(a.Name, value)
	interp.env.ClearError(a.Name) // Clear any stale error from a prior failed evaluation

	// Record doc-absolute, 0-indexed line for LSP completion position
	// filtering. AST positions are 1-indexed; subtract 1 to align with
	// LSP's 0-indexed protocol convention. Skip when the AST has no
	// Range (synthetic assignments from frontmatter etc. — those should
	// always be visible from any line, and absent from definedLines
	// signals "no constraint" to the consumer).
	if a.Range != nil && a.Range.Start.Line > 0 {
		interp.env.SetDefinedLine(a.Name, a.Range.Start.Line-1+interp.lineOffset)
	}

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
