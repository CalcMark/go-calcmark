package lsp

import (
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// textDocumentCompletion handles the textDocument/completion request.
func (s *Server) textDocumentCompletion(_ *glsp.Context, _ *protocol.CompletionParams) (any, error) {
	// TODO: implement completion with function, unit, and variable sources
	return nil, nil
}
