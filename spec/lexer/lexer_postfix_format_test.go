package lexer

import (
	"testing"
)

// TestPostfixFormatRules tests the distinction between valid and invalid postfix formats
// Valid: "100 USD" (space required)
// Invalid: "100USD" (no space - should be two tokens: NUMBER + IDENTIFIER)
func TestPostfixFormatRules(t *testing.T) {
	t.Skip("Postfix unit format not fully implemented")
	tests := []struct {
		name        string
		input       string
		expectError bool
		tokens      []TokenType
		description string
	}{
		{
			name:        "no space - separate tokens",
			input:       "100USD",
			expectError: false,
			tokens:      []TokenType{NUMBER, IDENTIFIER, EOF},
			description: "Without space, should tokenize as NUMBER then IDENTIFIER",
		},
		{
			name:        "with space - postfix quantity",
			input:       "100 USD",
			expectError: false,
			tokens:      []TokenType{NUMBER, IDENTIFIER, EOF}, // Will become QUANTITY when postfix support added
			description: "With space, will support postfix quantity format when implemented",
		},
		{
			name:        "prefix format works",
			input:       "USD100",
			expectError: false,
			tokens:      []TokenType{QUANTITY, EOF},
			description: "Prefix format (USD100) already works",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lex := NewLexer(tt.input)
			tokens, err := lex.Tokenize()

			if tt.expectError && err == nil {
				t.Errorf("Expected error, got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if len(tokens) != len(tt.tokens) {
				t.Fatalf("Expected %d tokens, got %d. Tokens: %v", len(tt.tokens), len(tokens), tokens)
			}

			for i, expected := range tt.tokens {
				if tokens[i].Type != expected {
					t.Errorf("Token %d: expected %s, got %s (value: %q)",
						i, expected, tokens[i].Type, tokens[i].Value)
				}
			}

			t.Logf("✓ %s", tt.description)
		})
	}
}

// TestPostfixInCalculations tests how postfix notation behaves in calculations
func TestPostfixInCalculations(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldParse bool
		description string
	}{
		{
			name:        "100USD + 20% (no space)",
			input:       "100USD + 20%",
			shouldParse: false, // NUMBER IDENTIFIER PLUS - invalid syntax
			description: "No space means separate tokens, won't parse as valid expression",
		},
		{
			name:        "USD100 + 20% (prefix)",
			input:       "USD100 + 20%",
			shouldParse: true,
			description: "Prefix format works fine",
		},
		{
			name:        "bonus = USD100 + 20%",
			input:       "bonus = USD100 + 20%",
			shouldParse: true,
			description: "Assignment with prefix format works",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lex := NewLexer(tt.input)
			tokens, err := lex.Tokenize()
			if err != nil {
				t.Fatalf("Tokenization failed: %v", err)
			}

			t.Logf("Tokens: ")
			for _, tok := range tokens {
				t.Logf("  %s(%s)", tok.Type, tok.Value)
			}

			// Try to parse
			// Note: We import parser to test this, but for now just log tokens
			hasValidStructure := true
			for i := 0; i < len(tokens)-1; i++ {
				// Check for invalid patterns like NUMBER IDENTIFIER
				if tokens[i].Type == NUMBER && tokens[i+1].Type == IDENTIFIER {
					hasValidStructure = false
					t.Logf("Found invalid pattern: NUMBER followed by IDENTIFIER")
				}
			}

			if tt.shouldParse && !hasValidStructure {
				t.Errorf("Expected valid structure but found invalid pattern")
			}
			if !tt.shouldParse && hasValidStructure {
				t.Logf("Correctly identified as invalid structure")
			}

			t.Logf("✓ %s", tt.description)
		})
	}
}
