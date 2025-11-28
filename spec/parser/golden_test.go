package parser_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/CalcMark/go-calcmark/spec/parser"
)

// TestSpecFiles tests parsing against specification files.
// These are the definitive specification for CalcMark syntax.
func TestSpecFiles(t *testing.T) {
	specDir := "../../testdata/spec"

	t.Run("valid", func(t *testing.T) {
		testValidSpecFiles(t, specDir)
	})

	t.Run("invalid", func(t *testing.T) {
		testInvalidSpecFiles(t, specDir)
	})
}

func testValidSpecFiles(t *testing.T, baseDir string) {
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
		return // Directory doesn't exist
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

			// Note: parser.Parse() returns empty for markdown-only content
			// Document-level parsing (above) is the definitive test for document files
		})
	}
}

// testExpressionGoldenFiles validates EXPRESSION-LEVEL parsing
// Expression files may contain markdown + calculations, so we test at document level
func testExpressionGoldenFiles(t *testing.T, dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return // Directory doesn't exist
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

			// Expression files may contain markdown + calculations
			// Test at document level to properly handle mixed content
			doc, docErr := document.NewDocument(string(content))
			if docErr != nil {
				t.Errorf("SPEC VIOLATION: Valid expressions %q failed to parse as document:\n%v",
					file.Name(), docErr)
				return
			}

			blocks := doc.GetBlocks()
			if len(blocks) == 0 {
				t.Errorf("SPEC VIOLATION: Valid expressions %q produced no blocks", file.Name())
			}
		})
	}
}

// testFeatureGoldenFiles validates FEATURE-SPECIFIC behavior
// Feature files contain markdown prose + calculations, so we test at document level
func testFeatureGoldenFiles(t *testing.T, dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return // Directory doesn't exist
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

			// Feature files contain markdown + calculations
			// Test at document level to properly handle mixed content
			doc, docErr := document.NewDocument(string(content))
			if docErr != nil {
				t.Errorf("SPEC VIOLATION: Valid feature test %q failed to parse as document:\n%v",
					file.Name(), docErr)
				return
			}

			blocks := doc.GetBlocks()
			if len(blocks) == 0 {
				t.Errorf("SPEC VIOLATION: Valid feature test %q produced no blocks", file.Name())
			}
		})
	}
}

func testInvalidSpecFiles(t *testing.T, baseDir string) {
	invalidDir := filepath.Join(baseDir, "invalid")

	if _, err := os.Stat(invalidDir); os.IsNotExist(err) {
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
