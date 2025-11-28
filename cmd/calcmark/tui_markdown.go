package main

import (
	"fmt"
	"strings"

	"github.com/CalcMark/go-calcmark/spec/document"
)

// insertMarkdownBlock adds the markdown content from mdInput as a text block.
func (m model) insertMarkdownBlock() model {
	content := m.mdInput.Value()
	if strings.TrimSpace(content) == "" {
		return m
	}

	// Split content into lines for the document API
	lines := strings.Split(content, "\n")

	blocks := m.doc.GetBlocks()

	var afterID string
	if len(blocks) > 0 {
		afterID = blocks[len(blocks)-1].ID

		// Insert markdown as text block
		_, err := m.doc.InsertBlock(afterID, document.BlockText, lines)
		if err != nil {
			m.err = fmt.Errorf("insert markdown block: %w", err)
			return m
		}
	} else {
		// Create document with markdown block
		source := content
		if !strings.HasSuffix(source, "\n") {
			source += "\n"
		}
		doc, err := document.NewDocument(source)
		if err != nil {
			m.err = fmt.Errorf("parse markdown: %w", err)
			return m
		}
		m.doc = doc
	}

	// Add to output history as confirmation
	m.outputHistory = append(m.outputHistory, outputHistoryItem{
		input:   "/md",
		output:  fmt.Sprintf("Added markdown block (%d lines)", len(lines)),
		isError: false,
	})

	return m
}
