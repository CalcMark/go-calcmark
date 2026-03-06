// Command doceval scans Hugo markdown files for ```cm code blocks, evaluates
// each through the CalcMark interpreter, and writes the results to
// site/data/cm_results.json. Hugo's render-codeblock-cm.html render hook
// reads this file to display inline results alongside code blocks.
//
// All ```cm blocks within the same markdown file share a single interpreter
// context, so variables defined in earlier blocks are visible to later ones.
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
	var evalErrors []string

	for _, mdFile := range mdFiles {
		data, err := os.ReadFile(mdFile)
		if err != nil {
			return fmt.Errorf("read %s: %w", mdFile, err)
		}

		blocks := extractCMBlocks(string(data))
		if len(blocks) == 0 {
			continue
		}

		// Pages under docs/examples/ use shared interpreter state so
		// variables carry across code blocks. All other pages evaluate
		// each block independently (reference docs reuse variable names).
		shared := strings.Contains(mdFile, filepath.Join("docs", "examples"))

		if shared {
			// Extract CalcMark frontmatter from yaml blocks in the markdown
			frontmatter := extractFrontmatter(string(data))

			// Concatenate all blocks into a single document
			combined := frontmatter + strings.Join(blocks, "\n\n")

			doc, err := document.NewDocument(combined)
			if err != nil {
				evalErrors = append(evalErrors, fmt.Sprintf("  %s: parse error: %s", mdFile, err))
				continue
			}

			eval := implDoc.NewEvaluator()
			if err := eval.Evaluate(doc); err != nil {
				evalErrors = append(evalErrors, fmt.Sprintf("  %s: eval error: %s", mdFile, err))
				continue
			}

			// Collect all statement results
			var allStmts []stmtResult
			for _, node := range doc.GetBlocks() {
				switch block := node.Block.(type) {
				case *document.CalcBlock:
					stmts := format.AlignResults(block)
					for _, stmt := range stmts {
						sr := stmtResult{source: stmt.Source, isBlank: stmt.IsBlank, variable: stmt.Variable}
						if stmt.Result != nil {
							sr.result = df.Format(stmt.Result)
						}
						allStmts = append(allStmts, sr)
					}
				case *document.TextBlock:
					for _, line := range block.Source() {
						allStmts = append(allStmts, stmtResult{
							source:  line,
							isBlank: strings.TrimSpace(line) == "",
						})
					}
				}
			}

			// Map results back to individual blocks
			stmtIdx := 0
			for _, block := range blocks {
				key := hashKey(block)
				if _, exists := results[key]; exists {
					lines := strings.Split(block, "\n")
					for _, line := range lines {
						if strings.TrimSpace(line) != "" && stmtIdx < len(allStmts) {
							stmtIdx++
						}
					}
					continue
				}
				br := mapBlockResults(block, allStmts, &stmtIdx)
				results[key] = br
			}
		} else {
			// Independent evaluation: each block is a standalone document.
			// Errors are expected (blocks may demo frontmatter features) so
			// they are recorded in the JSON but don't fail the build.
			for _, block := range blocks {
				key := hashKey(block)
				if _, exists := results[key]; exists {
					continue
				}
				br := evalBlockIndependent(block, df)
				results[key] = br
			}
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

	if len(evalErrors) > 0 {
		fmt.Fprintf(os.Stderr, "\n%d evaluation error(s):\n", len(evalErrors))
		for _, e := range evalErrors {
			fmt.Fprintln(os.Stderr, e)
		}
		return fmt.Errorf("%d file(s) failed evaluation", len(evalErrors))
	}

	return nil
}

// mapBlockResults maps evaluated statement results back to the source lines
// of a single code block, advancing stmtIdx through the shared result list.
func mapBlockResults(block string, allStmts []stmtResult, stmtIdx *int) BlockResult {
	lines := strings.Split(block, "\n")
	var lineResults []LineResult

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lr := LineResult{Source: line, IsBlank: trimmed == ""}

		if trimmed == "" {
			lineResults = append(lineResults, lr)
			continue
		}

		if *stmtIdx < len(allStmts) {
			stmt := allStmts[*stmtIdx]
			lr.Result = stmt.result
			lr.Variable = stmt.variable
			*stmtIdx++
		}
		lineResults = append(lineResults, lr)
	}

	return BlockResult{Lines: lineResults}
}

// evalBlockIndependent evaluates a single code block as a standalone document.
func evalBlockIndependent(source string, df display.Formatter) BlockResult {
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

// stmtResult holds a single evaluated statement result (package-level for mapBlockResults).
type stmtResult struct {
	source   string
	result   string
	isBlank  bool
	variable string
}

// extractFrontmatter scans for ```yaml blocks containing CalcMark frontmatter
// (exchange rates, globals) and returns them as a CalcMark frontmatter string.
// This allows cm blocks that depend on frontmatter to evaluate correctly.
func extractFrontmatter(markdown string) string {
	lines := strings.Split(markdown, "\n")
	var yamlBlocks []string

	inYaml := false
	var current []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inYaml {
			if trimmed == "```yaml" {
				inYaml = true
				current = nil
				continue
			}
		} else {
			if trimmed == "```" {
				inYaml = false
				block := strings.Join(current, "\n")
				// Only include blocks with CalcMark frontmatter keys
				if strings.Contains(block, "exchange:") || strings.Contains(block, "globals:") {
					// Strip the --- delimiters if present, we'll add our own
					block = strings.TrimSpace(block)
					block = strings.TrimPrefix(block, "---")
					block = strings.TrimSuffix(block, "---")
					block = strings.TrimSpace(block)
					if block != "" {
						yamlBlocks = append(yamlBlocks, block)
					}
				}
				current = nil
			} else {
				current = append(current, line)
			}
		}
	}

	if len(yamlBlocks) == 0 {
		return ""
	}

	return "---\n" + strings.Join(yamlBlocks, "\n") + "\n---\n"
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

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "doceval: %v\n", err)
		os.Exit(1)
	}
}
