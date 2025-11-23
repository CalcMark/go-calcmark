package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// TestEvalStdin tests evaluation from stdin
func TestEvalStdin(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantText string
		wantErr  bool
	}{
		{
			name:     "simple expression",
			input:    "3 + 4 * 5\n",
			wantText: "= 23",
			wantErr:  false,
		},
		{
			name:     "assignment",
			input:    "x = 10\n",
			wantText: "= 10",
			wantErr:  false,
		},
		{
			name:     "multiple calculations",
			input:    "x = 10\ny = x + 5\n",
			wantText: "= 15",
			wantErr:  false,
		},
		{
			name:     "currency",
			input:    "price = 100 USD\n",
			wantText: "= USD100.00",
			wantErr:  false,
		},
		{
			name:    "undefined variable",
			input:   "result = undefined_var + 10\n",
			wantErr: true,
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

			// Run eval
			err := runEval([]string{})

			// Restore
			w.Close()
			os.Stdout = oldStdout
			os.Stdin = oldStdin

			// Read output
			var bufOut bytes.Buffer
			bufOut.ReadFrom(r)
			output := bufOut.String()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if !strings.Contains(output, tt.wantText) {
					t.Errorf("Output doesn't contain %q\nGot: %s", tt.wantText, output)
				}
			}
		})
	}
}

// TestEvalFile tests evaluation from file
func TestEvalFile(t *testing.T) {
	// Create temp file
	content := "x = 10\ny = x + 5\nz = y * 2\n"
	tmpfile, err := os.CreateTemp("", "test*.cm")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run eval
	runEval([]string{tmpfile.Name()})

	w.Close()
	os.Stdout = oldStdout

	// Read output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Check for expected values
	if !strings.Contains(output, "x = 10") {
		t.Errorf("Missing 'x = 10' in output\nGot: %s", output)
	}
	if !strings.Contains(output, "y = x + 5") {
		t.Errorf("Missing 'y = x + 5' in output\nGot: %s", output)
	}
	if !strings.Contains(output, "= 30") {
		t.Errorf("Missing '= 30' (z result) in output\nGot: %s", output)
	}
}

// TestEvalJSON tests JSON output
func TestEvalJSON(t *testing.T) {
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

	// Run eval with --json
	runEval([]string{"--json"})

	w.Close()
	os.Stdout = oldStdout
	os.Stdin = oldStdin

	// Read output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Check for JSON structure
	if !strings.Contains(output, `"type"`) {
		t.Errorf("Missing JSON type field")
	}
	if !strings.Contains(output, `"value"`) {
		t.Errorf("Missing JSON value field")
	}
	if !strings.Contains(output, `"10"`) {
		t.Errorf("Missing value 10 in JSON")
	}
}

// TestLoadAndEvaluate tests file loading helper
func TestLoadAndEvaluate(t *testing.T) {
	// Create temp file
	content := "x = 42\n"
	tmpfile, err := os.CreateTemp("", "test*.cm")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	tmpfile.Write([]byte(content))
	tmpfile.Close()

	// Load and evaluate
	doc, err := loadAndEvaluate(tmpfile.Name())
	if err != nil {
		t.Fatalf("loadAndEvaluate failed: %v", err)
	}

	// Check document was evaluated
	blocks := doc.GetBlocks()
	if len(blocks) == 0 {
		t.Fatal("No blocks in document")
	}
}

// TestValidateFilePath tests security validation
func TestValidateFilePath(t *testing.T) {
	// Create test fixture
	os.MkdirAll("testdata", 0755)
	testFile := "testdata/test.cm"
	os.WriteFile(testFile, []byte("x = 1\n"), 0644)
	defer os.RemoveAll("testdata")

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"valid file", testFile, false},
		{"path traversal", "../../../etc/passwd", true},
		{"wrong extension", "eval_test.go", true},
		{"non-existent", "nonexistent.cm", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFilePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFilePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
