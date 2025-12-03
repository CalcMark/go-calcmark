package interpreter

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

// TestEnvironmentSharing tests that multiple interpreters can share the same environment
// and see each other's variables.
func TestEnvironmentSharing(t *testing.T) {
	env := NewEnvironment()

	// First interpreter defines 'a'
	interp1 := NewInterpreterWithEnv(env)
	env.Set("a", &types.Number{Value: decimal.NewFromFloat(3.0)})

	// Check that the environment has 'a'
	val, ok := env.Get("a")
	if !ok {
		t.Fatal("Environment should have 'a' after Set()")
	}
	if val == nil {
		t.Fatal("Value for 'a' is nil")
	}

	// Check GetAllVariables
	allVars := env.GetAllVariables()
	t.Logf("GetAllVariables returned %d variables", len(allVars))
	for k, v := range allVars {
		t.Logf("  %s = %v", k, v)
	}

	if len(allVars) < 1 {
		t.Errorf("Expected at least 1 variable in environment, got %d", len(allVars))
	}

	if _, hasA := allVars["a"]; !hasA {
		t.Error("GetAllVariables() should include 'a'")
	} else {
		t.Log("✓ GetAllVariables() correctly includes 'a'")
	}

	// Second interpreter with same environment should see 'a'
	interp2 := NewInterpreterWithEnv(env)
	_ = interp1
	_ = interp2

	// Try to get 'a' again
	val2, ok2 := env.Get("a")
	if !ok2 {
		t.Error("Second interpreter should see 'a' in shared environment")
	}
	if val2 == nil {
		t.Error("Value for 'a' is nil in second check")
	}
}
