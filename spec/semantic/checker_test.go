package semantic

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/ast"
)

// TestEnvironment tests the Environment type
func TestEnvironment(t *testing.T) {
	env := NewEnvironment()

	// Test Set and Get
	env.Set("x", nil)
	if !env.Has("x") {
		t.Error("Expected variable 'x' to be defined")
	}

	_, ok := env.Get("x")
	if !ok {
		t.Error("Expected Get to return true for defined variable")
	}

	// Test undefined variable
	if env.Has("y") {
		t.Error("Expected variable 'y' to be undefined")
	}

	_, ok = env.Get("y")
	if ok {
		t.Error("Expected Get to return false for undefined variable")
	}
}

// TestEnvironmentClone tests environment cloning
func TestEnvironmentClone(t *testing.T) {
	env := NewEnvironment()
	env.Set("x", nil)

	cloned := env.Clone()
	if !cloned.Has("x") {
		t.Error("Cloned environment should have 'x'")
	}

	// Modifying clone shouldn't affect original
	cloned.Set("y", nil)
	if env.Has("y") {
		t.Error("Original environment shouldn't have 'y'")
	}
}

// TestEnvironmentBuiltinConstants verifies PI and E are pre-defined in
// the semantic environment so static analysis doesn't flag them as
// undefined. Mirrors the runtime interpreter's TestBuiltinConstants.
func TestEnvironmentBuiltinConstants(t *testing.T) {
	env := NewEnvironment()

	for _, name := range []string{"PI", "E"} {
		if !env.Has(name) {
			t.Errorf("expected built-in constant %q to be defined in a fresh semantic environment", name)
		}
		val, ok := env.Get(name)
		if !ok {
			t.Errorf("Get(%q) returned ok=false; expected true", name)
			continue
		}
		if val == nil {
			t.Errorf("Get(%q) returned nil type; expected a number type", name)
		}
	}
}

// TestEnvironmentBuiltinConstantsCloneIncludesConstants ensures the
// constants survive Clone — scoped environments must still resolve
// built-ins.
func TestEnvironmentBuiltinConstantsCloneIncludesConstants(t *testing.T) {
	cloned := NewEnvironment().Clone()
	if !cloned.Has("PI") || !cloned.Has("E") {
		t.Error("cloned semantic environment should still contain built-in constants PI and E")
	}
}

// TestCurrencyValidation tests currency code validation
func TestCurrencyValidation(t *testing.T) {
	tests := []struct {
		code    string
		isValid bool
	}{
		{"USD", true},
		{"EUR", true},
		{"GBP", true},
		{"JPY", true},
		{"$", true},     // Symbol
		{"€", true},     // Symbol
		{"XYZ", false},  // Invalid code
		{"usd", false},  // Lowercase
		{"US", false},   // Too short
		{"USDA", false}, // Too long
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			got := ValidateCurrencyCode(tt.code)
			if got != tt.isValid {
				t.Errorf("ValidateCurrencyCode(%q) = %v, want %v", tt.code, got, tt.isValid)
			}
		})
	}
}

// TestNormalizeCurrencySymbol tests currency symbol normalization
func TestNormalizeCurrencySymbol(t *testing.T) {
	tests := []struct {
		input        string
		wantCode     string
		wantIsSymbol bool
	}{
		{"$", "USD", true},
		{"€", "EUR", true},
		{"£", "GBP", true},
		{"¥", "JPY", true},
		{"USD", "USD", false},
		{"GBP", "GBP", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			code, isSymbol := NormalizeCurrencySymbol(tt.input)
			if code != tt.wantCode {
				t.Errorf("NormalizeCurrencySymbol(%q) code = %v, want %v", tt.input, code, tt.wantCode)
			}
			if isSymbol != tt.wantIsSymbol {
				t.Errorf("NormalizeCurrencySymbol(%q) isSymbol = %v, want %v", tt.input, isSymbol, tt.wantIsSymbol)
			}
		})
	}
}

// TestUndefinedVariable tests undefined variable detection
func TestUndefinedVariable(t *testing.T) {
	checker := NewChecker()

	// Reference undefined variable
	id := &ast.Identifier{
		Name:  "x",
		Range: &ast.Range{},
	}

	expr := &ast.Expression{
		Expr:  id,
		Range: &ast.Range{},
	}

	diagnostics := checker.Check([]ast.Node{expr})

	if len(diagnostics) != 1 {
		t.Fatalf("Expected 1 diagnostic, got %d", len(diagnostics))
	}

	d := diagnostics[0]
	if d.Code != DiagUndefinedVariable {
		t.Errorf("Expected diagnostic code %s, got %s", DiagUndefinedVariable, d.Code)
	}

	if d.Severity != Error {
		t.Errorf("Expected ERROR severity, got %s", d.Severity)
	}
}

// TestDefinedVariable tests that defined variables don't produce diagnostics
func TestDefinedVariable(t *testing.T) {
	checker := NewChecker()

	// Define a variable
	assignment := &ast.Assignment{
		Name:  "x",
		Value: &ast.NumberLiteral{Value: "42"},
		Range: &ast.Range{},
	}

	// Reference the variable
	id := &ast.Identifier{
		Name:  "x",
		Range: &ast.Range{},
	}

	expr := &ast.Expression{
		Expr:  id,
		Range: &ast.Range{},
	}

	diagnostics := checker.Check([]ast.Node{assignment, expr})

	if len(diagnostics) > 0 {
		t.Errorf("Expected no diagnostics for defined variable, got %d", len(diagnostics))
	}
}

// TestBooleanKeywords tests that boolean keywords don't trigger undefined variable warnings
func TestBooleanKeywords(t *testing.T) {
	// Only lowercase true/false are boolean keywords
	keywords := []string{"true", "false"}

	for _, kw := range keywords {
		t.Run(kw, func(t *testing.T) {
			checker := NewChecker()

			id := &ast.Identifier{
				Name:  kw,
				Range: &ast.Range{},
			}

			expr := &ast.Expression{
				Expr:  id,
				Range: &ast.Range{},
			}

			diagnostics := checker.Check([]ast.Node{expr})

			if len(diagnostics) > 0 {
				t.Errorf("Expected no diagnostics for boolean keyword %q, got %d", kw, len(diagnostics))
			}
		})
	}
}

// TestDivisionByZero tests division by zero detection
func TestDivisionByZero(t *testing.T) {
	checker := NewChecker()

	// 10 / 0
	binaryOp := &ast.BinaryOp{
		Operator: "/",
		Left:     &ast.NumberLiteral{Value: "10"},
		Right:    &ast.NumberLiteral{Value: "0"},
		Range:    &ast.Range{},
	}

	expr := &ast.Expression{
		Expr:  binaryOp,
		Range: &ast.Range{},
	}

	diagnostics := checker.Check([]ast.Node{expr})

	if len(diagnostics) != 1 {
		t.Fatalf("Expected 1 diagnostic, got %d", len(diagnostics))
	}

	d := diagnostics[0]
	if d.Code != DiagDivisionByZero {
		t.Errorf("Expected diagnostic code %s, got %s", DiagDivisionByZero, d.Code)
	}

	if d.Severity != Warning {
		t.Errorf("Expected WARNING severity, got %s", d.Severity)
	}
}

// TestUnsupportedUnit removed - quantity literals are now valid
// Unit compatibility is checked during operations, not at parse time

// TestFunctionCallArgumentValidation tests that function arguments are checked
func TestFunctionCallArgumentValidation(t *testing.T) {
	checker := NewChecker()

	// avg(x, y) where x and y are undefined
	funcCall := &ast.FunctionCall{
		Name: "avg",
		Arguments: []ast.Node{
			&ast.Identifier{Name: "x", Range: &ast.Range{}},
			&ast.Identifier{Name: "y", Range: &ast.Range{}},
		},
		Range: &ast.Range{},
	}

	expr := &ast.Expression{
		Expr:  funcCall,
		Range: &ast.Range{},
	}

	diagnostics := checker.Check([]ast.Node{expr})

	// Should have at least 1 diagnostic (arguments are checked)
	if len(diagnostics) < 1 {
		t.Fatal("Expected at least 1 diagnostic for undefined variables in function call")
	}

	// All diagnostics should be for undefined variables
	for _, d := range diagnostics {
		if d.Code != DiagUndefinedVariable {
			t.Errorf("Expected diagnostic code %s, got %s", DiagUndefinedVariable, d.Code)
		}
	}
}

// TestLineOffsetInRedefinitionMessage verifies that when a line offset is set
// (e.g., because frontmatter precedes the calc block), the "first defined at
// line N" message uses the document-absolute line number, not the block-relative one.
func TestLineOffsetInRedefinitionMessage(t *testing.T) {
	// Simulate a document with 4 lines of frontmatter (lines 1-4),
	// then a calc block starting at line 5.
	// Block-relative line 1 → document line 5.
	checker := NewChecker()
	checker.SetLineOffset(4) // 4 frontmatter lines before this block

	// First assignment: "x = 10" at block-relative line 1
	nodes := []ast.Node{
		&ast.Assignment{
			Name:  "x",
			Value: &ast.NumberLiteral{Value: "10"},
			Range: &ast.Range{
				Start: ast.Position{Line: 1, Column: 1},
				End:   ast.Position{Line: 1, Column: 6},
			},
		},
		// Second assignment: "x = 20" at block-relative line 2 — redefinition
		&ast.Assignment{
			Name:  "x",
			Value: &ast.NumberLiteral{Value: "20"},
			Range: &ast.Range{
				Start: ast.Position{Line: 2, Column: 1},
				End:   ast.Position{Line: 2, Column: 6},
			},
		},
	}

	diagnostics := checker.Check(nodes)
	if len(diagnostics) == 0 {
		t.Fatal("Expected a redefinition diagnostic")
	}

	msg := diagnostics[0].Message
	// Should say "line 5" (4 offset + block line 1), NOT "line 1"
	if !strings.Contains(msg, "line 5") {
		t.Errorf("Expected 'line 5' (document-absolute) in message, got: %s", msg)
	}
	if strings.Contains(msg, "line 1") {
		t.Errorf("Should NOT contain block-relative 'line 1', got: %s", msg)
	}
}

// TestLineOffsetZeroPreservesOriginalBehavior verifies that when no line offset
// is set (default 0), line numbers are unchanged.
func TestLineOffsetZeroPreservesOriginalBehavior(t *testing.T) {
	checker := NewChecker()
	// No SetLineOffset call — default is 0

	nodes := []ast.Node{
		&ast.Assignment{
			Name:  "x",
			Value: &ast.NumberLiteral{Value: "10"},
			Range: &ast.Range{
				Start: ast.Position{Line: 1, Column: 1},
				End:   ast.Position{Line: 1, Column: 6},
			},
		},
		&ast.Assignment{
			Name:  "x",
			Value: &ast.NumberLiteral{Value: "20"},
			Range: &ast.Range{
				Start: ast.Position{Line: 2, Column: 1},
				End:   ast.Position{Line: 2, Column: 6},
			},
		},
	}

	diagnostics := checker.Check(nodes)
	if len(diagnostics) == 0 {
		t.Fatal("Expected a redefinition diagnostic")
	}

	msg := diagnostics[0].Message
	// With no offset, should say "line 1"
	if !strings.Contains(msg, "line 1") {
		t.Errorf("Expected 'line 1' with zero offset, got: %s", msg)
	}
}

func TestFractionSemanticChecks(t *testing.T) {
	t.Run("zero denominator warning", func(t *testing.T) {
		env := NewEnvironment()
		checker := NewCheckerWithEnv(env)
		nodes := []ast.Node{
			&ast.Expression{
				Expr: &ast.FractionLiteral{Numerator: 1, Denominator: 0},
			},
		}
		diagnostics := checker.Check(nodes)
		if len(diagnostics) != 1 {
			t.Fatalf("expected 1 diagnostic, got %d", len(diagnostics))
		}
		if diagnostics[0].Code != DiagDivisionByZero {
			t.Errorf("expected %s, got %s", DiagDivisionByZero, diagnostics[0].Code)
		}
	})

	t.Run("valid fraction no diagnostics", func(t *testing.T) {
		env := NewEnvironment()
		checker := NewCheckerWithEnv(env)
		nodes := []ast.Node{
			&ast.Expression{
				Expr: &ast.FractionLiteral{Numerator: 1, Denominator: 3},
			},
		}
		diagnostics := checker.Check(nodes)
		if len(diagnostics) != 0 {
			t.Errorf("expected 0 diagnostics, got %d: %v", len(diagnostics), diagnostics)
		}
	})

	t.Run("fraction with valid unit", func(t *testing.T) {
		env := NewEnvironment()
		checker := NewCheckerWithEnv(env)
		nodes := []ast.Node{
			&ast.Expression{
				Expr: &ast.FractionLiteral{Numerator: 1, Denominator: 3, Unit: "cup"},
			},
		}
		diagnostics := checker.Check(nodes)
		if len(diagnostics) != 0 {
			t.Errorf("expected 0 diagnostics, got %d: %v", len(diagnostics), diagnostics)
		}
	})

	t.Run("fraction with arbitrary unit allowed", func(t *testing.T) {
		env := NewEnvironment()
		checker := NewCheckerWithEnv(env)
		nodes := []ast.Node{
			&ast.Expression{
				Expr: &ast.FractionLiteral{Numerator: 1, Denominator: 3, Unit: "pizza"},
			},
		}
		diagnostics := checker.Check(nodes)
		if len(diagnostics) != 0 {
			t.Errorf("expected 0 diagnostics for arbitrary unit, got %d", len(diagnostics))
		}
	})
}
