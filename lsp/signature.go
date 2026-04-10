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

// textDocumentSignatureHelp handles the textDocument/signatureHelp request.
func (s *Server) textDocumentSignatureHelp(_ *glsp.Context, params *protocol.SignatureHelpParams) (*protocol.SignatureHelp, error) {
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

	return signatureHelpForFunction(funcName, paramIdx), nil
}

// signatureHelpForFunction returns signature help for a known function, or nil.
func signatureHelpForFunction(funcName string, activeParam int) *protocol.SignatureHelp {
	// Look up the function spec for parameter info
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
		// Build a signature from the spec
		var params []string
		for _, p := range spec.Params {
			params = append(params, p.Name)
		}
		signature = fmt.Sprintf("%s(%s)", funcName, strings.Join(params, ", "))
	}

	// Build parameter information
	var paramInfos []protocol.ParameterInformation
	for _, p := range spec.Params {
		label := p.Name
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

		paramInfos = append(paramInfos, protocol.ParameterInformation{
			Label: label,
			Documentation: protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: doc,
			},
		})
	}

	activeIdx := protocol.UInteger(activeParam)

	return &protocol.SignatureHelp{
		Signatures: []protocol.SignatureInformation{
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
