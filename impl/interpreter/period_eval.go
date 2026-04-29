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
	"strings"
	"time"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
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

// evalBetweenExpr evaluates `between A and B` / `from A to B` to a
// *types.Period (PeriodCustom kind). Both endpoints must evaluate
// to a Date (or a Period that narrows to its Start). Errors when
// end < start (NewCustomPeriod enforces).
func (interp *Interpreter) evalBetweenExpr(b *ast.BetweenExpr) (types.Type, error) {
	if b.Start == nil {
		return nil, fmt.Errorf("between: missing start endpoint")
	}
	if b.End == nil {
		return nil, fmt.Errorf("between: missing end endpoint")
	}
	startVal, err := interp.evalNode(b.Start)
	if err != nil {
		return nil, fmt.Errorf("between start: %w", err)
	}
	endVal, err := interp.evalNode(b.End)
	if err != nil {
		return nil, fmt.Errorf("between end: %w", err)
	}
	startDate, err := asDate(startVal)
	if err != nil {
		return nil, fmt.Errorf("between requires Date inputs; start: %w", err)
	}
	endDate, err := asDate(endVal)
	if err != nil {
		return nil, fmt.Errorf("between requires Date inputs; end: %w", err)
	}
	return types.NewCustomPeriod(startDate, endDate)
}

// isPeriodBasisTarget reports whether the unit-conversion target
// names a Period basis (fiscal / calendar). Triggers
// convertPeriodBasis instead of the duration-conversion path in
// evalUnitConversion.
func isPeriodBasisTarget(unit string) bool {
	switch strings.ToLower(unit) {
	case "fiscal", "fy", "calendar", "cy":
		return true
	}
	return false
}

// convertPeriodBasis converts a Period to the requested basis
// (fiscal ↔ calendar). Two distinct rules apply depending on
// granularity:
//
// Year-grain (CY ↔ FY): label-match. The numeric label carries
// across — `CY2026 as fiscal` = FY2026, `FY2027 as calendar` =
// CY2027. The dates differ (FY2027 with Jul start = Jul 2026 –
// Jun 2027, CY2027 = Jan – Dec 2027) but the user's "year 2027"
// mental model is preserved. Round-trip is identity.
//
// Quarter-grain (Q ↔ FQ): midpoint-contains. The dates are
// preserved across the conversion — `Q1 as fiscal` with Jul-start
// FY returns FQ3 of FY2026 (Jan–Mar 2026), the same date range
// relabeled in fiscal terms. Useful for cross-referencing against
// fiscal calendars without changing what dates the user means.
//
// Calendar-side targets always succeed (calendar boundaries don't
// depend on configuration). Fiscal-side targets require
// fiscal_year_starts in frontmatter.
//
// Other kinds (months, weeks, named-month, custom) error — they
// don't have a canonical fiscal equivalent.
func (interp *Interpreter) convertPeriodBasis(p *types.Period, target string) (*types.Period, error) {
	target = strings.ToLower(target)
	switch target {
	case "fiscal", "fy":
		return interp.toFiscalBasis(p)
	case "calendar", "cy":
		return interp.toCalendarBasis(p)
	}
	return nil, fmt.Errorf("unknown period basis: %q (use 'fiscal' or 'calendar')", target)
}

// toFiscalBasis converts a calendar-grain period to its fiscal
// counterpart. Year inputs use label-match; quarter inputs use
// midpoint-contains.
func (interp *Interpreter) toFiscalBasis(p *types.Period) (*types.Period, error) {
	if interp.fiscalYearStarts == nil {
		return nil, fmt.Errorf("fiscal basis requires a 'fiscal_year_starts' frontmatter key")
	}
	fc := interp.fiscalYearStarts
	fyStartMonth := time.Month(fc.month)
	fyStartDay := fc.day

	switch p.Kind {
	case types.PeriodCalendarYear:
		// CY<Y> → FY<Y>: numeric label carries across.
		return types.NewFiscalYear(p.Year, fyStartMonth, fyStartDay), nil

	case types.PeriodFiscalYear, types.PeriodRelativeYear, types.PeriodRelativeFiscalYear:
		// CY<Y> seen via `this year` etc. → use the calendar year of
		// the period's midpoint as the FY label. For PeriodFiscalYear
		// it's already fiscal (no-op of sorts; reuse the same label).
		span := p.End.Time.Sub(p.Start.Time)
		midpoint := p.Start.Time.Add(span / 2)
		return types.NewFiscalYear(midpoint.Year(), fyStartMonth, fyStartDay), nil

	case types.PeriodCalendarQuarter, types.PeriodFiscalQuarter,
		types.PeriodRelativeQuarter, types.PeriodRelativeFiscalQuarter:
		// Quarter-grain: find the FQ whose dates contain the input's
		// midpoint. Preserves the underlying date range when the
		// quarters happen to align (e.g., Q1 ↔ FQ3 with Jul-start FY).
		span := p.End.Time.Sub(p.Start.Time)
		midpoint := p.Start.Time.Add(span / 2)
		fyStartYear := fiscalYearWithDay(midpoint, fyStartMonth, fyStartDay)
		fyStart := time.Date(fyStartYear, fyStartMonth, fyStartDay, 0, 0, 0, 0, time.UTC)
		for q := 1; q <= 4; q++ {
			fqStart := fyStart.AddDate(0, (q-1)*3, 0)
			fqEnd := fqStart.AddDate(0, 3, -1)
			if !midpoint.Before(fqStart) && !midpoint.After(fqEnd) {
				return types.NewFiscalQuarter(fyStart, q)
			}
		}
		return nil, fmt.Errorf("internal: midpoint %v not in any FQ of FY starting %v", midpoint, fyStart)
	}
	return nil, fmt.Errorf("`as fiscal` only supports year-grain and quarter-grain periods (got kind %v)", p.Kind)
}

// toCalendarBasis is the symmetric calendar-side converter.
func (interp *Interpreter) toCalendarBasis(p *types.Period) (*types.Period, error) {
	switch p.Kind {
	case types.PeriodFiscalYear:
		// FY<Y> → CY<Y>: numeric label carries across.
		return types.NewCalendarYear(p.Year), nil

	case types.PeriodCalendarYear, types.PeriodRelativeYear, types.PeriodRelativeFiscalYear:
		// `this year` → CY of period's midpoint year. Relative fiscal
		// year → CY whose label matches the FY's resolved year.
		span := p.End.Time.Sub(p.Start.Time)
		midpoint := p.Start.Time.Add(span / 2)
		return types.NewCalendarYear(midpoint.Year()), nil

	case types.PeriodCalendarQuarter, types.PeriodFiscalQuarter,
		types.PeriodRelativeQuarter, types.PeriodRelativeFiscalQuarter:
		// Calendar quarter containing midpoint: 1=Jan-Mar, ..., 4=Oct-Dec.
		span := p.End.Time.Sub(p.Start.Time)
		midpoint := p.Start.Time.Add(span / 2)
		q := (int(midpoint.Month())-1)/3 + 1
		return types.NewCalendarQuarter(midpoint.Year(), q)
	}
	return nil, fmt.Errorf("`as calendar` only supports year-grain and quarter-grain periods (got kind %v)", p.Kind)
}

// evalLengthOfExpr evaluates `length of <P>` (returns Duration in
// days) and `days in <P>` (returns Number — integer day count).
// AsNumber discriminates the surface form. Both forms compute
// (Period.End - Period.Start) + 1 day for closed-interval semantics.
func (interp *Interpreter) evalLengthOfExpr(l *ast.LengthOfExpr) (types.Type, error) {
	if l.Period == nil {
		op := "length of"
		if l.AsNumber {
			op = "days in"
		}
		return nil, fmt.Errorf("%s: missing inner expression", op)
	}
	innerVal, err := interp.evalNode(l.Period)
	if err != nil {
		return nil, err
	}
	p, err := asPeriod(innerVal)
	if err != nil {
		op := "length of"
		if l.AsNumber {
			op = "days in"
		}
		return nil, fmt.Errorf("%s requires a period; got %T", op, innerVal)
	}

	// Closed interval, day precision: end - start + 1 day.
	days := int(p.End.Time.Sub(p.Start.Time).Hours()/24) + 1

	if l.AsNumber {
		// `days in <P>` → Number (integer day count). Used in
		// arithmetic like `cost = days in cycle * 1000`.
		return &types.Number{Value: decimal.NewFromInt(int64(days))}, nil
	}
	// `length of <P>` → Duration in days. Composes with duration
	// arithmetic (`length of Q1 + 5 days` = 95 days).
	return &types.Duration{
		Value: decimal.NewFromInt(int64(days)),
		Unit:  "day",
	}, nil
}
