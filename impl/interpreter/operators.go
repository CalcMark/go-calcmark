package interpreter

import (
	"fmt"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/types"
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
	// Number operations
	if leftNum, ok := left.(*types.Number); ok {
		if rightNum, ok := right.(*types.Number); ok {
			// Check if right is a percentage (value < 1.0 and originally had %)
			// For now, we handle percentage operations specially
			// 100 + 20% -> 100 + (100 * 0.20) = 120
			// 100 - 20% -> 100 - (100 * 0.20) = 80

			// Note: We can't distinguish if rightNum came from a % literal
			// So we'll handle this in a special case if needed
			return evalNumberOperation(leftNum, rightNum, operator)
		}
		// Number * Currency → Currency
		if rightCur, ok := right.(*types.Currency); ok && operator == "*" {
			result := leftNum.Value.Mul(rightCur.Value)
			return types.NewCurrency(result, rightCur.Symbol), nil
		}
		// Number * Duration → Duration
		if rightDur, ok := right.(*types.Duration); ok && operator == "*" {
			result := leftNum.Value.Mul(rightDur.Value)
			return &types.Duration{Value: result, Unit: rightDur.Unit}, nil
		}
	}

	// Currency operations
	if leftCur, ok := left.(*types.Currency); ok {
		// Currency * Number → Currency
		if rightNum, ok := right.(*types.Number); ok && operator == "*" {
			result := leftCur.Value.Mul(rightNum.Value)
			return types.NewCurrency(result, leftCur.Symbol), nil
		}
		// Currency op Currency (same type)
		if rightCur, ok := right.(*types.Currency); ok {
			if leftCur.Symbol != rightCur.Symbol {
				return nil, fmt.Errorf("cannot %s different currencies: %s and %s",
					operator, leftCur.Symbol, rightCur.Symbol)
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

	// Date operations
	if leftDate, ok := left.(*types.Date); ok {
		if rightDur, ok := right.(*types.Duration); ok {
			return evalDateDurationOperation(leftDate, rightDur, operator)
		}
		if rightDate, ok := right.(*types.Date); ok {
			return evalDateDateOperation(leftDate, rightDate, operator)
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

	// Quantity operations (with unit conversion - USER REQUIREMENT: first-unit-wins)
	if leftQty, ok := left.(*types.Quantity); ok {
		if rightQty, ok := right.(*types.Quantity); ok {
			return evalQuantityOperation(leftQty, rightQty, operator)
		}
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

	return nil, fmt.Errorf("unsupported operation: %T %s %T", left, operator, right)
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

// evalCurrencyOperation performs operations on two currencies.
func evalCurrencyOperation(left, right *types.Currency, operator string) (types.Type, error) {
	// Check if same currency
	if left.Code != right.Code {
		return nil, fmt.Errorf("cannot %s different currencies: %s and %s", operator, left.Code, right.Code)
	}

	var result decimal.Decimal

	switch operator {
	case "+":
		result = left.Value.Add(right.Value)
	case "-":
		result = left.Value.Sub(right.Value)
	default:
		return nil, fmt.Errorf("unsupported currency operation: %s", operator)
	}

	return types.NewCurrency(result, left.Symbol), nil
}

// evalCurrencyNumberOperation handles currency + number operations (relaxed rules).
func evalCurrencyNumberOperation(cur *types.Currency, num *types.Number, operator string) (types.Type, error) {
	var result decimal.Decimal

	switch operator {
	case "+":
		result = cur.Value.Add(num.Value)
	case "-":
		result = cur.Value.Sub(num.Value)
	case "*":
		result = cur.Value.Mul(num.Value)
	case "/":
		if num.Value.IsZero() {
			return nil, fmt.Errorf("division by zero")
		}
		result = cur.Value.Div(num.Value)
	default:
		return nil, fmt.Errorf("unsupported currency-number operation: %s", operator)
	}

	return types.NewCurrency(result, cur.Symbol), nil
}

// evalDateDurationOperation handles date ± duration.
func evalDateDurationOperation(date *types.Date, dur *types.Duration, operator string) (types.Type, error) {
	// Convert duration to days (approximate for non-day units)
	days := durationToDays(dur)

	switch operator {
	case "+":
		return types.NewDateFromTime(date.Time.AddDate(0, 0, days)), nil
	case "-":
		return types.NewDateFromTime(date.Time.AddDate(0, 0, -days)), nil
	default:
		return nil, fmt.Errorf("unsupported date-duration operation: %s", operator)
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
	targetFactor := decimal.NewFromInt(getDurationFactor(left.Unit))
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

// evalUnaryOperation performs unary operations (-, +).
func evalUnaryOperation(operand types.Type, operator string) (types.Type, error) {
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

	return nil, fmt.Errorf("unsupported unary operation on %T", operand)
}

// evalComparison performs comparison operations.
func evalComparison(left, right types.Type, operator string) (types.Type, error) {
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

func durationToDays(dur *types.Duration) int {
	factor := getDurationFactor(dur.Unit)
	seconds := dur.Value.Mul(decimal.NewFromInt(factor))
	days := seconds.Div(decimal.NewFromInt(86400)) // seconds per day
	return int(days.IntPart())
}

func getDurationFactor(unit string) int64 {
	factors := map[string]int64{
		"second": 1, "seconds": 1,
		"minute": 60, "minutes": 60,
		"hour": 3600, "hours": 3600,
		"day": 86400, "days": 86400,
		"week": 604800, "weeks": 604800,
		"month": 2592000, "months": 2592000, // 30 days
		"year": 31536000, "years": 31536000, // 365 days
	}
	return factors[unit]
}
