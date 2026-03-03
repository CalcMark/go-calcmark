package lexer

import (
	"testing"
)

// TestMultiplierTokens tests that the lexer correctly recognizes multiplier suffixes
// as distinct token types (NUMBER_K, NUMBER_M, NUMBER_B, NUMBER_T, NUMBER_PERCENT, NUMBER_SCI)
func TestMultiplierTokens(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedType  TokenType
		expectedValue string
	}{
		{
			name:          "12k",
			input:         "12k",
			expectedType:  NUMBER_K,
			expectedValue: "12k",
		},
		{
			name:          "1.2K",
			input:         "1.2K",
			expectedType:  NUMBER_K,
			expectedValue: "1.2K",
		},
		{
			name:          "1.2M",
			input:         "1.2M",
			expectedType:  NUMBER_M,
			expectedValue: "1.2M",
		},
		{
			name:          "5B",
			input:         "5B",
			expectedType:  NUMBER_B,
			expectedValue: "5B",
		},
		{
			name:          "2.5T",
			input:         "2.5T",
			expectedType:  NUMBER_T,
			expectedValue: "2.5T",
		},
		{
			name:          "20%",
			input:         "20%",
			expectedType:  NUMBER_PERCENT,
			expectedValue: "20%",
		},
		{
			name:          "1.2e10",
			input:         "1.2e10",
			expectedType:  NUMBER_SCI,
			expectedValue: "1.2e10",
		},
		{
			name:          "4.5e-7",
			input:         "4.5e-7",
			expectedType:  NUMBER_SCI,
			expectedValue: "4.5e-7",
		},
		{
			name:          "1.2E+5",
			input:         "1.2E+5",
			expectedType:  NUMBER_SCI,
			expectedValue: "1.2E+5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer(tt.input) // Fixed: use tt.input
			tokens, err := l.Tokenize()
			if err != nil {
				t.Fatalf("Tokenize() error = %v", err)
			}

			// Should have 2 tokens: the number and EOF
			if len(tokens) != 2 {
				t.Fatalf("Expected 2 tokens, got %d: %v", len(tokens), tokens)
			}

			tok := tokens[0]
			if tok.Type != tt.expectedType {
				t.Errorf("Token type = %v, want %v", tok.Type, tt.expectedType)
			}
			if tok.Value != tt.expectedValue {
				t.Errorf("Token value = %q, want %q", tok.Value, tt.expectedValue)
			}
		})
	}
}

// TestMultiplierPrefixedUnits tests that units starting with multiplier characters
// (e.g., kg, km, kW) are NOT incorrectly consumed as multipliers.
// Bug: https://github.com/CalcMark/go-calcmark/issues/14
func TestMultiplierPrefixedUnits(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedType  TokenType
		expectedValue string
	}{
		{
			name:          "2kg should be quantity not multiplier",
			input:         "2kg",
			expectedType:  QUANTITY,
			expectedValue: "2:kg",
		},
		{
			name:          "5km should be quantity not multiplier",
			input:         "5km",
			expectedType:  QUANTITY,
			expectedValue: "5:km",
		},
		{
			name:          "10kW should be quantity not multiplier",
			input:         "10kW",
			expectedType:  QUANTITY,
			expectedValue: "10:kW",
		},
		{
			name:          "3kJ should be quantity not multiplier",
			input:         "3kJ",
			expectedType:  QUANTITY,
			expectedValue: "3:kJ",
		},
		{
			name:          "standalone 2k is still a multiplier",
			input:         "2k",
			expectedType:  NUMBER_K,
			expectedValue: "2k",
		},
		{
			name:          "standalone 5K is still a multiplier",
			input:         "5K",
			expectedType:  NUMBER_K,
			expectedValue: "5K",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer(tt.input)
			tokens, err := l.Tokenize()
			if err != nil {
				t.Fatalf("Tokenize() error = %v", err)
			}

			if len(tokens) < 1 {
				t.Fatalf("Expected at least 1 token, got %d", len(tokens))
			}

			tok := tokens[0]
			if tok.Type != tt.expectedType {
				t.Errorf("Token type = %v, want %v (tokens: %v)", tok.Type, tt.expectedType, tokens)
			}
			if tok.Value != tt.expectedValue {
				t.Errorf("Token value = %q, want %q", tok.Value, tt.expectedValue)
			}
		})
	}
}

// TestMultiplierVsUnit tests the critical distinction between multipliers (no space)
// and units (with space). This is essential for the CalcMark language.
func TestMultiplierVsUnit(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		tokens []struct {
			typ TokenType
			val string
		}
	}{
		{
			name:  "1.1k K (1100 Kelvin)",
			input: "1.1k K",
			tokens: []struct {
				typ TokenType
				val string
			}{
				{NUMBER_K, "1.1k"}, // 1100 (multiplier, no space)
				{IDENTIFIER, "K"},  // Kelvin unit (with space)
				{EOF, ""},
			},
		},
		{
			name:  "12k meters",
			input: "12k meters",
			tokens: []struct {
				typ TokenType
				val string
			}{
				{NUMBER_K, "12k"},      // 12000 (multiplier)
				{IDENTIFIER, "meters"}, // unit
				{EOF, ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLexer(tt.input) // Fixed: use tt.input
			tokens, err := l.Tokenize()
			if err != nil {
				t.Fatalf("Tokenize() error = %v", err)
			}

			if len(tokens) != len(tt.tokens) {
				t.Fatalf("Expected %d tokens, got %d: %v", len(tt.tokens), len(tokens), tokens)
			}

			for i, expected := range tt.tokens {
				if tokens[i].Type != expected.typ {
					t.Errorf("Token %d: type = %v, want %v", i, tokens[i].Type, expected.typ)
				}
				if tokens[i].Value != expected.val {
					t.Errorf("Token %d: value = %q, want %q", i, tokens[i].Value, expected.val)
				}
			}
		})
	}
}
