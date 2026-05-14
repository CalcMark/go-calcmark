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
//   - A leading ---...--- segment is parsed as the document's
//     *specDoc.Frontmatter via spec/document.ParseFrontmatter and
//     does NOT also produce a TextBlock. Any text remaining after
//     the closing --- (in the same passthrough segment) projects
//     as a TextBlock.
//
// Empty source returns an empty *Document with zero blocks. Sources
// with zero cm/calcmark fences return a single TextBlock containing
// the whole source (Embedded mode degrades gracefully to "all
// passthrough markdown" for fence-less input).
//
// Frontmatter parse failures (e.g., unclosed `---`) silently fall
// back to projecting the offending segment as a TextBlock so the
// body still renders — matches NewDocument's tolerance for
// malformed frontmatter at the body level.
func BuildDocument(source string) (*specDoc.Document, error) {
	segments := Scan(source)
	if len(segments) == 0 {
		return specDoc.NewDocumentFromBlocks(nil, nil), nil
	}

	var fm *specDoc.Frontmatter
	blocks := make([]specDoc.Block, 0, len(segments))
	// offsets[i] is the host-doc line offset for blocks[i] —
	// applied to the resulting BlockNode after construction so the
	// evaluator's blockLineOffset picks it up. See spec/document
	// BlockNode.HostLineOffset comment for semantics.
	offsets := make([]int, 0, len(segments))

	// cursorLine tracks the 0-based line position in the host-doc
	// where the next-to-be-projected segment begins. Updated after
	// each segment is projected.
	cursorLine := 0

	first := segments[0]
	startIdx := 0
	if first.Kind == Passthrough && hasLeadingFrontmatter(first.Text) {
		parsedFm, remaining, err := specDoc.ParseFrontmatter(first.Text)
		if err == nil {
			fm = parsedFm
			// Compute frontmatter line consumption from the diff
			// between the segment's full text and the post-FM
			// remainder. This is more reliable than introspecting
			// the Frontmatter struct's LineCount() across edge cases.
			fmLines := lineCount(first.Text) - lineCount(remaining)
			cursorLine += fmLines
			if remaining != "" {
				blocks = append(blocks, specDoc.NewTextBlock(splitSegmentText(remaining)))
				offsets = append(offsets, cursorLine)
				cursorLine += lineCount(remaining)
			}
			startIdx = 1
		}
		// On error, fall through (cursorLine stays 0 and the loop
		// below projects the segment as a TextBlock at offset 0).
	}

	for _, seg := range segments[startIdx:] {
		switch seg.Kind {
		case CalcMarkBlock:
			blocks = append(blocks, specDoc.NewCalcBlock(splitSegmentText(seg.Text)))
			// seg.OpenLine is the 1-based host-doc line of the
			// opening fence. Inner content's first line is at
			// host-doc line OpenLine+1, so HostLineOffset = OpenLine
			// (block source line 1 + offset = host line OpenLine+1).
			offsets = append(offsets, seg.OpenLine)
			// Cursor jumps past the close fence: open fence (1) +
			// inner content lines + close fence (1).
			cursorLine = seg.OpenLine + lineCount(seg.Text) + 1
		case Passthrough:
			blocks = append(blocks, specDoc.NewTextBlock(splitSegmentText(seg.Text)))
			offsets = append(offsets, cursorLine)
			cursorLine += lineCount(seg.Text)
		}
	}

	doc := specDoc.NewDocumentFromBlocks(fm, blocks)
	nodes := doc.GetBlocks()
	for i, off := range offsets {
		if i < len(nodes) {
			nodes[i].HostLineOffset = off
		}
	}
	return doc, nil
}

// lineCount returns the number of newline-terminated or
// newline-separated lines in text. Empty text → 0. "a" → 1.
// "a\n" → 1. "a\nb" → 2. "a\nb\n" → 2. Used to advance the host-doc
// line cursor as BuildDocument projects each segment.
func lineCount(text string) int {
	if text == "" {
		return 0
	}
	n := strings.Count(text, "\n")
	if !strings.HasSuffix(text, "\n") {
		n++
	}
	return n
}

// hasLeadingFrontmatter reports whether the given text begins with
// `---\n` — the opening of a YAML frontmatter block. Used to gate
// the frontmatter-parsing branch in BuildDocument so non-frontmatter
// passthrough segments aren't fed through ParseFrontmatter
// unnecessarily.
func hasLeadingFrontmatter(text string) bool {
	return strings.HasPrefix(text, "---\n") || strings.HasPrefix(text, "---\r\n")
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
	text = strings.TrimSuffix(text, "\n")
	return strings.Split(text, "\n")
}
