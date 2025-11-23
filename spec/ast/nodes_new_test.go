package ast

import (
	"testing"
)

// TestConvertExpr tests the ConvertExpr AST node
func TestConvertExpr(t *testing.T) {
	tests := []struct {
		name       string
		value      Node
		targetUnit string
		want       string
	}{
		{
			name: "convert quantity to unit",
			value: &NumberLiteral{
				Value:      "1",
				SourceText: "1",
				Range:      &Range{Start: Position{Line: 1, Column: 9}, End: Position{Line: 1, Column: 9 + 1}},
			},
			targetUnit: "ounces",
			want:       "ConvertExpr(NumberLiteral(1) to ounces)",
		},
		{
			name: "convert with currency",
			value: &CurrencyLiteral{
				Value:      "100",
				Symbol:     "$",
				SourceText: "$100",
				Range:      &Range{Start: Position{Line: 1, Column: 9}, End: Position{Line: 1, Column: 9 + 1}},
			},
			targetUnit: "EUR",
			want:       "ConvertExpr(CurrencyLiteral($100) to EUR)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node := &ConvertExpr{
				Value:      tt.value,
				TargetUnit: tt.targetUnit,
				Range:      &Range{Start: Position{Line: 1, Column: 1}, End: Position{Line: 1, Column: 1 + 1}},
			}

			if got := node.String(); got != tt.want {
				t.Errorf("ConvertExpr.String() = %v, want %v", got, tt.want)
			}

			if node.GetRange() == nil {
				t.Error("ConvertExpr.GetRange() returned nil")
			}
		})
	}
}

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
	var _ Node = (*ConvertExpr)(nil)
	var _ Node = (*DateLiteral)(nil)
	var _ Node = (*DurationLiteral)(nil)
}

// TestConvertExprWithComplexValue tests ConvertExpr with complex expressions
func TestConvertExprWithComplexValue(t *testing.T) {
	// convert (a + b) to meters
	binaryOp := &BinaryOp{
		Operator: "+",
		Left: &Identifier{
			Name:  "a",
			Range: &Range{Start: Position{Line: 1, Column: 10}, End: Position{Line: 1, Column: 10 + 1}},
		},
		Right: &Identifier{
			Name:  "b",
			Range: &Range{Start: Position{Line: 1, Column: 14}, End: Position{Line: 1, Column: 14 + 1}},
		},
		Range: &Range{Start: Position{Line: 1, Column: 10}, End: Position{Line: 1, Column: 10 + 1}},
	}

	node := &ConvertExpr{
		Value:      binaryOp,
		TargetUnit: "meters",
		Range:      &Range{Start: Position{Line: 1, Column: 1}, End: Position{Line: 1, Column: 1 + 1}},
	}

	want := "ConvertExpr(BinaryOp(\"+\", Identifier(\"a\"), Identifier(\"b\")) to meters)"
	if got := node.String(); got != want {
		t.Errorf("ConvertExpr.String() = %v, want %v", got, want)
	}
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

			// AST node should always be creatable
			if node == nil {
				t.Error("Failed to create DateLiteral node")
			}

			// Should implement Node interface
			var _ Node = node
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
