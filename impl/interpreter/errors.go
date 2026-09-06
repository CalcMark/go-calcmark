package interpreter

import (
	"errors"
	"fmt"

	"github.com/CalcMark/go-calcmark/v2/spec/ast"
)

// CascadingError indicates that evaluation failed because a referenced
// variable has an error from a prior evaluation failure.
type CascadingError struct {
	VarName string // the errored variable that was referenced
	Cause   error  // the original error on that variable
}

func (e *CascadingError) Error() string {
	return fmt.Sprintf("depends on errored variable %q: %s", e.VarName, e.Cause)
}

func (e *CascadingError) Unwrap() error {
	return e.Cause
}

// PositionedError attaches the source range of the expression that
// failed to a runtime error so consumers (LSP, editors) can underline
// the offending token instead of the whole line (go-calcmark#164).
//
// The innermost failing expression wins: evalNode wraps every error
// on the way out, but withPosition never overwrites a position that is
// already present, so an argument-level rejection keeps the argument's
// range even though the enclosing call and statement wrap it again.
type PositionedError struct {
	Range *ast.Range
	Err   error
}

func (e *PositionedError) Error() string {
	return e.Err.Error()
}

func (e *PositionedError) Unwrap() error {
	return e.Err
}

// PositionOf returns the source range err carries, or nil.
//
// It walks the wrap chain from the outside in and stops at a
// CascadingError: a cascade is a fresh error at the *reference* site,
// so the position of its root cause (a different statement, possibly a
// different block) must not leak through. Consumers use this instead
// of errors.As for exactly that reason.
func PositionOf(err error) *ast.Range {
	for err != nil {
		switch e := err.(type) {
		case *PositionedError:
			return e.Range
		case *CascadingError:
			return nil
		}
		err = errors.Unwrap(err)
	}
	return nil
}

// withPosition wraps err with node's range unless err is nil, already
// positioned, or node has no range. Safe to call unconditionally on
// every return path.
func withPosition(node ast.Node, err error) error {
	if err == nil {
		return nil
	}
	if PositionOf(err) != nil {
		return err
	}
	if node == nil {
		return err
	}
	r := node.GetRange()
	if r == nil {
		return err
	}
	return &PositionedError{Range: r, Err: err}
}
