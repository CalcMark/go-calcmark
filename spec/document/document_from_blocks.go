package document

import "github.com/google/uuid"

// NewDocumentFromBlocks constructs a *Document from pre-built
// blocks plus an optional frontmatter. Unlike NewDocument (which
// parses a source string), this constructor takes the segmentation
// decision from its caller and just stitches the result into a
// real Document — wraps each block in a BlockNode with a fresh
// UUID, sets the frontmatter, and rebuilds the dependency graph
// so the evaluator sees cross-block variable references correctly.
//
// Intended use: parser front-ends that have already done their own
// segmentation, so they can produce a Document whose evaluator
// behavior matches what NewDocument would produce for an
// equivalent source. The Embedded-mode parser at impl/embedded
// uses this to project fenced segments into CalcBlocks and
// passthrough segments into TextBlocks while keeping all blocks
// in one Document for whole-doc variable scoping.
//
// Most consumers should use NewDocument(source) instead — it
// handles parsing AND construction. Reach for this helper only
// when you have a non-flat-CM source format that needs a custom
// segmentation strategy.
//
// nil blocks slice and nil frontmatter both produce an empty but
// usable Document.
func NewDocumentFromBlocks(fm *Frontmatter, blocks []Block) *Document {
	doc := &Document{
		blocks:      []*BlockNode{},
		blockIndex:  make(map[string]*BlockNode),
		varToBlocks: make(map[string][]string),
		frontmatter: fm,
	}
	for _, block := range blocks {
		node := &BlockNode{
			ID:    uuid.New().String(),
			Block: block,
		}
		doc.blocks = append(doc.blocks, node)
		doc.blockIndex[node.ID] = node
	}
	doc.rebuildDependencies()
	return doc
}
