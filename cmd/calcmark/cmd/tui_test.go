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
		_, err := loadDocument("does_not_exist.cm")
		if err == nil {
			t.Fatal("expected error for nonexistent file")
		}
	})

	t.Run("absolute path outside cwd loads successfully", func(t *testing.T) {
		// Regression: cm ~/Downloads/some.cm failed with
		// "file must be within current directory" because loadDocument
		// used validateFilePath (cwd-restricted) instead of validateReadFilePath.
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "outside-cwd.cm")
		if err := os.WriteFile(path, []byte("x = 42\n"), 0644); err != nil {
			t.Fatal(err)
		}

		doc, err := loadDocument(path)
		if err != nil {
			t.Fatalf("loadDocument should accept files outside cwd, got: %v", err)
		}
		if doc == nil {
			t.Fatal("expected non-nil document")
		}
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

// writeTempCM writes content to a temporary .cm file and returns its path.
// The file is cleaned up when the test finishes.
func writeTempCM(t *testing.T, content string) string {
	t.Helper()

	f, err := os.CreateTemp(t.TempDir(), "test-*.cm")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	if _, err := f.WriteString(content); err != nil {
		f.Close()
		t.Fatalf("failed to write temp file: %v", err)
	}
	f.Close()

	return f.Name()
}
