// Package document provides the CalcMark document model with blocks.
//
// This package implements a higher-level document abstraction that manages
// mixed markdown and calculation content as a structured collection of blocks.
//
// # Architecture
//
// Key components:
//  1. Block types (CalcBlock, TextBlock)
//  2. Document structure with block management
//  3. Dependency tracking between calc blocks
//  4. Incremental evaluation for editors
//
// # Block Types
//
// CalcBlock: calculation code
//
//	block := &CalcBlock{}
//	block.AppendLine("x = 5")
//	block.AppendLine("y = x + 3")
//
// TextBlock: markdown or prose
//
//	block := &TextBlock{}
//	block.AppendLine("# My Document")
//	block.AppendLine("Some text here")
//
// # Document Model
//
// Documents manage blocks in order:
//
//	doc := document.NewDocument()
//	calcBlock := &CalcBlock{}
//	calcBlock.AppendLine("total = 100")
//	doc.AppendBlock(calcBlock)
//
//	textBlock := &TextBlock{}
//	textBlock.AppendLine("# Results")
//	doc.AppendBlock(textBlock)
//
// # Incremental Evaluation
//
// The document model supports incremental updates for interactive editors:
//
//	// Initial evaluation
//	doc.Evaluate()
//
//	// User edits a block
//	doc.ReplaceBlockSource(blockID, []string{"total = 200"})
//
//	// Re-evaluate only affected blocks
//	doc.EvaluateBlock(blockID)
//
// # Dependency Tracking
//
// The document tracks dependencies between blocks to enable smart
// re-evaluation. When a block changes, only dependent blocks are
// re-evaluated.
//
// # Line Detection
//
// Automatic detection of calculation vs markdown:
//
//	detector := document.NewDetector()
//	blockType := detector.DetectLine("x = 5")  // CalcBlock
//	blockType = detector.DetectLine("# Title") // TextBlock
//
// # Use Cases
//
// This package is designed for:
//   - Interactive markdown editors with live calculation
//   - Document processing pipelines
//   - Static site generators with embedded calculations
//   - Notebook-style interfaces
//
// # Performance
//
// Document operations are optimized for incremental updates.
// Evaluation complexity is O(affected blocks) not O(total blocks).
package document
