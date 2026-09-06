package lexer

import "testing"

// `table.column` tokenizes as IDENTIFIER DOT IDENTIFIER so the parser can
// build a MemberAccess (go-calcmark#118). DOT is emitted after an
// identifier only when an identifier-start character follows the dot,
// so decimals and `@globals.field` are untouched.
func TestLexer_DotAfterIdentifier(t *testing.T) {
	cases := []struct {
		src  string
		want []TokenType
	}{
		{"sales.q1", []TokenType{IDENTIFIER, DOT, IDENTIFIER}},
		{"x = rates.rate * rates.hc", []TokenType{IDENTIFIER, ASSIGN, IDENTIFIER, DOT, IDENTIFIER, MULTIPLY, IDENTIFIER, DOT, IDENTIFIER}},
		{"a.b.c", []TokenType{IDENTIFIER, DOT, IDENTIFIER, DOT, IDENTIFIER}},
		{"x = 1.5", []TokenType{IDENTIFIER, ASSIGN, NUMBER}},
		{"@globals.tax_rate", []TokenType{AT_SIGN, IDENTIFIER, DOT, IDENTIFIER}},
	}
	for _, c := range cases {
		toks, err := NewLexer(c.src).Tokenize()
		if err != nil {
			t.Fatalf("Tokenize(%q): %v", c.src, err)
		}
		var got []TokenType
		for _, tk := range toks {
			if tk.Type == EOF || tk.Type == NEWLINE {
				continue
			}
			got = append(got, tk.Type)
		}
		if len(got) != len(c.want) {
			t.Errorf("%q: got %v, want %v", c.src, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%q: token %d = %v, want %v (all: %v)", c.src, i, got[i], c.want[i], got)
				break
			}
		}
	}
}

func TestLexer_DotNotFollowedByIdentifierIsNotMemberAccess(t *testing.T) {
	// A trailing dot (end of sentence) or a dot before a digit must not
	// produce DOT — that would turn prose into a calculation.
	for _, src := range []string{"done.", "a.123"} {
		toks, _ := NewLexer(src).Tokenize()
		for _, tk := range toks {
			if tk.Type == DOT {
				t.Errorf("%q: unexpected DOT token", src)
			}
		}
	}
}
