package document

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/parser"
	"github.com/CalcMark/go-calcmark/spec/semantic"
)

// Evaluate evaluates all blocks in the document in dependency order.
// CalcBlocks are evaluated top-down with accumulated environment.
// TextBlocks are skipped (no evaluation needed).
//
// Returns error if any block has parse/semantic/evaluation errors.
func (d *Document) Evaluate() error {
	// Reset environment for clean evaluation
	d.env = interpreter.NewEnvironment()

	// Evaluate blocks in document order (top-down)
	// Dependency graph ensures proper ordering was maintained during insertion
	for _, node := range d.blocks {
		if calcBlock, ok := node.Block.(*CalcBlock); ok {
			err := d.evaluateCalcBlock(node.ID, calcBlock)
			if err != nil {
				return fmt.Errorf("block %s: %w", node.ID[:8], err)
			}
		}
		// TextBlocks don't need evaluation
	}

	return nil
}

// EvaluateBlock evaluates a single block and all blocks that depend on it.
// Used for incremental updates after ReplaceBlockSource.
//
// Returns error if evaluation fails for the block or any dependent.
func (d *Document) EvaluateBlock(blockID string) error {
	// Find the block
	node, ok := d.blockIndex[blockID]
	if !ok {
		return fmt.Errorf("block not found: %s", blockID)
	}

	// Only evaluate CalcBlocks
	if _, ok := node.Block.(*CalcBlock); !ok {
		return nil // TextBlocks don't need evaluation
	}

	// Find position of this block
	startIdx := -1
	for i, n := range d.blocks {
		if n.ID == blockID {
			startIdx = i
			break
		}
	}

	if startIdx == -1 {
		return fmt.Errorf("block not found in blocks array: %s", blockID)
	}

	// Re-evaluate from this block forward (top-down semantics)
	// All blocks after this one might be affected by the change
	for i := startIdx; i < len(d.blocks); i++ {
		node := d.blocks[i]
		if cb, ok := node.Block.(*CalcBlock); ok {
			err := d.evaluateCalcBlock(node.ID, cb)
			if err != nil {
				return fmt.Errorf("block %s: %w", node.ID[:8], err)
			}
		}
	}

	return nil
}

// evaluateCalcBlock evaluates a single CalcBlock.
// Steps: parse → semantic check → interpret → store results
func (d *Document) evaluateCalcBlock(blockID string, block *CalcBlock) error {
	// Clear previous error
	block.SetError(nil)

	// 1. Parse source to AST
	source := strings.Join(block.Source(), "\n")
	if !strings.HasSuffix(source, "\n") {
		source += "\n"
	}

	nodes, err := parser.Parse(source)
	if err != nil {
		block.SetError(err)
		return fmt.Errorf("parse error: %w", err)
	}

	// Store parsed AST
	block.SetStatements(nodes)

	// 2. Semantic check with current environment
	checker := semantic.NewChecker()

	// Pre-populate checker environment with interpreter's environment
	for varName, value := range d.env.GetAllVariables() {
		checker.GetEnvironment().Set(varName, value)
	}

	diagnostics := checker.Check(nodes)

	// Check for errors
	for _, diag := range diagnostics {
		if diag.Severity == semantic.Error {
			err := fmt.Errorf("%s: %s", diag.Code, diag.Message)
			block.SetError(err)
			return err
		}
	}

	// 3. Interpret statements with shared environment
	interp := interpreter.NewInterpreterWithEnv(d.env)

	results, err := interp.Eval(nodes)
	if err != nil {
		block.SetError(err)
		return fmt.Errorf("eval error: %w", err)
	}

	// 4. Store all results (for inline display) and last result
	block.SetResults(results)
	if len(results) > 0 {
		block.SetLastValue(results[len(results)-1])
	}

	// Mark as clean (evaluated successfully)
	block.SetDirty(false)

	return nil
}
