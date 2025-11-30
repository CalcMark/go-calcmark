package interpreter

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

// Test pure helper functions

func TestExpandNumberLiteral(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"42", "42"},
		{"1k", "1000"},
		{"1.2k", "1200"},
		{"1M", "1000000"},
		{"1.5M", "1500000"},
		{"1B", "1000000000"},
		{"2.5B", "2500000000"},
		{"1e3", "1000"},
		{"1.2e6", "1200000"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := expandNumberLiteral(tt.input)
			if err != nil {
				t.Fatalf("expandNumberLiteral(%q) error = %v", tt.input, err)
			}
			if got.String() != tt.want {
				t.Errorf("expandNumberLiteral(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseBooleanValue(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"true", true},
		{"TRUE", true},
		{"false", false},
		{"FALSE", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseBooleanValue(tt.input)
			if err != nil {
				t.Fatalf("parseBooleanValue(%q) error = %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("parseBooleanValue(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestParseBooleanValue_InvalidInputs verifies that y/n/t/yes/no are NOT parsed as booleans
func TestParseBooleanValue_InvalidInputs(t *testing.T) {
	invalidInputs := []string{"y", "n", "t", "yes", "no", "YES", "NO", "1", "0"}

	for _, input := range invalidInputs {
		t.Run(input, func(t *testing.T) {
			_, err := parseBooleanValue(input)
			if err == nil {
				t.Errorf("parseBooleanValue(%q) should return error but didn't", input)
			}
		})
	}
}

func TestParseMonth(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"Jan", 1},
		{"January", 1},
		{"Feb", 2},
		{"December", 12},
		{"dec", 12},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseMonth(tt.input)
			if err != nil {
				t.Fatalf("parseMonth(%q) error = %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("parseMonth(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// Test literal evaluation

func TestEvalNumberLiteral(t *testing.T) {
	interp := NewInterpreter()

	node := &ast.NumberLiteral{Value: "1.2k"}
	result, err := interp.evalNumberLiteral(node)
	if err != nil {
		t.Fatalf("evalNumberLiteral error = %v", err)
	}

	if result.String() != "1200" {
		t.Errorf("Result = %v, want 1200", result.String())
	}
}

func TestEvalCurrencyLiteral(t *testing.T) {
	interp := NewInterpreter()

	node := &ast.CurrencyLiteral{
		Value:  "100",
		Symbol: "$",
	}

	result, err := interp.evalCurrencyLiteral(node)
	if err != nil {
		t.Fatalf("evalCurrencyLiteral error = %v", err)
	}

	if result.String() != "$100.00" {
		t.Errorf("Result = %v, want $100.00", result.String())
	}
}

// Test binary operations

func TestEvalNumberOperation(t *testing.T) {
	tests := []struct {
		name     string
		left     string
		right    string
		operator string
		want     string
	}{
		{"addition", "10", "5", "+", "15"},
		{"subtraction", "10", "5", "-", "5"},
		{"multiplication", "10", "5", "*", "50"},
		{"division", "10", "5", "/", "2"},
		{"modulus", "10", "3", "%", "1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			left, _ := decimal.NewFromString(tt.left)
			right, _ := decimal.NewFromString(tt.right)

			leftNum := &types.Number{Value: left}
			rightNum := &types.Number{Value: right}

			result, err := evalNumberOperation(leftNum, rightNum, tt.operator)
			if err != nil {
				t.Fatalf("evalNumberOperation error = %v", err)
			}

			if result.String() != tt.want {
				t.Errorf("Result = %v, want %v", result.String(), tt.want)
			}
		})
	}
}

// Test environment

func TestEnvironment(t *testing.T) {
	env := NewEnvironment()

	// Test setting and getting
	env.Set("x", nil)
	if !env.Has("x") {
		t.Error("Expected variable 'x' to be defined")
	}

	// Test undefined variable
	if env.Has("y") {
		t.Error("Expected variable 'y' to be undefined")
	}
}

// Test variable assignment and lookup

func TestAssignmentAndLookup(t *testing.T) {
	interp := NewInterpreter()

	// x = 42
	assignment := &ast.Assignment{
		Name:  "x",
		Value: &ast.NumberLiteral{Value: "42"},
	}

	_, err := interp.evalAssignment(assignment)
	if err != nil {
		t.Fatalf("evalAssignment error = %v", err)
	}

	// Look up x
	id := &ast.Identifier{Name: "x"}
	result, err := interp.evalIdentifier(id)
	if err != nil {
		t.Fatalf("evalIdentifier error = %v", err)
	}

	if result.String() != "42" {
		t.Errorf("Result = %v, want 42", result.String())
	}
}

// TestBuiltinConstants verifies PI and E are pre-defined.
func TestBuiltinConstants(t *testing.T) {
	interp := NewInterpreter()

	tests := []struct {
		name   string
		prefix string // Expected prefix of the value
	}{
		{"PI", "3.14159265358979"},
		{"E", "2.71828182845904"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := &ast.Identifier{Name: tt.name}
			result, err := interp.evalIdentifier(id)
			if err != nil {
				t.Fatalf("evalIdentifier(%q) error = %v", tt.name, err)
			}

			num, ok := result.(*types.Number)
			if !ok {
				t.Fatalf("Expected *types.Number, got %T", result)
			}

			str := num.String()
			if len(str) < len(tt.prefix) || str[:len(tt.prefix)] != tt.prefix {
				t.Errorf("%s = %v, want prefix %v", tt.name, str, tt.prefix)
			}
		})
	}
}

// TestBuiltinConstantsUsage verifies PI and E can be used in expressions.
func TestBuiltinConstantsUsage(t *testing.T) {
	interp := NewInterpreter()

	// Test: 2 * PI (should be ~6.28...)
	node := &ast.BinaryOp{
		Operator: "*",
		Left:     &ast.NumberLiteral{Value: "2"},
		Right:    &ast.Identifier{Name: "PI"},
	}

	result, err := interp.evalBinaryOp(node)
	if err != nil {
		t.Fatalf("evalBinaryOp error = %v", err)
	}

	num, ok := result.(*types.Number)
	if !ok {
		t.Fatalf("Expected *types.Number, got %T", result)
	}

	// 2*PI should be approximately 6.28
	val := num.Value.InexactFloat64()
	if val < 6.28 || val > 6.29 {
		t.Errorf("2 * PI = %v, want ~6.28", val)
	}
}

// TestFrontmatterGlobalAssignment tests @global.name = value syntax.
func TestFrontmatterGlobalAssignment(t *testing.T) {
	interp := NewInterpreter()

	// @global.tax_rate = 0.32
	fmAssign := &ast.FrontmatterAssignment{
		Namespace: "global",
		Property:  "tax_rate",
		Value:     &ast.NumberLiteral{Value: "0.32"},
	}

	result, err := interp.evalFrontmatterAssignment(fmAssign)
	if err != nil {
		t.Fatalf("evalFrontmatterAssignment error = %v", err)
	}

	// Verify the result value
	num, ok := result.(*types.Number)
	if !ok {
		t.Fatalf("Expected *types.Number, got %T", result)
	}
	if num.String() != "0.32" {
		t.Errorf("Result = %v, want 0.32", num.String())
	}

	// Verify the variable is accessible
	id := &ast.Identifier{Name: "tax_rate"}
	varResult, err := interp.evalIdentifier(id)
	if err != nil {
		t.Fatalf("evalIdentifier error = %v", err)
	}
	if varResult.String() != "0.32" {
		t.Errorf("tax_rate = %v, want 0.32", varResult.String())
	}
}

// TestFrontmatterExchangeRateAssignment tests @exchange.FROM_TO = rate syntax.
func TestFrontmatterExchangeRateAssignment(t *testing.T) {
	interp := NewInterpreter()

	// @exchange.USD_EUR = 0.92
	fmAssign := &ast.FrontmatterAssignment{
		Namespace: "exchange",
		Property:  "USD_EUR",
		Value:     &ast.NumberLiteral{Value: "0.92"},
	}

	result, err := interp.evalFrontmatterAssignment(fmAssign)
	if err != nil {
		t.Fatalf("evalFrontmatterAssignment error = %v", err)
	}

	// Verify the result value
	num, ok := result.(*types.Number)
	if !ok {
		t.Fatalf("Expected *types.Number, got %T", result)
	}
	if num.String() != "0.92" {
		t.Errorf("Result = %v, want 0.92", num.String())
	}

	// Verify the exchange rate is set in environment
	rate, ok := interp.env.GetExchangeRate("USD", "EUR")
	if !ok {
		t.Fatal("Exchange rate USD_EUR not found")
	}
	if !rate.Equal(decimal.NewFromFloat(0.92)) {
		t.Errorf("Exchange rate = %v, want 0.92", rate)
	}
}

// TestFrontmatterExchangeRateInvalidFormat tests invalid exchange key formats.
func TestFrontmatterExchangeRateInvalidFormat(t *testing.T) {
	interp := NewInterpreter()

	tests := []struct {
		name     string
		property string
	}{
		{"missing separator", "USDEUR"},
		{"single currency", "USD"},
		{"empty to currency", "USD_"},
		{"empty from currency", "_EUR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fmAssign := &ast.FrontmatterAssignment{
				Namespace: "exchange",
				Property:  tt.property,
				Value:     &ast.NumberLiteral{Value: "1.0"},
			}

			_, err := interp.evalFrontmatterAssignment(fmAssign)
			if err == nil {
				t.Errorf("Expected error for property %q, got none", tt.property)
			}
		})
	}
}

// TestFrontmatterUnknownNamespace tests that unknown namespaces are rejected.
func TestFrontmatterUnknownNamespace(t *testing.T) {
	interp := NewInterpreter()

	fmAssign := &ast.FrontmatterAssignment{
		Namespace: "unknown",
		Property:  "foo",
		Value:     &ast.NumberLiteral{Value: "42"},
	}

	_, err := interp.evalFrontmatterAssignment(fmAssign)
	if err == nil {
		t.Error("Expected error for unknown namespace, got none")
	}
}

// TestParseExchangeKey tests the parseExchangeKey helper function.
func TestParseExchangeKey(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		wantFrom string
		wantTo   string
		wantErr  bool
	}{
		{"valid key", "USD_EUR", "USD", "EUR", false},
		{"lowercase", "usd_eur", "USD", "EUR", false},
		{"mixed case", "Usd_Eur", "USD", "EUR", false},
		{"with spaces", " USD _ EUR ", "USD", "EUR", false},
		{"missing separator", "USDEUR", "", "", true},
		{"too many parts", "USD_EUR_GBP", "", "", true},
		{"empty from", "_EUR", "", "", true},
		{"empty to", "USD_", "", "", true},
		{"both empty", "_", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, to, err := parseExchangeKey(tt.key)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for %q, got none", tt.key)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error for %q: %v", tt.key, err)
			}
			if from != tt.wantFrom {
				t.Errorf("from = %q, want %q", from, tt.wantFrom)
			}
			if to != tt.wantTo {
				t.Errorf("to = %q, want %q", to, tt.wantTo)
			}
		})
	}
}
