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
	scale := &document.ScaleConfig{Factor: decimal.NewFromInt(4), UnitCategories: []string{"Mass"}}

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
	scale := &document.ScaleConfig{Factor: decimal.NewFromInt(4), UnitCategories: []string{"Mass"}}
	convertTo := &document.ConvertToConfig{System: "imperial"}

	result := Apply(q, scale, convertTo)
	qty := result.(*types.Quantity)
	f, _ := qty.Value.Float64()
	// 280 * 4 = 1120 grams -> ~39.5 ounces
	if math.Abs(f-39.5) > 0.5 {
		t.Errorf("expected ~39.5, got %f", f)
	}
}

func TestApply_TemperatureNotScaledWithoutCategory(t *testing.T) {
	q := &types.Quantity{Value: decimal.NewFromInt(220), Unit: "celsius"}
	scale := &document.ScaleConfig{Factor: decimal.NewFromInt(4), UnitCategories: []string{"Mass"}}

	result := Apply(q, scale, nil)
	qty := result.(*types.Quantity)
	if !qty.Value.Equal(decimal.NewFromInt(220)) {
		t.Errorf("expected 220 (temperature not in categories), got %s", qty.Value.String())
	}
}

func TestApply_NoUnitCategoriesMeansNoScaling(t *testing.T) {
	q := &types.Quantity{Value: decimal.NewFromInt(280), Unit: "grams"}
	scale := &document.ScaleConfig{Factor: decimal.NewFromInt(4)}

	result := Apply(q, scale, nil)
	qty := result.(*types.Quantity)
	if !qty.Value.Equal(decimal.NewFromInt(280)) {
		t.Errorf("expected 280 (no unit_categories = no scaling), got %s", qty.Value.String())
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

func TestApply_CurrencyScaledWithCategory(t *testing.T) {
	c := types.NewCurrency(decimal.NewFromFloat(0.50), "$")
	scale := &document.ScaleConfig{
		Factor:         decimal.NewFromInt(3),
		UnitCategories: []string{"Currency"},
	}

	result := Apply(c, scale, nil)
	cur, ok := result.(*types.Currency)
	if !ok {
		t.Fatalf("expected *types.Currency, got %T", result)
	}
	if !cur.Value.Equal(decimal.NewFromFloat(1.50)) {
		t.Errorf("expected 1.50, got %s", cur.Value.String())
	}
	if cur.Symbol != "$" {
		t.Errorf("expected symbol '$', got %q", cur.Symbol)
	}
}

func TestApply_CurrencyNotScaledWithoutCategory(t *testing.T) {
	// When unit_categories is set but does NOT include Currency,
	// currency values must remain immune.
	c := types.NewCurrency(decimal.NewFromFloat(0.50), "$")
	scale := &document.ScaleConfig{
		Factor:         decimal.NewFromInt(3),
		UnitCategories: []string{"Mass", "Volume"},
	}

	result := Apply(c, scale, nil)
	cur, ok := result.(*types.Currency)
	if !ok {
		t.Fatalf("expected *types.Currency, got %T", result)
	}
	if !cur.Value.Equal(decimal.NewFromFloat(0.50)) {
		t.Errorf("expected 0.50 (immune), got %s", cur.Value.String())
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

func TestApply_CustomUnitNotScaledByDefault(t *testing.T) {
	// Custom units (like "eggs") have no category — they won't match
	// any unit_categories entry, so they don't scale.
	q := &types.Quantity{Value: decimal.NewFromInt(5), Unit: "eggs"}
	scale := &document.ScaleConfig{Factor: decimal.NewFromInt(4), UnitCategories: []string{"Mass"}}

	result := Apply(q, scale, nil)
	qty := result.(*types.Quantity)
	if !qty.Value.Equal(decimal.NewFromInt(5)) {
		t.Errorf("expected 5 eggs (arbitrary unit not in categories), got %s", qty.Value.String())
	}
}

func TestApply_CustomUnitNoConvert(t *testing.T) {
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
	scale := &document.ScaleConfig{Factor: decimal.NewFromInt(4), UnitCategories: []string{"Mass"}}

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

func TestApply_NumberImmune(t *testing.T) {
	// Numbers are immune to scale by default (no unit_categories listing)
	n := types.NewNumber(decimal.NewFromFloat(0.085))
	scale := &document.ScaleConfig{Factor: decimal.NewFromInt(3)}

	result := Apply(n, scale, nil)
	num, ok := result.(*types.Number)
	if !ok {
		t.Fatalf("expected *types.Number, got %T", result)
	}
	if !num.Value.Equal(decimal.NewFromFloat(0.085)) {
		t.Errorf("expected 0.085 (immune), got %s", num.Value.String())
	}
}

func TestApply_NumberScaledWithCategory(t *testing.T) {
	n := types.NewNumber(decimal.NewFromFloat(0.085))
	scale := &document.ScaleConfig{
		Factor:         decimal.NewFromInt(3),
		UnitCategories: []string{"Number"},
	}

	result := Apply(n, scale, nil)
	num, ok := result.(*types.Number)
	if !ok {
		t.Fatalf("expected *types.Number, got %T", result)
	}
	if !num.Value.Equal(decimal.NewFromFloat(0.255)) {
		t.Errorf("expected 0.255, got %s", num.Value.String())
	}
}

func TestApply_NumberNotScaledWithoutCategory(t *testing.T) {
	// When unit_categories is set but does NOT include Number,
	// number values must remain immune.
	n := types.NewNumber(decimal.NewFromFloat(0.085))
	scale := &document.ScaleConfig{
		Factor:         decimal.NewFromInt(3),
		UnitCategories: []string{"Mass", "Volume"},
	}

	result := Apply(n, scale, nil)
	num, ok := result.(*types.Number)
	if !ok {
		t.Fatalf("expected *types.Number, got %T", result)
	}
	if !num.Value.Equal(decimal.NewFromFloat(0.085)) {
		t.Errorf("expected 0.085 (immune), got %s", num.Value.String())
	}
}

func TestWouldScale_QuantityWithCategory(t *testing.T) {
	q := &types.Quantity{Value: decimal.NewFromInt(280), Unit: "grams"}
	scale := &document.ScaleConfig{Factor: decimal.NewFromInt(4), UnitCategories: []string{"Mass"}}
	if !WouldScale(q, scale) {
		t.Error("expected grams to be scaled when Mass in unit_categories")
	}
}

func TestWouldScale_QuantityWithoutCategory(t *testing.T) {
	q := &types.Quantity{Value: decimal.NewFromInt(280), Unit: "grams"}
	scale := &document.ScaleConfig{Factor: decimal.NewFromInt(4)}
	if WouldScale(q, scale) {
		t.Error("expected no scaling without unit_categories")
	}
}

func TestWouldScale_TemperatureNotInCategories(t *testing.T) {
	q := &types.Quantity{Value: decimal.NewFromInt(220), Unit: "celsius"}
	scale := &document.ScaleConfig{Factor: decimal.NewFromInt(4), UnitCategories: []string{"Mass"}}
	if WouldScale(q, scale) {
		t.Error("expected temperature not to scale when not in unit_categories")
	}
}

func TestWouldScale_NumberDefault(t *testing.T) {
	n := types.NewNumber(decimal.NewFromInt(10))
	scale := &document.ScaleConfig{Factor: decimal.NewFromInt(2)}
	if WouldScale(n, scale) {
		t.Error("expected Number to be immune by default")
	}
}

func TestWouldScale_NumberWithCategory(t *testing.T) {
	n := types.NewNumber(decimal.NewFromInt(10))
	scale := &document.ScaleConfig{
		Factor:         decimal.NewFromInt(2),
		UnitCategories: []string{"Number"},
	}
	if !WouldScale(n, scale) {
		t.Error("expected Number to be scaled when in unit_categories")
	}
}

func TestWouldScale_CurrencyWithCategory(t *testing.T) {
	c := types.NewCurrency(decimal.NewFromFloat(5.00), "$")
	scale := &document.ScaleConfig{
		Factor:         decimal.NewFromInt(3),
		UnitCategories: []string{"Currency"},
	}
	if !WouldScale(c, scale) {
		t.Error("expected Currency to be scaled when in unit_categories")
	}
}

func TestWouldScale_NilScale(t *testing.T) {
	q := &types.Quantity{Value: decimal.NewFromInt(280), Unit: "grams"}
	if WouldScale(q, nil) {
		t.Error("expected false with nil scale")
	}
}

func TestWouldScale_Duration(t *testing.T) {
	d := &types.Duration{Value: decimal.NewFromInt(3), Unit: "days"}
	scale := &document.ScaleConfig{Factor: decimal.NewFromInt(4)}
	if WouldScale(d, scale) {
		t.Error("expected Duration to be immune")
	}
}

func TestApply_AllCategoryScalesEverything(t *testing.T) {
	scale := &document.ScaleConfig{Factor: decimal.NewFromInt(3), UnitCategories: []string{"All"}}

	// Quantity (Mass)
	q := &types.Quantity{Value: decimal.NewFromInt(100), Unit: "grams"}
	result := Apply(q, scale, nil)
	if qty := result.(*types.Quantity); !qty.Value.Equal(decimal.NewFromInt(300)) {
		t.Errorf("expected grams scaled to 300, got %s", qty.Value.String())
	}

	// Currency
	c := types.NewCurrency(decimal.NewFromInt(50), "$")
	result = Apply(c, scale, nil)
	if cur := result.(*types.Currency); !cur.Value.Equal(decimal.NewFromInt(150)) {
		t.Errorf("expected currency scaled to 150, got %s", cur.Value.String())
	}

	// Number
	n := types.NewNumber(decimal.NewFromInt(10))
	result = Apply(n, scale, nil)
	if num := result.(*types.Number); !num.Value.Equal(decimal.NewFromInt(30)) {
		t.Errorf("expected number scaled to 30, got %s", num.Value.String())
	}

	// Temperature — also matched by All
	temp := &types.Quantity{Value: decimal.NewFromInt(200), Unit: "celsius"}
	result = Apply(temp, scale, nil)
	if qty := result.(*types.Quantity); !qty.Value.Equal(decimal.NewFromInt(600)) {
		t.Errorf("expected celsius scaled to 600, got %s", qty.Value.String())
	}
}

func TestWouldScale_AllCategory(t *testing.T) {
	scale := &document.ScaleConfig{Factor: decimal.NewFromInt(2), UnitCategories: []string{"All"}}

	q := &types.Quantity{Value: decimal.NewFromInt(100), Unit: "grams"}
	if !WouldScale(q, scale) {
		t.Error("expected WouldScale=true for quantity with All category")
	}

	c := types.NewCurrency(decimal.NewFromInt(50), "$")
	if !WouldScale(c, scale) {
		t.Error("expected WouldScale=true for currency with All category")
	}

	n := types.NewNumber(decimal.NewFromInt(10))
	if !WouldScale(n, scale) {
		t.Error("expected WouldScale=true for number with All category")
	}
}

func TestApply_CustomCategoryScalesCustomUnits(t *testing.T) {
	scale := &document.ScaleConfig{Factor: decimal.NewFromInt(2), UnitCategories: []string{"Custom"}}

	// Custom unit like "bananas" has empty category — should be scaled with Custom
	q := &types.Quantity{Value: decimal.NewFromInt(3), Unit: "bananas"}
	result := Apply(q, scale, nil)
	if qty := result.(*types.Quantity); !qty.Value.Equal(decimal.NewFromInt(6)) {
		t.Errorf("expected bananas scaled to 6, got %s", qty.Value.String())
	}

	// Known unit like "grams" should NOT be scaled (only Custom is listed)
	g := &types.Quantity{Value: decimal.NewFromInt(100), Unit: "grams"}
	result = Apply(g, scale, nil)
	if qty := result.(*types.Quantity); !qty.Value.Equal(decimal.NewFromInt(100)) {
		t.Errorf("expected grams unchanged (not in Custom), got %s", qty.Value.String())
	}
}

func TestWouldScale_CustomCategory(t *testing.T) {
	scale := &document.ScaleConfig{Factor: decimal.NewFromInt(2), UnitCategories: []string{"Custom"}}

	// Custom unit — should report would scale
	q := &types.Quantity{Value: decimal.NewFromInt(3), Unit: "eggs"}
	if !WouldScale(q, scale) {
		t.Error("expected WouldScale=true for custom unit with Custom category")
	}

	// Known unit — should not report would scale
	g := &types.Quantity{Value: decimal.NewFromInt(100), Unit: "grams"}
	if WouldScale(g, scale) {
		t.Error("expected WouldScale=false for known unit with only Custom category")
	}
}

func TestApply_AllCategoryAlsoScalesCustomUnits(t *testing.T) {
	scale := &document.ScaleConfig{Factor: decimal.NewFromInt(3), UnitCategories: []string{"All"}}

	q := &types.Quantity{Value: decimal.NewFromInt(5), Unit: "bananas"}
	result := Apply(q, scale, nil)
	if qty := result.(*types.Quantity); !qty.Value.Equal(decimal.NewFromInt(15)) {
		t.Errorf("expected bananas scaled to 15 with All, got %s", qty.Value.String())
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

// --- WouldConvert tests ---

func TestWouldConvert_QuantityToSI(t *testing.T) {
	q := &types.Quantity{Value: decimal.NewFromInt(2), Unit: "cups"}
	convertTo := &document.ConvertToConfig{System: "si", UnitCategories: []string{"Volume"}}
	if !WouldConvert(q, convertTo) {
		t.Error("expected cups to be converted to SI")
	}
}

func TestWouldConvert_ExplicitQuantity(t *testing.T) {
	q := &types.Quantity{Value: decimal.NewFromInt(280), Unit: "grams", IsExplicit: true}
	convertTo := &document.ConvertToConfig{System: "imperial"}
	if WouldConvert(q, convertTo) {
		t.Error("expected explicit quantity to skip conversion")
	}
}

func TestWouldConvert_CustomUnit(t *testing.T) {
	q := &types.Quantity{Value: decimal.NewFromInt(3), Unit: "buns"}
	convertTo := &document.ConvertToConfig{System: "si"}
	if WouldConvert(q, convertTo) {
		t.Error("expected custom unit to skip conversion")
	}
}

func TestWouldConvert_AlreadyInTarget(t *testing.T) {
	q := &types.Quantity{Value: decimal.NewFromInt(500), Unit: "ml"}
	convertTo := &document.ConvertToConfig{System: "si"}
	if WouldConvert(q, convertTo) {
		t.Error("expected ml (already SI) to skip conversion")
	}
}

func TestWouldConvert_CategoryMismatch(t *testing.T) {
	q := &types.Quantity{Value: decimal.NewFromInt(280), Unit: "grams"}
	convertTo := &document.ConvertToConfig{System: "imperial", UnitCategories: []string{"Volume"}}
	if WouldConvert(q, convertTo) {
		t.Error("expected grams (Mass) not to convert when only Volume in categories")
	}
}

func TestWouldConvert_NilConfig(t *testing.T) {
	q := &types.Quantity{Value: decimal.NewFromInt(280), Unit: "grams"}
	if WouldConvert(q, nil) {
		t.Error("expected false with nil convertTo")
	}
}

func TestWouldConvert_NilResult(t *testing.T) {
	convertTo := &document.ConvertToConfig{System: "si"}
	if WouldConvert(nil, convertTo) {
		t.Error("expected false with nil result")
	}
}

func TestWouldConvert_Rate(t *testing.T) {
	r := &types.Rate{
		Amount:  &types.Quantity{Value: decimal.NewFromInt(100), Unit: "km/h"},
		PerUnit: "hour",
	}
	convertTo := &document.ConvertToConfig{System: "imperial"}
	if !WouldConvert(r, convertTo) {
		t.Error("expected rate with convertible amount to return true")
	}
}

func TestWouldConvert_Boolean(t *testing.T) {
	b := types.NewBoolean(false)
	convertTo := &document.ConvertToConfig{System: "si"}
	if WouldConvert(b, convertTo) {
		t.Error("expected boolean to skip conversion")
	}
}

func TestWouldConvert_Number(t *testing.T) {
	n := types.NewNumber(decimal.NewFromInt(42))
	convertTo := &document.ConvertToConfig{System: "si"}
	if WouldConvert(n, convertTo) {
		t.Error("expected number to skip conversion")
	}
}
