package editor

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
}

// ComputeAlignedModel computes the visual line alignment from the given inputs.
// This is a pure function - same inputs always produce same outputs.
func ComputeAlignedModel(input AlignedModelInput, renderCalcLine func(r LineResult, width int) string, renderMarkdown func(line string, width int) []string) AlignedModel {
	var sourceLines []AlignedLine
	var previewLines []AlignedLine
	sourceToVisual := make(map[int]int)
	visualToSource := make(map[int]int)

	// Process results block by block
	i := 0
	for i < len(input.Results) {
		blockID := input.Results[i].BlockID
		isCalcBlock := input.Results[i].IsCalc

		// Collect all results in this block
		var blockResults []LineResult
		for i < len(input.Results) && input.Results[i].BlockID == blockID {
			blockResults = append(blockResults, input.Results[i])
			i++
		}

		// Process each line in the block
		for _, r := range blockResults {
			if r.LineNum >= len(input.Lines) {
				continue
			}
			line := input.Lines[r.LineNum]
			isCursor := r.LineNum == input.CursorLine

			// Wrap source content
			wrappedSource := WrapText(line, input.SourceContentWidth)

			// Render and wrap preview content
			var wrappedPreview []string
			if isCalcBlock && renderCalcLine != nil {
				previewContent := renderCalcLine(r, input.PreviewWidth)
				wrappedPreview = wrapStyledLine(previewContent, input.PreviewWidth)
			} else if renderMarkdown != nil {
				wrappedPreview = renderMarkdown(r.Source, input.PreviewWidth)
			} else {
				wrappedPreview = WrapText(r.Source, input.PreviewWidth)
			}

			// Ensure we have at least one preview line
			if len(wrappedPreview) == 0 {
				wrappedPreview = []string{""}
			}

			// Determine max visual lines needed for alignment
			sourceCount := len(wrappedSource)
			previewCount := len(wrappedPreview)
			maxLines := sourceCount
			if previewCount > maxLines {
				maxLines = previewCount
			}

			// Record mapping: source line -> first visual line index
			visualIdx := len(sourceLines)
			if _, exists := sourceToVisual[r.LineNum]; !exists {
				sourceToVisual[r.LineNum] = visualIdx
			}

			// Emit visual lines (source and preview in parallel)
			for j := 0; j < maxLines; j++ {
				// Record reverse mapping
				visualToSource[len(sourceLines)] = r.LineNum

				// Build source visual line
				var sl AlignedLine
				if j < sourceCount {
					sl = AlignedLine{
						Content:       wrappedSource[j],
						SourceLineIdx: r.LineNum,
						BlockID:       blockID,
						IsCalc:        isCalcBlock,
					}
					if j == 0 {
						sl.LineNum = r.LineNum + 1
						if isCursor {
							sl.Kind = AlignedLineCursor
						} else {
							sl.Kind = AlignedLineNormal
						}
					} else {
						sl.LineNum = 0
						if isCursor {
							sl.Kind = AlignedLineCursorWrapped
						} else {
							sl.Kind = AlignedLineWrapped
						}
					}
				} else {
					// Padding line (preview wrapped more than source)
					sl = AlignedLine{
						Content:       "",
						SourceLineIdx: r.LineNum,
						LineNum:       0,
						Kind:          AlignedLinePadding,
						BlockID:       blockID,
						IsCalc:        isCalcBlock,
					}
				}
				sourceLines = append(sourceLines, sl)

				// Build preview visual line
				var pl AlignedLine
				if j < previewCount {
					pl = AlignedLine{
						Content:       wrappedPreview[j],
						SourceLineIdx: r.LineNum,
						BlockID:       blockID,
						IsCalc:        isCalcBlock,
					}
					if j == 0 {
						pl.LineNum = r.LineNum + 1
						pl.Kind = AlignedLineNormal
					} else {
						pl.LineNum = 0
						pl.Kind = AlignedLineWrapped
					}
				} else {
					// Padding line (source wrapped more than preview)
					pl = AlignedLine{
						Content:       "",
						SourceLineIdx: r.LineNum,
						LineNum:       0,
						Kind:          AlignedLinePadding,
						BlockID:       blockID,
						IsCalc:        isCalcBlock,
					}
				}
				previewLines = append(previewLines, pl)
			}
		}
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
