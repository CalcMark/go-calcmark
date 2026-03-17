package lsp

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/features"
	"github.com/CalcMark/go-calcmark/spec/lexer"
	"github.com/CalcMark/go-calcmark/spec/semantic"
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

	// Use latest source text (not snapshot — snapshot may be stale during debounce)
	source := ds.getSource()

	// Get the current line text for context-sensitive filtering
	line := int(params.Position.Line)
	col := int(params.Position.Character)
	lineText := getLineText(source, line)

	// Determine prefix (text before cursor on this line, back to last non-identifier char)
	prefix := extractPrefix(lineText, col)

	// Context-sensitive filtering: suppress completions in markdown-classified lines
	if isMarkdownLine(lineText) {
		return nil, nil
	}

	var items []protocol.CompletionItem

	// After "in" or "as" keywords → units + conversion keywords.
	// Check position before the prefix so "as nap|" is detected (not just "as |").
	prefixStart := col - len([]rune(prefix))
	if isAfterUnitKeyword(lineText, prefixStart) {
		items = append(items, unitCompletionItems(prefix)...)
		items = append(items, conversionKeywordItems(prefix)...)
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
		doc := buildFunctionDoc(fn.Name, fn.Description)

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

// conversionKeywordItems returns completion items for keywords valid after "as" or "in".
// Derives the list from the lexer's ReservedKeywords — only includes keywords that are
// meaningful in a conversion context (NAPKIN, PRECISE).
func conversionKeywordItems(prefix string) []protocol.CompletionItem {
	prefix = strings.ToLower(prefix)

	// Token types that are valid conversion modifiers (after "as" or "in")
	conversionTokens := map[lexer.TokenType]bool{
		lexer.NAPKIN:  true,
		lexer.PRECISE: true,
	}

	var items []protocol.CompletionItem
	for name, tokenType := range lexer.ReservedKeywords {
		if !conversionTokens[tokenType] {
			continue
		}
		if prefix != "" && !strings.HasPrefix(name, prefix) {
			continue
		}
		kind := protocol.CompletionItemKindKeyword
		doc := keywordDoc(tokenType)
		items = append(items, protocol.CompletionItem{
			Label:      name,
			Kind:       &kind,
			InsertText: &name,
			Documentation: &protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: doc,
			},
		})
	}
	return items
}

// keywordDoc returns documentation for a keyword token type.
func keywordDoc(tt lexer.TokenType) string {
	switch tt {
	case lexer.NAPKIN:
		return "Human-readable rounded estimate (e.g., `1234567 as napkin` → ~1.2M)"
	case lexer.PRECISE:
		return "Full-precision display, no rounding (e.g., `1 second as hour as precise`)"
	default:
		return ""
	}
}

// buildFunctionDoc creates rich markdown documentation for a function,
// including parameter types, examples, and valid values.
func buildFunctionDoc(funcName, description string) string {
	var b strings.Builder
	b.WriteString(description)

	spec := semantic.GetFunctionSpec(funcName)
	if spec == nil || len(spec.Params) == 0 {
		return b.String()
	}

	b.WriteString("\n\n**Parameters:**\n")
	for _, p := range spec.Params {
		b.WriteString(fmt.Sprintf("\n- `%s`", p.Name))
		if p.Type != "" {
			b.WriteString(fmt.Sprintf(" (%s)", p.Type))
		}
		if p.Optional {
			b.WriteString(" — optional")
		}
		if p.Variadic {
			b.WriteString(" — accepts multiple values")
		}
		if len(p.Examples) > 0 {
			b.WriteString(fmt.Sprintf(": %s", strings.Join(p.Examples, ", ")))
		}
	}

	return b.String()
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
// Uses rune-aware indexing for UTF-8 safety (CalcMark supports Unicode identifiers).
func extractPrefix(lineText string, col int) string {
	runes := []rune(lineText)
	if col > len(runes) {
		col = len(runes)
	}
	// Walk backward from cursor to find start of identifier
	start := col
	for start > 0 && (unicode.IsLetter(runes[start-1]) || unicode.IsDigit(runes[start-1]) || runes[start-1] == '_') {
		start--
	}
	return string(runes[start:col])
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
