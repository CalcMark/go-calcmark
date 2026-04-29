package interpreter

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/types"
)

// resultStartDate returns the start Date from a result that's
// either a Date (legacy point) or a Period (v2.0 period-bearing
// keyword). Used by the U18 test migration to adapt assertions
// that historically expected (*types.Date) for `Q1`, `FY2026`,
// `this month`, etc., which now evaluate to (*types.Period).
//
// Tests that specifically need to verify the Period contract
// (Start AND End populated, Kind set correctly) should
// type-assert (*types.Period) directly rather than using this
// helper.
func resultStartDate(t *testing.T, r types.Type) *types.Date {
	t.Helper()
	if d, ok := r.(*types.Date); ok {
		return d
	}
	if p, ok := r.(*types.Period); ok {
		return p.Start
	}
	t.Fatalf("expected *types.Date or *types.Period; got %T (%v)", r, r)
	return nil
}

// resultIsDateOrPeriod reports whether r is either *types.Date or
// *types.Period. Used by tests that just want to confirm the
// expression produced "something date-like" without caring whether
// the v2.0 routing returns a Date or a Period.
func resultIsDateOrPeriod(r types.Type) bool {
	switch r.(type) {
	case *types.Date, *types.Period:
		return true
	}
	return false
}
