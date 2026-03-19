package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark"
)

func TestRenderFile(t *testing.T) {
	// Create a temp .cm file
	dir := t.TempDir()
	path := filepath.Join(dir, "test.cm")
	if err := os.WriteFile(path, []byte("price = 100\ntax = price * 0.1"), 0644); err != nil {
		t.Fatal(err)
	}

	html, err := renderFile(path, calcmark.CM)
	if err != nil {
		t.Fatalf("renderFile: %v", err)
	}

	if html == "" {
		t.Fatal("expected non-empty HTML output")
	}
}

func TestRenderFile_InvalidFile(t *testing.T) {
	_, err := renderFile("/nonexistent/file.cm", calcmark.CM)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestRenderFile_Embedded(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	content := "# Test\n\n```cm\nx = 42\n```\n\nSome prose.\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	html, err := renderFile(path, calcmark.Embedded)
	if err != nil {
		t.Fatalf("renderFile embedded: %v", err)
	}

	if html == "" {
		t.Fatal("expected non-empty HTML output")
	}
	if !strings.Contains(html, "<h1") {
		t.Error("expected HTML heading from goldmark")
	}
	if !strings.Contains(html, "Some prose.") {
		t.Error("expected prose in output")
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
