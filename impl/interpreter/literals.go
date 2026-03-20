package interpreter

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

// Literal evaluation methods.
// Each method converts an AST literal node to a typed value.

func (interp *Interpreter) evalNumberLiteral(n *ast.NumberLiteral) (types.Type, error) {
	// Percentage literals produce a Percentage type, not a Number.
	if strings.HasSuffix(n.Value, "%") {
		value, err := expandNumberLiteral(n.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid percentage literal %q: %w", n.Value, err)
		}
		return types.NewPercentage(value), nil
	}

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

func (interp *Interpreter) evalFractionLiteral(f *ast.FractionLiteral) (types.Type, error) {
	// Defense-in-depth: validate denominator even if semantic checker didn't run
	if f.Denominator == 0 {
		return nil, fmt.Errorf("division by zero: fraction %d/0", f.Numerator)
	}
	frac, err := types.NewFraction(f.Numerator, f.Denominator)
	if err != nil {
		return nil, err
	}
	frac.Unit = f.Unit
	return frac, nil
}

func (interp *Interpreter) evalQuantityLiteral(q *ast.QuantityLiteral) (types.Type, error) {
	var value decimal.Decimal
	var err error

	if q.Expr != nil {
		// Expression-based quantity (e.g., "@scale meters")
		result, evalErr := interp.evalNode(q.Expr)
		if evalErr != nil {
			return nil, evalErr
		}
		switch v := result.(type) {
		case *types.Number:
			value = v.Value
		case *types.Quantity:
			value = v.Value
		case *types.Currency:
			value = v.Value
		case *types.Percentage:
			value = v.Value
		default:
			return nil, fmt.Errorf("cannot use %T as quantity value", result)
		}
	} else {
		// Literal quantity (e.g., "5 meters")
		value, err = expandNumberLiteral(q.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid quantity value %q: %w", q.Value, err)
		}
	}

	// Resolve ambiguous unit names using measurement conventions.
	unit := interp.resolveUnit(q.Unit)
	return types.NewQuantity(value, unit), nil
}
