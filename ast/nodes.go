package ast

import "fmt"

// Node is the interface that all AST nodes implement
type Node interface {
	String() string
	GetRange() *Range
}

// NumberLiteral represents a numeric literal
type NumberLiteral struct {
	Value string
	Range *Range
}

func (n *NumberLiteral) String() string {
	return fmt.Sprintf("NumberLiteral(%s)", n.Value)
}

func (n *NumberLiteral) GetRange() *Range {
	return n.Range
}

// CurrencyLiteral represents a currency literal
type CurrencyLiteral struct {
	Value  string
	Symbol string
	Range  *Range
}

func (c *CurrencyLiteral) String() string {
	return fmt.Sprintf("CurrencyLiteral(%s%s)", c.Symbol, c.Value)
}

func (c *CurrencyLiteral) GetRange() *Range {
	return c.Range
}

// BooleanLiteral represents a boolean literal
type BooleanLiteral struct {
	Value string // "true", "false", "yes", "no", etc.
	Range *Range
}

func (b *BooleanLiteral) String() string {
	return fmt.Sprintf("BooleanLiteral(%s)", b.Value)
}

func (b *BooleanLiteral) GetRange() *Range {
	return b.Range
}

// Identifier represents a variable identifier
type Identifier struct {
	Name  string
	Range *Range
}

func (i *Identifier) String() string {
	return fmt.Sprintf("Identifier(%q)", i.Name)
}

func (i *Identifier) GetRange() *Range {
	return i.Range
}

// BinaryOp represents a binary operation (+, -, *, /, etc.)
type BinaryOp struct {
	Operator string
	Left     Node
	Right    Node
	Range    *Range
}

func (b *BinaryOp) String() string {
	return fmt.Sprintf("BinaryOp(%q, %s, %s)", b.Operator, b.Left, b.Right)
}

func (b *BinaryOp) GetRange() *Range {
	return b.Range
}

// ComparisonOp represents a comparison operation (>, <, ==, etc.)
type ComparisonOp struct {
	Operator string // ">", "<", ">=", "<=", "==", "!="
	Left     Node
	Right    Node
	Range    *Range
}

func (c *ComparisonOp) String() string {
	return fmt.Sprintf("ComparisonOp(%q, %s, %s)", c.Operator, c.Left, c.Right)
}

func (c *ComparisonOp) GetRange() *Range {
	return c.Range
}

// Assignment represents a variable assignment
type Assignment struct {
	Name  string
	Value Node
	Range *Range
}

func (a *Assignment) String() string {
	return fmt.Sprintf("Assignment(%q, %s)", a.Name, a.Value)
}

func (a *Assignment) GetRange() *Range {
	return a.Range
}

// Expression represents a standalone expression (no assignment)
type Expression struct {
	Expr  Node
	Range *Range
}

func (e *Expression) String() string {
	return fmt.Sprintf("Expression(%s)", e.Expr)
}

func (e *Expression) GetRange() *Range {
	return e.Range
}
