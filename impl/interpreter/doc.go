// Package interpreter provides the CalcMark expression interpreter.
//
// The interpreter executes validated Abstract Syntax Trees (ASTs) and
// computes results while maintaining variable bindings in an environment.
//
// # Architecture
//
// The interpreter follows a tree-walking pattern:
//  1. Traverse the AST nodes
//  2. Evaluate each node recursively
//  3. Store variable assignments in environment
//  4. Return computed values as typed results
//
// # Usage
//
// Basic interpretation:
//
//	env := interpreter.NewEnvironment()
//	interp := interpreter.NewInterpreterWithEnv(env)
//
//	nodes, _ := parser.Parse("x = 5\ny = x + 3\n")
//	results, err := interp.Eval(nodes)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for _, result := range results {
//	    fmt.Println(result.String())
//	}
//
// # Environment
//
// The environment tracks variable bindings:
//
//	env := interpreter.NewEnvironment()
//	env.Set("x", types.NewNumber(decimal.NewFromInt(5)))
//	value, exists := env.Get("x")
//
// Variables persist across evaluations when using the same environment,
// enabling line-by-line evaluation in interactive editors.
//
// # Type System Integration
//
// The interpreter produces typed values:
//   - types.Number: Arbitrary-precision decimals
//   - types.Currency: Monetary values with symbols
//   - types.Quantity: Physical quantities with units
//   - types.Date: Calendar dates
//   - types.Time: Time of day
//   - types.Duration: Time intervals
//   - types.Boolean: True/false values
//
// # Operators
//
// Supported operations:
//   - Arithmetic: +, -, *, /, %, ^
//   - Comparison: ==, !=, <, >, <=, >=
//   - Unary: -, +
//
// # Functions
//
// Built-in functions:
//   - avg(...): Average of values (variadic)
//   - sqrt(x): Square root
//
// # Unit Handling
//
// The interpreter implements special semantics for units:
//
// Binary operations (first-unit-wins):
//
//	5 kg + 10 lb    → ~9.54 kg (converts lb to kg, keeps first unit)
//	$100 + €50      → ~$155.00 (converts EUR to USD based on rates)
//	10 m + 5 ft     → ~11.52 m (converts ft to m)
//
// Functions (drop mixed units):
//
//	avg($100, $200)      → $150.00 (same unit preserved)
//	avg($100, €200)      → 150 (mixed units → dimensionless number)
//	avg(5 kg, 10 kg)     → 7.5 kg (same unit preserved)
//	avg(5 kg, 10 lb)     →  (error: incompatible units)
//
// # Error Handling
//
// Runtime errors include:
//   - Undefined variables
//   - Type mismatches
//   - Division by zero
//   - Incompatible units
//
// # Performance
//
// The interpreter is designed for interactive use and completes
// evaluations in microseconds. See benchmark tests for metrics.
package interpreter
