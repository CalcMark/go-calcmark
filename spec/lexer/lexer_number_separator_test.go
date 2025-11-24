package lexer

import (
	"testing"
)

// TestNumberThousandsSeparators tests that commas are only treated as separators in valid positions
// Commas are valid thousands separators only when followed by exactly 3 digits
func TestNumberThousandsSeparators(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantTokens []TokenType
		wantValues []string
	}{
		// Valid thousands separators
		{
			name:       "valid thousands",
			input:      "1,000",
			wantTokens: []TokenType{NUMBER, EOF},
			wantValues: []string{"1000", ""},
		},
		{
			name:       "valid millions",
			input:      "1,000,000",
			wantTokens: []TokenType{NUMBER, EOF},
			wantValues: []string{"1000000", ""},
		},
		{
			name:       "valid billions",
			input:      "1,234,567,890",
			wantTokens: []TokenType{NUMBER, EOF},
			wantValues: []string{"1234567890", ""},
		},
		{
			name:       "thousands with underscore",
			input:      "1_000",
			wantTokens: []TokenType{NUMBER, EOF},
			wantValues: []string{"1000", ""},
		},
		{
			name:       "millions with underscore",
			input:      "1_000_000",
			wantTokens: []TokenType{NUMBER, EOF},
			wantValues: []string{"1000000", ""},
		},

		// Decimals with thousands separators
		{
			name:       "thousands with decimal",
			input:      "1,000.50",
			wantTokens: []TokenType{NUMBER, EOF},
			wantValues: []string{"1000.50", ""},
		},
		{
			name:       "thousands with decimal cents",
			input:      "1,000.01",
			wantTokens: []TokenType{NUMBER, EOF},
			wantValues: []string{"1000.01", ""},
		},
		{
			name:       "millions with decimal",
			input:      "1,234,567.89",
			wantTokens: []TokenType{NUMBER, EOF},
			wantValues: []string{"1234567.89", ""},
		},

		// Invalid separators (not followed by exactly 3 digits)
		{
			name:       "comma before single digit",
			input:      "1,2",
			wantTokens: []TokenType{NUMBER, COMMA, NUMBER, EOF},
			wantValues: []string{"1", ",", "2", ""},
		},
		{
			name:       "comma before two digits",
			input:      "12,34",
			wantTokens: []TokenType{NUMBER, COMMA, NUMBER, EOF},
			wantValues: []string{"12", ",", "34", ""},
		},
		{
			name:       "comma before four digits",
			input:      "1,2345",
			wantTokens: []TokenType{NUMBER, COMMA, NUMBER, EOF},
			wantValues: []string{"1", ",", "2345", ""},
		},

		// Comma-separated lists (not thousands separators)
		{
			name:       "list of three numbers",
			input:      "1,2,3",
			wantTokens: []TokenType{NUMBER, COMMA, NUMBER, COMMA, NUMBER, EOF},
			wantValues: []string{"1", ",", "2", ",", "3", ""},
		},
		{
			name:       "list with spaces",
			input:      "1, 2, 3",
			wantTokens: []TokenType{NUMBER, COMMA, NUMBER, COMMA, NUMBER, EOF},
			wantValues: []string{"1", ",", "2", ",", "3", ""},
		},

		// Mixed: thousands separators and list separators
		{
			name:       "thousands in list",
			input:      "1,000, 2,000, 3,000",
			wantTokens: []TokenType{NUMBER, COMMA, NUMBER, COMMA, NUMBER, EOF},
			wantValues: []string{"1000", ",", "2000", ",", "3000", ""},
		},

		// Decimals with thousands
		{
			name:       "decimal with thousands",
			input:      "1,234.56",
			wantTokens: []TokenType{NUMBER, EOF},
			wantValues: []string{"1234.56", ""},
		},
		{
			name:       "large decimal with thousands",
			input:      "9,876,543.21",
			wantTokens: []TokenType{NUMBER, EOF},
			wantValues: []string{"9876543.21", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lex := NewLexer(tt.input)
			tokens, err := lex.Tokenize()
			if err != nil {
				t.Fatalf("Tokenize(%q) error = %v, want nil", tt.input, err)
			}

			if len(tokens) != len(tt.wantTokens) {
				t.Fatalf("Tokenize(%q) returned %d tokens, want %d", tt.input, len(tokens), len(tt.wantTokens))
			}

			for i, want := range tt.wantTokens {
				if tokens[i].Type != want {
					t.Errorf("Tokenize(%q) token[%d] type = %s, want %s", tt.input, i, tokens[i].Type, want)
				}
				if tokens[i].Value != tt.wantValues[i] {
					t.Errorf("Tokenize(%q) token[%d] value = %q, want %q", tt.input, i, tokens[i].Value, tt.wantValues[i])
				}
			}
		})
	}
}

// TestCurrencyThousandsSeparators tests currency values with thousands separators
func TestCurrencyThousandsSeparators(t *testing.T) {
	t.Skip("Currency symbol tokenization not fully implemented")
	tests := []struct {
		name       string
		input      string
		wantTokens []TokenType
		wantValues []string
	}{
		{
			name:       "currency thousands",
			input:      "$1,000",
			wantTokens: []TokenType{QUANTITY, EOF},
			wantValues: []string{"1000:$", ""},
		},
		{
			name:       "currency millions",
			input:      "$1,000,000",
			wantTokens: []TokenType{QUANTITY, EOF},
			wantValues: []string{"1000000:$", ""},
		},
		{
			name:       "currency with decimal",
			input:      "$1,234.56",
			wantTokens: []TokenType{QUANTITY, EOF},
			wantValues: []string{"1234.56:$", ""},
		},
		{
			name:       "euro thousands",
			input:      "€5,000",
			wantTokens: []TokenType{QUANTITY, EOF},
			wantValues: []string{"5000:€", ""},
		},
		{
			name:       "currency list",
			input:      "$100, $200, $300",
			wantTokens: []TokenType{QUANTITY, COMMA, QUANTITY, COMMA, QUANTITY, EOF},
			wantValues: []string{"100:$", ",", "200:$", ",", "300:$", ""},
		},
		{
			name:       "currency list with thousands",
			input:      "$1,000, $2,000",
			wantTokens: []TokenType{QUANTITY, COMMA, QUANTITY, EOF},
			wantValues: []string{"1000:$", ",", "2000:$", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lex := NewLexer(tt.input)
			tokens, err := lex.Tokenize()
			if err != nil {
				t.Fatalf("Tokenize(%q) error = %v, want nil", tt.input, err)
			}

			if len(tokens) != len(tt.wantTokens) {
				t.Fatalf("Tokenize(%q) returned %d tokens, want %d", tt.input, len(tokens), len(tt.wantTokens))
			}

			for i, want := range tt.wantTokens {
				if tokens[i].Type != want {
					t.Errorf("Tokenize(%q) token[%d] type = %s, want %s", tt.input, i, tokens[i].Type, want)
				}
				if tokens[i].Value != tt.wantValues[i] {
					t.Errorf("Tokenize(%q) token[%d] value = %q, want %q", tt.input, i, tokens[i].Value, tt.wantValues[i])
				}
			}
		})
	}
}
