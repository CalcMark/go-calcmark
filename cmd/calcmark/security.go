package main

import (
	"fmt"
	"os"
	"strings"
)

// validateFilePath performs security checks on file path
func validateFilePath(path string) error {
	// Security: Validate file path
	if strings.Contains(path, "..") {
		return fmt.Errorf("invalid path: contains '..'")
	}

	// Security: Check file extension
	if !strings.HasSuffix(path, ".cm") && !strings.HasSuffix(path, ".calcmark") {
		return fmt.Errorf("invalid file extension: expected .cm or .calcmark")
	}

	// Security: Get file info to check size
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat file: %w", err)
	}

	// Security: Limit file size to 1MB
	const maxFileSize = 1 * 1024 * 1024 // 1MB
	if info.Size() > maxFileSize {
		return fmt.Errorf("file too large: %d bytes (max %d)", info.Size(), maxFileSize)
	}

	return nil
}
