package types

import (
	"fmt"
	"math/big"

	"github.com/shopspring/decimal"
)

// Fraction represents an exact rational number using Go's math/big.Rat.
// Fractions are always stored in lowest terms (big.Rat normalizes automatically).
type Fraction struct {
	Value    *big.Rat
	IsNapkin bool   // true if this value was rounded by "as napkin"
	Unit     string // empty for dimensionless fractions
}

// MaxDisplayDenominator is the threshold above which fractions fall back to decimal display.
const MaxDisplayDenominator = 1000

// maxComputationDenominator is the threshold above which fractions permanently convert to decimal.
var maxComputationDenominator = new(big.Int).Exp(big.NewInt(10), big.NewInt(9), nil) // 10^9

// maxNumeratorMagnitude is the threshold above which fractions convert to decimal for int64 safety.
var maxNumeratorMagnitude = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil) // 10^18

// NewFraction creates a new Fraction from numerator and denominator.
// Returns an error if denominator is zero.
// The fraction is automatically reduced to lowest terms by big.Rat.
func NewFraction(num, denom int64) (*Fraction, error) {
	if denom == 0 {
		return nil, fmt.Errorf("division by zero: denominator cannot be zero")
	}
	rat := new(big.Rat).SetFrac64(num, denom)
	return &Fraction{Value: rat}, nil
}

// NewFractionFromRat creates a Fraction from an existing big.Rat.
func NewFractionFromRat(rat *big.Rat) *Fraction {
	return &Fraction{Value: new(big.Rat).Set(rat)}
}

// Num returns the numerator of the fraction (after GCD reduction).
func (f *Fraction) Num() *big.Int {
	return f.Value.Num()
}

// Denom returns the denominator of the fraction (after GCD reduction).
func (f *Fraction) Denom() *big.Int {
	return f.Value.Denom()
}

// ToDecimal converts the fraction to a shopspring/decimal.Decimal.
// Uses NewFromBigRat for exact conversion — never goes through float64.
func (f *Fraction) ToDecimal() decimal.Decimal {
	return decimal.NewFromBigRat(f.Value, 15)
}

// IsProper returns true if |numerator| < denominator.
func (f *Fraction) IsProper() bool {
	return new(big.Int).Abs(f.Num()).Cmp(f.Denom()) < 0
}

// ExceedsComputationLimit returns true if the denominator exceeds 10^9 after GCD reduction,
// or if |numerator| exceeds 10^18 (int64 safety).
func (f *Fraction) ExceedsComputationLimit() bool {
	if f.Denom().Cmp(maxComputationDenominator) > 0 {
		return true
	}
	absNum := new(big.Int).Abs(f.Num())
	return absNum.Cmp(maxNumeratorMagnitude) > 0
}

// String returns the display representation of the fraction.
// Rules:
//  1. denominator == 1 → integer
//  2. denominator > 1000 → decimal fallback
//  3. |numerator| > denominator → mixed number (e.g., "2 1/3")
//  4. else → simple fraction (e.g., "2/3")
//  5. IsNapkin → prefix "~"
//  6. Unit → append " cup" etc.
func (f *Fraction) String() string {
	num := new(big.Int).Set(f.Num())
	denom := f.Denom()

	// Handle negative: track sign, work with absolute numerator
	negative := num.Sign() < 0
	if negative {
		num.Abs(num)
	}

	var result string

	// Rule 1: denominator == 1 → integer
	if denom.Cmp(big.NewInt(1)) == 0 {
		result = num.String()
	} else if denom.Cmp(big.NewInt(MaxDisplayDenominator)) > 0 {
		// Rule 2: denominator > 1000 → decimal fallback
		d := f.ToDecimal()
		if negative {
			d = d.Abs()
		}
		result = d.String()
	} else if num.Cmp(denom) >= 0 {
		// Rule 3: improper fraction → mixed number
		whole := new(big.Int).Div(num, denom)
		remainder := new(big.Int).Mod(num, denom)
		if remainder.Sign() == 0 {
			result = whole.String()
		} else {
			result = fmt.Sprintf("%s %s/%s", whole, remainder, denom)
		}
	} else {
		// Rule 4: proper fraction
		result = fmt.Sprintf("%s/%s", num, denom)
	}

	// Apply sign
	if negative {
		result = "-" + result
	}

	// Rule 5: napkin prefix
	if f.IsNapkin {
		result = "~" + result
	}

	// Rule 6: unit suffix
	if f.Unit != "" {
		result = result + " " + f.Unit
	}

	return result
}
