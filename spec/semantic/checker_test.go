package semantic

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/ast"
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
