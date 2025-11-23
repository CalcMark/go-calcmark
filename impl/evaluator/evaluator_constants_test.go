package evaluator

import (
	"testing"
)

// TestMathematicalConstants tests that PI and E are available as constants
func TestMathematicalConstants(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "PI uppercase",
			input: "PI",
			want:  "3.141592653589793",
		},
		{
			name:  "pi lowercase",
			input: "pi",
			want:  "3.141592653589793",
		},
		{
			name:  "E uppercase",
			input: "E",
			want:  "2.718281828459045",
		},
		{
			name:  "e lowercase",
			input: "e",
			want:  "2.718281828459045",
		},
		{
			name:  "PI in expression",
			input: "2 * PI",
			want:  "6.283185307179586",
		},
		{
			name:  "circle area",
			input: "PI * 10 * 10",
			want:  "314.1592653589793",
		},
		{
			name:  "E in expression",
			input: "E * 2",
			want:  "5.43656365691809",
		},
		{
			name:  "PI in function",
			input: "avg(PI, E)",
			want:  "2.929937241024419",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewContext()
			results, err := Evaluate(tt.input, ctx)
			if err != nil {
				t.Fatalf("Evaluate(%q) error = %v, want nil", tt.input, err)
			}

			if len(results) != 1 {
				t.Fatalf("Evaluate(%q) returned %d results, want 1", tt.input, len(results))
			}

			if results[0].String() != tt.want {
				t.Errorf("Evaluate(%q) = %s, want %s", tt.input, results[0].String(), tt.want)
			}
		})
	}
}

// TestConstantsCannotBeAssigned tests that PI and E cannot be used as variable names
func TestConstantsCannotBeAssigned(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError string
	}{
		{
			name:      "cannot assign to PI",
			input:     "PI = 3",
			wantError: "Cannot assign to constant 'PI'",
		},
		{
			name:      "cannot assign to pi",
			input:     "pi = 3.14",
			wantError: "Cannot assign to constant 'pi'",
		},
		{
			name:      "cannot assign to E",
			input:     "E = 2",
			wantError: "Cannot assign to constant 'E'",
		},
		{
			name:      "cannot assign to e",
			input:     "e = 2.71",
			wantError: "Cannot assign to constant 'e'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewContext()
			_, err := Evaluate(tt.input, ctx)

			if err == nil {
				t.Fatalf("Evaluate(%q) expected error, got nil", tt.input)
			}

			if err.Error() != tt.wantError && !contains(err.Error(), tt.wantError) {
				t.Errorf("Evaluate(%q) error = %q, want %q", tt.input, err.Error(), tt.wantError)
			}
		})
	}
}

// TestConstantsInContext tests that constants work alongside user variables
func TestConstantsInContext(t *testing.T) {
	ctx := NewContext()

	// Set a user variable
	_, err := Evaluate("radius = 5", ctx)
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// Use both constant and variable
	results, err := Evaluate("PI * radius * radius", ctx)
	if err != nil {
		t.Fatalf("Evaluate error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	expected := "78.539816339744825"
	if results[0].String() != expected {
		t.Errorf("PI * radius * radius = %s, want %s", results[0].String(), expected)
	}
}
