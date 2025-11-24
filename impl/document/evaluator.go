package document

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/CalcMark/go-calcmark/spec/parser"
	"github.com/CalcMark/go-calcmark/spec/semantic"
)

// Evaluator evaluates CalcMark documents using the interpreter.
// This lives in impl/ because it performs execution, not just validation.
type Evaluator struct {
	env *interpreter.Environment
}

// NewEvaluator creates a new document evaluator.
func NewEvaluator() *Evaluator {
	return &Evaluator{
		env: interpreter.NewEnvironment(),
	}
}

// Evaluate evaluates all blocks in the document in dependency order.
// CalcBlocks are evaluated top-down with accumulated environment.
// TextBlocks are skipped (no evaluation needed).
func (e *Evaluator) Evaluate(doc *document.Document) error {
	// Reset environment for clean evaluation
	e.env = interpreter.NewEnvironment()

	// Evaluate blocks in document order (top-down)
	for _, node := range doc.GetBlocks() {
		if calcBlock, ok := node.Block.(*document.CalcBlock); ok {
			err := e.evaluateCalcBlock(node.ID, calcBlock)
			if err != nil {
				return fmt.Errorf("block %s: %w", node.ID[:8], err)
			}
		}
	}

	return nil
}

// EvaluateBlock evaluates a single block and all blocks that depend on it.
func (e *Evaluator) EvaluateBlock(doc *document.Document, blockID string) error {
	// Find the block
	node, ok := doc.GetBlock(blockID)
	if !ok {
		return fmt.Errorf("block not found: %s", blockID)
	}

	// Only evaluate CalcBlocks
	if _, ok := node.Block.(*document.CalcBlock); !ok {
		return nil // TextBlocks don't need evaluation
	}

	// Find position of this block
	blocks := doc.GetBlocks()
	startIdx := -1
	for i, n := range blocks {
		if n.ID == blockID {
			startIdx = i
			break
		}
	}

	if startIdx == -1 {
		return fmt.Errorf("block not found in blocks array: %s", blockID)
	}

	// Re-evaluate from this block forward (top-down semantics)
	for i := startIdx; i < len(blocks); i++ {
		node := blocks[i]
		if cb, ok := node.Block.(*document.CalcBlock); ok {
			err := e.evaluateCalcBlock(node.ID, cb)
			if err != nil {
				return fmt.Errorf("block %s: %w", node.ID[:8], err)
			}
		}
	}

	return nil
}

// evaluateCalcBlock evaluates a single CalcBlock.
// Steps: parse → semantic check → interpret → store results
func (e *Evaluator) evaluateCalcBlock(blockID string, block *document.CalcBlock) error {
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
	for varName, value := range e.env.GetAllVariables() {
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
	interp := interpreter.NewInterpreterWithEnv(e.env)

	results, err := interp.Eval(nodes)
	if err != nil {
		block.SetError(err)
		return fmt.Errorf("eval error: %w", err)
	}

	// 4. Store last result
	if len(results) > 0 {
		block.SetLastValue(results[len(results)-1])
	}

	// Mark as clean (evaluated successfully)
	block.SetDirty(false)

	return nil
}
