package lsp

import (
	_ "embed"
	"fmt"
	"strings"
	"text/template"
	"unicode"

	specDoc "github.com/CalcMark/go-calcmark/v2/spec/document"
	"github.com/CalcMark/go-calcmark/v2/spec/features"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
	"github.com/CalcMark/go-calcmark/v2/spec/units"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

//go:embed templates/function_hover.md.tmpl
var functionHoverTemplateSrc string

// functionHoverTmpl renders the markdown body for function hover responses.
// Parsed once at package init; execution is safe for concurrent use.
var functionHoverTmpl = template.Must(
	template.New("function_hover").Parse(functionHoverTemplateSrc),
)

// functionHoverData is the view model for functionHoverTmpl.
type functionHoverData struct {
	Name        string
	Syntax      string
	Description string
	Params      []functionHoverParam
	Example     string
	NLExample   string
}

// functionHoverParam is one row in the Parameters section of a function hover.
type functionHoverParam struct {
	Name         string
	Type         types.ArgType
	Optional     bool
	Variadic     bool
	FirstExample string
}

// lspHover is a superset of protocol.Hover that adds a structured Data field
// so clients that want structured hover content (e.g. calcmark-web's gutter
// panel) don't have to parse markdown out of Contents. Standard LSP clients
// (VS Code, Neovim, Zed, …) ignore the unknown `data` field and render
// Contents normally — no capability negotiation required.
type lspHover struct {
	Contents any             `json:"contents"`
	Range    *protocol.Range `json:"range,omitempty"`
	Data     any             `json:"data,omitempty"`
}

// hoverData is the structured payload attached to lspHover.Data. The Kind
// field discriminates the shape of the remaining fields:
//
//   - "variable" → Name, VariableType, Value
//   - "function" → Name, Syntax, Description, Params, Example, NLExample
//   - "unit"     → Name, Symbol, Description, Category, Aliases
//
// Name is always set and should be used uniformly across kinds. Clients use
// Kind to decide which additional fields are relevant.
type hoverData struct {
	Kind string `json:"kind"`

	// Common to every kind.
	Name string `json:"name,omitempty"`

	// Variable kind.
	VariableType types.ArgType `json:"variableType,omitempty"`
	Value        string        `json:"value,omitempty"`

	// Function kind.
	Syntax      string          `json:"syntax,omitempty"`
	Description string          `json:"description,omitempty"`
	Params      []wireParamData `json:"params,omitempty"`
	Example     string          `json:"example,omitempty"`
	NLExample   string          `json:"nlExample,omitempty"`

	// Unit kind.
	Symbol   string   `json:"symbol,omitempty"`
	Category string   `json:"category,omitempty"`
	Aliases  []string `json:"aliases,omitempty"`
}

// textDocumentHover is a no-op satisfying the glsp handler struct's field
// type. Real dispatch happens via interceptingHandler → hoverHandle so the
// extended lspHover wire shape reaches JSON marshaling intact.
func (s *Server) textDocumentHover(_ *glsp.Context, _ *protocol.HoverParams) (*protocol.Hover, error) {
	return nil, nil
}

// hoverHandle is the actual hover implementation. Returns `any` so the
// extended lspHover shape can carry a structured `data` field.
func (s *Server) hoverHandle(params *protocol.HoverParams) (any, error) {
	ds := s.getDocument(params.TextDocument.URI)
	if ds == nil {
		return nil, nil
	}

	snap := ds.getSnapshot()
	if snap == nil {
		return nil, nil
	}

	source := ds.getSource()

	// Frontmatter hover takes priority. When the cursor sits inside the
	// frontmatter region we never fall through to calc-block hover: either
	// we return a registered-key hover, or we return null (Extra keys,
	// fences, blanks). This prevents the word-under-cursor code below from
	// producing bogus variable/function matches against YAML text.
	if region, ok := DetectRegion(source); ok {
		ctx := ClassifyCursor(region, params.Position)
		if ctx.InRegion {
			h := buildFrontmatterHover(source, params.Position)
			if h == nil {
				return nil, nil
			}
			return h, nil
		}
	}

	line := int(params.Position.Line)
	col := int(params.Position.Character)
	lineText := getLineText(source, line)

	// Extract the word under the cursor
	word := extractWordAt(lineText, col)
	if word == "" {
		return nil, nil
	}

	// Try variable hover
	if snap.Evaluator != nil {
		env := snap.Evaluator.GetEnvironment()
		if env != nil {
			vars := env.GetAllVariables()
			if val, ok := vars[word]; ok {
				argType := runtimeTypeToArgType(val)
				value := fmt.Sprintf("%v", val)
				content := fmt.Sprintf("**%s**: `%s` = `%s`", word, argType, value)
				return &lspHover{
					Contents: protocol.MarkupContent{
						Kind:  protocol.MarkupKindMarkdown,
						Value: content,
					},
					Data: hoverData{
						Kind:         "variable",
						Name:         word,
						VariableType: argType,
						Value:        value,
					},
				}, nil
			}
		}
	}

	// Try function hover
	registry := features.DefaultRegistry()
	for _, f := range registry.ByCategory(features.CategoryFunction) {
		if strings.EqualFold(f.Name, word) {
			return &lspHover{
				Contents: protocol.MarkupContent{
					Kind:  protocol.MarkupKindMarkdown,
					Value: buildFunctionHoverContent(f.Name, f.Syntax, f.Description, f.NLExample, f.Example),
				},
				Data: buildFunctionHoverData(f.Name, f.Syntax, f.Description, f.Example, f.NLExample),
			}, nil
		}
		for _, syn := range f.Synonyms {
			if strings.EqualFold(syn, word) {
				body := buildFunctionHoverContent(f.Name, f.Syntax, f.Description, f.NLExample, f.Example)
				content := fmt.Sprintf("**%s** (synonym for **%s**)\n\n%s", syn, f.Name, body)
				return &lspHover{
					Contents: protocol.MarkupContent{
						Kind:  protocol.MarkupKindMarkdown,
						Value: content,
					},
					Data: buildFunctionHoverData(f.Name, f.Syntax, f.Description, f.Example, f.NLExample),
				}, nil
			}
		}
	}

	// Try unit hover
	for _, unit := range units.StandardUnits {
		if strings.EqualFold(unit.Canonical, word) || strings.EqualFold(unit.Symbol, word) {
			content := fmt.Sprintf("**%s** (`%s`)\n\n%s\n\nCategory: %s", unit.Canonical, unit.Symbol, unit.Description, unit.Quantity)
			if len(unit.Aliases) > 0 {
				content += fmt.Sprintf("\n\nAliases: %s", strings.Join(unit.Aliases, ", "))
			}
			return &lspHover{
				Contents: protocol.MarkupContent{
					Kind:  protocol.MarkupKindMarkdown,
					Value: content,
				},
				Data: hoverData{
					Kind:        "unit",
					Name:        unit.Canonical,
					Symbol:      unit.Symbol,
					Description: unit.Description,
					Category:    unit.Quantity,
					Aliases:     unit.Aliases,
				},
			}, nil
		}
		for _, alias := range unit.Aliases {
			if strings.EqualFold(alias, word) {
				content := fmt.Sprintf("**%s** (`%s`) — alias for **%s**\n\n%s", alias, unit.Symbol, unit.Canonical, unit.Description)
				return &lspHover{
					Contents: protocol.MarkupContent{
						Kind:  protocol.MarkupKindMarkdown,
						Value: content,
					},
					Data: hoverData{
						Kind:        "unit",
						Name:        unit.Canonical,
						Symbol:      unit.Symbol,
						Description: unit.Description,
						Category:    unit.Quantity,
						Aliases:     unit.Aliases,
					},
				}, nil
			}
		}
	}

	return nil, nil
}

// buildFunctionHoverData constructs the structured hoverData for a function,
// pulling param metadata from the spec.
func buildFunctionHoverData(name, syntax, description, example, nlExample string) hoverData {
	d := hoverData{
		Kind:        "function",
		Name:        name,
		Syntax:      syntax,
		Description: description,
		Example:     example,
		NLExample:   nlExample,
	}
	if spec := types.GetFunctionSpec(name); spec != nil {
		d.Params = paramSpecsToData(spec.Params)
	}
	return d
}

// buildFunctionHoverContent renders the markdown hover body for a function
// through the embedded text/template. Gathers parameter data from the function
// spec so clients see names, types, and first-examples inline.
func buildFunctionHoverContent(name, syntax, description, nlExample, example string) string {
	data := functionHoverData{
		Name:        name,
		Syntax:      syntax,
		Description: description,
		Example:     example,
		NLExample:   nlExample,
	}
	if spec := types.GetFunctionSpec(name); spec != nil {
		data.Params = make([]functionHoverParam, len(spec.Params))
		for i, p := range spec.Params {
			fp := functionHoverParam{
				Name:     p.Name,
				Type:     p.Type,
				Optional: p.Optional,
				Variadic: p.Variadic,
			}
			if len(p.Examples) > 0 {
				fp.FirstExample = p.Examples[0]
			}
			data.Params[i] = fp
		}
	}

	var b strings.Builder
	if err := functionHoverTmpl.Execute(&b, data); err != nil {
		// Template is compiled in-package — this can only fail on a broken
		// template file, which is caught by tests. Fall back to a minimal
		// string so hover still returns something useful.
		return fmt.Sprintf("**%s**\n\n`%s`\n\n%s", name, syntax, description)
	}
	return b.String()
}

// textDocumentDefinition handles the textDocument/definition request.
func (s *Server) textDocumentDefinition(_ *glsp.Context, params *protocol.DefinitionParams) (any, error) {
	ds := s.getDocument(params.TextDocument.URI)
	if ds == nil {
		return nil, nil
	}

	source := ds.getSource()

	line := int(params.Position.Line)
	col := int(params.Position.Character)
	lineText := getLineText(source, line)
	word := extractWordAt(lineText, col)
	if word == "" {
		return nil, nil
	}

	// Search for the assignment line where this variable is defined
	lines := strings.Split(source, "\n")
	for i, l := range lines {
		trimmed := strings.TrimSpace(l)
		// Check if line is an assignment: varName = ...
		if eqIdx := strings.Index(trimmed, "="); eqIdx > 0 {
			varName := strings.TrimSpace(trimmed[:eqIdx])
			if varName == word {
				return protocol.Location{
					URI: params.TextDocument.URI,
					Range: protocol.Range{
						Start: protocol.Position{Line: protocol.UInteger(i), Character: 0},
						End:   protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(eqIdx)},
					},
				}, nil
			}
		}
	}

	return nil, nil
}

// textDocumentDocumentSymbol handles the textDocument/documentSymbol request.
func (s *Server) textDocumentDocumentSymbol(_ *glsp.Context, params *protocol.DocumentSymbolParams) (any, error) {
	ds := s.getDocument(params.TextDocument.URI)
	if ds == nil {
		return nil, nil
	}

	snap := ds.getSnapshot()
	source := ds.getSource()

	var symbols []protocol.DocumentSymbol

	// Prepend frontmatter symbols for registered keys. A malformed YAML
	// frontmatter must not kill the outline — if parsing fails we simply
	// skip the frontmatter section and still emit calc-block symbols.
	if fm, _, err := specDoc.ParseFrontmatter(source); err == nil && fm != nil {
		symbols = append(symbols, buildFrontmatterSymbols(*fm)...)
	}

	lines := strings.Split(source, "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Headings → String symbol kind
		if strings.HasPrefix(trimmed, "#") {
			name := strings.TrimLeft(trimmed, "# ")
			if name == "" {
				continue
			}
			kind := protocol.SymbolKindString
			r := protocol.Range{
				Start: protocol.Position{Line: protocol.UInteger(i), Character: 0},
				End:   protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(len(line))},
			}
			symbols = append(symbols, protocol.DocumentSymbol{
				Name:           name,
				Kind:           kind,
				Range:          r,
				SelectionRange: r,
			})
		}

		// Assignments → Variable symbol kind
		if eqIdx := strings.Index(trimmed, "="); eqIdx > 0 {
			// Avoid == (comparison operator)
			if eqIdx+1 < len(trimmed) && trimmed[eqIdx+1] == '=' {
				continue
			}
			if eqIdx > 0 && trimmed[eqIdx-1] == '!' {
				continue
			}
			varName := strings.TrimSpace(trimmed[:eqIdx])
			// Basic validation: variable name should be a simple identifier
			if varName != "" && isIdentifier(varName) {
				kind := protocol.SymbolKindVariable
				r := protocol.Range{
					Start: protocol.Position{Line: protocol.UInteger(i), Character: 0},
					End:   protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(len(line))},
				}
				selRange := protocol.Range{
					Start: protocol.Position{Line: protocol.UInteger(i), Character: 0},
					End:   protocol.Position{Line: protocol.UInteger(i), Character: protocol.UInteger(eqIdx)},
				}

				detail := ""
				if snap.Evaluator != nil {
					env := snap.Evaluator.GetEnvironment()
					if env != nil {
						if val, ok := env.GetAllVariables()[varName]; ok {
							detail = fmt.Sprintf("%v", val)
						}
					}
				}

				sym := protocol.DocumentSymbol{
					Name:           varName,
					Kind:           kind,
					Range:          r,
					SelectionRange: selRange,
				}
				if detail != "" {
					sym.Detail = &detail
				}
				symbols = append(symbols, sym)
			}
		}
	}

	return symbols, nil
}

// extractWordAt extracts the word at the given column position in a line.
// Uses rune-aware indexing for UTF-8 safety (CalcMark supports Unicode identifiers).
func extractWordAt(lineText string, col int) string {
	runes := []rune(lineText)
	if col > len(runes) {
		col = len(runes)
	}

	start := col
	for start > 0 && isIdentRune(runes[start-1]) {
		start--
	}

	end := col
	for end < len(runes) && isIdentRune(runes[end]) {
		end++
	}

	if start == end {
		return ""
	}
	return string(runes[start:end])
}

// isIdentRune returns true if the rune can be part of a CalcMark identifier.
// Matches the lexer's rules: Unicode letters, digits, and underscore.
func isIdentRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// isIdentifier returns true if the string is a valid CalcMark identifier.
// Uses Unicode-aware character classification matching the lexer's rules.
func isIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if !unicode.IsLetter(r) && r != '_' {
				return false
			}
		} else {
			if !isIdentRune(r) {
				return false
			}
		}
	}
	return true
}
