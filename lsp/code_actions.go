package lsp

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/v2/spec/semantic"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// textDocumentCodeAction handles the textDocument/codeAction request.
func (s *Server) textDocumentCodeAction(_ *glsp.Context, params *protocol.CodeActionParams) (any, error) {
	ds := s.getDocument(params.TextDocument.URI)
	if ds == nil {
		return nil, nil
	}

	var actions []protocol.CodeAction

	for _, diag := range params.Context.Diagnostics {
		code := diagnosticCode(diag)
		if code != semantic.DiagUndefinedVariable {
			continue
		}

		// Extract suggestions from the diagnostic message.
		// Message format: `undefined variable "foo" — did you mean "bar"?`
		// or: `undefined variable "foo" — did you mean one of: bar, baz?`
		suggestions := extractSuggestions(diag.Message)
		if len(suggestions) == 0 {
			continue
		}

		// The diagnostic range now has precise AST positions from the semantic
		// checker, pointing exactly at the undefined identifier.
		for _, suggestion := range suggestions {
			title := fmt.Sprintf("Replace with '%s'", suggestion)
			kind := protocol.CodeActionKindQuickFix
			isPreferred := len(suggestions) == 1

			edit := protocol.WorkspaceEdit{
				Changes: map[protocol.DocumentUri][]protocol.TextEdit{
					params.TextDocument.URI: {
						{
							Range:   diag.Range,
							NewText: suggestion,
						},
					},
				},
			}

			actions = append(actions, protocol.CodeAction{
				Title:       title,
				Kind:        &kind,
				Diagnostics: []protocol.Diagnostic{diag},
				IsPreferred: &isPreferred,
				Edit:        &edit,
			})
		}
	}

	if len(actions) == 0 {
		return nil, nil
	}
	return actions, nil
}

// diagnosticCode extracts the code string from an LSP diagnostic.
func diagnosticCode(diag protocol.Diagnostic) string {
	if diag.Code == nil {
		return ""
	}
	return fmt.Sprintf("%v", diag.Code.Value)
}

// extractSuggestions parses "did you mean" suggestions from a diagnostic message.
// Handles two formats:
//   - `— did you mean "bar"?` → ["bar"]
//   - `— did you mean one of: bar, baz?` → ["bar", "baz"]
func extractSuggestions(msg string) []string {
	// Single suggestion: `did you mean "suggestion"?`
	const singlePrefix = `did you mean "`
	if _, after, ok := strings.Cut(msg, singlePrefix); ok {
		if suggestion, _, ok := strings.Cut(after, `"`); ok {
			return []string{suggestion}
		}
	}

	// Multiple suggestions: `did you mean one of: a, b, c?`
	const multiPrefix = `did you mean one of: `
	if _, after, ok := strings.Cut(msg, multiPrefix); ok {
		after = strings.TrimSuffix(after, "?")
		parts := strings.Split(after, ", ")
		var result []string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				result = append(result, p)
			}
		}
		return result
	}

	return nil
}
