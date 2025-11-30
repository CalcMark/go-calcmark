package document

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/CalcMark/go-calcmark/spec/parser"
	"github.com/CalcMark/go-calcmark/spec/semantic"
	"github.com/CalcMark/go-calcmark/spec/types"
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

	// Apply frontmatter (exchange rates, globals) to environment before evaluation
	if err := doc.ApplyFrontmatter(e.env); err != nil {
		return fmt.Errorf("frontmatter: %w", err)
	}

	// Evaluate blocks in document order (top-down)
	for _, node := range doc.GetBlocks() {
		switch block := node.Block.(type) {
		case *document.CalcBlock:
			// Pass doc so @global/@exchange update frontmatter
			err := e.evaluateCalcBlockWithDoc(node.ID, block, doc)
			if err != nil {
				return err
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

// GetEnvironment returns the interpreter's environment.
// Useful for accessing current variable values.
func (e *Evaluator) GetEnvironment() *interpreter.Environment {
	return e.env
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

// EvaluateBlock evaluates a single block and re-evaluates all blocks for reactive semantics.
// This is the simple API for REPL usage where full consistency is needed.
//
// For WYSIWYG editors that need surgical updates, use EvaluateAffectedBlocks instead
// with the AffectedBlockIDs from InsertBlock/ReplaceBlockSource.
func (e *Evaluator) EvaluateBlock(doc *document.Document, blockID string) error {
	// Find the block to verify it exists
	_, ok := doc.GetBlock(blockID)
	if !ok {
		return fmt.Errorf("block not found: %s", blockID)
	}

	// PASS 1: Evaluate all blocks to collect final variable values
	// This builds the environment with all variable assignments
	e.env = interpreter.NewEnvironment()

	for _, node := range doc.GetBlocks() {
		if cb, ok := node.Block.(*document.CalcBlock); ok {
			// Evaluate to collect variable values (pass doc for frontmatter updates)
			_ = e.evaluateCalcBlockWithDoc(node.ID, cb, doc)
		}
	}

	// Snapshot the final environment state
	// This has the LAST value for each variable (e.g., a=20 from block 3)
	finalEnv := e.env.Clone()

	// Find which block has the LAST definition of each variable
	// These are the "authoritative" assignments that shouldn't be overwritten
	lastDefBlock := make(map[string]string) // varName -> blockID
	for _, node := range doc.GetBlocks() {
		if cb, ok := node.Block.(*document.CalcBlock); ok {
			for _, varName := range cb.Variables() {
				lastDefBlock[varName] = node.ID
			}
		}
	}

	// PASS 2: Re-evaluate each block using the final variable values
	// Only allow a block to SET a variable if it's the last definition
	// This ensures earlier definitions (like a=10) don't overwrite later ones (a=20)
	reactiveEnv := finalEnv.Clone()

	for _, node := range doc.GetBlocks() {
		if cb, ok := node.Block.(*document.CalcBlock); ok {
			err := e.evaluateCalcBlockSelective(node.ID, cb, reactiveEnv, lastDefBlock)
			if err != nil {
				return err
			}
		}
	}

	// Store reactive environment for variable lookups
	e.env = reactiveEnv

	return nil
}

// EvaluateAffectedBlocks efficiently re-evaluates only the specified blocks.
// This is the surgical API for WYSIWYG editors that need minimal updates.
//
// Usage:
//
//	result, _ := doc.InsertBlock(afterID, BlockCalculation, source)
//	orderedBlocks := doc.GetBlocksInDependencyOrder(result.AffectedBlockIDs)
//	eval.EvaluateAffectedBlocks(doc, orderedBlocks)
//
// The blocks should be in dependency order (use GetBlocksInDependencyOrder).
// The environment is NOT reset - it maintains accumulated state from previous evaluations.
func (e *Evaluator) EvaluateAffectedBlocks(doc *document.Document, blockIDs []string) error {
	for _, blockID := range blockIDs {
		node, ok := doc.GetBlock(blockID)
		if !ok {
			continue // Skip missing blocks
		}

		if cb, ok := node.Block.(*document.CalcBlock); ok {
			err := e.evaluateCalcBlock(blockID, cb)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// evaluateCalcBlockSelective evaluates a CalcBlock, but only updates the environment
// for variables where this block is the authoritative (last) definition.
// This ensures reactive semantics: later assignments "win" over earlier ones.
func (e *Evaluator) evaluateCalcBlockSelective(blockID string, block *document.CalcBlock, env *interpreter.Environment, lastDefBlock map[string]string) error {
	// Clear previous errors and diagnostics
	block.SetError(nil)
	block.ClearDiagnostics()

	// 1. Parse source to AST
	source := strings.Join(block.Source(), "\n")
	if !strings.HasSuffix(source, "\n") {
		source += "\n"
	}

	nodes, err := parser.Parse(source)
	if err != nil {
		block.SetError(err)
		// Convert ParseError to Diagnostic for position info
		if pe, ok := err.(*parser.ParseError); ok && pe.Line > 0 {
			block.AddDiagnostic(document.Diagnostic{
				Severity: "error",
				Code:     "parse_error",
				Message:  pe.Message,
				Line:     pe.Line,
				Column:   pe.Column,
			})
		}
		return err
	}

	// Store parsed AST
	block.SetStatements(nodes)

	// 2. Semantic check with the provided environment
	checker := semantic.NewChecker()
	for varName, value := range env.GetAllVariables() {
		checker.GetEnvironment().Set(varName, value)
	}

	diagnostics := checker.Check(nodes)
	for _, diag := range diagnostics {
		if diag.Severity == semantic.Error {
			// Store structured diagnostic with position info
			blockDiag := document.Diagnostic{
				Severity: "error",
				Code:     diag.Code,
				Message:  diag.Message,
			}
			if diag.Range != nil {
				blockDiag.Line = diag.Range.Start.Line
				blockDiag.Column = diag.Range.Start.Column
			}
			block.AddDiagnostic(blockDiag)

			// Also set legacy error for backwards compatibility
			err := fmt.Errorf("%s: %s", diag.Code, diag.Message)
			block.SetError(err)
			return err
		}
	}

	// 3. Interpret with a COPY of the environment
	// We'll selectively copy back only authoritative assignments
	evalEnv := env.Clone()
	interp := interpreter.NewInterpreterWithEnv(evalEnv)
	results, err := interp.Eval(nodes)
	if err != nil {
		block.SetError(err)
		return err
	}

	// 4. Store results
	block.SetResults(results)
	if len(results) > 0 {
		block.SetLastValue(results[len(results)-1])
	}

	// 5. Only update env for variables where this block is the last definition
	// This prevents earlier blocks (a=10) from overwriting later ones (a=20)
	for _, varName := range block.Variables() {
		if lastDefBlock[varName] == blockID {
			if val, ok := evalEnv.Get(varName); ok {
				env.Set(varName, val)
			}
		}
	}

	block.SetDirty(false)
	return nil
}

// evaluateCalcBlock evaluates a single CalcBlock.
// Steps: parse → semantic check → interpret → store results
func (e *Evaluator) evaluateCalcBlock(blockID string, block *document.CalcBlock) error {
	return e.evaluateCalcBlockWithDoc(blockID, block, nil)
}

// evaluateCalcBlockWithDoc evaluates a CalcBlock and optionally updates document frontmatter.
// If doc is non-nil, frontmatter assignments (@global, @exchange) update the document.
func (e *Evaluator) evaluateCalcBlockWithDoc(blockID string, block *document.CalcBlock, doc *document.Document) error {
	// Clear previous errors and diagnostics
	block.SetError(nil)
	block.ClearDiagnostics()

	// 1. Parse source to AST
	source := strings.Join(block.Source(), "\n")
	if !strings.HasSuffix(source, "\n") {
		source += "\n"
	}

	nodes, err := parser.Parse(source)
	if err != nil {
		block.SetError(err)
		// Convert ParseError to Diagnostic for position info
		if pe, ok := err.(*parser.ParseError); ok && pe.Line > 0 {
			block.AddDiagnostic(document.Diagnostic{
				Severity: "error",
				Code:     "parse_error",
				Message:  pe.Message,
				Line:     pe.Line,
				Column:   pe.Column,
			})
		}
		return err
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
			// Store structured diagnostic with position info
			blockDiag := document.Diagnostic{
				Severity: "error",
				Code:     diag.Code,
				Message:  diag.Message,
			}
			if diag.Range != nil {
				blockDiag.Line = diag.Range.Start.Line
				blockDiag.Column = diag.Range.Start.Column
			}
			block.AddDiagnostic(blockDiag)

			// Also set legacy error for backwards compatibility
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
		return err
	}

	// 4. Store all results (for inline display) and last result
	block.SetResults(results)
	if len(results) > 0 {
		block.SetLastValue(results[len(results)-1])
	}

	// 5. Update document frontmatter for @global and @exchange assignments
	if doc != nil {
		e.updateFrontmatterFromNodes(doc, nodes, results)
	}

	// Mark as clean (evaluated successfully)
	block.SetDirty(false)

	return nil
}

// updateFrontmatterFromNodes updates the document's frontmatter based on
// FrontmatterAssignment nodes that were just evaluated.
func (e *Evaluator) updateFrontmatterFromNodes(doc *document.Document, nodes []ast.Node, results []types.Type) {
	resultIdx := 0
	for _, node := range nodes {
		fmAssign, ok := node.(*ast.FrontmatterAssignment)
		if !ok {
			// Non-frontmatter nodes also produce results
			if resultIdx < len(results) {
				resultIdx++
			}
			continue
		}

		// Get the result value for this assignment
		if resultIdx >= len(results) {
			continue
		}
		result := results[resultIdx]
		resultIdx++

		fm := doc.EnsureFrontmatter()

		switch fmAssign.Namespace {
		case "global":
			// Store the result's string representation as the expression
			fm.SetGlobal(fmAssign.Property, result.String())

		case "exchange":
			// Store the exchange rate
			if dec, err := types.ToDecimal(result); err == nil {
				fm.SetExchangeRate(fmAssign.Property, dec)
			}
		}
	}
}
