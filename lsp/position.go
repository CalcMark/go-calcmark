// Package lsp bridges CalcMark's spec/impl/format layers to the Language
// Server Protocol. Dependencies flow inward: lsp → spec, lsp → impl,
// lsp → format. No package should ever import lsp.
package lsp

import (
	"github.com/CalcMark/go-calcmark/spec/ast"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// ToLSPPosition converts a CalcMark 1-indexed position to a 0-indexed LSP position.
func ToLSPPosition(pos ast.Position) protocol.Position {
	return protocol.Position{
		Line:      protocol.UInteger(max(pos.Line-1, 0)),
		Character: protocol.UInteger(max(pos.Column-1, 0)),
	}
}

// ToLSPRange converts a CalcMark Range to an LSP Range.
func ToLSPRange(r *ast.Range) protocol.Range {
	if r == nil {
		return protocol.Range{}
	}
	return protocol.Range{
		Start: ToLSPPosition(r.Start),
		End:   ToLSPPosition(r.End),
	}
}
