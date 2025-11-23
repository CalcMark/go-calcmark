package interpreter

import (
	"fmt"
	"strings"
	"time"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/types"
)

// Date and time literal evaluation.

func (interp *Interpreter) evalDateLiteral(d *ast.DateLiteral) (types.Type, error) {
	month, err := parseMonth(d.Month)
	if err != nil {
		return nil, err
	}

	day, err := parseInt(d.Day)
	if err != nil {
		return nil, fmt.Errorf("invalid day: %w", err)
	}

	year := 0
	if d.Year != nil {
		year, err = parseInt(*d.Year)
		if err != nil {
			return nil, fmt.Errorf("invalid year: %w", err)
		}
	}

	return types.NewDate(year, month, day)
}

func (interp *Interpreter) evalTimeLiteral(t *ast.TimeLiteral) (types.Type, error) {
	hour, err := parseInt(t.Hour)
	if err != nil {
		return nil, fmt.Errorf("invalid hour: %w", err)
	}

	minute, err := parseInt(t.Minute)
	if err != nil {
		return nil, fmt.Errorf("invalid minute: %w", err)
	}

	second := -1
	if t.Second != nil {
		second, err = parseInt(*t.Second)
		if err != nil {
			return nil, fmt.Errorf("invalid second: %w", err)
		}
	}

	isPM := false
	if t.Period != nil && strings.ToUpper(*t.Period) == "PM" {
		isPM = true
	}

	utcOffsetMinutes := 0
	if t.UTCOffset != nil {
		utcOffsetMinutes, err = parseUTCOffset(t.UTCOffset)
		if err != nil {
			return nil, err
		}
	}

	return types.NewTime(hour, minute, second, isPM, utcOffsetMinutes)
}

func (interp *Interpreter) evalDurationLiteral(d *ast.DurationLiteral) (types.Type, error) {
	return types.NewDurationFromString(d.Value, d.Unit)
}

func (interp *Interpreter) evalRelativeDateLiteral(r *ast.RelativeDateLiteral) (types.Type, error) {
	now := time.Now()
	keyword := strings.ToLower(r.Keyword)

	switch keyword {
	case "today", "now":
		return types.NewDateFromTime(now), nil
	case "tomorrow":
		return types.NewDateFromTime(now.AddDate(0, 0, 1)), nil
	case "yesterday":
		return types.NewDateFromTime(now.AddDate(0, 0, -1)), nil
	default:
		return nil, fmt.Errorf("unknown relative date keyword: %q", r.Keyword)
	}
}

// parseUTCOffset converts AST UTC offset to minutes.
func parseUTCOffset(offset *ast.UTCOffset) (int, error) {
	hours, err := parseInt(offset.Hours)
	if err != nil {
		return 0, err
	}

	minutes := 0
	if offset.Minutes != nil {
		minutes, err = parseInt(*offset.Minutes)
		if err != nil {
			return 0, err
		}
	}

	totalMinutes := hours*60 + minutes

	if offset.Sign == "-" {
		totalMinutes = -totalMinutes
	}

	return totalMinutes, nil
}
