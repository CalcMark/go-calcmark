package interpreter

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/v2/spec/types"
	"github.com/shopspring/decimal"
)

// Notes are informational messages attached to a statement that
// evaluated successfully. They surface the assumptions the evaluator
// made on the user's behalf — "3 days = 72 hours (rate is per hour)" —
// so a correct-but-surprising number is explained next to its result
// instead of silently standing there. They are never errors: the
// statement's value is valid and downstream references are not blocked.
//
// Notes are derived after the fact from the operand values, in the
// interpreter methods that see them. The pure dispatchers
// (`evalBinaryOperation`, `evalAccumulate`) stay untouched.

// Note is one informational message for the statement being evaluated.
type Note struct {
	Code    string
	Message string
}

// NoteUnitConversion marks a note that reports a time-unit conversion
// performed during rate arithmetic.
const NoteUnitConversion = "unit_conversion"

// addNote records a note for the statement currently being evaluated.
func (interp *Interpreter) addNote(n Note) {
	interp.notes = append(interp.notes, n)
}

// TakeNotes returns the notes recorded since the last Eval (or the
// last TakeNotes) and clears them. Callers that evaluate one statement
// at a time drain this after each statement to attach the notes to it.
func (interp *Interpreter) TakeNotes() []Note {
	notes := interp.notes
	interp.notes = nil
	return notes
}

// conversionNote describes the duration conversion that `rate * duration`
// (in either order) performed, when the duration was not already in the
// rate's time unit. Returns false when nothing was converted.
func conversionNote(left, right types.Type, operator string) (Note, bool) {
	if operator != "*" {
		return Note{}, false
	}
	rate, dur := rateAndDuration(left, right)
	if rate == nil || dur == nil {
		return Note{}, false
	}
	if !isTimeUnit(rate.PerUnit) {
		return Note{}, false
	}
	from := types.NormalizeTimeUnit(dur.Unit)
	to := types.NormalizeTimeUnit(rate.PerUnit)
	if from == to {
		return Note{}, false
	}
	converted, err := convertWithinCategory(dur.Value, dur.Unit, rate.PerUnit)
	if err != nil {
		return Note{}, false
	}
	msg := fmt.Sprintf("%s = %s (rate is per %s)",
		formatTimeSpan(dur.Value, from), formatTimeSpan(converted, to), to)
	if to == "hour" && isCalendarUnit(from) {
		// An hourly rate times whole days almost always means working
		// hours, not 24-hour days. Point at the shape that says so.
		msg += ". For a working day, write the hours explicitly, e.g. 8 hours/day"
	}
	return Note{Code: NoteUnitConversion, Message: msg}, true
}

// accumulateNote describes the period conversion `rate over period`
// performed when the period was not in the rate's time unit.
func accumulateNote(rate *types.Rate, periodValue decimal.Decimal, periodUnit string) (Note, bool) {
	from := types.NormalizeTimeUnit(periodUnit)
	to := types.NormalizeTimeUnit(rate.PerUnit)
	if from == to {
		return Note{}, false
	}
	fromSeconds, err := types.TimeUnitToSeconds(periodUnit)
	if err != nil {
		return Note{}, false
	}
	toSeconds, err := types.TimeUnitToSeconds(rate.PerUnit)
	if err != nil || toSeconds.IsZero() {
		return Note{}, false
	}
	converted := periodValue.Mul(fromSeconds).Div(toSeconds)
	msg := fmt.Sprintf("%s = %s (rate is per %s)",
		formatTimeSpan(periodValue, from), formatTimeSpan(converted, to), to)
	return Note{Code: NoteUnitConversion, Message: msg}, true
}

// accumulateNoteFromArgs adapts accumulateNote to the evaluated
// arguments of accumulate()/`over`, mirroring evalAccumulate's accepted
// shapes. Anything evalAccumulate would reject yields no note.
func accumulateNoteFromArgs(args []types.Type) (Note, bool) {
	if len(args) != 2 {
		return Note{}, false
	}
	rate, ok := args[0].(*types.Rate)
	if !ok {
		return Note{}, false
	}
	switch period := args[1].(type) {
	case *types.Duration:
		return accumulateNote(rate, period.Value, period.Unit)
	case *types.Quantity:
		return accumulateNote(rate, period.Value, period.Unit)
	}
	return Note{}, false
}

// rateAndDuration picks the Rate and Duration out of a binary operation's
// operands regardless of order. Either is nil when absent.
func rateAndDuration(left, right types.Type) (*types.Rate, *types.Duration) {
	if r, ok := left.(*types.Rate); ok {
		if d, ok := right.(*types.Duration); ok {
			return r, d
		}
	}
	if d, ok := left.(*types.Duration); ok {
		if r, ok := right.(*types.Rate); ok {
			return r, d
		}
	}
	return nil, nil
}

// isCalendarUnit reports whether a canonical time unit is a whole
// calendar span (day or longer), as opposed to a clock unit.
func isCalendarUnit(unit string) bool {
	switch unit {
	case "day", "week", "month", "quarter", "year":
		return true
	}
	return false
}

// formatTimeSpan renders a value with a pluralized canonical time unit
// for note text: "1 year", "3 days", "52.14 weeks". Two decimals at
// most, trailing zeros dropped.
func formatTimeSpan(value decimal.Decimal, unit string) string {
	num := value.Round(2).String()
	if strings.Contains(num, ".") {
		num = strings.TrimRight(strings.TrimRight(num, "0"), ".")
	}
	if num != "1" {
		unit += "s"
	}
	return num + " " + unit
}
