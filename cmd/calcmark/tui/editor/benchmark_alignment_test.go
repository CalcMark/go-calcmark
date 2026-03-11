package editor

import (
	"fmt"
	"strings"
	"testing"
)

// BenchmarkComputeAlignedModel_RenderedTextBlocks benchmarks the full alignment
// pipeline with pre-rendered TextBlock content — the hot path during typing in
// PreviewRendered mode.
func BenchmarkComputeAlignedModel_RenderedTextBlocks(b *testing.B) {
	// Simulate a document with 3 TextBlocks and 2 CalcBlocks (~50 lines).
	lines, results, rendered := buildMixedDocument(50)

	input := AlignedModelInput{
		Lines:              lines,
		Results:            results,
		SourceContentWidth: 60,
		PreviewWidth:       60,
		CursorLine:         10,
		PreviewMode:        PreviewRendered,
		RenderedTextBlocks: rendered,
	}

	b.ResetTimer()
	for b.Loop() {
		ComputeAlignedModel(input, mockRenderCalcLine, mockRenderMarkdown)
	}
}

// BenchmarkComputeAlignedModel_LargeDocument benchmarks alignment with a large
// document (~200 lines, 10 blocks). This is the worst case for alignment computation.
func BenchmarkComputeAlignedModel_LargeDocument(b *testing.B) {
	lines, results, rendered := buildMixedDocument(200)

	input := AlignedModelInput{
		Lines:              lines,
		Results:            results,
		SourceContentWidth: 60,
		PreviewWidth:       60,
		CursorLine:         50,
		PreviewMode:        PreviewRendered,
		RenderedTextBlocks: rendered,
	}

	b.ResetTimer()
	for b.Loop() {
		ComputeAlignedModel(input, mockRenderCalcLine, mockRenderMarkdown)
	}
}

// BenchmarkComputeAlignedModel_CacheHitPath benchmarks alignment when typing in
// a CalcBlock (TextBlock results are pre-rendered and cached, CalcBlocks use
// line-level alignment). This is the most common typing path.
func BenchmarkComputeAlignedModel_CacheHitPath(b *testing.B) {
	lines, results, rendered := buildMixedDocument(50)

	// Cursor on a CalcBlock line (not a TextBlock)
	calcLineIdx := -1
	for i, r := range results {
		if r.IsCalc {
			calcLineIdx = i
			break
		}
	}
	if calcLineIdx < 0 {
		b.Fatal("no calc lines in test document")
	}

	input := AlignedModelInput{
		Lines:              lines,
		Results:            results,
		SourceContentWidth: 60,
		PreviewWidth:       60,
		CursorLine:         calcLineIdx,
		PreviewMode:        PreviewRendered,
		RenderedTextBlocks: rendered,
		EditBuf:            "total = price * quantity + tax",
		EditBufLine:        calcLineIdx,
	}

	b.ResetTimer()
	for b.Loop() {
		ComputeAlignedModel(input, mockRenderCalcLine, mockRenderMarkdown)
	}
}

// BenchmarkRenderedBlockCache_RenderAndAlign benchmarks the combined cache
// lookup + alignment, simulating the full View() pipeline per-frame.
func BenchmarkRenderedBlockCache_RenderAndAlign(b *testing.B) {
	cache := NewRenderedBlockCache(128)

	// Pre-warm the cache with a typical TextBlock
	blockContent := strings.Split(mixedBlock20, "\n")
	_ = cache.Render("text1", blockContent, 60)

	lines, results, _ := buildMixedDocument(50)

	b.ResetTimer()
	for b.Loop() {
		// Cache lookup (should be a hit)
		rendered := map[string][]string{
			"text1": cache.Render("text1", blockContent, 60),
		}

		input := AlignedModelInput{
			Lines:              lines,
			Results:            results,
			SourceContentWidth: 60,
			PreviewWidth:       60,
			CursorLine:         10,
			PreviewMode:        PreviewRendered,
			RenderedTextBlocks: rendered,
		}
		ComputeAlignedModel(input, mockRenderCalcLine, mockRenderMarkdown)
	}
}

// buildMixedDocument creates a realistic document with alternating TextBlocks
// and CalcBlocks for benchmarking. Returns (lines, results, renderedTextBlocks).
func buildMixedDocument(totalLines int) ([]string, []LineResult, map[string][]string) {
	var lines []string
	var results []LineResult
	rendered := make(map[string][]string)

	blockNum := 0
	lineNum := 0
	isCalc := false

	for lineNum < totalLines {
		blockID := fmt.Sprintf("block%d", blockNum)
		blockSize := 10 // ~10 lines per block
		if lineNum+blockSize > totalLines {
			blockSize = totalLines - lineNum
		}

		if isCalc {
			// CalcBlock: variable assignments
			for i := range blockSize {
				varName := fmt.Sprintf("var_%d", lineNum)
				src := fmt.Sprintf("%s = %d * 100", varName, i+1)
				lines = append(lines, src)
				results = append(results, LineResult{
					LineNum: lineNum,
					Source:  src,
					BlockID: blockID,
					IsCalc:  true,
					VarName: varName,
					Value:   fmt.Sprintf("%d", (i+1)*100),
				})
				lineNum++
			}
		} else {
			// TextBlock: markdown content
			var renderedLines []string
			for i := range blockSize {
				var src string
				switch i % 5 {
				case 0:
					src = fmt.Sprintf("## Section %d", blockNum)
				case 1:
					src = "This is a paragraph with **bold** and _italic_ text."
				case 2:
					src = fmt.Sprintf("- Item %d: some list content", i)
				case 3:
					src = ""
				case 4:
					src = "More descriptive text explaining the calculations below."
				}
				lines = append(lines, src)
				results = append(results, LineResult{
					LineNum: lineNum,
					Source:  src,
					BlockID: blockID,
					IsCalc:  false,
				})
				renderedLines = append(renderedLines, fmt.Sprintf("rendered_%d_%d", blockNum, i))
				lineNum++
			}
			rendered[blockID] = renderedLines
		}

		blockNum++
		isCalc = !isCalc
	}

	return lines, results, rendered
}
