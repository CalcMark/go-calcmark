package interpreter

import (
	"fmt"
	"math"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

// Function call evaluation.

func (interp *Interpreter) evalFunctionCall(f *ast.FunctionCall) (types.Type, error) {
	// Evaluate all arguments
	args := make([]types.Type, len(f.Arguments))
	for i, arg := range f.Arguments {
		val, err := interp.evalNode(arg)
		if err != nil {
			return nil, err
		}
		args[i] = val
	}

	// Call the appropriate function
	switch f.Name {
	case "avg", "average":
		return evalAverage(args)
	case "sqrt":
		return evalSqrt(args)
	default:
		return nil, fmt.Errorf("unknown function: %s", f.Name)
	}
}

// evalAverage calculates the average of numbers.
func evalAverage(args []types.Type) (types.Type, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("avg() requires at least one argument")
	}

	// Extract numeric values
	numbers, err := extractNumbers(args)
	if err != nil {
		return nil, err
	}

	// Calculate sum
	sum := numbers[0]
	for i := 1; i < len(numbers); i++ {
		sum = sum.Add(numbers[i])
	}

	// Calculate average
	count := len(numbers)
	avg := sum.Div(decimal.NewFromInt(int64(count)))

	return types.NewNumber(avg), nil
}

// evalSqrt calculates the square root.
func evalSqrt(args []types.Type) (types.Type, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("sqrt() requires exactly one argument")
	}

	num, ok := args[0].(*types.Number)
	if !ok {
		return nil, fmt.Errorf("sqrt() argument must be a number")
	}

	// Use simple Newton's method for sqrt since decimal doesn't have built-in
	if num.Value.IsNegative() {
		return nil, fmt.Errorf("sqrt() argument must be non-negative")
	}

	// TODO: Implement proper decimal sqrt using Newton's method
	// For now, convert to float64, take sqrt, convert back
	f, _ := num.Value.Float64()
	result := decimal.NewFromFloat(math.Sqrt(f))

	return types.NewNumber(result), nil
}

// extractNumbers extracts decimal values from typed arguments.
func extractNumbers(args []types.Type) ([]decimal.Decimal, error) {
	numbers := make([]decimal.Decimal, 0, len(args))

	for _, arg := range args {
		switch v := arg.(type) {
		case *types.Number:
			numbers = append(numbers, v.Value)
		case *types.Currency:
			numbers = append(numbers, v.Value)
		default:
			return nil, fmt.Errorf("argument must be a number or currency, got %T", arg)
		}
	}

	return numbers, nil
}
