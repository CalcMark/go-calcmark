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
		wantPrefix   string
	}{
		{
			name:      "empty throughput string arg",
			line:      `x = throughput("`,
			col:       len(`x = throughput("`),
			wantFunc:  "throughput",
			wantParam: 0,
			wantInString: true,
			wantPrefix:   "",
		},
		{
			name:      "partial throughput value",
			line:      `x = throughput("gig`,
			col:       len(`x = throughput("gig`),
			wantFunc:  "throughput",
			wantParam: 0,
			wantInString: true,
			wantPrefix:   "gig",
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
			name:      "nested call inner wins",
			line:      `z = accumulate(convert_rate(10 MB/s, "`,
			col:       len(`z = accumulate(convert_rate(10 MB/s, "`),
			wantFunc:  "convert_rate",
			wantParam: 1,
			wantInString: true,
			wantPrefix:   "",
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
			name:      "escaped quote still in string",
			line:      `throughput("a\"b`,
			col:       len(`throughput("a\"b`),
			wantFunc:  "throughput",
			wantParam: 0,
			wantInString: true,
			wantPrefix:   `a\"b`,
		},
		{
			name:      "rtt partial scope",
			line:      `r = rtt("re`,
			col:       len(`r = rtt("re`),
			wantFunc:  "rtt",
			wantParam: 0,
			wantInString: true,
			wantPrefix:   "re",
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
			if ctx.stringPrefix != tc.wantPrefix {
				t.Errorf("stringPrefix = %q, want %q", ctx.stringPrefix, tc.wantPrefix)
			}
		})
	}
}

// TestExtractFunctionContext_BackwardCompat ensures the adapter still returns
// the legacy (name, paramIdx) tuple correctly.
func TestExtractFunctionContext_BackwardCompatStillWorks(t *testing.T) {
	name, idx := extractFunctionContext(`accumulate(10 MB/s, `, len(`accumulate(10 MB/s, `))
	if name != "accumulate" || idx != 1 {
		t.Errorf("extractFunctionContext = (%q, %d), want (accumulate, 1)", name, idx)
	}
}
