// Period helpers for v2.0: asDate (unwraps Period.Start when a
// site that historically expected *types.Date now receives a
// *types.Period) and relative-kind factories (this/next/last
// week/month/year) that depend on the interpreter's clock.
//
// Calendar / fiscal factories (NewCalendarQuarter, NewFiscalYear,
// NewCalendarMonth, NewRelativeQuarter, NewRelativeFiscalQuarter)
// live in spec/types/period.go because they take `now` as an
// argument rather than reading interpreter state. Week / month /
// year relative factories belong here so the spec/-never-imports-
// impl/ boundary stays clean even when more interpreter state
// (e.g., week-start configuration) eventually plugs in.
package interpreter

import (
	"fmt"
	"time"

	"github.com/CalcMark/go-calcmark/spec/types"
)

// asDate unwraps a *types.Period to its Start date, returns
// *types.Date as-is, and errors otherwise. Used at every site that
// historically type-asserted (*types.Date) and now also accepts a
// Period — the legacy callers inherit the period-narrows-to-start
// convention. New callers that need a Period's End should
// type-assert *types.Period directly.
func asDate(t types.Type) (*types.Date, error) {
	switch v := t.(type) {
	case *types.Date:
		return v, nil
	case *types.Period:
		return v.Start, nil
	default:
		return nil, fmt.Errorf("expected Date or Period; got %T", t)
	}
}

// asPeriod is the symmetric: returns *types.Period as-is, errors
// otherwise. Sites that EXPECT a Period (length-of, days-in,
// between operands) use this. Date is intentionally rejected — at
// these call sites a Date is always a type error.
func asPeriod(t types.Type) (*types.Period, error) {
	if p, ok := t.(*types.Period); ok {
		return p, nil
	}
	return nil, fmt.Errorf("expected Period; got %T", t)
}

// relativeWeekPeriod constructs the Period for `this/next/last
// week` relative to `now`. Direction: -1 (last) / 0 (this) / +1
// (next). Week starts Monday (CalcMark convention; matches the
// existing Date-based eval at evalRelativeDateLiteral).
func relativeWeekPeriod(now time.Time, direction int) *types.Period {
	daysFromMonday := (int(now.Weekday()) - int(time.Monday) + 7) % 7
	monday := now.AddDate(0, 0, -daysFromMonday+direction*7)
	startT := time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)
	endT := startT.AddDate(0, 0, 6)
	return &types.Period{
		Start:     types.NewDateFromTime(startT),
		End:       types.NewDateFromTime(endT),
		Kind:      types.PeriodRelativeWeek,
		Direction: direction,
	}
}

// relativeMonthPeriod constructs the Period for `this/next/last
// month` relative to `now`. Calendar-month aligned: start = 1st of
// month, end = last day of month (handles February leap year via
// time.Date normalization).
func relativeMonthPeriod(now time.Time, direction int) *types.Period {
	month := now.Month() + time.Month(direction)
	year := now.Year()
	for month < 1 {
		month += 12
		year--
	}
	for month > 12 {
		month -= 12
		year++
	}
	p := types.NewCalendarMonth(year, month)
	p.Kind = types.PeriodRelativeMonth
	p.Direction = direction
	return p
}

// relativeYearPeriod constructs the Period for `this/next/last
// year` relative to `now`. Calendar-year aligned.
func relativeYearPeriod(now time.Time, direction int) *types.Period {
	year := now.Year() + direction
	p := types.NewCalendarYear(year)
	p.Kind = types.PeriodRelativeYear
	p.Direction = direction
	return p
}

// namedMonthPeriod constructs the Period for a `this/next/last
// <named-month>` form, resolved relative to `now`. Mirrors the
// existing resolveMonthExpression Date-based logic. Direction
// applies in years: `this April` = the April most natural relative
// to now (the upcoming or past nearest one); `next April` = next
// year's April; `last April` = previous year's April.
//
// The "this" rule matches resolveMonthExpression: when the named
// month equals `now`'s month, return that month. Otherwise return
// the month in `now`'s calendar year (e.g., `this April` from
// July returns April of this year).
func namedMonthPeriod(month time.Month, now time.Time, direction int) *types.Period {
	year := now.Year() + direction
	p := types.NewCalendarMonth(year, month)
	p.Kind = types.PeriodNamedMonth
	p.Direction = direction
	return p
}
