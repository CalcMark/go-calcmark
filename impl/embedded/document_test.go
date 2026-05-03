package embedded

import (
	"strings"
	"testing"

	specDoc "github.com/CalcMark/go-calcmark/v2/spec/document"
)

// BuildDocument is the structural parser front-end for Embedded-mode
// sources. It scans for cm/calcmark fences, projects each segment
// into a single block of the matching kind (calc fence -> CalcBlock,
// passthrough markdown -> TextBlock), and stitches them into a real
// *specDoc.Document via spec/document.NewDocumentFromBlocks.
//
// Per-segment projection rule (load-bearing decision from the
// 2026-05-02-001 plan): exactly ONE block per segment. Do NOT call
// Detector.DetectBlocks on segment content — that would reintroduce
// the heuristic-bleed bug calcmark-web's prove-it surfaced. The
// fence boundaries from Scan are the truth.
//
// Frontmatter handling is U4's concern; U2 always passes nil for
// the frontmatter argument (so a leading ---...--- segment becomes
// a TextBlock until U4 refines).

func TestBuildDocument_EmptySource_ReturnsEmptyDocument(t *testing.T) {
	doc, err := BuildDocument("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc == nil {
		t.Fatal("expected non-nil *Document, got nil")
	}
	if len(doc.GetBlocks()) != 0 {
		t.Errorf("expected 0 blocks for empty source, got %d", len(doc.GetBlocks()))
	}
}

func TestBuildDocument_FlatProseSource_ReturnsSingleTextBlock(t *testing.T) {
	// No fences = one passthrough segment = one TextBlock.
	src := "# heading\n\nprose paragraph\n"
	doc, err := BuildDocument(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := doc.GetBlocks()
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	tb, ok := blocks[0].Block.(*specDoc.TextBlock)
	if !ok {
		t.Fatalf("expected *TextBlock, got %T", blocks[0].Block)
	}
	got := strings.Join(tb.Source(), "\n")
	if !strings.Contains(got, "heading") || !strings.Contains(got, "prose paragraph") {
		t.Errorf("text block content lost expected lines, got: %q", got)
	}
}

func TestBuildDocument_SingleCmFence_ReturnsCalcBlock(t *testing.T) {
	// Just a fence — no surrounding prose.
	src := "```cm\nprice = 100\n```\n"
	doc, err := BuildDocument(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := doc.GetBlocks()
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	cb, ok := blocks[0].Block.(*specDoc.CalcBlock)
	if !ok {
		t.Fatalf("expected *CalcBlock, got %T", blocks[0].Block)
	}
	got := strings.Join(cb.Source(), "\n")
	if !strings.Contains(got, "price = 100") {
		t.Errorf("calc block lost expected line, got: %q", got)
	}
	// Fence delimiters must NOT appear in the calc block source —
	// the fence is the boundary, the inner content is the source.
	if strings.Contains(got, "```") {
		t.Errorf("calc block source should not contain fence delimiters, got: %q", got)
	}
}

func TestBuildDocument_HeadingThenFenceThenProse_ProjectsThreeBlocks(t *testing.T) {
	// Canonical mixed-content shape — one TextBlock for the leading
	// markdown, one CalcBlock for the fence, one TextBlock for the
	// trailing markdown.
	src := "# heading\n\nprose before.\n\n```cm\nprice = 100\n```\n\nprose after.\n"
	doc, err := BuildDocument(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := doc.GetBlocks()
	if len(blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d (kinds: %s)", len(blocks), blockKinds(blocks))
	}
	if _, ok := blocks[0].Block.(*specDoc.TextBlock); !ok {
		t.Errorf("block[0]: expected *TextBlock, got %T", blocks[0].Block)
	}
	if _, ok := blocks[1].Block.(*specDoc.CalcBlock); !ok {
		t.Errorf("block[1]: expected *CalcBlock, got %T", blocks[1].Block)
	}
	if _, ok := blocks[2].Block.(*specDoc.TextBlock); !ok {
		t.Errorf("block[2]: expected *TextBlock, got %T", blocks[2].Block)
	}
}

func TestBuildDocument_LongFormCalcmarkInfoString_ProjectsCalcBlock(t *testing.T) {
	src := "```calcmark\nx = 1\n```\n"
	doc, _ := BuildDocument(src)
	blocks := doc.GetBlocks()
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if _, ok := blocks[0].Block.(*specDoc.CalcBlock); !ok {
		t.Errorf("```calcmark fence: expected *CalcBlock, got %T", blocks[0].Block)
	}
}

func TestBuildDocument_TildeFence_ProjectsCalcBlock(t *testing.T) {
	src := "~~~cm\nprice = 100\n~~~\n"
	doc, _ := BuildDocument(src)
	blocks := doc.GetBlocks()
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if _, ok := blocks[0].Block.(*specDoc.CalcBlock); !ok {
		t.Errorf("~~~cm fence: expected *CalcBlock, got %T", blocks[0].Block)
	}
}

func TestBuildDocument_AttributeInfoString_ProjectsCalcBlock(t *testing.T) {
	// Hugo-style attributes after the info-string.
	src := "```cm {.highlight}\nx = 1\n```\n"
	doc, _ := BuildDocument(src)
	blocks := doc.GetBlocks()
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if _, ok := blocks[0].Block.(*specDoc.CalcBlock); !ok {
		t.Errorf("```cm with attributes: expected *CalcBlock, got %T", blocks[0].Block)
	}
}

func TestBuildDocument_TextFence_ProjectsAsTextBlockContent(t *testing.T) {
	// Static-rendering escape hatch: ```text is regular markdown,
	// projects as part of the surrounding TextBlock — NOT a CalcBlock.
	src := "```text\nstatic example\n```\n"
	doc, _ := BuildDocument(src)
	blocks := doc.GetBlocks()
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block (passthrough), got %d", len(blocks))
	}
	if _, ok := blocks[0].Block.(*specDoc.TextBlock); !ok {
		t.Errorf("```text fence: expected *TextBlock, got %T (would defeat the static-rendering escape hatch)", blocks[0].Block)
	}
}

func TestBuildDocument_CmExtraInfoString_ProjectsAsTextBlock(t *testing.T) {
	// Per scanner spec: info-string token must be EXACTLY `cm` or
	// `calcmark`. `cm-extra` is one token and is NOT a CalcMarkBlock.
	src := "```cm-extra\nfoo = bar\n```\n"
	doc, _ := BuildDocument(src)
	blocks := doc.GetBlocks()
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if _, ok := blocks[0].Block.(*specDoc.TextBlock); !ok {
		t.Errorf("```cm-extra: expected *TextBlock, got %T", blocks[0].Block)
	}
}

func TestBuildDocument_TwoCmFencesSeparatedByProse_ProjectsFiveBlocks(t *testing.T) {
	src := "intro\n\n```cm\nprice = 100\n```\n\nmiddle prose\n\n```cm\ntax = price * 0.1\n```\n\nfooter\n"
	doc, _ := BuildDocument(src)
	blocks := doc.GetBlocks()
	if len(blocks) != 5 {
		t.Fatalf("expected 5 blocks (text/calc/text/calc/text), got %d (kinds: %s)", len(blocks), blockKinds(blocks))
	}
	expected := []string{"text", "calc", "text", "calc", "text"}
	for i, want := range expected {
		got := kindOf(blocks[i].Block)
		if got != want {
			t.Errorf("block[%d]: expected %s, got %s", i, want, got)
		}
	}
}

func TestBuildDocument_EmptyCmFence_ProjectsEmptyCalcBlock(t *testing.T) {
	src := "```cm\n```\n"
	doc, err := BuildDocument(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := doc.GetBlocks()
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block (empty calc), got %d", len(blocks))
	}
	cb, ok := blocks[0].Block.(*specDoc.CalcBlock)
	if !ok {
		t.Fatalf("expected *CalcBlock, got %T", blocks[0].Block)
	}
	src_lines := cb.Source()
	if len(src_lines) > 1 || (len(src_lines) == 1 && src_lines[0] != "") {
		t.Errorf("expected empty calc source, got %q", src_lines)
	}
}

func TestBuildDocument_UniqueBlockUUIDs(t *testing.T) {
	src := "intro\n\n```cm\nx = 1\n```\n\n```cm\ny = 2\n```\n"
	doc, _ := BuildDocument(src)
	seen := map[string]bool{}
	for _, node := range doc.GetBlocks() {
		if seen[node.ID] {
			t.Errorf("duplicate UUID: %s", node.ID)
		}
		seen[node.ID] = true
	}
}

// --- helpers ---

func kindOf(b specDoc.Block) string {
	switch b.(type) {
	case *specDoc.CalcBlock:
		return "calc"
	case *specDoc.TextBlock:
		return "text"
	default:
		return "unknown"
	}
}

func blockKinds(nodes []*specDoc.BlockNode) string {
	parts := make([]string, len(nodes))
	for i, n := range nodes {
		parts[i] = kindOf(n.Block)
	}
	return strings.Join(parts, ",")
}
