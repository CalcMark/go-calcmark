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

	year := interp.now().Year()
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
	now := interp.now()
	keyword := strings.ToLower(r.Keyword)

	switch keyword {
	case "today", "now":
		return types.NewDateFromTime(now), nil
	case "tomorrow":
		return types.NewDateFromTime(now.AddDate(0, 0, 1)), nil
	case "yesterday":
		return types.NewDateFromTime(now.AddDate(0, 0, -1)), nil
	default:
		// Try weekday expressions: "friday", "this friday", "next friday", "last friday"
		if d, ok := interp.resolveWeekdayExpression(keyword, now); ok {
			return d, nil
		}
		return nil, fmt.Errorf("unknown relative date keyword: %q", r.Keyword)
	}
}

// weekdayNames maps lowercase weekday names to Go's time.Weekday.
var weekdayNames = map[string]time.Weekday{
	"monday": time.Monday, "tuesday": time.Tuesday, "wednesday": time.Wednesday,
	"thursday": time.Thursday, "friday": time.Friday, "saturday": time.Saturday,
	"sunday": time.Sunday,
}

// resolveWeekdayExpression handles "friday", "this friday", "next friday", "last friday".
func (interp *Interpreter) resolveWeekdayExpression(keyword string, now time.Time) (*types.Date, bool) {
	// Parse modifier and weekday name
	modifier := "this" // bare weekday = "this <weekday>"
	weekdayStr := keyword

	for _, prefix := range []string{"this ", "next ", "last "} {
		if strings.HasPrefix(keyword, prefix) {
			modifier = strings.TrimSpace(prefix)
			weekdayStr = keyword[len(prefix):]
			break
		}
	}

	target, ok := weekdayNames[weekdayStr]
	if !ok {
		return nil, false
	}

	current := now.Weekday()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	switch modifier {
	case "this":
		// This <weekday> = the occurrence in the current calendar week (Mon-Sun).
		// Calculate offset from Monday of current week.
		daysFromMonday := (int(current) - int(time.Monday) + 7) % 7
		monday := today.AddDate(0, 0, -daysFromMonday)
		daysToTarget := (int(target) - int(time.Monday) + 7) % 7
		return types.NewDateFromTime(monday.AddDate(0, 0, daysToTarget)), true

	case "next":
		// Next <weekday> = soonest future occurrence. If today IS that day, skip to next week.
		diff := (int(target) - int(current) + 7) % 7
		if diff == 0 {
			diff = 7 // today is that weekday — skip to next week
		}
		return types.NewDateFromTime(today.AddDate(0, 0, diff)), true

	case "last":
		// Last <weekday> = most recent past occurrence. If today IS that day, go to last week.
		diff := (int(current) - int(target) + 7) % 7
		if diff == 0 {
			diff = 7 // today is that weekday — go to last week
		}
		return types.NewDateFromTime(today.AddDate(0, 0, -diff)), true
	}

	return nil, false
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
