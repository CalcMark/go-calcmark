package types

import (
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

func nums(vals ...int64) []Type {
	out := make([]Type, len(vals))
	for i, v := range vals {
		out[i] = NewNumber(decimal.NewFromInt(v))
	}
	return out
}

func TestArray_HomogeneousNumbers(t *testing.T) {
	arr, err := NewArray(nums(1, 2, 3))
	if err != nil {
		t.Fatal(err)
	}
	if arr.Len() != 3 || arr.ElementType != "Number" {
		t.Errorf("Len/ElementType = %d/%q, want 3/Number", arr.Len(), arr.ElementType)
	}
	if arr.String() != "[1, 2, 3]" {
		t.Errorf("String() = %q, want [1, 2, 3]", arr.String())
	}
}

func TestArray_CurrencyElementsKeepSymbols(t *testing.T) {
	arr, err := NewArray([]Type{
		NewCurrency(decimal.NewFromInt(250), "$"),
		NewCurrency(decimal.NewFromInt(150), "$"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(arr.String(), "[$") {
		t.Errorf("String() = %q, want currency symbols preserved", arr.String())
	}
}

func TestArray_EmptyIsValid(t *testing.T) {
	arr, err := NewArray(nil)
	if err != nil {
		t.Fatal(err)
	}
	if arr.Len() != 0 || arr.String() != "[]" || arr.ElementType != "" {
		t.Errorf("empty array = %+v / %q", arr, arr.String())
	}
}

func TestArray_MixedTypesRejected(t *testing.T) {
	_, err := NewArray([]Type{NewNumber(decimal.NewFromInt(1)), NewCurrency(decimal.NewFromInt(2), "$")})
	if err == nil || !strings.Contains(err.Error(), "mixed types") {
		t.Errorf("want mixed-types error, got %v", err)
	}
}

func TestText_IsNotNumeric(t *testing.T) {
	if _, err := ToDecimal(NewText("Senior")); err == nil {
		t.Error("ToDecimal(Text) must fail")
	}
	if NewText("Senior").String() != "Senior" {
		t.Error("Text.String() must return the text verbatim")
	}
}
