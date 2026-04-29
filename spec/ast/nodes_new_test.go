package ast

import (
	"testing"
)

// TestDateLiteral tests the DateLiteral AST node
func TestDateLiteral(t *testing.T) {
	year2024 := "2024"
	year2025 := "2025"
	tests := []struct {
		name       string
		month      string
		day        string
		year       *string
		sourceText string
		want       string
	}{
		{
			name:       "basic date",
			month:      "Dec",
			day:        "25",
			year:       &year2024,
			sourceText: "Dec 25 2024",
			want:       "DateLiteral(Dec 25 2024)",
		},
		{
			name:       "short month without year",
			month:      "Dec",
			day:        "25",
			year:       nil,
			sourceText: "Dec 25",
			want:       "DateLiteral(Dec 25)",
		},
		{
			name:       "full month without year",
			month:      "December",
			day:        "25",
			year:       nil,
			sourceText: "December 25",
			want:       "DateLiteral(December 25)",
		},
		{
			name:       "with year",
			month:      "Jan",
			day:        "1",
			year:       &year2025,
			sourceText: "Jan 1 2025",
			want:       "DateLiteral(Jan 1 2025)",
		},
		{
			name:       "single digit day",
			month:      "Feb",
			day:        "5",
			year:       nil,
			sourceText: "Feb 5",
			want:       "DateLiteral(Feb 5)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &DateLiteral{
				Month:      tt.month,
				Day:        tt.day,
				Year:       tt.year,
				SourceText: tt.sourceText,
				Range:      &Range{Start: Position{Line: 1, Column: 1}, End: Position{Line: 1, Column: 1 + 1}},
			}

			if got := node.String(); got != tt.want {
				t.Errorf("DateLiteral.String() = %v, want %v", got, tt.want)
			}

			if node.GetRange() == nil {
				t.Error("DateLiteral.GetRange() returned nil")
			}
		})
	}
}

// TestDurationLiteral tests the DurationLiteral AST node
func TestDurationLiteral(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		unit       string
		sourceText string
		want       string
	}{
		{
			name:       "days plural",
			value:      "5",
			unit:       "days",
			sourceText: "5 days",
			want:       "DurationLiteral(5 days)",
		},
		{
			name:       "day singular",
			value:      "1",
			unit:       "day",
			sourceText: "1 day",
			want:       "DurationLiteral(1 day)",
		},
		{
			name:       "hours",
			value:      "3",
			unit:       "hours",
			sourceText: "3 hours",
			want:       "DurationLiteral(3 hours)",
		},
		{
			name:       "minutes",
			value:      "30",
			unit:       "minutes",
			sourceText: "30 minutes",
			want:       "DurationLiteral(30 minutes)",
		},
		{
			name:       "weeks",
			value:      "2",
			unit:       "weeks",
			sourceText: "2 weeks",
			want:       "DurationLiteral(2 weeks)",
		},
		{
			name:       "decimal value",
			value:      "1.5",
			unit:       "hours",
			sourceText: "1.5 hours",
			want:       "DurationLiteral(1.5 hours)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &DurationLiteral{
				Value:      tt.value,
				Unit:       tt.unit,
				SourceText: tt.sourceText,
				Range:      &Range{Start: Position{Line: 1, Column: 1}, End: Position{Line: 1, Column: 1 + 1}},
			}

			if got := node.String(); got != tt.want {
				t.Errorf("DurationLiteral.String() = %v, want %v", got, tt.want)
			}

			if node.GetRange() == nil {
				t.Error("DurationLiteral.GetRange() returned nil")
			}
		})
	}
}

// TestNewNodesImplementNodeInterface ensures new nodes implement Node interface
func TestNewNodesImplementNodeInterface(t *testing.T) {
	var _ Node = (*DateLiteral)(nil)
	var _ Node = (*DurationLiteral)(nil)
}

// TestDateLiteralEdgeCases tests edge cases for DateLiteral
func TestDateLiteralEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		month string
		day   string
		year  *string
		valid bool
	}{
		{
			name:  "valid date",
			month: "Dec",
			day:   "25",
			year:  nil,
			valid: true,
		},
		{
			name:  "day 31",
			month: "Jan",
			day:   "31",
			year:  nil,
			valid: true,
		},
		{
			name:  "day 1",
			month: "Feb",
			day:   "1",
			year:  nil,
			valid: true,
		},
		// Note: Semantic validation (e.g., Feb 30 is invalid) happens in analyzer, not AST
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &DateLiteral{
				Month:      tt.month,
				Day:        tt.day,
				Year:       tt.year,
				SourceText: tt.month + " " + tt.day,
				Range:      &Range{Start: Position{Line: 1, Column: 1}, End: Position{Line: 1, Column: 1 + 1}},
			}

			// Should implement Node interface
			var _ Node = node

			// Verify fields are set correctly
			if node.Month != tt.month {
				t.Errorf("Month = %q, want %q", node.Month, tt.month)
			}
		})
	}
}

// TestDurationLiteralUnits tests all supported time units
func TestDurationLiteralUnits(t *testing.T) {
	units := []string{
		"day", "days",
		"hour", "hours",
		"minute", "minutes",
		"second", "seconds",
		"week", "weeks",
		"month", "months",
		"year", "years",
	}

	for _, unit := range units {
		t.Run(unit, func(t *testing.T) {
			node := &DurationLiteral{
				Value:      "5",
				Unit:       unit,
				SourceText: "5 " + unit,
				Range:      &Range{Start: Position{Line: 1, Column: 1}, End: Position{Line: 1, Column: 1 + 1}},
			}

			expected := "DurationLiteral(5 " + unit + ")"
			if got := node.String(); got != expected {
				t.Errorf("DurationLiteral.String() = %v, want %v", got, expected)
			}
		})
	}
}

// TestEndOfExpr_StringIncludesInner verifies the AST debug shape.
// String() output is used in interpreter trace logs; nesting must
// be visible.
func TestEndOfExpr_StringIncludesInner(t *testing.T) {
	inner := &RelativeDateLiteral{Keyword: "Q:1", SourceText: "Q1"}
	node := &EndOfExpr{
		Period:     inner,
		SourceText: "end of Q1",
		Range:      &Range{Start: Position{Line: 5, Column: 1}, End: Position{Line: 5, Column: 10}},
	}
	got := node.String()
	if got == "" {
		t.Fatal("EndOfExpr.String() should not be empty")
	}
	// Sanity: must mention the inner node so trace logs are useful.
	if !contains(got, "Q") {
		t.Errorf("EndOfExpr.String() = %q should mention inner; expected to contain %q", got, "Q")
	}
}

// TestEndOfExpr_GetRangeReturnsStored — pin the Range pass-through.
func TestEndOfExpr_GetRangeReturnsStored(t *testing.T) {
	r := &Range{Start: Position{Line: 7, Column: 3}, End: Position{Line: 7, Column: 12}}
	node := &EndOfExpr{Range: r}
	if node.GetRange() != r {
		t.Errorf("GetRange() should return the stored Range pointer")
	}
}

// TestStartOfExpr_String — symmetric to EndOf.
func TestStartOfExpr_String(t *testing.T) {
	inner := &RelativeDateLiteral{Keyword: "FQ:1", SourceText: "FQ1"}
	node := &StartOfExpr{Period: inner, SourceText: "start of FQ1"}
	if got := node.String(); got == "" {
		t.Errorf("StartOfExpr.String() should not be empty")
	}
}

// TestEndOfExpr_PeriodFieldAcceptsAnyNode — the inner is `Node`,
// not a constrained interface. Identifier / RelativeDateLiteral /
// any expression-producing node must work. Type-check happens in
// spec/semantic; at the AST layer this is unconstrained.
func TestEndOfExpr_PeriodFieldAcceptsAnyNode(t *testing.T) {
	cases := []Node{
		&RelativeDateLiteral{Keyword: "Q:1"},
		&Identifier{Name: "q"},
		&NumberLiteral{Value: "5"}, // semantic checker rejects this; AST allows it
	}
	for _, inner := range cases {
		_ = &EndOfExpr{Period: inner, SourceText: "end of <expr>"}
	}
}

// contains is a tiny helper to avoid importing strings just for this file.
func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// --- U3: BetweenExpr ---
//
// `between A and B` / `from A to B` — user-defined custom periods.
// AST shape mirrors EndOfExpr: Start and End are generic Node so any
// expression types as Date at semantic check time. Variable-bound
// (`between start_dt and end_dt`) typechecks at semantic; runtime
// validates Date.

func TestBetweenExpr_StringIncludesEndpoints(t *testing.T) {
	start := &DateLiteral{Month: "April", Day: "1", SourceText: "April 1"}
	end := &DateLiteral{Month: "June", Day: "30", SourceText: "June 30"}
	node := &BetweenExpr{
		Start:      start,
		End:        end,
		SourceText: "between April 1 and June 30",
		Range:      &Range{Start: Position{Line: 3, Column: 5}, End: Position{Line: 3, Column: 30}},
	}
	got := node.String()
	if got == "" {
		t.Fatal("BetweenExpr.String() should not be empty")
	}
	if !contains(got, "April") {
		t.Errorf("BetweenExpr.String() = %q should reference Start child", got)
	}
	if !contains(got, "June") {
		t.Errorf("BetweenExpr.String() = %q should reference End child", got)
	}
}

func TestBetweenExpr_GetRangeReturnsStored(t *testing.T) {
	r := &Range{Start: Position{Line: 1, Column: 1}, End: Position{Line: 1, Column: 25}}
	node := &BetweenExpr{Range: r}
	if node.GetRange() != r {
		t.Errorf("GetRange() should return the stored Range pointer")
	}
}

// TestBetweenExpr_FieldsAcceptAnyNode — Start and End are typed
// `Node`, not constrained at the AST layer. Semantic checks (U7)
// enforce both endpoints type as Date.
func TestBetweenExpr_FieldsAcceptAnyNode(t *testing.T) {
	cases := []struct{ start, end Node }{
		{&DateLiteral{Month: "April", Day: "1"}, &DateLiteral{Month: "June", Day: "30"}},
		{&Identifier{Name: "start_dt"}, &Identifier{Name: "end_dt"}},
		{&NumberLiteral{Value: "5"}, &NumberLiteral{Value: "10"}}, // semantic rejects; AST allows
	}
	for _, c := range cases {
		_ = &BetweenExpr{Start: c.start, End: c.end, SourceText: "between <a> and <b>"}
	}
}

// TestContainsScaleRef_BetweenExpr — the new arm in
// ContainsScaleRef. Recurses into both endpoints.
func TestContainsScaleRef_BetweenExpr(t *testing.T) {
	scaleNode := &DirectiveRef{Directive: "scale"}
	plain := &DateLiteral{Month: "April", Day: "1"}

	// Scale on Start
	n1 := &BetweenExpr{Start: scaleNode, End: plain}
	if !ContainsScaleRef(n1) {
		t.Error("BetweenExpr with @scale on Start should return true")
	}
	// Scale on End
	n2 := &BetweenExpr{Start: plain, End: scaleNode}
	if !ContainsScaleRef(n2) {
		t.Error("BetweenExpr with @scale on End should return true")
	}
	// Neither
	n3 := &BetweenExpr{Start: plain, End: plain}
	if ContainsScaleRef(n3) {
		t.Error("BetweenExpr without @scale should return false")
	}
}

// --- U4: LengthOfExpr ---
//
// `length of <Period>` returns Duration; `days in <Period>` returns
// Number. Both desugar to LengthOfExpr; AsNumber discriminates.

func TestLengthOfExpr_StringDistinguishesForms(t *testing.T) {
	inner := &RelativeDateLiteral{Keyword: "Q:1", SourceText: "Q1"}

	asDuration := &LengthOfExpr{
		Period:     inner,
		AsNumber:   false,
		SourceText: "length of Q1",
	}
	asNumber := &LengthOfExpr{
		Period:     inner,
		AsNumber:   true,
		SourceText: "days in Q1",
	}

	got1 := asDuration.String()
	got2 := asNumber.String()
	if got1 == "" || got2 == "" {
		t.Fatal("LengthOfExpr.String() should not be empty")
	}
	if got1 == got2 {
		t.Errorf("LengthOfExpr.String() should distinguish AsNumber forms; got %q for both", got1)
	}
	// Each form must surface the inner.
	if !contains(got1, "Q") || !contains(got2, "Q") {
		t.Errorf("String() should reference inner Period child; got %q / %q", got1, got2)
	}
}

func TestLengthOfExpr_GetRangeReturnsStored(t *testing.T) {
	r := &Range{Start: Position{Line: 2, Column: 1}, End: Position{Line: 2, Column: 14}}
	node := &LengthOfExpr{Range: r}
	if node.GetRange() != r {
		t.Errorf("GetRange() should return the stored Range pointer")
	}
}

// TestLengthOfExpr_AsNumberRoundtrip — pin the discriminator field.
func TestLengthOfExpr_AsNumberRoundtrip(t *testing.T) {
	inner := &RelativeDateLiteral{Keyword: "Q:1"}
	asDuration := &LengthOfExpr{Period: inner, AsNumber: false}
	asNumber := &LengthOfExpr{Period: inner, AsNumber: true}
	if asDuration.AsNumber {
		t.Error("AsNumber should be false for `length of` form")
	}
	if !asNumber.AsNumber {
		t.Error("AsNumber should be true for `days in` form")
	}
}

func TestLengthOfExpr_PeriodAcceptsAnyNode(t *testing.T) {
	cases := []Node{
		&RelativeDateLiteral{Keyword: "Q:1"},
		&Identifier{Name: "p"},
		&NumberLiteral{Value: "5"}, // semantic rejects; AST allows
	}
	for _, inner := range cases {
		_ = &LengthOfExpr{Period: inner, SourceText: "length of <expr>"}
	}
}

// TestContainsScaleRef_LengthOfExpr — recurses into Period child.
func TestContainsScaleRef_LengthOfExpr(t *testing.T) {
	scaleNode := &DirectiveRef{Directive: "scale"}
	plain := &RelativeDateLiteral{Keyword: "Q:1"}

	n1 := &LengthOfExpr{Period: scaleNode}
	if !ContainsScaleRef(n1) {
		t.Error("LengthOfExpr with @scale on Period should return true")
	}
	n2 := &LengthOfExpr{Period: plain}
	if ContainsScaleRef(n2) {
		t.Error("LengthOfExpr without @scale should return false")
	}
}
