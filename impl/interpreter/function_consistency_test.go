package interpreter

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/features"
	"github.com/CalcMark/go-calcmark/spec/types"
)

// TestFunctionSpecConsistency verifies that FunctionSpecs (spec layer) and
// BuiltinFunctions (interpreter layer) stay in sync. This catches the class
// of bug from #30 where one layer changes and the others silently fall behind.
//
// See: https://github.com/CalcMark/go-calcmark/issues/32

func TestEveryBuiltinFunctionHasFunctionSpec(t *testing.T) {
	for _, fn := range BuiltinFunctions {
		spec := types.GetFunctionSpec(fn.Name)
		if spec == nil {
			t.Errorf("BuiltinFunction %q has no corresponding FunctionSpec in spec/semantic", fn.Name)
		}
	}
}

func TestEveryFunctionSpecHasBuiltinFunction(t *testing.T) {
	for name := range types.FunctionSpecs {
		if _, found := GetFunctionByName(name); !found {
			t.Errorf("FunctionSpec %q has no corresponding BuiltinFunction in interpreter", name)
		}
	}
}

func TestEveryBuiltinFunctionHasEval(t *testing.T) {
	for _, fn := range BuiltinFunctions {
		if fn.Eval == nil {
			t.Errorf("BuiltinFunction %q has nil Eval — missing entry in functionEvalMap?", fn.Name)
		}
	}
}

// TestEveryBuiltinFunctionHasFeature checks that every BuiltinFunctions entry
// has a matching Feature in the spec/features registry.
func TestEveryBuiltinFunctionHasFeature(t *testing.T) {
	reg := features.NewRegistry()
	for _, fn := range BuiltinFunctions {
		f := reg.GetByName(fn.Name)
		if f == nil {
			t.Errorf("BuiltinFunction %q has no matching Feature in spec/features registry", fn.Name)
			continue
		}
		if f.Description == "" {
			t.Errorf("Feature for %q has empty Description", fn.Name)
		}
		if f.Syntax == "" {
			t.Errorf("Feature for %q has empty Syntax", fn.Name)
		}
		if f.Category == "" {
			t.Errorf("Feature for %q has empty Category", fn.Name)
		}
	}
}

func TestSuggestionTagCoversAllCategories(t *testing.T) {
	reg := features.NewRegistry()
	categories := make(map[string]bool)
	for _, fn := range BuiltinFunctions {
		f := reg.GetByName(fn.Name)
		if f != nil {
			categories[string(f.Category)] = true
		}
	}

	// suggestionTag is in the TUI package — we can't call it here.
	// Instead, we verify all categories are in the known set that
	// suggestionTag handles. If a new category is added, this test
	// will fail and remind the developer to update suggestionTag.
	knownCategories := map[string]bool{
		"function": true,
	}

	for cat := range categories {
		if !knownCategories[cat] {
			t.Errorf("BuiltinFunction category %q is not in the known set — "+
				"update suggestionTag() in view_overlays.go and this test", cat)
		}
	}
}
