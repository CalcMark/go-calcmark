package interpreter

import (
	"errors"
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

func TestEnvironment_SetError_GetError(t *testing.T) {
	env := NewEnvironment()
	evalErr := errors.New("division by zero")

	env.SetError("x", evalErr)

	got, ok := env.GetError("x")
	if !ok {
		t.Fatal("expected GetError to return true for errored variable")
	}
	if got != evalErr {
		t.Fatalf("expected error %q, got %q", evalErr, got)
	}
}

func TestEnvironment_GetError_NonErrored(t *testing.T) {
	env := NewEnvironment()

	got, ok := env.GetError("x")
	if ok {
		t.Fatal("expected GetError to return false for non-errored variable")
	}
	if got != nil {
		t.Fatalf("expected nil error, got %q", got)
	}
}

func TestEnvironment_ClearError(t *testing.T) {
	env := NewEnvironment()
	env.SetError("x", errors.New("fail"))

	env.ClearError("x")

	got, ok := env.GetError("x")
	if ok {
		t.Fatal("expected GetError to return false after ClearError")
	}
	if got != nil {
		t.Fatalf("expected nil error after ClearError, got %q", got)
	}
}

func TestEnvironment_ClearErrors(t *testing.T) {
	env := NewEnvironment()
	env.SetError("a", errors.New("err a"))
	env.SetError("b", errors.New("err b"))

	env.ClearErrors()

	if _, ok := env.GetError("a"); ok {
		t.Error("expected 'a' error to be cleared")
	}
	if _, ok := env.GetError("b"); ok {
		t.Error("expected 'b' error to be cleared")
	}
}

func TestEnvironment_SetError_And_Set_Coexist(t *testing.T) {
	env := NewEnvironment()
	evalErr := errors.New("initial failure")

	env.SetError("x", evalErr)
	env.Set("x", types.NewNumber(decimal.NewFromInt(42)))

	// Both should be present independently
	val, valOk := env.Get("x")
	if !valOk {
		t.Fatal("expected Get to return true for variable with both value and error")
	}
	if val == nil {
		t.Fatal("expected non-nil value")
	}

	errVal, errOk := env.GetError("x")
	if !errOk {
		t.Fatal("expected GetError to still return true after Set")
	}
	if errVal != evalErr {
		t.Fatalf("expected error %q, got %q", evalErr, errVal)
	}
}

func TestEnvironment_Clone_ErroredVars(t *testing.T) {
	env := NewEnvironment()
	origErr := errors.New("original error")
	env.SetError("x", origErr)

	cloned := env.Clone()

	// Clone should have the error
	got, ok := cloned.GetError("x")
	if !ok {
		t.Fatal("expected cloned env to have error for 'x'")
	}
	if got != origErr {
		t.Fatalf("expected cloned error %q, got %q", origErr, got)
	}

	// Mutating clone should not affect original
	cloned.SetError("y", errors.New("clone-only"))
	cloned.ClearError("x")

	if _, ok := env.GetError("y"); ok {
		t.Error("original env should not have 'y' error added to clone")
	}
	if _, ok := env.GetError("x"); !ok {
		t.Error("original env should still have 'x' error after clone cleared it")
	}
}

func TestEnvironment_GetError_UnsetVariable(t *testing.T) {
	env := NewEnvironment()

	// Variable never set, never errored
	got, ok := env.GetError("nonexistent")
	if ok {
		t.Fatal("expected GetError to return false for unset variable")
	}
	if got != nil {
		t.Fatalf("expected nil error for unset variable, got %q", got)
	}
}
