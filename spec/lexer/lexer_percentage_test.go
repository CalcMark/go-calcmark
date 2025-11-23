package lexer

import (
	"testing"
)

// Test that percentage literals are tokenized correctly
// NOTE: Lexer returns NUMBER_PERCENT token, conversion happens in types.NewNumber
func TestPercentageLiterals(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name:  "Integer percentage",
			input: "20%",
			expected: []Token{
				{Type: NUMBER_PERCENT, Value: "20%"},
				{Type: EOF},
			},
		},
		{
			name:  "Decimal percentage",
			input: "0.5%",
			expected: []Token{
				{Type: NUMBER_PERCENT, Value: "0.5%"},
				{Type: EOF},
			},
		},
		{
			name:  "Large percentage",
			input: "150%",
			expected: []Token{
				{Type: NUMBER_PERCENT, Value: "150%"},
				{Type: EOF},
			},
		},
		{
			name:  "Percentage with thousands separator (comma)",
			input: "1,000%",
			expected: []Token{
				{Type: NUMBER_PERCENT, Value: "1000%"},
				{Type: EOF},
			},
		},
		{
			name:  "Percentage with thousands separator (underscore)",
			input: "1_000%",
			expected: []Token{
				{Type: NUMBER_PERCENT, Value: "1000%"},
				{Type: EOF},
			},
		},
		{
			name:  "Percentage in expression",
			input: "$100 * 20%",
			expected: []Token{
				{Type: CURRENCY_SYM, Value: "$"},
				{Type: NUMBER, Value: "100"},
				{Type: MULTIPLY, Value: ""},
				{Type: NUMBER_PERCENT, Value: "20%"},
				{Type: EOF},
			},
		},
		{
			name:  "Multiple percentages",
			input: "20% + 30%",
			expected: []Token{
				{Type: NUMBER_PERCENT, Value: "20%"},
				{Type: PLUS, Value: ""},
				{Type: NUMBER_PERCENT, Value: "30%"},
				{Type: EOF},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lex := NewLexer(tt.input)
			tokens, err := lex.Tokenize()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(tokens) != len(tt.expected) {
				t.Fatalf("Expected %d tokens, got %d\nExpected: %v\nGot: %v",
					len(tt.expected), len(tokens), tt.expected, tokens)
			}

			for i, token := range tokens {
				if token.Type != tt.expected[i].Type {
					t.Errorf("Token %d: expected type %s, got %s",
						i, tt.expected[i].Type, token.Type)
				}
				if tt.expected[i].Value != "" && token.Value != tt.expected[i].Value {
					t.Errorf("Token %d: expected value %q, got %q",
						i, tt.expected[i].Value, token.Value)
				}
			}
		})
	}
}

// TestPercentageInvalidCases tests that percentages require NO whitespace
// and identifiers cannot end with %.
func TestPercentageInvalidCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		description string
	}{
		{
			name:        "Percentage with space before %",
			input:       "20 %",
			expectError: false, // Tokenizes as: NUMBER(20) MODULUS(%)
			description: "Space before % makes it a modulus operator, not percentage",
		},
		{
			name:        "Identifier with %",
			input:       "my_variable%",
			expectError: true, // "%" cannot follow identifier
			description: "Identifiers cannot end with %",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lex := NewLexer(tt.input)
			tokens, err := lex.Tokenize()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none. Tokens: %v", tokens)
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError && err == nil {
				t.Logf("✓ %s", tt.description)
				t.Logf("  Tokens: %v", tokens)
			}
		})
	}
}

// TestPercentageEdgeCases tests edge cases for percentage parsing
func TestPercentageEdgeCases(t *testing.T) {
	t.Skip("Percentage auto-conversion not yet implemented")
	tests := []struct {
		name  string
		input string
		want  string // expected normalized value
	}{
		{
			name:  "Zero percent",
			input: "0%",
			want:  "0.00",
		},
		{
			name:  "Fractional percentage",
			input: "0.1%",
			want:  "0.001",
		},
		{
			name:  "100 percent",
			input: "100%",
			want:  "1.00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lex := NewLexer(tt.input)
			tokens, err := lex.Tokenize()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tokens[0].Type != NUMBER {
				t.Errorf("Expected NUMBER token, got %s", tokens[0].Type)
			}

			if tokens[0].Value != tt.want {
				t.Errorf("Expected value %q, got %q", tt.want, tokens[0].Value)
			}
		})
	}
}

// TestModulusOperatorVsPercentage tests distinguishing % as modulus vs percentage
func TestModulusOperatorVsPercentage(t *testing.T) {
	t.Skip("Percentage auto-conversion not yet implemented")
	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name:  "Modulus operator with spaces",
			input: "10 % 3",
			expected: []Token{
				{Type: NUMBER, Value: "10"},
				{Type: MODULUS},
				{Type: NUMBER, Value: "3"},
				{Type: EOF},
			},
		},
		{
			name:  "Modulus with no space before but space after",
			input: "10% 3",
			expected: []Token{
				{Type: NUMBER, Value: "0.10"}, // This is 10% as percentage
				{Type: NUMBER, Value: "3"},
				{Type: EOF},
			},
		},
		{
			name:  "Modulus in expression",
			input: "x = 10 % 3",
			expected: []Token{
				{Type: IDENTIFIER, Value: "x"},
				{Type: ASSIGN},
				{Type: NUMBER, Value: "10"},
				{Type: MODULUS},
				{Type: NUMBER, Value: "3"},
				{Type: EOF},
			},
		},
		{
			name:  "Percentage in assignment",
			input: "x = 10%",
			expected: []Token{
				{Type: IDENTIFIER, Value: "x"},
				{Type: ASSIGN},
				{Type: NUMBER, Value: "0.10"},
				{Type: EOF},
			},
		},
		{
			name:  "Mixed percentage and modulus",
			input: "20% + 10 % 3",
			expected: []Token{
				{Type: NUMBER, Value: "0.20"}, // 20%
				{Type: PLUS},
				{Type: NUMBER, Value: "10"},
				{Type: MODULUS},
				{Type: NUMBER, Value: "3"},
				{Type: EOF},
			},
		},
		{
			name:  "Currency followed by %",
			input: "$100 % 10",
			expected: []Token{
				{Type: QUANTITY, Value: "100:$"},
				{Type: MODULUS},
				{Type: NUMBER, Value: "10"},
				{Type: EOF},
			},
		},
		{
			name:  "Parentheses with percentage",
			input: "(20%)",
			expected: []Token{
				{Type: LPAREN},
				{Type: NUMBER, Value: "0.20"},
				{Type: RPAREN},
				{Type: EOF},
			},
		},
		{
			name:  "Parentheses with modulus",
			input: "(10 % 3)",
			expected: []Token{
				{Type: LPAREN},
				{Type: NUMBER, Value: "10"},
				{Type: MODULUS},
				{Type: NUMBER, Value: "3"},
				{Type: RPAREN},
				{Type: EOF},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lex := NewLexer(tt.input)
			tokens, err := lex.Tokenize()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(tokens) != len(tt.expected) {
				t.Fatalf("Expected %d tokens, got %d\nExpected: %v\nGot: %v",
					len(tt.expected), len(tokens), tt.expected, tokens)
			}

			for i, token := range tokens {
				if token.Type != tt.expected[i].Type {
					t.Errorf("Token %d: expected type %s, got %s",
						i, tt.expected[i].Type, token.Type)
				}
				if tt.expected[i].Value != "" && token.Value != tt.expected[i].Value {
					t.Errorf("Token %d: expected value %q, got %q",
						i, tt.expected[i].Value, token.Value)
				}
			}
		})
	}
}

// TestPercentageAfterReservedKeyword tests that % cannot follow reserved keywords
func TestPercentageAfterReservedKeyword(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{
			name:        "true followed by %",
			input:       "true%",
			expectError: false, // 'true' is BOOLEAN token, not identifier, so no error but % becomes separate MODULUS
		},
		{
			name:        "false followed by %",
			input:       "false%",
			expectError: false, // Same as above
		},
		{
			name:        "Reserved keyword 'and' followed by %",
			input:       "and%",
			expectError: false, // 'and' is reserved keyword token, not identifier
		},
		{
			name:        "Regular identifier followed by %",
			input:       "myvar%",
			expectError: true, // Should error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lex := NewLexer(tt.input)
			tokens, err := lex.Tokenize()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none. Tokens: %v", tokens)
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
