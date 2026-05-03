package embedded

import (
	"strings"
	"testing"

	specDoc "github.com/CalcMark/go-calcmark/v2/spec/document"
)

// U4: frontmatter handling for Embedded sources.
//
// A leading ---...--- region of the source must be parsed as
// *specDoc.Frontmatter via spec/document.ParseFrontmatter — same
// path NewDocument uses for flat-CM. Frontmatter does NOT also
// appear as a TextBlock. Behavior parity with NewDocument is the
// goal so downstream consumers see the same Document shape.

func TestBuildDocument_Frontmatter_LeadingFenceWithGlobals_StoredOnDocument(t *testing.T) {
	src := "---\nglobals:\n  price: 100\n---\n\n```cm\ntotal = price * 2\n```\n"
	doc, err := BuildDocument(src)
	if err != nil {
		t.Fatalf("BuildDocument: %v", err)
	}

	fm := doc.GetFrontmatter()
	if fm == nil {
		t.Fatal("expected frontmatter to be parsed and stored, got nil")
	}
	// Globals map should contain price.
	if fm.Globals == nil {
		t.Fatalf("expected frontmatter.Globals to be populated, got nil")
	}
	if _, ok := fm.Globals["price"]; !ok {
		t.Errorf("expected globals[price], got globals=%v", fm.Globals)
	}
}

func TestBuildDocument_Frontmatter_DoesNotProduceTextBlockForFenceRegion(t *testing.T) {
	src := "---\nglobals:\n  price: 100\n---\n\n```cm\ntotal = price * 2\n```\n"
	doc, _ := BuildDocument(src)
	blocks := doc.GetBlocks()

	// The frontmatter region itself should NOT be a TextBlock — only
	// the calc fence (and any post-frontmatter prose) becomes blocks.
	for i, node := range blocks {
		if tb, ok := node.Block.(*specDoc.TextBlock); ok {
			content := strings.Join(tb.Source(), "\n")
			if strings.Contains(content, "globals:") || strings.Contains(content, "---") {
				t.Errorf("block[%d]: TextBlock contains frontmatter content (%q) — frontmatter should be stripped from blocks", i, content)
			}
		}
	}
}

func TestBuildDocument_Frontmatter_TrailingProseAfterFence_BecomesTextBlock(t *testing.T) {
	// ParseFrontmatter returns the post-`---` "remaining" string.
	// When that string is non-empty (typical: prose right after the
	// closing `---`), it should project as a TextBlock in front of
	// any subsequent blocks.
	src := "---\nglobals:\n  price: 100\n---\nleading prose\n\n```cm\ntotal = price * 2\n```\n"
	doc, _ := BuildDocument(src)

	fm := doc.GetFrontmatter()
	if fm == nil {
		t.Fatal("expected frontmatter to be parsed, got nil")
	}

	blocks := doc.GetBlocks()
	if len(blocks) < 2 {
		t.Fatalf("expected at least 2 blocks (text + calc), got %d", len(blocks))
	}
	// First non-frontmatter block should be a TextBlock containing the prose.
	tb, ok := blocks[0].Block.(*specDoc.TextBlock)
	if !ok {
		t.Errorf("block[0]: expected *TextBlock for trailing prose, got %T", blocks[0].Block)
	} else if !strings.Contains(strings.Join(tb.Source(), "\n"), "leading prose") {
		t.Errorf("block[0]: missing 'leading prose', got: %q", strings.Join(tb.Source(), "\n"))
	}
}

func TestBuildDocument_Frontmatter_NoFrontmatter_FrontmatterIsNil(t *testing.T) {
	src := "# heading\n\n```cm\nx = 1\n```\n"
	doc, _ := BuildDocument(src)
	if doc.GetFrontmatter() != nil {
		t.Errorf("expected nil frontmatter for no-frontmatter source, got %v", doc.GetFrontmatter())
	}
}

func TestBuildDocument_Frontmatter_MalformedFrontmatter_FallsBackToTextBlock(t *testing.T) {
	// Unclosed `---` (no closing delimiter): ParseFrontmatter returns
	// an error. Builder should NOT crash; falls back to projecting the
	// segment as a TextBlock so the body still renders.
	src := "---\nglobals:\n  price: 100\nno-closing-fence\n\n```cm\nx = 1\n```\n"
	doc, err := BuildDocument(src)
	if err != nil {
		t.Fatalf("BuildDocument should not return an error for malformed frontmatter, got: %v", err)
	}
	if doc.GetFrontmatter() != nil {
		t.Errorf("expected nil frontmatter on parse failure, got %v", doc.GetFrontmatter())
	}
	// Body must still produce blocks (the calc fence at minimum).
	if len(doc.GetBlocks()) == 0 {
		t.Error("expected at least one block (calc) even with malformed frontmatter")
	}
}

func TestBuildDocument_Frontmatter_MidSourceTripleHyphen_NotTreatedAsFrontmatter(t *testing.T) {
	// `---` mid-source is a markdown horizontal rule, not frontmatter.
	// Frontmatter is leading-only per CalcMark spec.
	src := "# heading\n\n---\n\nprose\n\n```cm\nx = 1\n```\n"
	doc, _ := BuildDocument(src)
	if doc.GetFrontmatter() != nil {
		t.Errorf("expected nil frontmatter for mid-source `---`, got %v", doc.GetFrontmatter())
	}
}
