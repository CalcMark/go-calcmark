package semantic

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
)

// ParamSpec describes a function parameter with type constraints and examples.
// This is the semantic definition - what types are valid for this parameter.
type ParamSpec struct {
	Name     string   // Parameter name (e.g., "rate", "duration")
	Type     ArgType  // Expected argument type
	Optional bool     // True if parameter has a default value
	Variadic bool     // True if parameter accepts multiple values
	Examples []string // Example values showing how to create this type
}

// FunctionSpec describes a function's type signature for semantic analysis.
// This is separate from the implementation - it defines what's valid.
type FunctionSpec struct {
	Name   string      // Function name
	Params []ParamSpec // Parameter specifications
}

// ArgTypeExamples provides example values for each argument type.
// Used to show contextual help when user is typing function arguments.
var ArgTypeExamples = map[ArgType][]string{
	ArgTypeNumber:     {"42", "3.14", "1000", "1e6"},
	ArgTypeQuantity:   {"10 meters", "5 kg", "100 MB", "2.5 liters"},
	ArgTypeRate:       {"10 MB/s", "60 km/h", "100 requests/second", "5 GB/day"},
	ArgTypeDuration:   {"2 hours", "30 minutes", "1 day", "500 ms"},
	ArgTypePercentage: {"99.9%", "15%", "50%", "0.1%"},
	ArgTypeString:     {`"datacenter"`, `"ssd"`, `"gzip"`},
	ArgTypeAny:        {"<any value>"},
}

// FunctionSpecs defines the type signatures for all built-in functions.
// This is the authoritative source for parameter type information.
var FunctionSpecs = map[string]FunctionSpec{
	// Math functions
	"avg": {
		Name: "avg",
		Params: []ParamSpec{
			{Name: "values", Type: ArgTypeNumber, Variadic: true, Examples: []string{"1, 2, 3", "10, 20", "x, y, z"}},
		},
	},
	"sqrt": {
		Name: "sqrt",
		Params: []ParamSpec{
			{Name: "value", Type: ArgTypeNumber, Examples: []string{"16", "2", "100"}},
		},
	},
	"accumulate": {
		Name: "accumulate",
		Params: []ParamSpec{
			{Name: "rate", Type: ArgTypeRate, Examples: []string{"10 MB/s", "100 requests/second", "5 GB/day"}},
			{Name: "duration", Type: ArgTypeDuration, Examples: []string{"1 hour", "30 minutes", "1 day"}},
		},
	},

	// Conversion functions
	"convert_rate": {
		Name: "convert_rate",
		Params: []ParamSpec{
			{Name: "rate", Type: ArgTypeRate, Examples: []string{"10 MB/s", "1 GB/hour", "100 req/s"}},
			{Name: "time_unit", Type: ArgTypeString, Examples: []string{`"per second"`, `"per hour"`, `"per day"`}},
		},
	},

	// Network functions
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
			{Name: "scope", Type: ArgTypeString, Examples: []string{`"local"`, `"regional"`, `"continental"`, `"global"`}},
		},
	},
	"throughput": {
		Name: "throughput",
		Params: []ParamSpec{
			{Name: "network_type", Type: ArgTypeString, Examples: []string{`"gigabit"`, `"ten_gig"`, `"wifi"`, `"four_g"`, `"five_g"`}},
		},
	},
	"transfer_time": {
		Name: "transfer_time",
		Params: []ParamSpec{
			{Name: "size", Type: ArgTypeQuantity, Examples: []string{"1 GB", "100 MB", "10 TB"}},
			{Name: "scope", Type: ArgTypeString, Examples: []string{`"local"`, `"regional"`, `"continental"`, `"global"`}},
			{Name: "network_type", Type: ArgTypeString, Examples: []string{`"gigabit"`, `"ten_gig"`, `"wifi"`}},
		},
	},

	// Storage functions
	"read": {
		Name: "read",
		Params: []ParamSpec{
			{Name: "size", Type: ArgTypeQuantity, Examples: []string{"1 MB", "4 KB", "1 GB"}},
			{Name: "storage_type", Type: ArgTypeString, Examples: []string{`"ssd"`, `"hdd"`, `"memory"`}},
		},
	},
	"seek": {
		Name: "seek",
		Params: []ParamSpec{
			{Name: "storage_type", Type: ArgTypeString, Examples: []string{`"ssd"`, `"hdd"`, `"memory"`}},
		},
	},
	"compress": {
		Name: "compress",
		Params: []ParamSpec{
			{Name: "size", Type: ArgTypeQuantity, Examples: []string{"1 MB", "100 KB", "1 GB"}},
			{Name: "compression_type", Type: ArgTypeString, Examples: []string{`"gzip"`, `"snappy"`, `"zstd"`}},
		},
	},

	// Growth functions
	"compound": {
		Name: "compound",
		Params: []ParamSpec{
			{Name: "principal", Type: ArgTypeAny, Examples: []string{"1000", "$1000", "100 users"}},
			{Name: "rate", Type: ArgTypePercentage, Examples: []string{"5%", "0.05", "12%"}},
			{Name: "periods", Type: ArgTypeNumber, Examples: []string{"10", "12", "5"}},
		},
	},
	"grow": {
		Name: "grow",
		Params: []ParamSpec{
			{Name: "amount", Type: ArgTypeAny, Examples: []string{"100", "50 GB", "1000 users"}},
			{Name: "increment", Type: ArgTypeAny, Examples: []string{"20 GB", "100 users", "5"}},
			{Name: "periods", Type: ArgTypeNumber, Examples: []string{"12", "6", "24"}},
		},
	},
	"depreciate": {
		Name: "depreciate",
		Params: []ParamSpec{
			{Name: "value", Type: ArgTypeAny, Examples: []string{"10000", "$50000", "1000 units"}},
			{Name: "rate", Type: ArgTypePercentage, Examples: []string{"20%", "0.15", "10%"}},
			{Name: "periods", Type: ArgTypeNumber, Examples: []string{"5", "10", "7"}},
		},
	},

	// Capacity functions
	"capacity": {
		Name: "capacity",
		Params: []ParamSpec{
			{Name: "demand", Type: ArgTypeRate, Examples: []string{"1000 requests/second", "10 GB/day"}},
			{Name: "capacity_per_unit", Type: ArgTypeRate, Examples: []string{"100 requests/second", "1 GB/day"}},
			{Name: "unit", Type: ArgTypeString, Examples: []string{`"servers"`, `"instances"`}},
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

	// If index is beyond params, check if last param is variadic
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
