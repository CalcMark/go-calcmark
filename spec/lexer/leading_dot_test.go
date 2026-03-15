package lexer

import (
	"testing"
)

func TestLeadingDotDecimalNumbers(t *testing.T) {
	tests := []struct {
		input        string
		expectedType TokenType
		expectedVal  string
	}{
		{".5", NUMBER, ".5"},
		{".25", NUMBER, ".25"},
		{".123456", NUMBER, ".123456"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lex := NewLexer(tt.input)
			tokens, err := lex.Tokenize()
			if err != nil {
				t.Fatalf("Tokenize(%q) error = %v", tt.input, err)
			}

			// Filter out EOF
			var nonEOF []Token
			for _, tok := range tokens {
				if tok.Type != EOF {
					nonEOF = append(nonEOF, tok)
				}
			}

			if len(nonEOF) != 1 {
				t.Fatalf("Tokenize(%q) got %d non-EOF tokens, want 1: %v", tt.input, len(nonEOF), nonEOF)
			}

			tok := nonEOF[0]
			if tok.Type != tt.expectedType {
				t.Errorf("Tokenize(%q) type = %v, want %v", tt.input, tok.Type, tt.expectedType)
			}
			if tok.Value != tt.expectedVal {
				t.Errorf("Tokenize(%q) value = %q, want %q", tt.input, tok.Value, tt.expectedVal)
			}
		})
	}
}

func TestLeadingDotMultipliers(t *testing.T) {
	tests := []struct {
		input        string
		expectedType TokenType
		expectedVal  string
	}{
		{".5k", NUMBER_K, ".5k"},
		{".5M", NUMBER_M, ".5M"},
		{".5B", NUMBER_B, ".5B"},
		{".5T", NUMBER_T, ".5T"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lex := NewLexer(tt.input)
			tokens, err := lex.Tokenize()
			if err != nil {
				t.Fatalf("Tokenize(%q) error = %v", tt.input, err)
			}

			var nonEOF []Token
			for _, tok := range tokens {
				if tok.Type != EOF {
					nonEOF = append(nonEOF, tok)
				}
			}

			if len(nonEOF) != 1 {
				t.Fatalf("Tokenize(%q) got %d non-EOF tokens, want 1: %v", tt.input, len(nonEOF), nonEOF)
			}

			tok := nonEOF[0]
			if tok.Type != tt.expectedType {
				t.Errorf("Tokenize(%q) type = %v, want %v", tt.input, tok.Type, tt.expectedType)
			}
			if tok.Value != tt.expectedVal {
				t.Errorf("Tokenize(%q) value = %q, want %q", tt.input, tok.Value, tt.expectedVal)
			}
		})
	}
}

func TestLeadingDotPercentage(t *testing.T) {
	lex := NewLexer(".5%")
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatalf("Tokenize(.5%%) error = %v", err)
	}

	var nonEOF []Token
	for _, tok := range tokens {
		if tok.Type != EOF {
			nonEOF = append(nonEOF, tok)
		}
	}

	if len(nonEOF) != 1 {
		t.Fatalf("got %d non-EOF tokens, want 1: %v", len(nonEOF), nonEOF)
	}

	if nonEOF[0].Type != NUMBER_PERCENT {
		t.Errorf("type = %v, want NUMBER_PERCENT", nonEOF[0].Type)
	}
}

func TestLeadingDotScientific(t *testing.T) {
	lex := NewLexer(".5e3")
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatalf("Tokenize(.5e3) error = %v", err)
	}

	var nonEOF []Token
	for _, tok := range tokens {
		if tok.Type != EOF {
			nonEOF = append(nonEOF, tok)
		}
	}

	if len(nonEOF) != 1 {
		t.Fatalf("got %d non-EOF tokens, want 1: %v", len(nonEOF), nonEOF)
	}

	if nonEOF[0].Type != NUMBER_SCI {
		t.Errorf("type = %v, want NUMBER_SCI", nonEOF[0].Type)
	}
}

func TestLeadingDotWithUnit(t *testing.T) {
	tests := []struct {
		input        string
		expectedType TokenType
	}{
		{".5 tomatoes", QUANTITY},
		{".5kg", QUANTITY},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			lex := NewLexer(tt.input)
			tokens, err := lex.Tokenize()
			if err != nil {
				t.Fatalf("Tokenize(%q) error = %v", tt.input, err)
			}

			var nonEOF []Token
			for _, tok := range tokens {
				if tok.Type != EOF {
					nonEOF = append(nonEOF, tok)
				}
			}

			if len(nonEOF) < 1 {
				t.Fatalf("Tokenize(%q) returned no tokens", tt.input)
			}

			if nonEOF[0].Type != tt.expectedType {
				t.Errorf("Tokenize(%q) type = %v, want %v", tt.input, nonEOF[0].Type, tt.expectedType)
			}
		})
	}
}

func TestLeadingDotWithTimeUnit(t *testing.T) {
	// .5 days should produce QUANTITY (same as 0.5 days — duration lookahead
	// only catches integer values, so decimal durations are quantities)
	lex := NewLexer(".5 days")
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatalf("Tokenize(.5 days) error = %v", err)
	}

	var nonEOF []Token
	for _, tok := range tokens {
		if tok.Type != EOF {
			nonEOF = append(nonEOF, tok)
		}
	}

	if len(nonEOF) != 1 {
		t.Fatalf("got %d non-EOF tokens, want 1: %v", len(nonEOF), nonEOF)
	}

	if nonEOF[0].Type != QUANTITY {
		t.Errorf("type = %v, want QUANTITY (matching 0.5 days behavior)", nonEOF[0].Type)
	}
}

func TestLeadingDotDivisionNotFraction(t *testing.T) {
	// .5/2 should be NUMBER + DIVIDE + NUMBER, not FRACTION
	lex := NewLexer(".5/2")
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatalf("Tokenize(.5/2) error = %v", err)
	}

	var nonEOF []Token
	for _, tok := range tokens {
		if tok.Type != EOF {
			nonEOF = append(nonEOF, tok)
		}
	}

	if len(nonEOF) != 3 {
		t.Fatalf("got %d non-EOF tokens, want 3 (NUMBER DIVIDE NUMBER): %v", len(nonEOF), nonEOF)
	}

	if nonEOF[0].Type != NUMBER {
		t.Errorf("first token type = %v, want NUMBER", nonEOF[0].Type)
	}
	if nonEOF[1].Type != DIVIDE {
		t.Errorf("second token type = %v, want DIVIDE", nonEOF[1].Type)
	}
	if nonEOF[2].Type != NUMBER {
		t.Errorf("third token type = %v, want NUMBER", nonEOF[2].Type)
	}
}

func TestLeadingDotEdgeCases(t *testing.T) {
	// Bare dot should remain an error
	t.Run("bare dot", func(t *testing.T) {
		lex := NewLexer(".")
		_, err := lex.Tokenize()
		if err == nil {
			t.Error("expected error for bare dot '.', got nil")
		}
	})

	// Double dot should remain an error
	t.Run("double dot", func(t *testing.T) {
		lex := NewLexer("..5")
		_, err := lex.Tokenize()
		if err == nil {
			t.Error("expected error for '..5', got nil")
		}
	})
}
