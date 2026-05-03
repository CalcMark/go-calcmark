package embedded

import (
	"testing"

	specDoc "github.com/CalcMark/go-calcmark/v2/spec/document"
)

// U5: document-absolute line ranges + diagnostic offsets.
//
// The existing impl/document.blockLineOffset function (used by the
// evaluator to compute diagnostic DocLine values) assumes blocks
// are contiguous — it sums `len(node.Block.Source())` for every
// block preceding the target. For Embedded sources, fence
// delimiter lines (the ```cm and closing ```) aren't part of any
// block's Source(), so contiguous summing under-counts.
//
// The fix: BlockNode carries an explicit HostLineOffset that the
// Embedded builder sets per-segment. blockLineOffset prefers it
// when nonzero; otherwise falls back to today's contiguous-sum
// arithmetic so flat-CM behavior is unchanged.
//
// HostLineOffset semantics: the count of host-doc lines that
// precede the block's first source line. So a diagnostic on the
// block's source line N (1-based) maps to host-doc line
// HostLineOffset + N.

func TestBuildDocument_HostLineOffset_FenceAtLine1_OffsetIsOne(t *testing.T) {
	// Source: ```cm at line 1, content "price = 100" at line 2.
	// HostLineOffset for the calc block = 1 (so source line 1 + 1 = host line 2).
	src := "```cm\nprice = 100\n```\n"
	doc, _ := BuildDocument(src)
	cb, off := firstCalcOffset(t, doc)
	if cb == nil {
		t.Fatal("no calc block found")
	}
	if off != 1 {
		t.Errorf("expected HostLineOffset=1 (fence opens at line 1), got %d", off)
	}
}

func TestBuildDocument_HostLineOffset_FenceAfterProse_OffsetCountsLeadingProse(t *testing.T) {
	// Source layout (1-based):
	//   line 1: # heading
	//   line 2: (blank)
	//   line 3: prose
	//   line 4: (blank)
	//   line 5: ```cm
	//   line 6: price = 100
	//   line 7: ```
	src := "# heading\n\nprose\n\n```cm\nprice = 100\n```\n"
	doc, _ := BuildDocument(src)
	cb, off := firstCalcOffset(t, doc)
	if cb == nil {
		t.Fatal("no calc block found")
	}
	// Open fence at line 5; HostLineOffset = 5 so source line 1
	// (= "price = 100") maps to host-doc line 6.
	if off != 5 {
		t.Errorf("expected HostLineOffset=5 (fence at line 5), got %d", off)
	}
}

func TestBuildDocument_HostLineOffset_PassthroughLeading_StartsAtZero(t *testing.T) {
	// Leading passthrough should have HostLineOffset = 0 (the
	// passthrough's first source line IS host-doc line 1).
	src := "leading prose\n\n```cm\nx = 1\n```\n"
	doc, _ := BuildDocument(src)
	blocks := doc.GetBlocks()
	if len(blocks) < 1 {
		t.Fatal("no blocks")
	}
	if blocks[0].HostLineOffset != 0 {
		t.Errorf("expected leading passthrough HostLineOffset=0, got %d", blocks[0].HostLineOffset)
	}
}

func TestBuildDocument_HostLineOffset_TwoFences_SecondCalcOffsetCorrect(t *testing.T) {
	// First fence at line 1-3 (open, content, close), trailing
	// blank, second fence at line 5-7.
	//   line 1: ```cm
	//   line 2: a = 1
	//   line 3: ```
	//   line 4: (blank)
	//   line 5: ```cm
	//   line 6: b = 2
	//   line 7: ```
	src := "```cm\na = 1\n```\n\n```cm\nb = 2\n```\n"
	doc, _ := BuildDocument(src)
	calcs := allCalcBlocksWithOffsets(t, doc)
	if len(calcs) != 2 {
		t.Fatalf("expected 2 calc blocks, got %d", len(calcs))
	}
	// First calc: open at line 1 → offset 1.
	if calcs[0].off != 1 {
		t.Errorf("first calc: expected offset 1, got %d", calcs[0].off)
	}
	// Second calc: open at line 5 → offset 5.
	if calcs[1].off != 5 {
		t.Errorf("second calc: expected offset 5, got %d", calcs[1].off)
	}
}

func TestBuildDocument_HostLineOffset_FrontmatterCounts_CalcOffsetSkipsFrontmatter(t *testing.T) {
	// Source layout:
	//   line 1: ---
	//   line 2: globals:
	//   line 3:   price: 100
	//   line 4: ---
	//   line 5: (blank)
	//   line 6: ```cm
	//   line 7: total = price * 2
	//   line 8: ```
	src := "---\nglobals:\n  price: 100\n---\n\n```cm\ntotal = price * 2\n```\n"
	doc, _ := BuildDocument(src)
	cb, off := firstCalcOffset(t, doc)
	if cb == nil {
		t.Fatal("no calc block found")
	}
	if off != 6 {
		t.Errorf("expected HostLineOffset=6 (fence at line 6, after frontmatter), got %d", off)
	}
}

// --- helpers ---

type calcWithOffset struct {
	cb  *specDoc.CalcBlock
	off int
}

func firstCalcOffset(t *testing.T, doc *specDoc.Document) (*specDoc.CalcBlock, int) {
	t.Helper()
	for _, node := range doc.GetBlocks() {
		if cb, ok := node.Block.(*specDoc.CalcBlock); ok {
			return cb, node.HostLineOffset
		}
	}
	return nil, 0
}

func allCalcBlocksWithOffsets(t *testing.T, doc *specDoc.Document) []calcWithOffset {
	t.Helper()
	var out []calcWithOffset
	for _, node := range doc.GetBlocks() {
		if cb, ok := node.Block.(*specDoc.CalcBlock); ok {
			out = append(out, calcWithOffset{cb: cb, off: node.HostLineOffset})
		}
	}
	return out
}
