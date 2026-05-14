package semantic

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/parser"
)

// runChecker parses + checks the input and returns its diagnostics.
func runChecker(t *testing.T, input string) []Diagnostic {
	t.Helper()
	nodes, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse(%q): %v", input, err)
	}
	c := NewChecker()
	c.Check(nodes)
	return c.diagnostics
}

// hasEndOfPeriodDiagnostic reports whether the diag list contains
// the period-required error (used for both `end of` and `start of`).
func hasEndOfPeriodDiagnostic(diags []Diagnostic) bool {
	for _, d := range diags {
		if d.Code == DiagInvalidEndOfPeriod {
			return true
		}
	}
	return false
}

// findInvalidEndOfPeriodMessage returns the first matching diag's
// message for substring assertions, or "" if no diagnostic exists.
func findInvalidEndOfPeriodMessage(diags []Diagnostic) string {
	for _, d := range diags {
		if d.Code == DiagInvalidEndOfPeriod {
			return d.Message
		}
	}
	return ""
}

func TestSemantic_EndOfPeriodLiteralIsClean(t *testing.T) {
	for _, input := range []string{
		"x = end of Q1",
		"x = end of Q2",
		"x = end of FQ1",
		"x = end of this month",
		"x = end of this fiscal quarter",
		"x = end of this fiscal year",
		"x = end of next quarter",
		"x = end of last fiscal year",
	} {
		t.Run(input, func(t *testing.T) {
			diags := runChecker(t, input)
			if hasEndOfPeriodDiagnostic(diags) {
				t.Errorf("expected no end-of-period diagnostic for %q; got %v", input, diags)
			}
		})
	}
}

func TestSemantic_StartOfPeriodLiteralIsClean(t *testing.T) {
	for _, input := range []string{
		"x = start of Q1",
		"x = start of FQ2",
		"x = start of this month",
	} {
		t.Run(input, func(t *testing.T) {
			diags := runChecker(t, input)
			if hasEndOfPeriodDiagnostic(diags) {
				t.Errorf("expected no diagnostic for %q; got %v", input, diags)
			}
		})
	}
}

func TestSemantic_EndOfTodayRejected(t *testing.T) {
	diags := runChecker(t, "x = end of today")
	msg := findInvalidEndOfPeriodMessage(diags)
	if msg == "" {
		t.Fatalf("expected end-of-period diagnostic for `end of today`; got %v", diags)
	}
	if !strings.Contains(msg, "Date") || !strings.Contains(msg, "end of") {
		t.Errorf("diagnostic message %q should mention `Date` and `end of`", msg)
	}
}

func TestSemantic_EndOfNumberRejected(t *testing.T) {
	diags := runChecker(t, "x = end of 5")
	msg := findInvalidEndOfPeriodMessage(diags)
	if msg == "" {
		t.Fatalf("expected end-of-period diagnostic for `end of 5`; got %v", diags)
	}
	if !strings.Contains(msg, "Number") {
		t.Errorf("diagnostic message %q should mention `Number`", msg)
	}
}

func TestSemantic_StartOfTomorrowRejected(t *testing.T) {
	diags := runChecker(t, "x = start of tomorrow")
	msg := findInvalidEndOfPeriodMessage(diags)
	if msg == "" {
		t.Fatalf("expected diagnostic for `start of tomorrow`; got %v", diags)
	}
	if !strings.Contains(msg, "start of") {
		t.Errorf("diagnostic message %q should mention `start of`", msg)
	}
}

// TestSemantic_EndOfBareMonthNameIsAccepted — regression for a
// PR-1b bug. Pre-PR-1b, `end of April` (bare month name, no `this`
// modifier) worked because the old string-prefix dispatch was
// forgiving. PR-1b's stricter AST-level type check rejects it
// because the parser emits *ast.DateLiteral{Month: "April",
// Day: "1", Year: nil, SourceText: "April"} for the bare form,
// and DateLiteral wasn't recognized as a period. The downstream
// brainstorm decided April should become a Period type entirely,
// but until that migration lands, bare month names need to be
// accepted as period-bearing inner expressions.
//
// Discriminator for "bare month name" (vs specific date like
// `April 15`): SourceText contains no digits. Bare month names
// don't have a day or year written; specific dates always do.
func TestSemantic_EndOfBareMonthNameIsAccepted(t *testing.T) {
	for _, input := range []string{
		"x = end of April",
		"x = end of January",
		"x = end of December",
		"x = end of Apr",
		"x = end of jan", // case-insensitive
		"x = start of April",
		"x = start of November",
	} {
		t.Run(input, func(t *testing.T) {
			diags := runChecker(t, input)
			if hasEndOfPeriodDiagnostic(diags) {
				t.Errorf("bare month name should be accepted as period; %q got %v",
					input, diags)
			}
		})
	}
}

// TestSemantic_EndOfSpecificDateRejected — the inverse: when the
// user writes an explicit date (with day or year), it's a Date
// not a Period, and `end of` rejects it. Discriminator: source
// text contains digits.
func TestSemantic_EndOfSpecificDateRejected(t *testing.T) {
	for _, input := range []string{
		"x = end of April 15",
		"x = end of Apr 1 2026",
		"x = end of December 25 2026",
		"x = end of Jan 1",
	} {
		t.Run(input, func(t *testing.T) {
			diags := runChecker(t, input)
			if !hasEndOfPeriodDiagnostic(diags) {
				t.Errorf("specific date should be rejected by end of; %q got no diagnostic",
					input)
			}
		})
	}
}

func TestSemantic_EndOfIdentifierAcceptedAtTypeCheck(t *testing.T) {
	// Variable-bound case is the R9 demoted path -- semantic
	// permits, runtime catches non-Period values. Asserts NO
	// invalid-end-of-period diagnostic fires.
	diags := runChecker(t, "q = Q1\ne = end of q")
	if hasEndOfPeriodDiagnostic(diags) {
		t.Errorf("type-checker should not reject `end of q` (variable-bound); deferred to runtime. Got: %v", diags)
	}
}

func TestSemantic_NoTypePeriodInTypeKindEnum(t *testing.T) {
	// Invariant per the deepening pass: do NOT extend the TypeKind
	// enum with TypePeriod. Avoids breaking downstream consumers
	// switching on TypeKind without `default`.
	//
	// This is structural: if a future maintainer adds TypePeriod,
	// the test below fails -- forcing them to revisit the design.
	const wantKindCount = 8 // Number / Currency / Boolean / Date / Time / Duration / Quantity / Percentage
	last := TypePercentage  // last constant in the iota sequence
	if int(last) != wantKindCount-1 {
		t.Errorf("TypeKind enum size changed: last value %d, want %d. "+
			"If TypePeriod was added, revisit the API-break analysis in the PR-1b plan.",
			int(last), wantKindCount-1)
	}
}
