package transform

import (
	"math"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

func TestApply_ScaleQuantity(t *testing.T) {
	q := &types.Quantity{Value: decimal.NewFromInt(280), Unit: "grams"}
	scale := &document.ScaleConfig{Factor: decimal.NewFromInt(4)}

	result := Apply(q, scale, nil)
	qty, ok := result.(*types.Quantity)
	if !ok {
		t.Fatalf("expected *types.Quantity, got %T", result)
	}
	if !qty.Value.Equal(decimal.NewFromInt(1120)) {
		t.Errorf("expected 1120, got %s", qty.Value.String())
	}
	if qty.Unit != "grams" {
		t.Errorf("expected unit 'grams', got %q", qty.Unit)
	}
}

func TestApply_ConvertToImperial(t *testing.T) {
	q := &types.Quantity{Value: decimal.NewFromInt(280), Unit: "grams"}
	convertTo := &document.ConvertToConfig{System: "imperial"}

	result := Apply(q, nil, convertTo)
	qty, ok := result.(*types.Quantity)
	if !ok {
		t.Fatalf("expected *types.Quantity, got %T", result)
	}
	f, _ := qty.Value.Float64()
	if math.Abs(f-9.88) > 0.1 {
		t.Errorf("expected ~9.88, got %f", f)
	}
	if qty.Unit != "ounce" {
		t.Errorf("expected unit 'ounce', got %q", qty.Unit)
	}
}

func TestApply_ScaleThenConvert(t *testing.T) {
	q := &types.Quantity{Value: decimal.NewFromInt(280), Unit: "grams"}
	scale := &document.ScaleConfig{Factor: decimal.NewFromInt(4)}
	convertTo := &document.ConvertToConfig{System: "imperial"}

	result := Apply(q, scale, convertTo)
	qty := result.(*types.Quantity)
	f, _ := qty.Value.Float64()
	// 280 * 4 = 1120 grams -> ~39.5 ounces
	if math.Abs(f-39.5) > 0.5 {
		t.Errorf("expected ~39.5, got %f", f)
	}
}

func TestApply_TemperatureExcludedFromScale(t *testing.T) {
	q := &types.Quantity{Value: decimal.NewFromInt(220), Unit: "celsius"}
	scale := &document.ScaleConfig{Factor: decimal.NewFromInt(4)}

	result := Apply(q, scale, nil)
	qty := result.(*types.Quantity)
	if !qty.Value.Equal(decimal.NewFromInt(220)) {
		t.Errorf("expected 220 (temperature immune to scale), got %s", qty.Value.String())
	}
}

func TestApply_TemperatureConvertTo(t *testing.T) {
	q := &types.Quantity{Value: decimal.NewFromInt(220), Unit: "celsius"}
	convertTo := &document.ConvertToConfig{System: "imperial"}

	result := Apply(q, nil, convertTo)
	qty := result.(*types.Quantity)
	f, _ := qty.Value.Float64()
	if math.Abs(f-428) > 0.5 {
		t.Errorf("expected ~428, got %f", f)
	}
	if qty.Unit != "fahrenheit" {
		t.Errorf("expected unit 'fahrenheit', got %q", qty.Unit)
	}
}

func TestApply_CurrencyImmune(t *testing.T) {
	c := &types.Currency{Value: decimal.NewFromInt(100), Code: "USD"}
	scale := &document.ScaleConfig{Factor: decimal.NewFromInt(4)}
	convertTo := &document.ConvertToConfig{System: "imperial"}

	result := Apply(c, scale, convertTo)
	cur, ok := result.(*types.Currency)
	if !ok {
		t.Fatalf("expected *types.Currency, got %T", result)
	}
	if !cur.Value.Equal(decimal.NewFromInt(100)) {
		t.Errorf("expected 100, got %s", cur.Value.String())
	}
}

func TestApply_DurationImmune(t *testing.T) {
	d := &types.Duration{Value: decimal.NewFromInt(3), Unit: "days"}
	scale := &document.ScaleConfig{Factor: decimal.NewFromInt(4)}

	result := Apply(d, scale, nil)
	dur, ok := result.(*types.Duration)
	if !ok {
		t.Fatalf("expected *types.Duration, got %T", result)
	}
	if !dur.Value.Equal(decimal.NewFromInt(3)) {
		t.Errorf("expected 3, got %s", dur.Value.String())
	}
}

func TestApply_ArbitraryUnitScales(t *testing.T) {
	q := &types.Quantity{Value: decimal.NewFromInt(5), Unit: "eggs"}
	scale := &document.ScaleConfig{Factor: decimal.NewFromInt(4)}

	result := Apply(q, scale, nil)
	qty := result.(*types.Quantity)
	if !qty.Value.Equal(decimal.NewFromInt(20)) {
		t.Errorf("expected 20 eggs, got %s", qty.Value.String())
	}
}

func TestApply_ArbitraryUnitNoConvert(t *testing.T) {
	q := &types.Quantity{Value: decimal.NewFromInt(5), Unit: "eggs"}
	convertTo := &document.ConvertToConfig{System: "imperial"}

	result := Apply(q, nil, convertTo)
	qty := result.(*types.Quantity)
	if qty.Unit != "eggs" {
		t.Errorf("expected unit 'eggs' unchanged, got %q", qty.Unit)
	}
}

func TestApply_RateImmuneToScale(t *testing.T) {
	r := &types.Rate{
		Amount:  &types.Quantity{Value: decimal.NewFromInt(100), Unit: "km/h"},
		PerUnit: "hour",
	}
	scale := &document.ScaleConfig{Factor: decimal.NewFromInt(4)}

	result := Apply(r, scale, nil)
	rate, ok := result.(*types.Rate)
	if !ok {
		t.Fatalf("expected *types.Rate, got %T", result)
	}
	if !rate.Amount.Value.Equal(decimal.NewFromInt(100)) {
		t.Errorf("expected rate amount 100 (immune to scale), got %s", rate.Amount.Value.String())
	}
}

func TestApply_RateConvertTo(t *testing.T) {
	r := &types.Rate{
		Amount:  &types.Quantity{Value: decimal.NewFromInt(100), Unit: "km/h"},
		PerUnit: "hour",
	}
	convertTo := &document.ConvertToConfig{System: "imperial"}

	result := Apply(r, nil, convertTo)
	rate := result.(*types.Rate)
	f, _ := rate.Amount.Value.Float64()
	// 100 km/h -> ~62.14 mph
	if math.Abs(f-62.14) > 0.5 {
		t.Errorf("expected ~62.14, got %f", f)
	}
	if rate.Amount.Unit != "mph" {
		t.Errorf("expected unit 'mph', got %q", rate.Amount.Unit)
	}
	if rate.PerUnit != "hour" {
		t.Errorf("expected perUnit 'hour' unchanged, got %q", rate.PerUnit)
	}
}

func TestApply_ExplicitSkipsConvert(t *testing.T) {
	q := &types.Quantity{Value: decimal.NewFromInt(280), Unit: "grams", IsExplicit: true}
	convertTo := &document.ConvertToConfig{System: "imperial"}

	result := Apply(q, nil, convertTo)
	qty := result.(*types.Quantity)
	if qty.Unit != "grams" {
		t.Errorf("expected explicit quantity unchanged, got unit %q", qty.Unit)
	}
}

func TestApply_ScaleStillAppliesWithExplicit(t *testing.T) {
	q := &types.Quantity{Value: decimal.NewFromInt(280), Unit: "grams", IsExplicit: true}
	scale := &document.ScaleConfig{Factor: decimal.NewFromInt(4)}

	result := Apply(q, scale, nil)
	qty := result.(*types.Quantity)
	if !qty.Value.Equal(decimal.NewFromInt(1120)) {
		t.Errorf("expected 1120 (scale applies even with explicit), got %s", qty.Value.String())
	}
}

func TestApply_ScaleWithCategories(t *testing.T) {
	scale := &document.ScaleConfig{
		Factor:         decimal.NewFromInt(4),
		UnitCategories: []string{"Mass"},
	}

	// Mass quantity should be scaled
	mass := &types.Quantity{Value: decimal.NewFromInt(280), Unit: "grams"}
	result := Apply(mass, scale, nil)
	if !result.(*types.Quantity).Value.Equal(decimal.NewFromInt(1120)) {
		t.Errorf("mass should be scaled, got %s", result.(*types.Quantity).Value.String())
	}

	// Volume quantity should NOT be scaled
	vol := &types.Quantity{Value: decimal.NewFromInt(500), Unit: "ml"}
	result = Apply(vol, scale, nil)
	if !result.(*types.Quantity).Value.Equal(decimal.NewFromInt(500)) {
		t.Errorf("volume should not be scaled with Mass-only filter, got %s", result.(*types.Quantity).Value.String())
	}
}

func TestApply_ConvertToWithCategories(t *testing.T) {
	convertTo := &document.ConvertToConfig{
		System:         "imperial",
		UnitCategories: []string{"Temperature"},
	}

	// Temperature should convert
	temp := &types.Quantity{Value: decimal.NewFromInt(220), Unit: "celsius"}
	result := Apply(temp, nil, convertTo)
	if result.(*types.Quantity).Unit != "fahrenheit" {
		t.Errorf("temperature should be converted, got unit %q", result.(*types.Quantity).Unit)
	}

	// Mass should NOT convert
	mass := &types.Quantity{Value: decimal.NewFromInt(280), Unit: "grams"}
	result = Apply(mass, nil, convertTo)
	if result.(*types.Quantity).Unit != "grams" {
		t.Errorf("mass should not be converted with Temperature-only filter, got unit %q", result.(*types.Quantity).Unit)
	}
}

func TestApply_DoesNotMutateOriginal(t *testing.T) {
	original := &types.Quantity{Value: decimal.NewFromInt(280), Unit: "grams"}
	scale := &document.ScaleConfig{Factor: decimal.NewFromInt(4)}

	Apply(original, scale, nil)

	if !original.Value.Equal(decimal.NewFromInt(280)) {
		t.Errorf("original should not be mutated, got %s", original.Value.String())
	}
}

func TestApply_Nil(t *testing.T) {
	result := Apply(nil, &document.ScaleConfig{Factor: decimal.NewFromInt(4)}, nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestApply_AlreadyInTargetSystem(t *testing.T) {
	// Ounces are already imperial — should not be converted
	q := &types.Quantity{Value: decimal.NewFromInt(10), Unit: "ounces"}
	convertTo := &document.ConvertToConfig{System: "imperial"}

	result := Apply(q, nil, convertTo)
	qty := result.(*types.Quantity)
	if qty.Unit != "ounces" {
		t.Errorf("expected ounces unchanged (already imperial), got %q", qty.Unit)
	}
	if !qty.Value.Equal(decimal.NewFromInt(10)) {
		t.Errorf("expected value 10 unchanged, got %s", qty.Value.String())
	}
}
