package interpreter

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"
	"testing"
)

// TestRegistryMatchesFunctions verifies that every function name in the
// evalFunctionCall switch statement has a corresponding registry entry.
// This prevents registry drift from the implementation.
func TestRegistryMatchesFunctions(t *testing.T) {
	// Parse functions.go to extract function names from switch cases
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "functions.go", nil, 0)
	if err != nil {
		t.Fatalf("failed to parse functions.go: %v", err)
	}

	// Find all string literals in case clauses (function names)
	implementedFuncs := make(map[string]bool)
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CaseClause:
			for _, expr := range x.List {
				if lit, ok := expr.(*ast.BasicLit); ok && lit.Kind == token.STRING {
					// Remove quotes from string literal
					funcName := strings.Trim(lit.Value, `"`)
					implementedFuncs[funcName] = true
				}
			}
		}
		return true
	})

	// Also check for special case handling at the top of evalFunctionCall
	// These are handled before the switch: convert_rate, downtime, rtt, throughput, transfer_time, read, seek, compress, capacity
	specialCases := []string{
		"convert_rate", "downtime", "rtt", "throughput", "transfer_time",
		"read", "seek", "compress", "capacity",
	}
	for _, name := range specialCases {
		implementedFuncs[name] = true
	}

	// Get all registered function names including synonyms
	registeredFuncs := make(map[string]bool)
	for _, fn := range FunctionRegistry {
		registeredFuncs[fn.Name] = true
		for _, synonym := range fn.Synonyms {
			registeredFuncs[synonym] = true
		}
	}

	// Check that every implemented function is in the registry
	for funcName := range implementedFuncs {
		if !registeredFuncs[funcName] {
			t.Errorf("function %q is implemented but not in registry", funcName)
		}
	}

	// Check that every registered function is implemented
	for funcName := range registeredFuncs {
		if !implementedFuncs[funcName] {
			t.Errorf("function %q is in registry but not implemented", funcName)
		}
	}
}

// TestGetAllFunctionsSorted verifies that GetAllFunctions returns
// functions in alphabetical order by name.
func TestGetAllFunctionsSorted(t *testing.T) {
	functions := GetAllFunctions()

	if len(functions) == 0 {
		t.Fatal("GetAllFunctions returned empty slice")
	}

	// Verify sorted order
	names := make([]string, len(functions))
	for i, fn := range functions {
		names[i] = fn.Name
	}

	if !sort.StringsAreSorted(names) {
		t.Errorf("functions not sorted alphabetically: %v", names)
	}
}

// TestFunctionInfoComplete verifies all FunctionInfo have non-empty
// Name, Description, and Signature fields.
func TestFunctionInfoComplete(t *testing.T) {
	for _, fn := range FunctionRegistry {
		if fn.Name == "" {
			t.Error("FunctionInfo has empty Name")
		}
		if fn.Description == "" {
			t.Errorf("FunctionInfo %q has empty Description", fn.Name)
		}
		if fn.Signature == "" {
			t.Errorf("FunctionInfo %q has empty Signature", fn.Name)
		}
		if fn.Category == "" {
			t.Errorf("FunctionInfo %q has empty Category", fn.Name)
		}
	}
}

// TestGetFunctionsByCategory verifies grouping works correctly.
func TestGetFunctionsByCategory(t *testing.T) {
	byCategory := GetFunctionsByCategory()

	// Verify we have expected categories
	expectedCategories := []string{"Math", "Conversion", "Network", "Storage", "Capacity"}
	for _, category := range expectedCategories {
		if _, ok := byCategory[category]; !ok {
			t.Errorf("missing category %q", category)
		}
	}

	// Verify each category has at least one function
	for category, funcs := range byCategory {
		if len(funcs) == 0 {
			t.Errorf("category %q has no functions", category)
		}
	}

	// Verify functions within categories are sorted
	for category, funcs := range byCategory {
		names := make([]string, len(funcs))
		for i, fn := range funcs {
			names[i] = fn.Name
		}
		if !sort.StringsAreSorted(names) {
			t.Errorf("functions in category %q not sorted: %v", category, names)
		}
	}
}

// TestRegistryFunctionCount verifies we have the expected number of functions.
func TestRegistryFunctionCount(t *testing.T) {
	functions := GetAllFunctions()

	// We expect 12 primary functions based on the plan
	expectedCount := 12
	if len(functions) != expectedCount {
		t.Errorf("expected %d functions, got %d", expectedCount, len(functions))
	}
}
