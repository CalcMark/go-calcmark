package interpreter

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/CalcMark/go-calcmark/v2/spec/ast"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
	"github.com/CalcMark/go-calcmark/v2/spec/units"
	"github.com/shopspring/decimal"
)

// Binary and unary operators.

func (interp *Interpreter) evalBinaryOp(b *ast.BinaryOp) (types.Type, error) {
	left, err := interp.evalNode(b.Left)
	if err != nil {
		return nil, err
	}

	right, err := interp.evalNode(b.Right)
	if err != nil {
		return nil, err
	}

	return evalBinaryOperation(left, right, b.Operator)
}

func (interp *Interpreter) evalComparisonOp(c *ast.ComparisonOp) (types.Type, error) {
	left, err := interp.evalNode(c.Left)
	if err != nil {
		return nil, err
	}

	right, err := interp.evalNode(c.Right)
	if err != nil {
		return nil, err
	}

	return evalComparison(left, right, c.Operator)
}

func (interp *Interpreter) evalUnaryOp(u *ast.UnaryOp) (types.Type, error) {
	operand, err := interp.evalNode(u.Operand)
	if err != nil {
		return nil, err
	}

	return evalUnaryOperation(operand, u.Operator)
}

// evalBinaryOperation performs binary arithmetic operations.
// This is a pure function for easier testing.
func evalBinaryOperation(left, right types.Type, operator string) (types.Type, error) {
	// Fraction normalization — handles Fraction×Fraction, Fraction×Number,
	// and Fraction as scalar multiplier for other types.

	// Fraction ^ anything: three cases
	if leftFrac, ok := left.(*types.Fraction); ok && operator == "^" {
		if rightNum, ok := right.(*types.Number); ok {
			// Case 1: exponent > maxExponent → convert to decimal (DoS protection)
			if rightNum.Value.Abs().GreaterThan(decimal.NewFromInt(maxExponent)) {
				return evalBinaryOperation(fractionToNumber(leftFrac), right, operator)
			}
			if rightNum.Value.Equal(rightNum.Value.Truncate(0)) {
				// Case 2: integer exponent → exact fraction result
				n := rightNum.Value.IntPart()
				return fractionPow(leftFrac, n)
			}
			// Case 3: fractional exponent → convert to decimal
			return evalBinaryOperation(fractionToNumber(leftFrac), right, operator)
		}
		// Fraction ^ Fraction → convert both to decimal
		if rightFrac, ok := right.(*types.Fraction); ok {
			return evalBinaryOperation(fractionToNumber(leftFrac), fractionToNumber(rightFrac), operator)
		}
	}

	// Denominator pre-check: if product would be too large, convert to decimal
	if leftFrac, ok := left.(*types.Fraction); ok {
		if rightFrac, ok := right.(*types.Fraction); ok {
			if leftFrac.Denom().BitLen()+rightFrac.Denom().BitLen() > denominatorBitLenLimit {
				return evalBinaryOperation(fractionToNumber(leftFrac), fractionToNumber(rightFrac), operator)
			}
			return evalFractionOperation(leftFrac, rightFrac, operator)
		}
		if _, ok := right.(*types.Number); ok {
			rightFrac := numberToFraction(right.(*types.Number))
			return evalFractionOperation(leftFrac, rightFrac, operator)
		}
		// Fraction × non-numeric type (Currency, Duration, Rate, Quantity)
		// → convert fraction to decimal, fall through to existing dispatch
		left = fractionToNumber(leftFrac)
	}
	// Mirror for right-side fraction
	if rightFrac, ok := right.(*types.Fraction); ok {
		if _, ok := left.(*types.Number); ok {
			leftFrac := numberToFraction(left.(*types.Number))
			return evalFractionOperation(leftFrac, rightFrac, operator)
		}
		// Non-numeric × Fraction → convert fraction to decimal
		right = fractionToNumber(rightFrac)
	}

	// Rate arithmetic widening: when a Rate appears on the RIGHT side of
	// * or /, extract the rate's Amount (a Quantity) and drop the time
	// denominator. This makes "2 * (2 posts/week)" yield "4 posts".
	//
	// Rate on the LEFT is NOT widened — it preserves rate semantics:
	//   Rate * Number → Rate  (scaling: "read_rate * 3" stays a rate)
	//   Rate / Number → Rate  (scaling)
	//   Rate * Quantity → Quantity (cross-type, already handled below)
	//   Rate / Rate → Number  (ratio, already handled below)
	if operator == "*" || operator == "/" {
		if rightRate, ok := right.(*types.Rate); ok {
			if _, leftIsRate := left.(*types.Rate); !leftIsRate {
				return evalBinaryOperation(left, rightRate.Amount, operator)
			}
		}
	}

	// Percentage arithmetic widening: when a Percentage appears on the RIGHT
	// side of + or -, apply it proportionally to the left operand.
	// "salary + 32%" means "salary * 1.32" (increase by 32%).
	// "price - 20%" means "price * 0.80" (decrease by 20%).
	//
	// Percentage on the LEFT with a non-Percentage right operand is an error.
	// Percentage + Percentage does decimal addition (32% + 10% = 42%).
	if operator == "+" || operator == "-" {
		if rightPct, ok := right.(*types.Percentage); ok {
			if _, leftIsPct := left.(*types.Percentage); !leftIsPct {
				return evalPercentageWidening(left, rightPct, operator)
			}
		}
		if _, leftIsPct := left.(*types.Percentage); leftIsPct {
			if _, rightIsPct := right.(*types.Percentage); !rightIsPct {
				return nil, fmt.Errorf("cannot add percentage to %s; percentage must appear on the right (e.g., value %s %s)",
					formatTypeForError(right), operator, left.(*types.Percentage).String())
			}
		}
	}

	// Percentage normalization for * and /: extract the decimal value.
	// "100 * 20%" = "100 * 0.2" = 20. No widening on multiplication.
	if operator == "*" || operator == "/" || operator == "%" || operator == "^" {
		if pct, ok := left.(*types.Percentage); ok {
			return evalBinaryOperation(types.NewNumber(pct.Value), right, operator)
		}
		if pct, ok := right.(*types.Percentage); ok {
			return evalBinaryOperation(left, types.NewNumber(pct.Value), operator)
		}
	}

	// Normalize unitless quantities to numbers before type dispatch.
	// Unitless quantities arise from accumulate/over on unitless rates,
	// or from rate widening on unitless rates (e.g., 100/second → Amount{100, ""}).
	if q, ok := left.(*types.Quantity); ok && q.Unit == "" {
		return evalBinaryOperation(types.NewNumber(q.Value), right, operator)
	}
	if q, ok := right.(*types.Quantity); ok && q.Unit == "" {
		return evalBinaryOperation(left, types.NewNumber(q.Value), operator)
	}

	// Boolean operations (AND, OR)
	if leftBool, ok := left.(*types.Boolean); ok {
		if rightBool, ok := right.(*types.Boolean); ok {
			switch operator {
			case "and":
				return types.NewBoolean(leftBool.Value && rightBool.Value), nil
			case "or":
				return types.NewBoolean(leftBool.Value || rightBool.Value), nil
			}
		}
		// AND/OR with non-boolean right operand
		if operator == "and" || operator == "or" {
			return nil, fmt.Errorf("'%s' operator requires boolean operands, got %T and %T", operator, left, right)
		}
	}

	// Number operations
	if leftNum, ok := left.(*types.Number); ok {
		if rightNum, ok := right.(*types.Number); ok {
			return evalNumberOperation(leftNum, rightNum, operator)
		}
		// Number op Currency → Currency (e.g., "2 + $10" = "$12", "2 * $10" = "$20")
		if rightCur, ok := right.(*types.Currency); ok {
			switch operator {
			case "+":
				return types.NewCurrency(leftNum.Value.Add(rightCur.Value), rightCur.Symbol), nil
			case "-":
				return types.NewCurrency(leftNum.Value.Sub(rightCur.Value), rightCur.Symbol), nil
			case "*":
				return types.NewCurrency(leftNum.Value.Mul(rightCur.Value), rightCur.Symbol), nil
			}
		}
		// Number * Duration → Duration
		if rightDur, ok := right.(*types.Duration); ok && operator == "*" {
			result := leftNum.Value.Mul(rightDur.Value)
			return &types.Duration{Value: result, Unit: rightDur.Unit}, nil
		}
		// Note: Number * Rate is widened above (rate on right → extract amount).
	}

	// Number * Quantity → Quantity (e.g., "2 * 10 dogs" already handled above)
	// Quantity * Currency → Currency (coerce quantity to number, then multiply)
	if leftQty, ok := left.(*types.Quantity); ok {
		if rightCur, ok := right.(*types.Currency); ok && operator == "*" {
			return types.NewCurrency(leftQty.Value.Mul(rightCur.Value), rightCur.Symbol), nil
		}
	}

	// Currency operations
	if leftCur, ok := left.(*types.Currency); ok {
		// Currency * Quantity → Currency (coerce quantity to number, then multiply)
		if rightQty, ok := right.(*types.Quantity); ok && operator == "*" {
			return types.NewCurrency(leftCur.Value.Mul(rightQty.Value), leftCur.Symbol), nil
		}
		// Currency op Number → Currency (e.g., "10 EUR + 2" = "12 EUR")
		if rightNum, ok := right.(*types.Number); ok {
			switch operator {
			case "+":
				return types.NewCurrency(leftCur.Value.Add(rightNum.Value), leftCur.Symbol), nil
			case "-":
				return types.NewCurrency(leftCur.Value.Sub(rightNum.Value), leftCur.Symbol), nil
			case "*":
				return types.NewCurrency(leftCur.Value.Mul(rightNum.Value), leftCur.Symbol), nil
			case "/":
				if rightNum.Value.IsZero() {
					return nil, fmt.Errorf("division by zero")
				}
				return types.NewCurrency(leftCur.Value.Div(rightNum.Value), leftCur.Symbol), nil
			}
		}
		// Currency / Duration → Rate (divide value by duration magnitude)
		// e.g., "$1000 / 4 days" = "$250/day"
		if rightDur, ok := right.(*types.Duration); ok && operator == "/" {
			if rightDur.Value.IsZero() {
				return nil, fmt.Errorf("division by zero")
			}
			return &types.Rate{
				Amount: &types.Quantity{
					Value: leftCur.Value.Div(rightDur.Value),
					Unit:  leftCur.Symbol,
				},
				PerUnit: types.NormalizeTimeUnit(rightDur.Unit),
			}, nil
		}
		// Currency op Currency (same type)
		if rightCur, ok := right.(*types.Currency); ok {
			if !leftCur.IsSameCurrency(rightCur) { // #95: compare by Code so $ and USD match
				return nil, fmt.Errorf("cannot %s different currencies: %s and %s",
					operator, leftCur.Code, rightCur.Code)
			}
			// Same-type * and / are nonsensical (square dollars, unitless ratio).
			// Users should use number() to extract the raw value first.
			switch operator {
			case "*":
				return nil, fmt.Errorf("cannot multiply currency by currency — the result would be \"square dollars\" which isn't a real unit. Use number() to get the raw values: number(%s) * number(%s)",
					leftCur.String(), rightCur.String())
			case "/":
				return nil, fmt.Errorf("cannot divide currency by currency — dividing %s by %s doesn't produce a dollar amount. Use number() to get a ratio: number(%s) / number(%s)",
					leftCur.String(), rightCur.String(), leftCur.String(), rightCur.String())
			}
			result, err := evalNumberOperation(
				&types.Number{Value: leftCur.Value},
				&types.Number{Value: rightCur.Value},
				operator,
			)
			if err != nil {
				return nil, err
			}
			return types.NewCurrency(result.(*types.Number).Value, leftCur.Symbol), nil
		}
	}

	// Period arithmetic (v2.0): asymmetric — `+` extends from the
	// period's END, `-` extends from its START. So `Q1 + 30 days` =
	// Mar 31 + 30 days; `Q1 - 5 days` = Jan 1 - 5 days. Period ==
	// Period is structural (start.Equal && end.Equal). Period ==
	// Date is a type error and falls through to the generic
	// type-mismatch error below.
	if leftPeriod, ok := left.(*types.Period); ok {
		if rightDur, ok := right.(*types.Duration); ok {
			switch operator {
			case "+":
				// End + Duration → Date past the period's end.
				return evalDateDurationOperation(leftPeriod.End, rightDur, "+")
			case "-":
				// Start - Duration → Date before the period's start.
				return evalDateDurationOperation(leftPeriod.Start, rightDur, "-")
			}
		}
		if rightPeriod, ok := right.(*types.Period); ok {
			switch operator {
			case "==":
				eq := leftPeriod.Start.Time.Equal(rightPeriod.Start.Time) &&
					leftPeriod.End.Time.Equal(rightPeriod.End.Time)
				return &types.Boolean{Value: eq}, nil
			case "!=":
				eq := leftPeriod.Start.Time.Equal(rightPeriod.Start.Time) &&
					leftPeriod.End.Time.Equal(rightPeriod.End.Time)
				return &types.Boolean{Value: !eq}, nil
			case "+", "-":
				return nil, fmt.Errorf("cannot %s two periods; use 'length of' to get a duration", operator)
			}
		}
	}

	// Date operations. Periods masquerading as dates (e.g., when a
	// Period flows through addition with a Duration on the right and
	// the Period was the LHS — already handled above; this Date arm
	// catches the case where a Period appears on the RHS of a Date
	// operation, which is unusual but not impossible).
	if leftDate, ok := left.(*types.Date); ok {
		if rightDur, ok := right.(*types.Duration); ok {
			return evalDateDurationOperation(leftDate, rightDur, operator)
		}
		if rightDate, ok := right.(*types.Date); ok {
			return evalDateDateOperation(leftDate, rightDate, operator)
		}
		// Date - Period: treat Period as its Start (legacy unwrap).
		if rightPeriod, ok := right.(*types.Period); ok && operator == "-" {
			return evalDateDateOperation(leftDate, rightPeriod.Start, "-")
		}
	}

	// Duration operations
	if leftDur, ok := left.(*types.Duration); ok {
		if rightDur, ok := right.(*types.Duration); ok {
			return evalDurationOperation(leftDur, rightDur, operator)
		}
		if rightNum, ok := right.(*types.Number); ok {
			return evalDurationNumberOperation(leftDur, rightNum, operator)
		}
	}

	// Rate operations (rate on the left — preserves rate type)
	if leftRate, ok := left.(*types.Rate); ok {
		// Rate * Number → Rate (scale the rate)
		// Rate / Number → Rate (divide the rate)
		if rightNum, ok := right.(*types.Number); ok {
			switch operator {
			case "*":
				return &types.Rate{
					Amount:  &types.Quantity{Value: leftRate.Amount.Value.Mul(rightNum.Value), Unit: leftRate.Amount.Unit},
					PerUnit: leftRate.PerUnit,
				}, nil
			case "/":
				if rightNum.Value.IsZero() {
					return nil, fmt.Errorf("division by zero")
				}
				return &types.Rate{
					Amount:  &types.Quantity{Value: leftRate.Amount.Value.Div(rightNum.Value), Unit: leftRate.Amount.Unit},
					PerUnit: leftRate.PerUnit,
				}, nil
			}
		}
		// Rate / Rate → Number (if same per-units, dimensionless ratio)
		if rightRate, ok := right.(*types.Rate); ok {
			if operator == "/" && leftRate.PerUnit == rightRate.PerUnit {
				result := leftRate.Amount.Value.Div(rightRate.Amount.Value)
				return types.NewNumber(result), nil
			}
		}
		// Rate * Quantity → Quantity (e.g., "100/second * 10 KB" = "1000 KB")
		if rightQty, ok := right.(*types.Quantity); ok && operator == "*" {
			result := leftRate.Amount.Value.Mul(rightQty.Value)
			return &types.Quantity{Value: result, Unit: rightQty.Unit}, nil
		}
	}

	// Quantity operations (with unit conversion - USER REQUIREMENT: first-unit-wins)
	if leftQty, ok := left.(*types.Quantity); ok {
		if rightQty, ok := right.(*types.Quantity); ok {
			return evalQuantityOperation(leftQty, rightQty, operator)
		}
		// Note: Quantity * Rate is handled by rate widening normalization above.
		// Quantity op Number (e.g., "10 dogs * 2" = "20 dogs", "5 dogs + 3" = "8 dogs")
		if rightNum, ok := right.(*types.Number); ok {
			switch operator {
			case "*":
				return &types.Quantity{Value: leftQty.Value.Mul(rightNum.Value), Unit: leftQty.Unit}, nil
			case "/":
				return &types.Quantity{Value: leftQty.Value.Div(rightNum.Value), Unit: leftQty.Unit}, nil
			case "+":
				return &types.Quantity{Value: leftQty.Value.Add(rightNum.Value), Unit: leftQty.Unit}, nil
			case "-":
				return &types.Quantity{Value: leftQty.Value.Sub(rightNum.Value), Unit: leftQty.Unit}, nil
			}
		}
	}

	// Number op Quantity (e.g., "2 * 10 dogs" = "20 dogs", "1 + 1 dogs" = "2 dogs")
	if leftNum, ok := left.(*types.Number); ok {
		if rightQty, ok := right.(*types.Quantity); ok {
			switch operator {
			case "*":
				return &types.Quantity{Value: leftNum.Value.Mul(rightQty.Value), Unit: rightQty.Unit}, nil
			case "+":
				return &types.Quantity{Value: leftNum.Value.Add(rightQty.Value), Unit: rightQty.Unit}, nil
			case "-":
				// Number - Quantity (keeps the quantity unit)
				return &types.Quantity{Value: leftNum.Value.Sub(rightQty.Value), Unit: rightQty.Unit}, nil
			}
		}
	}

	// Percentage + Percentage → Percentage (decimal add/sub)
	if leftPct, ok := left.(*types.Percentage); ok {
		if rightPct, ok := right.(*types.Percentage); ok {
			switch operator {
			case "+":
				return types.NewPercentage(leftPct.Value.Add(rightPct.Value)), nil
			case "-":
				return types.NewPercentage(leftPct.Value.Sub(rightPct.Value)), nil
			}
		}
	}

	// Bridge: Speed quantity * Duration → distance (e.g., "60 mph * 2 hours")
	// Only fires for Speed-category quantities, not all quantities.
	if operator == "*" {
		if leftQty, ok := left.(*types.Quantity); ok && units.IsSpeedUnit(leftQty.Unit) {
			if rightDur, ok := right.(*types.Duration); ok {
				return speedTimesDuration(leftQty, rightDur)
			}
		}
		if rightQty, ok := right.(*types.Quantity); ok && units.IsSpeedUnit(rightQty.Unit) {
			if leftDur, ok := left.(*types.Duration); ok {
				return speedTimesDuration(rightQty, leftDur)
			}
		}
	}

	// Provide helpful error messages for common mistakes
	return nil, unsupportedOperationError(left, right, operator)
}

// evalNumberOperation performs operations on two numbers.
func evalNumberOperation(left, right *types.Number, operator string) (types.Type, error) {
	var result decimal.Decimal

	switch operator {
	case "+":
		result = left.Value.Add(right.Value)
	case "-":
		result = left.Value.Sub(right.Value)
	case "*":
		result = left.Value.Mul(right.Value)
	case "/":
		if right.Value.IsZero() {
			return nil, fmt.Errorf("division by zero")
		}
		result = left.Value.Div(right.Value)
	case "%":
		if right.Value.IsZero() {
			return nil, fmt.Errorf("division by zero")
		}
		result = left.Value.Mod(right.Value)
	case "^":
		// Exponentiation
		result = left.Value.Pow(right.Value)
	default:
		return nil, fmt.Errorf("unknown operator: %s", operator)
	}

	return types.NewNumber(result), nil
}

// evalPercentageWidening applies percentage widening to a left operand.
// value + pct → value * (1 + pct), value - pct → value * (1 - pct).
// The result preserves the left operand's type.
func evalPercentageWidening(left types.Type, pct *types.Percentage, operator string) (types.Type, error) {
	one := decOne
	var multiplier decimal.Decimal
	if operator == "+" {
		multiplier = one.Add(pct.Value)
	} else {
		multiplier = one.Sub(pct.Value)
	}

	switch v := left.(type) {
	case *types.Number:
		return types.NewNumber(v.Value.Mul(multiplier)), nil
	case *types.Currency:
		return types.NewCurrency(v.Value.Mul(multiplier), v.Symbol), nil
	case *types.Quantity:
		return &types.Quantity{
			Value:      v.Value.Mul(multiplier),
			Unit:       v.Unit,
			IsNapkin:   v.IsNapkin,
			IsExplicit: v.IsExplicit,
			IsPrecise:  v.IsPrecise,
		}, nil
	case *types.Duration:
		return &types.Duration{Value: v.Value.Mul(multiplier), Unit: v.Unit}, nil
	case *types.Rate:
		return &types.Rate{
			Amount:  &types.Quantity{Value: v.Amount.Value.Mul(multiplier), Unit: v.Amount.Unit},
			PerUnit: v.PerUnit,
		}, nil
	default:
		return nil, fmt.Errorf("cannot apply percentage to %s", formatTypeForError(left))
	}
}

// evalDateDurationOperation handles date ± duration with unit-aware dispatch.
// Calendar units (year, month) use time.AddDate for correct month/year arithmetic.
// Day/week units use time.AddDate for exact day addition.
// Sub-day units (hour, minute, second, ms) use time.Add for nanosecond precision.
func evalDateDurationOperation(date *types.Date, dur *types.Duration, operator string) (types.Type, error) {
	sign := 1
	if operator == "-" {
		sign = -1
	} else if operator != "+" {
		return nil, fmt.Errorf("unsupported date-duration operation: %s", operator)
	}

	unit := strings.ToLower(dur.Unit)
	val := int(dur.Value.IntPart())

	switch unit {
	case "year", "years", "yr", "yrs":
		result := addMonthsClipped(date.Time, sign*val*12)
		if date.HasTime {
			return types.NewDateTime(result), nil
		}
		return types.NewDateFromTime(result), nil

	case "month", "months", "mo":
		result := addMonthsClipped(date.Time, sign*val)
		if date.HasTime {
			return types.NewDateTime(result), nil
		}
		return types.NewDateFromTime(result), nil

	case "week", "weeks", "wk":
		result := date.Time.AddDate(0, 0, sign*val*7)
		if date.HasTime {
			return types.NewDateTime(result), nil
		}
		return types.NewDateFromTime(result), nil

	case "day", "days":
		result := date.Time.AddDate(0, 0, sign*val)
		if date.HasTime {
			return types.NewDateTime(result), nil
		}
		return types.NewDateFromTime(result), nil

	case "hour", "hours", "hr":
		d := time.Duration(sign*val) * time.Hour
		return types.NewDateTime(date.Time.Add(d)), nil

	case "minute", "minutes", "min":
		d := time.Duration(sign*val) * time.Minute
		return types.NewDateTime(date.Time.Add(d)), nil

	case "second", "seconds", "sec":
		d := time.Duration(sign*val) * time.Second
		return types.NewDateTime(date.Time.Add(d)), nil

	case "millisecond", "milliseconds", "ms":
		d := time.Duration(sign*val) * time.Millisecond
		return types.NewDateTime(date.Time.Add(d)), nil

	default:
		// Fallback: convert to days using legacy path for unknown units
		days := durationToDays(dur)
		switch operator {
		case "+":
			return types.NewDateFromTime(date.Time.AddDate(0, 0, days)), nil
		default:
			return types.NewDateFromTime(date.Time.AddDate(0, 0, -days)), nil
		}
	}
}

// evalDateDateOperation handles date - date → duration.
func evalDateDateOperation(left, right *types.Date, operator string) (types.Type, error) {
	if operator != "-" {
		return nil, fmt.Errorf("can only subtract dates, not %s", operator)
	}

	days := left.DaysBetween(right)
	return &types.Duration{
		Value: decimal.NewFromInt(int64(days)),
		Unit:  "days",
	}, nil
}

// evalDurationOperation handles duration ± duration.
func evalDurationOperation(left, right *types.Duration, operator string) (types.Type, error) {
	// Convert both to seconds for arithmetic
	leftSec := left.ToSeconds()
	rightSec := right.ToSeconds()

	var resultSec decimal.Decimal

	switch operator {
	case "+":
		resultSec = leftSec.Add(rightSec)
	case "-":
		resultSec = leftSec.Sub(rightSec)
	default:
		return nil, fmt.Errorf("unsupported duration operation: %s", operator)
	}

	// Return in left's unit
	targetFactor := getDurationFactorDecimal(left.Unit)
	resultValue := resultSec.Div(targetFactor)

	return &types.Duration{Value: resultValue, Unit: left.Unit}, nil
}

// evalDurationNumberOperation handles duration * number or duration / number.
func evalDurationNumberOperation(dur *types.Duration, num *types.Number, operator string) (types.Type, error) {
	var result decimal.Decimal

	switch operator {
	case "*":
		result = dur.Value.Mul(num.Value)
	case "/":
		if num.Value.IsZero() {
			return nil, fmt.Errorf("division by zero")
		}
		result = dur.Value.Div(num.Value)
	default:
		return nil, fmt.Errorf("unsupported duration-number operation: %s", operator)
	}

	return &types.Duration{Value: result, Unit: dur.Unit}, nil
}

// evalUnaryOperation performs unary operations (-, +, not).
func evalUnaryOperation(operand types.Type, operator string) (types.Type, error) {
	// Handle NOT operator on Boolean first
	if operator == "not" {
		if b, ok := operand.(*types.Boolean); ok {
			return types.NewBoolean(!b.Value), nil
		}
		return nil, fmt.Errorf("'not' operator requires boolean, got %T", operand)
	}

	if frac, ok := operand.(*types.Fraction); ok {
		switch operator {
		case "-":
			neg := new(big.Rat).Neg(frac.Value)
			f := types.NewFractionFromRat(neg)
			f.Unit = frac.Unit
			return f, nil
		case "+":
			return frac, nil
		default:
			return nil, fmt.Errorf("unknown unary operator: %s", operator)
		}
	}

	if num, ok := operand.(*types.Number); ok {
		switch operator {
		case "-":
			return types.NewNumber(num.Value.Neg()), nil
		case "+":
			return num, nil
		default:
			return nil, fmt.Errorf("unknown unary operator: %s", operator)
		}
	}

	if pct, ok := operand.(*types.Percentage); ok {
		switch operator {
		case "-":
			return types.NewPercentage(pct.Value.Neg()), nil
		case "+":
			return pct, nil
		default:
			return nil, fmt.Errorf("unknown unary operator: %s", operator)
		}
	}

	if cur, ok := operand.(*types.Currency); ok {
		switch operator {
		case "-":
			return types.NewCurrency(cur.Value.Neg(), cur.Symbol), nil
		case "+":
			return cur, nil
		default:
			return nil, fmt.Errorf("unknown unary operator: %s", operator)
		}
	}

	if qty, ok := operand.(*types.Quantity); ok {
		switch operator {
		case "-":
			return &types.Quantity{Value: qty.Value.Neg(), Unit: qty.Unit}, nil
		case "+":
			return qty, nil
		default:
			return nil, fmt.Errorf("unknown unary operator: %s", operator)
		}
	}

	if dur, ok := operand.(*types.Duration); ok {
		switch operator {
		case "-":
			return &types.Duration{Value: dur.Value.Neg(), Unit: dur.Unit}, nil
		case "+":
			return dur, nil
		default:
			return nil, fmt.Errorf("unknown unary operator: %s", operator)
		}
	}

	return nil, fmt.Errorf("unsupported unary operation on %T", operand)
}

// evalComparison performs comparison operations.
func evalComparison(left, right types.Type, operator string) (types.Type, error) {
	// Fraction comparisons: convert to decimal for exact comparison
	if leftFrac, ok := left.(*types.Fraction); ok {
		return evalComparison(fractionToNumber(leftFrac), right, operator)
	}
	if rightFrac, ok := right.(*types.Fraction); ok {
		return evalComparison(left, fractionToNumber(rightFrac), operator)
	}

	// Percentage comparisons
	if leftPct, ok := left.(*types.Percentage); ok {
		if rightPct, ok := right.(*types.Percentage); ok {
			return compareNumbers(leftPct.Value, rightPct.Value, operator), nil
		}
		if rightNum, ok := right.(*types.Number); ok {
			return compareNumbers(leftPct.Value, rightNum.Value, operator), nil
		}
	}
	if leftNum, ok := left.(*types.Number); ok {
		if rightPct, ok := right.(*types.Percentage); ok {
			return compareNumbers(leftNum.Value, rightPct.Value, operator), nil
		}
	}

	// Number comparisons
	if leftNum, ok := left.(*types.Number); ok {
		if rightNum, ok := right.(*types.Number); ok {
			return compareNumbers(leftNum.Value, rightNum.Value, operator), nil
		}
	}

	// Currency comparisons (same currency only)
	if leftCur, ok := left.(*types.Currency); ok {
		if rightCur, ok := right.(*types.Currency); ok {
			if leftCur.Code != rightCur.Code {
				return nil, fmt.Errorf("cannot compare different currencies: %s and %s", leftCur.Code, rightCur.Code)
			}
			return compareNumbers(leftCur.Value, rightCur.Value, operator), nil
		}
	}

	// Boolean comparisons
	if leftBool, ok := left.(*types.Boolean); ok {
		if rightBool, ok := right.(*types.Boolean); ok {
			switch operator {
			case "==":
				return types.NewBoolean(leftBool.Value == rightBool.Value), nil
			case "!=":
				return types.NewBoolean(leftBool.Value != rightBool.Value), nil
			default:
				return nil, fmt.Errorf("unsupported boolean comparison: %s", operator)
			}
		}
	}

	// Period comparisons (v2.0): Period == Period is structural —
	// matching Start.Time AND End.Time. Period vs Date is a type
	// error (not in this branch — falls through to the generic
	// unsupported-comparison error below).
	if leftPeriod, ok := left.(*types.Period); ok {
		if rightPeriod, ok := right.(*types.Period); ok {
			switch operator {
			case "==":
				eq := leftPeriod.Start.Time.Equal(rightPeriod.Start.Time) &&
					leftPeriod.End.Time.Equal(rightPeriod.End.Time)
				return types.NewBoolean(eq), nil
			case "!=":
				eq := leftPeriod.Start.Time.Equal(rightPeriod.Start.Time) &&
					leftPeriod.End.Time.Equal(rightPeriod.End.Time)
				return types.NewBoolean(!eq), nil
			default:
				return nil, fmt.Errorf("unsupported period comparison: %s — periods only support == and !=", operator)
			}
		}
	}

	return nil, fmt.Errorf("unsupported comparison: %T %s %T", left, operator, right)
}

// compareNumbers is a helper for numeric comparisons.
func compareNumbers(left, right decimal.Decimal, operator string) *types.Boolean {
	var result bool

	switch operator {
	case ">":
		result = left.GreaterThan(right)
	case "<":
		result = left.LessThan(right)
	case ">=":
		result = left.GreaterThanOrEqual(right)
	case "<=":
		result = left.LessThanOrEqual(right)
	case "==":
		result = left.Equal(right)
	case "!=":
		result = !left.Equal(right)
	}

	return types.NewBoolean(result)
}

// Helper functions

// addMonthsClipped adds months to a time, clipping to the last day of the target
// month when the source day exceeds it. This fixes Go's time.AddDate behavior where
// Jan 31 + 1 month = March 3 (via Feb 31 normalization). Instead, we get Feb 28/29.
func addMonthsClipped(t time.Time, months int) time.Time {
	y, m, d := t.Date()
	targetMonth := time.Month(int(m) + months)
	// Normalize year/month
	targetYear := y + int(targetMonth-1)/12
	targetMonth = (targetMonth-1)%12 + 1
	if targetMonth <= 0 {
		targetMonth += 12
		targetYear--
	}

	// Find last day of target month
	lastDay := time.Date(targetYear, targetMonth+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if d > lastDay {
		d = lastDay
	}

	return time.Date(targetYear, targetMonth, d, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}

func durationToDays(dur *types.Duration) int {
	factor := getDurationFactorDecimal(dur.Unit)
	seconds := dur.Value.Mul(factor)
	days := seconds.Div(decimal.NewFromInt(86400)) // seconds per day
	return int(days.IntPart())
}

// durationFactors maps duration unit names to their conversion factor in seconds.
// Package-level to avoid allocating on every duration operation.
var durationFactors = map[string]decimal.Decimal{
	"millisecond": decimal.NewFromFloat(0.001), "milliseconds": decimal.NewFromFloat(0.001),
	"second": decimal.NewFromInt(1), "seconds": decimal.NewFromInt(1),
	"minute": decimal.NewFromInt(60), "minutes": decimal.NewFromInt(60),
	"hour": decimal.NewFromInt(3600), "hours": decimal.NewFromInt(3600),
	"day": decimal.NewFromInt(86400), "days": decimal.NewFromInt(86400),
	"week": decimal.NewFromInt(604800), "weeks": decimal.NewFromInt(604800),
	"month": decimal.NewFromInt(2592000), "months": decimal.NewFromInt(2592000), // 30 days
	"year": decimal.NewFromInt(31536000), "years": decimal.NewFromInt(31536000), // 365 days
}

// getDurationFactorDecimal returns the conversion factor to seconds for a duration unit.
func getDurationFactorDecimal(unit string) decimal.Decimal {
	if f, ok := durationFactors[unit]; ok {
		return f
	}
	return decimal.NewFromInt(1) // fallback to seconds
}

// unsupportedOperationError provides helpful error messages for common type mismatches.
func unsupportedOperationError(left, right types.Type, operator string) error {
	// Quantity * Duration or Duration * Quantity → suggest rate syntax
	_, leftIsQty := left.(*types.Quantity)
	_, rightIsQty := right.(*types.Quantity)
	_, leftIsDur := left.(*types.Duration)
	_, rightIsDur := right.(*types.Duration)

	if (leftIsQty && rightIsDur) || (leftIsDur && rightIsQty) {
		var qty *types.Quantity
		var dur *types.Duration
		if leftIsQty {
			qty = left.(*types.Quantity)
			dur = right.(*types.Duration)
		} else {
			qty = right.(*types.Quantity)
			dur = left.(*types.Duration)
		}

		// Suggest: "100k users * 1 month" → "100k users/month" or "RATE over 1 month"
		return fmt.Errorf("cannot multiply %s by %s directly\n"+
			"  Hint: To create a rate, use: %v %s/%s\n"+
			"  Hint: To accumulate a rate over time, use: RATE over %s",
			formatTypeForError(left), formatTypeForError(right),
			qty.Value, qty.Unit, dur.Unit,
			dur.String())
	}

	// Generic fallback
	return fmt.Errorf("cannot %s %s and %s",
		operatorVerb(operator), formatTypeForError(left), formatTypeForError(right))
}

// formatTypeForError returns a user-friendly description of a type.
func formatTypeForError(t types.Type) string {
	switch v := t.(type) {
	case *types.Number:
		return fmt.Sprintf("number (%s)", v.String())
	case *types.Quantity:
		return fmt.Sprintf("quantity (%s)", v.String())
	case *types.Duration:
		return fmt.Sprintf("duration (%s)", v.String())
	case *types.Rate:
		return fmt.Sprintf("rate (%s)", v.String())
	case *types.Currency:
		return fmt.Sprintf("currency (%s)", v.String())
	case *types.Date:
		return fmt.Sprintf("date (%s)", v.String())
	case *types.Boolean:
		return fmt.Sprintf("boolean (%s)", v.String())
	case *types.Percentage:
		return fmt.Sprintf("percentage (%s)", v.String())
	default:
		return fmt.Sprintf("%T", t)
	}
}

// operatorVerb returns a verb for the operator.
func operatorVerb(op string) string {
	switch op {
	case "+":
		return "add"
	case "-":
		return "subtract"
	case "*":
		return "multiply"
	case "/":
		return "divide"
	case "%":
		return "modulo"
	case "^":
		return "exponentiate"
	default:
		return "operate on"
	}
}
