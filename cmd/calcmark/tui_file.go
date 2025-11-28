package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/CalcMark/go-calcmark/format"
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

// saveFile saves the current document to a CalcMark file (plain text, round-trip safe).
// The output should be identical to the input when no modifications were made.
func (m model) saveFile(path string) model {
	// Validate extension
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".cm" && ext != ".calcmark" {
		m.err = fmt.Errorf("save requires .cm or .calcmark extension, got %q", ext)
		return m
	}

	// Use CalcMarkFormatter with non-verbose mode for round-trip safety
	var buf strings.Builder
	formatter := &format.CalcMarkFormatter{}
	err := formatter.Format(&buf, m.doc, format.Options{Verbose: false})
	if err != nil {
		m.err = fmt.Errorf("format document: %w", err)
		return m
	}

	// Write to file
	err = os.WriteFile(path, []byte(buf.String()), 0644)
	if err != nil {
		m.err = fmt.Errorf("write file: %w", err)
		return m
	}

	// Add success message to output history
	m.outputHistory = append(m.outputHistory, outputHistoryItem{
		input:   "/save " + path,
		output:  fmt.Sprintf("Saved to %s", path),
		isError: false,
	})

	return m
}

// outputFile exports the document to a formatted output file (with intermediate results).
// Supports: .html, .md, .json formats. Uses the format package.
func (m model) outputFile(path string) model {
	// Get formatter based on file extension
	formatter := format.GetFormatter("", path)

	// Use verbose mode to include intermediate calculation results
	var buf strings.Builder
	err := formatter.Format(&buf, m.doc, format.Options{
		Verbose:       true,
		IncludeErrors: true,
	})
	if err != nil {
		m.err = fmt.Errorf("format document: %w", err)
		return m
	}

	// Write to file
	err = os.WriteFile(path, []byte(buf.String()), 0644)
	if err != nil {
		m.err = fmt.Errorf("write file: %w", err)
		return m
	}

	// Add success message to output history
	ext := filepath.Ext(path)
	m.outputHistory = append(m.outputHistory, outputHistoryItem{
		input:   "/output " + path,
		output:  fmt.Sprintf("Exported to %s (%s format)", path, ext),
		isError: false,
	})

	return m
}
