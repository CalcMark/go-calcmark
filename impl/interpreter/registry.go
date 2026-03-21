package interpreter

import (
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/CalcMark/go-calcmark/spec/features"
)

// FunctionInfo contains metadata about a CalcMark function for help output.
type FunctionInfo struct {
	Name        string   // Primary function name (e.g., "avg")
	Synonyms    []string // Alternative names (e.g., ["average", "mean"])
	Description string   // Human-readable description
	Signature   string   // Usage pattern (e.g., "avg(value1, value2, ...)")
	Category    string   // Grouping for help display (e.g., "Math", "Network", "Storage")
}

// toFunctionInfo builds a FunctionInfo by looking up the feature from spec/features.Registry.
func toFunctionInfo(name string) FunctionInfo {
	f := features.DefaultRegistry().GetByName(name)
	if f == nil {
		return FunctionInfo{Name: name}
	}
	return FunctionInfo{
		Name:        f.Name,
		Synonyms:    f.Synonyms,
		Description: f.Description,
		Signature:   f.Syntax,
		Category:    f.Subcategory,
	}
}

// GetAllFunctions returns all registered functions sorted by name.
// Reads from BuiltinFunctions (single source of truth for which functions exist)
// and enriches with metadata from the features registry.
func GetAllFunctions() []FunctionInfo {
	result := make([]FunctionInfo, len(BuiltinFunctions))
	for i, fn := range BuiltinFunctions {
		result[i] = toFunctionInfo(fn.Name)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// GetFunctionsByCategory returns functions grouped by category.
// Reads from BuiltinFunctions and enriches with metadata from the features registry.
func GetFunctionsByCategory() map[string][]FunctionInfo {
	result := make(map[string][]FunctionInfo)
	for _, fn := range BuiltinFunctions {
		info := toFunctionInfo(fn.Name)
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
		info := toFunctionInfo(fn.Name)
		if !seen[info.Category] {
			seen[info.Category] = true
			order = append(order, info.Category)
		}
	}
	return order
}

// GetFunctionNames returns all function names including synonyms.
// Reads from BuiltinFunctions and enriches with synonyms from the features registry.
func GetFunctionNames() []string {
	names := make([]string, 0, len(BuiltinFunctions)*2)
	for _, fn := range BuiltinFunctions {
		names = append(names, fn.Name)
		f := features.DefaultRegistry().GetByName(fn.Name)
		if f != nil {
			names = append(names, f.Synonyms...)
		}
	}
	return names
}

// GetFunctionByName looks up a function by name or synonym.
// Returns the FunctionInfo and true if found, or empty FunctionInfo and false if not.
func GetFunctionByName(name string) (FunctionInfo, bool) {
	for _, fn := range BuiltinFunctions {
		if fn.Name == name {
			return toFunctionInfo(fn.Name), true
		}
		f := features.DefaultRegistry().GetByName(fn.Name)
		if f != nil && slices.Contains(f.Synonyms, name) {
			return toFunctionInfo(fn.Name), true
		}
	}
	return FunctionInfo{}, false
}

// synonymMap maps synonym names to their canonical function name.
// Built once via sync.Once from the features registry.
var (
	synonymMap     map[string]string
	synonymMapOnce sync.Once
)

func getSynonymMap() map[string]string {
	synonymMapOnce.Do(func() {
		synonymMap = make(map[string]string)
		reg := features.DefaultRegistry()
		for _, fn := range BuiltinFunctions {
			f := reg.GetByName(fn.Name)
			if f == nil {
				continue
			}
			for _, syn := range f.Synonyms {
				synonymMap[strings.ToLower(syn)] = fn.Name
			}
		}
	})
	return synonymMap
}
