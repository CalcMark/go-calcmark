package lexer

import (
	"testing"
)

func TestDirective_AtScale(t *testing.T) {
	lex := NewLexer("@scale")
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Expect: AT_SIGN, IDENTIFIER("scale"), NEWLINE, EOF
	filtered := filterMeaningful(tokens)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 meaningful tokens, got %d: %v", len(filtered), filtered)
	}
	if filtered[0].Type != AT_SIGN {
		t.Errorf("token 0: expected AT_SIGN, got %s", filtered[0].Type)
	}
	if filtered[1].Type != IDENTIFIER || filtered[1].Value != "scale" {
		t.Errorf("token 1: expected IDENTIFIER(scale), got %s(%s)", filtered[1].Type, filtered[1].Value)
	}
}

func TestDirective_AtGlobalsField(t *testing.T) {
	lex := NewLexer("@globals.tax_rate")
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Expect: AT_SIGN, IDENTIFIER("globals"), DOT, IDENTIFIER("tax_rate"), NEWLINE, EOF
	filtered := filterMeaningful(tokens)
	if len(filtered) != 4 {
		t.Fatalf("expected 4 meaningful tokens, got %d: %v", len(filtered), filtered)
	}
	if filtered[0].Type != AT_SIGN {
		t.Errorf("token 0: expected AT_SIGN, got %s", filtered[0].Type)
	}
	if filtered[1].Type != IDENTIFIER || filtered[1].Value != "globals" {
		t.Errorf("token 1: expected IDENTIFIER(globals), got %s(%s)", filtered[1].Type, filtered[1].Value)
	}
	if filtered[2].Type != DOT {
		t.Errorf("token 2: expected DOT, got %s", filtered[2].Type)
	}
	if filtered[3].Type != IDENTIFIER || filtered[3].Value != "tax_rate" {
		t.Errorf("token 3: expected IDENTIFIER(tax_rate), got %s(%s)", filtered[3].Type, filtered[3].Value)
	}
}

func TestDirective_AtAlone_Error(t *testing.T) {
	lex := NewLexer("@")
	_, err := lex.Tokenize()
	if err == nil {
		t.Fatal("expected error for bare '@', got nil")
	}
}

func TestDirective_AtNumber_Error(t *testing.T) {
	lex := NewLexer("@123")
	_, err := lex.Tokenize()
	if err == nil {
		t.Fatal("expected error for '@123', got nil")
	}
}

func TestDirective_NestedDots(t *testing.T) {
	// @globals.a.b should tokenize (parser will reject the second dot)
	lex := NewLexer("@globals.a.b")
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	filtered := filterMeaningful(tokens)
	// AT_SIGN, IDENTIFIER("globals"), DOT, IDENTIFIER("a"), DOT, IDENTIFIER("b")
	if len(filtered) != 6 {
		t.Fatalf("expected 6 meaningful tokens, got %d: %v", len(filtered), filtered)
	}
	if filtered[4].Type != DOT {
		t.Errorf("token 4: expected DOT, got %s", filtered[4].Type)
	}
	if filtered[5].Type != IDENTIFIER || filtered[5].Value != "b" {
		t.Errorf("token 5: expected IDENTIFIER(b), got %s(%s)", filtered[5].Type, filtered[5].Value)
	}
}

func TestDirective_DecimalUnchanged(t *testing.T) {
	// 3.14 must still parse as a NUMBER, not emit DOT
	lex := NewLexer("3.14")
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	filtered := filterMeaningful(tokens)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 meaningful token, got %d: %v", len(filtered), filtered)
	}
	if filtered[0].Type != NUMBER {
		t.Errorf("expected NUMBER, got %s", filtered[0].Type)
	}
	if filtered[0].Value != "3.14" {
		t.Errorf("expected value 3.14, got %s", filtered[0].Value)
	}
}

func TestDirective_InExpression(t *testing.T) {
	// x = 1 + @scale * 2
	lex := NewLexer("x = 1 + @scale * 2")
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	filtered := filterMeaningful(tokens)
	// IDENTIFIER(x), ASSIGN, NUMBER(1), PLUS, AT_SIGN, IDENTIFIER(scale), MULTIPLY, NUMBER(2)
	if len(filtered) != 8 {
		t.Fatalf("expected 8 meaningful tokens, got %d: %v", len(filtered), filtered)
	}
	if filtered[4].Type != AT_SIGN {
		t.Errorf("token 4: expected AT_SIGN, got %s", filtered[4].Type)
	}
	if filtered[5].Type != IDENTIFIER || filtered[5].Value != "scale" {
		t.Errorf("token 5: expected IDENTIFIER(scale), got %s(%s)", filtered[5].Type, filtered[5].Value)
	}
}

func TestDirective_AtGlobalsNoDot_Tokenizes(t *testing.T) {
	// @globals without dot should still tokenize (parser will reject)
	lex := NewLexer("@globals + 1")
	tokens, err := lex.Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	filtered := filterMeaningful(tokens)
	// AT_SIGN, IDENTIFIER("globals"), PLUS, NUMBER(1)
	if len(filtered) != 4 {
		t.Fatalf("expected 4 meaningful tokens, got %d: %v", len(filtered), filtered)
	}
	if filtered[0].Type != AT_SIGN {
		t.Errorf("token 0: expected AT_SIGN, got %s", filtered[0].Type)
	}
	if filtered[1].Type != IDENTIFIER || filtered[1].Value != "globals" {
		t.Errorf("token 1: expected IDENTIFIER(globals), got %s(%s)", filtered[1].Type, filtered[1].Value)
	}
}

// filterMeaningful strips NEWLINE and EOF tokens.
func filterMeaningful(tokens []Token) []Token {
	var result []Token
	for _, t := range tokens {
		if t.Type != NEWLINE && t.Type != EOF {
			result = append(result, t)
		}
	}
	return result
}
