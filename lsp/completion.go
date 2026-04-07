package lsp

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/spec/features"
	"github.com/CalcMark/go-calcmark/spec/lexer"
	"github.com/CalcMark/go-calcmark/spec/types"
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

	// Use latest source text (not snapshot -- snapshot may be stale during debounce)
	source := ds.getSource()

	// Get the current line text for context-sensitive filtering
	line := int(params.Position.Line)
	col := int(params.Position.Character)
	lineText := getLineText(source, line)

	// Determine prefix (text before cursor on this line, back to last non-identifier char)
	prefix := features.ExtractPrefix(lineText, col)

	// Tokenize the line up to the cursor to determine context.
	// This replaces string heuristics with the real lexer.
	ctx := classifyCompletionContext(lineText, col)

	switch ctx {
	case completionContextMarkdown:
		return nil, nil

	case completionContextAfterUnitKeyword:
		// After "in" or "as" -> units + conversion keywords (napkin, precise)
		var items []protocol.CompletionItem
		items = append(items, unitCompletionItems(prefix)...)
		items = append(items, conversionKeywordItems(prefix)...)
		return items, nil

	default:
		// General context -> functions, units, variables, directives, dates
		var items []protocol.CompletionItem
		items = append(items, functionCompletionItems(prefix)...)
		items = append(items, unitCompletionItems(prefix)...)
		items = append(items, dateCompletionItems(prefix)...)
		if snap.Evaluator != nil {
			items = append(items, variableCompletionItems(snap, prefix, line)...)
		}
		if snap.Document != nil {
			items = append(items, directiveCompletionItems(snap, prefix)...)
		}
		return items, nil
	}
}

// functionCompletionItems returns completion items for built-in functions.
// Delegates to features.FunctionSuggestions then enriches with LSP-specific
// snippets, parameter docs, and markdown documentation.
func functionCompletionItems(prefix string) []protocol.CompletionItem {
	suggestions := features.FunctionSuggestions(prefix, nil) // nil = all registry functions
	var items []protocol.CompletionItem

	for _, s := range suggestions {
		if s.Category == "example" {
			// NL example row -> snippet item
			kind := protocol.CompletionItemKindSnippet
			detail := s.Syntax
			insertText := s.InsertText
			items = append(items, protocol.CompletionItem{
				Label:      s.Name,
				Kind:       &kind,
				Detail:     &detail,
				InsertText: &insertText,
			})
		} else {
			// Function row -> enriched with snippet + docs
			kind := protocol.CompletionItemKindFunction
			detail := s.Syntax
			doc := buildFunctionDoc(s.InsertText, s.Description)
			snippetText := buildFunctionSnippet(s.InsertText)
			snippetFormat := protocol.InsertTextFormatSnippet

			items = append(items, protocol.CompletionItem{
				Label:            s.InsertText,
				Kind:             &kind,
				Detail:           &detail,
				InsertText:       &snippetText,
				InsertTextFormat: &snippetFormat,
				Documentation: &protocol.MarkupContent{
					Kind:  protocol.MarkupKindMarkdown,
					Value: doc,
				},
			})
		}
	}

	return items
}

// completionContext describes the syntactic context at the cursor for completion.
type completionContext int

const (
	completionContextGeneral          completionContext = iota
	completionContextAfterUnitKeyword                   // cursor is after "in" or "as"
	completionContextMarkdown                           // line is markdown, suppress completions
)

// classifyCompletionContext tokenizes the line up to the cursor position and
// determines the completion context from the token stream. This uses the real
// lexer instead of string heuristics.
func classifyCompletionContext(lineText string, col int) completionContext {
	// Truncate line to cursor position for tokenization
	runes := []rune(lineText)
	if col > len(runes) {
		col = len(runes)
	}
	textBeforeCursor := string(runes[:col])

	// Try to tokenize. If the lexer produces no tokens, treat as markdown.
	l := lexer.NewLexer(textBeforeCursor)
	tokens, _ := l.Tokenize()

	// Filter to meaningful tokens (skip NEWLINE, EOF, ERROR)
	var meaningful []lexer.Token
	for _, tok := range tokens {
		switch tok.Type {
		case lexer.NEWLINE, lexer.EOF, lexer.ERROR:
			continue
		default:
			meaningful = append(meaningful, tok)
		}
	}

	if len(meaningful) == 0 {
		// No tokens -> could be blank or pure prose
		if isMarkdownLine(lineText) {
			return completionContextMarkdown
		}
		return completionContextGeneral
	}

	// Check if the line starts with a markdown prefix (headings, lists, blockquotes).
	// The lexer won't catch these since they're not calc syntax.
	if isMarkdownLine(lineText) {
		return completionContextMarkdown
	}

	// Find the last complete token before the cursor.
	// If the user is mid-typing an identifier (prefix), the last token might be
	// the incomplete identifier itself. Look at the token before that.
	last := meaningful[len(meaningful)-1]

	// If the last token is an identifier that ends at the cursor position,
	// it's the prefix being typed. Look at the previous token for context.
	lastEndRune := runeCountStr(textBeforeCursor, last.EndPos)
	if last.Type == lexer.IDENTIFIER && lastEndRune >= col {
		if len(meaningful) >= 2 {
			last = meaningful[len(meaningful)-2]
		} else {
			return completionContextGeneral
		}
	}

	// Check if the context token is AS or IN
	switch last.Type {
	case lexer.AS, lexer.IN:
		return completionContextAfterUnitKeyword
	}

	return completionContextGeneral
}

// runeCountStr returns the number of runes in s[:byteOffset].
func runeCountStr(s string, byteOffset int) int {
	if byteOffset > len(s) {
		byteOffset = len(s)
	}
	return len([]rune(s[:byteOffset]))
}

// conversionKeywordItems returns completion items for keywords valid after "as" or "in".
// Derives the list from the lexer's ReservedKeywords -- only includes keywords that are
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
		return "Human-readable rounded estimate (e.g., `1234567 as napkin` -> ~1.2M)"
	case lexer.PRECISE:
		return "Full-precision display, no rounding (e.g., `1 second as hour as precise`)"
	default:
		return ""
	}
}

// buildFunctionSnippet creates an LSP snippet string for a function.
// E.g., accumulate -> "accumulate(${1:rate}, ${2:duration})"
// Uses parameter names from the function spec, with tab stops for each.
func buildFunctionSnippet(funcName string) string {
	spec := types.GetFunctionSpec(funcName)
	if spec == nil || len(spec.Params) == 0 {
		return funcName + "($0)"
	}

	var params []string
	for i, p := range spec.Params {
		placeholder := p.Name
		if len(p.Examples) > 0 {
			placeholder = p.Examples[0]
		}
		params = append(params, fmt.Sprintf("${%d:%s}", i+1, placeholder))
	}
	return funcName + "(" + strings.Join(params, ", ") + ")"
}

// buildFunctionDoc creates rich markdown documentation for a function,
// including parameter types, examples, and valid values.
func buildFunctionDoc(funcName, description string) string {
	var b strings.Builder
	b.WriteString(description)

	spec := types.GetFunctionSpec(funcName)
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
			b.WriteString(" -- optional")
		}
		if p.Variadic {
			b.WriteString(" -- accepts multiple values")
		}
		if len(p.Examples) > 0 {
			b.WriteString(fmt.Sprintf(": %s", strings.Join(p.Examples, ", ")))
		}
	}

	return b.String()
}

// unitCompletionItems returns completion items for units.
// Delegates to features.UnitSuggestions then converts to protocol.CompletionItem
// with LSP-specific enrichment (kind, documentation).
func unitCompletionItems(prefix string) []protocol.CompletionItem {
	suggestions := features.UnitSuggestions(prefix)
	var items []protocol.CompletionItem

	for _, s := range suggestions {
		kind := protocol.CompletionItemKindUnit
		detail := s.Syntax // symbol
		doc := fmt.Sprintf("%s (%s)", s.Description, s.Category)

		items = append(items, protocol.CompletionItem{
			Label:      s.Name,
			Kind:       &kind,
			Detail:     &detail,
			InsertText: &s.InsertText,
			Documentation: &protocol.MarkupContent{
				Kind:  protocol.MarkupKindPlainText,
				Value: doc,
			},
		})
	}

	return items
}

// dateCompletionItems returns completion items for date keywords.
// Delegates to features.DateSuggestions then converts to protocol.CompletionItem.
func dateCompletionItems(prefix string) []protocol.CompletionItem {
	suggestions := features.DateSuggestions(prefix)
	var items []protocol.CompletionItem

	for _, s := range suggestions {
		kind := protocol.CompletionItemKindKeyword
		detail := s.Syntax
		doc := s.Description

		items = append(items, protocol.CompletionItem{
			Label:      s.Name,
			Kind:       &kind,
			Detail:     &detail,
			InsertText: &s.InsertText,
			Documentation: &protocol.MarkupContent{
				Kind:  protocol.MarkupKindPlainText,
				Value: doc,
			},
		})
	}

	return items
}

// variableCompletionItems returns completion items for variables.
// Known limitation: position filtering is not applied because the LSP's
// Environment does not expose variable definition line numbers.
// The shared VariableSuggestions function supports it when definedLines is provided.
func variableCompletionItems(snap *DocumentSnapshot, prefix string, cursorLine int) []protocol.CompletionItem {
	env := snap.Evaluator.GetEnvironment()
	if env == nil {
		return nil
	}

	// Build vars map: name -> formatted value string
	allVars := env.GetAllVariables()
	vars := make(map[string]string, len(allVars))
	for name, val := range allVars {
		vars[name] = fmt.Sprintf("%v", val)
	}

	// nil definedLines = no position filtering (LSP limitation documented above)
	suggestions := features.VariableSuggestions(vars, prefix, cursorLine, nil)
	var items []protocol.CompletionItem

	for _, s := range suggestions {
		kind := protocol.CompletionItemKindVariable
		detail := s.Description

		items = append(items, protocol.CompletionItem{
			Label:      s.Name,
			Kind:       &kind,
			Detail:     &detail,
			InsertText: &s.InsertText,
		})
	}

	return items
}

// directiveCompletionItems returns completion items for @scale and @globals.field directives.
// Delegates to features.DirectiveSuggestions then converts to protocol.CompletionItem.
func directiveCompletionItems(snap *DocumentSnapshot, prefix string) []protocol.CompletionItem {
	fm := snap.Document.GetFrontmatter()
	if fm == nil {
		return nil
	}

	scaleFactor := ""
	if fm.Scale != nil {
		scaleFactor = fm.Scale.Factor.String()
	}

	suggestions := features.DirectiveSuggestions(prefix, scaleFactor, fm.Globals)
	var items []protocol.CompletionItem
	kind := protocol.CompletionItemKindConstant

	for _, s := range suggestions {
		detail := s.Description
		items = append(items, protocol.CompletionItem{
			Label:      s.Name,
			Kind:       &kind,
			Detail:     &detail,
			InsertText: &s.InsertText,
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
