package interpreter

import (
	"fmt"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/types"
)

// evalUnitConversion evaluates explicit unit conversion: "10 meters in feet"
func (interp *Interpreter) evalUnitConversion(u *ast.UnitConversion) (types.Type, error) {
	// Evaluate the quantity expression
	result, err := interp.evalNode(u.Quantity)
	if err != nil {
		return nil, err
	}

	// Must be a Quantity
	qty, ok := result.(*types.Quantity)
	if !ok {
		return nil, fmt.Errorf("'in' conversion requires a quantity, got %T", result)
	}

	// Use existing unit conversion logic
	converted, err := convertQuantity(qty, u.TargetUnit)
	if err != nil {
		return nil, err
	}

	return converted, nil
}
