package parser_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/parser"
)

// TestSpecInvalidFilesActuallyFailParse verifies that spec/invalid files fail to parse.
// This helps identify files that should be in eval/errors instead.
func TestSpecInvalidFilesActuallyFailParse(t *testing.T) {
	dirs := []string{
		"../../testdata/spec/invalid/syntax",
		"../../testdata/spec/invalid/features",
	}

	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		files, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("failed to read dir %s: %v", dir, err)
		}

		for _, file := range files {
			if !strings.HasSuffix(file.Name(), ".cm") {
				continue
			}

			t.Run(filepath.Join(filepath.Base(dir), file.Name()), func(t *testing.T) {
				path := filepath.Join(dir, file.Name())
				content, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("failed to read file: %v", err)
				}

				_, parseErr := parser.Parse(string(content))
				if parseErr == nil {
					t.Errorf("File %s PARSES OK - should be in eval/errors/ not spec/invalid/", file.Name())
				} else {
					t.Logf("Parse error (expected): %v", parseErr)
				}
			})
		}
	}
}
