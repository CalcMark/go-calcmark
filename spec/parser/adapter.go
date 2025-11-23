package parser

import (
	"github.com/CalcMark/go-calcmark/spec/ast"
)

// Parse parses CalcMark source code into an AST
// Uses custom recursive descent parser
func Parse(text string) ([]ast.Node, error) {
	p := NewRecursiveDescentParser(text)
	return p.Parse()
}
