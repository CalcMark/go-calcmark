package interpreter

import (
	"errors"
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/ast"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
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

func TestEnvironmentDefinedLines(t *testing.T) {
	// Tracking definition line numbers lets the LSP filter completions
	// to variables defined ABOVE the cursor — calcmark is strictly
	// ordered (text/markdown interpolation is the one exception, handled
	// elsewhere). Without this, an editor offers `flour` as a completion
	// on a line that references it before its assignment.

	t.Run("SetDefinedLine + GetAllDefinedLines round-trip", func(t *testing.T) {
		env := NewEnvironment()
		env.Set("price", &types.Number{Value: decimal.NewFromInt(100)})
		env.SetDefinedLine("price", 5)

		lines := env.GetAllDefinedLines()
		if got := lines["price"]; got != 5 {
			t.Errorf("definedLines[price] = %d, want 5", got)
		}
	})

	t.Run("returned map is a copy, not the live map", func(t *testing.T) {
		env := NewEnvironment()
		env.SetDefinedLine("a", 1)
		lines := env.GetAllDefinedLines()
		lines["a"] = 999
		got := env.GetAllDefinedLines()
		if got["a"] != 1 {
			t.Errorf("mutating returned map leaked back into env: got %d, want 1", got["a"])
		}
	})

	t.Run("variables without a recorded line do not appear in the map", func(t *testing.T) {
		// Built-in constants (PI, E) are added by addConstants without a
		// source line — they should not be filterable by position.
		env := NewEnvironment()
		lines := env.GetAllDefinedLines()
		if _, has := lines["PI"]; has {
			t.Error("PI has a definition line, want none (built-in constant)")
		}
	})

	t.Run("Clone preserves definedLines", func(t *testing.T) {
		env := NewEnvironment()
		env.SetDefinedLine("x", 7)
		clone := env.Clone()
		got := clone.GetAllDefinedLines()
		if got["x"] != 7 {
			t.Errorf("clone definedLines[x] = %d, want 7", got["x"])
		}
	})
}

func TestEvalAssignmentRecordsDefinedLine(t *testing.T) {
	// Bug repro: typing on line 1 of `grow $100 by flour over 10 \n flour = 10`
	// would offer `flour` as a completion even though `flour` is defined
	// AFTER the cursor. The LSP filters via `definedLines` returned by
	// the Environment; the interpreter populates it from each
	// Assignment node's Range plus the block's doc-line offset.

	t.Run("evalAssignment records line at offset 0 by default", func(t *testing.T) {
		// `flour = 10` on line 2 of a single-block doc → definedLines[flour] = 1 (0-indexed).
		interp := NewInterpreter()
		stmt := &ast.Assignment{
			Name:  "flour",
			Value: &ast.NumberLiteral{Value: "10"},
			Range: &ast.Range{
				Start: ast.Position{Line: 2, Column: 1},
				End:   ast.Position{Line: 2, Column: 11},
			},
		}
		_, err := interp.Eval([]ast.Node{stmt})
		if err != nil {
			t.Fatalf("eval: %v", err)
		}
		lines := interp.GetEnvironment().GetAllDefinedLines()
		if got, want := lines["flour"], 1; got != want {
			t.Errorf("definedLines[flour] = %d, want %d (0-indexed line 2)", got, want)
		}
	})

	t.Run("evalAssignment respects SetLineOffset for block-relative AST", func(t *testing.T) {
		// A calc block sitting on doc lines 5-7 has block-relative
		// AST line numbers 1-3. `SetLineOffset(4)` shifts them to
		// doc-absolute 0-indexed.
		interp := NewInterpreter()
		interp.SetLineOffset(4)
		stmt := &ast.Assignment{
			Name:  "flour",
			Value: &ast.NumberLiteral{Value: "10"},
			Range: &ast.Range{
				Start: ast.Position{Line: 2, Column: 1},
				End:   ast.Position{Line: 2, Column: 11},
			},
		}
		_, err := interp.Eval([]ast.Node{stmt})
		if err != nil {
			t.Fatalf("eval: %v", err)
		}
		lines := interp.GetEnvironment().GetAllDefinedLines()
		// Block-relative line 2 + offset 4 = doc-absolute 1-indexed line 6
		// → 0-indexed line 5.
		if got, want := lines["flour"], 5; got != want {
			t.Errorf("definedLines[flour] = %d, want %d (block-line 2 + offset 4, 0-indexed)", got, want)
		}
	})
}
