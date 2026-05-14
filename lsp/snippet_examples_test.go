package lsp

import (
	"fmt"
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/parser"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
)

// Snippet placeholder examples must be valid CalcMark when inserted
// into the call site. Issue #130 reported that `convert_rate` and
// `throughput` snippets used quoted-string placeholders that parsed
// to invalid syntax (e.g. `convert_rate(10 MB/s, "per second")`).
// The fix was to switch those parameters' Examples to bare time-unit
// / type identifiers, but there was no regression guard — so future
// `Examples` edits could re-introduce the bug silently.
//
// This test reconstructs the same snippet placeholder LSP would emit
// (the first Examples entry per parameter) and feeds the resulting
// call expression to the parser. Anything that fails to parse is a
// regression: the user would be inserting broken code on completion.
//
// Optional parameters (Optional: true) are excluded because the LSP
// snippet builder includes them in the placeholder string, and an
// invalid optional example would still surface; we only skip when
// the example would not produce a parseable expression in its own
// right (e.g. type stubs the LSP never inserts).
func TestFunctionSpecExamples_ParseAsSnippetPlaceholders(t *testing.T) {
	for name, spec := range types.FunctionSpecs {
		t.Run(name, func(t *testing.T) {
			if len(spec.Params) == 0 {
				return
			}
			args := make([]string, 0, len(spec.Params))
			for _, p := range spec.Params {
				if len(p.Examples) == 0 {
					t.Fatalf("function %q parameter %q has no Examples; "+
						"the LSP snippet builder will fall back to the bare name, "+
						"which is rarely valid CalcMark", name, p.Name)
				}
				args = append(args, p.Examples[0])
			}
			expr := fmt.Sprintf("%s(%s)", name, strings.Join(args, ", "))
			if _, err := parser.Parse(expr + "\n"); err != nil {
				t.Errorf("snippet placeholder %q does not parse: %v\n"+
					"  fix: edit spec/types/param_types.go so the first Examples entry "+
					"for each parameter is a valid CalcMark expression in this call context",
					expr, err)
			}
		})
	}
}
