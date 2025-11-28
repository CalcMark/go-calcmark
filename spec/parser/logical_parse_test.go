package parser_test

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/parser"
)

// TestLogicalOperatorParsing tests that logical operators parse correctly
func TestLogicalOperatorParsing(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"true and true", "true and true\n"},
		{"true or false", "true or false\n"},
		{"not true", "not true\n"},
		{"compound and", "true and true and true\n"},
		{"compound or", "false or false or true\n"},
		{"mixed precedence", "true or false and false\n"},
		{"parentheses", "(true or false) and true\n"},
		{"comparison with and", "5 > 3 and 10 < 20\n"},
		{"comparison with or", "1 > 2 or 3 > 2\n"},
		{"not with comparison", "not (5 == 5)\n"},
		{"complex", "(5 > 3) and (10 < 20) and not false\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Errorf("Parse(%q) failed: %v", tt.input, err)
			}
			if len(nodes) == 0 {
				t.Errorf("Parse(%q) returned no nodes", tt.input)
			}
		})
	}
}

// TestLogicalOperatorParseErrors tests expressions that should fail to parse
func TestLogicalOperatorParseErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		// Missing operands
		{"missing right operand and", "true and\n"},
		{"missing right operand or", "false or\n"},
		{"missing operand not", "not\n"},
		{"unclosed paren with logical", "(true and false\n"},

		// Keywords at wrong positions (and/or are keywords, not identifiers)
		{"and at start", "and true\n"},
		{"or at start", "or false\n"},

		// Multiple statements without newlines (parser enforces one statement per line)
		{"multiple identifiers no newline", "adn true false\n"},
		{"double and", "true and and false\n"},
		{"double or", "false or or true\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.Parse(tt.input)
			if err == nil {
				t.Errorf("Parse(%q) should have failed but succeeded", tt.input)
			}
		})
	}
}
