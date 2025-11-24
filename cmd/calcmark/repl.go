package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	implDoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/spec/document"
)

// REPL provides an interactive CalcMark environment.
type REPL struct {
	doc    *document.Document
	reader *bufio.Reader
}

// NewREPL creates a new REPL instance.
func NewREPL() *REPL {
	// Start with empty document
	doc, _ := document.NewDocument("")

	return &REPL{
		doc:    doc,
		reader: bufio.NewReader(os.Stdin),
	}
}

// LoadFile loads a CalcMark file as seed content.
func (r *REPL) LoadFile(path string) error {
	// Security: Validate file path
	if strings.Contains(path, "..") {
		return fmt.Errorf("invalid path: contains '..'")
	}

	// Security: Check file extension (optional but recommended)
	if !strings.HasSuffix(path, ".cm") && !strings.HasSuffix(path, ".calcmark") {
		return fmt.Errorf("invalid file extension: expected .cm or .calcmark")
	}

	// Security: Get file info to check size
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat file: %w", err)
	}

	// Security: Limit file size to 1MB for REPL
	const maxFileSize = 1 * 1024 * 1024 // 1MB
	if info.Size() > maxFileSize {
		return fmt.Errorf("file too large: %d bytes (max %d)", info.Size(), maxFileSize)
	}

	// Now safe to read
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	// Create new document from file
	// Parser/semantic checker will validate CalcMark content
	doc, err := document.NewDocument(string(content))
	if err != nil {
		return fmt.Errorf("parse document: %w", err)
	}

	// Evaluate initial document
	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		return fmt.Errorf("evaluate document: %w", err)
	}

	r.doc = doc

	// Show loaded blocks
	r.showAllBlocks()

	return nil
}

// Run starts the interactive REPL loop.
func (r *REPL) Run() error {
	fmt.Println("CalcMark REPL")
	fmt.Println("Enter CalcMark expressions or commands")
	fmt.Println("Commands: :help, :vars, :blocks, :quit")
	fmt.Println()

	for {
		fmt.Print("> ")

		line, err := r.reader.ReadString('\n')
		if err != nil {
			return err
		}

		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Handle commands
		if strings.HasPrefix(line, ":") {
			if err := r.handleCommand(line); err != nil {
				if err.Error() == "quit" {
					return nil
				}
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
			continue
		}

		// Evaluate expression
		if err := r.evaluateExpression(line); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
	}
}

// handleCommand processes REPL commands.
func (r *REPL) handleCommand(cmd string) error {
	switch cmd {
	case ":help", ":h":
		r.showHelp()
	case ":vars", ":v":
		r.showVariables()
	case ":blocks", ":b":
		r.showAllBlocks()
	case ":quit", ":q":
		return fmt.Errorf("quit")
	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
	return nil
}

// evaluateExpression adds a new expression to the document and evaluates it.
func (r *REPL) evaluateExpression(expr string) error {
	// Find the last block to insert after
	blocks := r.doc.GetBlocks()

	var afterID string
	if len(blocks) > 0 {
		afterID = blocks[len(blocks)-1].ID

		// Insert new calc block
		result, err := r.doc.InsertBlock(afterID, document.BlockCalculation, []string{expr})
		if err != nil {
			return fmt.Errorf("insert block: %w", err)
		}

		// Evaluate the new block
		eval := implDoc.NewEvaluator()
		err = eval.EvaluateBlock(r.doc, result.ModifiedBlockID)
		if err != nil {
			return fmt.Errorf("evaluate: %w", err)
		}

		// Show result of new block
		r.showBlock(result.ModifiedBlockID)

		// Show affected blocks if any others were updated
		if len(result.AffectedBlockIDs) > 1 {
			fmt.Println("\nUpdated dependent blocks:")
			for _, id := range result.AffectedBlockIDs {
				if id != result.ModifiedBlockID {
					r.showBlock(id)
				}
			}
		}
	} else {
		// First block - create document with it
		doc, err := document.NewDocument(expr + "\n")
		if err != nil {
			return fmt.Errorf("parse: %w", err)
		}

		eval := implDoc.NewEvaluator()
		err = eval.Evaluate(doc)
		if err != nil {
			return fmt.Errorf("evaluate: %w", err)
		}

		r.doc = doc

		// Show result
		blocks := r.doc.GetBlocks()
		if len(blocks) > 0 {
			r.showBlock(blocks[0].ID)
		}
	}

	return nil
}

// showBlock displays a single block's result.
func (r *REPL) showBlock(blockID string) {
	node, ok := r.doc.GetBlock(blockID)
	if !ok {
		return
	}

	if calcBlock, ok := node.Block.(*document.CalcBlock); ok {
		source := strings.Join(calcBlock.Source(), " ")

		if calcBlock.Error() != nil {
			fmt.Printf("  %s\n  ❌ %v\n", source, calcBlock.Error())
		} else if calcBlock.LastValue() != nil {
			fmt.Printf("  %s\n  → %v\n", source, calcBlock.LastValue())
		}
	}
}

// showAllBlocks displays all blocks in the document.
func (r *REPL) showAllBlocks() {
	blocks := r.doc.GetBlocks()
	if len(blocks) == 0 {
		fmt.Println("No blocks")
		return
	}

	fmt.Println("Blocks:")
	for i, node := range blocks {
		if calcBlock, ok := node.Block.(*document.CalcBlock); ok {
			source := strings.Join(calcBlock.Source(), " ")

			if calcBlock.Error() != nil {
				fmt.Printf("%d. %s = ❌ %v\n", i+1, source, calcBlock.Error())
			} else if calcBlock.LastValue() != nil {
				fmt.Printf("%d. %s = %v\n", i+1, source, calcBlock.LastValue())
			}
		} else if textBlock, ok := node.Block.(*document.TextBlock); ok {
			fmt.Printf("%d. [markdown: %d lines]\n", i+1, len(textBlock.Source()))
		}
	}
}

// showVariables displays all variables in the environment.
func (r *REPL) showVariables() {
	blocks := r.doc.GetBlocks()
	if len(blocks) == 0 {
		fmt.Println("No variables defined")
		return
	}

	fmt.Println("Variables:")
	for _, node := range blocks {
		if calcBlock, ok := node.Block.(*document.CalcBlock); ok {
			for _, varName := range calcBlock.Variables() {
				if calcBlock.LastValue() != nil {
					fmt.Printf("  %s = %v\n", varName, calcBlock.LastValue())
				}
			}
		}
	}
}

// showHelp displays help information.
func (r *REPL) showHelp() {
	fmt.Println("CalcMark REPL Commands:")
	fmt.Println("  :help, :h      Show this help")
	fmt.Println("  :vars, :v      Show all variables")
	fmt.Println("  :blocks, :b    Show all blocks")
	fmt.Println("  :quit, :q      Exit REPL")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  x = 10")
	fmt.Println("  y = x + 5")
	fmt.Println("  price = 100 USD")
}
