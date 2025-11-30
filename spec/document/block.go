// Package document defines the CalcMark document structure with blocks.
//
// A CalcMark document consists of blocks:
// - CalcBlock: 1+ consecutive calculation lines (like Jupyter code cell)
// - TextBlock: markdown text (like Jupyter markdown cell)
//
// Block boundaries:
// - 2 consecutive empty lines (\n\n\n) = hard boundary between blocks
// - 1 empty line (\n\n) = soft boundary (stays within current block)
package document

import (
	"strings"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/types"
)

// BlockType identifies the type of block.
type BlockType int

const (
	// BlockCalculation represents consecutive calculation lines.
	BlockCalculation BlockType = iota
	// BlockText represents markdown text.
	BlockText
)

func (bt BlockType) String() string {
	switch bt {
	case BlockCalculation:
		return "Calculation"
	case BlockText:
		return "Text"
	default:
		return "Unknown"
	}
}

// Block represents a content block in a CalcMark document.
type Block interface {
	// Type returns the block type.
	Type() BlockType

	// Source returns the raw source lines.
	Source() []string

	// IsDirty returns true if the block needs re-evaluation/rendering.
	IsDirty() bool

	// SetDirty marks the block as needing update.
	SetDirty(bool)
}

// CalcBlock represents one or more consecutive calculation lines.
// Like a Jupyter code cell.
type CalcBlock struct {
	source       []string     // Raw source lines
	statements   []ast.Node   // Parsed AST nodes (one per line)
	lastValue    types.Type   // Value of last statement
	results      []types.Type // All statement results (for inline display)
	variables    []string     // Variables defined in this block
	dependencies []string     // Variables referenced from other blocks
	err          error        // Evaluation error (legacy, prefer diagnostics)
	diagnostics  []Diagnostic // Structured errors with position info
	dirty        bool
}

// NewCalcBlock creates a new calculation block.
func NewCalcBlock(source []string) *CalcBlock {
	return &CalcBlock{
		source:       source,
		statements:   make([]ast.Node, 0, len(source)),
		variables:    []string{},
		dependencies: []string{},
		dirty:        true,
	}
}

func (cb *CalcBlock) Type() BlockType {
	return BlockCalculation
}

func (cb *CalcBlock) Source() []string {
	return cb.source
}

func (cb *CalcBlock) IsDirty() bool {
	return cb.dirty
}

func (cb *CalcBlock) SetDirty(dirty bool) {
	cb.dirty = dirty
}

// LastValue returns the value of the last statement in the block.
func (cb *CalcBlock) LastValue() types.Type {
	return cb.lastValue
}

// SetLastValue sets the last evaluated value.
func (cb *CalcBlock) SetLastValue(val types.Type) {
	cb.lastValue = val
}

// Results returns all per-statement evaluation results.
func (cb *CalcBlock) Results() []types.Type {
	return cb.results
}

// SetResults sets the per-statement evaluation results.
func (cb *CalcBlock) SetResults(results []types.Type) {
	cb.results = results
}

// Statements returns the parsed AST nodes.
func (cb *CalcBlock) Statements() []ast.Node {
	return cb.statements
}

// SetStatements sets the parsed AST nodes.
func (cb *CalcBlock) SetStatements(stmts []ast.Node) {
	cb.statements = stmts
}

// Variables returns variables defined in this block.
func (cb *CalcBlock) Variables() []string {
	return cb.variables
}

// SetVariables sets the variables defined in this block.
func (cb *CalcBlock) SetVariables(vars []string) {
	cb.variables = vars
}

// Dependencies returns variables referenced from other blocks.
func (cb *CalcBlock) Dependencies() []string {
	return cb.dependencies
}

// SetDependencies sets the dependencies.
func (cb *CalcBlock) SetDependencies(deps []string) {
	cb.dependencies = deps
}

// Error returns any evaluation error.
func (cb *CalcBlock) Error() error {
	return cb.err
}

// SetError sets the evaluation error.
func (cb *CalcBlock) SetError(err error) {
	cb.err = err
}

// Diagnostics returns structured errors/warnings with position info.
func (cb *CalcBlock) Diagnostics() []Diagnostic {
	return cb.diagnostics
}

// SetDiagnostics sets the structured diagnostics for this block.
func (cb *CalcBlock) SetDiagnostics(diags []Diagnostic) {
	cb.diagnostics = diags
}

// AddDiagnostic adds a single diagnostic to this block.
func (cb *CalcBlock) AddDiagnostic(diag Diagnostic) {
	cb.diagnostics = append(cb.diagnostics, diag)
}

// ClearDiagnostics removes all diagnostics from this block.
func (cb *CalcBlock) ClearDiagnostics() {
	cb.diagnostics = nil
}

// TextBlock represents markdown text.
// Like a Jupyter markdown cell.
type TextBlock struct {
	source []string // Raw source lines
	html   string   // Rendered HTML
	dirty  bool
}

// NewTextBlock creates a new text block.
func NewTextBlock(source []string) *TextBlock {
	return &TextBlock{
		source: source,
		dirty:  true,
	}
}

func (tb *TextBlock) Type() BlockType {
	return BlockText
}

func (tb *TextBlock) Source() []string {
	return tb.source
}

func (tb *TextBlock) IsDirty() bool {
	return tb.dirty
}

func (tb *TextBlock) SetDirty(dirty bool) {
	tb.dirty = dirty
}

// HTML returns the rendered HTML.
func (tb *TextBlock) HTML() string {
	return tb.html
}

// SetHTML sets the rendered HTML.
func (tb *TextBlock) SetHTML(html string) {
	tb.html = html
}

// SourceText returns source as a single string.
func (tb *TextBlock) SourceText() string {
	return strings.Join(tb.source, "\n")
}
