package parser_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/CalcMark/go-calcmark/spec/parser"
)

// TestGoldenFiles tests parsing against golden files.
// These are the definitive specification for CalcMark syntax.
func TestGoldenFiles(t *testing.T) {
	goldenDir := "../../../testdata/golden"

	t.Run("valid", func(t *testing.T) {
		testValidGoldenFiles(t, goldenDir)
	})

	t.Run("invalid", func(t *testing.T) {
		testInvalidGoldenFiles(t, goldenDir)
	})
}

func testValidGoldenFiles(t *testing.T, baseDir string) {
	validDir := filepath.Join(baseDir, "valid")

	// Document-level tests - CRITICAL for preventing regressions
	t.Run("documents", func(t *testing.T) {
		docDir := filepath.Join(validDir, "documents")
		testDocumentGoldenFiles(t, docDir)
	})

	// Expression-level tests
	t.Run("expressions", func(t *testing.T) {
		exprDir := filepath.Join(validDir, "expressions")
		testExpressionGoldenFiles(t, exprDir)
	})

	// Feature-specific tests
	t.Run("features", func(t *testing.T) {
		featDir := filepath.Join(validDir, "features")
		testFeatureGoldenFiles(t, featDir)
	})
}

// testDocumentGoldenFiles validates DOCUMENT-LEVEL behavior:
// - TextBlock vs CalcBlock detection
// - Block boundaries (two empty line rule)
// - Mixed content handling
func testDocumentGoldenFiles(t *testing.T, dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Skip("Document golden files directory doesn't exist yet")
		return
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".cm") {
			continue
		}

		t.Run(file.Name(), func(t *testing.T) {
			path := filepath.Join(dir, file.Name())
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}

			source := string(content)

			// CRITICAL: Test document-level parsing
			doc, err := document.NewDocument(source)
			if err != nil {
				t.Errorf("SPEC VIOLATION: Valid document %q failed to parse into document:\n%v",
					file.Name(), err)
				return
			}

			blocks := doc.GetBlocks()
			if len(blocks) == 0 {
				t.Errorf("SPEC VIOLATION: Valid document %q produced no blocks", file.Name())
				return
			}

			// Log block structure for verification
			t.Logf("Document %q parsed into %d blocks:", file.Name(), len(blocks))
			for i, block := range blocks {
				blockType := "TextBlock"
				if block.Block.Type() == document.BlockCalculation {
					blockType = "CalcBlock"
				}
				t.Logf("  Block %d: %s (%d lines)",
					i, blockType, len(block.Block.Source()))
			}

			// Also test raw parsing (AST level)
			nodes, parseErr := parser.Parse(source)
			if parseErr != nil {
				t.Errorf("SPEC VIOLATION: Valid document %q failed to parse at AST level:\n%v",
					file.Name(), parseErr)
			}
			if len(nodes) == 0 {
				t.Errorf("SPEC VIOLATION: Valid document %q produced no AST nodes", file.Name())
			}
		})
	}
}

// testExpressionGoldenFiles validates EXPRESSION-LEVEL parsing
func testExpressionGoldenFiles(t *testing.T, dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Skip("Expression golden files directory doesn't exist yet")
		return
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".cm") {
			continue
		}

		t.Run(file.Name(), func(t *testing.T) {
			path := filepath.Join(dir, file.Name())
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}

			nodes, parseErr := parser.Parse(string(content))
			if parseErr != nil {
				t.Errorf("SPEC VIOLATION: Valid expressions %q failed to parse:\n%v",
					file.Name(), parseErr)
			}
			if len(nodes) == 0 {
				t.Errorf("SPEC VIOLATION: Valid expressions %q produced no nodes", file.Name())
			}
		})
	}
}

// testFeatureGoldenFiles validates FEATURE-SPECIFIC behavior
func testFeatureGoldenFiles(t *testing.T, dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Skip("Feature golden files directory doesn't exist yet")
		return
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".cm") {
			continue
		}

		t.Run(file.Name(), func(t *testing.T) {
			path := filepath.Join(dir, file.Name())
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}

			nodes, parseErr := parser.Parse(string(content))
			if parseErr != nil {
				t.Errorf("SPEC VIOLATION: Valid feature test %q failed to parse:\n%v",
					file.Name(), parseErr)
				return
			}
			if len(nodes) == 0 {
				t.Errorf("SPEC VIOLATION: Valid feature test %q produced no nodes", file.Name())
			}
		})
	}
}

func testInvalidGoldenFiles(t *testing.T, baseDir string) {
	invalidDir := filepath.Join(baseDir, "invalid")

	if _, err := os.Stat(invalidDir); os.IsNotExist(err) {
		t.Skip("Invalid golden files directory doesn't exist yet")
		return
	}

	// Walk all subdirectories
	err := filepath.WalkDir(invalidDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(d.Name(), ".cm") {
			return nil
		}

		// Extract relative path for test name
		relPath, _ := filepath.Rel(invalidDir, path)

		t.Run(relPath, func(t *testing.T) {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}

			_, parseErr := parser.Parse(string(content))
			if parseErr == nil {
				t.Errorf("SPEC VIOLATION: Invalid document %q should have failed to parse but succeeded",
					relPath)
			}
		})

		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
}
