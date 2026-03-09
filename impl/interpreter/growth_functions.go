package interpreter

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/parser"
	"github.com/CalcMark/go-calcmark/spec/types"
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
// Uses integer exponentiation when possible for O(log n) performance.
func compoundGrowth(principal, rate, periods decimal.Decimal) decimal.Decimal {
	base := decOne.Add(rate) // (1 + rate)
	var factor decimal.Decimal

	// Use integer exponentiation when periods is a whole number
	if periods.Equal(periods.Truncate(0)) {
		factor = base.Pow(periods)
	} else {
		factor = base.Pow(periods)
	}

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

// --- Value extraction helpers ---

// extractDecimalValue extracts the decimal value from any CalcMark type.
func extractDecimalValue(val types.Type) (decimal.Decimal, error) {
	switch v := val.(type) {
	case *types.Number:
		return v.Value, nil
	case *types.Currency:
		return v.Value, nil
	case *types.Quantity:
		return v.Value, nil
	case *types.Duration:
		return v.Value, nil
	default:
		return decZero, fmt.Errorf("cannot extract numeric value from %T", val)
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

// validateRate checks that rate is within (-1, 1] i.e. (-100%, 100%].
func validateRate(rate decimal.Decimal) error {
	if rate.GreaterThan(decOne) || rate.LessThanOrEqual(decNegOne) {
		return fmt.Errorf("compound: rate must be between -100%% and 100%% (exclusive), got %s%%",
			rate.Mul(decHundred).StringFixed(0))
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

	// Evaluate principal (1st arg)
	principalVal, err := interp.evalNode(f.Arguments[0])
	if err != nil {
		return nil, err
	}
	principal, err := extractDecimalValue(principalVal)
	if err != nil {
		return nil, fmt.Errorf("compound: invalid principal: %w", err)
	}

	// Evaluate rate (2nd arg)
	rateVal, err := interp.evalNode(f.Arguments[1])
	if err != nil {
		return nil, err
	}
	rate, err := extractDecimalValue(rateVal)
	if err != nil {
		return nil, fmt.Errorf("compound: invalid rate: %w", err)
	}
	if err := validateRate(rate); err != nil {
		return nil, err
	}

	// Evaluate duration/periods (3rd arg)
	periodsVal, err := interp.evalNode(f.Arguments[2])
	if err != nil {
		return nil, err
	}
	periodsNum, periodUnit, err := extractPeriodsFromDuration(periodsVal)
	if err != nil {
		return nil, fmt.Errorf("compound: invalid periods: %w", err)
	}

	// 3-arg form: compound(principal, rate, periods) — simple compound growth
	if len(f.Arguments) == 3 {
		if err := validatePeriods(periodsNum); err != nil {
			return nil, err
		}
		result := compoundGrowth(principal, rate, periodsNum)
		return wrapResult(result, principalVal), nil
	}

	// 4-arg form: check if 4th arg is a modifier (identifier) or a value
	modifierIdent, isIdent := f.Arguments[3].(*ast.Identifier)
	if !isIdent {
		return nil, fmt.Errorf("compound: 4th argument must be a period identifier (e.g., month) or compounded modifier")
	}

	modName := modifierIdent.Name

	// Mode 3: "compounded:monthly" — financial compounding A = P(1+r/n)^(nt)
	if freq, ok := strings.CutPrefix(modName, "compounded:"); ok {
		periodsPerYear, ok := types.PeriodToPeriodsPerYear(freq)
		if !ok {
			return nil, fmt.Errorf("compound: unknown compounding frequency %q", freq)
		}

		// Duration must be in years for financial formula
		totalYears := periodsNum
		if periodUnit != "" && periodUnit != "year" && periodUnit != "years" {
			// Convert duration to years
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

	// Bare frequency adverb (e.g., "monthly", "quarterly") — treat as
	// compounding frequency, equivalent to "compounded:monthly".
	// Distinguished from base period names (month, quarter) which are Mode 2 annotations.
	if isFrequencyAdverb(modName) {
		periodsPerYear, _ := types.PeriodToPeriodsPerYear(modName)
		ppyDec := decimal.NewFromInt(int64(periodsPerYear))

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

	// Evaluate starting amount
	amountVal, err := interp.evalNode(f.Arguments[0])
	if err != nil {
		return nil, err
	}
	amount, err := extractDecimalValue(amountVal)
	if err != nil {
		return nil, fmt.Errorf("grow: invalid starting amount: %w", err)
	}

	// Evaluate increment
	incrementVal, err := interp.evalNode(f.Arguments[1])
	if err != nil {
		return nil, err
	}
	increment, err := extractDecimalValue(incrementVal)
	if err != nil {
		return nil, fmt.Errorf("grow: invalid increment: %w", err)
	}

	// Evaluate periods
	periodsVal, err := interp.evalNode(f.Arguments[2])
	if err != nil {
		return nil, err
	}
	periodsNum, _, err := extractPeriodsFromDuration(periodsVal)
	if err != nil {
		return nil, fmt.Errorf("grow: invalid periods: %w", err)
	}

	result := linearGrow(amount, increment, periodsNum)
	return wrapResult(result, amountVal), nil
}

func evalDepreciateFunc(interp *Interpreter, f *ast.FunctionCall) (types.Type, error) {
	if len(f.Arguments) < 3 || len(f.Arguments) > 4 {
		return nil, fmt.Errorf("depreciate() requires 3 or 4 arguments (value, rate, periods, salvage?)")
	}

	// Evaluate principal value
	valueVal, err := interp.evalNode(f.Arguments[0])
	if err != nil {
		return nil, err
	}
	principal, err := extractDecimalValue(valueVal)
	if err != nil {
		return nil, fmt.Errorf("depreciate: invalid value: %w", err)
	}

	// Evaluate rate
	rateVal, err := interp.evalNode(f.Arguments[1])
	if err != nil {
		return nil, err
	}
	rate, err := extractDecimalValue(rateVal)
	if err != nil {
		return nil, fmt.Errorf("depreciate: invalid rate: %w", err)
	}
	if rate.LessThanOrEqual(decZero) {
		return nil, fmt.Errorf("depreciate: rate must be positive, got %s%%", rate.Mul(decHundred).StringFixed(0))
	}

	// Evaluate periods
	periodsVal, err := interp.evalNode(f.Arguments[2])
	if err != nil {
		return nil, err
	}
	periodsNum, _, err := extractPeriodsFromDuration(periodsVal)
	if err != nil {
		return nil, fmt.Errorf("depreciate: invalid periods: %w", err)
	}

	if err := validatePeriods(periodsNum); err != nil {
		return nil, err
	}

	// Depreciation = compound growth with negative rate
	result := compoundGrowth(principal, rate.Neg(), periodsNum)

	// Check for salvage floor (4th arg)
	if len(f.Arguments) == 4 {
		salvageVal, err := interp.evalNode(f.Arguments[3])
		if err != nil {
			return nil, err
		}
		salvage, err := extractDecimalValue(salvageVal)
		if err != nil {
			return nil, fmt.Errorf("depreciate: invalid salvage value: %w", err)
		}

		// Apply salvage floor
		if result.LessThan(salvage) {
			result = salvage
		}
	}

	return wrapResult(result, valueVal), nil
}
