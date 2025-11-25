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
	// Special case: convert_rate's second argument should NOT be evaluated
	// It's an identifier representing a time unit, not a variable
	if f.Name == "convert_rate" {
		if len(f.Arguments) != 2 {
			return nil, fmt.Errorf("convert_rate() requires 2 arguments")
		}

		// Evaluate first argument (the rate)
		rateArg, err := interp.evalNode(f.Arguments[0])
		if err != nil {
			return nil, err
		}

		rate, ok := rateArg.(*types.Rate)
		if !ok {
			return nil, fmt.Errorf("convert_rate() first argument must be a rate, got %T", rateArg)
		}

		// Extract time unit from second argument without evaluating
		var targetUnit string
		switch arg := f.Arguments[1].(type) {
		case *ast.Identifier:
			targetUnit = arg.Name
		default:
			// If it's not an identifier, evaluate it and use String()
			val, err := interp.evalNode(f.Arguments[1])
			if err != nil {
				return nil, err
			}
			targetUnit = val.String()
		}

		return convertRateTimeUnit(rate, targetUnit)
	}

	// Evaluate all arguments for other functions
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
	case "accumulate":
		return evalAccumulate(args)
	case "convert_rate":
		// Already handled above
		return nil, fmt.Errorf("convert_rate should have been handled")
	default:
		return nil, fmt.Errorf("unknown function: %s", f.Name)
	}
}

// evalAccumulate handles accumulate(rate, time_period) function calls.
func evalAccumulate(args []types.Type) (types.Type, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("accumulate() requires 2 arguments (rate, time_period)")
	}

	rate, ok := args[0].(*types.Rate)
	if !ok {
		return nil, fmt.Errorf("accumulate() first argument must be a rate, got %T", args[0])
	}

	// Second argument can be Duration or Quantity (number with time unit)
	var periodValue decimal.Decimal
	var periodUnit string

	switch period := args[1].(type) {
	case *types.Duration:
		// Duration stores value in its own unit (e.g., 1 day = Value:1, Unit:"day")
		periodValue = period.Value
		periodUnit = period.Unit
	case *types.Quantity:
		periodValue = period.Value
		periodUnit = period.Unit
	case *types.Number:
		// Plain number - assume it's in seconds
		periodValue = period.Value
		periodUnit = "second"
	default:
		return nil, fmt.Errorf("accumulate() second argument must be a duration or time quantity, got %T", args[1])
	}

	return accumulateRate(rate, periodValue, periodUnit)
}

// evalConvertRate handles convert_rate(rate, target_unit) function calls.
func evalConvertRate(args []types.Type) (types.Type, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("convert_rate() requires 2 arguments (rate, target_time_unit)")
	}

	rate, ok := args[0].(*types.Rate)
	if !ok {
		return nil, fmt.Errorf("convert_rate() first argument must be a rate, got %T", args[0])
	}

	// Second argument: for natural syntax the parser passes an ast.Identifier
	// which will fail to evaluate (no such variable). We'll catch that and extract
	// the identifier name from the error, or better yet, handle it in the function
	// call evaluation. For now, just use String() representation
	targetUnit := args[1].String()

	return convertRateTimeUnit(rate, targetUnit)
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
