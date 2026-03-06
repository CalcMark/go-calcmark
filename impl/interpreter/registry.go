package interpreter

import (
	"slices"
	"sort"
)

// FunctionInfo contains metadata about a CalcMark function for help output.
// This is derived from FunctionDef but excludes the Eval function pointer.
type FunctionInfo struct {
	Name        string   // Primary function name (e.g., "avg")
	Synonyms    []string // Alternative names (e.g., ["average", "mean"])
	Description string   // Human-readable description
	Signature   string   // Usage pattern (e.g., "avg(value1, value2, ...)")
	Category    string   // Grouping for help display (e.g., "Math", "Network", "Storage")
}

// toFunctionInfo converts a FunctionDef to FunctionInfo by excluding the Eval field.
func toFunctionInfo(fn FunctionDef) FunctionInfo {
	return FunctionInfo{
		Name:        fn.Name,
		Synonyms:    fn.Synonyms,
		Description: fn.Description,
		Signature:   fn.Signature,
		Category:    fn.Category,
	}
}

// GetAllFunctions returns all registered functions sorted by name.
// Reads from BuiltinFunctions (single source of truth).
func GetAllFunctions() []FunctionInfo {
	result := make([]FunctionInfo, len(BuiltinFunctions))
	for i, fn := range BuiltinFunctions {
		result[i] = toFunctionInfo(fn)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// GetFunctionsByCategory returns functions grouped by category.
// Reads from BuiltinFunctions (single source of truth).
func GetFunctionsByCategory() map[string][]FunctionInfo {
	result := make(map[string][]FunctionInfo)
	for _, fn := range BuiltinFunctions {
		info := toFunctionInfo(fn)
		result[info.Category] = append(result[info.Category], info)
	}
	// Sort functions within each category
	for category := range result {
		sort.Slice(result[category], func(i, j int) bool {
			return result[category][i].Name < result[category][j].Name
		})
	}
	return result
}

// GetCategoryOrder returns category names in their registration order
// (first-occurrence order from BuiltinFunctions).
func GetCategoryOrder() []string {
	seen := make(map[string]bool)
	var order []string
	for _, fn := range BuiltinFunctions {
		if !seen[fn.Category] {
			seen[fn.Category] = true
			order = append(order, fn.Category)
		}
	}
	return order
}

// GetFunctionNames returns all function names including synonyms.
// Reads from BuiltinFunctions (single source of truth).
func GetFunctionNames() []string {
	names := make([]string, 0, len(BuiltinFunctions)*2)
	for _, fn := range BuiltinFunctions {
		names = append(names, fn.Name)
		names = append(names, fn.Synonyms...)
	}
	return names
}

// GetFunctionByName looks up a function by name or synonym.
// Returns the FunctionInfo and true if found, or empty FunctionInfo and false if not.
func GetFunctionByName(name string) (FunctionInfo, bool) {
	for _, fn := range BuiltinFunctions {
		if fn.Name == name {
			return toFunctionInfo(fn), true
		}
		if slices.Contains(fn.Synonyms, name) {
			return toFunctionInfo(fn), true
		}
	}
	return FunctionInfo{}, false
}
