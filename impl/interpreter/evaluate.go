package interpreter

import (
	"github.com/CalcMark/go-calcmark/spec/parser"
)

// Evaluate is a convenience wrapper for parsing and evaluating CalcMark source.
// This provides API compatibility with the old evaluator package.
func Evaluate(source string, env *Environment) error {
	nodes, err := parser.Parse(source)
	if err != nil {
		return err
	}

	interp := NewInterpreterWithEnv(env)
	_, err = interp.Eval(nodes)
	return err
}
