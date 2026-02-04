package interpreter

import (
	"fmt"
	"math"
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
		Description: "Calculate the average of numbers",
		Signature:   "avg(value1, value2, ...)",
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
}

// functionEvalMap maps function names to their Eval implementations.
// Used by init() to populate BuiltinFunctions.Eval fields.
var functionEvalMap = map[string]func(interp *Interpreter, f *ast.FunctionCall) (types.Type, error){
	"avg":           evalAvgFunc,
	"sqrt":          evalSqrtFunc,
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

func evalSqrtFunc(interp *Interpreter, f *ast.FunctionCall) (types.Type, error) {
	args, err := interp.evalAllArgs(f)
	if err != nil {
		return nil, err
	}
	return evalSqrt(args)
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
