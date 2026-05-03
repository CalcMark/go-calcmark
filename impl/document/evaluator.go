package document

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/CalcMark/go-calcmark/v2/format/display"
	"github.com/CalcMark/go-calcmark/v2/impl/interpreter"
	"github.com/CalcMark/go-calcmark/v2/spec/ast"
	"github.com/CalcMark/go-calcmark/v2/spec/document"
	"github.com/CalcMark/go-calcmark/v2/spec/parser"
	"github.com/CalcMark/go-calcmark/v2/spec/semantic"
	"github.com/CalcMark/go-calcmark/v2/spec/transform"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
	"github.com/shopspring/decimal"
)

// ErrPartialEvaluation is returned when evaluation continued past errors.
// Callers can check errors.Is(err, ErrPartialEvaluation) to distinguish
// partial evaluation (some blocks failed) from total failure.
var ErrPartialEvaluation = errors.New("partial evaluation: one or more blocks had errors")

// Diagnostic codes for error recovery.
const (
	diagCodeCascadingError = "cascading_error"
)

// NewDirectiveResolver creates an interpreter.DirectiveResolver that adapts
// *document.Frontmatter for @directive resolution. Used by both the document
// evaluator and the top-level Eval API.
func NewDirectiveResolver(fm *document.Frontmatter) interpreter.DirectiveResolver {
	return &frontmatterDirectiveResolver{fm: fm}
}

// toFYLabelMode bridges the document layer's CalendarYearOffset enum
// to the types layer's FYLabelMode. Kept narrow so the document
// package doesn't import the types package's mode constants directly
// at every call site.
func toFYLabelMode(off document.CalendarYearOffset) types.FYLabelMode {
	if off == document.CalendarYearOffsetAfter {
		return types.FYLabelByStartYear
	}
	return types.FYLabelByEndYear
}

type frontmatterDirectiveResolver struct {
	fm *document.Frontmatter
}

func (r *frontmatterDirectiveResolver) ScaleFactor() (decimal.Decimal, bool) {
	if r.fm == nil || r.fm.Scale == nil {
		return decimal.Zero, false
	}
	return r.fm.Scale.Factor, true
}

func (r *frontmatterDirectiveResolver) ResolveGlobal(name string) (types.Type, bool, error) {
	if r.fm == nil || r.fm.Globals == nil {
		return nil, false, nil
	}
	exprStr, ok := r.fm.Globals[name]
	if !ok {
		return nil, false, nil
	}
	parsed, err := document.ParseGlobals(map[string]string{name: exprStr})
	if err != nil {
		return nil, false, err
	}
	return parsed.Values[name], true, nil
}

// Evaluator evaluates CalcMark documents using the interpreter.
// This lives in impl/ because it performs execution, not just validation.
type Evaluator struct {
	env              *interpreter.Environment
	diagnostics      []BlockDiagnostic
	displayFormatter *display.Formatter
}

// NewEvaluator creates a new document evaluator.
func NewEvaluator() *Evaluator {
	return &Evaluator{
		env: interpreter.NewEnvironment(),
	}
}

// SetDisplayFormatter sets the display formatter used for {{var}} interpolation
// in TextBlocks. If not set, a default en-US formatter is used.
func (e *Evaluator) SetDisplayFormatter(df display.Formatter) {
	e.displayFormatter = &df
}

// getDisplayFormatter returns the configured display formatter or a default.
func (e *Evaluator) getDisplayFormatter() display.Formatter {
	if e.displayFormatter != nil {
		return *e.displayFormatter
	}
	return display.DefaultFormatter()
}

// GetDisplayFormatter returns the evaluator's display formatter for use by output formatters.
// After Evaluate(), this includes measurement conventions wired from frontmatter.
func (e *Evaluator) GetDisplayFormatter() display.Formatter {
	return e.getDisplayFormatter()
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

	// Wire measurement conventions into the display formatter for strict annotation.
	// Default: strict=true (annotate bare ambiguous units like "oz" → "us oz").
	if fm := doc.GetFrontmatter(); fm != nil && fm.Measurement != nil {
		strict := true // default
		if fm.Measurement.Strict != nil {
			strict = *fm.Measurement.Strict
		}
		df := e.getDisplayFormatter()
		df.SetMeasurement(&fm.Measurement.MeasurementConfig, strict)
		e.displayFormatter = &df
	}

	// Evaluate blocks in document order (top-down).
	// Continue past block errors to collect all diagnostics across the document.
	var hasErrors bool
	for _, node := range doc.GetBlocks() {
		switch block := node.Block.(type) {
		case *document.CalcBlock:
			// Pass doc so @global/@exchange update frontmatter
			if err := e.evaluateCalcBlockWithDoc(node.ID, block, doc); err != nil {
				hasErrors = true
			}
		case *document.TextBlock:
			// Check TextBlocks for lines that look like failed calculations
			e.checkTextBlockForLikelyCalculations(node.ID, block)
		}
	}

	// Apply scale/convert_to transforms to all block results
	applyTransforms(doc, nil)

	// Resolve {{var}} tags in TextBlocks against the final environment
	interpolateTextBlocks(doc, e.env.GetAllVariables(), e.getDisplayFormatter())

	if hasErrors {
		return ErrPartialEvaluation
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
// Diagnostics are stored both on the TextBlock (for TUI/LSP consumption)
// and on the evaluator (for backwards compatibility with evaluator.Diagnostics()).
func (e *Evaluator) checkTextBlockForLikelyCalculations(blockID string, block *document.TextBlock) {
	block.ClearDiagnostics()
	for i, line := range block.Source() {
		isLikely, parseErr := looksLikeFailedCalculation(line)
		if isLikely {
			msg, detailed := reservedKeywordDiagnostic(line)
			if msg == "" {
				msg = "line looks like an assignment but failed to parse"
				if parseErr != nil {
					msg = fmt.Sprintf("line looks like an assignment: %v", parseErr)
				}
			}
			diag := BlockDiagnostic{
				BlockID:  blockID,
				Line:     i + 1, // 1-indexed
				Severity: Warning,
				Code:     DiagLikelyCalculation,
				Message:  msg,
				Source:   line,
			}
			e.diagnostics = append(e.diagnostics, diag)
			block.AddDiagnostic(document.Diagnostic{
				BlockID:  blockID,
				Severity: "warning",
				Code:     DiagLikelyCalculation,
				Message:  msg,
				Detailed: detailed,
				Line:     i + 1,
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

	// PASS 1: Evaluate all blocks to collect final variable values.
	// This builds the environment with all variable assignments.
	// We track defined variables across all blocks to detect redefinitions.
	// Continue past block errors to collect all diagnostics.
	e.env = interpreter.NewEnvironment()
	allDefinedVars := make(map[string]bool)
	nonRecoverableBlocks := make(map[string]bool) // blocks with parse/redefinition errors (skip in pass 2)
	var hasErrors bool

	for _, node := range doc.GetBlocks() {
		if cb, ok := node.Block.(*document.CalcBlock); ok {
			// Before evaluation, check if this block would redefine any variables
			// Parse to extract variable assignments
			source := strings.Join(cb.Source(), "\n")
			if !strings.HasSuffix(source, "\n") {
				source += "\n"
			}
			nodes, parseErr := parser.Parse(source)
			if parseErr != nil {
				cb.SetError(parseErr)
				// Add diagnostic but continue to next block
				if pe, ok := parseErr.(*parser.ParseError); ok && pe.Line > 0 {
					var lineOff int
					if doc != nil {
						lineOff = blockLineOffset(doc, node.ID)
					}
					cb.AddDiagnostic(document.Diagnostic{
						Severity: "error",
						Code:     "parse_error",
						Message:  pe.Message,
						Line:     pe.Line,
						Column:   pe.Column,
						DocLine:  pe.Line + lineOff,
					})
				}
				hasErrors = true
				nonRecoverableBlocks[node.ID] = true
				continue
			}

			// Extract variables that THIS execution will define
			currentlyDefining := make([]string, 0)
			for _, astNode := range nodes {
				if assign, ok := astNode.(*ast.Assignment); ok {
					currentlyDefining = append(currentlyDefining, assign.Name)
				}
			}

			// Check for redefinitions: if a variable is in allDefinedVars
			// and this block hasn't been successfully evaluated yet (dirty or no results),
			// it's a NEW definition that conflicts with a definition in another block.
			// If the block HAS been evaluated (has results), we allow re-evaluation of
			// the same variables it defined before.
			hasBeenEvaluated := len(cb.Results()) > 0 && !cb.IsDirty()
			previouslyDefined := cb.Variables()

			blockHasRedefError := false
			for _, varName := range currentlyDefining {
				// Skip if this block previously defined this variable
				if hasBeenEvaluated && slices.Contains(previouslyDefined, varName) {
					continue
				}
				// Check for conflict with other blocks
				if allDefinedVars[varName] {
					redefErr := fmt.Errorf("variable_redefinition: variable '%s' is already defined", varName)
					cb.SetError(redefErr)
					cb.AddDiagnostic(document.Diagnostic{
						Severity: "error",
						Code:     "variable_redefinition",
						Message:  redefErr.Error(),
					})
					// Mark only the conflicting variable as errored;
					// non-conflicting variables in this block will be evaluated
					// normally if the block proceeds (or marked errored individually
					// if evaluation fails for other reasons).
					e.env.SetError(varName, redefErr)
					hasErrors = true
					blockHasRedefError = true
					break
				}
			}
			if blockHasRedefError {
				nonRecoverableBlocks[node.ID] = true
				// Still track variables as defined for downstream redefinition checks
				for _, varName := range currentlyDefining {
					allDefinedVars[varName] = true
				}
				continue
			}

			// Now evaluate the block
			if err := e.evaluateCalcBlockWithDoc(node.ID, cb, doc); err != nil {
				hasErrors = true
			}

			// Mark variables as defined (includes errored vars from evaluation)
			for _, varName := range cb.Variables() {
				allDefinedVars[varName] = true
			}
			// Also mark variables extracted from AST (for blocks that partially failed)
			for _, varName := range currentlyDefining {
				allDefinedVars[varName] = true
			}
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

	// PASS 2: Re-evaluate each block using the final variable values.
	// Only allow a block to SET a variable if it's the last definition.
	// This ensures earlier definitions (like a=10) don't overwrite later ones (a=20).
	// Continue past block errors to collect all diagnostics.
	reactiveEnv := finalEnv.Clone()

	for _, node := range doc.GetBlocks() {
		if cb, ok := node.Block.(*document.CalcBlock); ok {
			// Skip blocks with non-recoverable errors from pass 1 (parse/redefinition).
			// These cannot succeed in pass 2 regardless of the environment state.
			// Runtime errors (e.g., forward references that failed in pass 1) should be
			// retried in pass 2 since the reactive env now has all variable values.
			if nonRecoverableBlocks[node.ID] {
				continue
			}
			if err := e.evaluateCalcBlockSelective(node.ID, cb, reactiveEnv, lastDefBlock, doc); err != nil {
				hasErrors = true
			}
		}
	}

	// Store reactive environment for variable lookups
	e.env = reactiveEnv

	// Apply scale/convert_to transforms to all block results
	applyTransforms(doc, nil)

	// Resolve {{var}} tags in TextBlocks against the final environment
	interpolateTextBlocks(doc, e.env.GetAllVariables(), e.getDisplayFormatter())

	if hasErrors {
		return ErrPartialEvaluation
	}
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
	var hasErrors bool
	for _, blockID := range blockIDs {
		node, ok := doc.GetBlock(blockID)
		if !ok {
			continue // Skip missing blocks
		}

		if cb, ok := node.Block.(*document.CalcBlock); ok {
			if err := e.evaluateCalcBlock(blockID, cb); err != nil {
				hasErrors = true
			}
		}
	}

	// Only transform affected blocks — other blocks already hold
	// transformed results from the previous full evaluation.
	applyTransforms(doc, blockIDs)

	// Re-run interpolation on all TextBlocks — variable values may have changed.
	interpolateTextBlocks(doc, e.env.GetAllVariables(), e.getDisplayFormatter())

	if hasErrors {
		return ErrPartialEvaluation
	}
	return nil
}

// evaluateCalcBlockSelective evaluates a CalcBlock, but only updates the environment
// for variables where this block is the authoritative (last) definition.
// This ensures reactive semantics: later assignments "win" over earlier ones.
func (e *Evaluator) evaluateCalcBlockSelective(blockID string, block *document.CalcBlock, env *interpreter.Environment, lastDefBlock map[string]string, doc *document.Document) error {
	// Clear previous errors and diagnostics
	block.SetError(nil)
	block.ClearDiagnostics()

	// Compute line offset for document-absolute line numbers in diagnostics
	var lineOff int
	if doc != nil {
		lineOff = blockLineOffset(doc, blockID)
	}

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
				DocLine:  pe.Line + lineOff,
			})
		}
		return err
	}

	// Store parsed AST
	block.SetStatements(nodes)

	// 2. Semantic check with the provided environment
	checker := semantic.NewChecker()

	// Pre-populate checker environment with interpreter's environment,
	// but EXCLUDE variables that were PREVIOUSLY successfully evaluated in THIS block
	// to avoid false redefinition errors during incremental re-evaluation.
	// If the block's source has changed to define NEW variables, those WILL be checked.
	previouslyDefinedVars := block.Variables()
	for varName, value := range env.GetAllVariables() {
		// Skip variables that this block previously defined successfully
		if !slices.Contains(previouslyDefinedVars, varName) {
			checker.GetEnvironment().Set(varName, value)
		}
	}
	// Also register errored variables so the semantic checker treats them as "defined".
	for varName := range env.GetAllErroredVars() {
		if !slices.Contains(previouslyDefinedVars, varName) {
			if _, alreadySet := checker.GetEnvironment().Get(varName); !alreadySet {
				checker.GetEnvironment().Set(varName, nil)
			}
		}
	}

	// Provide frontmatter context for @directive validation
	if doc != nil {
		checker.SetFrontmatter(doc.GetFrontmatter())
		checker.SetLineOffset(lineOff)
	}

	diagnostics := checker.Check(nodes)

	// Collect ALL semantic errors (not just the first) so every
	// redefinition or type error in the block gets a diagnostic.
	var firstSemanticErr error
	for _, diag := range diagnostics {
		if diag.Severity == semantic.Error {
			blockDiag := document.Diagnostic{
				Severity: "error",
				Code:     diag.Code,
				Message:  diag.Message,
				Detailed: diag.Detailed,
			}
			if diag.Range != nil {
				blockDiag.Line = diag.Range.Start.Line
				blockDiag.Column = diag.Range.Start.Column
				blockDiag.DocLine = diag.Range.Start.Line + lineOff
				blockDiag.EndLine = diag.Range.End.Line + lineOff
				blockDiag.EndColumn = diag.Range.End.Column
			}
			block.AddDiagnostic(blockDiag)

			if firstSemanticErr == nil {
				if blockDiag.DocLine > 0 {
					firstSemanticErr = fmt.Errorf("line %d: %s: %s", blockDiag.DocLine, diag.Code, diag.Message)
				} else {
					firstSemanticErr = fmt.Errorf("%s: %s", diag.Code, diag.Message)
				}
			}
		}
	}
	if firstSemanticErr != nil {
		block.SetError(firstSemanticErr)
		// Mark all variables defined in this block as errored (O(n) once)
		for _, astNode := range nodes {
			if assign, ok := astNode.(*ast.Assignment); ok {
				env.SetError(assign.Name, firstSemanticErr)
			}
		}
		return firstSemanticErr
	}

	// 3. Interpret with a COPY of the environment, statement by statement.
	// Failed statements get nil placeholders to maintain 1:1 alignment with nodes.
	evalEnv := env.Clone()
	interp := interpreter.NewInterpreterWithEnv(evalEnv)
	// Translate the parser's block-relative AST line numbers to
	// doc-absolute 0-indexed lines so `definedLines` recorded in the
	// environment compares directly against the LSP's cursor position.
	interp.SetLineOffset(lineOff)
	if doc != nil && doc.GetFrontmatter() != nil {
		fm := doc.GetFrontmatter()
		interp.SetDirectiveResolver(NewDirectiveResolver(fm))
		// Wire measurement conventions (pre-interpreter: affects unit name resolution)
		if fm.Measurement != nil {
			interp.SetMeasurement(&fm.Measurement.MeasurementConfig)
		}
		// Wire fiscal year configuration for FQ/FY expressions, including
		// the optional calendar_year_offset (defaults to "before").
		if fm.FiscalYearStarts != nil {
			interp.SetFiscalCalendar(
				fm.FiscalYearStarts.Month,
				fm.FiscalYearStarts.Day,
				toFYLabelMode(fm.CalendarYearOffset),
			)
		}
	}
	results := make([]types.Type, 0, len(nodes))
	var firstErr error
	hadErrors := false

	for _, node := range nodes {
		nodeResults, evalErr := interp.Eval([]ast.Node{node})
		if evalErr != nil {
			results = append(results, nil)
			hadErrors = true
			if firstErr == nil {
				firstErr = evalErr
			}

			// Create diagnostic based on error type
			var cascErr *interpreter.CascadingError
			diag := document.Diagnostic{
				Message: evalErr.Error(),
			}
			if errors.As(evalErr, &cascErr) {
				diag.Code = diagCodeCascadingError
				diag.Severity = "hint"
				diag.Detailed = cascErr.VarName
			} else {
				diag.Code = "eval_error"
				diag.Severity = "error"
			}
			if r := node.GetRange(); r != nil && r.Start.Line > 0 {
				diag.Line = r.Start.Line
				diag.Column = r.Start.Column
				diag.DocLine = r.Start.Line + lineOff
			}
			block.AddDiagnostic(diag)

			// Mark assigned variable as errored in the cloned env
			if assign, ok := node.(*ast.Assignment); ok {
				evalEnv.SetError(assign.Name, evalErr)
			}
			continue
		}
		if len(nodeResults) > 0 {
			results = append(results, nodeResults[0])
		} else {
			results = append(results, nil) // placeholder for zero-result statements (e.g., directives)
		}
	}

	// 4. Store results (may contain nil entries for failed statements)
	block.SetResults(results)

	// SetLastValue with the last non-nil result
	for i := len(results) - 1; i >= 0; i-- {
		if results[i] != nil {
			block.SetLastValue(results[i])
			break
		}
	}

	// 5. Only update env for variables where this block is the last definition.
	// Propagate errors back to the shared environment so downstream blocks see them.
	for _, varName := range block.Variables() {
		if lastDefBlock[varName] == blockID {
			if evalErrForVar, hasErr := evalEnv.GetError(varName); hasErr {
				// Variable errored in this block — propagate error to shared env
				env.SetError(varName, evalErrForVar)
			} else if val, ok := evalEnv.Get(varName); ok {
				env.Set(varName, val)
				// Clear any previous error for this variable (it succeeded now)
				env.ClearError(varName)
			}
		}
	}

	if hadErrors {
		if firstErr != nil {
			block.SetError(firstErr)
		}
		return firstErr
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

	// Compute line offset for document-absolute line numbers in diagnostics
	var lineOff int
	if doc != nil {
		lineOff = blockLineOffset(doc, blockID)
	}

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
				DocLine:  pe.Line + lineOff,
			})
		}
		return err
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

	for varName, value := range e.env.GetAllVariables() {
		// Skip variables that this block previously evaluated successfully
		if hasBeenEvaluated && slices.Contains(previouslyDefinedVars, varName) {
			continue
		}
		checker.GetEnvironment().Set(varName, value)
	}
	// Also register errored variables so the semantic checker treats them as "defined".
	// The interpreter will detect them as errored and produce CascadingError.
	for varName := range e.env.GetAllErroredVars() {
		if hasBeenEvaluated && slices.Contains(previouslyDefinedVars, varName) {
			continue
		}
		if _, alreadySet := checker.GetEnvironment().Get(varName); !alreadySet {
			checker.GetEnvironment().Set(varName, nil)
		}
	}

	// Provide frontmatter context for @directive validation
	if doc != nil {
		checker.SetFrontmatter(doc.GetFrontmatter())
		checker.SetLineOffset(lineOff)
	}

	diagnostics := checker.Check(nodes)

	// Collect ALL semantic errors (not just the first).
	var firstSemanticErr error
	for _, diag := range diagnostics {
		if diag.Severity == semantic.Error {
			blockDiag := document.Diagnostic{
				Severity: "error",
				Code:     diag.Code,
				Message:  diag.Message,
				Detailed: diag.Detailed,
			}
			if diag.Range != nil {
				blockDiag.Line = diag.Range.Start.Line
				blockDiag.Column = diag.Range.Start.Column
				blockDiag.DocLine = diag.Range.Start.Line + lineOff
				blockDiag.EndLine = diag.Range.End.Line + lineOff
				blockDiag.EndColumn = diag.Range.End.Column
			}
			block.AddDiagnostic(blockDiag)

			if firstSemanticErr == nil {
				if blockDiag.DocLine > 0 {
					firstSemanticErr = fmt.Errorf("line %d: %s: %s", blockDiag.DocLine, diag.Code, diag.Message)
				} else {
					firstSemanticErr = fmt.Errorf("%s: %s", diag.Code, diag.Message)
				}
			}
		}
	}
	if firstSemanticErr != nil {
		block.SetError(firstSemanticErr)
		for _, astNode := range nodes {
			if assign, ok := astNode.(*ast.Assignment); ok {
				e.env.SetError(assign.Name, firstSemanticErr)
			}
		}
		return firstSemanticErr
	}

	// 3. Interpret statements with shared environment
	// Evaluate statements one by one to collect partial results even if a later statement fails.
	// Failed statements get nil placeholders to maintain 1:1 alignment with nodes.
	interp := interpreter.NewInterpreterWithEnv(e.env)
	// Translate the parser's block-relative AST line numbers to
	// doc-absolute 0-indexed lines so `definedLines` recorded in the
	// environment compares directly against the LSP's cursor position.
	interp.SetLineOffset(lineOff)
	if doc != nil && doc.GetFrontmatter() != nil {
		fm := doc.GetFrontmatter()
		interp.SetDirectiveResolver(NewDirectiveResolver(fm))
		// Wire measurement conventions (pre-interpreter: affects unit name resolution)
		if fm.Measurement != nil {
			interp.SetMeasurement(&fm.Measurement.MeasurementConfig)
		}
		// Wire fiscal year configuration for FQ/FY expressions, including
		// the optional calendar_year_offset (defaults to "before").
		if fm.FiscalYearStarts != nil {
			interp.SetFiscalCalendar(
				fm.FiscalYearStarts.Month,
				fm.FiscalYearStarts.Day,
				toFYLabelMode(fm.CalendarYearOffset),
			)
		}
	}
	results := make([]types.Type, 0, len(nodes))
	var firstErr error
	hadErrors := false

	for _, node := range nodes {
		nodeResults, err := interp.Eval([]ast.Node{node})
		if err != nil {
			// Append nil placeholder to maintain 1:1 alignment with nodes
			results = append(results, nil)
			hadErrors = true

			// Track first error for legacy block.SetError() compatibility
			if firstErr == nil {
				firstErr = err
			}

			// Create diagnostic based on error type
			var cascErr *interpreter.CascadingError
			diag := document.Diagnostic{
				Message: err.Error(),
			}
			if errors.As(err, &cascErr) {
				diag.Code = diagCodeCascadingError
				diag.Severity = "hint"
				diag.Detailed = cascErr.VarName
			} else {
				diag.Code = "eval_error"
				diag.Severity = "error"
			}

			// Use node's Range if available to get line number.
			// All AST nodes implement GetRange() via the Node interface.
			// Guard against zero-valued ranges (some nodes use &ast.Range{}).
			if r := node.GetRange(); r != nil && r.Start.Line > 0 {
				diag.Line = r.Start.Line
				diag.Column = r.Start.Column
				diag.DocLine = r.Start.Line + lineOff
			}
			block.AddDiagnostic(diag)

			// If the statement is an assignment, mark the variable as errored
			if assign, ok := node.(*ast.Assignment); ok {
				e.env.SetError(assign.Name, err)
			}

			continue
		}
		if len(nodeResults) > 0 {
			results = append(results, nodeResults[0])
		} else {
			results = append(results, nil) // placeholder for zero-result statements (e.g., directives)
		}
	}

	// Store results (may contain nil entries for failed statements)
	block.SetResults(results)

	// SetLastValue with the last non-nil result
	for i := len(results) - 1; i >= 0; i-- {
		if results[i] != nil {
			block.SetLastValue(results[i])
			break
		}
	}

	// If there were errors, set legacy error and return first error
	if hadErrors {
		// Include document-absolute line number in returned error
		if firstErr != nil {
			block.SetError(firstErr)
		}
		return firstErr
	}

	// Mark as clean (evaluated successfully — all statements passed)
	block.SetDirty(false)

	return nil
}

// blockLineOffset computes the document-absolute line offset for a block.
// This counts frontmatter lines + all source lines of preceding blocks so that
// the semantic checker can produce document-absolute line numbers in diagnostics.
//
// Explicit-segmentation parsers (notably impl/embedded.BuildDocument for
// fenced sources) set BlockNode.HostLineOffset per-block so fence delimiter
// lines that aren't represented in any block's Source() get accounted for
// correctly. Prefer it when nonzero; otherwise fall back to the flat-CM
// contiguous-block-source arithmetic, which is correct for sources where
// every host-doc line lives inside some block's Source().
func blockLineOffset(doc *document.Document, targetID string) int {
	if node, ok := doc.GetBlock(targetID); ok && node.HostLineOffset > 0 {
		return node.HostLineOffset
	}

	offset := 0

	// Count frontmatter lines (including both --- delimiters)
	if fm := doc.GetFrontmatter(); fm != nil {
		offset = fm.LineCount()
	}

	// Count source lines of all blocks preceding the target
	for _, node := range doc.GetBlocks() {
		if node.ID == targetID {
			break
		}
		offset += len(node.Block.Source())
	}

	return offset
}

// applyTransforms applies scale/convert_to frontmatter transforms to block results.
// If onlyIDs is nil, all blocks are transformed (used after full evaluation).
// If onlyIDs is non-nil, only those blocks are transformed (used by
// EvaluateAffectedBlocks to avoid double-transforming unaffected blocks).
// The interpreter environment is NOT affected — variable values stay raw.
func applyTransforms(doc *document.Document, onlyIDs []string) {
	fm := doc.GetFrontmatter()
	if fm == nil || (fm.Scale == nil && fm.ConvertTo == nil) {
		return
	}

	var idSet map[string]bool
	if len(onlyIDs) > 0 {
		idSet = make(map[string]bool, len(onlyIDs))
		for _, id := range onlyIDs {
			idSet[id] = true
		}
	}

	for _, node := range doc.GetBlocks() {
		if idSet != nil && !idSet[node.ID] {
			continue
		}
		applyBlockTransform(node, fm)
	}
}

// applyBlockTransform applies scale/convert_to to a single block's results.
// Statements that explicitly reference @scale are exempt from scaling
// to prevent double-scaling (e.g., "per_loaf = cost / @scale").
func applyBlockTransform(node *document.BlockNode, fm *document.Frontmatter) {
	cb, ok := node.Block.(*document.CalcBlock)
	if !ok || cb.Error() != nil {
		return
	}

	results := cb.Results()
	if len(results) == 0 {
		return
	}

	statements := cb.Statements()

	// Compute per-statement scale exemption (cached for TUI reuse).
	exempt := make([]bool, len(results))
	for i := range results {
		if i < len(statements) {
			exempt[i] = ast.ContainsScaleRef(statements[i])
		}
	}
	cb.SetScaleExempt(exempt)

	// Snapshot pre-transform units for conversion detection.
	preUnits := make([]string, len(results))
	for i, r := range results {
		preUnits[i] = unitOf(r)
	}

	// Apply transforms per-result, skipping @scale-referencing statements.
	transformed := make([]types.Type, len(results))
	for i, r := range results {
		if exempt[i] {
			// Statement uses @scale explicitly — exempt from scale transform.
			// Still apply convert_to (unit conversion is independent of scaling).
			transformed[i] = transform.Apply(r, nil, fm.ConvertTo)
		} else {
			transformed[i] = transform.Apply(r, fm.Scale, fm.ConvertTo)
		}
	}

	// Detect which results had their unit changed by convert_to.
	convertApplied := make([]bool, len(results))
	for i, t := range transformed {
		postUnit := unitOf(t)
		if preUnits[i] != "" && postUnit != "" && preUnits[i] != postUnit {
			convertApplied[i] = true
		}
	}
	cb.SetConvertApplied(convertApplied)

	cb.SetResults(transformed)

	if len(transformed) > 0 {
		cb.SetLastValue(transformed[len(transformed)-1])
	}
}

// unitOf returns the unit string for a result, or "" for non-unit types.
func unitOf(v types.Type) string {
	switch t := v.(type) {
	case *types.Quantity:
		return t.Unit
	case *types.Rate:
		if t.Amount != nil {
			return t.Amount.Unit
		}
	case *types.Fraction:
		return t.Unit
	}
	return ""
}
