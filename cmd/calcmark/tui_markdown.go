package main

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/spec/document"
)

// insertMarkdownBlock adds accumulated markdown as a text block
func (m model) insertMarkdownBlock() model {
	blocks := m.doc.GetBlocks()

	var afterID string
	if len(blocks) > 0 {
		afterID = blocks[len(blocks)-1].ID

		// Insert markdown as text block
		_, err := m.doc.InsertBlock(afterID, document.BlockText, m.markdownLines)
		if err != nil {
			m.err = fmt.Errorf("insert markdown block: %w", err)
			return m
		}
	} else {
		// Create document with markdown block
		source := strings.Join(m.markdownLines, "\n") + "\n"
		doc, err := document.NewDocument(source)
		if err != nil {
			m.err = fmt.Errorf("parse markdown: %w", err)
			return m
		}
		m.doc = doc
	}

	return m
}
