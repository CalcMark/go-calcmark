package lsp

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

func TestRuntimeTypeToArgType(t *testing.T) {
	dur, err := types.NewDuration(decimal.NewFromInt(1), "hour")
	if err != nil {
		t.Fatalf("NewDuration: %v", err)
	}

	cases := []struct {
		name  string
		value types.Type
		want  types.ArgType
	}{
		{"number", types.NewNumber(decimal.NewFromInt(42)), types.ArgTypeNumber},
		{"percentage", types.NewPercentage(decimal.NewFromFloat(0.08)), types.ArgTypePercentage},
		{"quantity", types.NewQuantity(decimal.NewFromInt(10), "meter"), types.ArgTypeQuantity},
		{"rate", types.NewRate(types.NewQuantity(decimal.NewFromInt(10), "MB"), "second"), types.ArgTypeRate},
		{"duration", dur, types.ArgTypeDuration},
		{"currency", types.NewCurrency(decimal.NewFromInt(100), "$"), types.ArgTypeQuantity},
		{"nil", nil, types.ArgTypeAny},
		{"boolean", types.NewBoolean(true), types.ArgTypeAny},
		{"fraction", mustFraction(t, 1, 2), types.ArgTypeAny},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := runtimeTypeToArgType(tc.value)
			if got != tc.want {
				t.Errorf("runtimeTypeToArgType(%v) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}

// mustFraction builds a Fraction or fails the test — keeps test rows readable.
func mustFraction(t *testing.T, num, denom int64) types.Type {
	t.Helper()
	f, err := types.NewFraction(num, denom)
	if err != nil {
		t.Fatalf("NewFraction: %v", err)
	}
	return f
}

func TestArgTypesCompatible(t *testing.T) {
	cases := []struct {
		name             string
		actual, required types.ArgType
		want             bool
	}{
		{"exact rate", types.ArgTypeRate, types.ArgTypeRate, true},
		{"exact number", types.ArgTypeNumber, types.ArgTypeNumber, true},
		{"number vs rate", types.ArgTypeNumber, types.ArgTypeRate, false},
		{"number vs any required", types.ArgTypeNumber, types.ArgTypeAny, true},
		{"rate vs any required", types.ArgTypeRate, types.ArgTypeAny, true},
		{"empty required accepts anything", types.ArgTypeNumber, types.ArgType(""), true},
		{"duration vs number", types.ArgTypeDuration, types.ArgTypeNumber, false},
		// Unknown/any actual (e.g. Boolean mapped to ArgTypeAny) must NOT
		// leak into type-specific filters. Only an ArgTypeAny *required*
		// accepts everything; an ArgTypeAny *actual* is otherwise excluded.
		{"any actual does not match rate required", types.ArgTypeAny, types.ArgTypeRate, false},
		{"any actual does not match number required", types.ArgTypeAny, types.ArgTypeNumber, false},
		{"any actual matches any required", types.ArgTypeAny, types.ArgTypeAny, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := argTypesCompatible(tc.actual, tc.required)
			if got != tc.want {
				t.Errorf("argTypesCompatible(%q, %q) = %v, want %v", tc.actual, tc.required, got, tc.want)
			}
		})
	}
}
