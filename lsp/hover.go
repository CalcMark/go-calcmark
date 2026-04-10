package lsp

import (
	_ "embed"
	"fmt"
	"strings"
	"text/template"
	"unicode"

	"github.com/CalcMark/go-calcmark/spec/features"
	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/CalcMark/go-calcmark/spec/units"
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

// textDocumentHover handles the textDocument/hover request.
func (s *Server) textDocumentHover(_ *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	ds := s.getDocument(params.TextDocument.URI)
	if ds == nil {
		return nil, nil
	}

	snap := ds.getSnapshot()
	if snap == nil {
		return nil, nil
	}

	source := ds.getSource()
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
				content := fmt.Sprintf("**%s**: `%s` = `%v`", word, argType, val)
				return &protocol.Hover{
					Contents: protocol.MarkupContent{
						Kind:  protocol.MarkupKindMarkdown,
						Value: content,
					},
				}, nil
			}
		}
	}

	// Try function hover
	registry := features.DefaultRegistry()
	for _, f := range registry.ByCategory(features.CategoryFunction) {
		if strings.EqualFold(f.Name, word) {
			return &protocol.Hover{
				Contents: protocol.MarkupContent{
					Kind:  protocol.MarkupKindMarkdown,
					Value: buildFunctionHoverContent(f.Name, f.Syntax, f.Description, f.NLExample, f.Example),
				},
			}, nil
		}
		for _, syn := range f.Synonyms {
			if strings.EqualFold(syn, word) {
				body := buildFunctionHoverContent(f.Name, f.Syntax, f.Description, f.NLExample, f.Example)
				content := fmt.Sprintf("**%s** (synonym for **%s**)\n\n%s", syn, f.Name, body)
				return &protocol.Hover{
					Contents: protocol.MarkupContent{
						Kind:  protocol.MarkupKindMarkdown,
						Value: content,
					},
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
			return &protocol.Hover{
				Contents: protocol.MarkupContent{
					Kind:  protocol.MarkupKindMarkdown,
					Value: content,
				},
			}, nil
		}
		for _, alias := range unit.Aliases {
			if strings.EqualFold(alias, word) {
				content := fmt.Sprintf("**%s** (`%s`) — alias for **%s**\n\n%s", alias, unit.Symbol, unit.Canonical, unit.Description)
				return &protocol.Hover{
					Contents: protocol.MarkupContent{
						Kind:  protocol.MarkupKindMarkdown,
						Value: content,
					},
				}, nil
			}
		}
	}

	return nil, nil
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
