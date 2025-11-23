// Package semantic provides semantic analysis for CalcMark programs.
//
// The semantic checker validates CalcMark Abstract Syntax Trees (ASTs)
// for semantic correctness without executing them. It catches errors
// like undefined variables, type mismatches, and incompatible units.
//
// # Architecture
//
// The semantic checker operates in three phases:
//
//  1. Environment Setup: Tracks variable definitions and their types
//  2. AST Traversal: Visits each node and validates semantics
//  3. Diagnostic Collection: Accumulates errors, warnings, and hints
//
// # Usage
//
// Basic validation:
//
//	checker := semantic.NewChecker()
//	diagnostics := checker.Check(astNodes)
//
//	for _, diag := range diagnostics {
//	    if diag.Severity == semantic.Error {
//	        fmt.Printf("Error: %s\n", diag.Message)
//	    }
//	}
//
// With pre-populated environment:
//
//	checker := semantic.NewChecker()
//	checker.GetEnvironment().Set("x", types.NewNumber(decimal.NewFromInt(5)))
//	diagnostics := checker.Check(astNodes)
//
// # Diagnostic Codes
//
// The checker produces structured diagnostics with specific codes:
//
//   - DiagUndefinedVariable: Variable used before definition
//   - DiagIncompatibleUnits: Incompatible units in operation (e.g., "5 kg + 10 meters")
//   - DiagInvalidCurrency: Unknown currency code
//   - DiagTypeMismatch: Type error in operation
//   - DiagDivisionByZero: Division or modulus by zero
//
// # Severity Levels
//
//   - Error: Prevents evaluation, must be fixed
//   - Warning: Valid syntax but may cause runtime issues
//   - Hint: Style suggestions for improvement
//
// # Unit Validation
//
// The semantic checker validates unit compatibility:
//
//   - Compatible: "5 kg + 10 lb" (both mass)
//   - Incompatible: "5 kg + 10 meters" (mass + length)
//   - Currency: Validates against known ISO 4217 codes
//
// See unit_validation.go for detailed unit compatibility rules.
//
// # Performance
//
// Semantic checking is fast and typically completes in microseconds.
// It's designed to run on every keystroke in interactive editors.
package semantic
