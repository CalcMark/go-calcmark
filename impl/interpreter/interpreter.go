package interpreter

import (
	"fmt"
	"time"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/CalcMark/go-calcmark/spec/units"
	"github.com/shopspring/decimal"
)

// DirectiveResolver provides access to frontmatter values for @directive references.
// This interface breaks the import cycle between impl/interpreter and spec/document.
type DirectiveResolver interface {
	// ScaleFactor returns the scale factor and true if scale is defined.
	ScaleFactor() (decimal.Decimal, bool)
	// ResolveGlobal returns the typed value of a global and true if it exists.
	ResolveGlobal(name string) (types.Type, bool, error)
}

// Interpreter executes validated AST nodes and produces typed results.
// This is a Go-specific implementation of CalcMark execution.
type Interpreter struct {
	env              *Environment
	directive        DirectiveResolver
	measurement      *units.MeasurementConfig
	fiscalYearStarts *fiscalConfig // nil if not configured
	timeFunc         func() time.Time
	// lineOffset shifts AST node line numbers (which are 1-indexed and
	// block-relative when the parser ran on a single block's source)
	// to doc-absolute, 0-indexed lines. Used by `evalAssignment` to
	// record `definedLines` in the environment for LSP completion
	// position filtering. Zero is the safe default (single-block /
	// whole-doc parses); the document evaluator wires it per-block via
	// `SetLineOffset(blockLineOffset(...))`.
	lineOffset int
}

// NewInterpreter creates a new interpreter with an empty environment.
func NewInterpreter() *Interpreter {
	return &Interpreter{
		env:      NewEnvironment(),
		timeFunc: time.Now,
	}
}

// NewInterpreterWithEnv creates a new interpreter with a pre-populated environment.
func NewInterpreterWithEnv(env *Environment) *Interpreter {
	return &Interpreter{
		env:      env,
		timeFunc: time.Now,
	}
}

// SetTimeFunc overrides the clock used for date evaluation.
// Useful for deterministic testing with pinned dates.
func (interp *Interpreter) SetTimeFunc(f func() time.Time) {
	interp.timeFunc = f
}

// SetLineOffset sets the doc-absolute, 0-indexed line offset used to
// translate block-relative AST line numbers into doc-absolute lines
// when recording variable assignment positions.
//
// For a parser run over a single block's source starting at doc line 5
// (0-indexed), pass 5. The interpreter then records `flour` defined on
// AST line 2 as `definedLines[flour] = 5 + (2-1) = 6` (0-indexed,
// doc-absolute), which the LSP can compare directly to its 0-indexed
// cursor line.
func (interp *Interpreter) SetLineOffset(offset int) {
	interp.lineOffset = offset
}

// now returns the current time from the configured clock.
func (interp *Interpreter) now() time.Time {
	return interp.timeFunc()
}

// SetDirectiveResolver provides frontmatter context for resolving @directive references.
func (interp *Interpreter) SetDirectiveResolver(dr DirectiveResolver) {
	interp.directive = dr
}

// SetMeasurement configures how bare ambiguous unit names are resolved.
// This is a pre-interpreter directive: it affects how unit names are interpreted
// before any evaluation occurs (unlike convert_to which is post-evaluation).
func (interp *Interpreter) SetMeasurement(mc *units.MeasurementConfig) {
	interp.measurement = mc
}

// fiscalConfig holds the fiscal year start month/day plus the FY-label
// convention. labelMode defaults to FYLabelByEndYear (matches Australian
// government year, US tax year, and most publicly traded companies);
// documents can declare `calendar_year_offset: after` to switch to
// FYLabelByStartYear.
type fiscalConfig struct {
	month     int // 1-12
	day       int // 1-31
	labelMode types.FYLabelMode
}

// SetFiscalYearStarts configures the start of the fiscal year using
// the default FY-label convention (FYLabelByEndYear). Existing
// callers stay on this entry point unchanged. Month is 1-12, day is
// 1-31 (typically 1).
func (interp *Interpreter) SetFiscalYearStarts(month, day int) {
	interp.SetFiscalCalendar(month, day, types.FYLabelByEndYear)
}

// SetFiscalCalendar configures the fiscal-year start month/day plus
// the labeling convention. `mode` corresponds to the document's
// `calendar_year_offset` frontmatter setting (`before` →
// FYLabelByEndYear, `after` → FYLabelByStartYear).
func (interp *Interpreter) SetFiscalCalendar(month, day int, mode types.FYLabelMode) {
	interp.fiscalYearStarts = &fiscalConfig{month: month, day: day, labelMode: mode}
}

// resolveUnit maps a bare ambiguous unit name to its qualified form using
// the active measurement conventions. Returns the unit unchanged if no
// measurement config is set or the unit is not ambiguous.
func (interp *Interpreter) resolveUnit(unitName string) string {
	return units.ResolveUnit(unitName, interp.measurement)
}

// Eval executes a list of AST nodes and returns the results.
// Each node produces a typed value.
func (interp *Interpreter) Eval(nodes []ast.Node) ([]types.Type, error) {
	results := make([]types.Type, 0, len(nodes))

	for _, node := range nodes {
		result, err := interp.evalNode(node)
		if err != nil {
			return nil, err
		}
		if result != nil {
			results = append(results, result)
		}
	}

	return results, nil
}

// evalNode evaluates a single AST node.
func (interp *Interpreter) evalNode(node ast.Node) (types.Type, error) {
	if node == nil {
		return nil, nil
	}

	switch n := node.(type) {
	case *ast.Assignment:
		return interp.evalAssignment(n)
	case *ast.Expression:
		// Unwrap expression and evaluate the inner node
		return interp.evalNode(n.Expr)
	case *ast.BinaryOp:
		return interp.evalBinaryOp(n)
	case *ast.ComparisonOp:
		return interp.evalComparisonOp(n)
	case *ast.UnaryOp:
		return interp.evalUnaryOp(n)
	case *ast.Identifier:
		return interp.evalIdentifier(n)
	case *ast.NumberLiteral:
		return interp.evalNumberLiteral(n)
	case *ast.CurrencyLiteral:
		return interp.evalCurrencyLiteral(n)
	case *ast.BooleanLiteral:
		return interp.evalBooleanLiteral(n)
	case *ast.DateLiteral:
		return interp.evalDateLiteral(n)
	case *ast.TimeLiteral:
		return interp.evalTimeLiteral(n)
	case *ast.DurationLiteral:
		return interp.evalDurationLiteral(n)
	case *ast.RelativeDateLiteral:
		return interp.evalRelativeDateLiteral(n)
	case *ast.EndOfExpr:
		return interp.evalEndOfExpr(n)
	case *ast.StartOfExpr:
		return interp.evalStartOfExpr(n)
	case *ast.BetweenExpr:
		return interp.evalBetweenExpr(n)
	case *ast.LengthOfExpr:
		return interp.evalLengthOfExpr(n)
	case *ast.FractionLiteral:
		return interp.evalFractionLiteral(n)
	case *ast.QuantityLiteral:
		return interp.evalQuantityLiteral(n)
	case *ast.RateLiteral:
		return interp.evalRateLiteral(n)
	case *ast.UnitConversion:
		return interp.evalUnitConversion(n)
	case *ast.NapkinConversion:
		return interp.evalNapkinConversion(n)
	case *ast.PreciseConversion:
		return interp.evalPreciseConversion(n)
	case *ast.PercentageOf:
		return interp.evalPercentageOf(n)
	case *ast.AsPercentOf:
		return interp.evalAsPercentOf(n)
	case *ast.FunctionCall:
		return interp.evalFunctionCall(n)
	case *ast.DirectiveRef:
		return interp.evalDirectiveRef(n)
	default:
		return nil, fmt.Errorf("unknown node type: %T", node)
	}
}

// evalDirectiveRef resolves @scale and @globals.name to runtime values.
func (interp *Interpreter) evalDirectiveRef(ref *ast.DirectiveRef) (types.Type, error) {
	if interp.directive == nil {
		return nil, fmt.Errorf("directive @%s used but no frontmatter defined", ref.Directive)
	}

	switch ref.Directive {
	case "scale":
		factor, ok := interp.directive.ScaleFactor()
		if !ok {
			return nil, fmt.Errorf("@scale requires 'scale:' in frontmatter")
		}
		return types.NewNumber(factor), nil

	case "globals":
		value, ok, err := interp.directive.ResolveGlobal(ref.Field)
		if err != nil {
			return nil, fmt.Errorf("@globals.%s: %w", ref.Field, err)
		}
		if !ok {
			return nil, fmt.Errorf("undefined global '%s'", ref.Field)
		}
		return value, nil

	default:
		return nil, fmt.Errorf("unsupported directive '@%s'", ref.Directive)
	}
}

// GetEnvironment returns the interpreter's environment.
func (interp *Interpreter) GetEnvironment() *Environment {
	return interp.env
}
