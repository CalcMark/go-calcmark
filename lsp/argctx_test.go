package lsp

import "testing"

func TestExtractArgumentContext(t *testing.T) {
	cases := []struct {
		name         string
		line         string
		col          int
		wantFunc     string
		wantParam    int
		wantInString bool
	}{
		{
			name:         "double-quote opens string run",
			line:         `x = throughput("`,
			col:          len(`x = throughput("`),
			wantFunc:     "throughput",
			wantParam:    0,
			wantInString: true,
		},
		{
			name:         "single-quote opens string run symmetrically",
			line:         `x = throughput('`,
			col:          len(`x = throughput('`),
			wantFunc:     "throughput",
			wantParam:    0,
			wantInString: true,
		},
		{
			name:      "accumulate first arg",
			line:      `y = accumulate(`,
			col:       len(`y = accumulate(`),
			wantFunc:  "accumulate",
			wantParam: 0,
		},
		{
			name:      "accumulate second arg",
			line:      `y = accumulate(10 MB/s, `,
			col:       len(`y = accumulate(10 MB/s, `),
			wantFunc:  "accumulate",
			wantParam: 1,
		},
		{
			name:      "grow second arg after comma",
			line:      `g = grow(100, `,
			col:       len(`g = grow(100, `),
			wantFunc:  "grow",
			wantParam: 1,
		},
		{
			name:         "nested call inner wins",
			line:         `z = accumulate(convert_rate(10 MB/s, "`,
			col:          len(`z = accumulate(convert_rate(10 MB/s, "`),
			wantFunc:     "convert_rate",
			wantParam:    1,
			wantInString: true,
		},
		{
			name:         "comma inside string content does not advance paramIdx",
			line:         `throughput("gig, `,
			col:          len(`throughput("gig, `),
			wantFunc:     "throughput",
			wantParam:    0, // comma was inside the quoted run
			wantInString: true,
		},
		{
			name:         "single-quote comma also does not advance paramIdx",
			line:         `throughput('gig, `,
			col:          len(`throughput('gig, `),
			wantFunc:     "throughput",
			wantParam:    0,
			wantInString: true,
		},
		{
			name:      "outside any call",
			line:      `x = 1 + `,
			col:       len(`x = 1 + `),
			wantFunc:  "",
			wantParam: -1,
		},
		{
			name:      "unmatched close paren",
			line:      `foo()) `,
			col:       len(`foo()) `),
			wantFunc:  "",
			wantParam: -1,
		},
		{
			name: "backslash is a regular rune, not an escape",
			line: `throughput("a\"b`,
			col:  len(`throughput("a\"b`),
			// After opening `"`, the `a\` is content, the second `"` closes
			// the run. Now back at paren level, `b` is a plain identifier.
			wantFunc:     "throughput",
			wantParam:    0,
			wantInString: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := extractArgumentContext(tc.line, tc.col)
			if ctx.funcName != tc.wantFunc {
				t.Errorf("funcName = %q, want %q", ctx.funcName, tc.wantFunc)
			}
			if ctx.paramIdx != tc.wantParam {
				t.Errorf("paramIdx = %d, want %d", ctx.paramIdx, tc.wantParam)
			}
			if ctx.insideString != tc.wantInString {
				t.Errorf("insideString = %v, want %v", ctx.insideString, tc.wantInString)
			}
		})
	}
}

// TestExtractArgumentContext_DeepNesting covers 3+ levels and commas inside
// completed inner calls to make sure they don't inflate outer paramIdx.
func TestExtractArgumentContext_DeepNesting(t *testing.T) {
	cases := []struct {
		name      string
		line      string
		col       int
		wantFunc  string
		wantParam int
	}{
		{
			name:      "three levels nested, cursor innermost",
			line:      `a(b(c(`,
			col:       len(`a(b(c(`),
			wantFunc:  "c",
			wantParam: 0,
		},
		{
			name:      "outer comma after closed inner call",
			line:      `outer(inner(a, b), `,
			col:       len(`outer(inner(a, b), `),
			wantFunc:  "outer",
			wantParam: 1,
		},
		{
			name:      "outer third arg after two closed inners",
			line:      `outer(i1(1, 2), i2(3, 4), `,
			col:       len(`outer(i1(1, 2), i2(3, 4), `),
			wantFunc:  "outer",
			wantParam: 2,
		},
		{
			name:      "inner first arg, then outer still unaffected",
			line:      `accumulate(convert_rate(r, `,
			col:       len(`accumulate(convert_rate(r, `),
			wantFunc:  "convert_rate",
			wantParam: 1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := extractArgumentContext(tc.line, tc.col)
			if ctx.funcName != tc.wantFunc {
				t.Errorf("funcName = %q, want %q", ctx.funcName, tc.wantFunc)
			}
			if ctx.paramIdx != tc.wantParam {
				t.Errorf("paramIdx = %d, want %d", ctx.paramIdx, tc.wantParam)
			}
		})
	}
}

// TestExtractArgumentContext_NLFallback verifies that natural-language function
// calls (without parens) are detected by the NL fallback path.
func TestExtractArgumentContext_NLFallback(t *testing.T) {
	cases := []struct {
		name      string
		line      string
		col       int
		wantFunc  string
		wantParam int
	}{
		{
			name:      "grow NL first param",
			line:      "grow 100 by 20 over 5 months",
			col:       5, // on the '1' of 100
			wantFunc:  "grow",
			wantParam: 0,
		},
		{
			name:      "grow NL second param",
			line:      "grow 100 by 20 over 5 months",
			col:       13, // on the '2' of 20
			wantFunc:  "grow",
			wantParam: 1,
		},
		{
			name:      "grow NL third param",
			line:      "grow 100 by 20 over 5 months",
			col:       21, // on the '5'
			wantFunc:  "grow",
			wantParam: 2,
		},
		{
			name:      "compound with assignment and currency",
			line:      "goal = compound $1000 by 5% monthly over 10 years",
			col:       16, // on the '$' of $1000
			wantFunc:  "compound",
			wantParam: 0,
		},
		{
			name:      "compound second param with percent",
			line:      "goal = compound $1000 by 5% monthly over 10 years",
			col:       25, // on the '5' of 5%
			wantFunc:  "compound",
			wantParam: 1,
		},
		{
			name:      "compound third param",
			line:      "goal = compound $1000 by 5% monthly over 10 years",
			col:       41, // on the '1' of 10
			wantFunc:  "compound",
			wantParam: 2,
		},
		{
			name:      "synonym average maps to avg",
			line:      "average of 1, 2, 3",
			col:       15, // on the '2'
			wantFunc:  "avg",
			wantParam: 1,
		},
		{
			name:      "plain arithmetic not a function",
			line:      "x = 100 + 200",
			col:       5,
			wantFunc:  "",
			wantParam: -1,
		},
		{
			name:      "paren form still handled by paren scanner",
			line:      "grow(100, 20, 5)",
			col:       5,
			wantFunc:  "grow",
			wantParam: 0,
		},
		{
			name:      "assignment prefix stripped for grow",
			line:      "result = grow 100 by 20 over 5 months",
			col:       14, // on the '1' of 100
			wantFunc:  "grow",
			wantParam: 0,
		},
		{
			name:      "unknown function returns empty",
			line:      "foobar 100 by 200",
			col:       7,
			wantFunc:  "",
			wantParam: -1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := extractArgumentContext(tc.line, tc.col)
			if ctx.funcName != tc.wantFunc {
				t.Errorf("funcName = %q, want %q", ctx.funcName, tc.wantFunc)
			}
			if ctx.paramIdx != tc.wantParam {
				t.Errorf("paramIdx = %d, want %d", ctx.paramIdx, tc.wantParam)
			}
		})
	}
}

// FuzzExtractArgumentContext seeds the backward walker with edge-case inputs
// and asserts that arbitrary byte sequences never panic. The plan commits to
// a fuzz test as the mitigation for walker misclassification risk.
func FuzzExtractArgumentContext(f *testing.F) {
	seeds := []string{
		``,
		`x = throughput(`,
		`x = throughput("gig`,
		`accumulate(10 MB/s, `,
		`grow(100, `,
		`accumulate(convert_rate(10 MB/s, "`,
		`foo()) `,
		`x = 1 + `,
		`throughput("a\"b`,
		`rtt("re`,
		`a(b(c(`,
		`outer(i1(1, 2), i2(3, 4), `,
		string([]byte{0x00, '(', ')'}),
		"δ(ε, ",
	}
	for _, s := range seeds {
		for i := 0; i <= len(s); i++ {
			f.Add(s, i)
		}
	}
	f.Fuzz(func(t *testing.T, line string, col int) {
		_ = extractArgumentContext(line, col)
	})
}
