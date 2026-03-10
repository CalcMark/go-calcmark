package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCalcmarkSource(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "quoted value",
			content: "---\ntitle: \"Test\"\ncalcmark_source: \"testdata/examples/foo.cm\"\n---\n",
			want:    "testdata/examples/foo.cm",
		},
		{
			name:    "unquoted value",
			content: "---\ncalcmark_source: testdata/examples/foo.cm\n---\n",
			want:    "testdata/examples/foo.cm",
		},
		{
			name:    "absent",
			content: "---\ntitle: \"Test\"\n---\n",
			want:    "",
		},
		{
			name:    "no frontmatter",
			content: "# Just markdown\n",
			want:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calcmarkSource(tt.content); got != tt.want {
				t.Errorf("calcmarkSource() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHugoTitle(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "quoted",
			content: "---\ntitle: \"Services P&L\"\n---\n",
			want:    "Services P&L",
		},
		{
			name:    "unquoted",
			content: "---\ntitle: Services\n---\n",
			want:    "Services",
		},
		{
			name:    "absent",
			content: "---\nweight: 10\n---\n",
			want:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hugoTitle(tt.content); got != tt.want {
				t.Errorf("hugoTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsGenerated(t *testing.T) {
	generated := "---\ntitle: \"Test\"\n" + generatedMarker + "\n---\n"
	if !isGenerated(generated) {
		t.Error("expected isGenerated to return true for generated content")
	}

	normal := "---\ntitle: \"Test\"\n---\n# Hello\n"
	if isGenerated(normal) {
		t.Error("expected isGenerated to return false for normal content")
	}
}

func TestGenerateFullDoc(t *testing.T) {
	// Create a temp directory with a simple .cm file
	tmpDir := t.TempDir()
	cmFile := filepath.Join(tmpDir, "test.cm")
	if err := os.WriteFile(cmFile, []byte("x = 10 + 5\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a fake parent page directory
	parentDir := filepath.Join(tmpDir, "site", "content", "docs", "examples", "test")
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	parentMD := filepath.Join(parentDir, "index.md")
	if err := os.WriteFile(parentMD, []byte("---\ntitle: \"Test Example\"\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Generate full doc
	if err := generateFullDoc(parentMD, cmFile, "Test Example"); err != nil {
		t.Fatalf("generateFullDoc failed: %v", err)
	}

	// Verify full.md was created
	fullPath := filepath.Join(parentDir, "full.md")
	data, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("failed to read generated full.md: %v", err)
	}

	content := string(data)

	// Should have Hugo frontmatter with title
	if !strings.Contains(content, `title: "Test Example — Full Document"`) {
		t.Errorf("expected title in frontmatter, got:\n%s", content)
	}

	// Should have the generated marker
	if !strings.Contains(content, generatedMarker) {
		t.Errorf("expected generated marker, got:\n%s", content)
	}

	// Should have _build.list: never
	if !strings.Contains(content, "list: never") {
		t.Errorf("expected _build.list: never, got:\n%s", content)
	}

	// Should have the evaluated result
	if !strings.Contains(content, "→ 15") {
		t.Errorf("expected evaluated result '→ 15', got:\n%s", content)
	}

	// Should NOT start with raw CalcMark frontmatter (no double --- blocks)
	// Count --- occurrences: should be exactly 2 (Hugo frontmatter open + close)
	dashes := strings.Count(content, "---")
	if dashes != 2 {
		t.Errorf("expected exactly 2 --- delimiters (Hugo frontmatter), got %d in:\n%s", dashes, content)
	}
}

func TestGenerateFullDocWithFrontmatter(t *testing.T) {
	// Test that CalcMark frontmatter (globals) becomes a code fence, not raw ---
	tmpDir := t.TempDir()
	cmFile := filepath.Join(tmpDir, "test.cm")
	cmContent := "---\nglobals:\n  rate: 0.1\n---\nx = @globals.rate * 100\n"
	if err := os.WriteFile(cmFile, []byte(cmContent), 0o644); err != nil {
		t.Fatal(err)
	}

	parentDir := filepath.Join(tmpDir, "example")
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	parentMD := filepath.Join(parentDir, "index.md")
	if err := os.WriteFile(parentMD, []byte("---\ntitle: \"Test\"\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := generateFullDoc(parentMD, cmFile, "Test"); err != nil {
		t.Fatalf("generateFullDoc failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(parentDir, "full.md"))
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)

	// CalcMark frontmatter should be rendered as ```yaml code fence
	if !strings.Contains(content, "```yaml\n") {
		t.Errorf("expected CalcMark frontmatter as yaml code fence, got:\n%s", content)
	}

	// Should contain the globals content
	if !strings.Contains(content, "rate:") {
		t.Errorf("expected globals content preserved, got:\n%s", content)
	}

	// Hugo frontmatter should be present (exactly one --- pair)
	if !strings.HasPrefix(content, "---\n") {
		t.Errorf("expected Hugo frontmatter at start, got:\n%s", content)
	}
}

func TestGenerateFullDocMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	parentMD := filepath.Join(tmpDir, "index.md")
	if err := os.WriteFile(parentMD, []byte("---\ntitle: \"Test\"\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := generateFullDoc(parentMD, filepath.Join(tmpDir, "nonexistent.cm"), "Test")
	if err == nil {
		t.Fatal("expected error for missing .cm file")
	}
	if !strings.Contains(err.Error(), "nonexistent.cm") {
		t.Errorf("error should mention the file: %v", err)
	}
}
