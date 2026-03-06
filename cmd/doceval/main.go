// Command doceval scans Hugo markdown files for ```cm code blocks, evaluates
// each through the CalcMark interpreter, and writes the results to
// site/data/cm_results.json. Hugo's render-codeblock-cm.html render hook
// reads this file to display inline results alongside code blocks.
//
// Invoked by `task generate-docs` before Hugo builds the site.
package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/CalcMark/go-calcmark/format"
	"github.com/CalcMark/go-calcmark/format/display"
	implDoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/spec/document"
)

// BlockResult holds the evaluated results for a single ```cm code block.
type BlockResult struct {
	Lines []LineResult `json:"lines"`
	Error string       `json:"error,omitempty"`
}

// LineResult pairs a source line with its display-formatted result.
type LineResult struct {
	Source   string `json:"source"`
	Result   string `json:"result,omitempty"`
	IsBlank  bool   `json:"is_blank,omitempty"`
	Variable string `json:"variable,omitempty"`
}

func run() error {
	contentDir := filepath.Join("site", "content")

	// Find all .md files under site/content/
	var mdFiles []string
	err := filepath.Walk(contentDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".md") {
			mdFiles = append(mdFiles, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk %s: %w", contentDir, err)
	}

	results := make(map[string]BlockResult)
	df := display.DefaultFormatter()

	for _, mdFile := range mdFiles {
		data, err := os.ReadFile(mdFile)
		if err != nil {
			return fmt.Errorf("read %s: %w", mdFile, err)
		}

		blocks := extractCMBlocks(string(data))
		for _, block := range blocks {
			key := hashKey(block)
			if _, exists := results[key]; exists {
				continue // deduplicate identical blocks
			}

			br := evalBlock(block, df)
			results[key] = br
		}
	}

	outPath := filepath.Join("site", "data", "cm_results.json")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}

	if err := os.WriteFile(outPath, jsonData, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}

	fmt.Printf("Generated %s (%d blocks from %d files)\n", outPath, len(results), len(mdFiles))
	return nil
}

// extractCMBlocks pulls all ```cm fenced code block contents from markdown.
func extractCMBlocks(markdown string) []string {
	var blocks []string
	lines := strings.Split(markdown, "\n")

	inBlock := false
	var current []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !inBlock {
			// Match opening fence: ```cm with optional attributes
			if strings.HasPrefix(trimmed, "```cm") && !strings.HasPrefix(trimmed, "```cmd") {
				rest := strings.TrimPrefix(trimmed, "```cm")
				// Accept ```cm, ```cm{...}, or ```cm followed by space/nothing
				if rest == "" || rest[0] == '{' || rest[0] == ' ' {
					inBlock = true
					current = nil
					continue
				}
			}
		} else {
			if trimmed == "```" {
				inBlock = false
				blocks = append(blocks, strings.Join(current, "\n"))
				current = nil
			} else {
				current = append(current, line)
			}
		}
	}

	return blocks
}

// hashKey produces a stable SHA-256 hash of a code block's content,
// used as the lookup key in cm_results.json.
func hashKey(content string) string {
	// Normalize: trim trailing whitespace from each line, then join
	lines := strings.Split(content, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	normalized := strings.TrimSpace(strings.Join(lines, "\n"))
	h := sha256.Sum256([]byte(normalized))
	return fmt.Sprintf("%x", h)
}

// evalBlock evaluates a CalcMark code block and returns structured results.
func evalBlock(source string, df display.Formatter) BlockResult {
	doc, err := document.NewDocument(source)
	if err != nil {
		return BlockResult{Error: err.Error()}
	}

	eval := implDoc.NewEvaluator()
	if err := eval.Evaluate(doc); err != nil {
		return BlockResult{Error: err.Error()}
	}

	var lineResults []LineResult
	for _, node := range doc.GetBlocks() {
		switch block := node.Block.(type) {
		case *document.CalcBlock:
			stmts := format.AlignResults(block)
			for _, stmt := range stmts {
				lr := LineResult{Source: stmt.Source, IsBlank: stmt.IsBlank, Variable: stmt.Variable}
				if stmt.Result != nil {
					lr.Result = df.Format(stmt.Result)
				}
				lineResults = append(lineResults, lr)
			}
		case *document.TextBlock:
			// Markdown lines within a cm block (e.g., prose between calculations)
			for _, line := range block.Source() {
				lineResults = append(lineResults, LineResult{
					Source:  line,
					IsBlank: strings.TrimSpace(line) == "",
				})
			}
		}
	}

	return BlockResult{Lines: lineResults}
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "doceval: %v\n", err)
		os.Exit(1)
	}
}
