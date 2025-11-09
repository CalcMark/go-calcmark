package parser

import (
	"testing"

	"github.com/CalcMark/go-calcmark/ast"
)

func TestParseSimpleNumber(t *testing.T) {
	nodes, err := Parse("42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}

	expr, ok := nodes[0].(*ast.Expression)
	if !ok {
		t.Fatalf("expected Expression, got %T", nodes[0])
	}

	num, ok := expr.Expr.(*ast.NumberLiteral)
	if !ok {
		t.Fatalf("expected NumberLiteral, got %T", expr.Expr)
	}

	if num.Value != "42" {
		t.Errorf("expected value '42', got '%s'", num.Value)
	}
}

func TestParseCurrency(t *testing.T) {
	nodes, err := Parse("$100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expr := nodes[0].(*ast.Expression)
	curr, ok := expr.Expr.(*ast.CurrencyLiteral)
	if !ok {
		t.Fatalf("expected CurrencyLiteral, got %T", expr.Expr)
	}

	if curr.Symbol != "$" {
		t.Errorf("expected symbol '$', got '%s'", curr.Symbol)
	}
	if curr.Value != "100" {
		t.Errorf("expected value '100', got '%s'", curr.Value)
	}
}

func TestParseIdentifier(t *testing.T) {
	nodes, err := Parse("salary")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expr := nodes[0].(*ast.Expression)
	ident, ok := expr.Expr.(*ast.Identifier)
	if !ok {
		t.Fatalf("expected Identifier, got %T", expr.Expr)
	}

	if ident.Name != "salary" {
		t.Errorf("expected name 'salary', got '%s'", ident.Name)
	}
}

func TestParseBinaryOperations(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		operator string
	}{
		{"multiplication", "3 * 3", "*"},
		{"addition", "1 + 2", "+"},
		{"subtraction", "5 - 3", "-"},
		{"division", "10 / 2", "/"},
		{"modulus", "10 % 3", "%"},
		{"exponent **", "2 ** 3", "**"},
		{"exponent ^", "2 ^ 3", "^"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			expr := nodes[0].(*ast.Expression)
			binOp, ok := expr.Expr.(*ast.BinaryOp)
			if !ok {
				t.Fatalf("expected BinaryOp, got %T", expr.Expr)
			}

			if binOp.Operator != tt.operator {
				t.Errorf("expected operator '%s', got '%s'", tt.operator, binOp.Operator)
			}
		})
	}
}

func TestParseComparisons(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		operator string
	}{
		{"greater than", "5 > 3", ">"},
		{"less than", "3 < 5", "<"},
		{"greater equal", "5 >= 3", ">="},
		{"less equal", "3 <= 5", "<="},
		{"equal", "5 == 5", "=="},
		{"not equal", "5 != 3", "!="},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			expr := nodes[0].(*ast.Expression)
			compOp, ok := expr.Expr.(*ast.ComparisonOp)
			if !ok {
				t.Fatalf("expected ComparisonOp, got %T", expr.Expr)
			}

			if compOp.Operator != tt.operator {
				t.Errorf("expected operator '%s', got '%s'", tt.operator, compOp.Operator)
			}
		})
	}
}

func TestParseSimpleAssignment(t *testing.T) {
	nodes, err := Parse("x = 5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assign, ok := nodes[0].(*ast.Assignment)
	if !ok {
		t.Fatalf("expected Assignment, got %T", nodes[0])
	}

	if assign.Name != "x" {
		t.Errorf("expected name 'x', got '%s'", assign.Name)
	}

	num, ok := assign.Value.(*ast.NumberLiteral)
	if !ok {
		t.Fatalf("expected NumberLiteral, got %T", assign.Value)
	}

	if num.Value != "5" {
		t.Errorf("expected value '5', got '%s'", num.Value)
	}
}

func TestParseCurrencyAssignment(t *testing.T) {
	nodes, err := Parse("salary = $1000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assign := nodes[0].(*ast.Assignment)
	if assign.Name != "salary" {
		t.Errorf("expected name 'salary', got '%s'", assign.Name)
	}

	curr, ok := assign.Value.(*ast.CurrencyLiteral)
	if !ok {
		t.Fatalf("expected CurrencyLiteral, got %T", assign.Value)
	}

	if curr.Value != "1000" {
		t.Errorf("expected value '1000', got '%s'", curr.Value)
	}
}

func TestParseExpressionAssignment(t *testing.T) {
	nodes, err := Parse("total = salary + bonus")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assign := nodes[0].(*ast.Assignment)
	binOp, ok := assign.Value.(*ast.BinaryOp)
	if !ok {
		t.Fatalf("expected BinaryOp, got %T", assign.Value)
	}

	if binOp.Operator != "+" {
		t.Errorf("expected operator '+', got '%s'", binOp.Operator)
	}
}

func TestOperatorPrecedence(t *testing.T) {
	// 1 + 2 * 3 should parse as 1 + (2 * 3)
	nodes, err := Parse("1 + 2 * 3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expr := nodes[0].(*ast.Expression)
	add, ok := expr.Expr.(*ast.BinaryOp)
	if !ok || add.Operator != "+" {
		t.Fatal("expected addition at root")
	}

	// Left should be 1
	leftNum, ok := add.Left.(*ast.NumberLiteral)
	if !ok || leftNum.Value != "1" {
		t.Error("expected left to be 1")
	}

	// Right should be 2 * 3
	mult, ok := add.Right.(*ast.BinaryOp)
	if !ok || mult.Operator != "*" {
		t.Fatal("expected multiplication on right")
	}

	leftMultNum, ok := mult.Left.(*ast.NumberLiteral)
	if !ok || leftMultNum.Value != "2" {
		t.Error("expected mult left to be 2")
	}

	rightMultNum, ok := mult.Right.(*ast.NumberLiteral)
	if !ok || rightMultNum.Value != "3" {
		t.Error("expected mult right to be 3")
	}
}

func TestLeftAssociativity(t *testing.T) {
	// 10 - 5 - 2 should parse as (10 - 5) - 2
	nodes, err := Parse("10 - 5 - 2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expr := nodes[0].(*ast.Expression)
	sub2, ok := expr.Expr.(*ast.BinaryOp)
	if !ok || sub2.Operator != "-" {
		t.Fatal("expected subtraction at root")
	}

	// Left should be (10 - 5)
	sub1, ok := sub2.Left.(*ast.BinaryOp)
	if !ok || sub1.Operator != "-" {
		t.Fatal("expected subtraction on left")
	}

	// Right should be 2
	rightNum, ok := sub2.Right.(*ast.NumberLiteral)
	if !ok || rightNum.Value != "2" {
		t.Error("expected right to be 2")
	}
}

func TestRightAssociativity(t *testing.T) {
	// 2 ^ 3 ^ 2 should parse as 2 ^ (3 ^ 2)
	nodes, err := Parse("2 ^ 3 ^ 2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expr := nodes[0].(*ast.Expression)
	exp1, ok := expr.Expr.(*ast.BinaryOp)
	if !ok || exp1.Operator != "^" {
		t.Fatal("expected exponent at root")
	}

	// Left should be 2
	leftNum, ok := exp1.Left.(*ast.NumberLiteral)
	if !ok || leftNum.Value != "2" {
		t.Error("expected left to be 2")
	}

	// Right should be (3 ^ 2)
	exp2, ok := exp1.Right.(*ast.BinaryOp)
	if !ok || exp2.Operator != "^" {
		t.Fatal("expected exponent on right")
	}
}

func TestParseMultipleStatements(t *testing.T) {
	nodes, err := Parse("x = 5\ny = 10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}

	assign1, ok := nodes[0].(*ast.Assignment)
	if !ok || assign1.Name != "x" {
		t.Error("first statement should be x = 5")
	}

	assign2, ok := nodes[1].(*ast.Assignment)
	if !ok || assign2.Name != "y" {
		t.Error("second statement should be y = 10")
	}
}

func TestParseWithBlankLines(t *testing.T) {
	nodes, err := Parse("x = 5\n\ny = 10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
}

func TestParseIdentifierWithSpaces(t *testing.T) {
	nodes, err := Parse("weeks in year = 52")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assign := nodes[0].(*ast.Assignment)
	if assign.Name != "weeks in year" {
		t.Errorf("expected name 'weeks in year', got '%s'", assign.Name)
	}
}

func TestParseBooleanAsVariable(t *testing.T) {
	// Boolean keywords can be used as variable names
	nodes, err := Parse("true = 1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assign := nodes[0].(*ast.Assignment)
	if assign.Name != "true" {
		t.Errorf("expected name 'true', got '%s'", assign.Name)
	}
}

func TestParseBooleanAsValue(t *testing.T) {
	// Boolean keywords used as identifiers in expressions
	nodes, err := Parse("x = true")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assign := nodes[0].(*ast.Assignment)
	ident, ok := assign.Value.(*ast.Identifier)
	if !ok {
		t.Fatalf("expected Identifier, got %T", assign.Value)
	}

	if ident.Name != "true" {
		t.Errorf("expected name 'true', got '%s'", ident.Name)
	}
}

func TestParseComplexExpression(t *testing.T) {
	nodes, err := Parse("total = $1,000 * 52 + bonus")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assign := nodes[0].(*ast.Assignment)
	if assign.Name != "total" {
		t.Errorf("expected name 'total', got '%s'", assign.Name)
	}

	// Should be: ($1,000 * 52) + bonus
	add, ok := assign.Value.(*ast.BinaryOp)
	if !ok || add.Operator != "+" {
		t.Fatal("expected addition at root")
	}

	// Left should be $1,000 * 52
	mult, ok := add.Left.(*ast.BinaryOp)
	if !ok || mult.Operator != "*" {
		t.Fatal("expected multiplication on left")
	}
}

func TestParseEmptyInput(t *testing.T) {
	nodes, err := Parse("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(nodes))
	}
}

func TestParseError(t *testing.T) {
	// Test invalid syntax
	_, err := Parse("=")
	if err == nil {
		t.Error("expected error for invalid syntax")
	}
}

func TestParsePositions(t *testing.T) {
	nodes, err := Parse("x = 5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assign := nodes[0].(*ast.Assignment)
	if assign.Range == nil {
		t.Error("expected range to be set")
	}

	if assign.Range.Start.Line != 1 || assign.Range.Start.Column != 1 {
		t.Errorf("expected position 1:1, got %s", assign.Range.Start)
	}
}
