package lsp

import (
	"encoding/json"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// interceptingHandler wraps a protocol.Handler and intercepts methods whose
// native glsp return types are too narrow for our extended wire shapes.
//
// Today only textDocument/signatureHelp is intercepted: the protocol
// ParameterInformation struct has no `data` field, so we produce our own
// lspSignatureHelp shape and let JSON marshaling handle the extra field.
// Clients that don't know about `data` ignore it (standard JSON-over-LSP
// behavior for unknown fields).
type interceptingHandler struct {
	inner  *protocol.Handler
	server *Server
}

// Handle satisfies the glsp.Handler interface. Called once per incoming JSON-
// RPC request by the glsp server.
func (h *interceptingHandler) Handle(ctx *glsp.Context) (r any, validMethod bool, validParams bool, err error) {
	if ctx.Method == protocol.MethodTextDocumentSignatureHelp && h.inner.TextDocumentSignatureHelp != nil {
		validMethod = true
		var params protocol.SignatureHelpParams
		if err = json.Unmarshal(ctx.Params, &params); err != nil {
			return nil, true, false, err
		}
		validParams = true
		r, err = h.server.signatureHelpHandle(&params)
		return r, validMethod, validParams, err
	}
	return h.inner.Handle(ctx)
}
