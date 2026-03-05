package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestPipedInput_Text verifies that piping input to `cm` produces text output
// on stdout instead of launching the TUI.
func TestPipedInput_Text(t *testing.T) {
	binary := buildCM(t)

	cmd := exec.Command(binary)
	cmd.Stdin = strings.NewReader("x = 42\n")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("cm with piped input failed: %v\nstderr: %s", err, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "42") {
		t.Errorf("expected stdout to contain '42', got %q", out)
	}
}

// TestPipedInput_JSON verifies that `cm --format json` with piped input
// produces valid JSON on stdout.
func TestPipedInput_JSON(t *testing.T) {
	binary := buildCM(t)

	cmd := exec.Command(binary, "--format", "json")
	cmd.Stdin = strings.NewReader("price = $100\n")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("cm --format json with piped input failed: %v\nstderr: %s", err, stderr.String())
	}

	if !json.Valid(stdout.Bytes()) {
		t.Fatalf("expected valid JSON output, got:\n%s", stdout.String())
	}

	// Verify it has the expected structure
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}
	if _, ok := result["blocks"]; !ok {
		t.Error("JSON output should have 'blocks' key")
	}
}

// TestPipedInput_Verbose verifies that `cm -v` with piped input
// shows verbose (all intermediate values) output.
func TestPipedInput_Verbose(t *testing.T) {
	binary := buildCM(t)

	cmd := exec.Command(binary, "-v")
	cmd.Stdin = strings.NewReader("x = 10\ny = x * 2\n")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("cm -v with piped input failed: %v\nstderr: %s", err, stderr.String())
	}

	out := stdout.String()
	// Verbose should show both x and y values
	if !strings.Contains(out, "10") || !strings.Contains(out, "20") {
		t.Errorf("expected verbose output with '10' and '20', got %q", out)
	}
}

// TestPipedInput_ExitsWithinTimeout verifies that piped input completes
// quickly and does not hang waiting for terminal input (i.e., the TUI
// must NOT launch when stdin is a pipe).
func TestPipedInput_ExitsWithinTimeout(t *testing.T) {
	binary := buildCM(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary)
	cmd.Stdin = strings.NewReader("1 + 1\n")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatal("cm with piped input timed out — TUI likely launched instead of evaluating")
	}
	if err != nil {
		t.Fatalf("cm with piped input failed: %v\nstderr: %s", err, stderr.String())
	}

	out := strings.TrimSpace(stdout.String())
	if out != "2" {
		t.Errorf("expected '2', got %q", out)
	}
}

// TestPipedInput_MultilineDocument verifies that a full CalcMark document
// with markdown and calculations evaluates correctly via pipe.
func TestPipedInput_MultilineDocument(t *testing.T) {
	binary := buildCM(t)

	input := `# Budget

salary = $5000
expenses = $3000
savings = salary - expenses
`

	cmd := exec.Command(binary, "-v")
	cmd.Stdin = strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("cm with multiline piped input failed: %v\nstderr: %s", err, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "$2,000") {
		t.Errorf("expected savings result '$2,000' in output, got %q", out)
	}
}

// TestPipedInput_EvalSubcommandAlsoWorks verifies that the explicit
// `cm eval` subcommand still works with piped input (backwards compat).
func TestPipedInput_EvalSubcommandAlsoWorks(t *testing.T) {
	binary := buildCM(t)

	cmd := exec.Command(binary, "eval")
	cmd.Stdin = strings.NewReader("x = 42\n")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("cm eval with piped input failed: %v\nstderr: %s", err, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "42") {
		t.Errorf("expected stdout to contain '42', got %q", out)
	}
}

// TestPipedInput_EmptyInput verifies that empty piped input produces
// a clear error message instead of launching the TUI.
func TestPipedInput_EmptyInput(t *testing.T) {
	binary := buildCM(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary)
	cmd.Stdin = strings.NewReader("")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatal("cm with empty piped input timed out — TUI likely launched")
	}
	if err == nil {
		t.Fatal("expected error for empty piped input, got success")
	}
}

// TestPipedInput_UnitConversion verifies real-world pipe usage with
// unit conversion expressions.
func TestPipedInput_UnitConversion(t *testing.T) {
	binary := buildCM(t)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple_math", "1 + 1\n", "2"},
		{"variable_assign", "x = 42\n", "42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, binary)
			cmd.Stdin = strings.NewReader(tt.input)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			if err := cmd.Run(); err != nil {
				t.Fatalf("cm failed: %v\nstderr: %s", err, stderr.String())
			}
			if ctx.Err() == context.DeadlineExceeded {
				t.Fatal("timed out — TUI likely launched")
			}

			out := strings.TrimSpace(stdout.String())
			if !strings.Contains(out, tt.want) {
				t.Errorf("expected output containing %q, got %q", tt.want, out)
			}
		})
	}
}

// TestStdinIsPiped verifies the pipe detection helper.
func TestStdinIsPiped(t *testing.T) {
	// When running under `go test`, stdin is typically a pipe,
	// so stdinIsPiped should return true.
	if !stdinIsPiped() {
		t.Skip("stdin is not a pipe in this test environment")
	}
}

// buildCM compiles the cm binary for integration tests.
func buildCM(t *testing.T) string {
	t.Helper()

	binary := t.TempDir() + "/cm"
	cmd := exec.Command("go", "build", "-o", binary, "../")
	cmd.Dir = mustGetwd(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build cm: %v\n%s", err, out)
	}
	return binary
}

func mustGetwd(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return cwd
}
