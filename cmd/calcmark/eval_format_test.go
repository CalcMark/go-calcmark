package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// TestEvalWithFormat tests the new --format flag
func TestEvalWithFormat(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		format    string
		wantInOut string
	}{
		{
			name:      "text format (default)",
			input:     "x = 10\n",
			format:    "",
			wantInOut: "10",
		},
		{
			name:      "json format",
			input:     "x = 10\n",
			format:    "json",
			wantInOut: `"type"`,
		},
		{
			name:      "calcmark format",
			input:     "x = 10\n",
			format:    "cm",
			wantInOut: "x = 10",
		},
		{
			name:      "markdown format",
			input:     "x = 10\n",
			format:    "md",
			wantInOut: "```calcmark",
		},
		{
			name:      "html format",
			input:     "x = 10\n",
			format:    "html",
			wantInOut: "<!DOCTYPE html>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Create stdin
			oldStdin := os.Stdin
			rIn, wIn, _ := os.Pipe()
			os.Stdin = rIn
			wIn.Write([]byte(tt.input))
			wIn.Close()

			// Build args
			args := []string{}
			if tt.format != "" {
				args = append(args, "--format="+tt.format)
			}

			// Run eval
			err := runEval(args)

			// Restore
			w.Close()
			os.Stdout = oldStdout
			os.Stdin = oldStdin

			// Read output
			var bufOut bytes.Buffer
			bufOut.ReadFrom(r)
			output := bufOut.String()

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !strings.Contains(output, tt.wantInOut) {
				t.Errorf("Expected output to contain %q\nGot: %s", tt.wantInOut, output)
			}
		})
	}
}

// TestEvalWithVerbose tests the --verbose flag
func TestEvalWithVerbose(t *testing.T) {
	input := "x = 10 + 5\n"

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create stdin
	oldStdin := os.Stdin
	rIn, wIn, _ := os.Pipe()
	os.Stdin = rIn
	wIn.Write([]byte(input))
	wIn.Close()

	// Run eval with --verbose
	err := runEval([]string{"--verbose"})

	// Restore
	w.Close()
	os.Stdout = oldStdout
	os.Stdin = oldStdin

	// Read output
	var bufOut bytes.Buffer
	bufOut.ReadFrom(r)
	output := bufOut.String()

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verbose mode should show source
	if !strings.Contains(output, "x = 10 + 5") {
		t.Errorf("Expected verbose output to contain source\nGot: %s", output)
	}

	// Should also show result
	if !strings.Contains(output, "15") {
		t.Errorf("Expected output to contain result\nGot: %s", output)
	}
}

// TestEvalBackwardCompatibility ensures --json still works
func TestEvalBackwardCompatibility(t *testing.T) {
	input := "x = 10\n"

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create stdin
	oldStdin := os.Stdin
	rIn, wIn, _ := os.Pipe()
	os.Stdin = rIn
	wIn.Write([]byte(input))
	wIn.Close()

	// Run eval with old --json flag
	err := runEval([]string{"--json"})

	// Restore
	w.Close()
	os.Stdout = oldStdout
	os.Stdin = oldStdin

	// Read output
	var bufOut bytes.Buffer
	bufOut.ReadFrom(r)
	output := bufOut.String()

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Should still produce JSON
	if !strings.Contains(output, `"type"`) {
		t.Errorf("--json flag should still work\nGot: %s", output)
	}
}
