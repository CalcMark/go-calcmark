// Package document provides document structure and parsing for CalcMark.
package document

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

// evalLiteral evaluates a literal AST node to produce a typed value.
// This is a lightweight evaluator for frontmatter globals that doesn't
// require the full interpreter.
func evalLiteral(node ast.Node) (types.Type, error) {
	switch n := node.(type) {
	case *ast.Expression:
		return evalLiteral(n.Expr)

	case *ast.NumberLiteral:
		return evalNumberLiteral(n)

	case *ast.QuantityLiteral:
		return evalQuantityLiteral(n)

	case *ast.CurrencyLiteral:
		return evalCurrencyLiteral(n)

	case *ast.DateLiteral:
		return evalDateLiteral(n)

	case *ast.RelativeDateLiteral:
		return evalRelativeDateLiteral(n)

	case *ast.DurationLiteral:
		return evalDurationLiteral(n)

	case *ast.TimeLiteral:
		return evalTimeLiteral(n)

	case *ast.BooleanLiteral:
		return evalBooleanLiteral(n)

	case *ast.RateLiteral:
		return evalRateLiteral(n)

	default:
		return nil, fmt.Errorf("unsupported literal type: %T", node)
	}
}

func evalNumberLiteral(n *ast.NumberLiteral) (types.Type, error) {
	value, err := expandNumber(n.Value)
	if err != nil {
		return nil, err
	}
	return types.NewNumber(value), nil
}

func evalQuantityLiteral(n *ast.QuantityLiteral) (types.Type, error) {
	value, err := expandNumber(n.Value)
	if err != nil {
		return nil, err
	}
	return types.NewQuantity(value, n.Unit), nil
}

func evalCurrencyLiteral(n *ast.CurrencyLiteral) (types.Type, error) {
	value, err := expandNumber(n.Value)
	if err != nil {
		return nil, err
	}
	return types.NewCurrency(value, n.Symbol), nil
}

func evalDateLiteral(n *ast.DateLiteral) (types.Type, error) {
	// Convert canonical month name to number (lexer already normalized abbreviations)
	month, err := canonicalMonthToNumber(n.Month)
	if err != nil {
		return nil, err
	}

	// Parse day
	day, err := strconv.Atoi(n.Day)
	if err != nil {
		return nil, fmt.Errorf("invalid day: %s", n.Day)
	}

	// Parse year (0 means current year, handled by NewDate)
	year := 0
	if n.Year != nil {
		year, err = strconv.Atoi(*n.Year)
		if err != nil {
			return nil, fmt.Errorf("invalid year: %s", *n.Year)
		}
	}

	return types.NewDate(year, month, day)
}

func evalRelativeDateLiteral(n *ast.RelativeDateLiteral) (types.Type, error) {
	now := time.Now()
	var t time.Time

	switch n.Keyword {
	case "today":
		t = now
	case "tomorrow":
		t = now.AddDate(0, 0, 1)
	case "yesterday":
		t = now.AddDate(0, 0, -1)
	default:
		return nil, fmt.Errorf("unknown relative date keyword: %s", n.Keyword)
	}

	return types.NewDateFromTime(t), nil
}

func evalDurationLiteral(n *ast.DurationLiteral) (types.Type, error) {
	value, err := expandNumber(n.Value)
	if err != nil {
		return nil, err
	}
	return types.NewDuration(value, n.Unit)
}

func evalTimeLiteral(n *ast.TimeLiteral) (types.Type, error) {
	// Parse hour
	hour, err := strconv.Atoi(n.Hour)
	if err != nil {
		return nil, fmt.Errorf("invalid hour: %s", n.Hour)
	}

	// Parse minute
	minute, err := strconv.Atoi(n.Minute)
	if err != nil {
		return nil, fmt.Errorf("invalid minute: %s", n.Minute)
	}

	// Parse second (optional)
	second := -1
	if n.Second != nil {
		second, err = strconv.Atoi(*n.Second)
		if err != nil {
			return nil, fmt.Errorf("invalid second: %s", *n.Second)
		}
	}

	// Determine if PM
	isPM := false
	if n.Period != nil && strings.ToUpper(*n.Period) == "PM" {
		isPM = true
	}

	// Parse UTC offset (optional)
	utcOffsetMinutes := 0
	if n.UTCOffset != nil {
		offsetHours, err := strconv.Atoi(n.UTCOffset.Hours)
		if err != nil {
			return nil, fmt.Errorf("invalid UTC offset hours: %s", n.UTCOffset.Hours)
		}
		offsetMinutes := 0
		if n.UTCOffset.Minutes != nil {
			offsetMinutes, err = strconv.Atoi(*n.UTCOffset.Minutes)
			if err != nil {
				return nil, fmt.Errorf("invalid UTC offset minutes: %s", *n.UTCOffset.Minutes)
			}
		}
		utcOffsetMinutes = offsetHours*60 + offsetMinutes
		if n.UTCOffset.Sign == "-" {
			utcOffsetMinutes = -utcOffsetMinutes
		}
	}

	return types.NewTime(hour, minute, second, isPM, utcOffsetMinutes)
}

func evalBooleanLiteral(n *ast.BooleanLiteral) (types.Type, error) {
	val := strings.ToLower(n.Value) == "true"
	return types.NewBoolean(val), nil
}

func evalRateLiteral(n *ast.RateLiteral) (types.Type, error) {
	// Evaluate the amount (which should be a literal)
	amountVal, err := evalLiteral(n.Amount)
	if err != nil {
		return nil, fmt.Errorf("rate amount evaluation failed: %w", err)
	}

	// Convert amount to Quantity
	var amountQty *types.Quantity
	switch v := amountVal.(type) {
	case *types.Quantity:
		amountQty = v
	case *types.Number:
		amountQty = &types.Quantity{
			Value: v.Value,
			Unit:  "",
		}
	case *types.Currency:
		amountQty = &types.Quantity{
			Value: v.Value,
			Unit:  v.Symbol,
		}
	default:
		return nil, fmt.Errorf("rate amount must be a number, quantity, or currency, got %T", amountVal)
	}

	return types.NewRate(amountQty, n.PerUnit), nil
}

// expandNumber handles number suffixes like K, M, B, T and percentages.
func expandNumber(s string) (decimal.Decimal, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return decimal.Zero, nil
	}

	// Check for percentage
	if strings.HasSuffix(s, "%") {
		numStr := strings.TrimSuffix(s, "%")
		val, err := decimal.NewFromString(numStr)
		if err != nil {
			return decimal.Zero, err
		}
		return val.Div(decimal.NewFromInt(100)), nil
	}

	// Check for multiplier suffixes
	multiplier := decimal.NewFromInt(1)
	lastChar := s[len(s)-1]

	switch lastChar {
	case 'K', 'k':
		multiplier = decimal.NewFromInt(1000)
		s = s[:len(s)-1]
	case 'M', 'm':
		multiplier = decimal.NewFromInt(1_000_000)
		s = s[:len(s)-1]
	case 'B', 'b':
		multiplier = decimal.NewFromInt(1_000_000_000)
		s = s[:len(s)-1]
	case 'T', 't':
		multiplier = decimal.NewFromInt(1_000_000_000_000)
		s = s[:len(s)-1]
	}

	val, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero, err
	}

	return val.Mul(multiplier), nil
}

// canonicalMonthToNumber converts a canonical month name (from lexer.MonthNames)
// to a month number (1-12). The lexer normalizes abbreviations (e.g., "dec" -> "December")
// so we only need to handle the canonical forms here.
func canonicalMonthToNumber(name string) (int, error) {
	// Canonical month names as defined in spec/lexer/date_keywords.go
	months := map[string]int{
		"January":   1,
		"February":  2,
		"March":     3,
		"April":     4,
		"May":       5,
		"June":      6,
		"July":      7,
		"August":    8,
		"September": 9,
		"October":   10,
		"November":  11,
		"December":  12,
	}

	month, ok := months[name]
	if !ok {
		// Fall back to case-insensitive for robustness
		for canonical, num := range months {
			if strings.EqualFold(name, canonical) {
				return num, nil
			}
		}
		return 0, fmt.Errorf("invalid month name: %s", name)
	}
	return month, nil
}
