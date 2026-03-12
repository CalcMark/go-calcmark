package semantic

import "testing"

func TestHintForDiagnostic(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		message string
		want    string
	}{
		{
			name:    "undefined variable with quoted name",
			code:    DiagUndefinedVariable,
			message: `Undefined variable "budget" - not found`,
			want:    "Define it above: budget = <value>",
		},
		{
			name:    "undefined variable without quotes",
			code:    DiagUndefinedVariable,
			message: "Undefined variable",
			want:    "Define the variable before using it",
		},
		{
			name:    "division by zero",
			code:    DiagDivisionByZero,
			message: "Division by zero",
			want:    "Check that divisor is not zero",
		},
		{
			name:    "incompatible units",
			code:    DiagIncompatibleUnits,
			message: "Cannot add kg and meters",
			want:    "Units must be compatible for this operation",
		},
		{
			name:    "type mismatch",
			code:    DiagTypeMismatch,
			message: "Cannot compare date and number",
			want:    "Check that values are compatible types",
		},
		{
			name:    "parse error",
			code:    DiagParseError,
			message: "Unexpected token",
			want:    "Check syntax - see error message for details",
		},
		{
			name:    "invalid currency code",
			code:    DiagInvalidCurrencyCode,
			message: "XXX is not a known currency",
			want:    "Use a valid 3-letter currency code (e.g., USD, EUR)",
		},
		{
			name:    "frontmatter validation with valid categories",
			code:    DiagFrontmatterValidation,
			message: `frontmatter: invalid unit category "Weight" in scale.unit_categories; valid categories: All, Area, Currency, Custom`,
			want:    "valid categories: All, Area, Currency, Custom",
		},
		{
			name:    "frontmatter validation generic",
			code:    DiagFrontmatterValidation,
			message: "frontmatter: unexpected key 'foo'",
			want:    "Check frontmatter YAML syntax",
		},
		{
			name:    "unknown code returns empty",
			code:    "some_unknown_code",
			message: "Something happened",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HintForDiagnostic(tt.code, tt.message)
			if got != tt.want {
				t.Errorf("HintForDiagnostic(%q, %q) = %q, want %q", tt.code, tt.message, got, tt.want)
			}
		})
	}
}

func TestExtractQuoted(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`Undefined variable "budget"`, "budget"},
		{`No quotes here`, ""},
		{`One "quote only`, ""},
		{`"first" and "second"`, "first"},
	}

	for _, tt := range tests {
		got := extractQuoted(tt.input)
		if got != tt.want {
			t.Errorf("extractQuoted(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
