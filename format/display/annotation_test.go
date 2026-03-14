package display

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/CalcMark/go-calcmark/spec/units"
	"github.com/shopspring/decimal"
)

func TestAnnotateUnit_StrictMode(t *testing.T) {
	f := DefaultFormatter()
	mc := units.DefaultMeasurement()
	f.SetMeasurement(mc, true) // strict = true

	tests := []struct {
		unit string
		want string
	}{
		{"oz", "us oz"},
		{"ounce", "us ounce"},
		{"gallon", "us gallon"},
		{"gal", "us gal"},
		{"pint", "us pint"},
		{"ton", "short ton"},
		// Already qualified — no change
		{"troy oz", "troy oz"},
		{"imp gal", "imp gal"},
		{"us oz", "us oz"},
		// Not ambiguous — no change
		{"meter", "meter"},
		{"kg", "kg"},
		{"celsius", "celsius"},
	}
	for _, tt := range tests {
		got := f.annotateUnit(tt.unit)
		if got != tt.want {
			t.Errorf("annotateUnit(%q) = %q, want %q", tt.unit, got, tt.want)
		}
	}
}

func TestAnnotateUnit_StrictModeNonDefault(t *testing.T) {
	f := DefaultFormatter()
	mc := &units.MeasurementConfig{Volume: "imperial", Mass: "troy", Ton: "long"}
	f.SetMeasurement(mc, true)

	tests := []struct {
		unit string
		want string
	}{
		{"oz", "troy oz"},
		{"gallon", "imp gallon"},
		{"ton", "long ton"},
	}
	for _, tt := range tests {
		got := f.annotateUnit(tt.unit)
		if got != tt.want {
			t.Errorf("annotateUnit(%q) = %q, want %q", tt.unit, got, tt.want)
		}
	}
}

func TestAnnotateUnit_NonStrictMode(t *testing.T) {
	f := DefaultFormatter()
	mc := units.DefaultMeasurement()
	f.SetMeasurement(mc, false) // strict = false

	// No annotation when strict is false
	tests := []struct {
		unit string
		want string
	}{
		{"oz", "oz"},
		{"gallon", "gallon"},
		{"ton", "ton"},
	}
	for _, tt := range tests {
		got := f.annotateUnit(tt.unit)
		if got != tt.want {
			t.Errorf("annotateUnit(%q, strict=false) = %q, want %q", tt.unit, got, tt.want)
		}
	}
}

func TestAnnotateUnit_NoMeasurementConfig(t *testing.T) {
	f := DefaultFormatter()
	// No SetMeasurement call — default behavior

	got := f.annotateUnit("oz")
	if got != "oz" {
		t.Errorf("annotateUnit(oz, no config) = %q, want %q", got, "oz")
	}
}

func TestFormatQuantity_StrictAnnotation(t *testing.T) {
	f := DefaultFormatter()
	mc := units.DefaultMeasurement()
	f.SetMeasurement(mc, true)

	q := &types.Quantity{
		Value: decimal.NewFromInt(2),
		Unit:  "oz",
	}
	got := f.FormatQuantity(q)
	// Should show "2 us oz" in strict mode
	if got != "2 us oz" {
		t.Errorf("FormatQuantity(2 oz, strict) = %q, want %q", got, "2 us oz")
	}
}

func TestFormatQuantity_NoAnnotationWithoutConfig(t *testing.T) {
	f := DefaultFormatter()

	q := &types.Quantity{
		Value: decimal.NewFromInt(2),
		Unit:  "oz",
	}
	got := f.FormatQuantity(q)
	// Should show "2 oz" without measurement config
	if got != "2 oz" {
		t.Errorf("FormatQuantity(2 oz, no config) = %q, want %q", got, "2 oz")
	}
}

func TestFormatQuantity_AlreadyQualifiedNoDoubleAnnotation(t *testing.T) {
	f := DefaultFormatter()
	mc := units.DefaultMeasurement()
	f.SetMeasurement(mc, true)

	q := &types.Quantity{
		Value: decimal.NewFromInt(5),
		Unit:  "troy oz",
	}
	got := f.FormatQuantity(q)
	// Should NOT double-annotate: "5 troy oz", not "5 us troy oz"
	if got != "5 troy oz" {
		t.Errorf("FormatQuantity(5 troy oz, strict) = %q, want %q", got, "5 troy oz")
	}
}
