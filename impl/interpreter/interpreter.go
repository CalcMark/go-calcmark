package interpreter

import (
	"fmt"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/types"
)

// Interpreter executes validated AST nodes and produces typed results.
// This is a Go-specific implementation of CalcMark execution.
type Interpreter struct {
	env *Environment
}

// NewInterpreter creates a new interpreter with an empty environment.
func NewInterpreter() *Interpreter {
	return &Interpreter{
		env: NewEnvironment(),
	}
}

// NewInterpreterWithEnv creates a new interpreter with a pre-populated environment.
func NewInterpreterWithEnv(env *Environment) *Interpreter {
	return &Interpreter{
		env: env,
	}
}

// Eval executes a list of AST nodes and returns the results.
// Each node produces a typed value.
func (interp *Interpreter) Eval(nodes []ast.Node) ([]types.Type, error) {
	results := make([]types.Type, 0, len(nodes))

	for _, node := range nodes {
		result, err := interp.evalNode(node)
		if err != nil {
			return nil, err
		}
		if result != nil {
			results = append(results, result)
		}
	}

	return results, nil
}

// evalNode evaluates a single AST node.
func (interp *Interpreter) evalNode(node ast.Node) (types.Type, error) {
	if node == nil {
		return nil, nil
	}

	switch n := node.(type) {
	case *ast.Assignment:
		return interp.evalAssignment(n)
	case *ast.Expression:
		// Unwrap expression and evaluate the inner node
		return interp.evalNode(n.Expr)
	case *ast.BinaryOp:
		return interp.evalBinaryOp(n)
	case *ast.ComparisonOp:
		return interp.evalComparisonOp(n)
	case *ast.UnaryOp:
		return interp.evalUnaryOp(n)
	case *ast.Identifier:
		return interp.evalIdentifier(n)
	case *ast.NumberLiteral:
		return interp.evalNumberLiteral(n)
	case *ast.CurrencyLiteral:
		return interp.evalCurrencyLiteral(n)
	case *ast.BooleanLiteral:
		return interp.evalBooleanLiteral(n)
	case *ast.DateLiteral:
		return interp.evalDateLiteral(n)
	case *ast.TimeLiteral:
		return interp.evalTimeLiteral(n)
	case *ast.DurationLiteral:
		return interp.evalDurationLiteral(n)
	case *ast.RelativeDateLiteral:
		return interp.evalRelativeDateLiteral(n)
	case *ast.QuantityLiteral:
		return interp.evalQuantityLiteral(n)
	case *ast.UnitConversion:
		return interp.evalUnitConversion(n)
	case *ast.FunctionCall:
		return interp.evalFunctionCall(n)
	default:
		return nil, fmt.Errorf("unknown node type: %T", node)
	}
}

// GetEnvironment returns the interpreter's environment.
func (interp *Interpreter) GetEnvironment() *Environment {
	return interp.env
}
