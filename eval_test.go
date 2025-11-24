package calcmark

import (
	"testing"
)

// Test basic evaluation
func TestEvalSimple(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "simple addition",
			input: "1 + 1",
			want:  "2",
		},
		{
			name:  "multiplication",
			input: "5 * 3",
			want:  "15",
		},
		{
			name:  "number with multiplier",
			input: "1.2k",
			want:  "1200",
		},
		{
			name:  "currency",
			input: "$100",
			want:  "100 USD", // Parser converts $ → USD, formatter displays as needed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Eval(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Eval() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && result.Value != nil {
				got := result.Value.String()
				if got != tt.want {
					t.Errorf("Eval() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

// Test session with variable persistence
func TestSession(t *testing.T) {
	session := NewSession()

	// Set variable
	result, err := session.Eval("x = 10")
	if err != nil {
		t.Fatalf("session.Eval('x = 10') error = %v", err)
	}
	if result.Value.String() != "10" {
		t.Errorf("Expected 10, got %v", result.Value)
	}

	// Use variable
	result, err = session.Eval("x + 5")
	if err != nil {
		t.Fatalf("session.Eval('x + 5') error = %v", err)
	}
	if result.Value.String() != "15" {
		t.Errorf("Expected 15, got %v", result.Value)
	}

	// Reset session
	session.Reset()

	// Variable should be gone
	result, err = session.Eval("x")
	if err == nil && len(result.Diagnostics) == 0 {
		t.Error("Expected undefined variable error after reset")
	}
}

// Test multi-line evaluation
func TestEvalMultiLine(t *testing.T) {
	input := `x = 10
y = 20
x + y`

	result, err := Eval(input)
	if err != nil {
		t.Fatalf("Eval(multiline) error = %v", err)
	}

	// Should have 3 values
	if len(result.AllValues) != 3 {
		t.Errorf("Expected 3 values, got %d", len(result.AllValues))
	}

	// Last value should be 30
	if result.Value.String() != "30" {
		t.Errorf("Expected final value 30, got %v", result.Value)
	}
}

// Test diagnostics
func TestEvalDiagnostics(t *testing.T) {
	// Undefined variable should produce ERROR diagnostic and block evaluation
	result, _ := Eval("undefined_var")

	// Should have ERROR diagnostic (blocks evaluation)
	if result == nil {
		t.Fatal("Expected result with diagnostics, got nil")
	}

	if len(result.Diagnostics) == 0 {
		t.Error("Expected ERROR diagnostic for undefined variable")
	}

	hasError := false
	for _, d := range result.Diagnostics {
		if d.Severity == Error {
			hasError = true
			break
		}
	}

	if !hasError {
		t.Error("Expected ERROR diagnostic (not warning) for undefined variable")
	}

	// Should NOT have evaluated (blocked by semantic error)
	if result.Value != nil {
		t.Error("Expected no value when semantic errors block evaluation")
	}
}

// Test error handling
func TestEvalErrors(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldBlock bool // Should block interpretation
	}{
		{
			name:        "division by zero",
			input:       "10 / 0",
			shouldBlock: false, // Warning, not error
		},
		{
			name:        "invalid syntax",
			input:       "1 +",
			shouldBlock: true, // Parse error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Eval(tt.input)

			if tt.shouldBlock {
				// Should either return error or have ERROR diagnostics
				if err == nil && !hasErrorDiagnostic(result) {
					t.Error("Expected blocking error")
				}
			}
		})
	}
}

// Helper function
func hasErrorDiagnostic(result *Result) bool {
	if result == nil {
		return false
	}
	for _, d := range result.Diagnostics {
		if d.Severity == Error {
			return true
		}
	}
	return false
}
