package document

import (
	"fmt"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/google/uuid"
)

// Document represents a CalcMark document with incremental evaluation.
// Tracks blocks, dependencies, and provides efficient change propagation.
type Document struct {
	blocks      []*BlockNode             // Ordered blocks
	blockIndex  map[string]*BlockNode    // UUID → Block for fast lookup
	varToBlocks map[string][]string      // Dependency graph: Variable → Block UUIDs
	env         *interpreter.Environment // Accumulated environment (top-down)
}

// BlockNode wraps a Block with metadata for incremental updates.
type BlockNode struct {
	ID    string // UUID (session-ephemeral)
	Block Block  // Underlying block (CalcBlock or TextBlock)
}

// NewDocument creates a new document from CalcMark source and eagerly parses it.
func NewDocument(source string) (*Document, error) {
	doc := &Document{
		blocks:      []*BlockNode{},
		blockIndex:  make(map[string]*BlockNode),
		varToBlocks: make(map[string][]string),
		env:         interpreter.NewEnvironment(),
	}

	// Detect blocks from source
	detector := NewDetector()
	blocks, err := detector.DetectBlocks(source)
	if err != nil {
		return nil, err
	}

	// Wrap blocks with UUIDs
	for _, block := range blocks {
		node := &BlockNode{
			ID:    uuid.New().String(),
			Block: block,
		}
		doc.blocks = append(doc.blocks, node)
		doc.blockIndex[node.ID] = node
	}

	// Build dependency graph for calculation blocks
	doc.rebuildDependencies()

	return doc, nil
}

// GetBlocks returns all blocks in order.
func (d *Document) GetBlocks() []*BlockNode {
	return d.blocks
}

// GetBlock returns a block by ID.
func (d *Document) GetBlock(id string) (*BlockNode, bool) {
	node, ok := d.blockIndex[id]
	return node, ok
}

// UpdateResult contains information about what changed in the document.
type UpdateResult struct {
	// ModifiedBlockID is the block that was directly modified
	ModifiedBlockID string

	// AffectedBlockIDs are all blocks that need UI updates
	// (includes modified block + dependents)
	AffectedBlockIDs []string

	// Diagnostics from semantic validation
	Diagnostics []Diagnostic
}

// Diagnostic represents a validation issue.
type Diagnostic struct {
	BlockID  string
	Severity string // "error", "warning", "hint"
	Code     string
	Message  string
}

// ReplaceBlockSource replaces the source of a block and propagates changes.
// Returns only the affected block IDs for efficient UI updates.
func (d *Document) ReplaceBlockSource(blockID string, newSource []string) (*UpdateResult, error) {
	node, ok := d.blockIndex[blockID]
	if !ok {
		return nil, fmt.Errorf("block not found: %s", blockID)
	}

	// Update source
	switch b := node.Block.(type) {
	case *CalcBlock:
		b.source = newSource
		b.SetDirty(true)
	case *TextBlock:
		b.source = newSource
		b.SetDirty(true)
	}

	// Rebuild dependencies for this block
	affectedIDs := []string{blockID}

	if calcBlock, ok := node.Block.(*CalcBlock); ok {
		// Get old variables this block defined
		oldVars := calcBlock.Variables()

		// Re-analyze dependencies (would need parser integration here)
		// For now, mark as dirty

		// Find blocks that depended on variables this block defined
		for _, varName := range oldVars {
			if depBlockIDs, exists := d.varToBlocks[varName]; exists {
				affectedIDs = append(affectedIDs, depBlockIDs...)
			}
		}
	}

	// Mark affected blocks as dirty
	for _, id := range affectedIDs {
		if affNode, ok := d.blockIndex[id]; ok {
			affNode.Block.SetDirty(true)
		}
	}

	// Remove duplicates
	affectedIDs = uniqueStrings(affectedIDs)

	return &UpdateResult{
		ModifiedBlockID:  blockID,
		AffectedBlockIDs: affectedIDs,
	}, nil
}

// InsertBlock inserts a new block after the specified block ID.
func (d *Document) InsertBlock(afterBlockID string, blockType BlockType, source []string) (*UpdateResult, error) {
	// Find position
	pos := -1
	for i, node := range d.blocks {
		if node.ID == afterBlockID {
			pos = i
			break
		}
	}

	if pos == -1 {
		return nil, fmt.Errorf("block not found: %s", afterBlockID)
	}

	// Create new block
	var block Block
	if blockType == BlockCalculation {
		block = NewCalcBlock(source)
	} else {
		block = NewTextBlock(source)
	}

	newNode := &BlockNode{
		ID:    uuid.New().String(),
		Block: block,
	}

	// Insert into blocks array
	d.blocks = append(d.blocks[:pos+1], append([]*BlockNode{newNode}, d.blocks[pos+1:]...)...)
	d.blockIndex[newNode.ID] = newNode

	// Rebuild dependencies
	err := d.rebuildDependencies()
	if err != nil {
		return nil, err
	}

	// All blocks after this position might be affected (top-down resolution)
	affectedIDs := []string{newNode.ID}
	for i := pos + 2; i < len(d.blocks); i++ {
		if d.blocks[i].Block.Type() == BlockCalculation {
			affectedIDs = append(affectedIDs, d.blocks[i].ID)
		}
	}

	return &UpdateResult{
		ModifiedBlockID:  newNode.ID,
		AffectedBlockIDs: affectedIDs,
	}, nil
}

// DeleteBlock removes a block and updates dependents.
func (d *Document) DeleteBlock(blockID string) (*UpdateResult, error) {
	// Find position
	pos := -1
	for i, node := range d.blocks {
		if node.ID == blockID {
			pos = i
			break
		}
	}

	if pos == -1 {
		return nil, fmt.Errorf("block not found: %s", blockID)
	}

	// Remove from array
	d.blocks = append(d.blocks[:pos], d.blocks[pos+1:]...)
	delete(d.blockIndex, blockID)

	// Rebuild dependencies
	err := d.rebuildDependencies()
	if err != nil {
		return nil, err
	}

	// All blocks after this position might be affected
	affectedIDs := []string{}
	for i := pos; i < len(d.blocks); i++ {
		if d.blocks[i].Block.Type() == BlockCalculation {
			affectedIDs = append(affectedIDs, d.blocks[i].ID)
		}
	}

	return &UpdateResult{
		ModifiedBlockID:  blockID,
		AffectedBlockIDs: affectedIDs,
	}, nil
}

// rebuildDependencies rebuilds the variable → blocks dependency graph.
// This is called after structural changes to the document.
func (d *Document) rebuildDependencies() error {
	// Clear existing mappings
	d.varToBlocks = make(map[string][]string)
	analyzer := NewDependencyAnalyzer()

	// For each calc block, analyze dependencies
	for _, node := range d.blocks {
		if calcBlock, ok := node.Block.(*CalcBlock); ok {
			// Analyze the block to extract defined/referenced variables
			err := analyzer.AnalyzeBlock(calcBlock)
			if err != nil {
				// Store error but continue analyzing other blocks
				calcBlock.SetError(err)
				continue
			}

			// Variables and dependencies are tracked in the second pass below
		}
	}

	// Second pass: build dependency graph
	// For each block, find which earlier blocks define its dependencies
	envVars := make(map[string]string) // var name → block ID that defines it

	for _, node := range d.blocks {
		if calcBlock, ok := node.Block.(*CalcBlock); ok {
			// For each dependency of this block
			for _, depVar := range calcBlock.Dependencies() {
				// Find the block that defines this variable
				if _, exists := envVars[depVar]; exists {
					// Add this block to the dependents list of that variable
					d.varToBlocks[depVar] = append(d.varToBlocks[depVar], node.ID)

					// Also track reverse: this block depends on definingBlockID
					// (useful for topological sort later)
				} else {
					// Variable not yet defined - top-down error
					// (semantic checker should catch this)
				}
			}

			// Update environment with variables this block defines
			for _, varName := range calcBlock.Variables() {
				envVars[varName] = node.ID
			}
		}
	}

	return nil
}

// uniqueStrings removes duplicates from a string slice.
func uniqueStrings(strs []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, s := range strs {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
