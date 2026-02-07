package document

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/parser"
	"github.com/CalcMark/go-calcmark/spec/semantic"
	"github.com/CalcMark/go-calcmark/spec/types"
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

	// Pre-populate checker environment with interpreter's environment,
	// but EXCLUDE variables that were PREVIOUSLY successfully evaluated in THIS block
	// to avoid false redefinition errors during incremental re-evaluation.
	// Note: Variables() may be populated by dependency analysis before evaluation,
	// so we check if the block has Results to determine if it's been evaluated before.
	hasBeenEvaluated := len(block.Results()) > 0 && !block.IsDirty()
	previouslyDefinedVars := block.Variables()

	for varName, value := range d.env.GetAllVariables() {
		// Skip variables that this block previously evaluated successfully
		if hasBeenEvaluated && containsString(previouslyDefinedVars, varName) {
			continue
		}
		checker.GetEnvironment().Set(varName, value)
	}

	diagnostics := checker.Check(nodes)

	// Convert semantic diagnostics to document diagnostics
	var docDiags []Diagnostic
	for _, diag := range diagnostics {
		// Extract line number from Range (if available)
		line := 0
		column := 0
		if diag.Range != nil {
			line = diag.Range.Start.Line
			column = diag.Range.Start.Column
		}

		// Convert severity to string
		severity := "error"
		switch diag.Severity {
		case semantic.Error:
			severity = "error"
		case semantic.Warning:
			severity = "warning"
		case semantic.Hint:
			severity = "hint"
		}

		docDiag := Diagnostic{
			BlockID:  blockID,
			Severity: severity,
			Code:     diag.Code,
			Message:  diag.Message,
			Line:     line,
			Column:   column,
		}
		docDiags = append(docDiags, docDiag)
	}

	// Store diagnostics in block
	block.SetDiagnostics(docDiags)

	// Check for errors
	for _, diag := range diagnostics {
		if diag.Severity == semantic.Error {
			err := fmt.Errorf("%s: %s", diag.Code, diag.Message)
			block.SetError(err)
			return err
		}
	}

	// 3. Interpret statements with shared environment
	// Evaluate one node at a time to track which statement fails
	interp := interpreter.NewInterpreterWithEnv(d.env)
	results := make([]types.Type, 0, len(nodes))

	for _, node := range nodes {
		nodeResults, err := interp.Eval([]ast.Node{node})
		if err != nil {
			// Create diagnostic with position from the failing node
			line := 0
			column := 0
			if r := node.GetRange(); r != nil {
				line = r.Start.Line
				column = r.Start.Column
			}

			// Add runtime error as diagnostic with position info
			runtimeDiag := Diagnostic{
				BlockID:  blockID,
				Severity: "error",
				Code:     "RUNTIME",
				Message:  err.Error(),
				Line:     line,
				Column:   column,
			}
			docDiags = append(docDiags, runtimeDiag)
			block.SetDiagnostics(docDiags)

			block.SetError(err)
			return fmt.Errorf("eval error: %w", err)
		}
		results = append(results, nodeResults...)
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

// containsString checks if a string slice contains a given string.
func containsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
