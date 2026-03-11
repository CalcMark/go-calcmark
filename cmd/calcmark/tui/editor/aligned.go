package editor

import (
	"strings"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/geometry"
)

// AlignedModel represents the computed visual line structure for both panes.
// This is a pure computation result - no methods, just data.
// It's computed once when inputs change and cached until invalidation.
//
// Unlike LineModel (in linemodel.go), AlignedModel includes:
// - Bidirectional mappings (source<->visual)
// - Cursor tracking
// - Block metadata for styling
// - Invariant checking for debugging
type AlignedModel struct {
	// SourceLines contains all visual lines for the source pane.
	// This may include wrapped continuation lines and padding lines.
	SourceLines []AlignedLine

	// PreviewLines contains all visual lines for the preview pane.
	// Always has the same length as SourceLines for 1:1 alignment.
	PreviewLines []AlignedLine

	// SourceToVisual maps source line index to the first visual line index.
	// Used for cursor positioning and scroll synchronization.
	SourceToVisual map[int]int

	// VisualToSource maps visual line index to source line index.
	// Used for reverse lookups (e.g., clicking on a visual line).
	VisualToSource map[int]int

	// TotalSourceLines is the number of source lines in the document.
	TotalSourceLines int

	// TotalVisualLines is the total number of visual lines (len(SourceLines)).
	TotalVisualLines int
}

// AlignedLine represents a single visual line in either pane.
// This extends the basic VisualLine with additional metadata for alignment.
type AlignedLine struct {
	// Content is the text content for this visual line.
	Content string

	// SourceLineIdx is the source document line this visual line belongs to.
	SourceLineIdx int

	// LineNum is the display line number (1-indexed, 0 means no line number shown).
	LineNum int

	// Kind indicates the type of visual line.
	Kind AlignedLineKind

	// BlockID is the ID of the block this line belongs to (for preview styling).
	BlockID string

	// IsCalc indicates if this line is from a CalcBlock (for preview styling).
	IsCalc bool

	// IsFrontmatter indicates if this line is from the YAML frontmatter block.
	IsFrontmatter bool
}

// AlignedLineKind categorizes how a visual line should be rendered.
type AlignedLineKind int

const (
	// AlignedLineNormal is a regular source line with line number.
	AlignedLineNormal AlignedLineKind = iota

	// AlignedLineWrapped is a wrapped continuation of the previous line.
	AlignedLineWrapped

	// AlignedLinePadding is an empty line for alignment (when other pane has more lines).
	AlignedLinePadding

	// AlignedLineCursor is the line where the cursor is positioned.
	AlignedLineCursor

	// AlignedLineCursorWrapped is a wrapped continuation of the cursor line.
	AlignedLineCursorWrapped
)

// AlignedModelInput contains all inputs needed to compute an AlignedModel.
// Comparing these inputs allows efficient cache invalidation.
type AlignedModelInput struct {
	// Source content
	Lines   []string
	Results []LineResult

	// Pane dimensions (visual width after accounting for line numbers, etc.)
	SourceContentWidth int
	PreviewWidth       int

	// Cursor state (affects which line is marked as cursor)
	CursorLine int

	// Preview mode affects how calc results are rendered
	PreviewMode PreviewMode

	// EditBuf: Text currently being typed (not yet saved to document).
	// If non-empty, this overrides Lines[EditBufLine] for rendering preview.
	EditBuf     string
	EditBufLine int // Which line index EditBuf applies to

	// RenderedTextBlocks contains pre-rendered glamour output for TextBlocks.
	// Keyed by block ID. When present for a block, the rendered content is used
	// for preview instead of the per-line rendering callbacks.
	// Nil map or missing block ID falls back to existing behavior.
	RenderedTextBlocks map[string][]string
}

// ComputeAlignedModel computes the visual line alignment from the given inputs.
// This is a pure function - same inputs always produce same outputs.
func ComputeAlignedModel(input AlignedModelInput, renderCalcLine func(r LineResult, width int) string, renderMarkdown func(line string, width int) []string) AlignedModel {
	var sourceLines []AlignedLine
	var previewLines []AlignedLine
	sourceToVisual := make(map[int]int)
	visualToSource := make(map[int]int)

	// Process results block by block to maintain document structure.
	// A "block" is either a CalcBlock or TextBlock from the document model.
	blockStartIdx := 0
	for blockStartIdx < len(input.Results) {
		currentBlockID := input.Results[blockStartIdx].BlockID
		isCalcBlock := input.Results[blockStartIdx].IsCalc

		// Collect all LineResults belonging to this block.
		blockEndIdx := blockStartIdx
		for blockEndIdx < len(input.Results) && input.Results[blockEndIdx].BlockID == currentBlockID {
			blockEndIdx++
		}
		blockResults := input.Results[blockStartIdx:blockEndIdx]

		// Pre-render text block preview lines (headings, ordered lists, etc.)
		var textBlockPreviewCache map[int][]string
		if !isCalcBlock && renderMarkdown != nil {
			textBlockPreviewCache = renderTextBlockPreview(blockResults, input, renderMarkdown)
		}

		// Process each source line in the block to create aligned visual lines.
		for blockLineIdx, lineResult := range blockResults {
			if lineResult.LineNum >= len(input.Lines) {
				continue
			}

			sourceText := effectiveSourceText(input, lineResult)
			isCursorOnThisLine := lineResult.LineNum == input.CursorLine
			wrappedSourceLines := geometry.WrapText(sourceText, input.SourceContentWidth)

			// Determine preview content: calc result, cached markdown, or fallback.
			wrappedPreviewLines := resolvePreviewLines(
				lineResult, blockLineIdx, isCalcBlock,
				textBlockPreviewCache, input, renderCalcLine, renderMarkdown,
			)

			// Align source and preview visual lines to the same count.
			numAligned := max(len(wrappedSourceLines), len(wrappedPreviewLines))

			// Record source→visual mapping for cursor tracking.
			if _, ok := sourceToVisual[lineResult.LineNum]; !ok {
				sourceToVisual[lineResult.LineNum] = len(sourceLines)
			}

			for offset := range numAligned {
				visualToSource[len(sourceLines)] = lineResult.LineNum

				sourceLines = append(sourceLines, buildSourceVisualLine(
					offset, wrappedSourceLines, lineResult, currentBlockID, isCalcBlock, isCursorOnThisLine,
				))
				previewLines = append(previewLines, buildPreviewVisualLine(
					offset, wrappedPreviewLines, lineResult, currentBlockID, isCalcBlock,
				))
			}
		}

		blockStartIdx = blockEndIdx
	}

	return AlignedModel{
		SourceLines:      sourceLines,
		PreviewLines:     previewLines,
		SourceToVisual:   sourceToVisual,
		VisualToSource:   visualToSource,
		TotalSourceLines: len(input.Lines),
		TotalVisualLines: len(sourceLines),
	}
}

// effectiveSourceText returns the source text for a line, preferring editBuf
// when the user is actively typing on that line.
func effectiveSourceText(input AlignedModelInput, lr LineResult) string {
	if input.EditBuf != "" && lr.LineNum == input.EditBufLine {
		return input.EditBuf
	}
	return input.Lines[lr.LineNum]
}

// renderTextBlockPreview pre-renders markdown preview lines for a text block.
// Multi-line constructs (ordered lists) are rendered as a unit to preserve
// semantic numbering; other content is rendered per-line.
//
// When RenderedTextBlocks contains pre-rendered content for this block (from
// the RenderedBlockCache in PreviewRendered mode), that content is distributed
// across source lines using even distribution. Otherwise, falls back to the
// existing per-line or ordered-list rendering.
//
// Returns a map from block-line-index to rendered preview lines.
func renderTextBlockPreview(blockResults []LineResult, input AlignedModelInput, renderMarkdown func(string, int) []string) map[int][]string {
	cache := make(map[int][]string, len(blockResults))

	// Use pre-rendered content from RenderedBlockCache when available.
	if len(blockResults) > 0 && input.RenderedTextBlocks != nil {
		blockID := blockResults[0].BlockID
		if rendered, ok := input.RenderedTextBlocks[blockID]; ok {
			distributeRenderedLines(rendered, len(blockResults), cache)
			return cache
		}
	}

	if containsOrderedList(blockResults) {
		renderOrderedListBlock(blockResults, input, renderMarkdown, cache)
	} else {
		renderPerLinePreview(blockResults, input, renderMarkdown, cache)
	}
	return cache
}

// distributeRenderedLines distributes pre-rendered lines evenly across source lines.
// Same algorithm as renderOrderedListBlock but without re-rendering.
func distributeRenderedLines(rendered []string, numSource int, cache map[int][]string) {
	numRendered := len(rendered)
	perSource := numRendered / numSource
	remainder := numRendered % numSource

	idx := 0
	for i := range numSource {
		count := perSource
		if i < remainder {
			count++
		}
		var chunk []string
		for j := 0; j < count && idx < numRendered; j++ {
			chunk = append(chunk, rendered[idx])
			idx++
		}
		if len(chunk) == 0 {
			chunk = []string{""}
		}
		cache[i] = chunk
	}
}

// containsOrderedList checks if any line in the block starts with an ordered
// list marker (e.g., "1. " or "1) ").
func containsOrderedList(blockResults []LineResult) bool {
	for _, lr := range blockResults {
		line := lr.Source
		if len(line) >= 3 && line[0] >= '0' && line[0] <= '9' {
			if (line[1] == '.' || line[1] == ')') && line[2] == ' ' {
				return true
			}
		}
	}
	return false
}

// renderOrderedListBlock renders the entire text block as one markdown unit so
// that ordered list numbers are renumbered correctly (e.g., 1→2→3 not 1→1→1).
// Rendered lines are distributed evenly across source lines for alignment.
func renderOrderedListBlock(blockResults []LineResult, input AlignedModelInput, renderMarkdown func(string, int) []string, cache map[int][]string) {
	var blockText strings.Builder
	for i, lr := range blockResults {
		if i > 0 {
			blockText.WriteString("\n")
		}
		if input.EditBuf != "" && lr.LineNum == input.EditBufLine {
			blockText.WriteString(input.EditBuf)
		} else {
			blockText.WriteString(lr.Source)
		}
	}

	allRendered := renderMarkdown(blockText.String(), input.PreviewWidth)
	numSource := len(blockResults)
	numRendered := len(allRendered)
	perSource := numRendered / numSource
	remainder := numRendered % numSource

	idx := 0
	for i := range numSource {
		count := perSource
		if i < remainder {
			count++
		}
		var chunk []string
		for j := 0; j < count && idx < numRendered; j++ {
			chunk = append(chunk, allRendered[idx])
			idx++
		}
		if len(chunk) == 0 {
			chunk = []string{""}
		}
		cache[i] = chunk
	}
}

// renderPerLinePreview renders each text block line individually. Only headings
// and empty lines are shown in the preview pane; other markdown is filtered out.
func renderPerLinePreview(blockResults []LineResult, input AlignedModelInput, renderMarkdown func(string, int) []string, cache map[int][]string) {
	for i, lr := range blockResults {
		lineText := lr.Source
		if input.EditBuf != "" && lr.LineNum == input.EditBufLine {
			lineText = input.EditBuf
		}

		trimmed := strings.TrimSpace(lineText)
		switch {
		case trimmed == "":
			cache[i] = []string{""}
		case strings.HasPrefix(trimmed, "#"):
			cache[i] = renderMarkdown(lineText, input.PreviewWidth)
		default:
			cache[i] = []string{""}
		}
	}
}

// resolvePreviewLines determines the preview content for a single source line.
// Priority: calc result > cached text block > fallback markdown > plain text.
func resolvePreviewLines(
	lr LineResult, blockLineIdx int, isCalcBlock bool,
	textCache map[int][]string, input AlignedModelInput,
	renderCalcLine func(LineResult, int) string,
	renderMarkdown func(string, int) []string,
) []string {
	var lines []string

	switch {
	case isCalcBlock && renderCalcLine != nil:
		lines = wrapStyledLine(renderCalcLine(lr, input.PreviewWidth), input.PreviewWidth)
	case !isCalcBlock && textCache != nil:
		lines = textCache[blockLineIdx]
	case renderMarkdown != nil:
		lines = renderMarkdown(lr.Source, input.PreviewWidth)
	default:
		lines = geometry.WrapText(lr.Source, input.PreviewWidth)
	}

	if len(lines) == 0 {
		lines = []string{""}
	}
	return lines
}

// buildSourceVisualLine builds a single source-pane visual line for the given
// offset within the wrapped source content.
func buildSourceVisualLine(offset int, wrappedSource []string, lr LineResult, blockID string, isCalc, isCursor bool) AlignedLine {
	if offset >= len(wrappedSource) {
		// Padding: preview wrapped more than source.
		return AlignedLine{
			SourceLineIdx: lr.LineNum, Kind: AlignedLinePadding,
			BlockID: blockID, IsCalc: isCalc,
		}
	}
	al := AlignedLine{
		Content: wrappedSource[offset], SourceLineIdx: lr.LineNum,
		BlockID: blockID, IsCalc: isCalc,
	}
	if offset == 0 {
		al.LineNum = lr.LineNum + 1 // 1-indexed for display
		if isCursor {
			al.Kind = AlignedLineCursor
		} else {
			al.Kind = AlignedLineNormal
		}
	} else {
		if isCursor {
			al.Kind = AlignedLineCursorWrapped
		} else {
			al.Kind = AlignedLineWrapped
		}
	}
	return al
}

// buildPreviewVisualLine builds a single preview-pane visual line for the given
// offset within the wrapped preview content.
func buildPreviewVisualLine(offset int, wrappedPreview []string, lr LineResult, blockID string, isCalc bool) AlignedLine {
	if offset >= len(wrappedPreview) {
		// Padding: source wrapped more than preview.
		return AlignedLine{
			SourceLineIdx: lr.LineNum, Kind: AlignedLinePadding,
			BlockID: blockID, IsCalc: isCalc,
		}
	}
	al := AlignedLine{
		Content: wrappedPreview[offset], SourceLineIdx: lr.LineNum,
		BlockID: blockID, IsCalc: isCalc,
	}
	if offset == 0 {
		al.LineNum = lr.LineNum + 1
		al.Kind = AlignedLineNormal
	} else {
		al.Kind = AlignedLineWrapped
	}
	return al
}

// CursorVisualLine returns the visual line index for the given source line.
// Returns -1 if the source line is not in the mapping.
func (a *AlignedModel) CursorVisualLine(sourceLine int) int {
	if v, ok := a.SourceToVisual[sourceLine]; ok {
		return v
	}
	return -1
}

// SourceLineAt returns the source line index for the given visual line.
// Returns -1 if the visual line is out of bounds.
func (a *AlignedModel) SourceLineAt(visualLine int) int {
	if s, ok := a.VisualToSource[visualLine]; ok {
		return s
	}
	return -1
}

// VisibleRange calculates the range of visual lines to display given scroll offset and height.
// Returns (start, end) indices where end is exclusive.
func (a *AlignedModel) VisibleRange(scrollOffset, height int) (start, end int) {
	start = max(scrollOffset, 0)
	if start >= a.TotalVisualLines {
		start = max(0, a.TotalVisualLines-1)
	}

	end = min(start+height, a.TotalVisualLines)

	return start, end
}

// ScrollOffsetForCursor calculates the scroll offset needed to keep the cursor visible.
// Returns the adjusted scroll offset.
func (a *AlignedModel) ScrollOffsetForCursor(cursorSourceLine, currentScrollOffset, viewportHeight int) int {
	cursorVisual := a.CursorVisualLine(cursorSourceLine)
	if cursorVisual < 0 {
		return currentScrollOffset
	}

	// Ensure cursor is within visible range
	if cursorVisual < currentScrollOffset {
		return cursorVisual
	}
	if cursorVisual >= currentScrollOffset+viewportHeight {
		return cursorVisual - viewportHeight + 1
	}

	return currentScrollOffset
}

// Invariants returns a set of boolean checks for the model's consistency.
// Used for debugging and testing.
func (a *AlignedModel) Invariants() AlignedModelInvariants {
	// Check source/preview line count match
	sourcePreviewMatch := len(a.SourceLines) == len(a.PreviewLines)

	// Check all source lines have mappings
	mappingComplete := true
	for i := 0; i < a.TotalSourceLines; i++ {
		if _, ok := a.SourceToVisual[i]; !ok {
			mappingComplete = false
			break
		}
	}

	// Check visual-to-source mapping is complete
	reverseComplete := true
	for i := 0; i < a.TotalVisualLines; i++ {
		if _, ok := a.VisualToSource[i]; !ok {
			reverseComplete = false
			break
		}
	}

	return AlignedModelInvariants{
		SourcePreviewMatch: sourcePreviewMatch,
		MappingComplete:    mappingComplete,
		ReverseComplete:    reverseComplete,
	}
}

// AlignedModelInvariants holds consistency check results.
type AlignedModelInvariants struct {
	SourcePreviewMatch bool // Source and preview have same line count
	MappingComplete    bool // All source lines have visual mappings
	ReverseComplete    bool // All visual lines have source mappings
}
