// Package embedded — Embedded-mode structural parser.
//
// Scan() (defined in scanner.go) provides single-pass segmentation
// of a markdown source into Passthrough and CalcMarkBlock segments.
// BuildDocument() projects those segments into a real
// *spec/document.Document with proper block kinds: each cm/calcmark
// fence becomes one *CalcBlock, each passthrough segment becomes
// one *TextBlock, all wrapped in BlockNodes with UUIDs and stitched
// together via spec/document.NewDocumentFromBlocks.
//
// All CalcBlocks land in the SAME *Document, so the existing
// impldoc.Evaluator naturally resolves variables across fences
// (whole-doc scoping). This intentionally diverges from the
// rendering pipeline at convert.go:convertEmbedded, which evaluates
// each fence in isolation. Consumers that want cross-fence variable
// resolution and passthrough interpolation use BuildDocument (via
// the top-level calcmark.NewDocumentEmbedded facade); consumers
// that want per-fence-isolated rendering keep using Convert.
//
// Per-segment projection rule (from the 2026-05-02-001 plan, U2):
// EXACTLY ONE block per segment. Do NOT call Detector.DetectBlocks
// on segment content — that would reintroduce the heuristic-bleed
// bug that motivated this whole architecture (see calcmark-web's
// prove-it run 2026-05-02-1315). The fence boundaries from Scan
// are the truth; per-segment content is one block of its kind.
//
// Frontmatter handling lands in U4 — for now BuildDocument always
// passes nil for the frontmatter argument so a leading ---...---
// segment becomes a TextBlock.
package embedded

import (
	"strings"

	specDoc "github.com/CalcMark/go-calcmark/v2/spec/document"
)

// BuildDocument scans an Embedded-mode source (markdown with
// cm/calcmark fences) into a *spec/document.Document whose blocks
// reflect the fence boundaries:
//
//   - Each cm/calcmark fenced segment becomes one *specDoc.CalcBlock
//     with the fence's inner content as its source.
//   - Each passthrough markdown segment becomes one *specDoc.TextBlock
//     with the passthrough text as its source.
//
// Empty source returns an empty *Document with zero blocks. Sources
// with zero cm/calcmark fences return a single TextBlock containing
// the whole source (Embedded mode degrades gracefully to "all
// passthrough markdown" for fence-less input).
//
// Currently always returns nil for the error — the underlying Scan
// is total. The error return exists for future expansion (e.g.,
// frontmatter parse errors in U4) without breaking the signature.
func BuildDocument(source string) (*specDoc.Document, error) {
	segments := Scan(source)
	blocks := make([]specDoc.Block, 0, len(segments))
	for _, seg := range segments {
		blocks = append(blocks, projectSegment(seg))
	}
	return specDoc.NewDocumentFromBlocks(nil, blocks), nil
}

// projectSegment turns one embedded.Segment into one specDoc.Block.
// Fence segments project as CalcBlocks; passthrough segments project
// as TextBlocks. The "exactly one block per segment" rule is what
// keeps fence boundaries authoritative — see package doc.
func projectSegment(seg Segment) specDoc.Block {
	lines := splitSegmentText(seg.Text)
	if seg.Kind == CalcMarkBlock {
		return specDoc.NewCalcBlock(lines)
	}
	return specDoc.NewTextBlock(lines)
}

// splitSegmentText splits a segment's text into lines for the
// CalcBlock/TextBlock []string source representation. Embedded
// segment text typically ends with a trailing newline (the
// scanner preserves line terminators); strings.Split would
// produce a phantom empty trailing element which we strip so the
// block's Source() matches what the equivalent flat-CM parse
// would produce.
func splitSegmentText(text string) []string {
	if text == "" {
		return nil
	}
	// Trim a single trailing newline so "a\nb\n" splits to ["a","b"]
	// not ["a","b",""]. Multi-trailing-newlines (rare) collapse to
	// at most one trailing empty string, which matches today's
	// flat-CM block source semantics for blank-line-terminated
	// blocks.
	if strings.HasSuffix(text, "\n") {
		text = text[:len(text)-1]
	}
	return strings.Split(text, "\n")
}
