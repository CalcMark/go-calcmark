package evaluator

import (
	"testing"
)

// TestFunctionsMixedUnits tests that functions with mixed currency units return Number (no units)
func TestFunctionsMixedUnits(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		want        string
		wantType    string
		description string
	}{
		{
			name:        "avg mixed dollar and euro",
			input:       "avg($100, €200)",
			want:        "150",
			wantType:    "Number",
			description: "Mixed units returns Number with no unit",
		},
		{
			name:        "avg three different currencies",
			input:       "avg($100, €200, £300)",
			want:        "200",
			wantType:    "Number",
			description: "Three different units returns Number",
		},
		{
			name:        "avg dollar euro yen pound",
			input:       "avg($100, €100, ¥100, £100)",
			want:        "100",
			wantType:    "Number",
			description: "Four different units returns Number",
		},
		{
			name:        "avg mixed with regular number",
			input:       "avg($100, 200, €300)",
			want:        "200",
			wantType:    "Number",
			description: "Mixed currency and number returns Number",
		},
		{
			name:        "avg same currency returns that currency",
			input:       "avg($100, $200, $300)",
			want:        "$200.00",
			wantType:    "Currency",
			description: "Same currency preserves the unit",
		},
		{
			name:        "avg all euros returns euro",
			input:       "avg(€100, €200)",
			want:        "€150.00",
			wantType:    "Currency",
			description: "All euros preserves euro unit",
		},
		{
			name:        "sqrt mixed not applicable",
			input:       "sqrt(100)",
			want:        "10",
			wantType:    "Number",
			description: "sqrt always operates on numbers",
		},
		{
			name:        "average of mixed units",
			input:       "average of $50, €100, £150",
			want:        "100",
			wantType:    "Number",
			description: "Multi-token function with mixed units returns Number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Log(tt.description)

			ctx := NewContext()
			results, err := Evaluate(tt.input, ctx)
			if err != nil {
				t.Fatalf("Evaluate(%q) error = %v, want nil", tt.input, err)
			}

			if len(results) != 1 {
				t.Fatalf("Evaluate(%q) returned %d results, want 1", tt.input, len(results))
			}

			result := results[0]
			if result.TypeName() != tt.wantType {
				t.Errorf("Evaluate(%q) type = %s, want %s", tt.input, result.TypeName(), tt.wantType)
			}

			if result.String() != tt.want {
				t.Errorf("Evaluate(%q) = %s, want %s", tt.input, result.String(), tt.want)
			}
		})
	}
}

// TestFunctionsMixedUnitsInAssignments tests mixed units in variable assignments
func TestFunctionsMixedUnitsInAssignments(t *testing.T) {
	ctx := NewContext()

	// Assign result of mixed-unit average
	_, err := Evaluate("total = avg($100, €200, £300)", ctx)
	if err != nil {
		t.Fatalf("Assignment failed: %v", err)
	}

	// Retrieve the variable
	value, err := ctx.Get("total")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if value.TypeName() != "Number" {
		t.Errorf("Variable type = %s, want Number", value.TypeName())
	}

	if value.String() != "200" {
		t.Errorf("Variable value = %s, want 200", value.String())
	}
}

// TestFunctionsMixedUnitsWithThousands tests that thousands separators work with mixed units
func TestFunctionsMixedUnitsWithThousands(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		want     string
		wantType string
	}{
		{
			name:     "mixed with thousands",
			input:    "avg($1,000, €2,000, £3,000)",
			want:     "2000",
			wantType: "Number",
		},
		{
			name:     "same currency with thousands",
			input:    "avg($1,000, $2,000)",
			want:     "$1500.00",
			wantType: "Currency",
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

			if results[0].TypeName() != tt.wantType {
				t.Errorf("Evaluate(%q) type = %s, want %s", tt.input, results[0].TypeName(), tt.wantType)
			}

			if results[0].String() != tt.want {
				t.Errorf("Evaluate(%q) = %s, want %s", tt.input, results[0].String(), tt.want)
			}
		})
	}
}
