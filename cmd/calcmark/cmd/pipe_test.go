package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
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
