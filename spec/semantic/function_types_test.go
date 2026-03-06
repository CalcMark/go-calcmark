package semantic

import "testing"

func TestCompoundFunctionSpec_HasPeriodParam(t *testing.T) {
	spec := GetFunctionSpec("compound")
	if spec == nil {
		t.Fatal("compound function spec not found")
	}

	// compound(principal, rate, periods, period?) — 4th param is optional
	if len(spec.Params) != 4 {
		t.Fatalf("compound should have 4 params (including optional period?), got %d", len(spec.Params))
	}

	period := spec.GetParamAtIndex(3)
	if period == nil {
		t.Fatal("compound param at index 3 should not be nil")
	}
	if period.Name != "period" {
		t.Errorf("compound param 3 name = %q, want %q", period.Name, "period")
	}
	if !period.Optional {
		t.Error("compound param 3 (period) should be optional")
	}
}
