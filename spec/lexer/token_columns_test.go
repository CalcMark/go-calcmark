package lexer

import "testing"

// Token columns must be the 1-based rune column where the token starts,
// for every token kind and regardless of how much lookahead the lexer
// did before committing to it. Before go-calcmark#164 the duration and
// unit lookaheads rewound pos but not column, so every token after a
// number was shifted right by the number's width — and the drift
// accumulated across the line.
func TestTokenColumns_NoDriftAfterNumbers(t *testing.T) {
	cases := []struct {
		src  string
		want []int // start column of each non-EOF, non-NEWLINE token
	}{
		//         1234567890123456789012
		{"x = grow(100, 20%, 5)", []int{1, 3, 5, 9, 10, 13, 15, 18, 20, 21}},
		{"x = $5 / $2", []int{1, 3, 5, 6, 8, 10, 11}},
		{"x = 10 meters + 2 weeks", []int{1, 3, 5, 15, 17}},
		{"total = 1,000 * 3", []int{1, 7, 9, 15, 17}},
		{"d = today + 2 days", []int{1, 3, 5, 11, 13}},
		{"q = FQ1 + 1", []int{1, 3, 5, 9, 11}},
	}
	for _, c := range cases {
		toks, err := NewLexer(c.src).Tokenize()
		if err != nil {
			t.Fatalf("Tokenize(%q): %v", c.src, err)
		}
		var got []int
		for _, tk := range toks {
			if tk.Type == EOF || tk.Type == NEWLINE {
				continue
			}
			got = append(got, tk.Column)
		}
		if len(got) != len(c.want) {
			t.Errorf("%q: got %d tokens %v, want %d %v", c.src, len(got), got, len(c.want), c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%q: token %d (%s) column = %d, want %d", c.src, i, toks[i].Type, got[i], c.want[i])
			}
		}
	}
}

func TestTokenColumns_ResetPerLine(t *testing.T) {
	toks, err := NewLexer("a = 100\nb = a + 20").Tokenize()
	if err != nil {
		t.Fatal(err)
	}
	var line2 []Token
	for _, tk := range toks {
		if tk.Line == 2 && tk.Type != EOF && tk.Type != NEWLINE {
			line2 = append(line2, tk)
		}
	}
	want := []int{1, 3, 5, 7, 9}
	if len(line2) != len(want) {
		t.Fatalf("line 2 has %d tokens, want %d", len(line2), len(want))
	}
	for i, tk := range line2 {
		if tk.Column != want[i] {
			t.Errorf("line 2 token %d (%s) column = %d, want %d", i, tk.Type, tk.Column, want[i])
		}
	}
}
