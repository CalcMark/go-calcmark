package ast

import "testing"

func TestContainsScaleRef_Nil(t *testing.T) {
	if ContainsScaleRef(nil) {
		t.Error("nil should return false")
	}
}

func TestContainsScaleRef_DirectiveRefScale(t *testing.T) {
	node := &DirectiveRef{Directive: "scale"}
	if !ContainsScaleRef(node) {
		t.Error("@scale should return true")
	}
}

func TestContainsScaleRef_DirectiveRefGlobals(t *testing.T) {
	node := &DirectiveRef{Directive: "globals", Field: "tax_rate"}
	if ContainsScaleRef(node) {
		t.Error("@globals.tax_rate should return false")
	}
}

func TestContainsScaleRef_AssignmentWithScale(t *testing.T) {
	node := &Assignment{
		Name: "per_unit",
		Value: &BinaryOp{
			Operator: "/",
			Left:     &Identifier{Name: "cost"},
			Right:    &DirectiveRef{Directive: "scale"},
		},
	}
	if !ContainsScaleRef(node) {
		t.Error("assignment with @scale in value should return true")
	}
}

func TestContainsScaleRef_AssignmentWithoutScale(t *testing.T) {
	node := &Assignment{
		Name: "total",
		Value: &BinaryOp{
			Operator: "+",
			Left:     &NumberLiteral{Value: "1"},
			Right:    &NumberLiteral{Value: "2"},
		},
	}
	if ContainsScaleRef(node) {
		t.Error("assignment without @scale should return false")
	}
}

func TestContainsScaleRef_ExpressionWithScale(t *testing.T) {
	node := &Expression{
		Expr: &BinaryOp{
			Operator: "*",
			Left:     &NumberLiteral{Value: "5"},
			Right:    &DirectiveRef{Directive: "scale"},
		},
	}
	if !ContainsScaleRef(node) {
		t.Error("expression with @scale should return true")
	}
}

func TestContainsScaleRef_NestedBinaryOp(t *testing.T) {
	node := &BinaryOp{
		Operator: "+",
		Left:     &NumberLiteral{Value: "1"},
		Right: &BinaryOp{
			Operator: "*",
			Left:     &NumberLiteral{Value: "2"},
			Right:    &DirectiveRef{Directive: "scale"},
		},
	}
	if !ContainsScaleRef(node) {
		t.Error("nested @scale in right branch should return true")
	}
}

func TestContainsScaleRef_UnaryOp(t *testing.T) {
	node := &UnaryOp{
		Operator: "-",
		Operand:  &DirectiveRef{Directive: "scale"},
	}
	if !ContainsScaleRef(node) {
		t.Error("unary with @scale should return true")
	}
}

func TestContainsScaleRef_FunctionCall(t *testing.T) {
	node := &FunctionCall{
		Name: "sqrt",
		Arguments: []Node{
			&DirectiveRef{Directive: "scale"},
		},
	}
	if !ContainsScaleRef(node) {
		t.Error("function call with @scale arg should return true")
	}
}

func TestContainsScaleRef_FunctionCallNoScale(t *testing.T) {
	node := &FunctionCall{
		Name: "sqrt",
		Arguments: []Node{
			&NumberLiteral{Value: "9"},
		},
	}
	if ContainsScaleRef(node) {
		t.Error("function call without @scale should return false")
	}
}

func TestContainsScaleRef_UnitConversion(t *testing.T) {
	node := &UnitConversion{
		Quantity:   &DirectiveRef{Directive: "scale"},
		TargetUnit: "kg",
	}
	if !ContainsScaleRef(node) {
		t.Error("unit conversion with @scale should return true")
	}
}

func TestContainsScaleRef_NapkinConversion(t *testing.T) {
	node := &NapkinConversion{
		Expression: &DirectiveRef{Directive: "scale"},
	}
	if !ContainsScaleRef(node) {
		t.Error("napkin conversion with @scale should return true")
	}
}

func TestContainsScaleRef_PercentageOf(t *testing.T) {
	node := &PercentageOf{
		Percentage: &NumberLiteral{Value: "10"},
		Value:      &DirectiveRef{Directive: "scale"},
	}
	if !ContainsScaleRef(node) {
		t.Error("percentage of @scale should return true")
	}
}

func TestContainsScaleRef_ComparisonOp(t *testing.T) {
	node := &ComparisonOp{
		Operator: ">",
		Left:     &DirectiveRef{Directive: "scale"},
		Right:    &NumberLiteral{Value: "1"},
	}
	if !ContainsScaleRef(node) {
		t.Error("comparison with @scale should return true")
	}
}

func TestContainsScaleRef_LeafNodes(t *testing.T) {
	leaves := []Node{
		&NumberLiteral{Value: "42"},
		&CurrencyLiteral{Symbol: "$", Value: "100"},
		&QuantityLiteral{Value: "5", Unit: "kg"},
		&BooleanLiteral{Value: "true"},
		&Identifier{Name: "x"},
		&DurationLiteral{Value: "3", Unit: "days"},
	}
	for _, leaf := range leaves {
		if ContainsScaleRef(leaf) {
			t.Errorf("leaf node %T should return false", leaf)
		}
	}
}

func TestContainsScaleRef_RateLiteral(t *testing.T) {
	node := &RateLiteral{
		Amount:  &DirectiveRef{Directive: "scale"},
		PerUnit: "hour",
	}
	if !ContainsScaleRef(node) {
		t.Error("rate with @scale amount should return true")
	}
}

func TestContainsScaleRef_PreciseConversion(t *testing.T) {
	node := &PreciseConversion{
		Expression: &DirectiveRef{Directive: "scale"},
	}
	if !ContainsScaleRef(node) {
		t.Error("precise conversion with @scale should return true")
	}
}
