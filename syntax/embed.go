package syntax

import (
	_ "embed"
)

//go:generate go run ../cmd/calcmark generate

// SyntaxHighlighterSpec contains the embedded JSON specification for syntax highlighters.
// This file is generated from the Go implementation and validated by tests in lexer/syntax_spec_test.go.
//
// Usage in Go programs:
//
//	import "github.com/CalcMark/go-calcmark/syntax"
//
//	// Get the JSON spec as a string
//	jsonSpec := syntax.SyntaxHighlighterSpec
//
//	// Or as bytes
//	jsonBytes := syntax.SyntaxHighlighterSpecBytes()
//
// This enables CalcMark server implementations to serve the spec over HTTP:
//
//	http.HandleFunc("/syntax", func(w http.ResponseWriter, r *http.Request) {
//	    w.Header().Set("Content-Type", "application/json")
//	    w.Write(syntax.SyntaxHighlighterSpecBytes())
//	})
//
//go:embed SYNTAX_HIGHLIGHTER_SPEC.json
var SyntaxHighlighterSpec string

// SyntaxHighlighterSpecBytes returns the syntax highlighter spec as a byte slice.
// This is convenient for HTTP responses or file operations.
func SyntaxHighlighterSpecBytes() []byte {
	return []byte(SyntaxHighlighterSpec)
}
