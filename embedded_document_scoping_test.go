package calcmark

import (
	"strings"
	"testing"

	impldoc "github.com/CalcMark/go-calcmark/v2/impl/document"
	specDoc "github.com/CalcMark/go-calcmark/v2/spec/document"
)

// U3: whole-doc evaluator scoping verification.
//
// The structural API (NewDocumentEmbedded) puts every CalcBlock
// from every fence into the SAME *specDoc.Document, so the
// existing impldoc.Evaluator naturally resolves variables across
// fences when it walks the document. This intentionally diverges
// from Convert(Mode: Embedded)'s rendering pipeline, which
// evaluates each fence in a fresh Document.
//
// These tests pin the contract — a future refactor that splits
// the Document into per-fence pieces would break them
// immediately. The plan calls this out as load-bearing because
// downstream consumers (calcmark-web's editor today resolves
// {{ price }} in prose adjacent to a fence defining `price`)
// depend on it.

func TestEmbedded_VarDefinedInFenceA_ResolvesInFenceB(t *testing.T) {
	src := "```cm\nprice = 100\n```\n\n```cm\ntax = price * 0.1\n```\n"
	doc, err := NewDocumentEmbedded(src)
	if err != nil {
		t.Fatalf("NewDocumentEmbedded: %v", err)
	}

	if err := impldoc.NewEvaluator().Evaluate(doc); err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	got := evalResultsByVar(t, doc)
	if got["price"] == "" {
		t.Errorf("price was not evaluated; got map: %v", got)
	}
	if got["tax"] == "" {
		t.Errorf("tax was not evaluated (cross-fence reference to price failed); got map: %v", got)
	}
	if got["tax"] != "10" {
		t.Errorf("tax: expected 10 (cross-fence resolution of price=100), got %q", got["tax"])
	}
}

func TestEmbedded_ThreeFencesCrossReferencing_AllResolve(t *testing.T) {
	src := "" +
		"```cm\nprice = 100\n```\n\n" +
		"prose between\n\n" +
		"```cm\ntax = price * 0.1\n```\n\n" +
		"more prose\n\n" +
		"```cm\ntotal = price + tax\n```\n"

	doc, err := NewDocumentEmbedded(src)
	if err != nil {
		t.Fatalf("NewDocumentEmbedded: %v", err)
	}
	if err := impldoc.NewEvaluator().Evaluate(doc); err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	got := evalResultsByVar(t, doc)
	if got["price"] != "100" {
		t.Errorf("price: expected 100, got %q", got["price"])
	}
	if got["tax"] != "10" {
		t.Errorf("tax: expected 10 (= price * 0.1), got %q", got["tax"])
	}
	if got["total"] != "110" {
		t.Errorf("total: expected 110 (= price + tax), got %q", got["total"])
	}
}

func TestEmbedded_FenceB_ReferencesUndefinedVar_DiagnosticOnFenceB_FenceAUnaffected(t *testing.T) {
	// Fence A defines price; fence B references undefinedVar. Fence
	// A's eval should be unaffected; fence B should carry an error
	// diagnostic. Pins per-block error isolation alongside whole-doc
	// var visibility.
	src := "```cm\nprice = 100\n```\n\n```cm\ntax = undefinedVar * 0.1\n```\n"
	doc, err := NewDocumentEmbedded(src)
	if err != nil {
		t.Fatalf("NewDocumentEmbedded: %v", err)
	}
	// Evaluate may return a partial-evaluation error; that's fine —
	// what matters is the per-block result/diagnostic state.
	_ = impldoc.NewEvaluator().Evaluate(doc)

	calcBlocks := calcBlocksOf(doc)
	if len(calcBlocks) != 2 {
		t.Fatalf("expected 2 calc blocks, got %d", len(calcBlocks))
	}
	// Fence A: price = 100 evaluated cleanly.
	if got := firstResultString(calcBlocks[0]); got != "100" {
		t.Errorf("fence A (price): expected 100, got %q", got)
	}
	// Fence B: failed to resolve undefinedVar. Either the result is
	// nil/error or the block carries a diagnostic.
	failedB := firstResultString(calcBlocks[1]) == "" || len(calcBlocks[1].Diagnostics()) > 0
	if !failedB {
		t.Errorf("fence B (undefinedVar): expected nil result or diagnostic, got result=%q diagnostics=%v",
			firstResultString(calcBlocks[1]), calcBlocks[1].Diagnostics())
	}
}

// Note: forward-reference parity (defining a var LATER than its
// first use) is outside U3's scope — that's a language-spec
// question that lives downstream of this plan. The three tests
// above pin the in-document-order use case that U3 requires.

// --- helpers ---

func calcBlocksOf(doc *specDoc.Document) []*specDoc.CalcBlock {
	var out []*specDoc.CalcBlock
	for _, node := range doc.GetBlocks() {
		if cb, ok := node.Block.(*specDoc.CalcBlock); ok {
			out = append(out, cb)
		}
	}
	return out
}

// evalResultsByVar maps each variable defined in any calc block to
// its formatted result. Empty string means the var didn't evaluate
// (nil result or missing).
func evalResultsByVar(t *testing.T, doc *specDoc.Document) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, node := range doc.GetBlocks() {
		cb, ok := node.Block.(*specDoc.CalcBlock)
		if !ok {
			continue
		}
		results := cb.Results()
		// Walk the block's source lines; for each line that's an
		// assignment, pair its index with the matching result.
		for i, line := range cb.Source() {
			name := assignmentTarget(line)
			if name == "" {
				continue
			}
			if i >= len(results) || results[i] == nil {
				out[name] = ""
				continue
			}
			out[name] = results[i].String()
		}
	}
	return out
}

// assignmentTarget extracts the LHS variable name from a calc line
// like "x = 1 + 2". Returns "" for non-assignment lines.
func assignmentTarget(line string) string {
	idx := strings.Index(line, "=")
	if idx <= 0 {
		return ""
	}
	name := strings.TrimSpace(line[:idx])
	if name == "" {
		return ""
	}
	// Reject "==" and similar (the next char after = should not be =).
	if idx+1 < len(line) && line[idx+1] == '=' {
		return ""
	}
	return name
}

func firstResultString(cb *specDoc.CalcBlock) string {
	results := cb.Results()
	if len(results) == 0 || results[0] == nil {
		return ""
	}
	return results[0].String()
}
