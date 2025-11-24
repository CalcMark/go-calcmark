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
	}{
		// Date keywords
		{"today", "today\n"},
		{"tomorrow", "tomorrow\n"},
		{"yesterday", "yesterday\n"},

		// Relative keywords
		{"this week", "this week\n"},
		{"this month", "this month\n"},
		{"this year", "this year\n"},
		{"next week", "next week\n"},
		{"next month", "next month\n"},
		{"next year", "next year\n"},
		{"last week", "last week\n"},
		{"last month", "last month\n"},
		{"last year", "last year\n"},

		// Month + Day
		{"dec 12", "Dec 12\n"},
		{"december 25", "December 25\n"},
		{"jan 1", "Jan 1\n"},

		// Month + Day + Year
		{"dec 12 2026", "Dec 12 2026\n"},
		{"december 25 2025", "December 25 2025\n"},

		// Month + Year (implies 1st)
		{"december 2025", "December 2025\n"},
		{"jan 2026", "Jan 2026\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

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
	}{
		{"today + 2 weeks", "today + 2 weeks\n"},
		{"tomorrow + 1 day", "tomorrow + 1 day\n"},
		{"Dec 25 + 30 days", "Dec 25 + 30 days\n"},
		{"today - 3 days", "today - 3 days\n"},
		{"2 weeks from today", "2 weeks from today\n"},
		{"1 month from Dec 25", "1 month from Dec 25\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

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
	}{
		// Simple durations
		{"2 days", "2 days\n"},
		{"3 weeks", "3 weeks\n"},
		{"1 month", "1 month\n"},
		{"1 year", "1 year\n"},

		// Compound durations
		{"2 weeks and 3 days", "2 weeks and 3 days\n"},
		{"1 year and 6 months", "1 year and 6 months\n"},
		{"3 weeks and 2 days", "3 weeks and 2 days\n"},

		// Multi-term
		{"1 year and 2 months and 3 days", "1 year and 2 months and 3 days\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

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
	}{
		{"month without day", "December\n"},
		{"invalid month", "Decembr 12\n"},
		{"day without month", "12\n"}, // Ambiguous - could be number
		{"invalid day", "December 32\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			_, err := parser.Parse(tt.input)
			if err == nil {
				t.Errorf("Parse(%q) expected error, got nil", tt.input)
			}
		})
	}
}
