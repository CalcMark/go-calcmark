package lexer

import (
	"testing"
)

// Test prefix currency codes (GBP100, USD1000, etc.)
func TestPrefixCurrencyCodes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name:  "GBP with amount",
			input: "GBP100",
			expected: []Token{
				{Type: QUANTITY, Value: "100:GBP"},
				{Type: EOF},
			},
		},
		{
			name:  "USD with decimal",
			input: "USD50.25",
			expected: []Token{
				{Type: QUANTITY, Value: "50.25:USD"},
				{Type: EOF},
			},
		},
		{
			name:  "EUR with thousands separator (comma)",
			input: "EUR1,000",
			expected: []Token{
				{Type: QUANTITY, Value: "1000:EUR"},
				{Type: EOF},
			},
		},
		{
			name:  "JPY with thousands separator (underscore)",
			input: "JPY1_000_000",
			expected: []Token{
				{Type: QUANTITY, Value: "1000000:JPY"},
				{Type: EOF},
			},
		},
		{
			name:  "CHF with large decimal",
			input: "CHF999.99",
			expected: []Token{
				{Type: QUANTITY, Value: "999.99:CHF"},
				{Type: EOF},
			},
		},
		{
			name:  "CAD in expression",
			input: "CAD100 + CAD50",
			expected: []Token{
				{Type: QUANTITY, Value: "100:CAD"},
				{Type: PLUS},
				{Type: QUANTITY, Value: "50:CAD"},
				{Type: EOF},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizeHelper(tt.input)
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

// Test lowercase currency codes are treated as identifiers
func TestLowercaseCurrencyCodesAreIdentifiers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected TokenType
	}{
		{
			name:     "lowercase usd",
			input:    "usd100",
			expected: IDENTIFIER, // lowercase not allowed
		},
		{
			name:     "lowercase gbp",
			input:    "gbp50",
			expected: IDENTIFIER,
		},
		{
			name:     "mixed case",
			input:    "Usd100",
			expected: IDENTIFIER,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizeHelper(tt.input)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tokens[0].Type != tt.expected {
				t.Errorf("Expected token type %s, got %s for input %q",
					tt.expected, tokens[0].Type, tt.input)
			}
		})
	}
}

// Test invalid currency codes are treated as identifiers
func TestInvalidCurrencyCodesAreIdentifiers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected TokenType
	}{
		{
			name:     "XYZ (invalid code)",
			input:    "XYZ100",
			expected: IDENTIFIER, // XYZ is not a valid ISO 4217 code
		},
		{
			name:     "ABC (invalid code)",
			input:    "ABC50",
			expected: IDENTIFIER,
		},
		{
			name:     "ZZZ (invalid code)",
			input:    "ZZZ999",
			expected: IDENTIFIER,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizeHelper(tt.input)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tokens[0].Type != tt.expected {
				t.Errorf("Expected token type %s, got %s for input %q",
					tt.expected, tokens[0].Type, tt.input)
			}
		})
	}
}

// Test standalone currency codes (no number following)
func TestStandaloneCurrencyCodes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected TokenType
	}{
		{
			name:     "GBP alone",
			input:    "GBP",
			expected: IDENTIFIER, // Currency codes are reserved, treated as identifiers
		},
		{
			name:     "USD alone",
			input:    "USD",
			expected: IDENTIFIER,
		},
		{
			name:     "EUR followed by space",
			input:    "EUR ",
			expected: IDENTIFIER,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizeHelper(tt.input)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tokens[0].Type != tt.expected {
				t.Errorf("Expected token type %s, got %s for input %q",
					tt.expected, tokens[0].Type, tt.input)
			}
		})
	}
}

// Test postfix currency codes (100 USD, 50 GBP, etc.)
// NOTE: These will be implemented as part of general postfix quantity support
func TestPostfixCurrencyCodes(t *testing.T) {
	t.Skip("Postfix quantity support not yet implemented - will handle '100 USD' format")

	tests := []struct {
		name     string
		input    string
		expected []Token
	}{
		{
			name:  "100 USD (postfix)",
			input: "100 USD",
			expected: []Token{
				{Type: QUANTITY, Value: "100:USD"},
				{Type: EOF},
			},
		},
		{
			name:  "50 GBP (postfix)",
			input: "50 GBP",
			expected: []Token{
				{Type: QUANTITY, Value: "50:GBP"},
				{Type: EOF},
			},
		},
		{
			name:  "1000 EUR (postfix)",
			input: "1000 EUR",
			expected: []Token{
				{Type: QUANTITY, Value: "1000:EUR"},
				{Type: EOF},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizeHelper(tt.input)
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

// Test currency code assignments
func TestCurrencyCodeAssignments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenType
	}{
		{
			name:     "Assign prefix currency",
			input:    "total = GBP100",
			expected: []TokenType{IDENTIFIER, ASSIGN, QUANTITY, EOF},
		},
		{
			name:     "Assign multiple currencies",
			input:    "x = USD50 + EUR25",
			expected: []TokenType{IDENTIFIER, ASSIGN, QUANTITY, PLUS, QUANTITY, EOF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizeHelper(tt.input)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(tokens) != len(tt.expected) {
				t.Fatalf("Expected %d tokens, got %d", len(tt.expected), len(tokens))
			}

			for i, token := range tokens {
				if token.Type != tt.expected[i] {
					t.Errorf("Token %d: expected type %s, got %s",
						i, tt.expected[i], token.Type)
				}
			}
		})
	}
}
