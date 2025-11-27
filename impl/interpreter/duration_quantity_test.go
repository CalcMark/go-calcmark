package interpreter

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

// TestDurationQuantityOperations verifies that Duration + Quantity operations produce clear errors
// TDD: These operations should not be supported - they mix incompatible types
func TestDurationPlusQuantity(t *testing.T) {
	duration, _ := types.NewDuration(decimal.NewFromInt(5), "second")
	quantity := &types.Quantity{
		Value: decimal.NewFromInt(10),
		Unit:  "meter",
	}

	env := NewEnvironment()
	env.Set("d", duration)
	env.Set("q", quantity)

	err := Evaluate("d + q", env)
	if err == nil {
		t.Error("Expected error for Duration + Quantity, got none")
	}

	if err != nil && err.Error() != "unsupported operation: *types.Duration + *types.Quantity" {
		t.Logf("Got error: %v", err)
	}
}

func TestQuantityPlusDuration(t *testing.T) {
	duration, _ := types.NewDuration(decimal.NewFromInt(5), "second")
	quantity := &types.Quantity{
		Value: decimal.NewFromInt(10),
		Unit:  "meter",
	}

	env := NewEnvironment()
	env.Set("d", duration)
	env.Set("q", quantity)

	err := Evaluate("q + d", env)
	if err == nil {
		t.Error("Expected error for Quantity + Duration, got none")
	}
}

// TestDurationQuantityRejection ensures various Duration-Quantity mix operations fail properly
func TestDurationQuantityMixedOperations(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{"Duration + Quantity", "5 seconds + 10 meters"},
		{"Quantity + Duration", "10 meters + 5 seconds"},
		{"Duration - Quantity", "5 seconds - 10 meters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := NewEnvironment()
			err := Evaluate(tt.expr, env)
			if err == nil {
				t.Errorf("Expected error for %q, got none", tt.expr)
			}
		})
	}
}
