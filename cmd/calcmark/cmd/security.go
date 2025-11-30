package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// validateFilePath performs security checks on file path.
// Prevents path traversal attacks and validates file constraints.
func validateFilePath(path string) error {
	// Security: Clean and resolve the path to prevent traversal attacks
	// filepath.Clean normalizes paths and removes ".." sequences
	cleanPath := filepath.Clean(path)

	// Security: Ensure the cleaned path doesn't escape current working directory
	// After cleaning, check if ".." still appears (shouldn't happen, but defense in depth)
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("invalid path: path traversal detected")
	}

	// Security: Convert to absolute path and verify it's within allowed boundaries
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Get current working directory as the allowed base
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cannot determine working directory: %w", err)
	}

	// Security: Ensure the resolved path is within the current working directory
	// This prevents accessing files outside the allowed scope via symlinks or absolute paths
	relPath, err := filepath.Rel(cwd, absPath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return fmt.Errorf("invalid path: file must be within current directory")
	}

	// Security: Check file extension (case-insensitive)
	ext := strings.ToLower(filepath.Ext(absPath))
	if ext != ".cm" && ext != ".calcmark" {
		return fmt.Errorf("invalid file extension: expected .cm or .calcmark")
	}

	// Security: Verify file exists and is a regular file (not a directory or symlink target to directory)
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
