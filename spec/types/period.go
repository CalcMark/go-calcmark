package types

import (
	"fmt"
	"time"
)

// Period represents a span of time -- a thing whose start and end
// dates can be computed. The interpreter dispatches `end of` /
// `start of` operators by Period.Kind: each kind knows enough about
// its context (quarter index, year, fiscal-vs-calendar tag,
// direction) to compute its end without re-parsing source text.
//
// Kinds in this enum cover every period-bearing keyword the lexer
// accepts today: calendar quarters/years/months, fiscal
// quarters/years, and the relative `this/next/last <period>` forms.
// Adding a new kind requires (1) a new const here, (2) a factory
// function, (3) an `evalEndOf` arm in impl/interpreter, (4) tests
// across all three.
//
// Why a flat struct with conditional context fields rather than a
// type hierarchy: mirrors the existing `spec/types/Duration` flat
// shape (Value/Unit). Calcmark's value-type idiom is "one struct,
// kind tag, fields populated per kind." Keeps the upstream
// `types.Type` interface minimal (just String()) and avoids
// scattering Period's eval logic across many type files.
//
// Dependency boundary: this package is `spec/types` -- it must not
// import `impl/`. Period carries DATA, not behavior. End-of /
// start-of arithmetic lives in impl/interpreter.
type Period struct {
	// Start is the first day of this period. Always populated; the
	// rest of the fields are auxiliary context the interpreter uses
	// to compute the period's end without re-deriving from Start.
	Start *Date

	// Kind is the period's discriminant. Switch on this in any code
	// that needs kind-specific handling.
	Kind PeriodKind

	// QuarterIndex is 1-4 for PeriodCalendarQuarter and
	// PeriodFiscalQuarter; 0 for other kinds. Captures which quarter
	// within the year so the interpreter can compute the next-quarter
	// boundary (start + 3 months) without recomputing from Start.
	QuarterIndex int

	// Year is the calendar/fiscal year label:
	//   - PeriodCalendarQuarter / PeriodCalendarYear / PeriodCalendarMonth: calendar year
	//   - PeriodFiscalQuarter / PeriodFiscalYear: fiscal year label (Microsoft convention)
	//   - Relative kinds: 0 (Direction is the relevant signal)
	Year int

	// Month is the calendar month (1-12) for PeriodCalendarMonth and
	// PeriodNamedMonth; 0 for other kinds.
	Month int

	// Direction is -1 / 0 / +1 for last / this / next on the relative
	// kinds (PeriodRelativeWeek / Month / Year / Quarter /
	// FiscalQuarter / FiscalYear). 0 for non-relative kinds.
	Direction int
}

// PeriodKind discriminates Period values. Used by the interpreter's
// `evalEndOf` switch and by `spec/semantic` to recognize
// period-bearing AST shapes.
type PeriodKind int

const (
	PeriodCalendarQuarter PeriodKind = iota
	PeriodFiscalQuarter
	PeriodCalendarMonth
	PeriodCalendarYear
	PeriodFiscalYear
	PeriodNamedMonth
	PeriodRelativeWeek
	PeriodRelativeMonth
	PeriodRelativeYear
	PeriodRelativeQuarter
	PeriodRelativeFiscalQuarter
	PeriodRelativeFiscalYear
)

// String returns a human-readable description. Used in interpreter
// diagnostics ("end of <period>") and debug output.
func (p *Period) String() string {
	switch p.Kind {
	case PeriodCalendarQuarter:
		return fmt.Sprintf("Calendar Q%d %d", p.QuarterIndex, p.Year)
	case PeriodFiscalQuarter:
		return fmt.Sprintf("Fiscal Q%d %d", p.QuarterIndex, p.Year)
	case PeriodCalendarMonth:
		return fmt.Sprintf("%s %d", time.Month(p.Month).String(), p.Year)
	case PeriodCalendarYear:
		return fmt.Sprintf("Calendar Year %d", p.Year)
	case PeriodFiscalYear:
		return fmt.Sprintf("Fiscal Year %d", p.Year)
	case PeriodNamedMonth:
		return time.Month(p.Month).String()
	case PeriodRelativeWeek:
		return relativeLabel(p.Direction, "week")
	case PeriodRelativeMonth:
		return relativeLabel(p.Direction, "month")
	case PeriodRelativeYear:
		return relativeLabel(p.Direction, "year")
	case PeriodRelativeQuarter:
		return relativeLabel(p.Direction, "quarter")
	case PeriodRelativeFiscalQuarter:
		return relativeLabel(p.Direction, "fiscal quarter")
	case PeriodRelativeFiscalYear:
		return relativeLabel(p.Direction, "fiscal year")
	}
	return fmt.Sprintf("Period(unknown kind %d)", p.Kind)
}

func relativeLabel(direction int, base string) string {
	switch direction {
	case -1:
		return "last " + base
	case 0:
		return "this " + base
	case +1:
		return "next " + base
	}
	return base
}

// NewCalendarQuarter creates a Period for Q1-Q4 in the given year.
// Q1=Jan-Mar, Q2=Apr-Jun, Q3=Jul-Sep, Q4=Oct-Dec. Returns an error
// for quarters outside [1, 4].
func NewCalendarQuarter(year, quarter int) (*Period, error) {
	if quarter < 1 || quarter > 4 {
		return nil, fmt.Errorf("invalid calendar quarter: Q%d (must be 1-4)", quarter)
	}
	month := time.Month((quarter-1)*3 + 1) // Q1=Jan(1), Q2=Apr(4), Q3=Jul(7), Q4=Oct(10)
	start := NewDateFromTime(time.Date(year, month, 1, 0, 0, 0, 0, time.UTC))
	return &Period{
		Start:        start,
		Kind:         PeriodCalendarQuarter,
		QuarterIndex: quarter,
		Year:         year,
	}, nil
}

// NewFiscalQuarter creates a Period for the Nth fiscal quarter
// starting from `fyStart` (the first day of FQ1). FQ1 == fyStart;
// FQ2 = fyStart + 3 months; ... FQ4 = fyStart + 9 months.
// fyStart's day-of-month is preserved across all four quarters so a
// July-15 fiscal start yields FQ2 = Oct 15, FQ3 = Jan 15, etc.
// Returns an error for quarters outside [1, 4].
func NewFiscalQuarter(fyStart time.Time, quarter int) (*Period, error) {
	if quarter < 1 || quarter > 4 {
		return nil, fmt.Errorf("invalid fiscal quarter: FQ%d (must be 1-4)", quarter)
	}
	// Add (quarter-1)*3 months to fyStart while preserving the day.
	startMonth := fyStart.Month() + time.Month((quarter-1)*3)
	startYear := fyStart.Year()
	for startMonth > 12 {
		startMonth -= 12
		startYear++
	}
	start := NewDateFromTime(time.Date(startYear, startMonth, fyStart.Day(), 0, 0, 0, 0, time.UTC))
	return &Period{
		Start:        start,
		Kind:         PeriodFiscalQuarter,
		QuarterIndex: quarter,
		Year:         fyStart.Year(),
	}, nil
}

// NewCalendarYear creates a Period for Jan 1 - Dec 31 of the given
// year.
func NewCalendarYear(year int) *Period {
	return &Period{
		Start: NewDateFromTime(time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)),
		Kind:  PeriodCalendarYear,
		Year:  year,
	}
}

// NewFiscalYear creates a Period whose start is fyStartMonth /
// fyStartDay of the calendar year (fyLabel - 1) when fyStartMonth >
// 1, else of fyLabel itself. Microsoft convention: FY2027 with July
// start = Jul 1 2026 -> Jun 30 2027. The Period's Year field stores
// the FY label (the year the FY ENDS), not the start year.
func NewFiscalYear(fyLabel int, fyStartMonth time.Month, fyStartDay int) *Period {
	startYear := fyLabel
	if fyStartMonth > time.January {
		startYear = fyLabel - 1
	}
	return &Period{
		Start: NewDateFromTime(time.Date(startYear, fyStartMonth, fyStartDay, 0, 0, 0, 0, time.UTC)),
		Kind:  PeriodFiscalYear,
		Year:  fyLabel,
	}
}

// NewCalendarMonth creates a Period for the 1st through last day of
// the given month + year.
func NewCalendarMonth(year int, month time.Month) *Period {
	return &Period{
		Start: NewDateFromTime(time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)),
		Kind:  PeriodCalendarMonth,
		Year:  year,
		Month: int(month),
	}
}

// NewRelativeQuarter creates a Period for `this`/`next`/`last
// quarter` relative to `now`. Direction: -1 for last, 0 for this,
// +1 for next. Start is the first day of the calendar quarter
// containing the (offset-adjusted) now.
func NewRelativeQuarter(now time.Time, direction int) *Period {
	q := calendarQuarterOf(now.Month())
	year := now.Year()
	q += direction
	for q < 1 {
		q += 4
		year--
	}
	for q > 4 {
		q -= 4
		year++
	}
	month := time.Month((q-1)*3 + 1)
	return &Period{
		Start:        NewDateFromTime(time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)),
		Kind:         PeriodRelativeQuarter,
		QuarterIndex: q,
		Year:         year,
		Direction:    direction,
	}
}

// NewRelativeFiscalQuarter creates a Period for `this`/`next`/`last
// fiscal quarter` relative to `now`. Direction: -1 / 0 / +1.
// fyStartMonth + fyStartDay are the configured fiscal-year start.
func NewRelativeFiscalQuarter(now time.Time, fyStartMonth time.Month, fyStartDay int, direction int) *Period {
	// Find the start of the fiscal quarter containing `now`.
	// fyStart for the FY containing `now`:
	startYear := now.Year()
	if now.Month() < fyStartMonth || (now.Month() == fyStartMonth && now.Day() < fyStartDay) {
		startYear--
	}
	fyStart := time.Date(startYear, fyStartMonth, fyStartDay, 0, 0, 0, 0, time.UTC)

	// Walk forward in 3-month increments until we pass `now`. The
	// last increment's start is the FQ containing `now`.
	fqStart := fyStart
	for {
		next := fqStart.AddDate(0, 3, 0)
		if next.After(now) {
			break
		}
		fqStart = next
	}

	// Apply direction.
	fqStart = fqStart.AddDate(0, direction*3, 0)

	return &Period{
		Start:     NewDateFromTime(fqStart),
		Kind:      PeriodRelativeFiscalQuarter,
		Direction: direction,
	}
}

// calendarQuarterOf returns 1-4 for the calendar quarter containing
// the given month.
func calendarQuarterOf(month time.Month) int {
	return int((month-1)/3) + 1
}
