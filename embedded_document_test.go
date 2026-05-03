package calcmark

import (
	"strings"
	"testing"

	specDoc "github.com/CalcMark/go-calcmark/v2/spec/document"
)

// NewDocumentEmbedded is the structural-parsing peer to NewDocument
// for Embedded-mode sources (markdown with cm/calcmark fenced code
// blocks). The deep per-segment-projection coverage lives in
// impl/embedded/document_test.go; these top-level tests just
// verify the facade is wired correctly and the public surface
// behaves end-to-end.

func TestNewDocumentEmbedded_EmptySource_ReturnsEmptyDocument(t *testing.T) {
	doc, err := NewDocumentEmbedded("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc == nil {
		t.Fatal("expected non-nil *Document, got nil")
	}
	if len(doc.GetBlocks()) != 0 {
		t.Errorf("expected 0 blocks, got %d", len(doc.GetBlocks()))
	}
}

func TestNewDocumentEmbedded_FlatProseSource_ReturnsSingleTextBlock(t *testing.T) {
	// No fences = degrades to "all passthrough markdown" in one TextBlock.
	doc, err := NewDocumentEmbedded("# heading\n\nprose paragraph\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := doc.GetBlocks()
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if _, ok := blocks[0].Block.(*specDoc.TextBlock); !ok {
		t.Errorf("expected *TextBlock, got %T", blocks[0].Block)
	}
}

func TestNewDocumentEmbedded_HeadingThenFenceThenProse_ProjectsThreeBlocks(t *testing.T) {
	src := "# heading\n\nprose before.\n\n```cm\nprice = 100\n```\n\nprose after.\n"
	doc, err := NewDocumentEmbedded(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := doc.GetBlocks()
	if len(blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(blocks))
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
	// Calc block source is the fence inner content, no fence delimiters.
	cb := blocks[1].Block.(*specDoc.CalcBlock)
	if got := strings.Join(cb.Source(), "\n"); !strings.Contains(got, "price = 100") || strings.Contains(got, "```") {
		t.Errorf("calc block source: got %q (expected to contain 'price = 100' and not '```')", got)
	}
}
