package interpreter

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/v2/spec/ast"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
	"github.com/shopspring/decimal"
)

func decimalFromInt(n int) decimal.Decimal {
	return decimal.NewFromInt(int64(n))
}

// Arrays come from named tables (go-calcmark#118). This file holds the
// three things the interpreter does with them: read a column
// (MemberAccess), combine them element-wise with scalar broadcasting,
// and reduce them with the aggregate functions.

// evalMemberAccess resolves `table.column` to the column's Array.
func (interp *Interpreter) evalMemberAccess(m *ast.MemberAccess) (types.Type, error) {
	ident, ok := m.Object.(*ast.Identifier)
	if !ok {
		return nil, fmt.Errorf("dot access is only supported on a table name (e.g., rates.rate)")
	}
	obj, err := interp.evalIdentifier(ident)
	if err != nil {
		return nil, withPosition(ident, err)
	}
	table, ok := obj.(*types.Table)
	if !ok {
		return nil, withPosition(ident, fmt.Errorf(
			"%q is not a table (it is a %s) — dot access reads a column of a named table declared with <!-- table: %s (...) -->",
			ident.Name, types.TypeNameOf(obj), ident.Name))
	}
	col, ok := table.Column(m.Field)
	if !ok {
		return nil, fmt.Errorf("column %q not found in table %q (columns: %s)",
			m.Field, table.Name, strings.Join(table.ColumnOrder, ", "))
	}
	return col, nil
}

// evalArrayBinaryOp applies a binary operator element-wise. Two arrays
// must have the same length; an array and a scalar broadcast the scalar
// to every element. Each element pair goes through the ordinary
// evalBinaryOperation, so every scalar rule (units, currencies, rates)
// applies unchanged.
func evalArrayBinaryOp(left, right types.Type, operator string) (types.Type, error) {
	la, lok := left.(*types.Array)
	ra, rok := right.(*types.Array)
	if lok {
		if err := rejectTextArray(la, operator); err != nil {
			return nil, err
		}
	}
	if rok {
		if err := rejectTextArray(ra, operator); err != nil {
			return nil, err
		}
	}
	var n int
	switch {
	case lok && rok:
		if la.Len() != ra.Len() {
			return nil, fmt.Errorf("cannot combine arrays of different lengths (%d and %d) — element-wise arithmetic needs one value per row on both sides",
				la.Len(), ra.Len())
		}
		n = la.Len()
	case lok:
		n = la.Len()
	default:
		n = ra.Len()
	}
	out := make([]types.Type, n)
	for i := 0; i < n; i++ {
		l, r := left, right
		if lok {
			l = la.Elements[i]
		}
		if rok {
			r = ra.Elements[i]
		}
		v, err := evalBinaryOperation(l, r, operator)
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i+1, err)
		}
		out[i] = v
	}
	return types.NewArray(out)
}

// evalArrayUnaryOp applies a unary operator to every element.
func evalArrayUnaryOp(arr *types.Array, operator string) (types.Type, error) {
	if err := rejectTextArray(arr, operator); err != nil {
		return nil, err
	}
	out := make([]types.Type, arr.Len())
	for i, el := range arr.Elements {
		v, err := evalUnaryOperation(el, operator)
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", i+1, err)
		}
		out[i] = v
	}
	return types.NewArray(out)
}

func rejectTextArray(arr *types.Array, operator string) error {
	if arr.ElementType == "Text" {
		sample := ""
		if arr.Len() > 0 {
			sample = fmt.Sprintf(" (values like %q)", arr.Elements[0].String())
		}
		return fmt.Errorf("cannot apply %q to a text column%s — text cells are display-only", operator, sample)
	}
	return nil
}

// singleArrayArg reports whether a call was given exactly one array —
// the aggregate form of sum/avg/min/max/count.
func singleArrayArg(args []types.Type) (*types.Array, bool) {
	if len(args) != 1 {
		return nil, false
	}
	arr, ok := args[0].(*types.Array)
	return arr, ok
}

// rejectMixedArrayArgs refuses `sum(array, 1)`: a call is either one
// array or several scalars, never both (R11).
func rejectMixedArrayArgs(funcName string, args []types.Type) error {
	if len(args) < 2 {
		return nil
	}
	for _, a := range args {
		if _, ok := a.(*types.Array); ok {
			return fmt.Errorf("%s(): pass a single array or several scalar values, not a mix — e.g. %s(rates.rate) or %s(a, b, c)",
				funcName, funcName, funcName)
		}
	}
	return nil
}

// aggregateArray reduces an array with the named aggregate.
func aggregateArray(funcName string, arr *types.Array) (types.Type, error) {
	if funcName == "count" {
		return types.NewNumber(decimalFromInt(arr.Len())), nil
	}
	if arr.Len() == 0 {
		return nil, fmt.Errorf("%s(): the array is empty — the table has no data rows", funcName)
	}
	if arr.ElementType == "Text" {
		return nil, fmt.Errorf("%s(): cannot aggregate a text column (values like %q); only count() works on text",
			funcName, arr.Elements[0].String())
	}
	switch funcName {
	case "sum":
		return aggregateValues("sum", arr.Elements)
	case "avg":
		return evalAverage(arr.Elements)
	case "min", "max":
		return extremum(funcName, arr.Elements)
	}
	return nil, fmt.Errorf("%s(): unsupported aggregate", funcName)
}

// extremum returns the smallest (min) or largest (max) value, comparing
// through the ordinary comparison operator so units and currencies
// follow the same rules as `<` and `>`.
func extremum(funcName string, vals []types.Type) (types.Type, error) {
	if len(vals) == 0 {
		return nil, fmt.Errorf("%s() requires at least one value", funcName)
	}
	operator := "<"
	if funcName == "max" {
		operator = ">"
	}
	best := vals[0]
	for _, v := range vals[1:] {
		res, err := evalComparison(v, best, operator)
		if err != nil {
			return nil, fmt.Errorf("%s(): %w", funcName, err)
		}
		if b, ok := res.(*types.Boolean); ok && b.Value {
			best = v
		}
	}
	return best, nil
}

func evalMinFunc(interp *Interpreter, f *ast.FunctionCall) (types.Type, error) {
	return evalExtremumFunc(interp, f, "min")
}

func evalMaxFunc(interp *Interpreter, f *ast.FunctionCall) (types.Type, error) {
	return evalExtremumFunc(interp, f, "max")
}

func evalExtremumFunc(interp *Interpreter, f *ast.FunctionCall, funcName string) (types.Type, error) {
	args, err := interp.evalAllArgs(f)
	if err != nil {
		return nil, err
	}
	if arr, ok := singleArrayArg(args); ok {
		return aggregateArray(funcName, arr)
	}
	if err := rejectMixedArrayArgs(funcName, args); err != nil {
		return nil, err
	}
	return extremum(funcName, args)
}

func evalCountFunc(interp *Interpreter, f *ast.FunctionCall) (types.Type, error) {
	args, err := interp.evalAllArgs(f)
	if err != nil {
		return nil, err
	}
	if arr, ok := singleArrayArg(args); ok {
		return aggregateArray("count", arr)
	}
	if err := rejectMixedArrayArgs("count", args); err != nil {
		return nil, err
	}
	return types.NewNumber(decimalFromInt(len(args))), nil
}
