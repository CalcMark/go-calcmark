package ast

import (
	"fmt"
	"slices"
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

// FractionLiteral represents a fraction literal (e.g., "1/3", "7/8", "1/3 cup").
// Numerator and Denominator are stored as int64, parsed once at parse time.
type FractionLiteral struct {
	Numerator   int64  // The numerator value
	Denominator int64  // The denominator value
	Unit        string // Optional unit (e.g., "cup" for "1/3 cup"), empty if dimensionless
	SourceText  string // Original text from source (e.g., "1/3")
	Range       *Range
}

func (f *FractionLiteral) String() string {
	return fmt.Sprintf("FractionLiteral(%d/%d)", f.Numerator, f.Denominator)
}

func (f *FractionLiteral) GetRange() *Range {
	return f.Range
}

// QuantityLiteral represents a number with a unit (e.g., "5 kg", "10 meters").
// When Expr is non-nil, the quantity value comes from evaluating the expression
// (e.g., "@scale meters" uses a DirectiveRef as the value source).
type QuantityLiteral struct {
	Value      string // The numeric value (used when Expr is nil)
	Expr       Node   // Expression providing the value (e.g., DirectiveRef); nil for literal quantities
	Unit       string // The unit identifier
	SourceText string // Original text
	Range      *Range
}

func (q *QuantityLiteral) String() string {
	if q.Expr != nil {
		return fmt.Sprintf("QuantityLiteral(%s %s)", q.Expr.String(), q.Unit)
	}
	return fmt.Sprintf("QuantityLiteral(%s %s)", q.Value, q.Unit)
}

func (q *QuantityLiteral) GetRange() *Range {
	return q.Range
}

// UnitConversion represents explicit unit conversion (e.g., "10 meters in feet").
// For rate conversions (e.g., "10 m/s in inch/s"), TargetTimeUnit is set.
type UnitConversion struct {
	Quantity       Node   // The quantity expression to convert
	TargetUnit     string // The target unit to convert to
	TargetTimeUnit string // For rate conversions: the target time unit (e.g., "s" in "inch/s")
	Range          *Range
}

func (u *UnitConversion) String() string {
	if u.TargetTimeUnit != "" {
		return fmt.Sprintf("UnitConversion(%s in %s/%s)", u.Quantity.String(), u.TargetUnit, u.TargetTimeUnit)
	}
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

// PreciseConversion represents full-precision display (e.g., "1 second as hour as precise").
type PreciseConversion struct {
	Expression Node
	Range      *Range
}

func (n *PreciseConversion) String() string {
	return fmt.Sprintf("PreciseConversion(%s)", n.Expression.String())
}

func (n *PreciseConversion) GetRange() *Range {
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

// AsPercentOf represents "X as % of Y" — computes the ratio X/Y as a Percentage.
// This is the inverse of PercentageOf: "20% of 500" = 100, "100 as % of 500" = 20%.
type AsPercentOf struct {
	Numerator   Node // The value to express as a percentage
	Denominator Node // The reference value (the "whole")
	Range       *Range
}

func (a *AsPercentOf) String() string {
	return fmt.Sprintf("AsPercentOf(%s of %s)", a.Numerator.String(), a.Denominator.String())
}

func (a *AsPercentOf) GetRange() *Range {
	return a.Range
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
	Month string // "Dec", "December"
	Day   string // "25" — the literal day. When the user wrote no
	// day number in source, Day is "1" (lexer default) AND
	// HasExplicitDay is false.
	Year *string // nil if not provided, "2024" if provided

	// HasExplicitDay reports whether a day number was scanned from
	// the source. Discriminates `April 1` (true: specific date) from
	// `April` (false: bare month, semantically a Period). The parser
	// uses this flag to route bare-month forms to RelativeDateLiteral
	// instead of DateLiteral. Lexer-driven so the discriminator
	// doesn't depend on substring inspection of SourceText (which
	// drifts with whitespace / future lexer changes).
	HasExplicitDay bool

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

// EndOfExpr represents `end of <period>` -- the operator that
// resolves a Period-typed inner expression to its last day. The
// inner is a generic Node so any expression that types as Period
// (literal, identifier-bound period, parenthesized period
// expression) parses uniformly. Type-check enforcement of "must be
// a Period" lives in spec/semantic; at the AST layer the constraint
// is unenforced -- a NumberLiteral here is well-formed but
// semantically rejected.
//
// Replaces the previous string-flatten in spec/parser/primary.go
// which produced ast.RelativeDateLiteral{Keyword: "end of " +
// innerKeyword} -- losing the inner's structure, bounding the
// inner-token set to an enumeration, and forcing string-prefix
// dispatch in the interpreter.
type EndOfExpr struct {
	// Period is the inner expression. Must type as Period at semantic
	// check time; runtime panics if non-Period flows through. Parser
	// uses parsePrimary() so precedence is preserved
	// (`end of Q1 + 1 day` parses as `(end of Q1) + 1 day`).
	Period Node

	// SourceText is the original source slice ("end of Q1") for
	// diagnostics + AST round-tripping.
	SourceText string

	Range *Range
}

func (e *EndOfExpr) String() string {
	if e.Period == nil {
		return "EndOfExpr(<nil>)"
	}
	return fmt.Sprintf("EndOfExpr(%s)", e.Period.String())
}

func (e *EndOfExpr) GetRange() *Range {
	return e.Range
}

// StartOfExpr is the symmetric partner of EndOfExpr -- resolves a
// Period to its first day. Parser-level identical to EndOfExpr;
// interpreter just returns Period.Start unchanged. Useful in
// expressions where intent matters (e.g.
// `forecast = start of next fiscal quarter` reads better than the
// implicit `next fiscal quarter` even though they evaluate the
// same).
type StartOfExpr struct {
	Period     Node
	SourceText string
	Range      *Range
}

func (s *StartOfExpr) String() string {
	if s.Period == nil {
		return "StartOfExpr(<nil>)"
	}
	return fmt.Sprintf("StartOfExpr(%s)", s.Period.String())
}

func (s *StartOfExpr) GetRange() *Range {
	return s.Range
}

// BetweenExpr represents `between A and B` / `from A to B` -- a
// user-defined custom Period spanning two Date endpoints. Both
// endpoints are generic Node so any expression typing as Date is
// well-formed at the AST layer; semantic check (spec/semantic) is
// where Date typing + start <= end constraints are enforced. The
// runtime constructs a *types.Period (PeriodCustom kind) from the
// evaluated endpoints.
type BetweenExpr struct {
	// Start is the period's first day; End is its last (closed
	// interval, day precision -- mirrors *types.Period).
	Start Node
	End   Node

	// SourceText is the original source slice for diagnostics +
	// AST round-tripping.
	SourceText string

	Range *Range
}

func (b *BetweenExpr) String() string {
	startStr := "<nil>"
	endStr := "<nil>"
	if b.Start != nil {
		startStr = b.Start.String()
	}
	if b.End != nil {
		endStr = b.End.String()
	}
	return fmt.Sprintf("BetweenExpr(%s, %s)", startStr, endStr)
}

func (b *BetweenExpr) GetRange() *Range {
	return b.Range
}

// LengthOfExpr represents `length of <Period>` (returns Duration)
// and `days in <Period>` (returns Number). Both forms desugar to
// the same node; AsNumber discriminates the return shape.
//
// AST layer doesn't enforce that Period types as Period -- semantic
// (spec/semantic) does. Period field is generic Node so any
// expression typing as Period (literal, variable-bound,
// parenthesized) parses uniformly.
type LengthOfExpr struct {
	// Period is the inner expression. Must type as Period at
	// semantic check time. Parser uses parsePrimary() so precedence
	// is preserved.
	Period Node

	// AsNumber discriminates the surface form: false = `length of`
	// (returns Duration); true = `days in` (returns Number, integer
	// day count).
	AsNumber bool

	// SourceText is the original source slice.
	SourceText string

	Range *Range
}

func (l *LengthOfExpr) String() string {
	form := "length of"
	if l.AsNumber {
		form = "days in"
	}
	if l.Period == nil {
		return fmt.Sprintf("LengthOfExpr(%s <nil>)", form)
	}
	return fmt.Sprintf("LengthOfExpr(%s %s)", form, l.Period.String())
}

func (l *LengthOfExpr) GetRange() *Range {
	return l.Range
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

// DirectiveRef represents a reference to a frontmatter directive (@scale, @globals.name).
type DirectiveRef struct {
	Directive string // "scale" or "globals" (or any identifier — semantic checker validates)
	Field     string // "" for @scale, "tax_rate" for @globals.tax_rate
	Range     *Range
}

func (d *DirectiveRef) String() string {
	if d.Field != "" {
		return fmt.Sprintf("DirectiveRef(@%s.%s)", d.Directive, d.Field)
	}
	return fmt.Sprintf("DirectiveRef(@%s)", d.Directive)
}

func (d *DirectiveRef) GetRange() *Range {
	return d.Range
}

// ContainsScaleRef returns true if the AST node tree contains a
// DirectiveRef with Directive=="scale". Used by the scale transform
// to exempt statements that explicitly reference @scale.
func ContainsScaleRef(node Node) bool {
	if node == nil {
		return false
	}
	switch n := node.(type) {
	case *DirectiveRef:
		return n.Directive == "scale"
	case *Assignment:
		return ContainsScaleRef(n.Value)
	case *Expression:
		return ContainsScaleRef(n.Expr)
	case *BinaryOp:
		return ContainsScaleRef(n.Left) || ContainsScaleRef(n.Right)
	case *UnaryOp:
		return ContainsScaleRef(n.Operand)
	case *ComparisonOp:
		return ContainsScaleRef(n.Left) || ContainsScaleRef(n.Right)
	case *FunctionCall:
		return slices.ContainsFunc(n.Arguments, ContainsScaleRef)
	case *UnitConversion:
		return ContainsScaleRef(n.Quantity)
	case *NapkinConversion:
		return ContainsScaleRef(n.Expression)
	case *PreciseConversion:
		return ContainsScaleRef(n.Expression)
	case *PercentageOf:
		return ContainsScaleRef(n.Percentage) || ContainsScaleRef(n.Value)
	case *AsPercentOf:
		return ContainsScaleRef(n.Numerator) || ContainsScaleRef(n.Denominator)
	case *RateLiteral:
		return ContainsScaleRef(n.Amount)
	case *QuantityLiteral:
		return ContainsScaleRef(n.Expr)
	case *EndOfExpr:
		return ContainsScaleRef(n.Period)
	case *StartOfExpr:
		return ContainsScaleRef(n.Period)
	case *BetweenExpr:
		return ContainsScaleRef(n.Start) || ContainsScaleRef(n.End)
	case *LengthOfExpr:
		return ContainsScaleRef(n.Period)
	default:
		// Leaf nodes: NumberLiteral, FractionLiteral, CurrencyLiteral,
		// DateLiteral, TimeLiteral, DurationLiteral,
		// BooleanLiteral, Identifier, RelativeDateLiteral
		return false
	}
}
