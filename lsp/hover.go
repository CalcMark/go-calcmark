package lsp

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// textDocumentHover handles the textDocument/hover request.
func (s *Server) textDocumentHover(_ *glsp.Context, _ *protocol.HoverParams) (*protocol.Hover, error) {
	// TODO: implement hover with variable values, function signatures, unit descriptions
	return nil, nil
}

// textDocumentDefinition handles the textDocument/definition request.
func (s *Server) textDocumentDefinition(_ *glsp.Context, _ *protocol.DefinitionParams) (any, error) {
	// TODO: implement go-to-definition for variable references
	return nil, nil
}

// textDocumentDocumentSymbol handles the textDocument/documentSymbol request.
func (s *Server) textDocumentDocumentSymbol(_ *glsp.Context, _ *protocol.DocumentSymbolParams) (any, error) {
	// TODO: implement document symbols (variable assignments, headings)
	return nil, nil
}
