package interpreter

import (
	"fmt"
	"math"
	"math/big"
	"slices"
	"strings"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

// FunctionDef contains function metadata and implementation in one place.
// This is the single source of truth for help, autocomplete, and diagnostics.
type FunctionDef struct {
	Name        string   // Primary name (e.g., "avg")
	Synonyms    []string // Alternative names (e.g., ["average", "mean"])
	Description string   // Human-readable description
	Signature   string   // Usage pattern (e.g., "avg(value1, value2, ...)")
	Category    string   // Grouping: "Math", "Conversion", "Network", "Storage", "Capacity"
	// Eval is the function implementation. It receives the interpreter (for evaluating
	// arguments) and the AST node (for accessing raw arguments when needed).
	// This field is populated in init() to avoid initialization cycles.
	Eval func(interp *Interpreter, f *ast.FunctionCall) (types.Type, error)
}

// BuiltinFunctions is the single source of truth for all CalcMark functions.
// Adding a function here automatically makes it available in help, autocomplete,
// and the interpreter. Missing any field will cause a compile-time or test failure.
// Note: Eval fields are populated in init() to avoid initialization cycles.
var BuiltinFunctions = []FunctionDef{
	// Math functions
	{
		Name:        "avg",
		Synonyms:    []string{"average", "mean"},
		Description: "Calculate the average of values",
		Signature:   "avg(value1, value2, ...)",
		Category:    "Math",
	},
	{
		Name:        "sum",
		Synonyms:    []string{},
		Description: "Calculate the sum of values",
		Signature:   "sum(value1, value2, ...)",
		Category:    "Math",
	},
	{
		Name:        "sqrt",
		Synonyms:    []string{},
		Description: "Calculate square root of a number",
		Signature:   "sqrt(value)",
		Category:    "Math",
	},
	{
		Name:        "number",
		Synonyms:    []string{},
		Description: "Extract the numeric value from any type",
		Signature:   "number(value)",
		Category:    "Math",
	},
	{
		Name:        "accumulate",
		Synonyms:    []string{},
		Description: "Accumulate a rate over time (e.g., requests/sec over 1 day)",
		Signature:   "accumulate(rate, time_period)",
		Category:    "Math",
	},

	// Conversion functions
	{
		Name:        "convert_rate",
		Synonyms:    []string{},
		Description: "Convert rate to different time unit",
		Signature:   "convert_rate(rate, time_unit)",
		Category:    "Conversion",
	},

	// Network functions
	{
		Name:        "downtime",
		Synonyms:    []string{},
		Description: "Calculate downtime from availability percentage",
		Signature:   "downtime(availability_percent, time_period)",
		Category:    "Network",
	},
	{
		Name:        "rtt",
		Synonyms:    []string{},
		Description: "Get round-trip time for network scope",
		Signature:   "rtt(scope)",
		Category:    "Network",
	},
	{
		Name:        "throughput",
		Synonyms:    []string{},
		Description: "Get throughput for network type",
		Signature:   "throughput(network_type)",
		Category:    "Network",
	},
	{
		Name:        "transfer_time",
		Synonyms:    []string{},
		Description: "Calculate data transfer time",
		Signature:   "transfer_time(size, scope, network_type)",
		Category:    "Network",
	},

	// Storage functions
	{
		Name:        "read",
		Synonyms:    []string{},
		Description: "Calculate storage read time",
		Signature:   "read(size, storage_type)",
		Category:    "Storage",
	},
	{
		Name:        "seek",
		Synonyms:    []string{},
		Description: "Get storage seek latency",
		Signature:   "seek(storage_type)",
		Category:    "Storage",
	},
	{
		Name:        "compress",
		Synonyms:    []string{},
		Description: "Calculate compression time",
		Signature:   "compress(size, compression_type)",
		Category:    "Storage",
	},

	// Capacity functions
	{
		Name:        "capacity",
		Synonyms:    []string{},
		Description: "Calculate required capacity for demand",
		Signature:   "capacity(demand, capacity_per_unit, unit, buffer?)",
		Category:    "Capacity",
	},

	// Growth functions
	{
		Name:        "compound",
		Synonyms:    []string{},
		Description: "Compound growth: A = P(1+r)^n",
		Signature:   "compound(principal, rate, duration, period?)",
		Category:    "Growth",
	},
	{
		Name:        "grow",
		Synonyms:    []string{},
		Description: "Linear growth: A = P + (increment x periods)",
		Signature:   "grow(starting_amount, increment, duration)",
		Category:    "Growth",
	},
	{
		Name:        "depreciate",
		Synonyms:    []string{},
		Description: "Declining balance depreciation",
		Signature:   "depreciate(principal, rate, duration, salvage?)",
		Category:    "Growth",
	},
}

// functionEvalMap maps function names to their Eval implementations.
// Used by init() to populate BuiltinFunctions.Eval fields.
var functionEvalMap = map[string]func(interp *Interpreter, f *ast.FunctionCall) (types.Type, error){
	"avg":           evalAvgFunc,
	"sum":           evalSumFunc,
	"sqrt":          evalSqrtFunc,
	"number":        evalNumberFunc,
	"accumulate":    evalAccumulateFunc,
	"convert_rate":  evalConvertRateFunc,
	"downtime":      evalDowntimeFunc,
	"rtt":           evalRTTFunc,
	"throughput":    evalThroughputFunc,
	"transfer_time": evalTransferTimeFunc,
	"read":          evalReadFunc,
	"seek":          evalSeekFunc,
	"compress":      evalCompressFunc,
	"capacity":      evalCapacityFunc,
	"compound":      evalCompoundFunc,
	"grow":          evalGrowFunc,
	"depreciate":    evalDepreciateFunc,
}

func init() {
	// Populate Eval fields after all functions are defined to avoid initialization cycles.
	for i := range BuiltinFunctions {
		fn := &BuiltinFunctions[i]
		if evalFn, ok := functionEvalMap[fn.Name]; ok {
			fn.Eval = evalFn
		}
	}
}

// evalFunctionCall dispatches function calls to the appropriate implementation.
func (interp *Interpreter) evalFunctionCall(f *ast.FunctionCall) (types.Type, error) {
	name := strings.ToLower(f.Name)
	for _, fn := range BuiltinFunctions {
		if fn.Name == name || slices.Contains(fn.Synonyms, name) {
			return fn.Eval(interp, f)
		}
	}
	return nil, fmt.Errorf("unknown function: %s", f.Name)
}

// Function implementations that wrap the interpreter access pattern.
// Each function either evaluates its arguments (simple functions) or
// handles raw AST nodes (functions with identifier arguments).

func evalAvgFunc(interp *Interpreter, f *ast.FunctionCall) (types.Type, error) {
	args, err := interp.evalAllArgs(f)
	if err != nil {
		return nil, err
	}
	return evalAverage(args)
}

func evalSumFunc(interp *Interpreter, f *ast.FunctionCall) (types.Type, error) {
	args, err := interp.evalAllArgs(f)
	if err != nil {
		return nil, err
	}
	return evalSum(args)
}

func evalSqrtFunc(interp *Interpreter, f *ast.FunctionCall) (types.Type, error) {
	args, err := interp.evalAllArgs(f)
	if err != nil {
		return nil, err
	}
	return evalSqrt(args)
}

func evalNumberFunc(interp *Interpreter, f *ast.FunctionCall) (types.Type, error) {
	args, err := interp.evalAllArgs(f)
	if err != nil {
		return nil, err
	}
	return evalNumber(args)
}

// evalNumber extracts the numeric value from any type.
// Returns a plain Number with the value stripped of units, currency, etc.
func evalNumber(args []types.Type) (types.Type, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("number() requires exactly 1 argument, got %d", len(args))
	}
	return ExtractNumber(args[0])
}

// ExtractNumber extracts the decimal value from any CalcMark type as a plain Number.
// Exported so that operator coercion (Quantity × Currency) can reuse the logic.
func ExtractNumber(val types.Type) (*types.Number, error) {
	switch v := val.(type) {
	case *types.Number:
		return v, nil
	case *types.Quantity:
		return types.NewNumber(v.Value), nil
	case *types.Currency:
		return types.NewNumber(v.Value), nil
	case *types.Percentage:
		return types.NewNumber(v.Value), nil
	case *types.Duration:
		return types.NewNumber(v.Value), nil
	case *types.Fraction:
		return types.NewNumber(decimal.NewFromBigRat(v.Value, 15)), nil
	case *types.Rate:
		return types.NewNumber(v.Amount.Value), nil
	default:
		return nil, fmt.Errorf("number() cannot extract numeric value from %T", val)
	}
}

func evalAccumulateFunc(interp *Interpreter, f *ast.FunctionCall) (types.Type, error) {
	args, err := interp.evalAllArgs(f)
	if err != nil {
		return nil, err
	}
	return evalAccumulate(args)
}

func evalConvertRateFunc(interp *Interpreter, f *ast.FunctionCall) (types.Type, error) {
	// Second argument is an identifier (time unit), NOT evaluated
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

func evalDowntimeFunc(interp *Interpreter, f *ast.FunctionCall) (types.Type, error) {
	// Second argument can be an identifier (time period) or evaluated duration
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

func evalRTTFunc(interp *Interpreter, f *ast.FunctionCall) (types.Type, error) {
	// Argument is an identifier (network scope), NOT evaluated
	if len(f.Arguments) != 1 {
		return nil, fmt.Errorf("rtt() requires exactly 1 argument (scope)")
	}
	scopeIdent, ok := f.Arguments[0].(*ast.Identifier)
	if !ok {
		return nil, fmt.Errorf("rtt() scope must be an identifier (local, regional, continental, global)")
	}
	return calculateRTT(scopeIdent.Name)
}

func evalThroughputFunc(interp *Interpreter, f *ast.FunctionCall) (types.Type, error) {
	// Argument is an identifier (network type), NOT evaluated
	if len(f.Arguments) != 1 {
		return nil, fmt.Errorf("throughput() requires exactly 1 argument (network type)")
	}
	typeIdent, ok := f.Arguments[0].(*ast.Identifier)
	if !ok {
		return nil, fmt.Errorf("throughput() network type must be an identifier (gigabit, ten_gig, hundred_gig, wifi, four_g, five_g)")
	}
	return calculateThroughput(typeIdent.Name)
}

func evalTransferTimeFunc(interp *Interpreter, f *ast.FunctionCall) (types.Type, error) {
	// 2nd and 3rd arguments are identifiers, NOT evaluated
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
		return nil, fmt.Errorf("transfer_time() scope must be an identifier (local, regional, continental, global)")
	}
	typeIdent, ok := f.Arguments[2].(*ast.Identifier)
	if !ok {
		return nil, fmt.Errorf("transfer_time() network type must be an identifier (gigabit, ten_gig, hundred_gig, wifi, four_g, five_g)")
	}
	return calculateTransferTime(sizeQty, scopeIdent.Name, typeIdent.Name)
}

func evalReadFunc(interp *Interpreter, f *ast.FunctionCall) (types.Type, error) {
	// 2nd argument is an identifier (storage type), NOT evaluated
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

func evalSeekFunc(interp *Interpreter, f *ast.FunctionCall) (types.Type, error) {
	// Argument is an identifier (storage type), NOT evaluated
	if len(f.Arguments) != 1 {
		return nil, fmt.Errorf("seek() requires exactly 1 argument (storage_type)")
	}
	storageIdent, ok := f.Arguments[0].(*ast.Identifier)
	if !ok {
		return nil, fmt.Errorf("seek() storage type must be an identifier")
	}
	return calculateSeek(storageIdent.Name)
}

func evalCompressFunc(interp *Interpreter, f *ast.FunctionCall) (types.Type, error) {
	// 2nd argument is an identifier (compression type), NOT evaluated
	if len(f.Arguments) != 2 {
		return nil, fmt.Errorf("compress() requires exactly 2 arguments (size, compression_type)")
	}
	// Evaluate first argument (size)
	size, err := interp.evalNode(f.Arguments[0])
	if err != nil {
		return nil, err
	}
	sizeQty, ok := size.(*types.Quantity)
	if !ok {
		return nil, fmt.Errorf("compress() size must be a quantity, got %T", size)
	}
	// Extract compression type as identifier
	compressionIdent, ok := f.Arguments[1].(*ast.Identifier)
	if !ok {
		return nil, fmt.Errorf("compress() compression type must be an identifier")
	}
	return calculateCompression(sizeQty, compressionIdent.Name)
}

func evalCapacityFunc(interp *Interpreter, f *ast.FunctionCall) (types.Type, error) {
	// Third argument is an identifier (unit name), NOT evaluated
	if len(f.Arguments) < 3 || len(f.Arguments) > 4 {
		return nil, fmt.Errorf("capacity() requires 3 or 4 arguments (demand, capacity, unit, buffer?)")
	}

	// Evaluate first argument (demand)
	demand, err := interp.evalNode(f.Arguments[0])
	if err != nil {
		return nil, err
	}

	// Evaluate second argument (capacity per unit)
	capacityVal, err := interp.evalNode(f.Arguments[1])
	if err != nil {
		return nil, err
	}

	// Third argument is the unit identifier (NOT evaluated)
	unitIdent, ok := f.Arguments[2].(*ast.Identifier)
	if !ok {
		return nil, fmt.Errorf("capacity() unit must be an identifier (e.g., disk, server, crate)")
	}
	unitName := unitIdent.Name

	// Check for optional buffer (4th argument)
	if len(f.Arguments) == 4 {
		bufferVal, err := interp.evalNode(f.Arguments[3])
		if err != nil {
			return nil, err
		}
		// Extract buffer percentage as decimal
		var bufferPercent decimal.Decimal
		switch buf := bufferVal.(type) {
		case *types.Percentage:
			bufferPercent = buf.Value
		case *types.Number:
			bufferPercent = buf.Value
		default:
			return nil, fmt.Errorf("capacity() buffer must be a percentage number, got %T", bufferVal)
		}
		return capacityAtWithBuffer(demand, capacityVal, unitName, bufferPercent)
	}

	// No buffer
	return capacityAt(demand, capacityVal, unitName)
}

// evalAllArgs evaluates all function arguments and returns them as typed values.
// This is used by simple functions that don't need raw AST access.
func (interp *Interpreter) evalAllArgs(f *ast.FunctionCall) ([]types.Type, error) {
	args := make([]types.Type, len(f.Arguments))
	for i, arg := range f.Arguments {
		val, err := interp.evalNode(arg)
		if err != nil {
			return nil, err
		}
		args[i] = val
	}
	return args, nil
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

// evalAverage calculates the average of values.
// Supports Number, Currency, Quantity, Duration, and Percentage.
// Preserves currency type when all arguments share the same currency.
func evalAverage(args []types.Type) (types.Type, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("avg() requires at least one argument")
	}

	// Check if first arg is Quantity, Duration, or Percentage — use aggregateValues path
	switch args[0].(type) {
	case *types.Quantity, *types.Duration, *types.Percentage:
		result, err := aggregateValues("avg", args)
		if err != nil {
			return nil, err
		}
		// Divide by count to get average
		count := decimal.NewFromInt(int64(len(args)))
		switch v := result.(type) {
		case *types.Quantity:
			return &types.Quantity{Value: v.Value.Div(count), Unit: v.Unit}, nil
		case *types.Duration:
			return &types.Duration{Value: v.Value.Div(count), Unit: v.Unit}, nil
		case *types.Percentage:
			return types.NewPercentage(v.Value.Div(count)), nil
		default:
			return nil, fmt.Errorf("avg(): unexpected aggregate result type %T", result)
		}
	}

	// Number/Currency path (backwards compatible)
	numbers, err := extractNumbers(args)
	if err != nil {
		return nil, err
	}

	sum := numbers[0]
	for i := 1; i < len(numbers); i++ {
		sum = sum.Add(numbers[i])
	}

	count := len(numbers)
	avg := sum.Div(decimal.NewFromInt(int64(count)))

	if symbol, ok := uniformCurrency(args); ok {
		return types.NewCurrency(avg, symbol), nil
	}

	return types.NewNumber(avg), nil
}

// evalSum calculates the sum of values.
// Supports Number, Currency, Quantity, Duration, and Percentage.
func evalSum(args []types.Type) (types.Type, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("sum() requires at least 2 arguments, got %d", len(args))
	}

	return aggregateValues("sum", args)
}

// aggregateValues sums typed values with automatic unit conversion.
// The first argument determines the result type and target unit.
// All arguments must be the same type family.
func aggregateValues(funcName string, args []types.Type) (types.Type, error) {
	first := args[0]

	switch f := first.(type) {
	case *types.Number:
		sum := f.Value
		for _, arg := range args[1:] {
			n, ok := arg.(*types.Number)
			if !ok {
				return nil, fmt.Errorf("%s(): expected number, got %s", funcName, formatTypeForError(arg))
			}
			sum = sum.Add(n.Value)
		}
		return types.NewNumber(sum), nil

	case *types.Currency:
		sum := f.Value
		for _, arg := range args[1:] {
			c, ok := arg.(*types.Currency)
			if !ok {
				return nil, fmt.Errorf("%s(): expected currency, got %s", funcName, formatTypeForError(arg))
			}
			if c.Code != f.Code {
				return nil, fmt.Errorf("%s(): cannot mix currencies %s and %s; convert explicitly first", funcName, f.Code, c.Code)
			}
			sum = sum.Add(c.Value)
		}
		return types.NewCurrency(sum, f.Symbol), nil

	case *types.Quantity:
		sum := f.Value
		for _, arg := range args[1:] {
			q, ok := arg.(*types.Quantity)
			if !ok {
				return nil, fmt.Errorf("%s(): expected quantity, got %s", funcName, formatTypeForError(arg))
			}
			converted, err := convertQuantity(q, f.Unit)
			if err != nil {
				return nil, fmt.Errorf("%s(): %w", funcName, err)
			}
			sum = sum.Add(converted.Value)
		}
		return &types.Quantity{Value: sum, Unit: f.Unit}, nil

	case *types.Duration:
		sum := f.Value
		for _, arg := range args[1:] {
			d, ok := arg.(*types.Duration)
			if !ok {
				return nil, fmt.Errorf("%s(): expected duration, got %s", funcName, formatTypeForError(arg))
			}
			converted, err := d.Convert(f.Unit)
			if err != nil {
				return nil, fmt.Errorf("%s(): %w", funcName, err)
			}
			sum = sum.Add(converted.Value)
		}
		return &types.Duration{Value: sum, Unit: f.Unit}, nil

	case *types.Fraction:
		sum := new(big.Rat).Set(f.Value)
		for _, arg := range args[1:] {
			fr, ok := arg.(*types.Fraction)
			if !ok {
				return nil, fmt.Errorf("%s(): expected fraction, got %s", funcName, formatTypeForError(arg))
			}
			sum.Add(sum, fr.Value)
		}
		result := types.NewFractionFromRat(sum)
		result.Unit = f.Unit
		if result.ExceedsComputationLimit() {
			return fractionToNumber(result), nil
		}
		return result, nil

	case *types.Percentage:
		sum := f.Value
		for _, arg := range args[1:] {
			p, ok := arg.(*types.Percentage)
			if !ok {
				return nil, fmt.Errorf("%s(): expected percentage, got %s", funcName, formatTypeForError(arg))
			}
			sum = sum.Add(p.Value)
		}
		return types.NewPercentage(sum), nil

	default:
		return nil, fmt.Errorf("%s(): unsupported type %s", funcName, formatTypeForError(first))
	}
}

// evalSqrt calculates the square root.
func evalSqrt(args []types.Type) (types.Type, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("sqrt() requires exactly one argument")
	}

	// Fraction sqrt: exact result if both num and denom are perfect squares
	if frac, ok := args[0].(*types.Fraction); ok {
		if frac.Value.Sign() < 0 {
			return nil, fmt.Errorf("sqrt() argument must be non-negative")
		}
		absNum := new(big.Int).Abs(frac.Num())
		absDenom := frac.Denom()
		if isPerfectSquare(absNum) && isPerfectSquare(absDenom) {
			rootNum := isqrt(absNum)
			rootDenom := isqrt(absDenom)
			result := new(big.Rat).SetFrac(rootNum, rootDenom)
			return types.NewFractionFromRat(result), nil
		}
		// Not perfect squares — fall through to decimal sqrt
		args[0] = fractionToNumber(frac)
	}

	num, ok := args[0].(*types.Number)
	if !ok {
		return nil, fmt.Errorf("sqrt() argument must be a number")
	}

	// Use simple Newton's method for sqrt since decimal doesn't have built-in
	if num.Value.IsNegative() {
		return nil, fmt.Errorf("sqrt() argument must be non-negative")
	}

	// Precision note: converts through float64 — sufficient for typical use cases
	// but may lose precision for numbers exceeding float64 range.
	f, _ := num.Value.Float64()
	result := decimal.NewFromFloat(math.Sqrt(f))

	return types.NewNumber(result), nil
}

// uniformCurrency returns the shared currency symbol if all args are
// Currency with the same symbol. Returns ("", false) otherwise.
func uniformCurrency(args []types.Type) (string, bool) {
	if len(args) == 0 {
		return "", false
	}
	first, ok := args[0].(*types.Currency)
	if !ok {
		return "", false
	}
	for _, arg := range args[1:] {
		c, ok := arg.(*types.Currency)
		if !ok || c.Symbol != first.Symbol {
			return "", false
		}
	}
	return first.Symbol, true
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
