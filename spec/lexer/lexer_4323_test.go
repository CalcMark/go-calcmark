package lexer

import (
	"testing"
)

func TestFourDigitThousands(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantTokens []TokenType
		wantValues []string
	}{
		{
			name:  "4323 with comma",
			input: "4,323",
			wantTokens: []TokenType{
				NUMBER,
				EOF,
			},
			wantValues: []string{
				"4323",
			},
		},
		{
			name:  "in function",
			input: "average of 3, 4,323, 1003",
			wantTokens: []TokenType{
				FUNC_AVERAGE_OF,
				NUMBER, // 3
				COMMA,
				NUMBER, // 4323
				COMMA,
				NUMBER, // 1003
				EOF,
			},
			wantValues: []string{
				"average of",
				"3",
				",",
				"4323",
				",",
				"1003",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Tokenize(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(tokens) != len(tt.wantTokens) {
				t.Fatalf("got %d tokens, want %d", len(tokens), len(tt.wantTokens))
			}

			for i, token := range tokens {
				if token.Type != tt.wantTokens[i] {
					t.Errorf("token %d: got type %s, want %s", i, token.Type, tt.wantTokens[i])
				}
				if i < len(tt.wantValues) && token.Value != tt.wantValues[i] {
					t.Errorf("token %d: got value %q, want %q", i, token.Value, tt.wantValues[i])
				}
			}
		})
	}
}
