package lsp

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/CalcMark/go-calcmark/spec/features"
	"github.com/CalcMark/go-calcmark/spec/lexer"
	"github.com/CalcMark/go-calcmark/spec/types"
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

	// Tokenize the line up to the cursor to determine context.
	// This replaces string heuristics with the real lexer.
	ctx := classifyCompletionContext(lineText, col)

	switch ctx {
	case completionContextMarkdown:
		return nil, nil

	case completionContextAfterUnitKeyword:
		// After "in" or "as" → units + conversion keywords (napkin, precise)
		var items []protocol.CompletionItem
		items = append(items, unitCompletionItems(prefix)...)
		items = append(items, conversionKeywordItems(prefix)...)
		return items, nil

	default:
		// General context → functions, units, variables, directives
		var items []protocol.CompletionItem
		items = append(items, functionCompletionItems(prefix)...)
		items = append(items, unitCompletionItems(prefix)...)
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
func functionCompletionItems(prefix string) []protocol.CompletionItem {
	prefix = strings.ToLower(prefix)
	var items []protocol.CompletionItem

	registry := features.DefaultRegistry()
	for _, f := range registry.ByCategory(features.CategoryFunction) {
		if prefix != "" && !features.MatchesPrefix(f.Name, prefix) {
			// Check synonyms too
			matched := false
			for _, syn := range f.Synonyms {
				if features.MatchesPrefix(syn, prefix) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		kind := protocol.CompletionItemKindFunction
		detail := f.Syntax
		doc := buildFunctionDoc(f.Name, f.Description)
		snippetText := buildFunctionSnippet(f.Name)
		snippetFormat := protocol.InsertTextFormatSnippet

		items = append(items, protocol.CompletionItem{
			Label:            f.Name,
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

	// NL function aliases from the feature registry
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
			if prefix != "" && !features.MatchesPrefix(firstWord, prefix) {
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
		// No tokens → could be blank or pure prose
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

// buildFunctionSnippet creates an LSP snippet string for a function.
// E.g., accumulate → "accumulate(${1:rate}, ${2:duration})"
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
			features.MatchesPrefix(unit.Canonical, prefix) ||
			features.MatchesPrefix(unit.Symbol, prefix)
		if !matched {
			for _, alias := range unit.Aliases {
				if features.MatchesPrefix(alias, prefix) {
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

// variableCompletionItems returns completion items for variables.
// Known limitation: cursorLine is accepted but not used for position filtering.
// The TUI equivalent (VariableSuggestionSource) correctly filters variables
// defined at or after the cursor. This requires mapping document blocks to
// variable definition lines, which Environment does not currently expose.
func variableCompletionItems(snap *DocumentSnapshot, prefix string, cursorLine int) []protocol.CompletionItem {
	prefix = strings.ToLower(prefix)
	var items []protocol.CompletionItem

	env := snap.Evaluator.GetEnvironment()
	if env == nil {
		return nil
	}

	vars := env.GetAllVariables()
	for name, val := range vars {
		if prefix != "" && !features.MatchesPrefix(name, prefix) {
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

// directiveCompletionItems returns completion items for @scale and @globals.field directives.
func directiveCompletionItems(snap *DocumentSnapshot, prefix string) []protocol.CompletionItem {
	if !strings.HasPrefix(prefix, "@") {
		return nil
	}

	fm := snap.Document.GetFrontmatter()
	if fm == nil {
		return nil
	}

	var items []protocol.CompletionItem
	kind := protocol.CompletionItemKindConstant

	// @scale
	if fm.Scale != nil {
		name := "@scale"
		if features.MatchesPrefix(name, strings.ToLower(prefix)) {
			detail := fmt.Sprintf("Scale factor (%s)", fm.Scale.Factor.String())
			items = append(items, protocol.CompletionItem{
				Label:      name,
				Kind:       &kind,
				Detail:     &detail,
				InsertText: &name,
			})
		}
	}

	// @globals or @globals.field
	if len(fm.Globals) > 0 {
		if strings.HasPrefix("@globals", strings.ToLower(prefix)) && !strings.Contains(prefix, ".") {
			label := "@globals"
			detail := fmt.Sprintf("%d global(s) defined", len(fm.Globals))
			insertText := "@globals."
			items = append(items, protocol.CompletionItem{
				Label:      label,
				Kind:       &kind,
				Detail:     &detail,
				InsertText: &insertText,
			})
		} else if strings.HasPrefix(strings.ToLower(prefix), "@globals.") {
			fieldPrefix := strings.ToLower(prefix[len("@globals."):])
			for name, value := range fm.Globals {
				if features.MatchesPrefix(name, fieldPrefix) {
					fullName := "@globals." + name
					detail := value
					items = append(items, protocol.CompletionItem{
						Label:      fullName,
						Kind:       &kind,
						Detail:     &detail,
						InsertText: &fullName,
					})
				}
			}
		}
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
// Extends to include a leading '@' for directive completion (@scale, @globals.field).
func extractPrefix(lineText string, col int) string {
	runes := []rune(lineText)
	if col > len(runes) {
		col = len(runes)
	}

	isWord := func(ch rune) bool {
		return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_'
	}

	// Walk backward from cursor to find start of identifier
	start := col
	for start > 0 && isWord(runes[start-1]) {
		start--
	}

	// Extend to include '@' and dot-separated @globals.field patterns.
	// Only include '@' when preceded by non-word char (not email@example).
	if start > 0 && runes[start-1] == '.' {
		// @globals.field or @globals. (no field yet)
		dotPos := start - 1
		wordStart := dotPos
		for wordStart > 0 && isWord(runes[wordStart-1]) {
			wordStart--
		}
		if wordStart > 0 && runes[wordStart-1] == '@' && (wordStart <= 1 || !isWord(runes[wordStart-2])) {
			start = wordStart - 1
		}
	} else if start > 0 && runes[start-1] == '@' && (start <= 1 || !isWord(runes[start-2])) {
		start--
	} else if start >= col && col > 0 && runes[col-1] == '.' {
		// Cursor right after dot: @globals.
		dotPos := col - 1
		wordStart := dotPos
		for wordStart > 0 && isWord(runes[wordStart-1]) {
			wordStart--
		}
		if wordStart > 0 && runes[wordStart-1] == '@' && (wordStart <= 1 || !isWord(runes[wordStart-2])) {
			start = wordStart - 1
		}
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
