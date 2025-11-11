package evaluator

import (
	"fmt"
	"math"

	"github.com/CalcMark/go-calcmark/impl/types"
	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/shopspring/decimal"
)

// Functions Module
//
// This module implements all built-in CalcMark functions (avg, sqrt, etc.).
// Uses a registry pattern for extensibility - new functions can be added by:
// 1. Creating an evalXxx function with the functionImpl signature
// 2. Registering it in the init() function
//
// All functions are pure - they operate on values, not side effects.
// Unicode-aware: Function names are ASCII, but arguments support full Unicode.

// functionImpl is the signature for function implementations.
// All functions receive the evaluator (for recursive evaluation), arguments, and range for errors.
type functionImpl func(*Evaluator, []ast.Node, *ast.Range) (types.Type, error)

// functionRegistry maps function names to their implementations
// Using lazy initialization to avoid circular dependency
var functionRegistry map[string]functionImpl

func init() {
	functionRegistry = map[string]functionImpl{
		"avg":  evalAvg,
		"sqrt": evalSqrt,
	}
}

// evalAvg implements the avg() function
func evalAvg(e *Evaluator, args []ast.Node, nodeRange *ast.Range) (types.Type, error) {
	if len(args) == 0 {
		return nil, &EvaluationError{
			Message: "avg() requires at least one argument",
			Range:   nodeRange,
		}
	}

	// Evaluate all arguments
	values, err := evaluateArgs(e, args, nodeRange)
	if err != nil {
		return nil, err
	}

	// Sum values and track currency information
	sum := decimal.Zero
	currencyInfo := trackCurrencyTypes(values)

	for _, val := range values {
		decVal, _ := getDecimalValue(val)
		sum = sum.Add(decVal)
	}

	// Calculate average
	count := decimal.NewFromInt(int64(len(args)))
	average := sum.Div(count)

	// Return with appropriate type
	return createResultFromCurrencyInfo(average, currencyInfo), nil
}

// evalSqrt implements the sqrt() function
func evalSqrt(e *Evaluator, args []ast.Node, nodeRange *ast.Range) (types.Type, error) {
	if len(args) != 1 {
		return nil, &EvaluationError{
			Message: "sqrt() requires exactly one argument",
			Range:   nodeRange,
		}
	}

	arg, err := e.EvalNode(args[0])
	if err != nil {
		return nil, err
	}

	val, err := getDecimalValue(arg)
	if err != nil {
		return nil, &EvaluationError{Message: err.Error(), Range: nodeRange}
	}

	floatVal, _ := val.Float64()
	if floatVal < 0 {
		return nil, &EvaluationError{
			Message: "sqrt() of negative number is not supported",
			Range:   nodeRange,
		}
	}

	sqrtResult := math.Sqrt(floatVal)
	result := decimal.NewFromFloat(sqrtResult)

	// Preserve Currency type if input was Currency
	if curr, isCurrency := arg.(*types.Currency); isCurrency {
		return &types.Currency{Value: result, Symbol: curr.Symbol}, nil
	}
	return &types.Number{Value: result}, nil
}

// evaluateArgs evaluates all arguments in a function call
func evaluateArgs(e *Evaluator, args []ast.Node, nodeRange *ast.Range) ([]types.Type, error) {
	values := make([]types.Type, len(args))
	for i, arg := range args {
		result, err := e.EvalNode(arg)
		if err != nil {
			return nil, err
		}

		_, err = getDecimalValue(result)
		if err != nil {
			return nil, &EvaluationError{Message: err.Error(), Range: nodeRange}
		}

		values[i] = result
	}
	return values, nil
}

// currencyInfo tracks currency type information across multiple values
type currencyInfo struct {
	hasCurrency   bool
	currencySymbol string
	hasMixedUnits bool
}

// trackCurrencyTypes analyzes a list of values and determines currency status
func trackCurrencyTypes(values []types.Type) currencyInfo {
	info := currencyInfo{}

	for _, val := range values {
		if curr, isCurrency := val.(*types.Currency); isCurrency {
			if !info.hasCurrency {
				info.hasCurrency = true
				info.currencySymbol = curr.Symbol
			} else if info.currencySymbol != curr.Symbol {
				info.hasMixedUnits = true
			}
		} else if info.hasCurrency {
			// Number mixed with currency
			info.hasMixedUnits = true
		}
	}

	return info
}

// createResultFromCurrencyInfo creates the appropriate result type based on currency tracking
func createResultFromCurrencyInfo(value decimal.Decimal, info currencyInfo) types.Type {
	if info.hasCurrency && !info.hasMixedUnits {
		return &types.Currency{Value: value, Symbol: info.currencySymbol}
	}
	return &types.Number{Value: value}
}

// lookupFunction retrieves a function implementation by name
func lookupFunction(name string) (functionImpl, error) {
	impl, exists := functionRegistry[name]
	if !exists {
		return nil, fmt.Errorf("Unknown function: %s", name)
	}
	return impl, nil
}
