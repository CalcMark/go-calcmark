package interpreter_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	implDoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/CalcMark/go-calcmark/spec/parser"
)

// TestEvalFilesEvaluate tests that eval/success files not only parse but also evaluate successfully.
// This ensures the eval files represent valid CalcMark that works end-to-end.
// Uses the document evaluator to properly handle CalcMark files with markdown blocks.
func TestEvalFilesEvaluate(t *testing.T) {
	evalDir := "../../testdata/eval/success/features"

	files, err := os.ReadDir(evalDir)
	if err != nil {
		t.Fatalf("failed to read eval dir: %v", err)
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".cm") {
			continue
		}

		t.Run(file.Name(), func(t *testing.T) {
			path := filepath.Join(evalDir, file.Name())
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}

			// Create document (handles CalcMark with markdown blocks)
			doc, docErr := document.NewDocument(string(content))
			if docErr != nil {
				t.Fatalf("Document parse failed: %v", docErr)
			}

			// Evaluate using document evaluator
			eval := implDoc.NewEvaluator()
			evalErr := eval.Evaluate(doc)
			if evalErr != nil {
				t.Errorf("Eval failed: %v", evalErr)
			}
		})
	}
}

// TestEvalErrorFilesShouldFailEval tests that eval/errors files
// fail at eval time. These files document what CalcMark parses but cannot evaluate.
// These are semantic/runtime errors, not syntax errors.
func TestEvalErrorFilesShouldFailEval(t *testing.T) {
	// Test specific invalid expressions that PARSE successfully but FAIL at eval
	// These are the semantic errors documented in eval/errors/features/*.cm files
	tests := []struct {
		name  string
		input string
	}{
		// From incompatible_units.cm - mismatched arbitrary units
		{"mismatched arbitrary units", "5 apples + 3 oranges\n"},
		{"different arbitrary units", "10 widgets + 5 items\n"},

		// From logical_operators.cm - type mismatches
		{"number and boolean", "5 and true\n"},
		{"boolean or number", "true or 5\n"},
		{"not number", "not 5\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, parseErr := parser.Parse(tt.input)
			if parseErr != nil {
				// Parse error is acceptable - documents invalid syntax
				t.Logf("Parse error (expected): %v", parseErr)
				return
			}

			interp := interpreter.NewInterpreter()
			_, evalErr := interp.Eval(nodes)
			if evalErr == nil {
				t.Errorf("Expected eval to fail for %q but it succeeded", tt.input)
			} else {
				t.Logf("Eval error (expected): %v", evalErr)
			}
		})
	}
}

// TestEvalErrorFilesDocumentLevel tests that eval/errors .cm files with frontmatter
// produce errors at either the document parse or eval stage.
// This complements TestEvalErrorFilesShouldFailEval which uses parser.Parse()
// (no frontmatter support) for simple expression-level errors.
func TestEvalErrorFilesDocumentLevel(t *testing.T) {
	errorDir := "../../testdata/eval/errors/features"

	files, err := os.ReadDir(errorDir)
	if err != nil {
		t.Fatalf("failed to read error dir: %v", err)
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".cm") {
			continue
		}

		t.Run(file.Name(), func(t *testing.T) {
			path := filepath.Join(errorDir, file.Name())
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}

			// Try document-level parsing (handles frontmatter)
			doc, docErr := document.NewDocument(string(content))
			if docErr != nil {
				// Document parse error is expected for invalid frontmatter
				t.Logf("Document parse error (expected): %v", docErr)
				return
			}

			// Document parsed OK — eval should fail
			eval := implDoc.NewEvaluator()
			evalErr := eval.Evaluate(doc)
			if evalErr != nil {
				t.Logf("Eval error (expected): %v", evalErr)
				return
			}

			t.Errorf("Expected error for %s but document parsed and evaluated successfully", file.Name())
		})
	}
}
