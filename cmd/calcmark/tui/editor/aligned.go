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
}

// ComputeAlignedModel computes the visual line alignment from the given inputs.
// This is a pure function - same inputs always produce same outputs.
func ComputeAlignedModel(input AlignedModelInput, renderCalcLine func(r LineResult, width int) string, renderMarkdown func(line string, width int) []string) AlignedModel {
	var sourceLines []AlignedLine
	var previewLines []AlignedLine
	sourceToVisual := make(map[int]int)
	visualToSource := make(map[int]int)

	// IMPORTANT: Process results block by block to maintain document structure.
	// A "block" is either a CalcBlock or TextBlock from the document model.
	// All LineResults with the same BlockID belong to the same block.
	blockStartIdx := 0
	for blockStartIdx < len(input.Results) {
		// Identify the current block
		currentBlockID := input.Results[blockStartIdx].BlockID
		isCalcBlock := input.Results[blockStartIdx].IsCalc

		// Collect all LineResults belonging to this block
		var blockResults []LineResult
		blockEndIdx := blockStartIdx
		for blockEndIdx < len(input.Results) && input.Results[blockEndIdx].BlockID == currentBlockID {
			blockResults = append(blockResults, input.Results[blockEndIdx])
			blockEndIdx++
		}

		// === CRITICAL SECTION: TextBlock Rendering ===
		// TextBlocks contain markdown (headers, lists, paragraphs, etc.)
		// Multi-line markdown constructs (ordered lists, unordered lists) MUST be rendered
		// as a complete unit, not line-by-line, to preserve semantic meaning.
		//
		// Example: This source:
		//   1. First item
		//   1. Second item
		//   1. Third item
		//
		// Must be rendered as a single block to produce:
		//   1. First item
		//   2. Second item    <- Note: "2." not "1."
		//   3. Third item     <- Note: "3." not "1."
		//
		// If we rendered line-by-line, each line would independently render as "1."
		//
		// Strategy:
		// 1. Join all source lines in the block with newlines
		// 2. Pass complete block to renderMarkdown() which uses glamour
		// 3. Glamour correctly handles multi-line markdown semantics
		// 4. Distribute rendered output back to source lines for alignment
		//
		// This map stores the pre-computed preview lines for each source line in the block.
		// Key: index into blockResults (0-based)
		// Value: preview lines for that source line (usually 1 line, but could be multiple if wrapped)
		var textBlockPreviewCache map[int][]string

		if !isCalcBlock && renderMarkdown != nil {
			// For text blocks, we need to handle two cases:
			// 1. Multi-line constructs (ordered lists) need block-level rendering
			// 2. Single-line content (headings, paragraphs) should render per-line for proper wrapping
			//
			// Strategy: Detect ordered lists and render them as a complete block.
			// For other content, render per-line to maintain wrapping alignment.

			textBlockPreviewCache = make(map[int][]string, len(blockResults))

			// Detect if this block contains an ordered list
			// Check all lines in the block, not just the first one
			isOrderedList := false
			for _, lineResult := range blockResults {
				line := lineResult.Source
				// Ordered list pattern: "1. " or "1) " at start of line
				if len(line) >= 3 && line[0] >= '0' && line[0] <= '9' {
					if line[1] == '.' && line[2] == ' ' {
						isOrderedList = true
						break
					} else if line[1] == ')' && line[2] == ' ' {
						isOrderedList = true
						break
					}
				}
			}

			if isOrderedList {
				// Render entire block together to preserve list numbering
				var blockText strings.Builder
				for i, lineResult := range blockResults {
					if i > 0 {
						blockText.WriteString("\n")
					}
					lineText := lineResult.Source
					// If user is typing on this line, use live editBuf instead
					if input.EditBuf != "" && lineResult.LineNum == input.EditBufLine {
						lineText = input.EditBuf
					}
					blockText.WriteString(lineText)
				}

				// Render the complete block
				allRenderedLines := renderMarkdown(blockText.String(), input.PreviewWidth)

				// Distribute rendered lines across source lines to maintain 1:1 alignment
				// Strategy: distribute as evenly as possible
				numSourceLines := len(blockResults)
				numRenderedLines := len(allRenderedLines)

				// Calculate how many rendered lines per source line (base amount)
				linesPerSource := numRenderedLines / numSourceLines
				remainder := numRenderedLines % numSourceLines

				renderedIdx := 0
				for i := 0; i < numSourceLines; i++ {
					// Some source lines get an extra line to account for remainder
					count := linesPerSource
					if i < remainder {
						count++
					}

					// Assign rendered lines to this source line
					var sourceLinesForThis []string
					for j := 0; j < count && renderedIdx < numRenderedLines; j++ {
						sourceLinesForThis = append(sourceLinesForThis, allRenderedLines[renderedIdx])
						renderedIdx++
					}

					// If no lines assigned, give it an empty line to maintain structure
					if len(sourceLinesForThis) == 0 {
						sourceLinesForThis = []string{""}
					}

					textBlockPreviewCache[i] = sourceLinesForThis
				}
			} else {
				// Render each line individually for proper wrapping alignment
				for blockLineIdx, lineResult := range blockResults {
					lineText := lineResult.Source
					// If user is typing on this line, use live editBuf instead
					if input.EditBuf != "" && lineResult.LineNum == input.EditBufLine {
						lineText = input.EditBuf
					}

					// Render this line individually
					// renderMarkdown returns all visual lines for this source line
					renderedLines := renderMarkdown(lineText, input.PreviewWidth)

					// Store all rendered lines for this source line
					// This preserves wrapping: if a heading wraps to 2 lines, we store both
					textBlockPreviewCache[blockLineIdx] = renderedLines
				}
			}
		}
		// === END CRITICAL SECTION ===

		// Now process each source line in the block to create aligned visual lines
		for blockLineIdx, lineResult := range blockResults {
			// Validate line number is in bounds
			if lineResult.LineNum >= len(input.Lines) {
				continue
			}

			// Get the actual source text for this line
			// CRITICAL: If user is typing (editBuf is active), use editBuf instead of saved content
			sourceText := input.Lines[lineResult.LineNum]
			if input.EditBuf != "" && lineResult.LineNum == input.EditBufLine {
				// User is currently typing on this line - use live editBuf for preview
				sourceText = input.EditBuf
			}
			isCursorOnThisLine := lineResult.LineNum == input.CursorLine

			// Wrap source content to fit the source pane width
			wrappedSourceLines := geometry.WrapText(sourceText, input.SourceContentWidth)

			// Determine preview content for this line
			// THREE CASES:
			// 1. CalcBlock: Use renderCalcLine to show calculation results
			// 2. TextBlock with cache: Use pre-rendered markdown from cache (ordered list fix!)
			// 3. Fallback: Render line-by-line or use plain text
			var wrappedPreviewLines []string

			if isCalcBlock && renderCalcLine != nil {
				// CASE 1: Calculation block - render with values/errors
				calcPreviewContent := renderCalcLine(lineResult, input.PreviewWidth)
				wrappedPreviewLines = wrapStyledLine(calcPreviewContent, input.PreviewWidth)

			} else if !isCalcBlock && textBlockPreviewCache != nil {
				// CASE 2: TextBlock with pre-rendered cache (THIS IS THE ORDERED LIST FIX)
				// Use the pre-computed preview lines from the cache.
				// These were computed by rendering the entire block at once,
				// which preserves multi-line markdown semantics like ordered list numbering.
				wrappedPreviewLines = textBlockPreviewCache[blockLineIdx]

			} else if renderMarkdown != nil {
				// CASE 3a: Fallback to line-by-line markdown rendering
				// This path should rarely be taken - it exists for safety.
				// Line-by-line rendering breaks ordered lists but is better than nothing.
				wrappedPreviewLines = renderMarkdown(lineResult.Source, input.PreviewWidth)

			} else {
				// CASE 3b: No markdown renderer available - use plain text
				wrappedPreviewLines = geometry.WrapText(lineResult.Source, input.PreviewWidth)
			}

			// Ensure we have at least one preview line (safety check)
			if len(wrappedPreviewLines) == 0 {
				wrappedPreviewLines = []string{""}
			}

			// === ALIGNMENT COMPUTATION ===
			// Each source line may wrap into multiple visual lines.
			// Each preview line may also wrap into multiple visual lines.
			// We need to align them so both panes have the same number of visual lines.
			//
			// Example:
			//   Source: "short" (1 visual line)
			//   Preview: "long long long long long" (wraps to 3 visual lines)
			//   Result: 3 aligned visual line pairs, with source padded with 2 empty lines
			numSourceVisualLines := len(wrappedSourceLines)
			numPreviewVisualLines := len(wrappedPreviewLines)
			numAlignedVisualLines := numSourceVisualLines
			if numPreviewVisualLines > numAlignedVisualLines {
				numAlignedVisualLines = numPreviewVisualLines
			}

			// Record mapping: document source line number -> first visual line index
			firstVisualLineIdx := len(sourceLines)
			if _, alreadyMapped := sourceToVisual[lineResult.LineNum]; !alreadyMapped {
				sourceToVisual[lineResult.LineNum] = firstVisualLineIdx
			}

			// Emit pairs of aligned visual lines (source pane + preview pane)
			for visualLineOffset := 0; visualLineOffset < numAlignedVisualLines; visualLineOffset++ {
				// Record reverse mapping: visual line index -> document source line number
				visualToSource[len(sourceLines)] = lineResult.LineNum

				// === BUILD SOURCE PANE VISUAL LINE ===
				var sourceVisualLine AlignedLine
				if visualLineOffset < numSourceVisualLines {
					// This visual line has actual source content
					sourceVisualLine = AlignedLine{
						Content:       wrappedSourceLines[visualLineOffset],
						SourceLineIdx: lineResult.LineNum,
						BlockID:       currentBlockID,
						IsCalc:        isCalcBlock,
					}
					if visualLineOffset == 0 {
						// First visual line for this source line: show line number
						sourceVisualLine.LineNum = lineResult.LineNum + 1 // 1-indexed for display
						if isCursorOnThisLine {
							sourceVisualLine.Kind = AlignedLineCursor
						} else {
							sourceVisualLine.Kind = AlignedLineNormal
						}
					} else {
						// Wrapped continuation line: no line number
						sourceVisualLine.LineNum = 0
						if isCursorOnThisLine {
							sourceVisualLine.Kind = AlignedLineCursorWrapped
						} else {
							sourceVisualLine.Kind = AlignedLineWrapped
						}
					}
				} else {
					// Padding line: preview wrapped more than source
					sourceVisualLine = AlignedLine{
						Content:       "",
						SourceLineIdx: lineResult.LineNum,
						LineNum:       0,
						Kind:          AlignedLinePadding,
						BlockID:       currentBlockID,
						IsCalc:        isCalcBlock,
					}
				}
				sourceLines = append(sourceLines, sourceVisualLine)

				// === BUILD PREVIEW PANE VISUAL LINE ===
				var previewVisualLine AlignedLine
				if visualLineOffset < numPreviewVisualLines {
					// This visual line has actual preview content
					previewVisualLine = AlignedLine{
						Content:       wrappedPreviewLines[visualLineOffset],
						SourceLineIdx: lineResult.LineNum,
						BlockID:       currentBlockID,
						IsCalc:        isCalcBlock,
					}
					if visualLineOffset == 0 {
						// First visual line: show line number
						previewVisualLine.LineNum = lineResult.LineNum + 1 // 1-indexed for display
						previewVisualLine.Kind = AlignedLineNormal
					} else {
						// Wrapped continuation line: no line number
						previewVisualLine.LineNum = 0
						previewVisualLine.Kind = AlignedLineWrapped
					}
				} else {
					// Padding line: source wrapped more than preview
					previewVisualLine = AlignedLine{
						Content:       "",
						SourceLineIdx: lineResult.LineNum,
						LineNum:       0,
						Kind:          AlignedLinePadding,
						BlockID:       currentBlockID,
						IsCalc:        isCalcBlock,
					}
				}
				previewLines = append(previewLines, previewVisualLine)
			}
		}

		// Move to next block
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
	start = scrollOffset
	if start < 0 {
		start = 0
	}
	if start >= a.TotalVisualLines {
		start = max(0, a.TotalVisualLines-1)
	}

	end = start + height
	if end > a.TotalVisualLines {
		end = a.TotalVisualLines
	}

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
