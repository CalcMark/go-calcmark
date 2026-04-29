package lexer_test

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/lexer"
)

// U6 — new period-operator keywords for v2.0:
//   - `between`           → BETWEEN          (single-word reserved)
//   - `length of`         → LENGTH_OF        (multi-word, after `end of` pattern)
//   - `days in`           → DAYS_IN          (multi-word)
//
// `between` becomes a v2.0 breaking change: any user variable named
// `between` will fail to parse. Migration diagnostic for that case
// lives in U7 (parser).
//
// `to` stays as a ContextualKeyword (growth-NL: `depreciate ... to
// 5000`); the period synonym `from A to B` is parser-side
// disambiguation in U7, not a lexer keyword.

func tokenize(t *testing.T, input string) []lexer.Token {
	t.Helper()
	l := lexer.NewLexer(input)
	tokens, err := l.Tokenize()
	if err != nil {
		t.Fatalf("Tokenize(%q): %v", input, err)
	}
	return tokens
}

// firstToken returns tokens[0] for a single-token input. Skips the
// trailing EOF that NewLexer / Tokenize always appends.
func firstToken(t *testing.T, input string) lexer.Token {
	t.Helper()
	tokens := tokenize(t, input)
	if len(tokens) < 1 {
		t.Fatalf("Tokenize(%q) returned 0 tokens", input)
	}
	return tokens[0]
}

func TestLexer_BetweenKeyword(t *testing.T) {
	cases := []string{"between", "BETWEEN", "Between"}
	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			got := firstToken(t, input)
			if got.Type != lexer.BETWEEN {
				t.Errorf("Tokenize(%q)[0].Type = %v, want BETWEEN", input, got.Type)
			}
		})
	}
}

func TestLexer_LengthOfKeyword(t *testing.T) {
	cases := []string{"length of", "Length Of", "LENGTH OF"}
	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			got := firstToken(t, input)
			if got.Type != lexer.LENGTH_OF {
				t.Errorf("Tokenize(%q)[0].Type = %v, want LENGTH_OF", input, got.Type)
			}
		})
	}
}

func TestLexer_DaysInKeyword(t *testing.T) {
	cases := []string{"days in", "Days In", "DAYS IN"}
	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			got := firstToken(t, input)
			if got.Type != lexer.DAYS_IN {
				t.Errorf("Tokenize(%q)[0].Type = %v, want DAYS_IN", input, got.Type)
			}
		})
	}
}

// TestLexer_NewKeywordsRequireWordBoundary — the new keywords must
// not match prefixes of longer identifiers. A user variable named
// `betweens` or `lengths` continues to lex as IDENTIFIER (other
// migration concerns aside; `between` itself is the v2.0 break).
func TestLexer_NewKeywordsRequireWordBoundary(t *testing.T) {
	cases := []struct {
		input   string
		notType lexer.TokenType
	}{
		{"betweens", lexer.BETWEEN},
		{"betwixt", lexer.BETWEEN},
		{"lengths", lexer.LENGTH_OF},
		{"lengthof", lexer.LENGTH_OF}, // no space → not the multi-word keyword
		{"daysin", lexer.DAYS_IN},     // no space → not the multi-word keyword
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := firstToken(t, tc.input)
			if got.Type == tc.notType {
				t.Errorf("Tokenize(%q)[0].Type = %v, must NOT be %v (word-boundary)",
					tc.input, got.Type, tc.notType)
			}
		})
	}
}

// TestLexer_BackwardCompat_DurationFromKeyword — the existing
// `2 days from today` flow must keep working. `days` here is part
// of the duration literal, not the start of `days in`.
func TestLexer_BackwardCompat_DurationFromKeyword(t *testing.T) {
	tokens := tokenize(t, "2 days from today")
	// Expected: DURATION_LITERAL, FROM, DATE_TODAY, EOF
	if len(tokens) < 3 {
		t.Fatalf("expected at least 3 tokens, got %d: %+v", len(tokens), tokens)
	}
	if tokens[0].Type != lexer.DURATION_LITERAL {
		t.Errorf("tokens[0].Type = %v, want DURATION_LITERAL", tokens[0].Type)
	}
	if tokens[1].Type != lexer.FROM {
		t.Errorf("tokens[1].Type = %v, want FROM", tokens[1].Type)
	}
	if tokens[2].Type != lexer.DATE_TODAY {
		t.Errorf("tokens[2].Type = %v, want DATE_TODAY", tokens[2].Type)
	}
	for _, tok := range tokens {
		if tok.Type == lexer.DAYS_IN {
			t.Errorf("`days from today` must not lex as DAYS_IN; got %+v", tok)
		}
	}
}

// TestLexer_BackwardCompat_LengthAlone — `length` without `of`
// stays as an IDENTIFIER. Avoids accidentally consuming user
// variables / function names that happen to start with `length`.
func TestLexer_BackwardCompat_LengthAlone(t *testing.T) {
	tokens := tokenize(t, "length")
	if len(tokens) < 1 {
		t.Fatalf("expected at least 1 token")
	}
	if tokens[0].Type == lexer.LENGTH_OF {
		t.Errorf("bare `length` must not lex as LENGTH_OF; got %+v", tokens[0])
	}
}

// TestLexer_BetweenIsReserved — once between is a keyword, it
// classifies as a reserved keyword token (used by the parser to
// surface the `between = 50` migration diagnostic).
func TestLexer_BetweenIsReserved(t *testing.T) {
	tok := firstToken(t, "between")
	if !lexer.IsReservedKeywordToken(tok.Type) {
		t.Errorf("BETWEEN should be a reserved keyword token (so parser can flag `between = 50` as a migration error)")
	}
}
