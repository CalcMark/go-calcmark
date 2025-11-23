package parser_test

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/parser"
)

// TestDateParsing tests date literal parsing
// NOTE: These will pass once lexer-level date tokenization is implemented
func TestDateParsing(t *testing.T) {
	tests := []struct {
		name  string
		input string
		skip  bool // Skip until lexer implements date tokens
	}{
		// Date keywords
		{"today", "today\n", false},
		{"tomorrow", "tomorrow\n", false},
		{"yesterday", "yesterday\n", false},

		// Relative keywords
		{"this week", "this week\n", false},
		{"this month", "this month\n", false},
		{"this year", "this year\n", false},
		{"next week", "next week\n", false},
		{"next month", "next month\n", false},
		{"next year", "next year\n", false},
		{"last week", "last week\n", false},
		{"last month", "last month\n", false},
		{"last year", "last year\n", false},

		// Month + Day
		{"dec 12", "Dec 12\n", false},
		{"december 25", "December 25\n", false},
		{"jan 1", "Jan 1\n", false},

		// Month + Day + Year
		{"dec 12 2026", "Dec 12 2026\n", false},
		{"december 25 2025", "December 25 2025\n", false},

		// Month + Year (implies 1st)
		{"december 2025", "December 2025\n", false},
		{"jan 2026", "Jan 2026\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skip("Date parsing not yet implemented in lexer")
			}

			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			if len(nodes) != 1 {
				t.Fatalf("Parse(%q) returned %d nodes, want 1", tt.input, len(nodes))
			}

			// TODO: Verify node is DateLiteral with correct fields
		})
	}
}

// TestDateArithmetic tests date + duration expressions
func TestDateArithmetic(t *testing.T) {
	tests := []struct {
		name  string
		input string
		skip  bool
	}{
		{"today + 2 weeks", "today + 2 weeks\n", true},
		{"tomorrow + 1 day", "tomorrow + 1 day\n", true},
		{"Dec 25 + 30 days", "Dec 25 + 30 days\n", true},
		{"today - 3 days", "today - 3 days\n", true},
		{"2 weeks from today", "2 weeks from today\n", true},
		{"1 month from Dec 25", "1 month from Dec 25\n", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skip("Date arithmetic not yet implemented")
			}

			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			if len(nodes) == 0 {
				t.Fatalf("Parse(%q) returned 0 nodes", tt.input)
			}

			// TODO: Verify node is BinaryOp or DateArithmetic
		})
	}
}

// TestDurationParsing tests duration literal parsing
func TestDurationParsing(t *testing.T) {
	tests := []struct {
		name  string
		input string
		skip  bool
	}{
		// Simple durations
		{"2 days", "2 days\n", false},
		{"3 weeks", "3 weeks\n", false},
		{"1 month", "1 month\n", false},
		{"1 year", "1 year\n", false},

		// Compound durations
		{"2 weeks and 3 days", "2 weeks and 3 days\n", false},
		{"1 year and 6 months", "1 year and 6 months\n", false},
		{"3 weeks and 2 days", "3 weeks and 2 days\n", false},

		// Multi-term
		{"1 year and 2 months and 3 days", "1 year and 2 months and 3 days\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skip("Duration parsing not yet implemented in lexer")
			}

			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}

			if len(nodes) != 1 {
				t.Fatalf("Parse(%q) returned %d nodes, want 1", tt.input, len(nodes))
			}

			// TODO: Verify node is DurationLiteral
		})
	}
}

// TestInvalidDates tests error handling for invalid date syntax
func TestInvalidDates(t *testing.T) {
	tests := []struct {
		name  string
		input string
		skip  bool
	}{
		{"month without day", "December\n", true},
		{"invalid month", "Decembr 12\n", true},
		{"day without month", "12\n", true}, // Ambiguous - could be number
		{"invalid day", "December 32\n", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skip("Date validation not yet implemented")
			}

			_, err := parser.Parse(tt.input)
			if err == nil {
				t.Errorf("Parse(%q) expected error, got nil", tt.input)
			}
		})
	}
}
