package lsp

import (
	"encoding/json"

	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// interceptingHandler wraps a protocol.Handler and intercepts methods whose
// native glsp return types are too narrow for our extended wire shapes.
//
// Intercepted methods:
//
//   - textDocument/signatureHelp — protocol.ParameterInformation has no
//     `data` field, so we produce lspSignatureHelp with per-parameter data.
//   - textDocument/hover — protocol.Hover has no `data` field, so we produce
//     lspHover with a sibling `data` field for clients that want structured
//     content without parsing markdown.
//
// Clients that don't know about `data` ignore it (standard JSON-over-LSP
// behavior for unknown fields).
type interceptingHandler struct {
	inner  *protocol.Handler
	server *Server
}

// Handle satisfies the glsp.Handler interface. Called once per incoming JSON-
// RPC request by the glsp server. NewServer always registers the stub
// protocol.Handler fields for the intercepted methods, so the intercept
// unconditionally handles signatureHelp and hover; other methods fall through
// to the inner handler.
func (h *interceptingHandler) Handle(ctx *glsp.Context) (r any, validMethod bool, validParams bool, err error) {
	switch ctx.Method {
	case protocol.MethodTextDocumentSignatureHelp:
		validMethod = true
		var params protocol.SignatureHelpParams
		if err = json.Unmarshal(ctx.Params, &params); err != nil {
			return nil, true, false, err
		}
		validParams = true
		r, err = h.server.signatureHelpHandle(&params)
		return r, validMethod, validParams, err

	case protocol.MethodTextDocumentHover:
		validMethod = true
		var params protocol.HoverParams
		if err = json.Unmarshal(ctx.Params, &params); err != nil {
			return nil, true, false, err
		}
		validParams = true
		r, err = h.server.hoverHandle(&params)
		return r, validMethod, validParams, err
	}
	return h.inner.Handle(ctx)
}
