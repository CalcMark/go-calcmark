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
	case "today":
		return types.NewDateFromTime(now), nil
	case "now":
		// "now" preserves full time precision (used by "ago" desugaring for sub-day accuracy).
		// "today" normalizes to midnight (date-only).
		return types.NewDateTime(now), nil
	case "tomorrow":
		return types.NewDateFromTime(now.AddDate(0, 0, 1)), nil
	case "yesterday":
		return types.NewDateFromTime(now.AddDate(0, 0, -1)), nil
	// Period expressions: this/next/last week/month/year
	case "this week":
		// Monday of current week
		daysFromMonday := (int(now.Weekday()) - int(time.Monday) + 7) % 7
		return types.NewDateFromTime(now.AddDate(0, 0, -daysFromMonday)), nil
	case "next week":
		daysFromMonday := (int(now.Weekday()) - int(time.Monday) + 7) % 7
		monday := now.AddDate(0, 0, -daysFromMonday)
		return types.NewDateFromTime(monday.AddDate(0, 0, 7)), nil
	case "last week":
		daysFromMonday := (int(now.Weekday()) - int(time.Monday) + 7) % 7
		monday := now.AddDate(0, 0, -daysFromMonday)
		return types.NewDateFromTime(monday.AddDate(0, 0, -7)), nil
	case "this month":
		return types.NewDateFromTime(time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)), nil
	case "next month":
		return types.NewDateFromTime(time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC)), nil
	case "last month":
		return types.NewDateFromTime(time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, time.UTC)), nil
	case "this year":
		return types.NewDateFromTime(time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)), nil
	case "next year":
		return types.NewDateFromTime(time.Date(now.Year()+1, 1, 1, 0, 0, 0, 0, time.UTC)), nil
	case "last year":
		return types.NewDateFromTime(time.Date(now.Year()-1, 1, 1, 0, 0, 0, 0, time.UTC)), nil

	// Fiscal expressions (require fiscal_year_starts)
	case "this fiscal quarter", "next fiscal quarter", "last fiscal quarter",
		"this fiscal year", "next fiscal year", "last fiscal year":
		return interp.evalFiscalExpression(keyword, now)

	// Calendar quarter expressions
	case "this quarter":
		q := calendarQuarterStart(now.Month())
		return types.NewDateFromTime(time.Date(now.Year(), q, 1, 0, 0, 0, 0, time.UTC)), nil
	case "next quarter":
		q := calendarQuarterStart(now.Month())
		t := time.Date(now.Year(), q+3, 1, 0, 0, 0, 0, time.UTC) // Go normalizes month > 12
		return types.NewDateFromTime(t), nil
	case "last quarter":
		q := calendarQuarterStart(now.Month())
		t := time.Date(now.Year(), q-3, 1, 0, 0, 0, 0, time.UTC) // Go normalizes month < 1
		return types.NewDateFromTime(t), nil

	default:
		// "start of <period>" — strip prefix and evaluate the inner period
		if strings.HasPrefix(keyword, "start of ") {
			inner := keyword[len("start of "):]
			innerNode := &ast.RelativeDateLiteral{Keyword: inner, SourceText: inner}
			return interp.evalRelativeDateLiteral(innerNode)
		}

		// "end of <period>" — evaluate inner period to get start, then find last day
		if strings.HasPrefix(keyword, "end of ") {
			inner := keyword[len("end of "):]
			return interp.evalEndOf(inner, now)
		}

		// Try weekday expressions: "friday", "this friday", "next friday", "last friday"
		if d, ok := interp.resolveWeekdayExpression(keyword, now); ok {
			return d, nil
		}
		// Try month expressions: "this april", "next april", "last april"
		if d, ok := resolveMonthExpression(keyword, now); ok {
			return d, nil
		}
		return nil, fmt.Errorf("unknown relative date keyword: %q", r.Keyword)
	}
}

// evalEndOf resolves "end of <period>" to the last day of that period.
// Strategy: find the start of the NEXT period, then subtract 1 day.
func (interp *Interpreter) evalEndOf(innerKeyword string, now time.Time) (types.Type, error) {
	// Map the period to its "next" equivalent to find the boundary
	nextPeriod := ""
	switch innerKeyword {
	case "this week":
		nextPeriod = "next week"
	case "next week":
		// end of next week = start of the week after next - 1 day
		daysFromMonday := (int(now.Weekday()) - int(time.Monday) + 7) % 7
		monday := now.AddDate(0, 0, -daysFromMonday)
		startOfWeekAfterNext := monday.AddDate(0, 0, 14)
		return types.NewDateFromTime(startOfWeekAfterNext.AddDate(0, 0, -1)), nil
	case "last week":
		nextPeriod = "this week"
	case "this month":
		nextPeriod = "next month"
	case "next month":
		t := time.Date(now.Year(), now.Month()+2, 1, 0, 0, 0, 0, time.UTC)
		return types.NewDateFromTime(t.AddDate(0, 0, -1)), nil
	case "last month":
		nextPeriod = "this month"
	case "this year":
		return types.NewDateFromTime(time.Date(now.Year(), 12, 31, 0, 0, 0, 0, time.UTC)), nil
	case "next year":
		return types.NewDateFromTime(time.Date(now.Year()+1, 12, 31, 0, 0, 0, 0, time.UTC)), nil
	case "last year":
		return types.NewDateFromTime(time.Date(now.Year()-1, 12, 31, 0, 0, 0, 0, time.UTC)), nil
	case "this quarter":
		q := calendarQuarterStart(now.Month())
		nextQ := time.Date(now.Year(), q+3, 1, 0, 0, 0, 0, time.UTC)
		return types.NewDateFromTime(nextQ.AddDate(0, 0, -1)), nil
	case "next quarter":
		q := calendarQuarterStart(now.Month())
		nextNextQ := time.Date(now.Year(), q+6, 1, 0, 0, 0, 0, time.UTC)
		return types.NewDateFromTime(nextNextQ.AddDate(0, 0, -1)), nil
	case "last quarter":
		q := calendarQuarterStart(now.Month())
		thisQ := time.Date(now.Year(), q, 1, 0, 0, 0, 0, time.UTC)
		return types.NewDateFromTime(thisQ.AddDate(0, 0, -1)), nil
	case "this fiscal quarter", "next fiscal quarter", "last fiscal quarter",
		"this fiscal year", "next fiscal year", "last fiscal year":
		return interp.evalEndOfFiscal(innerKeyword, now)
	default:
		// Try named months: "end of january", "end of next april"
		if d, ok := resolveEndOfMonth(innerKeyword, now); ok {
			return d, nil
		}
		return nil, fmt.Errorf("'end of' not supported for: %q", innerKeyword)
	}

	// Generic path: evaluate the next period and subtract 1 day
	nextNode := &ast.RelativeDateLiteral{Keyword: nextPeriod, SourceText: nextPeriod}
	nextStart, err := interp.evalRelativeDateLiteral(nextNode)
	if err != nil {
		return nil, err
	}
	nextDate := nextStart.(*types.Date)
	return types.NewDateFromTime(nextDate.Time.AddDate(0, 0, -1)), nil
}

// evalEndOfFiscal handles "end of this/next/last fiscal quarter/year".
func (interp *Interpreter) evalEndOfFiscal(keyword string, now time.Time) (types.Type, error) {
	if interp.fiscalYearStarts == nil {
		return nil, fmt.Errorf("fiscal expressions require a 'fiscal_year_starts' frontmatter key")
	}

	fyStart := time.Month(*interp.fiscalYearStarts)

	switch keyword {
	case "this fiscal quarter":
		y, m := fiscalQuarterStart(now.Year(), now.Month(), fyStart)
		nextFQ := time.Date(y, m+3, 1, 0, 0, 0, 0, time.UTC)
		return types.NewDateFromTime(nextFQ.AddDate(0, 0, -1)), nil
	case "next fiscal quarter":
		y, m := fiscalQuarterStart(now.Year(), now.Month(), fyStart)
		nextNextFQ := time.Date(y, m+6, 1, 0, 0, 0, 0, time.UTC)
		return types.NewDateFromTime(nextNextFQ.AddDate(0, 0, -1)), nil
	case "last fiscal quarter":
		y, m := fiscalQuarterStart(now.Year(), now.Month(), fyStart)
		thisFQ := time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)
		return types.NewDateFromTime(thisFQ.AddDate(0, 0, -1)), nil
	case "this fiscal year":
		fy := fiscalYear(now.Year(), now.Month(), fyStart)
		nextFYStart := time.Date(fy+1, fyStart, 1, 0, 0, 0, 0, time.UTC)
		return types.NewDateFromTime(nextFYStart.AddDate(0, 0, -1)), nil
	case "next fiscal year":
		fy := fiscalYear(now.Year(), now.Month(), fyStart) + 1
		nextFYStart := time.Date(fy+1, fyStart, 1, 0, 0, 0, 0, time.UTC)
		return types.NewDateFromTime(nextFYStart.AddDate(0, 0, -1)), nil
	case "last fiscal year":
		fy := fiscalYear(now.Year(), now.Month(), fyStart) - 1
		nextFYStart := time.Date(fy+1, fyStart, 1, 0, 0, 0, 0, time.UTC)
		return types.NewDateFromTime(nextFYStart.AddDate(0, 0, -1)), nil
	}
	return nil, fmt.Errorf("unknown fiscal end-of expression: %q", keyword)
}

// resolveEndOfMonth handles "end of january", "end of next april", etc.
func resolveEndOfMonth(keyword string, now time.Time) (*types.Date, bool) {
	// First resolve the month start, then find the last day
	startDate, ok := resolveMonthExpression(keyword, now)
	if !ok {
		return nil, false
	}
	// Last day of the month = first day of next month - 1 day
	lastDay := time.Date(startDate.Time.Year(), startDate.Time.Month()+1, 0, 0, 0, 0, 0, time.UTC)
	return types.NewDateFromTime(lastDay), true
}

// evalFiscalExpression evaluates fiscal quarter and fiscal year expressions.
// Requires fiscal_year_starts to be configured via frontmatter.
func (interp *Interpreter) evalFiscalExpression(keyword string, now time.Time) (types.Type, error) {
	if interp.fiscalYearStarts == nil {
		return nil, fmt.Errorf("fiscal expressions require a 'fiscal_year_starts' frontmatter key")
	}

	fyStart := time.Month(*interp.fiscalYearStarts)

	switch keyword {
	case "this fiscal quarter":
		y, m := fiscalQuarterStart(now.Year(), now.Month(), fyStart)
		return types.NewDateFromTime(time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)), nil
	case "next fiscal quarter":
		y, m := fiscalQuarterStart(now.Year(), now.Month(), fyStart)
		t := time.Date(y, m+3, 1, 0, 0, 0, 0, time.UTC) // Go normalizes
		return types.NewDateFromTime(t), nil
	case "last fiscal quarter":
		y, m := fiscalQuarterStart(now.Year(), now.Month(), fyStart)
		t := time.Date(y, m-3, 1, 0, 0, 0, 0, time.UTC) // Go normalizes
		return types.NewDateFromTime(t), nil
	case "this fiscal year":
		y := fiscalYear(now.Year(), now.Month(), fyStart)
		return types.NewDateFromTime(time.Date(y, fyStart, 1, 0, 0, 0, 0, time.UTC)), nil
	case "next fiscal year":
		y := fiscalYear(now.Year(), now.Month(), fyStart) + 1
		return types.NewDateFromTime(time.Date(y, fyStart, 1, 0, 0, 0, 0, time.UTC)), nil
	case "last fiscal year":
		y := fiscalYear(now.Year(), now.Month(), fyStart) - 1
		return types.NewDateFromTime(time.Date(y, fyStart, 1, 0, 0, 0, 0, time.UTC)), nil
	default:
		return nil, fmt.Errorf("unknown fiscal expression: %q", keyword)
	}
}

// fiscalYear returns the calendar year in which the fiscal year begins.
// For a FY starting in July, August 2026 is in the FY that started July 2026.
// For a FY starting in July, May 2026 is in the FY that started July 2025.
func fiscalYear(calYear int, calMonth, fyStartMonth time.Month) int {
	if calMonth >= fyStartMonth {
		return calYear
	}
	return calYear - 1
}

// fiscalQuarterStart returns the year and month of the first day of the
// current fiscal quarter.
func fiscalQuarterStart(calYear int, calMonth, fyStartMonth time.Month) (int, time.Month) {
	// Calculate months since fiscal year start
	fy := fiscalYear(calYear, calMonth, fyStartMonth)
	monthsSinceFYStart := (calYear-fy)*12 + int(calMonth) - int(fyStartMonth)
	if monthsSinceFYStart < 0 {
		monthsSinceFYStart += 12
	}
	quarterIndex := monthsSinceFYStart / 3
	quarterStartMonth := time.Month(int(fyStartMonth) + quarterIndex*3)
	quarterStartYear := fy
	// Normalize if month > 12
	for quarterStartMonth > 12 {
		quarterStartMonth -= 12
		quarterStartYear++
	}
	return quarterStartYear, quarterStartMonth
}

// calendarQuarterStart returns the first month of the calendar quarter containing m.
// Q1=Jan, Q2=Apr, Q3=Jul, Q4=Oct.
func calendarQuarterStart(m time.Month) time.Month {
	return time.Month((int(m)-1)/3*3 + 1)
}

// canonicalMonths maps lowercase month names/abbreviations to time.Month.
var canonicalMonths = map[string]time.Month{
	"january": time.January, "jan": time.January,
	"february": time.February, "feb": time.February,
	"march": time.March, "mar": time.March,
	"april": time.April, "apr": time.April,
	"may": time.May,
	"june": time.June, "jun": time.June,
	"july": time.July, "jul": time.July,
	"august": time.August, "aug": time.August,
	"september": time.September, "sep": time.September, "sept": time.September,
	"october": time.October, "oct": time.October,
	"november": time.November, "nov": time.November,
	"december": time.December, "dec": time.December,
}

// resolveMonthExpression handles "this april", "next april", "last april".
func resolveMonthExpression(keyword string, now time.Time) (*types.Date, bool) {
	modifier := ""
	monthStr := keyword

	for _, prefix := range []string{"this ", "next ", "last "} {
		if strings.HasPrefix(keyword, prefix) {
			modifier = strings.TrimSpace(prefix)
			monthStr = keyword[len(prefix):]
			break
		}
	}

	target, ok := canonicalMonths[monthStr]
	if !ok {
		return nil, false
	}

	currentYear := now.Year()
	currentMonth := now.Month()

	switch modifier {
	case "this":
		// This <month> = the 1st of that month in the current year
		return types.NewDateFromTime(time.Date(currentYear, target, 1, 0, 0, 0, 0, time.UTC)), true

	case "next":
		// Next <month> = the 1st of that month, next occurrence strictly after current month
		year := currentYear
		if target <= currentMonth {
			year++ // target month is current or past — next year
		}
		return types.NewDateFromTime(time.Date(year, target, 1, 0, 0, 0, 0, time.UTC)), true

	case "last":
		// Last <month> = the 1st of that month, most recent past occurrence
		year := currentYear
		if target >= currentMonth {
			year-- // target month is current or future — last year
		}
		return types.NewDateFromTime(time.Date(year, target, 1, 0, 0, 0, 0, time.UTC)), true
	}

	return nil, false
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
