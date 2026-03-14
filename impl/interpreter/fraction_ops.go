package interpreter

import (
	"fmt"
	"math/big"

	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

// Fraction arithmetic operations.
// Pure functions for easier testing.

// fractionToNumber converts a Fraction to a Number using exact Rat→Decimal conversion.
func fractionToNumber(f *types.Fraction) *types.Number {
	return types.NewNumber(decimal.NewFromBigRat(f.Value, 15))
}

// numberToFraction converts a Number to a Fraction using exact Decimal→Rat conversion.
func numberToFraction(n *types.Number) *types.Fraction {
	rat := n.Value.Rat()
	return types.NewFractionFromRat(rat)
}

// evalFractionOperation performs arithmetic on two Fractions.
func evalFractionOperation(left, right *types.Fraction, operator string) (types.Type, error) {
	// Same-type * and / on fractions with units is nonsensical (e.g., cup * cup = square cups).
	// Only bare (dimensionless) fractions can multiply/divide freely.
	if (operator == "*" || operator == "/") && left.Unit != "" && right.Unit != "" {
		if operator == "*" {
			return nil, fmt.Errorf("cannot multiply quantity by quantity — the result would be \"square %s\" which isn't a real unit. Use number() to get the raw values: number(%s) * number(%s)",
				left.Unit, left.String(), right.String())
		}
		return nil, fmt.Errorf("cannot divide quantity by quantity — dividing %s by %s doesn't produce %s. Use number() to get a ratio: number(%s) / number(%s)",
			left.Unit, right.Unit, left.Unit, left.String(), right.String())
	}

	result := new(big.Rat)

	switch operator {
	case "+":
		result.Add(left.Value, right.Value)
	case "-":
		result.Sub(left.Value, right.Value)
	case "*":
		result.Mul(left.Value, right.Value)
	case "/":
		if right.Value.Sign() == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		result.Quo(left.Value, right.Value)
	case "%":
		if right.Value.Sign() == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		// Modulus for fractions: a - floor(a/b) * b
		quo := new(big.Rat).Quo(left.Value, right.Value)
		// Truncate to integer
		wholePart := new(big.Int).Quo(quo.Num(), quo.Denom())
		wholeRat := new(big.Rat).SetInt(wholePart)
		product := new(big.Rat).Mul(wholeRat, right.Value)
		result.Sub(left.Value, product)
	default:
		return nil, fmt.Errorf("unsupported operator '%s' for fractions", operator)
	}

	f := types.NewFractionFromRat(result)

	// Preserve unit: left wins if present, otherwise inherit from right.
	// This handles mixed numbers like "12 1/2 pints" where the integer part
	// (left, no unit) combines with the fraction part (right, unit="pints").
	if left.Unit != "" {
		f.Unit = left.Unit
	} else if right.Unit != "" {
		f.Unit = right.Unit
	}

	// Check computation limits — if exceeded, convert to decimal
	if f.ExceedsComputationLimit() {
		num := fractionToNumber(f)
		if f.Unit != "" {
			return &types.Quantity{Value: num.Value, Unit: f.Unit}, nil
		}
		return num, nil
	}

	return f, nil
}

// fractionPow computes exact (base)^exp for integer exponents.
// Handles negative exponents: (2/3)^(-2) = (3/2)^2 = 9/4.
func fractionPow(base *types.Fraction, exp int64) (types.Type, error) {
	if exp == 0 {
		f, _ := types.NewFraction(1, 1)
		return f, nil
	}

	num := new(big.Int).Set(base.Num())
	denom := new(big.Int).Set(base.Denom())

	// Handle negative exponents: invert the base
	if exp < 0 {
		if num.Sign() == 0 {
			return nil, fmt.Errorf("division by zero: 0 raised to negative power")
		}
		num, denom = denom, num
		exp = -exp
		// Handle sign: if original numerator was negative, sign moves to new numerator
		if denom.Sign() < 0 {
			num.Neg(num)
			denom.Neg(denom)
		}
	}

	bigExp := big.NewInt(exp)
	resultNum := new(big.Int).Exp(num, bigExp, nil)
	resultDenom := new(big.Int).Exp(denom, bigExp, nil)

	result := new(big.Rat).SetFrac(resultNum, resultDenom)
	f := types.NewFractionFromRat(result)

	// Check computation limits
	if f.ExceedsComputationLimit() {
		return fractionToNumber(f), nil
	}

	return f, nil
}

// maxExponent is the cap for fraction exponentiation to prevent DoS.
const maxExponent = 100

// denominatorBitLenLimit is the pre-check threshold for denominator product size.
const denominatorBitLenLimit = 30

// isqrt returns the integer square root of n, or nil if n is negative.
func isqrt(n *big.Int) *big.Int {
	if n.Sign() < 0 {
		return nil
	}
	return new(big.Int).Sqrt(n)
}

// isPerfectSquare checks if n is a perfect square.
func isPerfectSquare(n *big.Int) bool {
	if n.Sign() < 0 {
		return false
	}
	root := isqrt(n)
	if root == nil {
		return false
	}
	return new(big.Int).Mul(root, root).Cmp(n) == 0
}
