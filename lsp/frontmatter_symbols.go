package lsp

import (
	"sort"

	"github.com/CalcMark/go-calcmark/spec/ast"
	specDoc "github.com/CalcMark/go-calcmark/spec/document"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

// buildFrontmatterSymbols returns DocumentSymbol entries for every registered
// CalcMark frontmatter key physically present in the source (as captured in
// fm.KeyRanges). Emission order is source order — ascending Range.Start.Line —
// so the outline reflects the document top-to-bottom.
//
// Extra (unregistered) keys produce no symbols per the Extra-passthrough
// principle: they carry no CalcMark semantics and belong to generic Markdown
// front matter, not our language surface.
//
// Complexity: O(|registered keys in source|) ≈ ≤6 (per D10). The sort is over
// that same small slice.
func buildFrontmatterSymbols(fm specDoc.Frontmatter) []protocol.DocumentSymbol {
	if len(fm.KeyRanges) == 0 {
		return nil
	}

	type entry struct {
		name string
		r    ast.Range
	}
	registered := make([]entry, 0, len(fm.KeyRanges))
	for name, r := range fm.KeyRanges {
		if !specDoc.IsRegisteredKey(name) {
			continue
		}
		registered = append(registered, entry{name: name, r: r})
	}
	if len(registered) == 0 {
		return nil
	}
	sort.Slice(registered, func(i, j int) bool {
		return registered[i].r.Start.Line < registered[j].r.Start.Line
	})

	out := make([]protocol.DocumentSymbol, 0, len(registered))
	for _, e := range registered {
		r := e.r
		full := ToLSPRange(&r)
		// SelectionRange: identifier-only. Keys are YAML top-level identifiers,
		// always on a single line, starting at the key's start column.
		sel := protocol.Range{
			Start: full.Start,
			End: protocol.Position{
				Line:      full.Start.Line,
				Character: full.Start.Character + protocol.UInteger(len(e.name)),
			},
		}
		sym := protocol.DocumentSymbol{
			Name:           e.name,
			Kind:           protocol.SymbolKindProperty,
			Range:          full,
			SelectionRange: sel,
		}
		if key, ok := specDoc.LookupKey(e.name); ok {
			label := frontmatterTypeLabel(key)
			sym.Detail = &label
		}
		out = append(out, sym)
	}
	return out
}
