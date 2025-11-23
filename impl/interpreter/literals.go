package interpreter

import (
	"fmt"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/types"
)

// Literal evaluation methods.
// Each method converts an AST literal node to a typed value.

func (interp *Interpreter) evalNumberLiteral(n *ast.NumberLiteral) (types.Type, error) {
	value, err := expandNumberLiteral(n.Value)
	if err != nil {
		return nil, fmt.Errorf("invalid number literal %q: %w", n.Value, err)
	}

	return types.NewNumber(value), nil
}

func (interp *Interpreter) evalCurrencyLiteral(c *ast.CurrencyLiteral) (types.Type, error) {
	value, err := expandNumberLiteral(c.Value)
	if err != nil {
		return nil, fmt.Errorf("invalid currency value %q: %w", c.Value, err)
	}

	return types.NewCurrency(value, c.Symbol), nil
}

func (interp *Interpreter) evalBooleanLiteral(b *ast.BooleanLiteral) (types.Type, error) {
	value, err := parseBooleanValue(b.Value)
	if err != nil {
		return nil, err
	}

	return types.NewBoolean(value), nil
}

func (interp *Interpreter) evalQuantityLiteral(q *ast.QuantityLiteral) (types.Type, error) {
	return types.NewQuantityFromString(q.Value, q.Unit)
}
