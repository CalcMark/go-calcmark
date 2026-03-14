package interpreter

import (
	"fmt"
	"math/big"

	"github.com/CalcMark/go-calcmark/format/display"
	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

// evalNapkinConversion evaluates a napkin conversion expression.
// Returns a rounded value that preserves the input type.
// The value is rounded to 2 significant figures (adaptable).
//
// Type preservation:
//   - *types.Quantity  -> *types.Quantity (with normalized unit if value > 1000)
//   - *types.Currency  -> *types.Currency (same symbol, rounded value)
//   - *types.Rate      -> *types.Rate (same Amount.Unit and PerUnit, rounded value)
//   - *types.Duration  -> *types.Duration (same unit, rounded value)
//   - *types.Number    -> *types.Number (rounded value)
func (interp *Interpreter) evalNapkinConversion(n *ast.NapkinConversion) (types.Type, error) {
	// Evaluate the expression
	value, err := interp.evalNode(n.Expression)
	if err != nil {
		return nil, err
	}

	// Process based on input type - preserving the type
	switch v := value.(type) {
	case *types.Number:
		rounded := roundToNapkinPrecision(v.Value)
		return types.NewNumber(rounded), nil

	case *types.Quantity:
		// Round the value, then normalize to human-friendly unit
		rounded := roundToNapkinPrecision(v.Value)
		// Use NormalizeForDisplay to convert to human-friendly units (e.g., 432000 MB -> ~400 GB)
		normalizedValue, normalizedUnit := display.NormalizeForDisplay(rounded, v.Unit)
		return &types.Quantity{
			Value:    normalizedValue,
			Unit:     normalizedUnit,
			IsNapkin: true, // Mark as napkin estimate for display formatting
		}, nil

	case *types.Currency:
		// Preserve symbol, round value
		rounded := roundToNapkinPrecision(v.Value)
		return types.NewCurrency(rounded, v.Symbol), nil

	case *types.Duration:
		// Preserve unit, round value (keep in original unit)
		rounded := roundToNapkinPrecision(v.Value)
		duration, err := types.NewDuration(rounded, v.Unit)
		if err != nil {
			// Fallback: return with original unit even if validation fails
			return &types.Duration{Value: rounded, Unit: v.Unit}, nil
		}
		return duration, nil

	case *types.Rate:
		// Preserve Amount.Unit and PerUnit, round Amount.Value
		rounded := roundToNapkinPrecision(v.Amount.Value)
		return &types.Rate{
			Amount: &types.Quantity{
				Value: rounded,
				Unit:  v.Amount.Unit,
			},
			PerUnit: v.PerUnit,
		}, nil

	case *types.Fraction:
		result := roundFractionToNapkin(v)
		result.IsNapkin = true
		result.Unit = v.Unit
		return result, nil

	case *types.Percentage:
		// Percentages are already small decimal values; pass through unchanged.
		return v, nil

	default:
		return nil, fmt.Errorf("napkin conversion requires a numeric value, got %T", value)
	}
}

// roundToNapkinPrecision rounds a decimal value to 2 significant figures.
// This is the core napkin math rounding logic.
func roundToNapkinPrecision(numValue decimal.Decimal) decimal.Decimal {
	floatVal, _ := numValue.Abs().Float64()

	// Handle zero
	if floatVal == 0 {
		return decimal.Zero
	}

	var roundedFloat float64

	if floatVal >= 1000 {
		// Determine scale
		var magnitude float64
		if floatVal >= 1e12 {
			magnitude = 1e12
		} else if floatVal >= 1e9 {
			magnitude = 1e9
		} else if floatVal >= 1e6 {
			magnitude = 1e6
		} else {
			magnitude = 1000
		}

		// Scale, round to 2 sig figs, scale back
		scaled := floatVal / magnitude
		rounded := roundToSignificantFigures(scaled, 2)

		// Convert rounded value back to float
		var roundedScaled float64
		switch r := rounded.(type) {
		case string:
			if d, err := decimal.NewFromString(r); err == nil {
				roundedScaled, _ = d.Float64()
			} else {
				roundedScaled = scaled
			}
		case int:
			roundedScaled = float64(r)
		default:
			roundedScaled = scaled
		}

		roundedFloat = roundedScaled * magnitude
	} else {
		// For smaller numbers < 1000, round to 2 sig figs
		rounded := roundToSignificantFigures(floatVal, 2)
		switch r := rounded.(type) {
		case string:
			if d, err := decimal.NewFromString(r); err == nil {
				roundedFloat, _ = d.Float64()
			} else {
				roundedFloat = floatVal
			}
		case int:
			roundedFloat = float64(r)
		default:
			roundedFloat = floatVal
		}
	}

	// Preserve sign
	if numValue.IsNegative() {
		roundedFloat = -roundedFloat
	}

	return decimal.NewFromFloat(roundedFloat)
}

// napkinDenominators are the common fraction denominators for napkin rounding.
var napkinDenominators = []int64{1, 2, 3, 4, 6, 8}

// roundFractionToNapkin rounds a fraction to the nearest common fraction.
// Common denominators: 1, 2, 3, 4, 6, 8.
// For mixed numbers, the integer part is preserved and only the fractional remainder is rounded.
func roundFractionToNapkin(f *types.Fraction) *types.Fraction {
	// Extract sign and work with absolute value
	negative := f.Value.Sign() < 0
	abs := new(big.Rat).Abs(f.Value)

	// Extract integer part
	wholePart := new(big.Int).Div(abs.Num(), abs.Denom())
	remainder := new(big.Rat).Sub(abs, new(big.Rat).SetInt(wholePart))

	// If remainder is zero, return the integer
	if remainder.Sign() == 0 {
		result := new(big.Rat).SetInt(wholePart)
		if negative {
			result.Neg(result)
		}
		return types.NewFractionFromRat(result)
	}

	// Find nearest common fraction for the remainder
	bestCandidate := new(big.Rat)
	bestDist := new(big.Rat).SetFrac64(1, 1) // Start with max distance

	for _, d := range napkinDenominators {
		// Find nearest p/d to remainder: p = round(remainder * d)
		scaled := new(big.Rat).Mul(remainder, new(big.Rat).SetInt64(d))
		// Round: add 0.5 and truncate
		scaledPlusHalf := new(big.Rat).Add(scaled, new(big.Rat).SetFrac64(1, 2))
		p := new(big.Int).Div(scaledPlusHalf.Num(), scaledPlusHalf.Denom())

		candidate := new(big.Rat).SetFrac(p, big.NewInt(d))

		// Compute distance
		dist := new(big.Rat).Sub(remainder, candidate)
		dist.Abs(dist)

		if dist.Cmp(bestDist) < 0 {
			bestDist.Set(dist)
			bestCandidate.Set(candidate)
		}
	}

	// Combine: whole + best candidate
	result := new(big.Rat).Add(new(big.Rat).SetInt(wholePart), bestCandidate)
	if negative {
		result.Neg(result)
	}
	return types.NewFractionFromRat(result)
}
