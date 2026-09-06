package interpreter

import (
	"errors"
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/ast"
	"github.com/CalcMark/go-calcmark/v2/spec/parser"
)

// Runtime errors carry the source range of the expression that failed
// so editors can underline the offending token, not just the line
// (go-calcmark#164). Argument-type rejections point at the argument;
// operator errors point at the operation.

func parseOne(t *testing.T, src string) ast.Node {
	t.Helper()
	nodes, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("Parse(%q) error = %v", src, err)
	}
	if len(nodes) != 1 {
		t.Fatalf("Parse(%q) returned %d nodes, want 1", src, len(nodes))
	}
	return nodes[0]
}

func evalExpectingPositioned(t *testing.T, node ast.Node) *PositionedError {
	t.Helper()
	interp := newTestInterpreterWithClock(testClock)
	_, err := interp.Eval([]ast.Node{node})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	var pe *PositionedError
	if !errors.As(err, &pe) {
		t.Fatalf("error %v (%T) does not carry a position", err, err)
	}
	if pe.Range == nil {
		t.Fatal("PositionedError has a nil Range")
	}
	return pe
}

func TestPositionedError_GrowArgumentTypeRejection_PointsAtArgument(t *testing.T) {
	node := parseOne(t, "x = grow(100, 20%, 5)\n")
	call := node.(*ast.Assignment).Value.(*ast.FunctionCall)
	want := call.Arguments[1].GetRange()

	pe := evalExpectingPositioned(t, node)
	if *pe.Range != *want {
		t.Errorf("range = %s, want the increment argument at %s", pe.Range, want)
	}
	if pe.Error() != "grow: increment must be a number, quantity, or currency — got percentage" {
		t.Errorf("message changed: %q", pe.Error())
	}
}

func TestPositionedError_CompoundRateRejection_PointsAtArgument(t *testing.T) {
	node := parseOne(t, "x = compound($100, 2 weeks, 5)\n")
	call := node.(*ast.Assignment).Value.(*ast.FunctionCall)
	want := call.Arguments[1].GetRange()

	pe := evalExpectingPositioned(t, node)
	if *pe.Range != *want {
		t.Errorf("range = %s, want the rate argument at %s", pe.Range, want)
	}
}

func TestPositionedError_GrowPeriodsRejection_PointsAtArgument(t *testing.T) {
	node := parseOne(t, "x = grow(100, 10, 3 months)\n")
	call := node.(*ast.Assignment).Value.(*ast.FunctionCall)
	want := call.Arguments[2].GetRange()

	pe := evalExpectingPositioned(t, node)
	if *pe.Range != *want {
		t.Errorf("range = %s, want the periods argument at %s", pe.Range, want)
	}
}

func TestPositionedError_CapacityBufferRejection_PointsAtArgument(t *testing.T) {
	node := parseOne(t, "x = capacity(100 GB, 10 GB, disk, 2 weeks)\n")
	call := node.(*ast.Assignment).Value.(*ast.FunctionCall)
	want := call.Arguments[3].GetRange()

	pe := evalExpectingPositioned(t, node)
	if *pe.Range != *want {
		t.Errorf("range = %s, want the buffer argument at %s", pe.Range, want)
	}
}

func TestPositionedError_OperatorError_PointsAtOperation(t *testing.T) {
	node := parseOne(t, "x = 1 + $5 / $2\n")
	// The failing sub-expression is `$5 / $2`, the right operand of `+`.
	outer := node.(*ast.Assignment).Value.(*ast.BinaryOp)
	want := outer.Right.GetRange()

	pe := evalExpectingPositioned(t, node)
	if *pe.Range != *want {
		t.Errorf("range = %s, want the failing division at %s", pe.Range, want)
	}
}

func TestPositionedError_CascadingError_PointsAtReferenceAndStillUnwraps(t *testing.T) {
	interp := newTestInterpreterWithClock(testClock)
	nodes, err := parser.Parse("a = 1 / 0\nb = a + 1\n")
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}
	if _, err := interp.Eval(nodes[:1]); err == nil {
		t.Fatal("expected division by zero")
	}
	interp.env.SetError("a", errors.New("division by zero"))
	_, err = interp.Eval(nodes[1:])
	if err == nil {
		t.Fatal("expected cascading error")
	}
	var casc *CascadingError
	if !errors.As(err, &casc) || casc.VarName != "a" {
		t.Fatalf("cascading error not reachable through errors.As: %v", err)
	}
	var pe *PositionedError
	if !errors.As(err, &pe) {
		t.Fatalf("cascading error carries no position: %v", err)
	}
	ident := nodes[1].(*ast.Assignment).Value.(*ast.BinaryOp).Left
	if *pe.Range != *ident.GetRange() {
		t.Errorf("range = %s, want the errored reference at %s", pe.Range, ident.GetRange())
	}
}

func TestPositionOf_StopsAtCascadingError(t *testing.T) {
	causeRange := &ast.Range{Start: ast.Position{Line: 1, Column: 5}, End: ast.Position{Line: 1, Column: 10}}
	cause := &PositionedError{Range: causeRange, Err: errors.New("division by zero")}
	casc := &CascadingError{VarName: "a", Cause: cause}
	if got := PositionOf(casc); got != nil {
		t.Errorf("cascading error leaked its cause's position %v", got)
	}
	refRange := &ast.Range{Start: ast.Position{Line: 3, Column: 5}, End: ast.Position{Line: 3, Column: 6}}
	err := withPosition(&ast.Identifier{Name: "a", Range: refRange}, casc)
	if got := PositionOf(err); got == nil || *got != *refRange {
		t.Errorf("cascade position = %v, want the reference site %v", got, refRange)
	}
}

func TestPositionedError_DoesNotWrapNil(t *testing.T) {
	if withPosition(&ast.Identifier{Name: "x"}, nil) != nil {
		t.Error("withPosition(node, nil) must return nil")
	}
}

func TestPositionedError_InnermostPositionWins(t *testing.T) {
	inner := &ast.Range{Start: ast.Position{Line: 1, Column: 5}, End: ast.Position{Line: 1, Column: 8}}
	outer := &ast.Range{Start: ast.Position{Line: 1, Column: 1}, End: ast.Position{Line: 1, Column: 20}}
	err := withPosition(&ast.Identifier{Name: "x", Range: inner}, errors.New("boom"))
	err = withPosition(&ast.Identifier{Name: "y", Range: outer}, err)
	var pe *PositionedError
	if !errors.As(err, &pe) || *pe.Range != *inner {
		t.Errorf("outer wrap overwrote the inner position: %v", err)
	}
	if err.Error() != "boom" {
		t.Errorf("message = %q, want unchanged", err.Error())
	}
}
