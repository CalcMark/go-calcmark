package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	implDoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/CalcMark/go-calcmark/spec/parser"
)

// testdataRoot is the path to testdata directory relative to this test file
const testdataRoot = "../../testdata"

// =============================================================================
// SPEC/VALID TESTS - Files that must parse successfully
// =============================================================================

// TestGoldenSpecValidFeatures ensures all files in testdata/spec/valid/features parse correctly.
// These files define valid CalcMark feature syntax and serve as the canonical reference.
func TestGoldenSpecValidFeatures(t *testing.T) {
	runParseSuccessTests(t, filepath.Join(testdataRoot, "spec/valid/features"))
}

// TestGoldenSpecValidExpressions ensures all files in testdata/spec/valid/expressions parse correctly.
func TestGoldenSpecValidExpressions(t *testing.T) {
	runParseSuccessTests(t, filepath.Join(testdataRoot, "spec/valid/expressions"))
}

// TestGoldenSpecValidDocuments ensures all files in testdata/spec/valid/documents parse correctly.
func TestGoldenSpecValidDocuments(t *testing.T) {
	runParseSuccessTests(t, filepath.Join(testdataRoot, "spec/valid/documents"))
}

// =============================================================================
// SPEC/INVALID TESTS - Files that must fail to parse
// =============================================================================

// TestGoldenSpecInvalidSyntax ensures all files in testdata/spec/invalid/syntax fail to parse.
// Each file contains exactly one invalid expression that should be rejected by the parser.
func TestGoldenSpecInvalidSyntax(t *testing.T) {
	runParseFailureTests(t, filepath.Join(testdataRoot, "spec/invalid/syntax"))
}

// TestGoldenSpecInvalidFeatures ensures all files in testdata/spec/invalid/features fail to parse.
func TestGoldenSpecInvalidFeatures(t *testing.T) {
	runParseFailureTests(t, filepath.Join(testdataRoot, "spec/invalid/features"))
}

// =============================================================================
// EVAL/SUCCESS TESTS - Files that must parse AND evaluate successfully
// =============================================================================

// TestGoldenEvalSuccessFeatures ensures all files in testdata/eval/success/features
// both parse and evaluate successfully. These files document working CalcMark behavior.
func TestGoldenEvalSuccessFeatures(t *testing.T) {
	runEvalSuccessTests(t, filepath.Join(testdataRoot, "eval/success/features"))
}

// =============================================================================
// EVAL/ERRORS TESTS - Files that parse but fail at evaluation
// =============================================================================

// TestGoldenEvalErrorsFeatures ensures all files in testdata/eval/errors/features
// parse successfully but fail during evaluation (semantic/runtime errors).
func TestGoldenEvalErrorsFeatures(t *testing.T) {
	runEvalFailureTests(t, filepath.Join(testdataRoot, "eval/errors/features"))
}

// =============================================================================
// EXPECTED VALUE TESTS - Verify specific expected outputs from golden files
// =============================================================================

// TestGoldenExpectedValues parses expected values from comments in eval/success files
// and verifies the interpreter produces the correct results.
func TestGoldenExpectedValues(t *testing.T) {
	evalDir := filepath.Join(testdataRoot, "eval/success/features")

	files, err := os.ReadDir(evalDir)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("eval/success/features directory not found")
		}
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

			expectations := parseExpectedValues(string(content))
			if len(expectations) == 0 {
				// No explicit expectations - just verify it evaluates
				t.Logf("No expected values found, skipping value verification")
				return
			}

			// Parse and evaluate
			doc, err := document.NewDocument(string(content))
			if err != nil {
				t.Fatalf("Document parse failed: %v", err)
			}

			eval := implDoc.NewEvaluator()
			if err := eval.Evaluate(doc); err != nil {
				t.Fatalf("Evaluation failed: %v", err)
			}

			// Check each expectation
			env := eval.GetEnvironment()
			for varName, expected := range expectations {
				val, ok := env.Get(varName)
				if !ok {
					t.Errorf("Variable %q not found in environment", varName)
					continue
				}

				// Get string representation of the value
				actual := val.String()

				// Check if expected value appears in the actual result
				if !containsExpectedValue(actual, expected) {
					t.Errorf("Variable %q: expected %q, got %q", varName, expected, actual)
				} else {
					t.Logf("✓ %s = %s (expected: %s)", varName, actual, expected)
				}
			}
		})
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// runParseSuccessTests runs tests expecting all .cm files in dir to parse successfully.
func runParseSuccessTests(t *testing.T, dir string) {
	t.Helper()

	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skipf("Directory not found: %s", dir)
		}
		t.Fatalf("failed to read dir %s: %v", dir, err)
	}

	cmFiles := filterCMFiles(files)
	if len(cmFiles) == 0 {
		t.Skipf("No .cm files in %s", dir)
	}

	for _, file := range cmFiles {
		t.Run(file.Name(), func(t *testing.T) {
			path := filepath.Join(dir, file.Name())
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}

			// Use document parser for full CalcMark files (handles markdown)
			_, err = document.NewDocument(string(content))
			if err != nil {
				t.Errorf("Parse failed (expected success): %v", err)
			} else {
				t.Logf("✓ Parsed successfully: %s", file.Name())
			}
		})
	}
}

// runParseFailureTests runs tests expecting all .cm files in dir to fail parsing.
func runParseFailureTests(t *testing.T, dir string) {
	t.Helper()

	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skipf("Directory not found: %s", dir)
		}
		t.Fatalf("failed to read dir %s: %v", dir, err)
	}

	cmFiles := filterCMFiles(files)
	if len(cmFiles) == 0 {
		t.Skipf("No .cm files in %s", dir)
	}

	for _, file := range cmFiles {
		t.Run(file.Name(), func(t *testing.T) {
			path := filepath.Join(dir, file.Name())
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}

			// Use direct parser (not document) for single invalid expressions
			// Invalid syntax files contain only the invalid expression
			_, err = parser.Parse(string(content))
			if err == nil {
				t.Errorf("Parse succeeded (expected failure) for: %q", strings.TrimSpace(string(content)))
			} else {
				t.Logf("✓ Parse correctly failed: %v", err)
			}
		})
	}
}

// runEvalSuccessTests runs tests expecting all .cm files to parse AND evaluate successfully.
func runEvalSuccessTests(t *testing.T, dir string) {
	t.Helper()

	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skipf("Directory not found: %s", dir)
		}
		t.Fatalf("failed to read dir %s: %v", dir, err)
	}

	cmFiles := filterCMFiles(files)
	if len(cmFiles) == 0 {
		t.Skipf("No .cm files in %s", dir)
	}

	for _, file := range cmFiles {
		t.Run(file.Name(), func(t *testing.T) {
			path := filepath.Join(dir, file.Name())
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}

			// Parse as document
			doc, err := document.NewDocument(string(content))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			// Evaluate
			eval := implDoc.NewEvaluator()
			if err := eval.Evaluate(doc); err != nil {
				t.Errorf("Eval failed (expected success): %v", err)
			} else {
				t.Logf("✓ Evaluated successfully: %s", file.Name())
			}
		})
	}
}

// runEvalFailureTests runs tests expecting files to have evaluation errors.
// For documents with mixed markdown/calculations, we check that at least one
// calc block has an error (the document evaluates but individual blocks fail).
func runEvalFailureTests(t *testing.T, dir string) {
	t.Helper()

	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skipf("Directory not found: %s", dir)
		}
		t.Fatalf("failed to read dir %s: %v", dir, err)
	}

	cmFiles := filterCMFiles(files)
	if len(cmFiles) == 0 {
		t.Skipf("No .cm files in %s", dir)
	}

	for _, file := range cmFiles {
		t.Run(file.Name(), func(t *testing.T) {
			path := filepath.Join(dir, file.Name())
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}

			// Parse as document (handles markdown + calc blocks)
			doc, docErr := document.NewDocument(string(content))
			if docErr != nil {
				t.Logf("✓ Document parse failed (acceptable): %v", docErr)
				return
			}

			// Evaluate the document - this captures per-block errors
			eval := implDoc.NewEvaluator()
			_ = eval.Evaluate(doc) // Ignore top-level error

			// Check if any calc blocks have errors
			hasError := false
			blocks := doc.GetBlocks()
			for _, block := range blocks {
				if calcBlock, ok := block.Block.(*document.CalcBlock); ok {
					if calcBlock.Error() != nil {
						hasError = true
						t.Logf("✓ Calc block error: %v", calcBlock.Error())
					}
				}
			}

			if !hasError {
				// Fallback: try direct parse for single-expression files
				nodes, parseErr := parser.Parse(string(content))
				if parseErr != nil {
					t.Logf("✓ Direct parse failed: %v", parseErr)
					return
				}

				interp := interpreter.NewInterpreter()
				_, evalErr := interp.Eval(nodes)
				if evalErr != nil {
					t.Logf("✓ Direct eval failed: %v", evalErr)
					return
				}

				t.Errorf("No errors found in file (expected at least one): %s", file.Name())
			}
		})
	}
}

// filterCMFiles returns only .cm files from a directory listing.
func filterCMFiles(files []os.DirEntry) []os.DirEntry {
	var cmFiles []os.DirEntry
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".cm") {
			cmFiles = append(cmFiles, f)
		}
	}
	return cmFiles
}

// parseExpectedValues extracts expected values from comments in CalcMark files.
// Format: "varname = expression" followed by "# Expected: value"
// Returns map of variable name -> expected value string
func parseExpectedValues(content string) map[string]string {
	expectations := make(map[string]string)

	// Pattern to match: "varname = ..." followed by "# Expected: value"
	// We look for lines with assignments and the next comment with "Expected:"
	lines := strings.Split(content, "\n")

	var lastVarName string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for assignment: "varname = ..."
		if idx := strings.Index(trimmed, "="); idx > 0 && !strings.HasPrefix(trimmed, "#") {
			varPart := strings.TrimSpace(trimmed[:idx])
			// Validate it looks like an identifier (alphanumeric + underscore)
			if isValidIdentifier(varPart) {
				lastVarName = varPart
			}
		}

		// Check for expected comment: "# Expected: value"
		if strings.HasPrefix(trimmed, "#") && lastVarName != "" {
			// Look for "Expected:" pattern
			expectedPattern := regexp.MustCompile(`(?i)#\s*Expected:\s*(.+)`)
			if matches := expectedPattern.FindStringSubmatch(trimmed); len(matches) > 1 {
				expectedValue := strings.TrimSpace(matches[1])
				expectations[lastVarName] = expectedValue
				lastVarName = "" // Reset after capturing
			}
		}
	}

	return expectations
}

// isValidIdentifier checks if a string looks like a valid CalcMark identifier.
func isValidIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_') {
				return false
			}
		} else {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
				return false
			}
		}
	}
	return true
}

// containsExpectedValue checks if actual result contains the expected value.
// Handles numeric comparisons and unit matching.
func containsExpectedValue(actual, expected string) bool {
	// Direct substring match
	if strings.Contains(actual, expected) {
		return true
	}

	// Extract numeric value from expected (e.g., "5 disks" -> "5")
	numPattern := regexp.MustCompile(`^(\d+(?:\.\d+)?)`)
	if matches := numPattern.FindStringSubmatch(expected); len(matches) > 1 {
		expectedNum := matches[1]
		if strings.Contains(actual, expectedNum) {
			return true
		}
	}

	return false
}
