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
			return nil, fmt.Errorf("convert_rate() requires exactly 2 arguments")
		}

		// Evaluate first argument (the rate)
		rateVal, err := interp.evalNode(f.Arguments[0])
		if err != nil {
			return nil, err
		}

		rate, ok := rateVal.(*types.Rate)
		if !ok {
			return nil, fmt.Errorf("convert_rate() first argument must be a rate, got %T", rateVal)
		}

		// Extract second argument as identifier (time unit) WITHOUT evaluating
		targetUnit, ok := f.Arguments[1].(*ast.Identifier)
		if !ok {
			return nil, fmt.Errorf("convert_rate() second argument must be a time unit identifier")
		}

		return convertRateTimeUnit(rate, targetUnit.Name)
	}

	// Special case: downtime's second argument should NOT be evaluated
	// It's an identifier representing a time period (month, year, etc), not a variable
	if f.Name == "downtime" {
		if len(f.Arguments) != 2 {
			return nil, fmt.Errorf("downtime() requires exactly 2 arguments")
		}

		// Evaluate first argument (availability percentage)
		availability, err := interp.evalNode(f.Arguments[0])
		if err != nil {
			return nil, err
		}

		// Second argument can be an Identifier (time unit) or evaluated duration
		// Try to extract as identifier first
		if identArg, ok := f.Arguments[1].(*ast.Identifier); ok {
			// Pass the identifier directly without evaluation
			return calculateDowntime(availability, identArg)
		}

		// Otherwise evaluate it (could be a Duration literal or expression)
		timePeriod, err := interp.evalNode(f.Arguments[1])
		if err != nil {
			return nil, err
		}

		return calculateDowntime(availability, timePeriod)
	}

	// Special case: rtt's argument should NOT be evaluated
	// It's an identifier representing network scope
	if f.Name == "rtt" {
		if len(f.Arguments) != 1 {
			return nil, fmt.Errorf("rtt() requires exactly 1 argument (scope)")
		}
		scopeIdent, ok := f.Arguments[0].(*ast.Identifier)
		if !ok {
			return nil, fmt.Errorf("rtt() scope must be an identifier (local, regional, continental, global)")
		}
		return calculateRTT(scopeIdent.Name)
	}

	// Special case: throughput's argument should NOT be evaluated
	// It's an identifier representing network type
	if f.Name == "throughput" {
		if len(f.Arguments) != 1 {
			return nil, fmt.Errorf("throughput() requires exactly 1 argument (network type)")
		}
		typeIdent, ok := f.Arguments[0].(*ast.Identifier)
		if !ok {
			return nil, fmt.Errorf("throughput() network type must be an identifier (gigabit, 10g, 100g, wifi, 4g, 5g)")
		}
		return calculateThroughput(typeIdent.Name)
	}

	// Special case: transfer_time's 2nd and 3rd arguments should NOT be evaluated
	if f.Name == "transfer_time" {
		if len(f.Arguments) != 3 {
			return nil, fmt.Errorf("transfer_time() requires exactly 3 arguments (size, scope, network_type)")
		}
		// Evaluate first argument (size)
		size, err := interp.evalNode(f.Arguments[0])
		if err != nil {
			return nil, err
		}
		sizeQty, ok := size.(*types.Quantity)
		if !ok {
			return nil, fmt.Errorf("transfer_time() size must be a quantity, got %T", size)
		}
		// Extract scope and network type as identifiers
		scopeIdent, ok := f.Arguments[1].(*ast.Identifier)
		if !ok {
			return nil, fmt.Errorf("transfer_time() scope must be an identifier")
		}
		typeIdent, ok := f.Arguments[2].(*ast.Identifier)
		if !ok {
			return nil, fmt.Errorf("transfer_time() network type must be an identifier")
		}
		return calculateTransferTime(sizeQty, scopeIdent.Name, typeIdent.Name)
	}

	// Special case: read's 2nd argument should NOT be evaluated
	// It's an identifier representing storage type
	if f.Name == "read" {
		if len(f.Arguments) != 2 {
			return nil, fmt.Errorf("read() requires exactly 2 arguments (size, storage_type)")
		}
		// Evaluate first argument (size)
		size, err := interp.evalNode(f.Arguments[0])
		if err != nil {
			return nil, err
		}
		sizeQty, ok := size.(*types.Quantity)
		if !ok {
			return nil, fmt.Errorf("read() size must be a quantity, got %T", size)
		}
		// Extract storage type as identifier
		storageIdent, ok := f.Arguments[1].(*ast.Identifier)
		if !ok {
			return nil, fmt.Errorf("read() storage type must be an identifier")
		}
		return calculateRead(sizeQty, storageIdent.Name)
	}

	// Special case: seek's argument should NOT be evaluated
	// It's an identifier representing storage type
	if f.Name == "seek" {
		if len(f.Arguments) != 1 {
			return nil, fmt.Errorf("seek() requires exactly 1 argument (storage_type)")
		}
		storageIdent, ok := f.Arguments[0].(*ast.Identifier)
		if !ok {
			return nil, fmt.Errorf("seek() storage type must be an identifier")
		}
		return calculateSeek(storageIdent.Name)
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
	case "requires":
		return evalRequires(args)
	case "downtime":
		// Already handled above
		return nil, fmt.Errorf("downtime should have been handled")
	case "rtt":
		// Already handled above
		return nil, fmt.Errorf("rtt should have been handled")
	case "throughput":
		// Already handled above
		return nil, fmt.Errorf("throughput should have been handled")
	case "transfer_time":
		// Already handled above
		return nil, fmt.Errorf("transfer_time should have been handled")
	case "read":
		// Already handled above
		return nil, fmt.Errorf("read should have been handled")
	case "seek":
		// Already handled above
		return nil, fmt.Errorf("seek should have been handled")
	default:
		return nil, fmt.Errorf("unknown function: %s", f.Name)
	}
}

// eval Downtime handles downtime(availability%, time_period) function calls.
func evalDowntime(args []types.Type) (types.Type, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("downtime() requires 2 arguments (availability%%, time_period)")
	}

	return calculateDowntime(args[0], args[1])
}

// evalRequires handles requires(load, capacity, buffer?) function calls.
func evalRequires(args []types.Type) (types.Type, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("requires() requires 2 or 3 arguments (load, capacity, buffer?)")
	}

	load := args[0]
	capacity := args[1]

	// Handle optional buffer
	if len(args) == 3 {
		// Extract buffer percentage value
		var bufferPercent decimal.Decimal
		switch buf := args[2].(type) {
		case *types.Number:
			bufferPercent = buf.Value
		default:
			return nil, fmt.Errorf("requires() buffer must be a percentage number, got %T", args[2])
		}

		return requiresCapacity(load, capacity, bufferPercent)
	}

	// No buffer - use the no-buffer version
	return requiresCapacityNoBuffer(load, capacity)
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
