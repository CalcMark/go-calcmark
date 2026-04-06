package interpreter

import "fmt"

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
