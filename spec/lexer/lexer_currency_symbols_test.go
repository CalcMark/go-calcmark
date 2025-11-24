package lexer

import (
	"testing"
)

// TestCurrencySymbols tests that currency symbols are tokenized separately
func TestCurrencySymbols(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantTokens []TokenType
		wantValues []string
	}{
		{
			name:       "dollar sign",
			input:      "$100",
			wantTokens: []TokenType{CURRENCY_SYM, NUMBER, EOF},
			wantValues: []string{"$", "100", ""},
		},
		{
			name:       "euro sign",
			input:      "€100",
			wantTokens: []TokenType{CURRENCY_SYM, NUMBER, EOF},
			wantValues: []string{"€", "100", ""},
		},
		{
			name:       "pound sign",
			input:      "£100",
			wantTokens: []TokenType{CURRENCY_SYM, NUMBER, EOF},
			wantValues: []string{"£", "100", ""},
		},
		{
			name:       "yen sign",
			input:      "¥100",
			wantTokens: []TokenType{CURRENCY_SYM, NUMBER, EOF},
			wantValues: []string{"¥", "100", ""},
		},
		{
			name:       "euro with thousands",
			input:      "€5,000",
			wantTokens: []TokenType{CURRENCY_SYM, NUMBER, EOF},
			wantValues: []string{"€", "5000", ""},
		},
		{
			name:       "mixed currency list",
			input:      "$100, €200, £300, ¥400",
			wantTokens: []TokenType{CURRENCY_SYM, NUMBER, COMMA, CURRENCY_SYM, NUMBER, COMMA, CURRENCY_SYM, NUMBER, COMMA, CURRENCY_SYM, NUMBER, EOF},
			wantValues: []string{"$", "100", ",", "€", "200", ",", "£", "300", ",", "¥", "400", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lex := NewLexer(tt.input)
			tokens, err := lex.Tokenize()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(tokens) != len(tt.wantTokens) {
				t.Fatalf("Expected %d tokens, got %d\n\tExpected: %v\n\tGot: %v",
					len(tt.wantTokens), len(tokens), tt.wantTokens, tokenTypes(tokens))
			}

			for i, tok := range tokens {
				if tok.Type != tt.wantTokens[i] {
					t.Errorf("Token %d: expected type %s, got %s", i, tt.wantTokens[i], tok.Type)
				}
				if tok.Value != tt.wantValues[i] {
					t.Errorf("Token %d: expected value %q, got %q", i, tt.wantValues[i], tok.Value)
				}
			}
		})
	}
}

// tokenTypes extracts the types from a slice of Tokens.
func tokenTypes(tokens []Token) []TokenType {
	types := make([]TokenType, len(tokens))
	for i, tok := range tokens {
		types[i] = tok.Type
	}
	return types
}
