package calcmark

import (
	"github.com/CalcMark/go-calcmark/v2/impl/embedded"
	specDoc "github.com/CalcMark/go-calcmark/v2/spec/document"
)

// NewDocumentEmbedded parses an Embedded-mode CalcMark source —
// standard markdown with `cm` or `calcmark` fenced code blocks —
// and returns a *specDoc.Document whose blocks reflect the fence
// boundaries. Each `cm`/`calcmark` fenced segment becomes a
// *specDoc.CalcBlock (source = fence inner content); each
// passthrough markdown segment becomes a *specDoc.TextBlock; a
// leading `---...---` segment is parsed as the document's
// frontmatter via the existing spec/document.ParseFrontmatter.
//
// The returned document supports the same evaluator semantics as
// NewDocument: variables defined in any CalcBlock resolve in
// subsequent CalcBlocks (whole-doc scoping), `{{ varName }}` tags in
// prose interpolate, and named tables in the markdown are visible to
// the fences below them. Convert(Mode: Embedded) renders through this
// same document since go-calcmark#118 — it used to evaluate each fence
// in isolation.
//
// Static-rendering escape hatch: any fenced code block whose
// info-string is NOT exactly `cm` or `calcmark` (for example
// ```text, ```go, ```output) is a regular markdown code block
// and projects as part of the surrounding TextBlock content,
// NOT as a CalcBlock. This is already in the language spec —
// no new info-string is needed for static rendering. Do not
// invent constructs like `calcmark-source`.
//
// Empty source returns an empty *Document with zero blocks.
// Sources with zero `cm`/`calcmark` fences return a single
// TextBlock containing the whole source — Embedded mode
// gracefully degrades to "all-passthrough markdown."
func NewDocumentEmbedded(source string) (*specDoc.Document, error) {
	return embedded.BuildDocument(source)
}
