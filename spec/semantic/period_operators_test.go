package semantic

import (
	"strings"
	"testing"
)

// U9 — semantic checks for v2.0 period operators.
//
// `length of <P>` / `days in <P>` mirror `end of <P>` / `start of
// <P>`: inner must be period-bearing (RelativeDateLiteral with a
// period keyword, an Identifier deferred to runtime, or a future
// Period literal). Non-period inner (NumberLiteral, BareCurrency,
// DateLiteral with explicit day, true date keywords like `today`)
// is rejected at the semantic layer.
//
// `between A and B` / `from A to B` requires both endpoints to type
// as Date. Identifiers defer to runtime; obvious non-Date literals
// (NumberLiteral, BooleanLiteral) are rejected.

func hasDiag(diags []Diagnostic, code string) bool {
	for _, d := range diags {
		if d.Code == code {
			return true
		}
	}
	return false
}

func findDiagMessage(diags []Diagnostic, code string) string {
	for _, d := range diags {
		if d.Code == code {
			return d.Message
		}
	}
	return ""
}

// --- length of / days in ---

func TestSemantic_LengthOfPeriodLiteralIsClean(t *testing.T) {
	for _, input := range []string{
		"x = length of Q1",
		"x = length of FQ1",
		"x = length of this month",
		"x = length of this fiscal quarter",
		"x = length of next quarter",
		"x = length of April",
		"x = length of last fiscal year",
	} {
		t.Run(input, func(t *testing.T) {
			diags := runChecker(t, input)
			if hasDiag(diags, DiagInvalidLengthOfPeriod) {
				t.Errorf("expected no length-of diagnostic for %q; got %v", input, diags)
			}
		})
	}
}

func TestSemantic_DaysInPeriodLiteralIsClean(t *testing.T) {
	for _, input := range []string{
		"x = days in Q1",
		"x = days in this month",
		"x = days in April",
	} {
		t.Run(input, func(t *testing.T) {
			diags := runChecker(t, input)
			if hasDiag(diags, DiagInvalidLengthOfPeriod) {
				t.Errorf("expected no days-in diagnostic for %q; got %v", input, diags)
			}
		})
	}
}

func TestSemantic_LengthOfTodayRejected(t *testing.T) {
	diags := runChecker(t, "x = length of today")
	msg := findDiagMessage(diags, DiagInvalidLengthOfPeriod)
	if msg == "" {
		t.Fatalf("expected length-of diagnostic for `length of today`; got %v", diags)
	}
	// Wording locked: cmw e2e + LSP surfaces match against this.
	if !strings.Contains(msg, "length of requires a period") {
		t.Errorf("diagnostic %q should contain 'length of requires a period'", msg)
	}
	if !strings.Contains(msg, "Date") {
		t.Errorf("diagnostic %q should mention 'Date'", msg)
	}
}

func TestSemantic_DaysInTodayRejected(t *testing.T) {
	diags := runChecker(t, "x = days in today")
	msg := findDiagMessage(diags, DiagInvalidLengthOfPeriod)
	if msg == "" {
		t.Fatalf("expected days-in diagnostic for `days in today`; got %v", diags)
	}
	if !strings.Contains(msg, "days in requires a period") {
		t.Errorf("diagnostic %q should contain 'days in requires a period'", msg)
	}
	if !strings.Contains(msg, "Date") {
		t.Errorf("diagnostic %q should mention 'Date'", msg)
	}
}

func TestSemantic_DaysInNumberRejected(t *testing.T) {
	diags := runChecker(t, "x = days in 5")
	msg := findDiagMessage(diags, DiagInvalidLengthOfPeriod)
	if msg == "" {
		t.Fatalf("expected days-in diagnostic for `days in 5`; got %v", diags)
	}
	if !strings.Contains(msg, "days in requires a period") {
		t.Errorf("diagnostic %q should contain 'days in requires a period'", msg)
	}
	if !strings.Contains(msg, "Number") {
		t.Errorf("diagnostic %q should mention 'Number'", msg)
	}
}

func TestSemantic_LengthOfSpecificDateRejected(t *testing.T) {
	// `April 15` parses to DateLiteral with HasExplicitDay=true.
	// That's a specific Date, not a Period; length-of must reject.
	for _, input := range []string{
		"x = length of April 15",
		"x = length of Apr 1 2026",
		"x = length of December 25 2026",
	} {
		t.Run(input, func(t *testing.T) {
			diags := runChecker(t, input)
			if !hasDiag(diags, DiagInvalidLengthOfPeriod) {
				t.Errorf("specific date should be rejected by length-of; %q got no diagnostic", input)
			}
		})
	}
}

func TestSemantic_LengthOfIdentifierAcceptedAtTypeCheck(t *testing.T) {
	// Same R9-deferred path as `end of q`: classifier permits,
	// runtime catches non-Period bindings.
	diags := runChecker(t, "q = Q1\ne = length of q")
	if hasDiag(diags, DiagInvalidLengthOfPeriod) {
		t.Errorf("type-checker should not reject `length of q` (variable-bound); deferred to runtime. Got: %v", diags)
	}
}

// --- between / from-to ---

func TestSemantic_BetweenDateLiteralIsClean(t *testing.T) {
	for _, input := range []string{
		"x = between Apr 15 2026 and Jul 4 2026",
		"x = from Apr 15 2026 to Jul 4 2026",
		"x = between today and next month",
		"x = between today and tomorrow",
	} {
		t.Run(input, func(t *testing.T) {
			diags := runChecker(t, input)
			if hasDiag(diags, DiagInvalidBetweenEndpoint) {
				t.Errorf("expected no between-endpoint diagnostic for %q; got %v", input, diags)
			}
		})
	}
}

func TestSemantic_BetweenNumberEndpointRejected(t *testing.T) {
	diags := runChecker(t, "x = between today and 5")
	msg := findDiagMessage(diags, DiagInvalidBetweenEndpoint)
	if msg == "" {
		t.Fatalf("expected between-endpoint diagnostic for `between today and 5`; got %v", diags)
	}
	if !strings.Contains(msg, "between") {
		t.Errorf("diagnostic %q should mention 'between'", msg)
	}
	if !strings.Contains(msg, "Date") {
		t.Errorf("diagnostic %q should mention 'Date'", msg)
	}
}

func TestSemantic_BetweenBooleanEndpointRejected(t *testing.T) {
	diags := runChecker(t, "x = between true and today")
	msg := findDiagMessage(diags, DiagInvalidBetweenEndpoint)
	if msg == "" {
		t.Fatalf("expected between-endpoint diagnostic for `between true and today`; got %v", diags)
	}
}

// TestSemantic_BetweenIdentifierAcceptedAtTypeCheck — mirrors the
// R9-deferred path. Variable-bound endpoints type-check; runtime
// catches non-Date bindings.
func TestSemantic_BetweenIdentifierAcceptedAtTypeCheck(t *testing.T) {
	diags := runChecker(t, "a = today\nb = tomorrow\np = between a and b")
	if hasDiag(diags, DiagInvalidBetweenEndpoint) {
		t.Errorf("type-checker should not reject identifier endpoints; deferred to runtime. Got: %v", diags)
	}
}

// --- transitional shim collapse (U5 + U9) ---
//
// Pre-U5, bare month names like `April` parsed as DateLiteral with
// HasExplicitDay=false. The end-of/start-of checker had a
// transitional shim that accepted DateLiteral when SourceText
// contained no digits. After U5, bare months parse as
// RelativeDateLiteral{Keyword: "this April"} directly, so the shim
// is dead code. U9 collapses the shim back to "DateLiteral always
// rejects" — `end of April` still works because it goes through
// the RelativeDateLiteral arm via isPeriodKeyword("this April").

func TestSemantic_EndOfBareMonth_StillAcceptedViaRelativeDateLiteral(t *testing.T) {
	// Regression test for the shim collapse: bare month names must
	// still flow through end-of cleanly via the new
	// RelativeDateLiteral routing (U5).
	for _, input := range []string{
		"x = end of April",
		"x = end of January",
		"x = end of December",
		"x = start of April",
	} {
		t.Run(input, func(t *testing.T) {
			diags := runChecker(t, input)
			if hasDiag(diags, DiagInvalidEndOfPeriod) {
				t.Errorf("bare month should still be accepted (via RelativeDateLiteral path); %q got %v", input, diags)
			}
		})
	}
}

// TestSemantic_EndOfDateLiteral_AlwaysRejected — after the shim
// collapse, ANY DateLiteral inner is a specific date (since bare
// months no longer produce DateLiteral) and must be rejected.
// Pinning this so the simplification can't regress.
func TestSemantic_EndOfDateLiteral_AlwaysRejected(t *testing.T) {
	for _, input := range []string{
		"x = end of April 15",
		"x = end of Apr 1 2026",
		"x = end of January 2026", // year-only: no explicit day, but Year set → still DateLiteral
	} {
		t.Run(input, func(t *testing.T) {
			diags := runChecker(t, input)
			if !hasDiag(diags, DiagInvalidEndOfPeriod) {
				t.Errorf("DateLiteral should be rejected by end-of; %q got no diagnostic", input)
			}
		})
	}
}
