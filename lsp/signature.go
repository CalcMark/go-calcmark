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
// Calcmark has single-signature functions only, so the per-signature
// activeParameter field from LSP 3.16 is omitted — the top-level
// lspSignatureHelp.ActiveParameter is authoritative.
type lspSignatureInformation struct {
	Label         string                    `json:"label"`
	Documentation any                       `json:"documentation,omitempty"`
	Parameters    []lspParameterInformation `json:"parameters,omitempty"`
}

// lspSignatureHelp mirrors protocol.SignatureHelp's shape. Signatures uses
// a non-nil initialization in signatureHelpForFunction so the wire never
// emits {"signatures":null} — LSP clients expect an array.
type lspSignatureHelp struct {
	Signatures      []lspSignatureInformation `json:"signatures"`
	ActiveSignature *protocol.UInteger        `json:"activeSignature,omitempty"`
	ActiveParameter *protocol.UInteger        `json:"activeParameter,omitempty"`
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

	ctx := extractArgumentContext(lineText, col)
	if ctx.funcName == "" {
		return nil, nil
	}

	help := signatureHelpForFunction(ctx.funcName, ctx.paramIdx, ctx.isNL)
	if help == nil {
		return nil, nil
	}
	return help, nil
}

// signatureHelpForFunction returns signature help for a known function, or nil.
// `isNL` chooses the natural-language alias example as the signature label
// when true (e.g., `grow X by Y over Z months`); otherwise the paren-form
// `Syntax` is used.
func signatureHelpForFunction(funcName string, activeParam int, isNL bool) *lspSignatureHelp {
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
			// NL context: prefer the first parseable alias's `Example`
			// so the signature panel echoes the form the user is
			// typing. Falls back to `NLExample` (some entries set this
			// instead of populating the alias example), then to
			// `Syntax` as a last resort. The paren form is returned
			// unchanged for NL items without an alias example.
			if isNL {
				if ex := firstNLExample(f); ex != "" {
					signature = ex
				} else {
					signature = f.Syntax
				}
			} else {
				signature = f.Syntax
			}
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
			Data: paramSpecToData(p),
		})
	}

	// Clamp activeParameter to a valid index into paramInfos.
	// - Variadic functions declare one parameter but accept N arguments, so a
	//   cursor past the declared param must still point at the variadic slot.
	// - Non-variadic functions with over-applied arguments (user typed extra
	//   commas) must still produce a valid index rather than an out-of-bounds
	//   value that would violate the LSP protocol.
	clamped := max(activeParam, 0)
	if len(paramInfos) > 0 && clamped >= len(paramInfos) {
		clamped = len(paramInfos) - 1
	}
	activeIdx := protocol.UInteger(clamped)

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

func uintPtr(v uint32) *protocol.UInteger {
	u := protocol.UInteger(v)
	return &u
}

// firstNLExample returns the first parseable alias example or the
// feature's NLExample field, whichever is populated. Empty string when
// neither is available — caller should fall back to `Syntax`.
func firstNLExample(f features.Feature) string {
	for _, a := range f.Aliases {
		if a.Parseable && a.Example != "" {
			return a.Example
		}
	}
	return f.NLExample
}
