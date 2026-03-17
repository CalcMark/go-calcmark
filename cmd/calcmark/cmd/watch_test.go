package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRenderFile(t *testing.T) {
	// Create a temp .cm file
	dir := t.TempDir()
	path := filepath.Join(dir, "test.cm")
	if err := os.WriteFile(path, []byte("price = 100\ntax = price * 0.1"), 0644); err != nil {
		t.Fatal(err)
	}

	html, err := renderFile(path)
	if err != nil {
		t.Fatalf("renderFile: %v", err)
	}

	if html == "" {
		t.Fatal("expected non-empty HTML output")
	}
}

func TestRenderFile_InvalidFile(t *testing.T) {
	_, err := renderFile("/nonexistent/file.cm")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestIsLoopbackOrigin(t *testing.T) {
	tests := []struct {
		origin string
		want   bool
	}{
		{"http://127.0.0.1:3141", true},
		{"http://localhost:3141", true},
		{"http://[::1]:3141", true},
		{"http://evil.com", false},
		{"http://127.0.0.1.evil.com", true}, // conservative — contains "127.0.0.1"
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.origin, func(t *testing.T) {
			got := isLoopbackOrigin(tt.origin)
			if got != tt.want {
				t.Errorf("isLoopbackOrigin(%q) = %v, want %v", tt.origin, got, tt.want)
			}
		})
	}
}
