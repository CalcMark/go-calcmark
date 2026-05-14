package interpreter

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/v2/spec/ast"
	"github.com/CalcMark/go-calcmark/v2/spec/parser"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
	"github.com/shopspring/decimal"
)

var (
	decZero    = decimal.NewFromInt(0)
	decOne     = decimal.NewFromInt(1)
	decNegOne  = decimal.NewFromInt(-1)
	decHundred = decimal.NewFromInt(100)
)

// --- Core computation functions ---

// compoundGrowth computes principal * (1 + rate)^periods.
func compoundGrowth(principal, rate, periods decimal.Decimal) decimal.Decimal {
	factor := decOne.Add(rate).Pow(periods)
	return principal.Mul(factor).Round(20)
}

// compoundGrowthFinancial computes A = P * (1 + r/n)^(n*t) for financial compounding.
// nominalRate is the annual rate, periodsPerYear is n, totalYears is t.
func compoundGrowthFinancial(principal, nominalRate decimal.Decimal, periodsPerYear, totalYears decimal.Decimal) decimal.Decimal {
	ratePerPeriod := nominalRate.DivRound(periodsPerYear, 20)
	totalPeriods := periodsPerYear.Mul(totalYears)
	return compoundGrowth(principal, ratePerPeriod, totalPeriods)
}

// linearGrow computes startingAmount + (increment * periods).
func linearGrow(startingAmount, increment, periods decimal.Decimal) decimal.Decimal {
	return startingAmount.Add(increment.Mul(periods))
}

// requireAdditiveValue rejects types that can't sit in an additive
// expression like `amount + (increment × periods)`. Percentage, Rate,
// and Duration are out — silently coercing them to decimals produces
// nonsense (5% becomes 0.05, "5 hours" becomes 5, neither of which is
// what the user meant). Number, Quantity, and Currency are in.
//
// `paramName` is included in the error message so callers don't need
// to wrap. Used by grow's amount/increment, depreciate's value/salvage,
// compound's principal — anywhere a "starting value" or "additive
// increment" is expected.
func requireAdditiveValue(funcName, paramName string, val types.Type) (decimal.Decimal, error) {
	switch v := val.(type) {
	case *types.Number:
		return v.Value, nil
	case *types.Currency:
		return v.Value, nil
	case *types.Quantity:
		return v.Value, nil
	case *types.Percentage:
		return decZero, fmt.Errorf("%s: %s must be a number, quantity, or currency — got percentage", funcName, paramName)
	case *types.Duration:
		return decZero, fmt.Errorf("%s: %s must be a number, quantity, or currency — got duration", funcName, paramName)
	case *types.Rate:
		return decZero, fmt.Errorf("%s: %s must be a number, quantity, or currency — got rate", funcName, paramName)
	default:
		return decZero, fmt.Errorf("%s: invalid %s: cannot extract numeric value from %T", funcName, paramName, val)
	}
}

// requirePeriodsCount accepts a plain Number only. The argument is an
// iteration count, not a duration — `5 months` was previously coerced
// to 5 silently, which obscured the unit-mismatch question. Reject it
// so users write what they mean (`grow $100 by $20 over 5`, not `over
// 5 months`). Includes Percentage in the rejected set for the same
// reason: percentage iterations have no semantic meaning.
func requirePeriodsCount(funcName string, val types.Type) (decimal.Decimal, error) {
	switch v := val.(type) {
	case *types.Number:
		return v.Value, nil
	case *types.Duration:
		return decZero, fmt.Errorf("%s: periods must be a plain number (iteration count) — got duration; write the count without a time unit", funcName)
	case *types.Quantity:
		return decZero, fmt.Errorf("%s: periods must be a plain number (iteration count) — got quantity %s; drop the unit", funcName, v.Unit)
	case *types.Percentage:
		return decZero, fmt.Errorf("%s: periods must be a plain number (iteration count) — got percentage", funcName)
	case *types.Rate:
		return decZero, fmt.Errorf("%s: periods must be a plain number (iteration count) — got rate", funcName)
	case *types.Currency:
		return decZero, fmt.Errorf("%s: periods must be a plain number (iteration count) — got currency", funcName)
	default:
		return decZero, fmt.Errorf("%s: cannot extract period count from %T", funcName, val)
	}
}

// requirePercentageRate accepts a Percentage. A bare Number is also
// accepted for legacy reasons (5 read as 5%) — but new tests should
// pass a Percentage explicitly. Currency, Quantity, Rate, Duration
// are rejected.
func requirePercentageRate(funcName string, val types.Type) (decimal.Decimal, error) {
	switch v := val.(type) {
	case *types.Percentage:
		return v.Value, nil
	case *types.Number:
		return v.Value, nil
	case *types.Currency:
		return decZero, fmt.Errorf("%s: rate must be a percentage — got currency", funcName)
	case *types.Quantity:
		return decZero, fmt.Errorf("%s: rate must be a percentage — got quantity", funcName)
	case *types.Rate:
		return decZero, fmt.Errorf("%s: rate must be a percentage — got rate", funcName)
	case *types.Duration:
		return decZero, fmt.Errorf("%s: rate must be a percentage — got duration", funcName)
	default:
		return decZero, fmt.Errorf("%s: cannot extract rate from %T", funcName, val)
	}
}

// extractPeriodsFromDuration extracts the number of periods from a Duration.
// For "10 years", returns (10, "year"). For a plain number, returns (n, "").
func extractPeriodsFromDuration(val types.Type) (decimal.Decimal, string, error) {
	switch v := val.(type) {
	case *types.Duration:
		return v.Value, v.Unit, nil
	case *types.Number:
		return v.Value, "", nil
	case *types.Quantity:
		return v.Value, v.Unit, nil
	default:
		return decZero, "", fmt.Errorf("cannot extract period count from %T", val)
	}
}

// wrapResult wraps a decimal value in the same type as the original principal.
func wrapResult(result decimal.Decimal, original types.Type) types.Type {
	switch v := original.(type) {
	case *types.Currency:
		return &types.Currency{Value: result.Round(2), Symbol: v.Symbol, Code: v.Code}
	case *types.Quantity:
		return &types.Quantity{Value: result.Round(2), Unit: v.Unit}
	default:
		rounded := result.Round(2)
		return &types.Number{Value: rounded}
	}
}

// convertToMatchingUnit converts val to match target's unit when both are
// compatible quantities (first-unit-wins). Returns val unchanged if types
// don't both have units. Returns an error for incompatible quantity units.
func convertToMatchingUnit(target, val types.Type) (types.Type, error) {
	targetQty, targetIsQty := target.(*types.Quantity)
	valQty, valIsQty := val.(*types.Quantity)
	if !targetIsQty || !valIsQty {
		return val, nil
	}
	if targetQty.Unit == valQty.Unit {
		return val, nil
	}
	converted, err := convertQuantity(valQty, targetQty.Unit)
	if err != nil {
		return nil, fmt.Errorf("incompatible units %s and %s", targetQty.Unit, valQty.Unit)
	}
	return converted, nil
}

// validateRate checks that rate is within (-1, 1] i.e. (-100%, 100%].
// `fromBareNumber` is true when the rate came from a bare Number
// literal rather than a Percentage — in which case the failure is
// almost always "I typed 3 thinking 3%" and the message appends a
// "Did you mean N%?" nudge (issue #160). When the user already wrote
// `%` explicitly, no hint is added because the suggestion would
// duplicate what they typed.
func validateRate(rate decimal.Decimal, fromBareNumber bool) error {
	if rate.GreaterThan(decOne) || rate.LessThanOrEqual(decNegOne) {
		msg := fmt.Sprintf("compound: rate must be between -100%% and 100%% (exclusive), got %s%%",
			rate.Mul(decHundred).StringFixed(0))
		if fromBareNumber {
			// `3` → "Did you mean 3%?"; `-7` → "Did you mean -7%?"
			// Keep the original number formatting: integer-valued
			// decimals render without a fractional tail.
			msg += fmt.Sprintf(". Did you mean %s%%?", rate.String())
		}
		return fmt.Errorf("%s", msg)
	}
	return nil
}

// validatePeriods checks that period count doesn't exceed security limit.
func validatePeriods(periods decimal.Decimal) error {
	max := decimal.NewFromInt(int64(parser.MaxCompoundPeriods))
	if periods.GreaterThan(max) {
		return fmt.Errorf("compound: too many periods (%s exceeds limit of %d). Use a larger period or shorter duration",
			periods.StringFixed(0), parser.MaxCompoundPeriods)
	}
	return nil
}

// frequencyAdverbs are the adverbial forms that indicate financial compounding.
// Base period names (month, quarter, year) are Mode 2 semantic annotations.
var frequencyAdverbs = map[string]bool{
	"daily":     true,
	"weekly":    true,
	"monthly":   true,
	"quarterly": true,
	"yearly":    true,
}

// isFrequencyAdverb returns true for adverbial period forms (monthly, quarterly, etc.)
// that indicate financial compounding, as opposed to base period names (month, quarter)
// which are Mode 2 semantic annotations.
func isFrequencyAdverb(name string) bool {
	return frequencyAdverbs[strings.ToLower(name)]
}

// --- Eval function wrappers ---

func evalCompoundFunc(interp *Interpreter, f *ast.FunctionCall) (types.Type, error) {
	if len(f.Arguments) < 3 || len(f.Arguments) > 4 {
		return nil, fmt.Errorf("compound() requires 3 or 4 arguments (principal, rate, periods, modifier?)")
	}

	// principal: Additive (Number / Quantity / Currency)
	principalVal, err := interp.evalNode(f.Arguments[0])
	if err != nil {
		return nil, err
	}
	principal, err := requireAdditiveValue("compound", "principal", principalVal)
	if err != nil {
		return nil, err
	}

	// rate: Percentage (Number accepted for legacy)
	rateVal, err := interp.evalNode(f.Arguments[1])
	if err != nil {
		return nil, err
	}
	rate, err := requirePercentageRate("compound", rateVal)
	if err != nil {
		return nil, err
	}
	// Bare-Number rates trigger the "Did you mean N%?" hint in
	// validateRate when out of range — Percentage values silence it.
	_, fromBareNumber := rateVal.(*types.Number)
	if err := validateRate(rate, fromBareNumber); err != nil {
		return nil, err
	}

	// 3rd arg semantics depend on arg count:
	//   3-arg form  → iteration count (Number ONLY)
	//   4-arg form  → years / total time (Duration or Number-of-years),
	//                 because the 4th arg names a sub-period frequency
	//                 (`monthly`, `quarterly`, …) and the runtime
	//                 multiplies them together for the financial-
	//                 compounding formula.
	periodsVal, err := interp.evalNode(f.Arguments[2])
	if err != nil {
		return nil, err
	}

	// 3-arg form: compound(principal, rate, periods) — simple compound growth
	if len(f.Arguments) == 3 {
		periodsNum, err := requirePeriodsCount("compound", periodsVal)
		if err != nil {
			return nil, err
		}
		if err := validatePeriods(periodsNum); err != nil {
			return nil, err
		}
		result := compoundGrowth(principal, rate, periodsNum)
		return wrapResult(result, principalVal), nil
	}

	// 4-arg form: 3rd arg can be Duration (years) or Number-of-years.
	periodsNum, periodUnit, err := extractPeriodsFromDuration(periodsVal)
	if err != nil {
		return nil, fmt.Errorf("compound: invalid periods: %w", err)
	}

	// 4-arg form: check if 4th arg is a modifier (identifier) or a value
	modifierIdent, isIdent := f.Arguments[3].(*ast.Identifier)
	if !isIdent {
		return nil, fmt.Errorf("compound: 4th argument must be a period identifier (e.g., month) or compounded modifier")
	}

	modName := modifierIdent.Name

	// Resolve frequency: both "compounded:monthly" and bare "monthly" trigger
	// financial compounding A = P(1+r/n)^(nt). The "compounded:" prefix is
	// optional syntactic sugar.
	freq := ""
	if f, ok := strings.CutPrefix(modName, "compounded:"); ok {
		freq = f
	} else if isFrequencyAdverb(modName) {
		freq = modName
	}

	if freq != "" {
		periodsPerYear, ok := types.PeriodToPeriodsPerYear(freq)
		if !ok {
			return nil, fmt.Errorf("compound: unknown compounding frequency %q", freq)
		}

		// 3rd arg is years by default; convert non-year durations
		totalYears := periodsNum
		if periodUnit != "" && periodUnit != "year" && periodUnit != "years" {
			dur, err := types.NewDuration(periodsNum, periodUnit)
			if err != nil {
				return nil, fmt.Errorf("compound: invalid duration unit %q: %w", periodUnit, err)
			}
			yearDur, err := dur.Convert("years")
			if err != nil {
				return nil, fmt.Errorf("compound: cannot convert %s to years: %w", periodUnit, err)
			}
			totalYears = yearDur.Value
		}

		ppyDec := decimal.NewFromInt(int64(periodsPerYear))
		totalPeriods := ppyDec.Mul(totalYears)
		if err := validatePeriods(totalPeriods); err != nil {
			return nil, err
		}

		result := compoundGrowthFinancial(principal, rate, ppyDec, totalYears)
		return wrapResult(result, principalVal), nil
	}

	// Mode 2: explicit period (e.g., "year") — rate is per-period, periods = duration in that unit
	// Treat as simple compound growth with the given number of periods
	if err := validatePeriods(periodsNum); err != nil {
		return nil, err
	}
	result := compoundGrowth(principal, rate, periodsNum)
	return wrapResult(result, principalVal), nil
}

func evalGrowFunc(interp *Interpreter, f *ast.FunctionCall) (types.Type, error) {
	if len(f.Arguments) != 3 {
		return nil, fmt.Errorf("grow() requires exactly 3 arguments (amount, increment, periods)")
	}

	// Type contract — see growth_function_types_test.go for the
	// rationale and rejection matrix.

	// amount: Additive (Number / Quantity / Currency)
	amountVal, err := interp.evalNode(f.Arguments[0])
	if err != nil {
		return nil, err
	}
	amount, err := requireAdditiveValue("grow", "amount", amountVal)
	if err != nil {
		return nil, err
	}

	// increment: Additive matching amount's unit family
	incrementVal, err := interp.evalNode(f.Arguments[1])
	if err != nil {
		return nil, err
	}
	incrementVal, err = convertToMatchingUnit(amountVal, incrementVal)
	if err != nil {
		return nil, fmt.Errorf("grow: %w", err)
	}
	increment, err := requireAdditiveValue("grow", "increment", incrementVal)
	if err != nil {
		return nil, err
	}

	// periods: Number ONLY (iteration count, not a duration)
	periodsVal, err := interp.evalNode(f.Arguments[2])
	if err != nil {
		return nil, err
	}
	periodsNum, err := requirePeriodsCount("grow", periodsVal)
	if err != nil {
		return nil, err
	}

	result := linearGrow(amount, increment, periodsNum)
	return wrapResult(result, amountVal), nil
}

func evalDepreciateFunc(interp *Interpreter, f *ast.FunctionCall) (types.Type, error) {
	if len(f.Arguments) < 3 || len(f.Arguments) > 4 {
		return nil, fmt.Errorf("depreciate() requires 3 or 4 arguments (value, rate, periods, salvage?)")
	}

	// value: Additive
	valueVal, err := interp.evalNode(f.Arguments[0])
	if err != nil {
		return nil, err
	}
	principal, err := requireAdditiveValue("depreciate", "value", valueVal)
	if err != nil {
		return nil, err
	}

	// rate: Percentage
	rateVal, err := interp.evalNode(f.Arguments[1])
	if err != nil {
		return nil, err
	}
	rate, err := requirePercentageRate("depreciate", rateVal)
	if err != nil {
		return nil, err
	}
	if rate.LessThanOrEqual(decZero) {
		return nil, fmt.Errorf("depreciate: rate must be positive, got %s%%", rate.Mul(decHundred).StringFixed(0))
	}

	// periods: Number
	periodsVal, err := interp.evalNode(f.Arguments[2])
	if err != nil {
		return nil, err
	}
	periodsNum, err := requirePeriodsCount("depreciate", periodsVal)
	if err != nil {
		return nil, err
	}
	if err := validatePeriods(periodsNum); err != nil {
		return nil, err
	}

	// Depreciation = compound growth with negative rate
	result := compoundGrowth(principal, rate.Neg(), periodsNum)

	// Optional salvage floor (Additive matching value's type)
	if len(f.Arguments) == 4 {
		salvageVal, err := interp.evalNode(f.Arguments[3])
		if err != nil {
			return nil, err
		}
		salvageVal, err = convertToMatchingUnit(valueVal, salvageVal)
		if err != nil {
			return nil, fmt.Errorf("depreciate: %w", err)
		}
		salvage, err := requireAdditiveValue("depreciate", "salvage", salvageVal)
		if err != nil {
			return nil, err
		}
		if result.LessThan(salvage) {
			result = salvage
		}
	}

	return wrapResult(result, valueVal), nil
}
