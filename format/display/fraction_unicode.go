package display

import (
	"fmt"
	"math/big"

	"github.com/CalcMark/go-calcmark/v2/spec/types"
)

// unicodeFractions maps numerator/denominator pairs to Unicode Number Forms characters.
// These are from the Unicode "Number Forms" block (U+2150–U+215E) and Latin-1 Supplement.
// Only fractions with dedicated Unicode codepoints are included.
var unicodeFractions = map[[2]int64]string{
	{1, 2}:  "½", // U+00BD
	{1, 3}:  "⅓", // U+2153
	{2, 3}:  "⅔", // U+2154
	{1, 4}:  "¼", // U+00BC
	{3, 4}:  "¾", // U+00BE
	{1, 5}:  "⅕", // U+2155
	{2, 5}:  "⅖", // U+2156
	{3, 5}:  "⅗", // U+2157
	{4, 5}:  "⅘", // U+2158
	{1, 6}:  "⅙", // U+2159
	{5, 6}:  "⅚", // U+215A
	{1, 7}:  "⅐", // U+2150
	{1, 8}:  "⅛", // U+215B
	{3, 8}:  "⅜", // U+215C
	{5, 8}:  "⅝", // U+215D
	{7, 8}:  "⅞", // U+215E
	{1, 9}:  "⅑", // U+2151
	{1, 10}: "⅒", // U+2152
}

// FormatFractionUnicode renders a Fraction using Unicode Number Forms where available,
// falling back to ASCII (e.g., "7/12") for fractions without a Unicode codepoint.
// Mixed numbers combine integer + Unicode fraction: "2⅓" instead of "2 1/3".
func FormatFractionUnicode(f *types.Fraction) string {
	num := new(big.Int).Set(f.Num())
	denom := f.Denom()

	negative := num.Sign() < 0
	if negative {
		num.Abs(num)
	}

	var result string

	// Denominator == 1 → integer
	if denom.Cmp(big.NewInt(1)) == 0 {
		result = num.String()
	} else if denom.Cmp(big.NewInt(types.MaxDisplayDenominator)) > 0 {
		// Denominator > 1000 → decimal fallback
		d := f.ToDecimal()
		if negative {
			d = d.Abs()
		}
		result = d.String()
	} else {
		// Try mixed number decomposition
		whole := new(big.Int).Div(num, denom)
		remainder := new(big.Int).Mod(num, denom)

		if remainder.Sign() == 0 {
			result = whole.String()
		} else {
			// Try Unicode lookup for the fractional part
			remInt := remainder.Int64()
			denomInt := denom.Int64()
			unicodeChar, hasUnicode := unicodeFractions[[2]int64{remInt, denomInt}]

			if whole.Sign() > 0 {
				if hasUnicode {
					result = fmt.Sprintf("%s%s", whole, unicodeChar)
				} else {
					result = fmt.Sprintf("%s %d/%d", whole, remInt, denomInt)
				}
			} else {
				if hasUnicode {
					result = unicodeChar
				} else {
					result = fmt.Sprintf("%d/%d", remInt, denomInt)
				}
			}
		}
	}

	if negative {
		result = "-" + result
	}

	if f.IsNapkin {
		result = "~" + result
	}

	if f.Unit != "" {
		result = result + " " + f.Unit
	}

	return result
}
