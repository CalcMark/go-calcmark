package lexer

import (
	"testing"
)

// TestCurrencySymbols tests that various currency symbols are recognized
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
			wantTokens: []TokenType{QUANTITY, EOF},
			wantValues: []string{"100:$", ""},
		},
		{
			name:       "euro sign",
			input:      "€100",
			wantTokens: []TokenType{QUANTITY, EOF},
			wantValues: []string{"100:€", ""},
		},
		{
			name:       "pound sign",
			input:      "£100",
			wantTokens: []TokenType{QUANTITY, EOF},
			wantValues: []string{"100:£", ""},
		},
		{
			name:       "yen sign",
			input:      "¥100",
			wantTokens: []TokenType{QUANTITY, EOF},
			wantValues: []string{"100:¥", ""},
		},
		{
			name:       "euro with thousands",
			input:      "€5,000",
			wantTokens: []TokenType{QUANTITY, EOF},
			wantValues: []string{"5000:€", ""},
		},
		{
			name:       "mixed currency list",
			input:      "$100, €200, £300, ¥400",
			wantTokens: []TokenType{QUANTITY, COMMA, QUANTITY, COMMA, QUANTITY, COMMA, QUANTITY, EOF},
			wantValues: []string{"100:$", ",", "200:€", ",", "300:£", ",", "400:¥", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Tokenize(tt.input)
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
