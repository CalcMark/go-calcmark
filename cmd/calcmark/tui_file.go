package main

import (
	"fmt"
	"os"

	implDoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/spec/document"
)

// openFile loads a CalcMark file and replaces the current document
func (m model) openFile(path string) model {
	// Security: validate file path
	if err := validateFilePath(path); err != nil {
		m.err = fmt.Errorf("invalid file: %w", err)
		return m
	}

	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		m.err = fmt.Errorf("read file: %w", err)
		return m
	}

	// Parse document
	doc, err := document.NewDocument(string(content))
	if err != nil {
		m.err = fmt.Errorf("parse document: %w", err)
		return m
	}

	// Evaluate
	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		m.err = fmt.Errorf("evaluate document: %w", err)
		return m
	}

	// Replace current document
	m.doc = doc

	// Clear pinned vars (new document)
	m.pinnedVars = make(map[string]bool)

	// Clear error
	m.err = nil

	return m
}
