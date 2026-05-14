//go:build !wasm

package document

import (
	"fmt"
	"strings"
	"testing"
)

// Debug helper — dumps the joined render artifacts so the test
// failures show what goldmark produced. Comment out the t.Fatal
// in production tests so a debug print survives.
func dumpJoinedRender(t *testing.T, blocks []*TextBlock) {
	t.Helper()
	var b strings.Builder
	for i, blk := range blocks {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(blk.InterpolatedHTMLSourceText())
		b.WriteString("\n\n")
		fmt.Fprintf(&b, crossBlockMarkerFmt, i)
	}
	t.Logf("joined source:\n%s\n", b.String())
	t.Logf("rendered:\n%s\n", renderMarkdown(b.String()))
}

// Sanity test for the joined renderer: a single text block with a
// footnote reference + definition in the same source should render
// normally (covers the no-split case).
func TestRenderTextBlocksJoined_SingleBlock_Footnote(t *testing.T) {
	tb := NewTextBlock([]string{
		"Reference here.[^note1]",
		"",
		"[^note1]: Cross-block footnote definition.",
	})
	htmls, trailing, ok := RenderTextBlocksJoined([]*TextBlock{tb})
	if !ok {
		dumpJoinedRender(t, []*TextBlock{tb})
		t.Fatal("joined render reported failure for single block")
	}
	if len(htmls) != 1 {
		t.Fatalf("expected 1 block HTML, got %d", len(htmls))
	}
	// Reference must resolve to a sup/a — the literal token must not survive.
	if strings.Contains(htmls[0], "[^note1]") {
		t.Errorf("single-block: reference unresolved: %q", htmls[0])
	}
	// Footnote definitions get extracted to the trailing section.
	if !strings.Contains(trailing, "footnote") {
		t.Errorf("trailing should contain footnotes section, got: %q", trailing)
	}
}

// Two blocks with the reference in #1 and the definition in #2 —
// the cross-block case from issue #129. The reference should resolve.
func TestRenderTextBlocksJoined_RefAndDefSplit(t *testing.T) {
	block1 := NewTextBlock([]string{"Reference here.[^note1]"})
	block2 := NewTextBlock([]string{"[^note1]: Cross-block footnote definition."})
	htmls, trailing, ok := RenderTextBlocksJoined([]*TextBlock{block1, block2})
	if !ok {
		t.Fatal("joined render reported failure")
	}
	if len(htmls) != 2 {
		t.Fatalf("expected 2 block HTML pieces, got %d", len(htmls))
	}
	if strings.Contains(htmls[0], "[^note1]") {
		t.Errorf("block 1 reference unresolved: %q\ntrailing: %q",
			htmls[0], trailing)
	}
	if !strings.Contains(trailing, "footnote") {
		t.Errorf("trailing should contain footnotes section, got: %q", trailing)
	}
	// block 2 is definition-only; once goldmark extracts the
	// definition to the trailing section, block 2 should be empty
	// (or whitespace-only).
	if strings.Contains(htmls[1], "[^note1]:") {
		t.Errorf("block 2 still contains raw definition: %q", htmls[1])
	}
	if false { // debug dump on demand
		fmt.Println("block1:", htmls[0])
		fmt.Println("block2:", htmls[1])
		fmt.Println("trailing:", trailing)
	}
}
