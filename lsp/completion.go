package lsp

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/features"
	"github.com/CalcMark/go-calcmark/spec/units"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// textDocumentCompletion handles the textDocument/completion request.
func (s *Server) textDocumentCompletion(_ *glsp.Context, params *protocol.CompletionParams) (any, error) {
	ds := s.getDocument(params.TextDocument.URI)
	if ds == nil {
		return nil, nil
	}

	snap := ds.getSnapshot()
	if snap == nil {
		return nil, nil
	}

	// Get the current line text for context-sensitive filtering
	line := int(params.Position.Line)
	col := int(params.Position.Character)
	lineText := getLineText(snap.Source, line)

	// Determine prefix (text before cursor on this line, back to last non-identifier char)
	prefix := extractPrefix(lineText, col)

	// Context-sensitive filtering: suppress completions in markdown-classified lines
	if isMarkdownLine(lineText) {
		return nil, nil
	}

	var items []protocol.CompletionItem

	// After "in" or "as" keywords → units only
	if isAfterUnitKeyword(lineText, col) {
		items = append(items, unitCompletionItems(prefix)...)
		return items, nil
	}

	// Functions
	items = append(items, functionCompletionItems(prefix)...)

	// Units
	items = append(items, unitCompletionItems(prefix)...)

	// Variables from the evaluated environment
	if snap.Evaluator != nil {
		items = append(items, variableCompletionItems(snap, prefix, line)...)
	}

	return items, nil
}

// functionCompletionItems returns completion items for built-in functions.
func functionCompletionItems(prefix string) []protocol.CompletionItem {
	prefix = strings.ToLower(prefix)
	var items []protocol.CompletionItem

	for _, fn := range interpreter.BuiltinFunctions {
		if prefix != "" && !strings.HasPrefix(strings.ToLower(fn.Name), prefix) {
			// Check synonyms too
			matched := false
			for _, syn := range fn.Synonyms {
				if strings.HasPrefix(strings.ToLower(syn), prefix) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		kind := protocol.CompletionItemKindFunction
		detail := fn.Signature
		doc := fn.Description

		items = append(items, protocol.CompletionItem{
			Label:      fn.Name,
			Kind:       &kind,
			Detail:     &detail,
			InsertText: &fn.Name,
			Documentation: &protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: doc,
			},
		})
	}

	// NL function aliases from the feature registry
	registry := features.NewRegistry()
	for _, f := range registry.ByCategory(features.CategoryFunction) {
		for _, alias := range f.Aliases {
			if !alias.Parseable || alias.Example == "" {
				continue
			}
			aliasName := strings.ReplaceAll(alias.Name, "...", " ")
			firstWord := aliasName
			if before, _, ok := strings.Cut(aliasName, " "); ok {
				firstWord = before
			}
			if prefix != "" && !strings.HasPrefix(strings.ToLower(firstWord), prefix) {
				continue
			}

			kind := protocol.CompletionItemKindSnippet
			detail := alias.Example
			items = append(items, protocol.CompletionItem{
				Label:      aliasName,
				Kind:       &kind,
				Detail:     &detail,
				InsertText: &alias.Example,
			})
		}
	}

	return items
}

// unitCompletionItems returns completion items for units.
func unitCompletionItems(prefix string) []protocol.CompletionItem {
	prefix = strings.ToLower(prefix)
	var items []protocol.CompletionItem
	seen := make(map[string]bool)

	for _, unit := range units.StandardUnits {
		if seen[unit.Canonical] {
			continue
		}

		matched := prefix == "" ||
			strings.HasPrefix(strings.ToLower(unit.Canonical), prefix) ||
			strings.HasPrefix(strings.ToLower(unit.Symbol), prefix)
		if !matched {
			for _, alias := range unit.Aliases {
				if strings.HasPrefix(strings.ToLower(alias), prefix) {
					matched = true
					break
				}
			}
		}

		if matched {
			seen[unit.Canonical] = true
			kind := protocol.CompletionItemKindUnit
			detail := unit.Symbol
			doc := fmt.Sprintf("%s (%s)", unit.Description, unit.Quantity)

			items = append(items, protocol.CompletionItem{
				Label:      unit.Canonical,
				Kind:       &kind,
				Detail:     &detail,
				InsertText: &unit.Canonical,
				Documentation: &protocol.MarkupContent{
					Kind:  protocol.MarkupKindPlainText,
					Value: doc,
				},
			})
		}
	}

	return items
}

// variableCompletionItems returns completion items for variables defined above the cursor line.
func variableCompletionItems(snap *DocumentSnapshot, prefix string, cursorLine int) []protocol.CompletionItem {
	prefix = strings.ToLower(prefix)
	var items []protocol.CompletionItem

	env := snap.Evaluator.GetEnvironment()
	if env == nil {
		return nil
	}

	vars := env.GetAllVariables()
	for name, val := range vars {
		if prefix != "" && !strings.HasPrefix(strings.ToLower(name), prefix) {
			continue
		}

		kind := protocol.CompletionItemKindVariable
		detail := fmt.Sprintf("%v", val)

		items = append(items, protocol.CompletionItem{
			Label:      name,
			Kind:       &kind,
			Detail:     &detail,
			InsertText: &name,
		})
	}

	return items
}

// getLineText returns the text of a specific 0-indexed line from the source.
func getLineText(source string, line int) string {
	lines := strings.Split(source, "\n")
	if line < 0 || line >= len(lines) {
		return ""
	}
	return lines[line]
}

// extractPrefix extracts the identifier prefix before the cursor position.
func extractPrefix(lineText string, col int) string {
	if col > len(lineText) {
		col = len(lineText)
	}
	// Walk backward from cursor to find start of identifier
	start := col
	for start > 0 {
		ch := lineText[start-1]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' {
			start--
		} else {
			break
		}
	}
	return lineText[start:col]
}

// isMarkdownLine returns true if the line appears to be markdown (not a calculation).
// Conservative heuristic: lines starting with # or markdown prefixes, or lines
// without = and without known calculation patterns.
func isMarkdownLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return true
	}
	// Explicit markdown indicators
	if strings.HasPrefix(trimmed, "#") ||
		strings.HasPrefix(trimmed, ">") ||
		strings.HasPrefix(trimmed, "- ") ||
		strings.HasPrefix(trimmed, "* ") {
		return true
	}
	return false
}

// isAfterUnitKeyword returns true if the cursor is positioned after "in " or "as ".
func isAfterUnitKeyword(lineText string, col int) bool {
	if col > len(lineText) {
		col = len(lineText)
	}
	before := strings.ToLower(strings.TrimSpace(lineText[:col]))
	return strings.HasSuffix(before, " in ") ||
		strings.HasSuffix(before, " as ") ||
		strings.HasSuffix(before, " in") ||
		strings.HasSuffix(before, " as")
}
