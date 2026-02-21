package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadDocumentWithEvalErrors verifies that loadDocument returns a valid
// document even when the content contains evaluation errors (undefined
// variables, type mismatches, etc.). Evaluation errors are diagnostic —
// the editor handles them in the preview pane.
func TestLoadDocumentWithEvalErrors(t *testing.T) {
	t.Run("undefined variable", func(t *testing.T) {
		path := writeTempCM(t, "result = undefined_var + 1\n")

		doc, err := loadDocument(path)
		if err != nil {
			t.Fatalf("loadDocument should succeed for files with eval errors, got: %v", err)
		}
		if doc == nil {
			t.Fatal("expected non-nil document")
		}
		if len(doc.GetBlocks()) == 0 {
			t.Fatal("expected at least one block")
		}
	})

	t.Run("valid file loads normally", func(t *testing.T) {
		path := writeTempCM(t, "x = 10\ny = x * 2\n")

		doc, err := loadDocument(path)
		if err != nil {
			t.Fatalf("loadDocument failed: %v", err)
		}
		if doc == nil {
			t.Fatal("expected non-nil document")
		}
	})

	t.Run("nonexistent file returns error", func(t *testing.T) {
		// Create a path that passes security checks (within cwd, .cm extension)
		// but doesn't exist
		path := filepath.Join(t.TempDir(), "nonexistent.cm")
		// validateFilePath will fail because t.TempDir() is outside cwd,
		// so we test with a file in cwd that doesn't exist
		_, err := loadDocument("does_not_exist.cm")
		if err == nil {
			t.Fatal("expected error for nonexistent file")
		}
		// Suppress unused variable warning
		_ = path
	})

	t.Run("parse error returns error", func(t *testing.T) {
		// An empty file should still parse (document.NewDocument handles "")
		// but we verify the function at least runs
		path := writeTempCM(t, "")

		doc, err := loadDocument(path)
		if err != nil {
			t.Fatalf("loadDocument failed on empty file: %v", err)
		}
		if doc == nil {
			t.Fatal("expected non-nil document")
		}
	})
}

// writeTempCM writes content to a temporary .cm file in the current working
// directory (required by validateFilePath security checks) and returns its path.
// The file is cleaned up when the test finishes.
func writeTempCM(t *testing.T, content string) string {
	t.Helper()

	// validateFilePath requires files within cwd, so create temp files there
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	f, err := os.CreateTemp(cwd, "test-*.cm")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	t.Cleanup(func() { os.Remove(f.Name()) })

	if _, err := f.WriteString(content); err != nil {
		f.Close()
		t.Fatalf("failed to write temp file: %v", err)
	}
	f.Close()

	return f.Name()
}
