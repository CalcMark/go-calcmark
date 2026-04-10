package lsp

import (
	"fmt"
	"slices"
	"strings"

	"github.com/CalcMark/go-calcmark/spec/features"
	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// lspParameterInformation is a superset of protocol.ParameterInformation that
// adds a structured Data field. The LSP 3.16 spec does not define `data` on
// ParameterInformation, but the issue explicitly opts into this pragmatic
// extension — clients that don't know about `data` ignore it.
//
// JSON tags match the protocol's field names so the wire shape is a proper
// superset of the standard ParameterInformation.
type lspParameterInformation struct {
	Label         any `json:"label"` // string | [2]UInteger
	Documentation any `json:"documentation,omitempty"`
	Data          any `json:"data,omitempty"`
}

// lspSignatureInformation mirrors protocol.SignatureInformation but references
// lspParameterInformation so the extended Data field propagates to the wire.
type lspSignatureInformation struct {
	Label           string                    `json:"label"`
	Documentation   any                       `json:"documentation,omitempty"`
	Parameters      []lspParameterInformation `json:"parameters,omitempty"`
	ActiveParameter *protocol.UInteger        `json:"activeParameter,omitempty"`
}

// lspSignatureHelp mirrors protocol.SignatureHelp's shape.
type lspSignatureHelp struct {
	Signatures      []lspSignatureInformation `json:"signatures"`
	ActiveSignature *protocol.UInteger        `json:"activeSignature,omitempty"`
	ActiveParameter *protocol.UInteger        `json:"activeParameter,omitempty"`
}

// signatureParamData is the structured payload attached to each parameter.
type signatureParamData struct {
	Type       types.ArgType `json:"type"`
	Examples   []string      `json:"examples,omitempty"`
	EnumValues []string      `json:"enumValues,omitempty"`
	Optional   bool          `json:"optional,omitempty"`
	Variadic   bool          `json:"variadic,omitempty"`
}

// textDocumentSignatureHelp handles the textDocument/signatureHelp request.
//
// Returns a nil *protocol.SignatureHelp to satisfy the glsp handler struct's
// field type. The real response is produced by signatureHelpHandle and routed
// through the interceptingHandler wrapper so the extended lspSignatureHelp
// shape (with per-parameter `data`) reaches the wire intact.
func (s *Server) textDocumentSignatureHelp(_ *glsp.Context, _ *protocol.SignatureHelpParams) (*protocol.SignatureHelp, error) {
	return nil, nil
}

// signatureHelpHandle is the actual signatureHelp implementation. Called from
// interceptingHandler.Handle when the incoming method is
// textDocument/signatureHelp. Returns `any` so the custom wire shape survives.
func (s *Server) signatureHelpHandle(params *protocol.SignatureHelpParams) (any, error) {
	ds := s.getDocument(params.TextDocument.URI)
	if ds == nil {
		return nil, nil
	}

	source := ds.getSource()

	line := int(params.Position.Line)
	col := int(params.Position.Character)
	lineText := getLineText(source, line)

	funcName, paramIdx := extractFunctionContext(lineText, col)
	if funcName == "" {
		return nil, nil
	}

	help := signatureHelpForFunction(funcName, paramIdx)
	if help == nil {
		return nil, nil
	}
	return help, nil
}

// signatureHelpForFunction returns signature help for a known function, or nil.
func signatureHelpForFunction(funcName string, activeParam int) *lspSignatureHelp {
	spec := types.GetFunctionSpec(funcName)
	if spec == nil {
		return nil
	}

	// Find the feature for its signature string and description
	var signature string
	var description string
	registry := features.DefaultRegistry()
	for _, f := range registry.ByCategory(features.CategoryFunction) {
		if f.Name == funcName || slices.Contains(f.Synonyms, funcName) {
			signature = f.Syntax
			description = f.Description
			break
		}
	}

	if signature == "" {
		var params []string
		for _, p := range spec.Params {
			params = append(params, p.Name)
		}
		signature = fmt.Sprintf("%s(%s)", funcName, strings.Join(params, ", "))
	}

	// Build parameter information — retain existing markdown Documentation for
	// backward compat with clients that haven't upgraded to read `data`.
	paramInfos := make([]lspParameterInformation, 0, len(spec.Params))
	for _, p := range spec.Params {
		doc := fmt.Sprintf("Type: %s", p.Type)
		if len(p.Examples) > 0 {
			doc += fmt.Sprintf("\n\nExamples: %s", strings.Join(p.Examples, ", "))
		}
		if p.Optional {
			doc += "\n\n(optional)"
		}
		if p.Variadic {
			doc += "\n\n(accepts multiple values)"
		}

		paramInfos = append(paramInfos, lspParameterInformation{
			Label: p.Name,
			Documentation: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: doc,
			},
			Data: signatureParamData{
				Type:       p.Type,
				Examples:   p.Examples,
				EnumValues: p.EnumValues,
				Optional:   p.Optional,
				Variadic:   p.Variadic,
			},
		})
	}

	activeIdx := protocol.UInteger(activeParam)

	return &lspSignatureHelp{
		Signatures: []lspSignatureInformation{
			{
				Label:      signature,
				Parameters: paramInfos,
				Documentation: protocol.MarkupContent{
					Kind:  protocol.MarkupKindMarkdown,
					Value: description,
				},
			},
		},
		ActiveSignature: uintPtr(0),
		ActiveParameter: &activeIdx,
	}
}

// extractFunctionContext finds the function name and active parameter index
// at the given cursor position. Returns ("", -1) if the cursor is not inside
// a function call.
//
// Thin adapter over extractArgumentContext preserved for existing callers
// that don't need string-literal awareness.
func extractFunctionContext(lineText string, col int) (string, int) {
	ctx := extractArgumentContext(lineText, col)
	return ctx.funcName, ctx.paramIdx
}

func uintPtr(v uint32) *protocol.UInteger {
	u := protocol.UInteger(v)
	return &u
}
