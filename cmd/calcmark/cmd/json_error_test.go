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

// jsonErrorEnvelopeTest mirrors the JSON error structure we expect on stdout.
type jsonErrorEnvelopeTest struct {
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
		Line    int    `json:"line,omitempty"`
		Code    string `json:"code,omitempty"`
	} `json:"error"`
}

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

	// stdout must contain valid JSON
	if !json.Valid(stdout.Bytes()) {
		t.Fatalf("stdout should be valid JSON when --format json is used, got:\nstdout: %q\nstderr: %q", stdout.String(), stderr.String())
	}

	// Parse and verify the error envelope
	var envelope jsonErrorEnvelopeTest
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("failed to unmarshal JSON error: %v\nstdout: %s", err, stdout.String())
	}

	if envelope.Error.Type == "" {
		t.Error("error.type should not be empty")
	}
	if envelope.Error.Message == "" {
		t.Error("error.message should not be empty")
	}
}

// TestJSONError_EvalErrorHasStructuredFields verifies that the JSON error
// envelope includes type, line, code, and message fields extracted from
// structured error messages.
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

	cmd.Run() //nolint:errcheck // we expect failure

	var envelope jsonErrorEnvelopeTest
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("failed to unmarshal JSON error: %v\nstdout: %s", err, stdout.String())
	}

	if envelope.Error.Type != "evaluation_error" {
		t.Errorf("error.type = %q, want %q", envelope.Error.Type, "evaluation_error")
	}
	if envelope.Error.Code != "variable_redefinition" {
		t.Errorf("error.code = %q, want %q", envelope.Error.Code, "variable_redefinition")
	}
	if envelope.Error.Line == 0 {
		t.Error("error.line should be non-zero for variable_redefinition")
	}
	if !strings.Contains(envelope.Error.Message, "reassign") {
		t.Errorf("error.message should mention reassignment, got %q", envelope.Error.Message)
	}
}

// TestJSONError_UndefinedVariableProducesJSON verifies that undefined variable
// errors also produce JSON on stdout when --format json is active.
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

	var envelope jsonErrorEnvelopeTest
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if envelope.Error.Type == "" {
		t.Error("error.type should not be empty")
	}
}

// TestJSONError_PipedStdinErrorProducesJSON verifies that the piped-stdin
// path (cm --format json, without eval subcommand) also produces JSON errors.
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

	var envelope jsonErrorEnvelopeTest
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if envelope.Error.Type == "" {
		t.Error("error.type should not be empty")
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

	// stdout should be empty (default text format, error only on stderr)
	if stdout.Len() > 0 {
		t.Errorf("stdout should be empty for text format errors, got: %q", stdout.String())
	}

	// stderr should have the error
	if !strings.Contains(stderr.String(), "Error") {
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
