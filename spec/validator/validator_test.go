package validator

import (
	"testing"

	"github.com/CalcMark/go-calcmark/impl/evaluator"
	"github.com/CalcMark/go-calcmark/impl/types"
)

// TestValidCalculations tests validation of valid calculations
func TestSimpleLiteral(t *testing.T) {
	result := ValidateCalculation("42", nil)
	if !result.IsValid() {
		t.Error("expected valid result")
	}
	if len(result.Diagnostics) != 0 {
		t.Errorf("expected 0 diagnostics, got %d", len(result.Diagnostics))
	}
}

func TestSimpleAssignment(t *testing.T) {
	result := ValidateCalculation("x = 5", nil)
	if !result.IsValid() {
		t.Error("expected valid result")
	}
	if len(result.Diagnostics) != 0 {
		t.Errorf("expected 0 diagnostics, got %d", len(result.Diagnostics))
	}
}

func TestAssignmentWithExpression(t *testing.T) {
	result := ValidateCalculation("y = 10 + 5", nil)
	if !result.IsValid() {
		t.Error("expected valid result")
	}
}

func TestVariableReferenceWhenDefined(t *testing.T) {
	ctx := evaluator.NewContext()
	num, _ := types.NewNumber(5)
	ctx.Set("x", num)

	result := ValidateCalculation("y = x + 2", ctx)
	if !result.IsValid() {
		t.Error("expected valid result")
	}
}

func TestBooleanLiterals(t *testing.T) {
	values := []string{"true", "false"}
	for _, value := range values {
		result := ValidateCalculation("x = "+value, nil)
		if !result.IsValid() {
			t.Errorf("expected valid result for: %s", value)
		}
	}
}

// TestUndefinedVariables tests detection of undefined variables
func TestUndefinedVariableInExpression(t *testing.T) {
	result := ValidateCalculation("x + 2", nil)
	if result.IsValid() {
		t.Error("expected invalid result")
	}
	if !result.HasErrors() {
		t.Error("expected errors")
	}
	if len(result.Errors()) != 1 {
		t.Errorf("expected 1 error, got %d", len(result.Errors()))
	}
	if result.Errors()[0].Code != UndefinedVariable {
		t.Errorf("expected UndefinedVariable code")
	}
	if result.Errors()[0].VariableName != "x" {
		t.Errorf("expected variable name 'x', got '%s'", result.Errors()[0].VariableName)
	}
}

func TestUndefinedVariableInAssignmentRHS(t *testing.T) {
	result := ValidateCalculation("alice = bob + 2", nil)
	if result.IsValid() {
		t.Error("expected invalid result")
	}
	if len(result.Errors()) != 1 {
		t.Errorf("expected 1 error, got %d", len(result.Errors()))
	}
	if result.Errors()[0].VariableName != "bob" {
		t.Errorf("expected variable name 'bob', got '%s'", result.Errors()[0].VariableName)
	}
}

func TestMultipleUndefinedVariables(t *testing.T) {
	result := ValidateCalculation("result = foo + bar * baz", nil)
	if result.IsValid() {
		t.Error("expected invalid result")
	}
	if len(result.Errors()) != 3 {
		t.Errorf("expected 3 errors, got %d", len(result.Errors()))
	}

	varNames := make(map[string]bool)
	for _, err := range result.Errors() {
		varNames[err.VariableName] = true
	}

	expectedVars := map[string]bool{"foo": true, "bar": true, "baz": true}
	for name := range expectedVars {
		if !varNames[name] {
			t.Errorf("expected variable '%s' to be in errors", name)
		}
	}
}

func TestAssignmentLHSNotFlagged(t *testing.T) {
	result := ValidateCalculation("new_var = 42", nil)
	if !result.IsValid() {
		t.Error("expected valid result")
	}
}

func TestPositionInformation(t *testing.T) {
	result := ValidateCalculation("alice = bob + 2", nil)
	if len(result.Errors()) != 1 {
		t.Fatalf("expected 1 error, got %d", len(result.Errors()))
	}

	err := result.Errors()[0]
	if err.Range == nil {
		t.Fatal("expected range to be set")
	}

	// "bob" starts at column 9 (1-indexed: "alice = bob")
	if err.Range.Start.Line != 1 {
		t.Errorf("expected line 1, got %d", err.Range.Start.Line)
	}
	if err.Range.Start.Column != 9 {
		t.Errorf("expected column 9, got %d", err.Range.Start.Column)
	}
}

// TestContextFlow tests that context flows correctly
func TestVariableDefinedInContext(t *testing.T) {
	ctx := evaluator.NewContext()
	x, _ := types.NewNumber(10)
	y, _ := types.NewNumber(20)
	ctx.Set("x", x)
	ctx.Set("y", y)

	result := ValidateCalculation("z = x + y", ctx)
	if !result.IsValid() {
		t.Error("expected valid result")
	}
}

func TestUndefinedWithSomeDefined(t *testing.T) {
	ctx := evaluator.NewContext()
	x, _ := types.NewNumber(10)
	ctx.Set("x", x)

	result := ValidateCalculation("z = x + undefined", ctx)
	if result.IsValid() {
		t.Error("expected invalid result")
	}
	if len(result.Errors()) != 1 {
		t.Errorf("expected 1 error, got %d", len(result.Errors()))
	}
	if result.Errors()[0].VariableName != "undefined" {
		t.Errorf("expected variable name 'undefined', got '%s'", result.Errors()[0].VariableName)
	}
}

// TestValidationResultAPI tests ValidationResult convenience methods
func TestIsValidWithNoDiagnostics(t *testing.T) {
	result := ValidateCalculation("42", nil)
	if !result.IsValid() {
		t.Error("expected valid result")
	}
	if result.HasErrors() {
		t.Error("expected no errors")
	}
	if result.HasWarnings() {
		t.Error("expected no warnings")
	}
}

func TestIsValidWithErrors(t *testing.T) {
	result := ValidateCalculation("undefined_var", nil)
	if result.IsValid() {
		t.Error("expected invalid result")
	}
	if !result.HasErrors() {
		t.Error("expected errors")
	}
}

func TestErrorsProperty(t *testing.T) {
	result := ValidateCalculation("x + undefined", nil)
	if len(result.Errors()) != 2 {
		t.Errorf("expected 2 errors, got %d", len(result.Errors()))
	}
	for _, err := range result.Errors() {
		if err.Severity != Error {
			t.Error("expected Error severity")
		}
	}
}

func TestBoolConversion(t *testing.T) {
	if !ValidateCalculation("42", nil).Bool() {
		t.Error("expected true for valid calculation")
	}
	if ValidateCalculation("undefined", nil).Bool() {
		t.Error("expected false for invalid calculation")
	}
}

func TestStrRepresentation(t *testing.T) {
	result := ValidateCalculation("42", nil)
	str := result.String()
	if str != "Valid" {
		t.Errorf("expected 'Valid', got '%s'", str)
	}

	result = ValidateCalculation("undefined", nil)
	str = result.String()
	if str == "" {
		t.Error("expected non-empty string")
	}
}

// TestSyntaxErrors tests handling of syntax errors
func TestLexerError(t *testing.T) {
	// Invalid currency (no number after $)
	result := ValidateCalculation("x = $", nil)
	if result.IsValid() {
		t.Error("expected invalid result")
	}
	if result.Errors()[0].Code != SyntaxError {
		t.Errorf("expected SyntaxError code")
	}
}

func TestParserError(t *testing.T) {
	result := ValidateCalculation("x = ", nil)
	if result.IsValid() {
		t.Error("expected invalid result")
	}
	if result.Errors()[0].Code != SyntaxError {
		t.Errorf("expected SyntaxError code")
	}
}

func TestIncompleteExpression(t *testing.T) {
	result := ValidateCalculation("5 +", nil)
	if result.IsValid() {
		t.Error("expected invalid result")
	}
	if result.Errors()[0].Code != SyntaxError {
		t.Errorf("expected SyntaxError code")
	}
}

// TestDocumentValidation tests validation of multi-line documents
func TestValidDocument(t *testing.T) {
	document := `x = 5
y = 10
z = x + y`
	results := ValidateDocument(document, nil)

	// Check for errors (hints are OK)
	errorCount := 0
	for _, result := range results {
		if result.HasErrors() {
			errorCount++
		}
	}
	if errorCount != 0 {
		t.Errorf("expected 0 errors, got %d", errorCount)
	}
}

func TestDocumentWithUndefinedVariable(t *testing.T) {
	document := `x = 5
y = x + 2
z = unknown * 3`
	results := ValidateDocument(document, nil)

	// Line 3 should have error
	if _, exists := results[3]; !exists {
		t.Error("expected error on line 3")
	}
	if results[3].IsValid() {
		t.Error("expected invalid result for line 3")
	}
	if results[3].Errors()[0].VariableName != "unknown" {
		t.Errorf("expected variable name 'unknown', got '%s'", results[3].Errors()[0].VariableName)
	}
}

func TestContextFlowsBetweenLines(t *testing.T) {
	document := `x = 5
y = x + 2
z = y * 2`
	results := ValidateDocument(document, nil)

	// Check for errors (hints are OK)
	errorCount := 0
	for _, result := range results {
		if result.HasErrors() {
			errorCount++
		}
	}
	if errorCount != 0 {
		t.Errorf("expected 0 errors, got %d", errorCount)
	}
}

func TestMultipleErrorsDifferentLines(t *testing.T) {
	document := `x = 5
a = foo + 1
b = bar + 2`
	results := ValidateDocument(document, nil)

	if _, exists := results[2]; !exists {
		t.Error("expected error on line 2")
	}
	if _, exists := results[3]; !exists {
		t.Error("expected error on line 3")
	}
	if results[2].Errors()[0].VariableName != "foo" {
		t.Errorf("expected variable name 'foo', got '%s'", results[2].Errors()[0].VariableName)
	}
	if results[3].Errors()[0].VariableName != "bar" {
		t.Errorf("expected variable name 'bar', got '%s'", results[3].Errors()[0].VariableName)
	}
}

func TestBlankLinesIgnored(t *testing.T) {
	document := `x = 5

y = x + 2

z = y * 2`
	results := ValidateDocument(document, nil)
	if len(results) != 0 {
		t.Errorf("expected 0 errors, got %d", len(results))
	}
}

// TestComplexExpressions tests validation of complex expressions
func TestNestedBinaryOperations(t *testing.T) {
	result := ValidateCalculation("result = a + b * c - d", nil)
	if result.IsValid() {
		t.Error("expected invalid result")
	}
	if len(result.Errors()) != 4 {
		t.Errorf("expected 4 errors, got %d", len(result.Errors()))
	}

	varNames := make(map[string]bool)
	for _, err := range result.Errors() {
		varNames[err.VariableName] = true
	}

	expectedVars := map[string]bool{"a": true, "b": true, "c": true, "d": true}
	for name := range expectedVars {
		if !varNames[name] {
			t.Errorf("expected variable '%s' to be in errors", name)
		}
	}
}

func TestComparisonWithUndefined(t *testing.T) {
	result := ValidateCalculation("x > unknown", nil)
	if result.IsValid() {
		t.Error("expected invalid result")
	}
	if len(result.Errors()) != 2 {
		t.Errorf("expected 2 errors, got %d", len(result.Errors()))
	}

	varNames := make(map[string]bool)
	for _, err := range result.Errors() {
		varNames[err.VariableName] = true
	}

	if !varNames["x"] || !varNames["unknown"] {
		t.Error("expected both 'x' and 'unknown' in errors")
	}
}

func TestExponentWithUndefined(t *testing.T) {
	result := ValidateCalculation("base ^ exponent", nil)
	if result.IsValid() {
		t.Error("expected invalid result")
	}
	if len(result.Errors()) != 2 {
		t.Errorf("expected 2 errors, got %d", len(result.Errors()))
	}
}

// TestDiagnosticSerialization tests diagnostic serialization
func TestDiagnosticToDict(t *testing.T) {
	result := ValidateCalculation("alice = bob + 2", nil)
	if len(result.Errors()) != 1 {
		t.Fatalf("expected 1 error, got %d", len(result.Errors()))
	}

	diagMap := result.Errors()[0].ToMap()

	if diagMap["severity"] != "error" {
		t.Errorf("expected severity 'error', got '%s'", diagMap["severity"])
	}
	if diagMap["code"] != "undefined_variable" {
		t.Errorf("expected code 'undefined_variable', got '%s'", diagMap["code"])
	}
	msg := diagMap["message"].(string)
	if !contains(msg, "bob") {
		t.Errorf("expected message to contain 'bob', got '%s'", msg)
	}
	if diagMap["variable_name"] != "bob" {
		t.Errorf("expected variable_name 'bob', got '%s'", diagMap["variable_name"])
	}

	rangeMap := diagMap["range"].(map[string]any)
	startMap := rangeMap["start"].(map[string]int)
	if startMap["line"] != 1 {
		t.Errorf("expected line 1, got %d", startMap["line"])
	}
	if startMap["column"] != 9 {
		t.Errorf("expected column 9, got %d", startMap["column"])
	}
}

// TestRealWorldExamples tests validation with real-world calculation examples
func TestBudgetDocument(t *testing.T) {
	document := `salary = $5000
bonus = $500
rent = $1500
food = $800
utilities = $200
expenses = rent + food + utilities
savings = salary + bonus - expenses`

	results := ValidateDocument(document, nil)

	// Check for errors (hints are OK)
	errorCount := 0
	for _, result := range results {
		if result.HasErrors() {
			errorCount++
		}
	}
	if errorCount != 0 {
		t.Errorf("expected 0 errors, got %d", errorCount)
	}
}

func TestBudgetWithTypo(t *testing.T) {
	document := `salary = $5000
rent = $1500
savings = salry - rent`

	results := ValidateDocument(document, nil)
	if _, exists := results[3]; !exists {
		t.Error("expected error on line 3")
	}
	if results[3].Errors()[0].VariableName != "salry" {
		t.Errorf("expected variable name 'salry', got '%s'", results[3].Errors()[0].VariableName)
	}
}

func TestForwardReferenceError(t *testing.T) {
	document := `x = future_var + 2
future_var = 10`

	results := ValidateDocument(document, nil)
	if _, exists := results[1]; !exists {
		t.Error("expected error on line 1")
	}
	if results[1].Errors()[0].VariableName != "future_var" {
		t.Errorf("expected variable name 'future_var', got '%s'", results[1].Errors()[0].VariableName)
	}
}

// TestBlankLineIsolationHints tests explicit hint generation for missing blank lines
func TestBlankLineIsolationHints(t *testing.T) {
	tests := []struct {
		name          string
		document      string
		lineWithHints int
		expectBefore  bool
		expectAfter   bool
	}{
		{
			name: "calculation after text",
			document: `Some text
x = 5`,
			lineWithHints: 2,
			expectBefore:  true,
			expectAfter:   false, // end of document
		},
		{
			name: "calculation before text",
			document: `x = 5
Some text`,
			lineWithHints: 1,
			expectBefore:  false, // start of document
			expectAfter:   true,
		},
		{
			name: "calculation between text",
			document: `Top text
x = 5
Bottom text`,
			lineWithHints: 2,
			expectBefore:  true,
			expectAfter:   true,
		},
		{
			name: "properly isolated",
			document: `Text

x = 5

More text`,
			lineWithHints: -1, // No hints expected
			expectBefore:  false,
			expectAfter:   false,
		},
		{
			name: "multiple calculations no isolation",
			document: `x = 5
y = 10`,
			lineWithHints: 1, // First line should have hint about line after
			expectBefore:  false,
			expectAfter:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := ValidateDocument(tt.document, nil)

			if tt.lineWithHints == -1 {
				// Should have no hints anywhere
				for lineNum, result := range results {
					if result.HasHints() {
						t.Errorf("line %d: expected no hints, got %d", lineNum, len(result.Hints()))
					}
				}
				return
			}

			result, exists := results[tt.lineWithHints]
			if !exists {
				t.Fatalf("expected result for line %d", tt.lineWithHints)
			}

			hints := result.Hints()

			if tt.expectBefore && tt.expectAfter {
				if len(hints) != 2 {
					t.Errorf("expected 2 hints (before and after), got %d", len(hints))
				}
			} else if tt.expectBefore || tt.expectAfter {
				if len(hints) != 1 {
					t.Errorf("expected 1 hint, got %d", len(hints))
				}
			} else {
				if len(hints) != 0 {
					t.Errorf("expected 0 hints, got %d", len(hints))
				}
			}

			// Verify hints have correct code
			for _, hint := range hints {
				if hint.Code != BlankLineIsolation {
					t.Errorf("expected BlankLineIsolation code, got %s", hint.Code)
				}
				if hint.Severity != Hint {
					t.Errorf("expected Hint severity, got %s", hint.Severity)
				}
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestAmbiguousModulusHint tests that we get hints when % immediately follows a number
func TestAmbiguousModulusHint(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectHint  bool
		description string
	}{
		{
			name:        "Modulus with space before and after - no hint",
			input:       "10 % 3",
			expectHint:  false,
			description: "Clear modulus operator with spaces",
		},
		{
			name:        "Percentage literal - no hint",
			input:       "20%",
			expectHint:  false,
			description: "20% is a percentage literal (NUMBER token)",
		},
		{
			name:        "Percentage in expression - no hint",
			input:       "$100 * 20%",
			expectHint:  false,
			description: "20% in expression is clearly a percentage",
		},
		{
			name:        "Modulus without space after % - hint",
			input:       "10 %3",
			expectHint:  true,
			description: "Modulus with no space after % might confuse readers",
		},
		{
			name:        "Modulus in assignment without space after %",
			input:       "x = 10 %3",
			expectHint:  true,
			description: "Ambiguous: modulus should have space after %",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateCalculation(tt.input, nil)

			hasAmbiguousModulus := false
			for _, diag := range result.Diagnostics {
				if diag.Code == AmbiguousModulus {
					hasAmbiguousModulus = true
					if diag.Severity != Hint {
						t.Errorf("Expected Hint severity, got %v", diag.Severity)
					}
					t.Logf("✓ Hint: %s", diag.Message)
				}
			}

			if tt.expectHint && !hasAmbiguousModulus {
				t.Errorf("Expected ambiguous_modulus hint but got none.\nDiagnostics: %v", result.Diagnostics)
			}

			if !tt.expectHint && hasAmbiguousModulus {
				t.Errorf("Unexpected ambiguous_modulus hint.\nDiagnostics: %v", result.Diagnostics)
			}
		})
	}
}
