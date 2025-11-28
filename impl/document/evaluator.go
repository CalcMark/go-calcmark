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
	env         *interpreter.Environment
	diagnostics []BlockDiagnostic
}

// NewEvaluator creates a new document evaluator.
func NewEvaluator() *Evaluator {
	return &Evaluator{
		env: interpreter.NewEnvironment(),
	}
}

// Evaluate evaluates all blocks in the document in dependency order.
// CalcBlocks are evaluated top-down with accumulated environment.
// TextBlocks are checked for lines that look like failed calculations.
//
// Returns an error if any CalcBlock fails to evaluate.
// Use Diagnostics() to get warnings about TextBlocks with likely calculation errors.
func (e *Evaluator) Evaluate(doc *document.Document) error {
	// Reset environment and diagnostics for clean evaluation
	e.env = interpreter.NewEnvironment()
	e.diagnostics = nil

	// Evaluate blocks in document order (top-down)
	for _, node := range doc.GetBlocks() {
		switch block := node.Block.(type) {
		case *document.CalcBlock:
			err := e.evaluateCalcBlock(node.ID, block)
			if err != nil {
				return fmt.Errorf("block %s: %w", node.ID[:8], err)
			}
		case *document.TextBlock:
			// Check TextBlocks for lines that look like failed calculations
			e.checkTextBlockForLikelyCalculations(node.ID, block)
		}
	}

	return nil
}

// Diagnostics returns warnings and errors collected during evaluation.
// This includes warnings about TextBlock lines that look like failed calculations.
func (e *Evaluator) Diagnostics() []BlockDiagnostic {
	return e.diagnostics
}

// checkTextBlockForLikelyCalculations scans a TextBlock for lines that
// appear to be intended calculations but failed to parse.
func (e *Evaluator) checkTextBlockForLikelyCalculations(blockID string, block *document.TextBlock) {
	for i, line := range block.Source() {
		isLikely, parseErr := looksLikeFailedCalculation(line)
		if isLikely {
			msg := "line looks like an assignment but failed to parse"
			if parseErr != nil {
				msg = fmt.Sprintf("line looks like an assignment: %v", parseErr)
			}
			e.diagnostics = append(e.diagnostics, BlockDiagnostic{
				BlockID:  blockID,
				Line:     i + 1, // 1-indexed
				Severity: Warning,
				Code:     DiagLikelyCalculation,
				Message:  msg,
				Source:   line,
			})
		}
	}
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

	// 4. Store all results (for inline display) and last result
	block.SetResults(results)
	if len(results) > 0 {
		block.SetLastValue(results[len(results)-1])
	}

	// Mark as clean (evaluated successfully)
	block.SetDirty(false)

	return nil
}
