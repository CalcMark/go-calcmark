package editor

import "github.com/CalcMark/go-calcmark/cmd/calcmark/tui/geometry"

// LineModel computes the visual line layout for source and preview panes.
// This is a pure computation - no rendering, no side effects.
// It can be fully unit tested without any UI dependencies.
type LineModel struct {
	SourceLines  []VisualLine // Lines to display in source pane
	PreviewLines []VisualLine // Lines to display in preview pane
}

// VisualLine represents a single visual line in either pane.
// Multiple VisualLines may map to the same source line (wrapping).
type VisualLine struct {
	Content       string // Text content for this visual line
	SourceLineIdx int    // Original source line index (-1 for padding)
	LineNumber    int    // Line number to display (0 = don't show)
	Kind          LineKind
}

// LineKind describes what type of visual line this is.
type LineKind int

const (
	LineKindNormal    LineKind = iota // Regular source line
	LineKindWrapped                   // Continuation of wrapped line
	LineKindPadding                   // Padding for alignment
	LineKindEditFirst                 // First line of edit buffer
	LineKindEditWrap                  // Wrapped continuation of edit buffer
)

// LineModelInput contains all inputs needed to compute the line model.
// This makes the computation a pure function of its inputs.
type LineModelInput struct {
	// Source content
	Lines       []string // Document lines
	BlockIDs    []string // Block ID for each line
	IsCalcLine  []bool   // Whether each line is a calculation
	LineResults []string // Rendered result for each line (preview content)

	// Dimensions
	SourceWidth  int // Width available for source content (after line numbers)
	PreviewWidth int // Width available for preview content

	// Edit state
	EditMode      bool   // Whether we're in edit mode
	EditLineIdx   int    // Which line is being edited
	EditBuffer    string // Current edit buffer content
	EditCursorCol int    // Cursor position in edit buffer
}

// ComputeLineModel computes the visual line layout from inputs.
// This is a pure function - same inputs always produce same outputs.
func ComputeLineModel(input LineModelInput) LineModel {
	var sourceLines []VisualLine
	var previewLines []VisualLine

	// Group lines by block for alignment
	blocks := groupLinesByBlock(input)

	for _, block := range blocks {
		blockSource, blockPreview := computeBlockLines(block, input)

		// Align block heights by padding the shorter one
		sourceCount := len(blockSource)
		previewCount := len(blockPreview)

		if sourceCount < previewCount {
			// Pad source
			for i := 0; i < previewCount-sourceCount; i++ {
				blockSource = append(blockSource, VisualLine{
					Content:       "",
					SourceLineIdx: block.lastLineIdx,
					LineNumber:    0,
					Kind:          LineKindPadding,
				})
			}
		} else if previewCount < sourceCount {
			// Pad preview
			for i := 0; i < sourceCount-previewCount; i++ {
				blockPreview = append(blockPreview, VisualLine{
					Content:       "",
					SourceLineIdx: -1,
					LineNumber:    0,
					Kind:          LineKindPadding,
				})
			}
		}

		sourceLines = append(sourceLines, blockSource...)
		previewLines = append(previewLines, blockPreview...)
	}

	return LineModel{
		SourceLines:  sourceLines,
		PreviewLines: previewLines,
	}
}

// blockInfo groups consecutive lines with the same block ID.
type blockInfo struct {
	blockID     string
	lineIndices []int // Original line indices in this block
	lastLineIdx int
}

func groupLinesByBlock(input LineModelInput) []blockInfo {
	var blocks []blockInfo

	if len(input.Lines) == 0 {
		return blocks
	}

	currentBlock := blockInfo{
		blockID:     input.BlockIDs[0],
		lineIndices: []int{0},
		lastLineIdx: 0,
	}

	for i := 1; i < len(input.Lines); i++ {
		if input.BlockIDs[i] == currentBlock.blockID {
			currentBlock.lineIndices = append(currentBlock.lineIndices, i)
			currentBlock.lastLineIdx = i
		} else {
			blocks = append(blocks, currentBlock)
			currentBlock = blockInfo{
				blockID:     input.BlockIDs[i],
				lineIndices: []int{i},
				lastLineIdx: i,
			}
		}
	}
	blocks = append(blocks, currentBlock)

	return blocks
}

func computeBlockLines(block blockInfo, input LineModelInput) ([]VisualLine, []VisualLine) {
	var sourceLines []VisualLine
	var previewLines []VisualLine

	for _, lineIdx := range block.lineIndices {
		// Compute source lines for this line
		srcLines := computeSourceLinesForLine(lineIdx, input)
		sourceLines = append(sourceLines, srcLines...)

		// Compute preview lines for this line
		prvLines := computePreviewLinesForLine(lineIdx, input)
		previewLines = append(previewLines, prvLines...)
	}

	return sourceLines, previewLines
}

func computeSourceLinesForLine(lineIdx int, input LineModelInput) []VisualLine {
	var result []VisualLine

	// Determine content to wrap
	var content string
	isEditLine := input.EditMode && lineIdx == input.EditLineIdx

	if isEditLine {
		content = input.EditBuffer
	} else {
		content = input.Lines[lineIdx]
	}

	// Wrap the content
	wrapped := geometry.WrapText(content, input.SourceWidth)

	for i, segment := range wrapped {
		vl := VisualLine{
			Content:       segment,
			SourceLineIdx: lineIdx,
		}

		if i == 0 {
			vl.LineNumber = lineIdx + 1
			if isEditLine {
				vl.Kind = LineKindEditFirst
			} else {
				vl.Kind = LineKindNormal
			}
		} else {
			vl.LineNumber = 0
			if isEditLine {
				vl.Kind = LineKindEditWrap
			} else {
				vl.Kind = LineKindWrapped
			}
		}

		result = append(result, vl)
	}

	return result
}

func computePreviewLinesForLine(lineIdx int, input LineModelInput) []VisualLine {
	var result []VisualLine

	// Get the rendered result for this line
	content := ""
	if lineIdx < len(input.LineResults) {
		content = input.LineResults[lineIdx]
	}

	if content == "" {
		// Empty preview line
		return []VisualLine{{
			Content:       "",
			SourceLineIdx: lineIdx,
			LineNumber:    0,
			Kind:          LineKindNormal,
		}}
	}

	// Wrap preview content - NEVER truncate
	wrapped := geometry.WrapText(content, input.PreviewWidth)

	for _, segment := range wrapped {
		result = append(result, VisualLine{
			Content:       segment,
			SourceLineIdx: lineIdx,
			LineNumber:    0,
			Kind:          LineKindNormal,
		})
	}

	return result
}
