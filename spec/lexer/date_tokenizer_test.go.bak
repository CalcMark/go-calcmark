package lexer_test

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/lexer"
)

// TestDateKeywordTokenization tests that date keywords are recognized
func TestDateKeywordTokenization(t *testing.T) {
	tests := []struct {
		input    string
		expected lexer.TokenType
	}{
		{"today", lexer.DATE_TODAY},
		{"tomorrow", lexer.DATE_TOMORROW},
		{"yesterday", lexer.DATE_YESTERDAY},
		{"this week", lexer.DATE_THIS_WEEK},
		{"this month", lexer.DATE_THIS_MONTH},
		{"this year", lexer.DATE_THIS_YEAR},
		{"next week", lexer.DATE_NEXT_WEEK},
		{"next month", lexer.DATE_NEXT_MONTH},
		{"next year", lexer.DATE_NEXT_YEAR},
		{"last week", lexer.DATE_LAST_WEEK},
		{"last month", lexer.DATE_LAST_MONTH},
		{"last year", lexer.DATE_LAST_YEAR},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lex := lexer.NewLexer(tt.input)
			tokens, err := lex.Tokenize()

			if err != nil {
				t.Fatalf("Tokenize(%q) error = %v", tt.input, err)
			}

			if len(tokens) < 1 {
				t.Fatalf("Tokenize(%q) returned no tokens", tt.input)
			}

			if tokens[0].Type != tt.expected {
				t.Errorf("Tokenize(%q) token type = %v, want %v",
					tt.input, tokens[0].Type, tt.expected)
			}
		})
	}
}

// TestDateLiteralTokenization tests date literal recognition
func TestDateLiteralTokenization(t *testing.T) {
	tests := []struct {
		input         string
		expectedValue string // Format: "Month:Day:Year"
	}{
		{"Dec 12", "December:12:"},
		{"December 25", "December:25:"},
		{"Dec 12 2026", "December:12:2026"},
		{"December 25 2025", "December:25:2025"},
		{"January 2026", "January:1:2026"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lex := lexer.NewLexer(tt.input)
			tokens, err := lex.Tokenize()

			if err != nil {
				t.Fatalf("Tokenize(%q) error = %v", tt.input, err)
			}

			if len(tokens) < 1 {
				t.Fatalf("Tokenize(%q) returned no tokens", tt.input)
			}

			if tokens[0].Type != lexer.DATE_LITERAL {
				t.Errorf("Tokenize(%q) token type = %v, want DATE_LITERAL",
					tt.input, tokens[0].Type)
			}

			if string(tokens[0].Value) != tt.expectedValue {
				t.Errorf("Tokenize(%q) value = %q, want %q",
					tt.input, string(tokens[0].Value), tt.expectedValue)
			}
		})
	}
}

// TestDurationLiteralTokenization tests duration literal recognition
func TestDurationLiteralTokenization(t *testing.T) {
	tests := []struct {
		input         string
		expectedValue string // Format: "value:unit:value:unit:..."
	}{
		{"2 days", "2:day"},
		{"3 weeks", "3:week"},
		{"1 month", "1:month"},
		{"2 weeks and 3 days", "2:week:3:day"},
		{"1 year and 6 months", "1:year:6:month"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lex := lexer.NewLexer(tt.input)
			tokens, err := lex.Tokenize()

			if err != nil {
				t.Fatalf("Tokenize(%q) error = %v", tt.input, err)
			}

			if len(tokens) < 1 {
				t.Fatalf("Tokenize(%q) returned no tokens", tt.input)
			}

			if tokens[0].Type != lexer.DURATION_LITERAL {
				t.Errorf("Tokenize(%q) token type = %v, want DURATION_LITERAL",
					tt.input, tokens[0].Type)
			}

			if string(tokens[0].Value) != tt.expectedValue {
				t.Errorf("Tokenize(%q) value = %q, want %q",
					tt.input, string(tokens[0].Value), tt.expectedValue)
			}
		})
	}
}
