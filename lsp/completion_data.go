package lsp

import (
	"strings"

	"github.com/CalcMark/go-calcmark/v2/spec/types"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// wireParamData mirrors ParamSpec on the wire so clients never need to
// duplicate the spec. Shared across CompletionItem.data.params,
// signatureHelp ParameterInformation.data, and hover function-kind
// data.params so that any future ParamSpec field (e.g. Deprecated) lands
// in one place and every wire surface picks it up.
type wireParamData struct {
	Name       string        `json:"name,omitempty"`
	Type       types.ArgType `json:"type"`
	Examples   []string      `json:"examples,omitempty"`
	EnumValues []string      `json:"enumValues,omitempty"`
	Optional   bool          `json:"optional,omitempty"`
	Variadic   bool          `json:"variadic,omitempty"`
}

// paramSpecToData converts a ParamSpec to its wire representation.
func paramSpecToData(p types.ParamSpec) wireParamData {
	return wireParamData{
		Name:       p.Name,
		Type:       p.Type,
		Examples:   p.Examples,
		EnumValues: p.EnumValues,
		Optional:   p.Optional,
		Variadic:   p.Variadic,
	}
}

// paramSpecsToData converts a slice of ParamSpec to wire form.
func paramSpecsToData(params []types.ParamSpec) []wireParamData {
	if len(params) == 0 {
		return nil
	}
	out := make([]wireParamData, len(params))
	for i, p := range params {
		out[i] = paramSpecToData(p)
	}
	return out
}

// completionItemData is the structured payload attached to CompletionItem.Data
// so clients can read function metadata, variable types, and enum value context
// without parsing labels. Serialized to JSON under the LSP-standard "data" field.
//
// Allowed Kind values: "function", "variable", "enum_value", "keyword",
// "example_value", and "unit" (used by hover, see hoverData).
type completionItemData struct {
	Kind         string          `json:"kind"`
	FunctionName string          `json:"functionName,omitempty"`
	Params       []wireParamData `json:"params,omitempty"`
	// VariableType is set on variable completion items to the evaluator's
	// inferred ArgType (number, quantity, rate, duration, percentage).
	VariableType types.ArgType `json:"variableType,omitempty"`
	// ParamName is set on enum_value and example_value items to identify
	// which parameter the value is for (useful for UI labeling).
	ParamName string `json:"paramName,omitempty"`
}

// enumCompletionsForContext returns enum value completions for the active
// function argument, filtered by the current identifier prefix.
//
// Returns an empty slice when:
//   - the function is unknown
//   - the active parameter has no EnumValues
//   - the cursor is inside a string literal (calcmark enums are bare identifiers,
//     so offering them inside "..." would yield invalid syntax)
func enumCompletionsForContext(ctx argumentContext, prefix string) []protocol.CompletionItem {
	if ctx.funcName == "" || ctx.insideString {
		return nil
	}
	spec := types.GetFunctionSpec(ctx.funcName)
	if spec == nil {
		return nil
	}
	param := spec.GetParamAtIndex(ctx.paramIdx)
	if param == nil || len(param.EnumValues) == 0 {
		return nil
	}

	kind := protocol.CompletionItemKindValue
	var items []protocol.CompletionItem
	for _, v := range param.EnumValues {
		if prefix != "" && !strings.HasPrefix(v, prefix) {
			continue
		}
		insertText := v
		detail := param.Name
		items = append(items, protocol.CompletionItem{
			Label:      v,
			Kind:       &kind,
			Detail:     &detail,
			InsertText: &insertText,
			Data: completionItemData{
				Kind:      "enum_value",
				ParamName: param.Name,
			},
		})
	}
	return items
}
