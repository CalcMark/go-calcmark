package lsp

import "testing"

func TestBuildNLExampleSnippet(t *testing.T) {
	cases := []struct {
		name     string
		example  string
		expected string
	}{
		{
			name:     "grow — bare numeric tokens",
			example:  "grow 100 by 20 over 5",
			expected: "grow ${1:100} by ${2:20} over ${3:5}",
		},
		{
			name:     "currency prefix is absorbed into the placeholder",
			example:  "compound $1000 by 5% over 10 years",
			expected: "compound ${1:$1000} by ${2:5%} over ${3:10} years",
		},
		{
			name:     "percent suffix is absorbed into the placeholder",
			example:  "grow 100 by 5% over 12",
			expected: "grow ${1:100} by ${2:5%} over ${3:12}",
		},
		{
			name:     "decimal numbers",
			example:  "compound 1000 by 5.5% over 10 years",
			expected: "compound ${1:1000} by ${2:5.5%} over ${3:10} years",
		},
		{
			name:     "variadic example — every value gets its own tab stop",
			example:  "sum of $100, $200, $300",
			expected: "sum of ${1:$100}, ${2:$200}, ${3:$300}",
		},
		{
			name:     "single value",
			example:  "square root of 16",
			expected: "square root of ${1:16}",
		},
		{
			name:     "no numeric tokens — returned unchanged",
			example:  "today",
			expected: "today",
		},
		{
			name:     "trailing words preserved",
			example:  "depreciate $50000 by 15% over 5 years to $5000",
			expected: "depreciate ${1:$50000} by ${2:15%} over ${3:5} years to ${4:$5000}",
		},
		{
			name:     "euro currency prefix",
			example:  "compound €1000 by 5% over 10",
			expected: "compound ${1:€1000} by ${2:5%} over ${3:10}",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildNLExampleSnippet(tc.example)
			if got != tc.expected {
				t.Errorf("buildNLExampleSnippet(%q):\n  got  %q\n  want %q", tc.example, got, tc.expected)
			}
		})
	}
}
