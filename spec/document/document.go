package document

import (
	"fmt"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Document represents a CalcMark document with incremental evaluation.
// Tracks blocks, dependencies, and provides efficient change propagation.
type Document struct {
	blocks      []*BlockNode             // Ordered blocks
	blockIndex  map[string]*BlockNode    // UUID → Block for fast lookup
	varToBlocks map[string][]string      // Dependency graph: Variable → Block UUIDs
	env         *interpreter.Environment // Accumulated environment (top-down)
	frontmatter *Frontmatter             // Parsed frontmatter (exchange rates, globals)
}

// BlockNode wraps a Block with metadata for incremental updates.
type BlockNode struct {
	ID    string // UUID (session-ephemeral)
	Block Block  // Underlying block (CalcBlock or TextBlock)
}

// NewDocument creates a new document from CalcMark source and eagerly parses it.
func NewDocument(source string) (*Document, error) {
	// Parse frontmatter first (if present)
	fm, remaining, err := ParseFrontmatter(source)
	if err != nil {
		return nil, fmt.Errorf("frontmatter: %w", err)
	}

	doc := &Document{
		blocks:      []*BlockNode{},
		blockIndex:  make(map[string]*BlockNode),
		varToBlocks: make(map[string][]string),
		env:         interpreter.NewEnvironment(),
		frontmatter: fm,
	}

	// Detect blocks from remaining source (after frontmatter)
	detector := NewDetector()
	blocks, err := detector.DetectBlocks(remaining)
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

// Diagnostic represents a validation issue with source position info.
type Diagnostic struct {
	BlockID  string
	Severity string // "error", "warning", "hint"
	Code     string
	Message  string
	Line     int // 1-indexed line number within the block
	Column   int // 1-indexed column number
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

	// Rebuild dependencies (this analyzes the new block and updates varToBlocks)
	err := d.rebuildDependencies()
	if err != nil {
		return nil, err
	}

	// Collect affected blocks: the new block + all blocks that transitively depend
	// on variables defined by the new block
	affectedIDs := []string{newNode.ID}

	if calcBlock, ok := newNode.Block.(*CalcBlock); ok {
		// Use GetTransitiveDependents to find ALL affected blocks (direct + transitive)
		changedVars := calcBlock.Variables()
		transitiveIDs := d.GetTransitiveDependents(changedVars)
		affectedIDs = append(affectedIDs, transitiveIDs...)
	}

	// Remove duplicates
	affectedIDs = uniqueStrings(affectedIDs)

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

// GetTransitiveDependents returns all block IDs that transitively depend on
// any of the given variables. This follows the dependency chain:
// if a→b→c, changing 'a' returns blocks containing 'b' AND 'c'.
//
// This is the key API for reactive UIs: when a variable changes, call this
// to get the minimal set of blocks that need re-evaluation.
func (d *Document) GetTransitiveDependents(changedVars []string) []string {
	affected := make(map[string]bool)
	visited := make(map[string]bool)

	// Start with all variables that changed
	varsToProcess := make([]string, len(changedVars))
	copy(varsToProcess, changedVars)

	for len(varsToProcess) > 0 {
		// Pop a variable
		varName := varsToProcess[0]
		varsToProcess = varsToProcess[1:]

		if visited[varName] {
			continue
		}
		visited[varName] = true

		// Find all blocks that depend on this variable
		for _, blockID := range d.varToBlocks[varName] {
			if affected[blockID] {
				continue
			}
			affected[blockID] = true

			// Find variables defined by this block - they're now "changed" too
			if node, ok := d.blockIndex[blockID]; ok {
				if cb, ok := node.Block.(*CalcBlock); ok {
					for _, definedVar := range cb.Variables() {
						if !visited[definedVar] {
							varsToProcess = append(varsToProcess, definedVar)
						}
					}
				}
			}
		}
	}

	// Convert to slice
	result := make([]string, 0, len(affected))
	for blockID := range affected {
		result = append(result, blockID)
	}
	return result
}

// GetBlocksInDependencyOrder returns block IDs sorted so that dependencies
// come before dependents. This is a topological sort based on the dependency graph.
// Useful for evaluating blocks in the correct order after a change.
func (d *Document) GetBlocksInDependencyOrder(blockIDs []string) []string {
	// Build a set for quick lookup
	blockSet := make(map[string]bool)
	for _, id := range blockIDs {
		blockSet[id] = true
	}

	// Return in document order (which is already topologically sorted for top-down semantics)
	// This works because CalcMark has top-down semantics: variables must be defined before use
	result := make([]string, 0, len(blockIDs))
	for _, node := range d.blocks {
		if blockSet[node.ID] {
			result = append(result, node.ID)
		}
	}
	return result
}

// GetFrontmatter returns the document's frontmatter (may be nil).
func (d *Document) GetFrontmatter() *Frontmatter {
	return d.frontmatter
}

// SetFrontmatter replaces the document's frontmatter.
func (d *Document) SetFrontmatter(fm *Frontmatter) {
	d.frontmatter = fm
}

// EnsureFrontmatter returns the frontmatter, creating an empty one if nil.
func (d *Document) EnsureFrontmatter() *Frontmatter {
	if d.frontmatter == nil {
		d.frontmatter = &Frontmatter{
			Exchange: make(map[string]decimal.Decimal),
			Globals:  make(map[string]string),
		}
	}
	return d.frontmatter
}

// ApplyFrontmatter injects frontmatter values (exchange rates, globals) into
// the given interpreter environment. This should be called before evaluation.
func (d *Document) ApplyFrontmatter(env *interpreter.Environment) error {
	if d.frontmatter == nil {
		return nil
	}

	// Apply exchange rates
	for key, rate := range d.frontmatter.Exchange {
		from, to, err := ParseExchangeRateKey(key)
		if err != nil {
			return fmt.Errorf("apply frontmatter: %w", err)
		}
		env.SetExchangeRate(from, to, rate)
	}

	// Apply globals (parse literal values and inject as variables)
	if len(d.frontmatter.Globals) > 0 {
		parsed, err := ParseGlobals(d.frontmatter.Globals)
		if err != nil {
			return fmt.Errorf("apply frontmatter globals: %w", err)
		}
		for name, value := range parsed.Values {
			env.Set(name, value)
		}
	}

	return nil
}
