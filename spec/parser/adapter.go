package parser

import (
	"github.com/CalcMark/go-calcmark/spec/ast"
)

// Parse parses CalcMark source code into an AST
// Uses custom recursive descent parser
// NOTE: This does NOT perform semantic validation (like variable redefinition checks).
// Semantic validation happens during document evaluation where full environment context is available.
func Parse(text string) ([]ast.Node, error) {
	p := NewRecursiveDescentParser(text)
	return p.Parse()
}
