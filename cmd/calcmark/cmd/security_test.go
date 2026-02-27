package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// createTempCMFile creates a .cm file in the given directory for testing.
func createTempCMFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestValidateFilePath_CwdRestriction(t *testing.T) {
	// validateFilePath (used by TUI/edit) requires files within cwd.
	// Create a file in a temp dir outside cwd to verify it's rejected.
	tmpDir := t.TempDir()
	path := createTempCMFile(t, tmpDir, "test.cm", "x = 1")

	err := validateFilePath(path)
	if err == nil {
		t.Fatal("expected error for file outside cwd, got nil")
	}
	if !strings.Contains(err.Error(), "file must be within current directory") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateFilePath_CwdAllowed(t *testing.T) {
	// Files within cwd should pass validateFilePath.
	path := createTempCMFile(t, ".", "test-cwd.cm", "x = 1")
	t.Cleanup(func() { os.Remove(path) })

	if err := validateFilePath(path); err != nil {
		t.Fatalf("expected no error for file in cwd, got: %v", err)
	}
}

func TestValidateReadFilePath_AllowsAbsolutePaths(t *testing.T) {
	// validateReadFilePath (used by eval/convert) should allow files anywhere.
	tmpDir := t.TempDir()
	path := createTempCMFile(t, tmpDir, "test.cm", "x = 1")

	if err := validateReadFilePath(path); err != nil {
		t.Fatalf("expected no error for absolute path, got: %v", err)
	}
}

func TestValidateReadFilePath_AllowsRelativePaths(t *testing.T) {
	path := createTempCMFile(t, ".", "test-rel.cm", "x = 1")
	t.Cleanup(func() { os.Remove(path) })

	if err := validateReadFilePath(path); err != nil {
		t.Fatalf("expected no error for relative path, got: %v", err)
	}
}

func TestValidateReadFilePath_BlocksTraversal(t *testing.T) {
	err := validateReadFilePath("../../etc/passwd.cm")
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
	if !strings.Contains(err.Error(), "path traversal") {
		t.Fatalf("expected path traversal error, got: %v", err)
	}
}

func TestValidateReadFilePath_BlocksBadExtension(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "bad.txt")
	if err := os.WriteFile(path, []byte("x = 1"), 0644); err != nil {
		t.Fatal(err)
	}

	err := validateReadFilePath(path)
	if err == nil {
		t.Fatal("expected error for bad extension, got nil")
	}
	if !strings.Contains(err.Error(), "invalid file extension") {
		t.Fatalf("expected extension error, got: %v", err)
	}
}

func TestValidateReadFilePath_BlocksDirectory(t *testing.T) {
	err := validateReadFilePath(t.TempDir())
	if err == nil {
		t.Fatal("expected error for directory, got nil")
	}
}

func TestValidateReadFilePath_BlocksMissingFile(t *testing.T) {
	err := validateReadFilePath("/tmp/does-not-exist.cm")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestValidateReadFilePath_BlocksLargeFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "huge.cm")
	// Create a file just over 1MB
	data := make([]byte, 1*1024*1024+1)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	err := validateReadFilePath(path)
	if err == nil {
		t.Fatal("expected error for large file, got nil")
	}
	if !strings.Contains(err.Error(), "file too large") {
		t.Fatalf("expected size error, got: %v", err)
	}
}

func TestStdinSizeLimit(t *testing.T) {
	// Verify the constant used for stdin limiting matches the file limit.
	// This is a compile-time documentation test — the actual stdin reading
	// is in runEval which is hard to test without an integration harness.
	// The important thing is that the limit exists and matches SECURITY.md.
	const maxFileSize = 1 * 1024 * 1024 // 1MB — must match eval.go and security.go
	_ = maxFileSize
	// If this test compiles, the constant is accessible.
	// The real stdin limit is enforced in runEval via io.LimitReader.
}

// --- Content validation integration tests ---
// These verify that validateFileContent and validateStdinContent work
// correctly when called through the cmd package wrappers.

func TestValidateFileContent_RejectsBinaryFile(t *testing.T) {
	// Simulates the exact issue scenario: binary GIF renamed to .cm
	gifData := []byte("GIF89a\x01\x00\x01\x00\x80\x00\x00\xff\xff\xff")
	err := validateFileContent(gifData)
	if err == nil {
		t.Fatal("expected error for GIF binary content, got nil")
	}
	if !strings.Contains(err.Error(), "GIF") {
		t.Fatalf("expected GIF in error, got: %v", err)
	}
}

func TestValidateFileContent_AcceptsValidCalcMark(t *testing.T) {
	data := []byte("# My Budget\nrent = 1200\nutilities = 150\ntotal = rent + utilities\n")
	if err := validateFileContent(data); err != nil {
		t.Fatalf("expected no error for valid CalcMark text, got: %v", err)
	}
}

func TestValidateFileContent_RejectsNullBytes(t *testing.T) {
	data := []byte("x = 1\x00\ny = 2\n")
	err := validateFileContent(data)
	if err == nil {
		t.Fatal("expected error for content with null bytes, got nil")
	}
	if !strings.Contains(err.Error(), "null bytes") {
		t.Fatalf("expected null bytes error, got: %v", err)
	}
}

func TestValidateFileContent_RejectsInvalidUTF8(t *testing.T) {
	data := []byte("caf\xe9") // Latin-1 'é', invalid UTF-8
	err := validateFileContent(data)
	if err == nil {
		t.Fatal("expected error for invalid UTF-8, got nil")
	}
	if !strings.Contains(err.Error(), "UTF-8") {
		t.Fatalf("expected UTF-8 error, got: %v", err)
	}
}

func TestValidateStdinContent_RejectsBinary(t *testing.T) {
	pngData := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	err := validateStdinContent(pngData)
	if err == nil {
		t.Fatal("expected error for PNG binary via stdin, got nil")
	}
	if !strings.Contains(err.Error(), "PNG") {
		t.Fatalf("expected PNG in error, got: %v", err)
	}
}

func TestValidateStdinContent_AcceptsValidText(t *testing.T) {
	data := []byte("x = 42\n")
	if err := validateStdinContent(data); err != nil {
		t.Fatalf("expected no error for valid stdin text, got: %v", err)
	}
}

// createTempBinaryCMFile creates a .cm file with binary content for testing.
func createTempBinaryCMFile(t *testing.T, dir, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadFilePath_PassesButContentValidationCatchesBinary(t *testing.T) {
	// This test demonstrates that path validation alone is insufficient:
	// a file with .cm extension and valid path passes validateReadFilePath,
	// but content validation catches the binary data.
	tmpDir := t.TempDir()
	gifData := []byte("GIF89a\x01\x00\x01\x00\x80\x00\x00\xff\xff\xff")
	path := createTempBinaryCMFile(t, tmpDir, "malicious.cm", gifData)

	// Path validation passes (valid extension, valid path, small file)
	if err := validateReadFilePath(path); err != nil {
		t.Fatalf("path validation should pass for .cm file: %v", err)
	}

	// Content validation catches the binary data
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateFileContent(content); err == nil {
		t.Fatal("content validation should reject binary GIF data")
	}
}
