package interpreter

import (
	"fmt"
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/semantic"
)

// TestFunctionSpecConsistency verifies that FunctionSpecs (spec layer) and
// BuiltinFunctions (interpreter layer) stay in sync. This catches the class
// of bug from #30 where one layer changes and the others silently fall behind.
//
// See: https://github.com/CalcMark/go-calcmark/issues/32

func TestEveryBuiltinFunctionHasFunctionSpec(t *testing.T) {
	for _, fn := range BuiltinFunctions {
		spec := semantic.GetFunctionSpec(fn.Name)
		if spec == nil {
			t.Errorf("BuiltinFunction %q has no corresponding FunctionSpec in spec/semantic", fn.Name)
		}
	}
}

func TestEveryFunctionSpecHasBuiltinFunction(t *testing.T) {
	for name := range semantic.FunctionSpecs {
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

func TestArgCountConsistency(t *testing.T) {
	for _, fn := range BuiltinFunctions {
		spec := semantic.GetFunctionSpec(fn.Name)
		if spec == nil {
			continue // covered by TestEveryBuiltinFunctionHasFunctionSpec
		}

		// Derive min/max arg count from FunctionSpec
		specMin, specMax := specArgRange(spec)

		// Derive min/max arg count from the interpreter's Signature string
		sigMin, sigMax, err := signatureArgRange(fn.Signature)
		if err != nil {
			t.Errorf("%s: cannot parse signature %q: %v", fn.Name, fn.Signature, err)
			continue
		}

		if specMin != sigMin || specMax != sigMax {
			t.Errorf("%s: FunctionSpec arg range [%d, %d] != Signature %q arg range [%d, %d]",
				fn.Name, specMin, specMax, fn.Signature, sigMin, sigMax)
		}
	}
}

func TestSignatureParamCountMatchesFunctionSpec(t *testing.T) {
	for _, fn := range BuiltinFunctions {
		spec := semantic.GetFunctionSpec(fn.Name)
		if spec == nil {
			continue
		}

		sigParams := parseSignatureParams(fn.Signature)
		specParamCount := len(spec.Params)

		// For variadic functions, signature has "..." which doesn't map 1:1
		// to param count, so we skip strict count comparison.
		hasVariadic := false
		for _, p := range spec.Params {
			if p.Variadic {
				hasVariadic = true
				break
			}
		}
		if hasVariadic {
			continue
		}

		if len(sigParams) != specParamCount {
			t.Errorf("%s: Signature %q has %d params, FunctionSpec has %d params",
				fn.Name, fn.Signature, len(sigParams), specParamCount)
		}
	}
}

func TestSuggestionTagCoversAllCategories(t *testing.T) {
	categories := make(map[string]bool)
	for _, fn := range BuiltinFunctions {
		categories[fn.Category] = true
	}

	// suggestionTag is in the TUI package — we can't call it here.
	// Instead, we verify all categories are in the known set that
	// suggestionTag handles. If a new category is added, this test
	// will fail and remind the developer to update suggestionTag.
	knownCategories := map[string]bool{
		"Math":       true,
		"Conversion": true,
		"Network":    true,
		"Storage":    true,
		"Capacity":   true,
		"Growth":     true,
	}

	for cat := range categories {
		if !knownCategories[cat] {
			t.Errorf("BuiltinFunction category %q is not in the known set — "+
				"update suggestionTag() in view_overlays.go and this test", cat)
		}
	}
}

func TestEveryBuiltinFunctionHasRequiredMetadata(t *testing.T) {
	for _, fn := range BuiltinFunctions {
		if fn.Name == "" {
			t.Error("BuiltinFunction has empty Name")
		}
		if fn.Description == "" {
			t.Errorf("%s: missing Description", fn.Name)
		}
		if fn.Signature == "" {
			t.Errorf("%s: missing Signature", fn.Name)
		}
		if fn.Category == "" {
			t.Errorf("%s: missing Category", fn.Name)
		}
	}
}

func TestFunctionSpecOptionalFlagMatchesSignature(t *testing.T) {
	for _, fn := range BuiltinFunctions {
		spec := semantic.GetFunctionSpec(fn.Name)
		if spec == nil {
			continue
		}

		sigParams := parseSignatureParams(fn.Signature)

		for i, p := range spec.Params {
			if p.Variadic {
				continue // variadic params use "..." in signatures, not "?"
			}
			if i >= len(sigParams) {
				break // count mismatch caught by other test
			}

			sigOptional := strings.HasSuffix(sigParams[i], "?")
			if p.Optional != sigOptional {
				t.Errorf("%s param %d (%s): FunctionSpec.Optional=%v but Signature shows %q",
					fn.Name, i, p.Name, p.Optional, sigParams[i])
			}
		}
	}
}

// --- helpers ---

// specArgRange returns (minArgs, maxArgs) derived from FunctionSpec params.
// maxArgs is -1 for variadic functions (unbounded).
func specArgRange(spec *semantic.FunctionSpec) (int, int) {
	min := 0
	max := 0
	for _, p := range spec.Params {
		if p.Variadic {
			min++ // at least one required
			return min, -1
		}
		if !p.Optional {
			min++
		}
		max++
	}
	return min, max
}

// signatureArgRange parses a signature string like "capacity(demand, cap, unit, buffer?)"
// and returns (minArgs, maxArgs). maxArgs is -1 for variadic ("...").
func signatureArgRange(sig string) (int, int, error) {
	params := parseSignatureParams(sig)
	if len(params) == 0 {
		return 0, 0, fmt.Errorf("no params found")
	}

	// Check if variadic — "..." means unbounded max.
	// For variadic signatures like "avg(value1, value2, ...)", the named
	// params before "..." are illustrative; the real constraint is min=1.
	for _, p := range params {
		if strings.Contains(p, "...") {
			return 1, -1, nil
		}
	}

	min := 0
	max := 0
	for _, p := range params {
		if !strings.HasSuffix(p, "?") {
			min++
		}
		max++
	}
	return min, max, nil
}

// parseSignatureParams extracts parameter tokens from a signature string.
// "compound(principal, rate, duration, period?)" -> ["principal", "rate", "duration", "period?"]
// "avg(value1, value2, ...)" -> ["value1", "value2", "..."]
func parseSignatureParams(sig string) []string {
	start := strings.Index(sig, "(")
	end := strings.LastIndex(sig, ")")
	if start < 0 || end < 0 || end <= start+1 {
		return nil
	}
	inner := sig[start+1 : end]
	parts := strings.Split(inner, ",")
	params := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			params = append(params, p)
		}
	}
	return params
}
