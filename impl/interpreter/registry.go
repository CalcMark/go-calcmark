package interpreter

import (
	"sort"
)

// FunctionInfo contains metadata about a CalcMark function for help output.
type FunctionInfo struct {
	Name        string   // Primary function name (e.g., "avg")
	Synonyms    []string // Alternative names (e.g., ["average"])
	Description string   // Human-readable description
	Signature   string   // Usage pattern (e.g., "avg(value1, value2, ...)")
	Category    string   // Grouping for help display (e.g., "Math", "Network", "Storage")
}

// FunctionRegistry contains metadata for all CalcMark functions.
// This is the single source of truth for help output.
var FunctionRegistry = []FunctionInfo{
	// Math functions
	{
		Name:        "avg",
		Synonyms:    []string{"average"},
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

// GetAllFunctions returns all registered functions sorted by name.
func GetAllFunctions() []FunctionInfo {
	sorted := make([]FunctionInfo, len(FunctionRegistry))
	copy(sorted, FunctionRegistry)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})
	return sorted
}

// GetFunctionsByCategory returns functions grouped by category.
func GetFunctionsByCategory() map[string][]FunctionInfo {
	result := make(map[string][]FunctionInfo)
	for _, fn := range FunctionRegistry {
		result[fn.Category] = append(result[fn.Category], fn)
	}
	// Sort functions within each category
	for category := range result {
		sort.Slice(result[category], func(i, j int) bool {
			return result[category][i].Name < result[category][j].Name
		})
	}
	return result
}

// GetFunctionNames returns all function names including synonyms.
// This is used for validation tests to ensure registry matches implementation.
func GetFunctionNames() []string {
	names := make([]string, 0, len(FunctionRegistry)*2)
	for _, fn := range FunctionRegistry {
		names = append(names, fn.Name)
		names = append(names, fn.Synonyms...)
	}
	return names
}
