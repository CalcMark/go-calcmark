package lexer

import (
	"testing"
)

// TestEmojiWithModifiers tests what happens when emoji with skin tone modifiers
// are used in identifiers. This is important for WASM/web users who might paste
// emoji with modifiers into CalcMark documents.
//
// IMPORTANT: When tokenization fails (e.g., ZWJ sequences, unsupported emoji),
// the classifier treats the line as MARKDOWN. This is by design and allows
// emoji to work naturally in prose while providing clear boundaries for calculations.
func TestEmojiWithModifiers(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		description string
	}{
		{
			name:        "Thumbs up with light skin tone as variable",
			input:       "👍🏻 = 100",
			expectError: false,
			description: "U+1F44D (thumbs up) + U+1F3FB (light skin tone modifier) - SUPPORTED",
		},
		{
			name:        "Money bag emoji as variable (baseline)",
			input:       "💰 = 100",
			expectError: false,
			description: "U+1F4B0 (money bag) - single codepoint emoji - SUPPORTED",
		},
		{
			name:        "Wave with medium skin tone",
			input:       "👋🏽 = 50",
			expectError: false,
			description: "U+1F44B (waving hand) + U+1F3FD (medium skin tone) - SUPPORTED",
		},
		{
			name:        "Family emoji (ZWJ sequence)",
			input:       "👨‍👩‍👧‍👦 = 4",
			expectError: true,
			description: "Complex ZWJ sequence (man + woman + girl + boy) - NOT SUPPORTED, becomes markdown",
		},
		{
			name:        "Emoji in markdown context",
			input:       "Here's a thumbs up: 👍🏻",
			expectError: true,
			description: "Smart quotes not supported - becomes markdown (emoji modifier is fine)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizeHelper(tt.input)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none. Description: %s", tt.description)
				t.Logf("Tokens: %v", tokens)
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v. Description: %s", err, tt.description)
			}

			// Log what actually happened for inspection
			if err == nil {
				t.Logf("✓ Successfully tokenized: %s", tt.description)
				t.Logf("  Tokens: %v", tokens)
				for i, tok := range tokens {
					t.Logf("    [%d] Type=%s, Value=%q", i, tok.Type, tok.Value)
				}
			} else {
				t.Logf("✗ Tokenization failed: %s", tt.description)
				t.Logf("  Error: %v", err)
			}
		})
	}
}

// TestEmojiModifiersInCalculations tests if calculations with emoji modifiers
// can parse and evaluate correctly
func TestEmojiModifiersInCalculations(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
	}{
		{
			name: "Basic assignment with emoji modifier",
			lines: []string{
				"👍🏻 = 100",
				"👍🏻 + 50",
			},
		},
		{
			name: "Multiple emoji modifiers",
			lines: []string{
				"👍🏻 = 100",
				"👍🏿 = 200",
				"👍🏻 + 👍🏿",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i, line := range tt.lines {
				tokens, err := tokenizeHelper(line)
				t.Logf("Line %d: %q", i+1, line)
				if err != nil {
					t.Logf("  Error: %v", err)
				} else {
					t.Logf("  Tokens:")
					for j, tok := range tokens {
						if tok.Type != NEWLINE && tok.Type != EOF {
							t.Logf("    [%d] %s: %q", j, tok.Type, tok.Value)
						}
					}
				}
			}
		})
	}
}

// TestEmojiVariationSelectors tests emoji with variation selectors (text vs emoji presentation).
// These common emoji (⭐ ✅ ☀️) are in the supported BMP emoji ranges and should
// tokenize successfully as identifiers in calculations.
func TestEmojiVariationSelectors(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		note      string
		wantIdent string
	}{
		{
			name:      "Star emoji presentation",
			input:     "⭐ = 5",
			note:      "U+2B50 (star) - in Stars and Circles range",
			wantIdent: "⭐",
		},
		{
			name:      "Sun with variation selector",
			input:     "☀️ = 1",
			note:      "U+2600 (sun) + U+FE0F (variation selector-16) - in Miscellaneous Symbols range",
			wantIdent: "☀️",
		},
		{
			name:      "Check mark emoji",
			input:     "✅ = 5",
			note:      "U+2705 (check mark) - in Dingbats range",
			wantIdent: "✅",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizeHelper(tt.input)
			t.Logf("Input: %q", tt.input)
			t.Logf("Note: %s", tt.note)

			if err != nil {
				t.Fatalf("Expected tokenization to succeed, but got error: %v", err)
			}

			if tokens[0].Type != IDENTIFIER {
				t.Errorf("First token type: got %s, want IDENTIFIER", tokens[0].Type)
			}
			if tokens[0].Value != tt.wantIdent {
				t.Errorf("Identifier: got %q, want %q", tokens[0].Value, tt.wantIdent)
			}

			t.Logf("Tokens:")
			for i, tok := range tokens {
				if tok.Type != NEWLINE && tok.Type != EOF {
					t.Logf("  [%d] %s: %q (len=%d)", i, tok.Type, tok.Value, len(tok.Value))
				}
			}
		})
	}
}
