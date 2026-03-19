package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

// writeTempMD writes content to a temporary .md file and returns its path.
func writeTempMD(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}

func TestEmbedded_BasicCMBlock(t *testing.T) {
	binary := buildCM(t)
	input := "# Title\n\n```cm\na = 1 + 1\n```\n\nDone.\n"
	path := writeTempMD(t, input)

	cmd := exec.Command(binary, "convert", "--embedded", path)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("embedded failed: %v\nstderr: %s", err, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "# Title") {
		t.Errorf("expected passthrough title, got %q", out)
	}
	if !strings.Contains(out, "a = 1 + 1") {
		t.Errorf("expected source line, got %q", out)
	}
	if !strings.Contains(out, "→ 2") {
		t.Errorf("expected result arrow, got %q", out)
	}
	if !strings.Contains(out, "Done.") {
		t.Errorf("expected trailing text, got %q", out)
	}
}

func TestEmbedded_CalcmarkInfoString(t *testing.T) {
	binary := buildCM(t)
	input := "```calcmark\nb = 3 * 4\n```\n"
	path := writeTempMD(t, input)

	cmd := exec.Command(binary, "convert", "--embedded", path)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		t.Fatalf("embedded failed: %v", err)
	}

	if !strings.Contains(stdout.String(), "→ 12") {
		t.Errorf("expected result, got %q", stdout.String())
	}
}

func TestEmbedded_MultipleBlocks(t *testing.T) {
	binary := buildCM(t)
	input := "Intro\n\n```cm\na = 10\n```\n\nMiddle\n\n```calcmark\nb = 20\n```\n\nEnd\n"
	path := writeTempMD(t, input)

	cmd := exec.Command(binary, "convert", "--embedded", path)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		t.Fatalf("embedded failed: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "Intro") {
		t.Error("missing Intro")
	}
	if !strings.Contains(out, "Middle") {
		t.Error("missing Middle")
	}
	if !strings.Contains(out, "End") {
		t.Error("missing End")
	}
	if !strings.Contains(out, "a = 10") {
		t.Error("missing first block")
	}
	if !strings.Contains(out, "b = 20") {
		t.Error("missing second block")
	}
}

func TestEmbedded_NoBlocks_Passthrough(t *testing.T) {
	binary := buildCM(t)
	input := "# Just Markdown\n\nNo calcmark here.\n"
	path := writeTempMD(t, input)

	cmd := exec.Command(binary, "convert", "--embedded", path)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		t.Fatalf("embedded failed: %v", err)
	}

	if stdout.String() != input {
		t.Errorf("expected byte-for-byte passthrough\nwant: %q\ngot:  %q", input, stdout.String())
	}
}

func TestEmbedded_BlockError_InlineAndNonZeroExit(t *testing.T) {
	binary := buildCM(t)
	input := "# Post\n\n```cm\nx = undefined_var + 1\n```\n\nAfter.\n"
	path := writeTempMD(t, input)

	cmd := exec.Command(binary, "convert", "--embedded", path)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit code for block with error")
	}

	out := stdout.String()
	if !strings.Contains(out, "> **CalcMark Error:**") {
		t.Errorf("expected inline error blockquote, got %q", out)
	}
	if !strings.Contains(out, "# Post") {
		t.Error("expected passthrough before error block")
	}
	if !strings.Contains(out, "After.") {
		t.Error("expected passthrough after error block")
	}
}

func TestEmbedded_HugoFrontmatter_Passthrough(t *testing.T) {
	binary := buildCM(t)
	input := "---\ntitle: My Post\ndate: 2026-03-18\n---\n\n# Content\n\n```cm\na = 5\n```\n"
	path := writeTempMD(t, input)

	cmd := exec.Command(binary, "convert", "--embedded", path)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		t.Fatalf("embedded failed: %v", err)
	}

	out := stdout.String()
	if !strings.HasPrefix(out, "---\ntitle: My Post\ndate: 2026-03-18\n---\n") {
		t.Errorf("Hugo frontmatter not preserved, got prefix: %q", out[:min(len(out), 80)])
	}
}

func TestEmbedded_BlockFrontmatter_Suppressed(t *testing.T) {
	binary := buildCM(t)
	input := "# Post\n\n```cm\n---\nscale: 2\n---\na = 5\n```\n"
	path := writeTempMD(t, input)

	cmd := exec.Command(binary, "convert", "--embedded", path)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		t.Fatalf("embedded failed: %v", err)
	}

	out := stdout.String()
	// Block frontmatter should not appear in output.
	if strings.Contains(out, "scale") {
		t.Errorf("block frontmatter leaked into output: %q", out)
	}
	// But the calculation should still be evaluated (with scale applied).
	if !strings.Contains(out, "a = 5") {
		t.Errorf("expected calculation source, got %q", out)
	}
}

func TestEmbedded_FlagError_ToHTML(t *testing.T) {
	binary := buildCM(t)
	path := writeTempMD(t, "# Test\n")

	cmd := exec.Command(binary, "convert", "--embedded", "--to", "html", path)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err == nil {
		t.Fatal("expected error for --embedded --to html")
	}

	if !strings.Contains(stderr.String(), "--embedded only supports Markdown") {
		t.Errorf("unexpected error message: %s", stderr.String())
	}
}

func TestEmbedded_FlagError_Template(t *testing.T) {
	binary := buildCM(t)
	path := writeTempMD(t, "# Test\n")

	cmd := exec.Command(binary, "convert", "--embedded", "--template", "tpl.html", path)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err == nil {
		t.Fatal("expected error for --embedded --template")
	}

	if !strings.Contains(stderr.String(), "--template is not valid with --embedded") {
		t.Errorf("unexpected error message: %s", stderr.String())
	}
}

func TestEmbedded_ToMD_Redundant_OK(t *testing.T) {
	binary := buildCM(t)
	input := "```cm\na = 1\n```\n"
	path := writeTempMD(t, input)

	cmd := exec.Command(binary, "convert", "--embedded", "--to", "md", path)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		t.Fatalf("--embedded --to md should be accepted: %v", err)
	}

	if !strings.Contains(stdout.String(), "a = 1") {
		t.Errorf("expected result, got %q", stdout.String())
	}
}

func TestEmbedded_OutputToFile(t *testing.T) {
	binary := buildCM(t)
	input := "```cm\na = 42\n```\n"
	inputPath := writeTempMD(t, input)
	outputPath := filepath.Join(t.TempDir(), "out.md")

	cmd := exec.Command(binary, "convert", "--embedded", "-o", outputPath, inputPath)
	if err := cmd.Run(); err != nil {
		t.Fatalf("embedded with -o failed: %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if !strings.Contains(string(content), "→ 42") {
		t.Errorf("expected result in output file, got %q", string(content))
	}
}

func TestEmbedded_WrongExtension(t *testing.T) {
	binary := buildCM(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("# Test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binary, "convert", "--embedded", path)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err == nil {
		t.Fatal("expected error for .txt extension with --embedded")
	}

	if !strings.Contains(stderr.String(), "expected .md or .markdown") {
		t.Errorf("unexpected error: %s", stderr.String())
	}
}

func TestEmbedded_IndependentBlocks(t *testing.T) {
	binary := buildCM(t)
	// Second block should NOT see variables from first block.
	input := "```cm\nx = 100\n```\n\n```cm\ny = x + 1\n```\n"
	path := writeTempMD(t, input)

	cmd := exec.Command(binary, "convert", "--embedded", path)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit: second block references undefined 'x'")
	}

	out := stdout.String()
	// First block should succeed.
	if !strings.Contains(out, "x = 100") {
		t.Errorf("expected first block to succeed, got %q", out)
	}
	// Second block should error because x is not defined.
	if !strings.Contains(out, "> **CalcMark Error:**") {
		t.Errorf("expected error for second block, got %q", out)
	}
}

func TestEmbedded_GoldenComplexMarkdown(t *testing.T) {
	binary := buildCM(t)

	inputPath := "testdata/embedded/complex_markdown.md"
	expectedPath := "testdata/embedded/complex_markdown.expected.md"

	// Resolve paths relative to the cmd package directory.
	cwd := mustGetwd(t)
	absInput := filepath.Join(cwd, "..", "..", "..", inputPath)
	absExpected := filepath.Join(cwd, "..", "..", "..", expectedPath)

	expected, err := os.ReadFile(absExpected)
	if err != nil {
		t.Fatalf("read expected: %v", err)
	}

	cmd := exec.Command(binary, "convert", "--embedded", absInput)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Expect non-zero exit because the test file has a deliberate error block.
	if err := cmd.Run(); err == nil {
		t.Fatal("expected non-zero exit due to deliberate error block")
	}

	got := stdout.String()
	want := string(expected)

	if got != want {
		// Find first differing line for a clear error message.
		gotLines := strings.Split(got, "\n")
		wantLines := strings.Split(want, "\n")
		for i := 0; i < len(gotLines) && i < len(wantLines); i++ {
			if gotLines[i] != wantLines[i] {
				t.Fatalf("output differs at line %d:\n  want: %q\n  got:  %q", i+1, wantLines[i], gotLines[i])
			}
		}
		if len(gotLines) != len(wantLines) {
			t.Fatalf("output line count differs: want %d, got %d", len(wantLines), len(gotLines))
		}
	}

	// Verify key passthrough features survived.
	checks := []string{
		// Hugo frontmatter
		`title: "Infrastructure Cost Analysis"`,
		// Reference links
		`[cm]: https://calcmark.com "CalcMark Homepage"`,
		// Footnotes
		`[^1]: CalcMark was designed for exactly this kind of napkin math.`,
		// GFM table
		`| Servers   | $5,400`,
		// Task list
		`- [x] Model server costs`,
		`- [ ] Add redundancy multiplier`,
		// Definition list
		`:   An interpreted language`,
		// HTML pass-through
		`<div class="callout" data-type="warning">`,
		// Image
		`![Architecture diagram]`,
		// Autolink
		`<https://calcmark.com>`,
		// Nested blockquote
		`> > > Level 3`,
		// Non-CalcMark fences
		`def estimate_cost(servers, price):`,
		"```yaml",
		`~~~bash`,
		// Indented code block
		`    This is an indented code block.`,
		// Hard line break
		"backslash\\",
		// Horizontal rule
		`---`,
		// CalcMark results
		`→ $5,400.00`,
		`→ $1,150.00`,
		`→ $870.00`,
		// Scaled block (scale factor 1000, unit_categories: [Currency])
		`→ $1M`,
		// Error block
		`> **CalcMark Error:**`,
	}
	for _, check := range checks {
		if !strings.Contains(got, check) {
			t.Errorf("missing expected content: %q", check)
		}
	}
}

// TestEmbedded_GoldmarkValidation parses the embedded mode output with goldmark
// (Hugo's markdown parser) to verify the output is valid, well-formed markdown
// that a real CommonMark parser can walk without issues.
func TestEmbedded_GoldmarkValidation(t *testing.T) {
	binary := buildCM(t)

	cwd := mustGetwd(t)
	inputPath := filepath.Join(cwd, "..", "..", "..", "testdata", "embedded", "complex_markdown.md")

	cmd := exec.Command(binary, "convert", "--embedded", inputPath)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	// Ignore exit code — we expect non-zero due to the deliberate error block.
	cmd.Run()

	output := stdout.Bytes()
	if len(output) == 0 {
		t.Fatal("embedded mode produced no output")
	}

	// Parse the output with goldmark + GFM extensions + footnotes + definition lists.
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Footnote,
			extension.DefinitionList,
		),
	)

	reader := text.NewReader(output)
	doc := md.Parser().Parse(reader)

	// Walk the AST and collect node kinds to verify structure.
	kinds := make(map[ast.NodeKind]int)
	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			kinds[n.Kind()]++
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		t.Fatalf("goldmark AST walk failed: %v", err)
	}

	// Verify key structural elements are present in the parsed AST.
	checks := map[string]ast.NodeKind{
		"Heading":        ast.KindHeading,
		"FencedCodeBlock": ast.KindFencedCodeBlock,
		"Blockquote":     ast.KindBlockquote,
		"List":           ast.KindList,
		"Link":           ast.KindLink,
		"Image":          ast.KindImage,
		"ThematicBreak":  ast.KindThematicBreak,
		"HTMLBlock":      ast.KindHTMLBlock,
		"Paragraph":      ast.KindParagraph,
	}

	for name, kind := range checks {
		if kinds[kind] == 0 {
			t.Errorf("goldmark AST missing expected node type: %s", name)
		}
	}

	// Verify the output renders to HTML without error.
	var htmlBuf bytes.Buffer
	if err := md.Convert(output, &htmlBuf); err != nil {
		t.Fatalf("goldmark HTML conversion failed: %v", err)
	}

	html := htmlBuf.String()

	// Spot-check that key HTML elements are produced.
	htmlChecks := []string{
		"<h1",          // headings
		"<table>",      // GFM table
		"<blockquote>", // blockquotes (including error block)
		"<code",        // code spans/blocks
		"<a href=",     // links
		"<img ",        // images
		"<pre>",        // fenced code blocks
		"<hr",          // thematic break
	}
	for _, check := range htmlChecks {
		if !strings.Contains(html, check) {
			t.Errorf("goldmark HTML output missing expected element: %s", check)
		}
	}

	t.Logf("goldmark parsed %d AST nodes across %d node types; HTML output: %d bytes",
		countNodes(doc), len(kinds), len(html))
}

// countNodes counts all nodes in a goldmark AST.
func countNodes(doc ast.Node) int {
	count := 0
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			count++
		}
		return ast.WalkContinue, nil
	})
	return count
}

func TestEmbedded_HelpShowsFlag(t *testing.T) {
	binary := buildCM(t)

	cmd := exec.Command(binary, "convert", "--help")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		t.Fatalf("help failed: %v", err)
	}

	if !strings.Contains(stdout.String(), "--embedded") {
		t.Error("--embedded not shown in help output")
	}
}
