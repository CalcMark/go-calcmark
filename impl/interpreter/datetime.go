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

// periodAbbreviationExpansion normalises the short-form period
// abbreviations into the canonical phrases the evaluator switch
// already understands. Keeps the bulk of the switch body unchanged
// instead of growing it with a dozen new cases.
var periodAbbreviationExpansion = map[string]string{
	"cy":      "this year",
	"fy":      "this fiscal year",
	"cq":      "this quarter",
	"fq":      "this fiscal quarter",
	"this cy": "this year",
	"next cy": "next year",
	"last cy": "last year",
	"this fy": "this fiscal year",
	"next fy": "next fiscal year",
	"last fy": "last fiscal year",
	"this cq": "this quarter",
	"next cq": "next quarter",
	"last cq": "last quarter",
	"this fq": "this fiscal quarter",
	"next fq": "next fiscal quarter",
	"last fq": "last fiscal quarter",
}

func (interp *Interpreter) evalRelativeDateLiteral(r *ast.RelativeDateLiteral) (types.Type, error) {
	now := interp.now()
	keyword := strings.ToLower(r.Keyword)
	if canonical, ok := periodAbbreviationExpansion[keyword]; ok {
		keyword = canonical
	}

	switch keyword {
	// True date keywords — single point in time, return Date.
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

	// v2.0: period keywords return *types.Period (the whole span)
	// rather than the START date. Callers that historically
	// type-asserted *types.Date use asDate() to unwrap to Period.Start.
	case "this week":
		return relativeWeekPeriod(now, 0), nil
	case "next week":
		return relativeWeekPeriod(now, +1), nil
	case "last week":
		return relativeWeekPeriod(now, -1), nil
	case "this month":
		return relativeMonthPeriod(now, 0), nil
	case "next month":
		return relativeMonthPeriod(now, +1), nil
	case "last month":
		return relativeMonthPeriod(now, -1), nil
	case "this year":
		return relativeYearPeriod(now, 0), nil
	case "next year":
		return relativeYearPeriod(now, +1), nil
	case "last year":
		return relativeYearPeriod(now, -1), nil

	// Fiscal expressions (require fiscal_year_starts)
	case "this fiscal quarter", "next fiscal quarter", "last fiscal quarter",
		"this fiscal year", "next fiscal year", "last fiscal year":
		return interp.evalFiscalExpression(keyword, now)

	// Calendar quarter expressions — return Period.
	case "this quarter":
		return types.NewRelativeQuarter(now, 0), nil
	case "next quarter":
		return types.NewRelativeQuarter(now, +1), nil
	case "last quarter":
		return types.NewRelativeQuarter(now, -1), nil

	default:
		// Notation: Q:1, FQ:3, FY:2026, CY:26
		if d, err := interp.evalNotation(keyword, now); d != nil || err != nil {
			return d, err
		}

		// "start of <period>" / "end of <period>" string-prefix
		// dispatches were retired here (and at the equivalent site
		// in evalNotation) when the parser started emitting
		// ast.EndOfExpr / ast.StartOfExpr structured nodes. Operator
		// dispatch now happens at the AST layer (see evalEndOfExpr /
		// evalStartOfExpr); reaching this branch with a string-form
		// "end of ..." keyword would mean the parser regressed.

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

// evalFiscalExpression evaluates fiscal quarter and fiscal year expressions.
// Requires fiscal_year_starts to be configured via frontmatter.
func (interp *Interpreter) evalFiscalExpression(keyword string, now time.Time) (types.Type, error) {
	if interp.fiscalYearStarts == nil {
		return nil, fmt.Errorf("fiscal expressions require a 'fiscal_year_starts' frontmatter key")
	}

	fc := interp.fiscalYearStarts
	fyStartMonth := time.Month(fc.month)
	fyStartDay := fc.day

	// v2.0: fiscal relatives return *types.Period.
	switch keyword {
	case "this fiscal quarter":
		return types.NewRelativeFiscalQuarter(now, fyStartMonth, fyStartDay, 0), nil
	case "next fiscal quarter":
		return types.NewRelativeFiscalQuarter(now, fyStartMonth, fyStartDay, +1), nil
	case "last fiscal quarter":
		return types.NewRelativeFiscalQuarter(now, fyStartMonth, fyStartDay, -1), nil
	case "this fiscal year":
		fy := fiscalYearLabel(now, fyStartMonth, fyStartDay)
		return types.NewFiscalYear(fy, fyStartMonth, fyStartDay), nil
	case "next fiscal year":
		fy := fiscalYearLabel(now, fyStartMonth, fyStartDay) + 1
		return types.NewFiscalYear(fy, fyStartMonth, fyStartDay), nil
	case "last fiscal year":
		fy := fiscalYearLabel(now, fyStartMonth, fyStartDay) - 1
		return types.NewFiscalYear(fy, fyStartMonth, fyStartDay), nil
	default:
		return nil, fmt.Errorf("unknown fiscal expression: %q", keyword)
	}
}

// fiscalYearLabel returns the FY *label* (the year the FY ENDS in,
// per Microsoft convention) for the fiscal year containing `now`.
// E.g., now = 2026-08-15 with fyStart = July → FY label = 2027.
func fiscalYearLabel(now time.Time, fyStartMonth time.Month, fyStartDay int) int {
	// fiscalYearWithDay returns the calendar year the FY STARTS in.
	startYear := fiscalYearWithDay(now, fyStartMonth, fyStartDay)
	if fyStartMonth > time.January {
		return startYear + 1
	}
	return startYear
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
//
// The two HasPrefix(keyword, "end of ") / "start of " dispatches
// that lived here previously were retired when the parser started
// emitting ast.EndOfExpr / ast.StartOfExpr -- operator dispatch
// happens at the AST layer (evalEndOfExpr / evalStartOfExpr) and
// flows back into evalEndOfNotation via the AST inner.
func (interp *Interpreter) evalNotation(keyword string, now time.Time) (types.Type, error) {
	parts := strings.SplitN(keyword, ":", 2)
	if len(parts) != 2 {
		return nil, nil // not a notation
	}
	prefix, value := strings.ToUpper(parts[0]), parts[1]

	switch prefix {
	case "Q":
		// v2.0: Q1-Q4 evaluates to Period (calendar quarter of the
		// current year). Resolved against `now`'s year.
		q, err := strconv.Atoi(value)
		if err != nil || q < 1 || q > 4 {
			return nil, fmt.Errorf("invalid calendar quarter: Q%s", value)
		}
		return types.NewCalendarQuarter(now.Year(), q)

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
		// FQ1 starts at the fiscal year start; the fiscal year
		// containing `now` anchors which FQ year-cycle this is.
		fy := fiscalYearWithDay(now, fyStartMonth, fyStartDay)
		fyStart := time.Date(fy, fyStartMonth, fyStartDay, 0, 0, 0, 0, time.UTC)
		return types.NewFiscalQuarter(fyStart, q)

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
		return types.NewFiscalYear(fyLabel, time.Month(fc.month), fc.day), nil

	case "CY":
		year, err := strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("invalid calendar year: CY%s", value)
		}
		if year < 100 {
			year += 2000 // 2-digit: CY26 = 2026
		}
		return types.NewCalendarYear(year), nil
	}

	return nil, nil
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
	"may":  time.May,
	"june": time.June, "jun": time.June,
	"july": time.July, "jul": time.July,
	"august": time.August, "aug": time.August,
	"september": time.September, "sep": time.September, "sept": time.September,
	"october": time.October, "oct": time.October,
	"november": time.November, "nov": time.November,
	"december": time.December, "dec": time.December,
}

// resolveMonthExpression handles "this april", "next april", "last
// april" forms. v2.0: returns *types.Period (the whole month).
// Pre-v2.0 it returned the 1st-of-month Date; sites that need that
// behavior call asDate on the result.
func resolveMonthExpression(keyword string, now time.Time) (*types.Period, bool) {
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

	year := currentYear
	switch modifier {
	case "this":
		// `this <month>` = the named month in the current year
		// (existing behavior preserved).
	case "next":
		if target <= currentMonth {
			year++
		}
	case "last":
		if target >= currentMonth {
			year--
		}
	default:
		return nil, false
	}

	p := types.NewCalendarMonth(year, target)
	p.Kind = types.PeriodNamedMonth
	// Direction reflects user-typed modifier so format / debug
	// output can name the form.
	switch modifier {
	case "next":
		p.Direction = +1
	case "last":
		p.Direction = -1
	default:
		p.Direction = 0
	}
	return p, true
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

// evalEndOfExpr evaluates `end of <period>`. v2.0 implementation:
// eval the inner (which now returns *types.Period for period-bearing
// keywords thanks to U10) and return Period.End directly.
//
// Variable-bound case: when the inner is an Identifier and resolves
// to a *types.Period at runtime, this works emergently — the Period
// value flows through evalNode → here → Period.End. R9-deferred
// path now opens up because U10 makes Period a real value type.
//
// Pre-U10 the inner could resolve to *types.Date for period
// keywords; that path is gone after U10. A defensive guard remains
// for the rare case where an Identifier holds a Date (user wrote
// `q = today; end of q`) — the semantic check accepts the
// identifier and runtime catches the type mismatch here.
func (interp *Interpreter) evalEndOfExpr(e *ast.EndOfExpr) (types.Type, error) {
	if e.Period == nil {
		return nil, fmt.Errorf("end of: missing inner expression")
	}
	inner, err := interp.evalNode(e.Period)
	if err != nil {
		return nil, err
	}
	p, err := asPeriod(inner)
	if err != nil {
		return nil, fmt.Errorf("end of: inner expression must be a period; got %T", inner)
	}
	return p.End, nil
}

// evalStartOfExpr evaluates `start of <period>`. Symmetric to
// evalEndOfExpr — returns Period.Start directly.
func (interp *Interpreter) evalStartOfExpr(s *ast.StartOfExpr) (types.Type, error) {
	if s.Period == nil {
		return nil, fmt.Errorf("start of: missing inner expression")
	}
	inner, err := interp.evalNode(s.Period)
	if err != nil {
		return nil, err
	}
	p, err := asPeriod(inner)
	if err != nil {
		return nil, fmt.Errorf("start of: inner expression must be a period; got %T", inner)
	}
	return p.Start, nil
}
