package interpreter

import (
	"fmt"
	"strconv"
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
		// Notation: Q:1, FQ:3, FY:2026, CY:26
		if d, err := interp.evalNotation(keyword, now); d != nil || err != nil {
			return d, err
		}

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

	fc := interp.fiscalYearStarts
	fyStartMonth := time.Month(fc.month)
	fyStartDay := fc.day

	switch keyword {
	case "this fiscal quarter":
		y, m := fiscalQuarterStart(now.Year(), now.Month(), fyStartMonth)
		nextFQ := time.Date(y, m+3, fyStartDay, 0, 0, 0, 0, time.UTC)
		return types.NewDateFromTime(nextFQ.AddDate(0, 0, -1)), nil
	case "next fiscal quarter":
		y, m := fiscalQuarterStart(now.Year(), now.Month(), fyStartMonth)
		nextNextFQ := time.Date(y, m+6, fyStartDay, 0, 0, 0, 0, time.UTC)
		return types.NewDateFromTime(nextNextFQ.AddDate(0, 0, -1)), nil
	case "last fiscal quarter":
		y, m := fiscalQuarterStart(now.Year(), now.Month(), fyStartMonth)
		thisFQ := time.Date(y, m, fyStartDay, 0, 0, 0, 0, time.UTC)
		return types.NewDateFromTime(thisFQ.AddDate(0, 0, -1)), nil
	case "this fiscal year":
		fy := fiscalYearWithDay(now, fyStartMonth, fyStartDay)
		nextFYStart := time.Date(fy+1, fyStartMonth, fyStartDay, 0, 0, 0, 0, time.UTC)
		return types.NewDateFromTime(nextFYStart.AddDate(0, 0, -1)), nil
	case "next fiscal year":
		fy := fiscalYearWithDay(now, fyStartMonth, fyStartDay) + 1
		nextFYStart := time.Date(fy+1, fyStartMonth, fyStartDay, 0, 0, 0, 0, time.UTC)
		return types.NewDateFromTime(nextFYStart.AddDate(0, 0, -1)), nil
	case "last fiscal year":
		fy := fiscalYearWithDay(now, fyStartMonth, fyStartDay) - 1
		nextFYStart := time.Date(fy+1, fyStartMonth, fyStartDay, 0, 0, 0, 0, time.UTC)
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

	fc := interp.fiscalYearStarts
	fyStartMonth := time.Month(fc.month)
	fyStartDay := fc.day

	switch keyword {
	case "this fiscal quarter":
		y, m := fiscalQuarterStart(now.Year(), now.Month(), fyStartMonth)
		return types.NewDateFromTime(time.Date(y, m, fyStartDay, 0, 0, 0, 0, time.UTC)), nil
	case "next fiscal quarter":
		y, m := fiscalQuarterStart(now.Year(), now.Month(), fyStartMonth)
		t := time.Date(y, m+3, fyStartDay, 0, 0, 0, 0, time.UTC) // Go normalizes
		return types.NewDateFromTime(t), nil
	case "last fiscal quarter":
		y, m := fiscalQuarterStart(now.Year(), now.Month(), fyStartMonth)
		t := time.Date(y, m-3, fyStartDay, 0, 0, 0, 0, time.UTC) // Go normalizes
		return types.NewDateFromTime(t), nil
	case "this fiscal year":
		y := fiscalYearWithDay(now, fyStartMonth, fyStartDay)
		return types.NewDateFromTime(time.Date(y, fyStartMonth, fyStartDay, 0, 0, 0, 0, time.UTC)), nil
	case "next fiscal year":
		y := fiscalYearWithDay(now, fyStartMonth, fyStartDay) + 1
		return types.NewDateFromTime(time.Date(y, fyStartMonth, fyStartDay, 0, 0, 0, 0, time.UTC)), nil
	case "last fiscal year":
		y := fiscalYearWithDay(now, fyStartMonth, fyStartDay) - 1
		return types.NewDateFromTime(time.Date(y, fyStartMonth, fyStartDay, 0, 0, 0, 0, time.UTC)), nil
	default:
		return nil, fmt.Errorf("unknown fiscal expression: %q", keyword)
	}
}

// fiscalYear returns the calendar year in which the fiscal year begins (month-only, day=1).
func fiscalYear(calYear int, calMonth, fyStartMonth time.Month) int {
	if calMonth >= fyStartMonth {
		return calYear
	}
	return calYear - 1
}

// fiscalYearWithDay returns the calendar year in which the fiscal year begins,
// accounting for a non-first-of-month start day.
func fiscalYearWithDay(now time.Time, fyStartMonth time.Month, fyStartDay int) int {
	fyStartThisYear := time.Date(now.Year(), fyStartMonth, fyStartDay, 0, 0, 0, 0, time.UTC)
	if now.Before(fyStartThisYear) {
		return now.Year() - 1
	}
	return now.Year()
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

// evalNotation handles Q:N, FQ:N, FY:YYYY, CY:YYYY notation.
// Returns (nil, nil) if keyword is not a notation pattern.
func (interp *Interpreter) evalNotation(keyword string, now time.Time) (types.Type, error) {
	// Check for "end of" + notation
	if strings.HasPrefix(keyword, "end of ") {
		inner := keyword[len("end of "):]
		return interp.evalEndOfNotation(inner, now)
	}
	if strings.HasPrefix(keyword, "start of ") {
		inner := keyword[len("start of "):]
		return interp.evalNotation(inner, now)
	}

	parts := strings.SplitN(keyword, ":", 2)
	if len(parts) != 2 {
		return nil, nil // not a notation
	}
	prefix, value := strings.ToUpper(parts[0]), parts[1]

	switch prefix {
	case "Q":
		q, err := strconv.Atoi(value)
		if err != nil || q < 1 || q > 4 {
			return nil, fmt.Errorf("invalid calendar quarter: Q%s", value)
		}
		month := time.Month((q-1)*3 + 1) // Q1=Jan, Q2=Apr, Q3=Jul, Q4=Oct
		return types.NewDateFromTime(time.Date(now.Year(), month, 1, 0, 0, 0, 0, time.UTC)), nil

	case "FQ":
		if interp.fiscalYearStarts == nil {
			return nil, fmt.Errorf("fiscal expressions require a 'fiscal_year_starts' frontmatter key")
		}
		q, err := strconv.Atoi(value)
		if err != nil || q < 1 || q > 4 {
			return nil, fmt.Errorf("invalid fiscal quarter: FQ%s", value)
		}
		fc := interp.fiscalYearStarts
		fyStartMonth := time.Month(fc.month)
		fyStartDay := fc.day
		// FQ1 starts at fiscal year start month, FQ2 = +3 months, etc.
		fy := fiscalYearWithDay(now, fyStartMonth, fyStartDay)
		fqMonth := time.Month(int(fyStartMonth) + (q-1)*3)
		fqYear := fy
		for fqMonth > 12 {
			fqMonth -= 12
			fqYear++
		}
		return types.NewDateFromTime(time.Date(fqYear, fqMonth, fyStartDay, 0, 0, 0, 0, time.UTC)), nil

	case "FY":
		if interp.fiscalYearStarts == nil {
			return nil, fmt.Errorf("fiscal expressions require a 'fiscal_year_starts' frontmatter key")
		}
		fyLabel, err := strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("invalid fiscal year: FY%s", value)
		}
		if fyLabel < 100 {
			fyLabel += 2000 // 2-digit: FY27 = 2027
		}
		fc := interp.fiscalYearStarts
		// FY is labeled by the year it ENDS in (Microsoft convention).
		// FY2027 with July start = starts Jul 1, 2026 (ends Jun 30, 2027).
		// When fiscal starts in January, FY label = start year (no offset).
		startYear := fyLabel
		if fc.month > 1 {
			startYear = fyLabel - 1
		}
		return types.NewDateFromTime(time.Date(startYear, time.Month(fc.month), fc.day, 0, 0, 0, 0, time.UTC)), nil

	case "CY":
		year, err := strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("invalid calendar year: CY%s", value)
		}
		if year < 100 {
			year += 2000 // 2-digit: CY26 = 2026
		}
		return types.NewDateFromTime(time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)), nil
	}

	return nil, nil
}

// evalEndOfNotation handles "end of Q2", "end of FQ1".
func (interp *Interpreter) evalEndOfNotation(keyword string, now time.Time) (types.Type, error) {
	startDate, err := interp.evalNotation(keyword, now)
	if err != nil {
		return nil, err
	}
	if startDate == nil {
		return nil, nil
	}
	d := startDate.(*types.Date)
	// End of quarter = start of quarter + 3 months - 1 day
	endDate := time.Date(d.Time.Year(), d.Time.Month()+3, d.Time.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
	return types.NewDateFromTime(endDate), nil
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
