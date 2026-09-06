package types

import "github.com/CalcMark/go-calcmark/v2/spec/identifiers"

// ArgType represents the semantic type expected for a function argument.
// This defines what kinds of values are valid for each parameter position.
type ArgType string

const (
	ArgTypeNumber     ArgType = "number"     // Plain number: 42, 3.14, 1e6
	ArgTypeQuantity   ArgType = "quantity"   // Number with unit: 10 meters, 5 kg, 100 MB
	ArgTypeRate       ArgType = "rate"       // Quantity per time: 10 MB/s, 60 km/h
	ArgTypeDuration   ArgType = "duration"   // Time span: 2 hours, 30 minutes
	ArgTypePercentage ArgType = "percentage" // Percentage: 99.9%, 15%
	ArgTypeString     ArgType = "string"     // String literal: "datacenter", "ssd"
	ArgTypeAny        ArgType = "any"        // Any value type
	// ArgTypeAmount accepts any value that can be added to itself —
	// Number, Quantity, or Currency. Excludes Percentage, Duration,
	// and Rate, which produce nonsensical results when combined with
	// `amount + (increment × periods)` style expressions. Used by
	// `grow`'s `amount` and `increment` params to keep the autosuggest
	// dropdown free of percentage variables.
	ArgTypeAmount ArgType = "amount"
)

// ParamSpec describes a function parameter with type constraints and examples.
type ParamSpec struct {
	Name       string   // Parameter name (e.g., "rate", "duration")
	Type       ArgType  // Expected argument type
	Optional   bool     // True if parameter has a default value
	Variadic   bool     // True if parameter accepts multiple values
	Examples   []string // Example values showing how to create this type
	EnumValues []string // Canonical unquoted enum values when the parameter accepts a finite identifier set (nil for free-form params)
}

// FunctionSpec describes a function's type signature for semantic analysis.
type FunctionSpec struct {
	Name   string      // Function name
	Params []ParamSpec // Parameter specifications
}

// ArgTypeExamples provides example values for each argument type.
var ArgTypeExamples = map[ArgType][]string{
	ArgTypeNumber:     {"42", "3.14", "1000", "1e6"},
	ArgTypeQuantity:   {"10 meters", "5 kg", "100 MB", "2.5 liters"},
	ArgTypeRate:       {"10 MB/s", "60 km/h", "100 requests/second", "5 GB/day"},
	ArgTypeDuration:   {"2 hours", "30 minutes", "1 day", "500 ms"},
	ArgTypePercentage: {"99.9%", "15%", "50%", "0.1%"},
	ArgTypeString:     {"ssd", "gzip", "gigabit"},
	ArgTypeAny:        {"<any value>"},
}

// FunctionSpecs defines the type signatures for all built-in functions.
// This is the authoritative source for parameter type information.
var FunctionSpecs = map[string]FunctionSpec{
	"avg": {
		Name: "avg",
		Params: []ParamSpec{
			{Name: "values", Type: ArgTypeAny, Variadic: true, Examples: []string{"1, 2, 3", "10, 20", "$100, $200"}},
		},
	},
	"sum": {
		Name: "sum",
		Params: []ParamSpec{
			{Name: "values", Type: ArgTypeAny, Variadic: true, Examples: []string{"1, 2, 3", "$100, $200", "1 kg, 500 g", "rates.rate"}},
		},
	},
	"min": {
		Name: "min",
		Params: []ParamSpec{
			{Name: "values", Type: ArgTypeAny, Variadic: true, Examples: []string{"4, 9, 1", "rates.rate"}},
		},
	},
	"max": {
		Name: "max",
		Params: []ParamSpec{
			{Name: "values", Type: ArgTypeAny, Variadic: true, Examples: []string{"4, 9, 1", "rates.rate"}},
		},
	},
	"count": {
		Name: "count",
		Params: []ParamSpec{
			{Name: "values", Type: ArgTypeAny, Variadic: true, Examples: []string{"rates.role", "a, b, c"}},
		},
	},
	"sqrt": {
		Name: "sqrt",
		Params: []ParamSpec{
			{Name: "value", Type: ArgTypeNumber, Examples: []string{"16", "2", "100"}},
		},
	},
	"number": {
		Name: "number",
		Params: []ParamSpec{
			{Name: "value", Type: ArgTypeAny, Examples: []string{"10 kg", "$100", "25%", "42"}},
		},
	},
	"accumulate": {
		Name: "accumulate",
		Params: []ParamSpec{
			{Name: "rate", Type: ArgTypeRate, Examples: []string{"10 MB/s", "500 req/s", "5 GB/day"}},
			{Name: "duration", Type: ArgTypeDuration, Examples: []string{"1 hour", "30 minutes", "1 day"}},
		},
	},
	"convert_rate": {
		Name: "convert_rate",
		Params: []ParamSpec{
			{Name: "rate", Type: ArgTypeRate, Examples: []string{"10 MB/s", "1 GB/hour", "500 req/s"}},
			{Name: "time_unit", Type: ArgTypeString, Examples: identifiers.TimeUnits, EnumValues: identifiers.TimeUnits},
		},
	},
	"downtime": {
		Name: "downtime",
		Params: []ParamSpec{
			{Name: "availability", Type: ArgTypePercentage, Examples: []string{"99.9%", "99.99%", "99%"}},
			{Name: "duration", Type: ArgTypeDuration, Examples: []string{"1 year", "1 month", "1 day"}},
		},
	},
	"rtt": {
		Name: "rtt",
		Params: []ParamSpec{
			{Name: "scope", Type: ArgTypeString, Examples: identifiers.NetworkScopes, EnumValues: identifiers.NetworkScopes},
		},
	},
	"throughput": {
		Name: "throughput",
		Params: []ParamSpec{
			{Name: "network_type", Type: ArgTypeString, Examples: identifiers.NetworkTypes, EnumValues: identifiers.NetworkTypes},
		},
	},
	"transfer_time": {
		Name: "transfer_time",
		Params: []ParamSpec{
			{Name: "size", Type: ArgTypeQuantity, Examples: []string{"1 GB", "100 MB", "10 TB"}},
			{Name: "scope", Type: ArgTypeString, Examples: identifiers.NetworkScopes, EnumValues: identifiers.NetworkScopes},
			{Name: "network_type", Type: ArgTypeString, Examples: identifiers.NetworkTypes, EnumValues: identifiers.NetworkTypes},
		},
	},
	"read": {
		Name: "read",
		Params: []ParamSpec{
			{Name: "size", Type: ArgTypeQuantity, Examples: []string{"1 MB", "4 KB", "1 GB"}},
			{Name: "storage_type", Type: ArgTypeString, Examples: identifiers.StorageTypes, EnumValues: identifiers.StorageTypes},
		},
	},
	"seek": {
		Name: "seek",
		Params: []ParamSpec{
			{Name: "storage_type", Type: ArgTypeString, Examples: identifiers.StorageTypes, EnumValues: identifiers.StorageTypes},
		},
	},
	"compress": {
		Name: "compress",
		Params: []ParamSpec{
			{Name: "size", Type: ArgTypeQuantity, Examples: []string{"1 MB", "100 KB", "1 GB"}},
			{Name: "compression_type", Type: ArgTypeString, Examples: identifiers.CompressionTypes, EnumValues: identifiers.CompressionTypes},
		},
	},
	"compound": {
		Name: "compound",
		Params: []ParamSpec{
			{Name: "principal", Type: ArgTypeAny, Examples: []string{"$10000", "1000 users", "$1M"}},
			{Name: "rate", Type: ArgTypePercentage, Examples: []string{"5%", "8.5%", "12%"}},
			{Name: "periods", Type: ArgTypeNumber, Examples: []string{"30", "12", "5"}},
			{Name: "period", Type: ArgTypeString, Optional: true, Examples: []string{"monthly", "quarterly", "yearly"}},
		},
	},
	"grow": {
		Name: "grow",
		Params: []ParamSpec{
			{Name: "amount", Type: ArgTypeAmount, Examples: []string{"$1000", "50 GB", "100 users"}},
			{Name: "increment", Type: ArgTypeAmount, Examples: []string{"$200", "10 GB", "50 users"}},
			{Name: "periods", Type: ArgTypeNumber, Examples: []string{"12", "6", "24"}},
		},
	},
	"depreciate": {
		Name: "depreciate",
		Params: []ParamSpec{
			{Name: "value", Type: ArgTypeAny, Examples: []string{"$50000", "$10000", "1000 units"}},
			{Name: "rate", Type: ArgTypePercentage, Examples: []string{"20%", "15%", "10%"}},
			{Name: "periods", Type: ArgTypeNumber, Examples: []string{"5", "10", "7"}},
			{Name: "salvage", Type: ArgTypeAny, Optional: true, Examples: []string{"$5000", "$1000"}},
		},
	},
	"capacity": {
		Name: "capacity",
		Params: []ParamSpec{
			{Name: "demand", Type: ArgTypeRate, Examples: []string{"1000 requests/second", "10 GB/day"}},
			{Name: "capacity_per_unit", Type: ArgTypeRate, Examples: []string{"100 requests/second", "1 GB/day"}},
			{Name: "unit", Type: ArgTypeString, Examples: []string{"server", "instance", "pod"}},
			{Name: "buffer", Type: ArgTypePercentage, Optional: true, Examples: []string{"20%", "50%"}},
		},
	},
}

// GetFunctionSpec returns the type specification for a function by name.
// Returns nil if the function is not found.
func GetFunctionSpec(name string) *FunctionSpec {
	if spec, ok := FunctionSpecs[name]; ok {
		return &spec
	}
	return nil
}

// GetParamAtIndex returns the parameter spec for the given argument position.
// Handles variadic parameters (last param applies to all subsequent args).
func (f *FunctionSpec) GetParamAtIndex(index int) *ParamSpec {
	if len(f.Params) == 0 {
		return nil
	}
	if index >= len(f.Params) {
		lastParam := &f.Params[len(f.Params)-1]
		if lastParam.Variadic {
			return lastParam
		}
		return nil
	}
	return &f.Params[index]
}

// GetExamplesForType returns example values for a given argument type.
func GetExamplesForType(argType ArgType) []string {
	if examples, ok := ArgTypeExamples[argType]; ok {
		return examples
	}
	return nil
}
