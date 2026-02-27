package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/filecheck"
)

// validateReadFilePath performs security checks on a file path for read-only
// commands (eval, convert). Allows absolute paths and paths outside cwd, but
// still blocks path traversal (".."), enforces extension and size limits, and
// verifies the target is a regular file.
func validateReadFilePath(path string) error {
	cleanPath := filepath.Clean(path)

	// Security: Block ".." traversal sequences after cleaning
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("invalid path: path traversal detected")
	}

	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	return validateFileConstraints(absPath)
}

// validateFilePath performs security checks on file path for commands that
// write files (edit/TUI). Restricts access to the current working directory.
func validateFilePath(path string) error {
	cleanPath := filepath.Clean(path)

	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("invalid path: path traversal detected")
	}

	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Security: Ensure the resolved path is within the current working directory.
	// Write commands (edit) should not create or modify files outside cwd.
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot determine working directory: %w", err)
	}

	relPath, err := filepath.Rel(cwd, absPath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return fmt.Errorf("invalid path: file must be within current directory")
	}

	return validateFileConstraints(absPath)
}

// validateFileConstraints checks extension, existence, type, and size.
// Shared by both read-only and write path validators.
func validateFileConstraints(absPath string) error {
	// Security: Check file extension (case-insensitive)
	ext := strings.ToLower(filepath.Ext(absPath))
	if ext != ".cm" && ext != ".calcmark" {
		return fmt.Errorf("invalid file extension: expected .cm or .calcmark")
	}

	// Security: Verify file exists and is a regular file
	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("stat file: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("invalid path: expected file, got directory")
	}

	// Security: Limit file size to 1MB
	const maxFileSize = 1 * 1024 * 1024 // 1MB
	if info.Size() > maxFileSize {
		return fmt.Errorf("file too large: %d bytes (max %d)", info.Size(), maxFileSize)
	}

	return nil
}

// validateFileContent checks that data is valid text suitable for CalcMark.
// Delegates to the shared filecheck package for reuse by the TUI editor.
func validateFileContent(data []byte) error {
	return filecheck.ValidateContent(data)
}

// validateStdinContent checks piped stdin data for binary content.
// Applies the same content checks as file input.
func validateStdinContent(data []byte) error {
	return filecheck.ValidateContent(data)
}
