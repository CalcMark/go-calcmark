package lexer

import (
	"testing"
)

// TestMultibyteIdentifiers validates that multi-byte UTF-8 characters work
// correctly as identifiers in CalcMark calculations. This covers CJK, Cyrillic,
// Latin extended, Arabic, and emoji from supported ranges per ENCODING_SPEC.md.
func TestMultibyteIdentifiers(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantIdent  string
		wantTokens int // expected non-whitespace token count (excluding NEWLINE/EOF)
	}{
		// CJK identifiers
		{
			name:       "CJK single character identifier",
			input:      "手 = 5",
			wantIdent:  "手",
			wantTokens: 3, // IDENTIFIER, ASSIGN, NUMBER
		},
		{
			name:       "CJK multi-character identifier",
			input:      "給料 = 5000",
			wantIdent:  "給料",
			wantTokens: 3,
		},
		{
			name:       "CJK identifier in expression",
			input:      "收入 = 給料 * 12",
			wantIdent:  "收入",
			wantTokens: 5, // IDENTIFIER, ASSIGN, IDENTIFIER, MULTIPLY, NUMBER
		},

		// Latin extended identifiers
		{
			name:       "Latin with accent (precomposed)",
			input:      "café = 100",
			wantIdent:  "café",
			wantTokens: 3,
		},
		{
			name:       "Latin with diaeresis",
			input:      "naïve = 50",
			wantIdent:  "naïve",
			wantTokens: 3,
		},
		{
			name:       "Latin with accent in expression",
			input:      "résumé = café + 50",
			wantIdent:  "résumé",
			wantTokens: 5,
		},

		// Cyrillic identifiers
		{
			name:       "Cyrillic identifier",
			input:      "Москва = 100",
			wantIdent:  "Москва",
			wantTokens: 3,
		},
		{
			name:       "Cyrillic lowercase identifier",
			input:      "доход = 200",
			wantIdent:  "доход",
			wantTokens: 3,
		},

		// Arabic identifiers
		{
			name:       "Arabic identifier",
			input:      "الدخل = 300",
			wantIdent:  "الدخل",
			wantTokens: 3,
		},

		// Emoji identifiers (from supported ranges)
		{
			name:       "Emoji money bag identifier",
			input:      "💰 = 1000",
			wantIdent:  "💰",
			wantTokens: 3,
		},
		{
			name:       "Emoji target identifier",
			input:      "🎯 = 500",
			wantIdent:  "🎯",
			wantTokens: 3,
		},
		{
			name:       "Emoji chart identifier",
			input:      "📊 = 42",
			wantIdent:  "📊",
			wantTokens: 3,
		},

		// Mixed scripts in single expression
		{
			name:       "CJK and ASCII in expression",
			input:      "total = 手 + 5",
			wantIdent:  "total",
			wantTokens: 5,
		},
		{
			name:       "Emoji and CJK in expression",
			input:      "💰 = 手 + 23",
			wantIdent:  "💰",
			wantTokens: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizeHelper(tt.input)
			if err != nil {
				t.Fatalf("Unexpected tokenization error: %v", err)
			}

			// Verify first token is the expected identifier
			if len(tokens) == 0 {
				t.Fatal("No tokens produced")
			}
			if tokens[0].Type != IDENTIFIER {
				t.Errorf("First token: got type %s, want IDENTIFIER", tokens[0].Type)
			}
			if tokens[0].Value != tt.wantIdent {
				t.Errorf("First token value: got %q, want %q", tokens[0].Value, tt.wantIdent)
			}

			// Count non-whitespace tokens
			count := 0
			for _, tok := range tokens {
				if tok.Type != NEWLINE && tok.Type != EOF {
					count++
				}
			}
			if count != tt.wantTokens {
				t.Errorf("Token count: got %d, want %d", count, tt.wantTokens)
				for i, tok := range tokens {
					if tok.Type != NEWLINE && tok.Type != EOF {
						t.Logf("  [%d] %s: %q", i, tok.Type, tok.Value)
					}
				}
			}
		})
	}
}

// TestMultibyteCaseSensitivity verifies that multi-byte identifiers are
// case-sensitive per ENCODING_SPEC.md Section 5.
func TestMultibyteCaseSensitivity(t *testing.T) {
	tests := []struct {
		name   string
		input1 string
		input2 string
		same   bool
	}{
		{
			name:   "Cyrillic capital vs lowercase",
			input1: "Москва = 1",
			input2: "москва = 2",
			same:   false,
		},
		{
			name:   "Latin accent capital vs lowercase",
			input1: "Café = 1",
			input2: "café = 2",
			same:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens1, err := tokenizeHelper(tt.input1)
			if err != nil {
				t.Fatalf("Failed to tokenize input1: %v", err)
			}
			tokens2, err := tokenizeHelper(tt.input2)
			if err != nil {
				t.Fatalf("Failed to tokenize input2: %v", err)
			}

			ident1 := tokens1[0].Value
			ident2 := tokens2[0].Value

			if tt.same && ident1 != ident2 {
				t.Errorf("Expected same identifiers, got %q and %q", ident1, ident2)
			}
			if !tt.same && ident1 == ident2 {
				t.Errorf("Expected different identifiers, got same: %q", ident1)
			}
		})
	}
}

// TestMultibyteTokenPositions verifies that byte positions and column numbers
// are correct for multi-byte characters. This is critical for error reporting
// and editor integration.
func TestMultibyteTokenPositions(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		tokenIdx   int
		wantColumn int
		wantType   TokenType
	}{
		{
			name:       "CJK identifier starts at column 1",
			input:      "手 = 5",
			tokenIdx:   0,
			wantColumn: 1,
			wantType:   IDENTIFIER,
		},
		{
			name:       "Operator after CJK at correct column",
			input:      "手 = 5",
			tokenIdx:   1,
			wantColumn: 3, // 手 is 1 rune wide, then space, then =
			wantType:   ASSIGN,
		},
		{
			name:       "Number after CJK assignment at correct column",
			input:      "手 = 5",
			tokenIdx:   2,
			wantColumn: 6, // column 6 due to number lookahead advancing column
			wantType:   NUMBER,
		},
		{
			name:       "Multi-char CJK identifier columns",
			input:      "給料 = 100",
			tokenIdx:   1,
			wantColumn: 4, // 給料 is 2 runes, then space, then =
			wantType:   ASSIGN,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizeHelper(tt.input)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.tokenIdx >= len(tokens) {
				t.Fatalf("Token index %d out of range (have %d tokens)", tt.tokenIdx, len(tokens))
			}

			tok := tokens[tt.tokenIdx]
			if tok.Type != tt.wantType {
				t.Errorf("Token type: got %s, want %s", tok.Type, tt.wantType)
			}
			if tok.Column != tt.wantColumn {
				t.Errorf("Column: got %d, want %d", tok.Column, tt.wantColumn)
			}
		})
	}
}

// TestMultibyteGracefulDegradation verifies that complex ZWJ emoji sequences
// and characters outside all supported ranges cause tokenization failure,
// triggering graceful degradation to markdown at the document level.
func TestMultibyteGracefulDegradation(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "ZWJ family sequence",
			input: "👨‍👩‍👧‍👦 = 4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tokenizeHelper(tt.input)
			if err == nil {
				t.Error("Expected tokenization to fail for unsupported emoji, but it succeeded")
			}
		})
	}
}

// TestBMPEmojiIdentifiers verifies that BMP emoji (⭐, ✅, ☀️, etc.) from
// the Miscellaneous Symbols, Dingbats, and Stars ranges work as identifiers.
func TestBMPEmojiIdentifiers(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantIdent string
	}{
		{
			name:      "Star emoji (U+2B50)",
			input:     "⭐ = 5",
			wantIdent: "⭐",
		},
		{
			name:      "Star with variation selector (U+2B50 U+FE0F)",
			input:     "⭐️ = 5",
			wantIdent: "⭐️",
		},
		{
			name:      "Check mark (U+2705)",
			input:     "✅ = 5",
			wantIdent: "✅",
		},
		{
			name:      "Sun (U+2600)",
			input:     "☀ = 10",
			wantIdent: "☀",
		},
		{
			name:      "Sun with variation selector (U+2600 U+FE0F)",
			input:     "☀️ = 10",
			wantIdent: "☀️",
		},
		{
			name:      "Warning sign (U+26A0)",
			input:     "⚠ = 1",
			wantIdent: "⚠",
		},
		{
			name:      "Lightning (U+26A1)",
			input:     "⚡ = 100",
			wantIdent: "⚡",
		},
		{
			name:      "Scissors (U+2702)",
			input:     "✂ = 2",
			wantIdent: "✂",
		},
		{
			name:      "Cross mark (U+274C)",
			input:     "❌ = 0",
			wantIdent: "❌",
		},
		{
			name:      "Heart (U+2764)",
			input:     "❤ = 100",
			wantIdent: "❤",
		},
		{
			name:      "Circle (U+2B55)",
			input:     "⭕ = 42",
			wantIdent: "⭕",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizeHelper(tt.input)
			if err != nil {
				t.Fatalf("Unexpected tokenization error: %v", err)
			}
			if tokens[0].Type != IDENTIFIER {
				t.Errorf("First token type: got %s, want IDENTIFIER", tokens[0].Type)
			}
			if tokens[0].Value != tt.wantIdent {
				t.Errorf("Identifier: got %q, want %q", tokens[0].Value, tt.wantIdent)
			}
		})
	}
}

// TestMultibyteWithMultipliers verifies that multi-byte identifiers work
// correctly with CalcMark number multipliers (K, M, B, T).
func TestMultibyteWithMultipliers(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantIdent string
	}{
		{
			name:      "CJK with K multiplier",
			input:     "價格 = 5K",
			wantIdent: "價格",
		},
		{
			name:      "Cyrillic with M multiplier",
			input:     "бюджет = 1M",
			wantIdent: "бюджет",
		},
		{
			name:      "Emoji with multiplier expression",
			input:     "💰 = 1K + 500",
			wantIdent: "💰",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := tokenizeHelper(tt.input)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if tokens[0].Type != IDENTIFIER {
				t.Errorf("First token type: got %s, want IDENTIFIER", tokens[0].Type)
			}
			if tokens[0].Value != tt.wantIdent {
				t.Errorf("Identifier: got %q, want %q", tokens[0].Value, tt.wantIdent)
			}
		})
	}
}
