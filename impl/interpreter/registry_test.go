package interpreter

import (
	"slices"
	"sort"
	"testing"
)

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

// TestFunctionInfoComplete verifies all FunctionDef in BuiltinFunctions have non-empty
// Name and Eval fields, and that their corresponding Feature has metadata.
func TestFunctionInfoComplete(t *testing.T) {
	for _, fn := range BuiltinFunctions {
		if fn.Name == "" {
			t.Error("FunctionDef has empty Name")
		}
		if fn.Eval == nil {
			t.Errorf("FunctionDef %q has nil Eval", fn.Name)
		}
		// Metadata now comes from the features registry via toFunctionInfo
		info := toFunctionInfo(fn.Name)
		if info.Description == "" {
			t.Errorf("FunctionDef %q has empty Description (from features registry)", fn.Name)
		}
		if info.Signature == "" {
			t.Errorf("FunctionDef %q has empty Signature (from features registry)", fn.Name)
		}
		if info.Category == "" {
			t.Errorf("FunctionDef %q has empty Category (from features registry)", fn.Name)
		}
	}
}

// TestGetFunctionsByCategory verifies grouping works correctly.
func TestGetFunctionsByCategory(t *testing.T) {
	byCategory := GetFunctionsByCategory()

	// Should have at least one category
	if len(byCategory) == 0 {
		t.Fatal("GetFunctionsByCategory returned empty map")
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

	// We expect 17 primary functions (12 original + 3 growth + 1 sum + 1 number)
	expectedCount := 17
	if len(functions) != expectedCount {
		t.Errorf("expected %d functions, got %d", expectedCount, len(functions))
	}
}

// TestGetFunctionByName verifies lookup by name and synonym.
func TestGetFunctionByName(t *testing.T) {
	tests := []struct {
		name     string
		lookup   string
		found    bool
		expected string // expected Name field if found
	}{
		{"primary name", "avg", true, "avg"},
		{"synonym average", "average", true, "avg"},
		{"synonym mean", "mean", true, "avg"},
		{"sqrt", "sqrt", true, "sqrt"},
		{"unknown", "unknown", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, found := GetFunctionByName(tt.lookup)
			if found != tt.found {
				t.Errorf("GetFunctionByName(%q): found = %v, want %v", tt.lookup, found, tt.found)
			}
			if tt.found && info.Name != tt.expected {
				t.Errorf("GetFunctionByName(%q): Name = %q, want %q", tt.lookup, info.Name, tt.expected)
			}
		})
	}
}

// TestAvgHasMeanSynonym verifies that "mean" is a synonym for avg (required for SC4).
func TestAvgHasMeanSynonym(t *testing.T) {
	info, found := GetFunctionByName("mean")
	if !found {
		t.Fatal("expected 'mean' to be a synonym for avg, but not found")
	}
	if info.Name != "avg" {
		t.Errorf("expected 'mean' to resolve to 'avg', got %q", info.Name)
	}
}

// TestGetFunctionNames verifies all names including synonyms are returned.
func TestGetFunctionNames(t *testing.T) {
	names := GetFunctionNames()

	// Should include primary names
	expectedPrimary := []string{"avg", "sum", "sqrt", "accumulate", "convert_rate", "downtime", "rtt", "throughput", "transfer_time", "read", "seek", "compress", "capacity"}
	for _, name := range expectedPrimary {
		if !slices.Contains(names, name) {
			t.Errorf("expected primary name %q in GetFunctionNames result", name)
		}
	}

	// Should include synonyms
	expectedSynonyms := []string{"average", "mean"}
	for _, syn := range expectedSynonyms {
		if !slices.Contains(names, syn) {
			t.Errorf("expected synonym %q in GetFunctionNames result", syn)
		}
	}
}
