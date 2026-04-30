package types

import (
	"reflect"
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/identifiers"
)

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

// TestEnumValuesOnStringParams asserts that every string-typed parameter backed
// by a finite identifier list populates ParamSpec.EnumValues from spec/identifiers.
// This is the single source of truth for enum completions.
func TestEnumValuesOnStringParams(t *testing.T) {
	cases := []struct {
		fn        string
		paramIdx  int
		paramName string
		want      []string
	}{
		{"rtt", 0, "scope", identifiers.NetworkScopes},
		{"throughput", 0, "network_type", identifiers.NetworkTypes},
		{"transfer_time", 1, "scope", identifiers.NetworkScopes},
		{"transfer_time", 2, "network_type", identifiers.NetworkTypes},
		{"read", 1, "storage_type", identifiers.StorageTypes},
		{"seek", 0, "storage_type", identifiers.StorageTypes},
		{"compress", 1, "compression_type", identifiers.CompressionTypes},
		{"convert_rate", 1, "time_unit", identifiers.TimeUnits},
	}

	for _, tc := range cases {
		t.Run(tc.fn+"_"+tc.paramName, func(t *testing.T) {
			spec := GetFunctionSpec(tc.fn)
			if spec == nil {
				t.Fatalf("%s function spec not found", tc.fn)
			}
			p := spec.GetParamAtIndex(tc.paramIdx)
			if p == nil {
				t.Fatalf("%s param %d (%s) not found", tc.fn, tc.paramIdx, tc.paramName)
			}
			if p.Name != tc.paramName {
				t.Errorf("%s param %d name = %q, want %q", tc.fn, tc.paramIdx, p.Name, tc.paramName)
			}
			if !reflect.DeepEqual(p.EnumValues, tc.want) {
				t.Errorf("%s.%s EnumValues = %v, want %v", tc.fn, tc.paramName, p.EnumValues, tc.want)
			}
		})
	}
}

// TestEnumValuesNilForNonEnumStringParams ensures free-form string params don't
// accidentally pick up enum values.
func TestEnumValuesNilForNonEnumStringParams(t *testing.T) {
	cases := []struct {
		fn       string
		paramIdx int
	}{
		{"compound", 3}, // period — free-form
		{"capacity", 2}, // unit — free-form
	}

	for _, tc := range cases {
		t.Run(tc.fn, func(t *testing.T) {
			spec := GetFunctionSpec(tc.fn)
			if spec == nil {
				t.Fatalf("%s function spec not found", tc.fn)
			}
			p := spec.GetParamAtIndex(tc.paramIdx)
			if p == nil {
				t.Fatalf("%s param %d not found", tc.fn, tc.paramIdx)
			}
			if p.EnumValues != nil {
				t.Errorf("%s param %d (%s) should have nil EnumValues, got %v", tc.fn, tc.paramIdx, p.Name, p.EnumValues)
			}
		})
	}
}

// TestEnumValuesNilForNonStringParams ensures non-string params never carry enum values.
func TestEnumValuesNilForNonStringParams(t *testing.T) {
	cases := []struct {
		fn       string
		paramIdx int
	}{
		{"accumulate", 0}, // rate
		{"accumulate", 1}, // duration
		{"grow", 2},       // periods (number)
		{"sqrt", 0},       // number
	}

	for _, tc := range cases {
		t.Run(tc.fn, func(t *testing.T) {
			spec := GetFunctionSpec(tc.fn)
			if spec == nil {
				t.Fatalf("%s function spec not found", tc.fn)
			}
			p := spec.GetParamAtIndex(tc.paramIdx)
			if p == nil {
				t.Fatalf("%s param %d not found", tc.fn, tc.paramIdx)
			}
			if p.EnumValues != nil {
				t.Errorf("%s param %d (%s) should have nil EnumValues, got %v", tc.fn, tc.paramIdx, p.Name, p.EnumValues)
			}
		})
	}
}
