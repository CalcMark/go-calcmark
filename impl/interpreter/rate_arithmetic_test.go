package interpreter

import (
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

// Foundation helper tests for U1. The cancellation engine in U2 leans
// on `categoryOf` to decide whether two units can compose, and on
// `convertWithinCategory` to do the actual math when they can. Both
// must behave correctly across the boundary between time units (which
// live in spec/types/duration.go) and everything else (which lives in
// spec/units/conversion.go), plus the Custom-unit fallback case where
// only exact-string matches succeed.

func TestCategoryOf_TimeUnits(t *testing.T) {
	cases := []string{"second", "minute", "hour", "day", "week", "month", "quarter", "year"}
	for _, u := range cases {
		t.Run(u, func(t *testing.T) {
			if got := categoryOf(u); got != "time" {
				t.Errorf("categoryOf(%q) = %q, want \"time\"", u, got)
			}
		})
	}
}

func TestCategoryOf_KnownPhysicalUnits(t *testing.T) {
	// Pick one unit from each major units-package category and confirm
	// categoryOf returns the canonical category name. The exact
	// category strings come from spec/units/categories.go and the
	// individual unit registrations.
	cases := map[string]string{
		"kg":     "Mass",
		"g":      "Mass",
		"meter":  "Length",
		"meters": "Length",
		"GB":     "DataSize",
		"liter":  "Volume",
		"mph":    "Speed",
	}
	for unit, wantCat := range cases {
		t.Run(unit, func(t *testing.T) {
			if got := categoryOf(unit); got != wantCat {
				t.Errorf("categoryOf(%q) = %q, want %q", unit, got, wantCat)
			}
		})
	}
}

func TestCategoryOf_CustomUnits(t *testing.T) {
	// Unknown unit names fall through to units.CategoryForUnit which
	// returns "Custom" for anything outside its registry. Cancellation
	// between Custom units only works when the unit strings are equal
	// (handled by convertWithinCategory's fast path); this test just
	// pins the category-name contract.
	cases := []string{"cake", "cakes", "box", "boxes", "user", "request"}
	for _, u := range cases {
		t.Run(u, func(t *testing.T) {
			if got := categoryOf(u); got != "Custom" {
				t.Errorf("categoryOf(%q) = %q, want \"Custom\"", u, got)
			}
		})
	}
}

func TestCategoryOf_EmptyIsEmpty(t *testing.T) {
	// Empty unit (a unitless rate's numerator) returns empty category.
	// Used by tryRateCancellation to refuse cancellation when one side
	// has no unit at all.
	if got := categoryOf(""); got != "" {
		t.Errorf("categoryOf(\"\") = %q, want \"\"", got)
	}
}

func TestCategoryOf_TimeBeforeCustom(t *testing.T) {
	// Time units must NOT fall into the Custom bucket even though
	// they're not registered in spec/units/conversionRegistry. The
	// time-first short-circuit in categoryOf is load-bearing: without
	// it, `box × hour` would match by category (both Custom) and
	// trigger spurious cancellation attempts.
	if categoryOf("hour") == categoryOf("box") {
		t.Errorf("hour and box should NOT share a category; categoryOf(hour)=%q categoryOf(box)=%q",
			categoryOf("hour"), categoryOf("box"))
	}
}

func TestConvertWithinCategory_IdenticalUnitsPassThrough(t *testing.T) {
	// Same-unit conversion always succeeds and returns the value
	// unchanged, regardless of category. This is the path Custom
	// units take in the cakes/box × boxes case.
	cases := []struct {
		value string
		unit  string
	}{
		{"5", "hour"},
		{"3", "kg"},
		{"100", "box"},
		{"42.5", "miles"},
		{"0", ""},
	}
	for _, tc := range cases {
		t.Run(tc.unit, func(t *testing.T) {
			value := decimal.RequireFromString(tc.value)
			got, err := convertWithinCategory(value, tc.unit, tc.unit)
			if err != nil {
				t.Fatalf("convertWithinCategory(%s, %q, %q) errored: %v", tc.value, tc.unit, tc.unit, err)
			}
			if !got.Equal(value) {
				t.Errorf("got %s, want %s (identity must pass through)", got.String(), tc.value)
			}
		})
	}
}

func TestConvertWithinCategory_TimeConversion(t *testing.T) {
	// Time units convert through types.Duration.Convert. The 3-weeks
	// → 504-hours case is the canonical example from AE1 in the
	// brainstorm and locks the expected value.
	got, err := convertWithinCategory(decimal.NewFromInt(3), "week", "hour")
	if err != nil {
		t.Fatalf("convertWithinCategory: %v", err)
	}
	if got.String() != "504" {
		t.Errorf("3 week → hour = %s, want 504", got.String())
	}
}

func TestConvertWithinCategory_PhysicalConversion(t *testing.T) {
	// Mass conversion goes through units.Convert. Verifies the
	// non-time branch dispatches correctly.
	got, err := convertWithinCategory(decimal.NewFromInt(5), "kg", "g")
	if err != nil {
		t.Fatalf("convertWithinCategory: %v", err)
	}
	if got.String() != "5000" {
		t.Errorf("5 kg → g = %s, want 5000", got.String())
	}
}

func TestConvertWithinCategory_CrossCategoryRejected(t *testing.T) {
	// kg (mass) → hour (time): different categories, must error.
	// The error message must name both units AND the category split
	// so the R6 refusal in U4 can wrap it without losing information.
	_, err := convertWithinCategory(decimal.NewFromInt(5), "kg", "hour")
	if err == nil {
		t.Fatal("expected error for kg → hour, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "kg") {
		t.Errorf("error %q must mention 'kg'", msg)
	}
	if !strings.Contains(msg, "hour") {
		t.Errorf("error %q must mention 'hour'", msg)
	}
	if !strings.Contains(msg, "categor") { // matches "category" or "categories"
		t.Errorf("error %q should mention category mismatch", msg)
	}
}

func TestConvertWithinCategory_CustomDistinctRejected(t *testing.T) {
	// Two distinct Custom units (e.g., cake → box) refuse with a
	// clear "no converter" error. The cancellation engine in U2
	// turns this into an R6 refusal at the operator level rather
	// than leaking the message verbatim, but at the helper layer
	// the error must still be specific.
	_, err := convertWithinCategory(decimal.NewFromInt(3), "cake", "box")
	if err == nil {
		t.Fatal("expected error for cake → box, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "cake") {
		t.Errorf("error %q must mention 'cake'", msg)
	}
	if !strings.Contains(msg, "box") {
		t.Errorf("error %q must mention 'box'", msg)
	}
}

func TestConvertWithinCategory_UnknownUnitRejected(t *testing.T) {
	// Empty-string unit on either side: refuse with a message
	// pointing at the bad input.
	_, err := convertWithinCategory(decimal.NewFromInt(1), "", "hour")
	if err == nil {
		t.Fatal("expected error for empty-from, got nil")
	}
}

func TestIsTimeUnit_StillWorks(t *testing.T) {
	// Backward-compat shim — existing callers in tryRateRateCancellation
	// continue to call isTimeUnit. The behavior is now categoryOf-driven;
	// confirm the equivalence holds.
	if !isTimeUnit("hour") {
		t.Error("isTimeUnit(hour) should be true")
	}
	if isTimeUnit("kg") {
		t.Error("isTimeUnit(kg) should be false")
	}
	if isTimeUnit("") {
		t.Error("isTimeUnit(\"\") should be false")
	}
}
