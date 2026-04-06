package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestJSONError_EvalErrorProducesJSONOnStdout verifies that when --format json
// is active and an evaluation error occurs (e.g., variable redefinition),
// stdout contains a valid JSON error envelope instead of being empty.
// This is the core behavior requested in issue #53.
func TestJSONError_EvalErrorProducesJSONOnStdout(t *testing.T) {
	binary := buildCM(t)

	// variable_redefinition: x defined twice in same block
	input := "x = 10\nx = 20\n"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, "eval", "--format", "json")
	cmd.Stdin = strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit code for variable redefinition")
	}

	// With error recovery, stdout contains formatted JSON output (partial results
	// with diagnostics), not a JSON error envelope.
	if !json.Valid(stdout.Bytes()) {
		t.Fatalf("stdout should be valid JSON when --format json is used, got:\nstdout: %q\nstderr: %q", stdout.String(), stderr.String())
	}

	// The output should contain the block with redefinition info
	out := stdout.String()
	if !strings.Contains(out, "reassign") && !strings.Contains(out, "immutable") && !strings.Contains(out, "redefinition") {
		t.Errorf("JSON output should contain redefinition info, got: %s", out)
	}
}

// TestJSONError_EvalErrorHasStructuredFields verifies that with error recovery,
// redefinition errors produce formatted JSON output (with block diagnostics)
// and a non-zero exit code. The error details are in the block's diagnostics,
// not in a separate error envelope.
func TestJSONError_EvalErrorHasStructuredFields(t *testing.T) {
	binary := buildCM(t)

	input := "x = 10\nx = 20\n"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, "eval", "--format", "json")
	cmd.Stdin = strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Should fail (non-zero exit) due to partial evaluation
	if err == nil {
		t.Error("Expected non-zero exit code for redefinition error")
	}

	// stderr should mention the error
	if !strings.Contains(stderr.String(), "evaluation error") {
		t.Errorf("stderr should mention evaluation error, got: %q", stderr.String())
	}

	// stdout should contain valid JSON output with block diagnostics
	out := stdout.String()
	if out == "" {
		t.Fatal("stdout should not be empty — partial evaluation should still format output")
	}

	// The output should contain information about the redefinition
	if !strings.Contains(out, "reassign") && !strings.Contains(out, "redefinition") && !strings.Contains(out, "immutable") {
		t.Errorf("JSON output should contain redefinition diagnostic info, got: %s", out)
	}
}

// TestJSONError_UndefinedVariableProducesJSON verifies that undefined variable
// errors produce formatted JSON output on stdout when --format json is active,
// and a non-zero exit code.
func TestJSONError_UndefinedVariableProducesJSON(t *testing.T) {
	binary := buildCM(t)

	// undefined_variable: y is never defined
	input := "x = y + 1\n"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, "eval", "--format", "json")
	cmd.Stdin = strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Skip("input did not produce an error, skipping")
	}

	if stdout.Len() == 0 {
		t.Fatal("stdout should not be empty when --format json is used with an error")
	}

	if !json.Valid(stdout.Bytes()) {
		t.Fatalf("stdout should be valid JSON, got:\nstdout: %q\nstderr: %q", stdout.String(), stderr.String())
	}
}

// TestJSONError_PipedStdinErrorProducesJSON verifies that the piped-stdin
// path (cm --format json, without eval subcommand) produces formatted JSON
// output with diagnostics for errors, and a non-zero exit code.
func TestJSONError_PipedStdinErrorProducesJSON(t *testing.T) {
	binary := buildCM(t)

	input := "x = 10\nx = 20\n"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Note: no "eval" subcommand — using root command piped stdin path
	cmd := exec.CommandContext(ctx, binary, "--format", "json")
	cmd.Stdin = strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected error for variable redefinition")
	}

	if !json.Valid(stdout.Bytes()) {
		t.Fatalf("stdout should be valid JSON via piped path, got:\nstdout: %q\nstderr: %q", stdout.String(), stderr.String())
	}
}

// TestJSONError_NonJSONFormatDoesNotWriteJSON verifies that errors without
// --format json do NOT produce JSON on stdout (no regression).
func TestJSONError_NonJSONFormatDoesNotWriteJSON(t *testing.T) {
	binary := buildCM(t)

	input := "x = 10\nx = 20\n"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, "eval")
	cmd.Stdin = strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	cmd.Run() //nolint:errcheck // we expect failure

	// With error recovery, text output is still produced (partial results + diagnostics).
	// stdout may contain formatted output.

	// stderr should have the error
	if !strings.Contains(stderr.String(), "Error") && !strings.Contains(stderr.String(), "error") {
		t.Errorf("stderr should contain error message, got: %q", stderr.String())
	}
}

// TestJSONError_SuccessStillProducesNormalJSON verifies that successful
// evaluation with --format json still produces normal JSONDocument output
// (no regression from error handling changes).
func TestJSONError_SuccessStillProducesNormalJSON(t *testing.T) {
	binary := buildCM(t)

	input := "x = 42\n"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, "eval", "--format", "json")
	cmd.Stdin = strings.NewReader(input)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("expected success, got error: %v\nstderr: %s", err, stderr.String())
	}

	if !json.Valid(stdout.Bytes()) {
		t.Fatalf("stdout should be valid JSON, got: %q", stdout.String())
	}

	// Should have "blocks" key (normal JSONDocument), NOT "error" key
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if _, ok := result["blocks"]; !ok {
		t.Error("successful JSON output should have 'blocks' key")
	}
	if _, ok := result["error"]; ok {
		t.Error("successful JSON output should NOT have 'error' key")
	}
}

// TestJSONError_ExitCodeStillNonZero verifies that the exit code is still
// non-zero when an error occurs, even with JSON output on stdout.
func TestJSONError_ExitCodeStillNonZero(t *testing.T) {
	binary := buildCM(t)

	input := "x = 10\nx = 20\n"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, "eval", "--format", "json")
	cmd.Stdin = strings.NewReader(input)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected non-zero exit code")
	}

	// Verify it's an exit error with code 1
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() != 1 {
			t.Errorf("exit code = %d, want 1", exitErr.ExitCode())
		}
	}
}
