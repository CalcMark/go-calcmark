// Package parser provides the CalcMark parser implementation.
//
// The parser converts tokenized input into an Abstract Syntax Tree (AST)
// that represents the structure of CalcMark programs. It uses a recursive
// descent parsing strategy with operator precedence climbing.
//
// # Architecture
//
// The parser consists of two main components:
//
//  1. Recursive Descent Parser (recursive_descent.go): Implements the core
//     parsing logic with support for expressions, operators, and precedence.
//
//  2. Adapter (adapter.go): Provides a simple public API that wraps the
//     recursive descent parser.
//
// # Usage
//
// Basic parsing:
//
//	nodes, err := parser.Parse("x = 5 + 3\n")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Operator Precedence
//
// From highest to lowest:
//  1. Parentheses ()
//  2. Exponentiation ^ (right-associative)
//  3. Unary -, + (prefix)
//  4. Multiplicative *, /, % (left-associative)
//  5. Additive +, - (left-associative)
//  6. Comparison >, <, >=, <=, ==, != (non-associative)
//
// # Grammar
//
// See calcmark.bnf for the complete EBNF grammar specification.
//
// # Error Handling
//
// Parse errors include position information (line and column) to help
// with error reporting and IDE integration:
//
//	parse error at line 1, column 5: expected '(' after function name
//
// # Performance
//
// The parser is designed for speed and typically completes in microseconds:
//   - Simple expressions: ~1-2μs
//   - Complex expressions: ~5μs
//   - Multi-line programs: ~50μs
//
// See benchmark_test.go for detailed performance metrics.
package parser
