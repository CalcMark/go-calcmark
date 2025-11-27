package ast

import (
	"fmt"
	"strings"
)

// Node is the interface that all AST nodes implement
type Node interface {
	String() string
	GetRange() *Range
}

// NumberLiteral represents a numeric literal
type NumberLiteral struct {
	Value      string // Normalized value (e.g., "1000" for both "1000" and "1,000")
	SourceText string // Original text from source (e.g., "1,000", "1k", "10000")
	Range      *Range
}

func (n *NumberLiteral) String() string {
	return fmt.Sprintf("NumberLiteral(%s)", n.Value)
}

func (n *NumberLiteral) GetRange() *Range {
	return n.Range
}

// CurrencyLiteral represents a currency value (e.g., "$100", "50 EUR").
type CurrencyLiteral struct {
	Symbol     string // Currency symbol or code
	Value      string // Numeric value
	SourceText string // Original text for debugging
	Range      *Range
}

func (c *CurrencyLiteral) String() string {
	return fmt.Sprintf("CurrencyLiteral(%s%s)", c.Symbol, c.Value)
}

func (c *CurrencyLiteral) GetRange() *Range {
	return c.Range
}

// QuantityLiteral represents a number with a unit (e.g., "5 kg", "10 meters").
type QuantityLiteral struct {
	Value      string // The numeric value
	Unit       string // The unit identifier
	SourceText string // Original text
	Range      *Range
}

func (q *QuantityLiteral) String() string {
	return fmt.Sprintf("QuantityLiteral(%s %s)", q.Value, q.Unit)
}

func (q *QuantityLiteral) GetRange() *Range {
	return q.Range
}

// UnitConversion represents explicit unit conversion (e.g., "10 meters in feet").
type UnitConversion struct {
	Quantity   Node   // The quantity expression to convert
	TargetUnit string // The target unit to convert to
	Range      *Range
}

func (u *UnitConversion) String() string {
	return fmt.Sprintf("UnitConversion(%s in %s)", u.Quantity.String(), u.TargetUnit)
}

func (u *UnitConversion) GetRange() *Range {
	return u.Range
}

// NapkinConversion represents human-readable number formatting (e.g., "1234567 as napkin").
type NapkinConversion struct {
	Expression Node
	Range      *Range
}

func (n *NapkinConversion) String() string {
	return fmt.Sprintf("NapkinConversion(%s)", n.Expression.String())
}

func (n *NapkinConversion) GetRange() *Range {
	return n.Range
}

// PercentageOf represents percentage of a value (e.g., "10% of 200").
type PercentageOf struct {
	Percentage Node // The percentage value (e.g., NumberLiteral for "10%")
	Value      Node // The value to take percentage of
	Range      *Range
}

func (p *PercentageOf) String() string {
	return fmt.Sprintf("PercentageOf(%s of %s)", p.Percentage.String(), p.Value.String())
}

func (p *PercentageOf) GetRange() *Range {
	return p.Range
}

// RateLiteral represents a rate expression (e.g., "100 MB/s", "5 GB per day", "$0.10 per hour").
// Rates combine a quantity (amount) with a time period.
type RateLiteral struct {
	Amount     Node   // The quantity numerator (e.g., QuantityLiteral for "100 MB")
	PerUnit    string // The time unit denominator (e.g., "second", "hour", "day")
	SourceText string //Original text
	Range      *Range
}

func (r *RateLiteral) String() string {
	return fmt.Sprintf("RateLiteral(%s per %s)", r.Amount.String(), r.PerUnit)
}

func (r *RateLiteral) GetRange() *Range {
	return r.Range
}

// DateLiteral represents a date literal: "Dec 25" or "Dec 25 2024"
type DateLiteral struct {
	Month      string  // "Dec", "December"
	Day        string  // "25"
	Year       *string // nil if not provided, "2024" if provided
	SourceText string
	Range      *Range
}

func (d *DateLiteral) String() string {
	if d.Year != nil {
		return fmt.Sprintf("DateLiteral(%s %s %s)", d.Month, d.Day, *d.Year)
	}
	return fmt.Sprintf("DateLiteral(%s %s)", d.Month, d.Day)
}

func (d *DateLiteral) GetRange() *Range {
	return d.Range
}

// TimeLiteral represents a time literal: "10:30AM", "14:30", "10:30:45PM", "10:30AM UTC-7"
type TimeLiteral struct {
	Hour       string     // "10", "14"
	Minute     string     // "30"
	Second     *string    // nil or "45"
	Period     *string    // nil, "AM", or "PM"
	UTCOffset  *UTCOffset // nil or offset spec
	SourceText string
	Range      *Range
}

func (t *TimeLiteral) String() string {
	var parts []string

	if t.Second != nil {
		parts = append(parts, fmt.Sprintf("%s:%s:%s", t.Hour, t.Minute, *t.Second))
	} else {
		parts = append(parts, fmt.Sprintf("%s:%s", t.Hour, t.Minute))
	}

	if t.Period != nil {
		parts = append(parts, *t.Period)
	}

	if t.UTCOffset != nil {
		parts = append(parts, t.UTCOffset.String())
	}

	return fmt.Sprintf("TimeLiteral(%s)", strings.Join(parts, " "))
}

func (t *TimeLiteral) GetRange() *Range {
	return t.Range
}

// UTCOffset represents a UTC timezone offset: UTC-7, UTC+5:30
type UTCOffset struct {
	Sign    string  // "+" or "-"
	Hours   string  // "7", "5"
	Minutes *string // nil or "30" (for UTC+5:30)
}

func (u *UTCOffset) String() string {
	if u.Minutes != nil {
		return fmt.Sprintf("UTC%s%s:%s", u.Sign, u.Hours, *u.Minutes)
	}
	return fmt.Sprintf("UTC%s%s", u.Sign, u.Hours)
}

// RelativeDateLiteral represents relative date keywords: today, tomorrow, yesterday, now
type RelativeDateLiteral struct {
	Keyword    string // "today", "tomorrow", "yesterday", "now"
	SourceText string
	Range      *Range
}

func (r *RelativeDateLiteral) String() string {
	return fmt.Sprintf("RelativeDateLiteral(%s)", r.Keyword)
}

func (r *RelativeDateLiteral) GetRange() *Range {
	return r.Range
}

// DurationLiteral represents a duration literal: "5 days", "3 hours"
type DurationLiteral struct {
	Value      string // Numeric value ("5", "3.5", etc.)
	Unit       string // Time unit ("days", "hours", "minutes", etc.)
	SourceText string // Original text from source
	Range      *Range
}

func (d *DurationLiteral) String() string {
	return fmt.Sprintf("DurationLiteral(%s %s)", d.Value, d.Unit)
}

func (d *DurationLiteral) GetRange() *Range {
	return d.Range
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

// UnaryOp represents a unary operation (-, +)
type UnaryOp struct {
	Operator string
	Operand  Node
	Range    *Range
}

func (u *UnaryOp) String() string {
	return fmt.Sprintf("UnaryOp(%q, %s)", u.Operator, u.Operand)
}

func (u *UnaryOp) GetRange() *Range {
	return u.Range
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

// FunctionCall represents a function call (avg, sqrt, etc.)
type FunctionCall struct {
	Name      string // Canonical function name: "avg", "sqrt"
	Arguments []Node
	Range     *Range
}

func (f *FunctionCall) String() string {
	return fmt.Sprintf("FunctionCall(%q, %v)", f.Name, f.Arguments)
}

func (f *FunctionCall) GetRange() *Range {
	return f.Range
}
