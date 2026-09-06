package lsp

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/CalcMark/go-calcmark/v2/spec/document"
	"github.com/CalcMark/go-calcmark/v2/spec/features"
	"github.com/CalcMark/go-calcmark/v2/spec/lexer"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
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

	// Frontmatter completion takes priority. When the cursor sits inside the
	// frontmatter region we never fall through to calc-block completion:
	// either we return registered-key / enum-value items, or we return an
	// empty list (non-enum value positions, unregistered keys). Falling
	// through would otherwise surface bogus function/variable completions
	// against YAML text.
	if region, ok := DetectRegion(source); ok {
		ctx := ClassifyCursor(region, params.Position)
		if ctx.InRegion {
			if items := buildFrontmatterCompletions(source, params.Position); items != nil {
				return items, nil
			}
			return []protocol.CompletionItem{}, nil
		}
	}

	// Get the current line text for context-sensitive filtering
	line := int(params.Position.Line)
	col := int(params.Position.Character)
	lineText := getLineText(source, line)

	// Determine prefix (text before cursor on this line, back to last non-identifier char)
	prefix := features.ExtractPrefix(lineText, col)

	// Build the doc-scoped known-names set so the classifier can admit
	// lines whose leading IDENT is a known variable (`revenue * 0.1`
	// after `revenue = 1000` earlier). Built-in constants and frontmatter
	// globals don't need to be in this set — they don't change the
	// markdown-vs-calc classification, only what completions resolve to.
	knownNames := snapshotKnownNames(snap)

	// Tokenize the line up to the cursor to determine context.
	// This replaces string heuristics with the real lexer.
	ctx := classifyCompletionContext(lineText, col, knownNames)

	// Detect enclosing function call and active parameter position — used for
	// enum completions and type-filtered variable completions below.
	argCtx := extractArgumentContext(lineText, col)

	switch ctx {
	case completionContextMarkdown:
		return nil, nil

	case completionContextInterpolation:
		// Bug B (2026-05-06): cursor inside `{{ ... }}` in prose. Only
		// variable references resolve here — surface those, nothing else.
		// No required type filter (interpolation accepts any value).
		if snap.Evaluator == nil {
			return []protocol.CompletionItem{}, nil
		}
		return variableCompletionItems(snap, prefix, line, ""), nil

	case completionContextAfterUnitKeyword:
		// After "in" or "as" -> units + conversion keywords (napkin, precise)
		var items []protocol.CompletionItem
		items = append(items, unitCompletionItems(prefix)...)
		items = append(items, conversionKeywordItems(prefix)...)
		return items, nil

	default:
		// `table.` — only that table's columns make sense here
		// (go-calcmark#118, R16). Exclusive, like enum parameters.
		if snap.Evaluator != nil {
			if cols, ok := tableColumnCompletionItems(snap, lineText, col, prefix); ok {
				return cols, nil
			}
		}

		// Enum-backed string parameter: exclusive context — return only the
		// valid enum values. Mixing in functions/variables here would clutter
		// the dropdown with clearly invalid choices.
		if enumItems := enumCompletionsForContext(argCtx, prefix); len(enumItems) > 0 {
			return enumItems, nil
		}

		// Determine active parameter type for variable filtering, if any.
		// A keyword-operator operand (e.g. the percentage operand of
		// `X% of Y`) carries its type directly and takes precedence; only
		// when it is absent do we look up an enclosing function's ParamSpec.
		var requiredType types.ArgType
		switch {
		case argCtx.requiredType != "":
			requiredType = argCtx.requiredType
		case argCtx.funcName != "" && !argCtx.insideString:
			if spec := types.GetFunctionSpec(argCtx.funcName); spec != nil {
				if p := spec.GetParamAtIndex(argCtx.paramIdx); p != nil {
					requiredType = p.Type
				}
			}
		}

		// General context -> functions, keywords, units, variables, directives, dates
		var items []protocol.CompletionItem
		items = append(items, functionCompletionItems(prefix)...)
		items = append(items, keywordCompletionItems(prefix)...)
		items = append(items, unitCompletionItems(prefix)...)
		items = append(items, dateCompletionItems(prefix)...)
		if snap.Evaluator != nil {
			items = append(items, variableCompletionItems(snap, prefix, line, requiredType)...)
		}
		if snap.Document != nil {
			items = append(items, directiveCompletionItems(snap, prefix)...)
		}
		return items, nil
	}
}

// tableColumnCompletionItems returns the columns of the named table
// whose name immediately precedes the `.` before the current prefix,
// filtered by prefix. ok is false when the cursor is not in a
// `<table>.<partial>` position, so the caller falls through to the
// general completions.
func tableColumnCompletionItems(snap *DocumentSnapshot, lineText string, col int, prefix string) ([]protocol.CompletionItem, bool) {
	tableName, ok := tableNameBeforeDot(lineText, col, prefix)
	if !ok {
		return nil, false
	}
	env := snap.Evaluator.GetEnvironment()
	if env == nil {
		return nil, false
	}
	value, found := env.Get(tableName)
	if !found {
		return nil, false
	}
	table, isTable := value.(*types.Table)
	if !isTable {
		return nil, false
	}
	items := []protocol.CompletionItem{}
	kind := protocol.CompletionItemKindField
	lowerPrefix := strings.ToLower(prefix)
	for _, name := range table.ColumnOrder {
		if !strings.HasPrefix(strings.ToLower(name), lowerPrefix) {
			continue
		}
		detail := "column of " + table.Name
		if arr, ok := table.Column(name); ok && arr.ElementType != "" {
			detail = fmt.Sprintf("%s column of %s (%d rows)", strings.ToLower(arr.ElementType), table.Name, arr.Len())
		}
		insert := name
		items = append(items, protocol.CompletionItem{
			Label:      name,
			Kind:       &kind,
			Detail:     &detail,
			InsertText: &insert,
			Data:       completionItemData{Kind: "table_column"},
		})
	}
	return items, true
}

// tableNameBeforeDot returns the identifier that sits immediately before
// the `.` that precedes `prefix` at the cursor: for `cost = rates.ra|`
// it returns "rates". Decimal numbers never match because the run before
// the dot must start with a letter or underscore.
func tableNameBeforeDot(lineText string, col int, prefix string) (string, bool) {
	runes := []rune(lineText)
	if col > len(runes) {
		col = len(runes)
	}
	dot := col - len([]rune(prefix)) - 1
	if dot < 0 || runes[dot] != '.' {
		return "", false
	}
	end := dot
	start := end
	for start > 0 && isIdentRune(runes[start-1]) {
		start--
	}
	if start == end || unicode.IsDigit(runes[start]) {
		return "", false
	}
	return string(runes[start:end]), true
}

// functionCompletionItems returns completion items for built-in functions.
// Delegates to features.FunctionSuggestions then enriches with LSP-specific
// snippets, parameter docs, markdown documentation, and structured
// CompletionItem.data so clients can read function metadata without parsing
// labels (see R3 in the plan).
func functionCompletionItems(prefix string) []protocol.CompletionItem {
	suggestions := features.FunctionSuggestions(prefix, nil) // nil = all registry functions
	var items []protocol.CompletionItem

	for _, s := range suggestions {
		// Use the canonical function name for spec lookup. FunctionName is set
		// by FunctionSuggestions to f.Name for both paren-form and NL rows.
		canonical := s.FunctionName
		if canonical == "" {
			canonical = s.Name
		}
		spec := types.GetFunctionSpec(canonical)
		data := completionItemData{
			Kind:         "function",
			FunctionName: canonical,
		}
		if spec != nil {
			data.Params = paramSpecsToData(spec.Params)
		}

		if s.Category == "example" {
			// NL example row -> snippet item. The example's numeric
			// tokens (digits, optional `$`/`€` prefix, optional `%`
			// suffix) are wrapped in `${N:token}` placeholders by
			// `buildNLExampleSnippet`. The LSP — not the client — owns
			// the boundary detection so every consumer (Web, TUI, any
			// future client) gets the same placeholder shape.
			//
			// Carry the same markdown documentation that the paren-form
			// item gets so the completion-detail panel has something to
			// show when the user highlights an NL row in the dropdown.
			// Without this the right-panel detail surface goes empty
			// for any NL alias, which reads as "the completion is
			// broken."
			kind := protocol.CompletionItemKindSnippet
			detail := s.Syntax
			insertText := buildNLExampleSnippet(s.InsertText)
			snippetFormat := protocol.InsertTextFormatSnippet
			doc := buildFunctionDoc(canonical, s.Description)
			items = append(items, protocol.CompletionItem{
				Label:            s.Name,
				Kind:             &kind,
				Detail:           &detail,
				InsertText:       &insertText,
				InsertTextFormat: &snippetFormat,
				Documentation: &protocol.MarkupContent{
					Kind:  protocol.MarkupKindMarkdown,
					Value: doc,
				},
				Data: data,
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
				Data: data,
			})
		}
	}

	return items
}

// keywordCompletionItems returns snippet completion items for the calc
// keyword-operator forms (`X% of Y`, `X as a % of Y`, `X in unit`). Delegates
// to features.KeywordSuggestions for the canonical templates, then attaches
// the LSP-specific snippet format and structured data so clients (slash
// palette, inline completion) get the same delivery vehicle functions use.
//
// The snippet `InsertText` from the registry already carries `${N:default}`
// placeholders with units/`%` correctly absorbed (see the registry keyword
// block), so — unlike the NL-function path — it is shipped verbatim without
// re-running buildNLExampleSnippet.
func keywordCompletionItems(prefix string) []protocol.CompletionItem {
	suggestions := features.KeywordSuggestions(prefix)
	var items []protocol.CompletionItem

	for _, s := range suggestions {
		kind := protocol.CompletionItemKindKeyword
		detail := s.Syntax
		insertText := s.InsertText
		snippetFormat := protocol.InsertTextFormatSnippet
		items = append(items, protocol.CompletionItem{
			Label:            s.Name,
			Kind:             &kind,
			Detail:           &detail,
			InsertText:       &insertText,
			InsertTextFormat: &snippetFormat,
			Documentation: &protocol.MarkupContent{
				Kind:  protocol.MarkupKindMarkdown,
				Value: s.Description,
			},
			Data: completionItemData{
				Kind:    "keyword",
				Keyword: s.Name,
			},
		})
	}

	return items
}

// completionContext describes the syntactic context at the cursor for completion.
type completionContext int

const (
	completionContextGeneral          completionContext = iota
	completionContextAfterUnitKeyword                   // cursor is after "in" or "as"
	completionContextMarkdown                           // line is markdown, suppress completions
	// Cursor sits inside an open `{{ ... }}` interpolation in prose.
	// The narrow contract: only variable references make sense here.
	// No functions, no units, no dates. The result-pill / chip rendering
	// machinery for embedded interpolation reads the same shape, so
	// users can type `{{ pri` in prose and get `price`-style variable
	// completions without surfacing the rest of the calc-language
	// vocabulary alongside.
	completionContextInterpolation
)

// detectorForCompletion is built once per process. The Detector is stateless
// after construction (it just memoises lexer-trigger / IDENT-callable name
// sets from the registry), so reusing one is safe across goroutines.
var detectorForCompletion = document.NewDetector()

// classifyCompletionContext tokenizes the line up to the cursor position and
// determines the completion context from the token stream. This uses the real
// lexer instead of string heuristics.
//
// `knownNames` is the doc-scoped set of identifiers assigned earlier in the
// document. It lets us classify a leading-IDENT line like `revenue * 0.1`
// as a calculation when `revenue` is a known variable, even though the
// IDENT alone wouldn't admit it. Pass nil to apply the rule with no doc
// context (treats every unknown leading IDENT as prose).
func classifyCompletionContext(lineText string, col int, knownNames map[string]bool) completionContext {
	// Truncate line to cursor position for tokenization
	runes := []rune(lineText)
	if col > len(runes) {
		col = len(runes)
	}
	textBeforeCursor := string(runes[:col])

	// Bug B (2026-05-06): cursor inside `{{ ... }}` opens a narrow
	// completion context. The prose around the interpolation is markdown;
	// only the slot between `{{` and `}}` accepts a variable reference.
	// Detect first because the rest of the classifier walks lexer tokens
	// that don't recognise `{{` / `}}` as anything special.
	if insideInterpolation(textBeforeCursor) {
		return completionContextInterpolation
	}

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
		// No tokens — could mean genuinely empty input, or that the
		// lexer rejected the line entirely (notably `@globals.` and
		// other in-flight directive forms it validates strictly).
		// Defer to the detector's calc-vs-text check; it has its own
		// fast paths for the cases the lexer rejects (`@<letter>`,
		// lines containing a plain assignment `=`, …) and otherwise
		// classifies as prose.
		//
		// Use the completion-lenient variant so a single bare IDENT
		// that matches a registered function prefix (`com` for
		// `compound`) admits as calc-shape. The strict variant
		// classifies bare IDENTs as prose because that shape is
		// also a common prose word — but in the LSP context the
		// user is mid-typing a function call and expects autocomplete.
		if detectorForCompletion.LooksLikeCalculationForCompletion(lineText, knownNames) {
			return completionContextGeneral
		}
		return completionContextMarkdown
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
		}
	}

	// Check if the context token is AS or IN
	switch last.Type {
	case lexer.AS, lexer.IN:
		return completionContextAfterUnitKeyword
	}

	// Bug B (2026-05-06): the line passed every "is this calc-shape?"
	// heuristic above (no markdown prefix, has tokens, not after AS/IN),
	// but it might still be plain prose like `some te` or `this is a
	// sentence`. Run the detector's calc-vs-text token-shape classifier —
	// same logic that decides whether a line lands in a calc block or a
	// text block during normal block detection. If the detector says
	// it's not calc, suppress completions.
	//
	// `knownNames` lets us admit lines whose leading IDENT is a known
	// document variable (`revenue * 0.1` when `revenue` was assigned
	// earlier). Without it, every IDENT-leading line that isn't a
	// registered NL-trigger / function would classify as prose.
	//
	// We use `LooksLikeCalculationForCompletion` (token-shape check
	// with the function-prefix admit) rather than the strict
	// `LooksLikeCalculation` because the user is in-flight typing
	// and the partial line — `accumulate(`, `compound 1000`,
	// `revenue * `, or just bare `com` — fails the strict parser
	// even though it's unambiguously calc-shape from the user's
	// perspective. The strict variant rejects bare-IDENT lines as
	// prose; the completion-lenient variant admits them when they
	// match a registered function prefix, preserving block-detection
	// safety while giving the user autocomplete on the first
	// keystroke past the prefix length threshold.
	if !detectorForCompletion.LooksLikeCalculationForCompletion(lineText, knownNames) {
		return completionContextMarkdown
	}

	return completionContextGeneral
}

// insideInterpolation returns true when `textBeforeCursor` ends inside an
// open `{{ ... }}` group — i.e. there's a `{{` somewhere in the string and
// no matching `}}` after it. Pure string scan, no tokenisation.
//
// Used to detect the variable-only completion context for prose like
// `Hello {{ pri|`. Calc-block usage doesn't reach this helper (the
// detector / parser handle calc lines first) so we don't worry about
// `{{` appearing inside calc-source.
func insideInterpolation(textBeforeCursor string) bool {
	lastOpen := strings.LastIndex(textBeforeCursor, "{{")
	if lastOpen == -1 {
		return false
	}
	// If there's a `}}` between the last `{{` and the cursor, the
	// interpolation already closed before we got here — back to prose.
	if strings.Contains(textBeforeCursor[lastOpen+2:], "}}") {
		return false
	}
	return true
}

// snapshotKnownNames returns the set of variable names defined anywhere in
// the document, suitable for the calc-vs-text classifier in
// `classifyCompletionContext`. Pure: builds a fresh map per call so the
// caller can mutate it without affecting the snapshot.
//
// Built-in constants and frontmatter globals are intentionally included —
// they're valid leading-IDENT references, so a line like `pi * radius`
// should classify as calc even when the user hasn't assigned `pi` themselves.
func snapshotKnownNames(snap *DocumentSnapshot) map[string]bool {
	if snap == nil || snap.Evaluator == nil {
		return nil
	}
	env := snap.Evaluator.GetEnvironment()
	if env == nil {
		return nil
	}
	all := env.GetAllVariables()
	if len(all) == 0 {
		return nil
	}
	out := make(map[string]bool, len(all))
	for name := range all {
		out[name] = true
	}
	return out
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
//
// When requiredType is non-empty, variables whose evaluator-inferred type is
// not compatible with the required parameter type are filtered out. An empty
// requiredType means no filter (bare expression context).
//
// Variables defined AT or AFTER `cursorLine` (doc-absolute, 0-indexed)
// are excluded — calcmark is strictly ordered, so a reference cannot
// point at a definition that appears later in the document. The
// Environment exposes per-variable definition lines via
// `GetAllDefinedLines`; built-in constants and frontmatter globals
// (which apply before any block) are intentionally absent from that
// map and therefore unfiltered.
func variableCompletionItems(snap *DocumentSnapshot, prefix string, cursorLine int, requiredType types.ArgType) []protocol.CompletionItem {
	env := snap.Evaluator.GetEnvironment()
	if env == nil {
		return nil
	}

	allVars := env.GetAllVariables()

	// Build the filtered vars map plus a parallel map of runtime types for
	// attaching to completion item data.
	vars := make(map[string]string, len(allVars))
	argTypes := make(map[string]types.ArgType, len(allVars))
	for name, val := range allVars {
		varType := runtimeTypeToArgType(val)
		if !argTypesCompatible(varType, requiredType) {
			continue
		}
		vars[name] = fmt.Sprintf("%v", val)
		argTypes[name] = varType
	}

	// Position filtering: variables defined at or after `cursorLine`
	// are excluded. Variables without a recorded line (constants,
	// frontmatter globals) pass through unfiltered.
	definedLines := env.GetAllDefinedLines()
	suggestions := features.VariableSuggestions(vars, prefix, cursorLine, definedLines)
	var items []protocol.CompletionItem

	for _, s := range suggestions {
		kind := protocol.CompletionItemKindVariable
		detail := s.Description

		items = append(items, protocol.CompletionItem{
			Label:      s.Name,
			Kind:       &kind,
			Detail:     &detail,
			InsertText: &s.InsertText,
			Data: completionItemData{
				Kind:         "variable",
				VariableType: argTypes[s.Name],
			},
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
