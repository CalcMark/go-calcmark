package lexer

import "testing"

func TestTryParseFraction(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		pos         int
		integerPart string
		wantDenom   string
		wantConsum  int
		wantOk      bool
	}{
		{"simple 1/3", "/3", 0, "1", "3", 2, true},
		{"simple 7/8", "/8", 0, "7", "8", 2, true},
		{"multi-digit denom", "/12", 0, "1", "12", 3, true},
		{"multi-digit both", "/300", 0, "100", "300", 4, true},
		{"zero numerator", "/3", 0, "0", "3", 2, true},
		{"zero denominator", "/0", 0, "1", "0", 2, true}, // lexer allows; semantic checker catches
		{"decimal numerator rejected", "/3", 0, "1.5", "", 0, false},
		{"no slash", "3", 0, "1", "", 0, false},
		{"space before denom", "/ 3", 0, "1", "", 0, false},
		{"followed by ident", "/3cup", 0, "1", "", 0, false},
		{"followed by slash", "/3/4", 0, "1", "", 0, false},
		{"at end of input", "/3", 0, "1", "3", 2, true},
		{"comma-separated numerator ok", "/3", 0, "1000", "3", 2, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runes := []rune(tt.text)
			gotDenom, gotConsum, gotOk := tryParseFraction(runes, tt.pos, tt.integerPart)
			if gotOk != tt.wantOk {
				t.Errorf("ok: got %v, want %v", gotOk, tt.wantOk)
			}
			if gotDenom != tt.wantDenom {
				t.Errorf("denom: got %q, want %q", gotDenom, tt.wantDenom)
			}
			if gotConsum != tt.wantConsum {
				t.Errorf("consumed: got %d, want %d", gotConsum, tt.wantConsum)
			}
		})
	}
}

func TestLexerFractionToken(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantTypes  []TokenType
		wantValues []string
	}{
		{
			"fraction literal",
			"1/3",
			[]TokenType{FRACTION, EOF},
			[]string{"1/3", ""},
		},
		{
			"division with spaces",
			"1 / 3",
			[]TokenType{NUMBER, DIVIDE, NUMBER, EOF},
			[]string{"1", "/", "3", ""},
		},
		{
			"fraction in expression",
			"a = 1/3",
			[]TokenType{IDENTIFIER, ASSIGN, FRACTION, EOF},
			[]string{"a", "=", "1/3", ""},
		},
		{
			"fraction with large numbers",
			"100/3",
			[]TokenType{FRACTION, EOF},
			[]string{"100/3", ""},
		},
		{
			"zero denominator allowed by lexer",
			"1/0",
			[]TokenType{FRACTION, EOF},
			[]string{"1/0", ""},
		},
		{
			"rate unchanged",
			"100 MB/s",
			// 100 MB is a QUANTITY, /s triggers rate parsing
			nil, // don't check exact types, just ensure no FRACTION
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lex := NewLexer(tt.input)
			tokens, err := lex.Tokenize()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantTypes == nil {
				// Just check no FRACTION token
				for _, tok := range tokens {
					if tok.Type == FRACTION {
						t.Errorf("unexpected FRACTION token in %q", tt.input)
					}
				}
				return
			}

			if len(tokens) != len(tt.wantTypes) {
				var gotTypes []string
				for _, tok := range tokens {
					gotTypes = append(gotTypes, tok.Type.String())
				}
				t.Fatalf("token count: got %d %v, want %d", len(tokens), gotTypes, len(tt.wantTypes))
			}

			for i, tok := range tokens {
				if tok.Type != tt.wantTypes[i] {
					t.Errorf("token[%d] type: got %s, want %s", i, tok.Type, tt.wantTypes[i])
				}
				if tt.wantValues[i] != "" && tok.Value != tt.wantValues[i] {
					t.Errorf("token[%d] value: got %q, want %q", i, tok.Value, tt.wantValues[i])
				}
			}
		})
	}
}
