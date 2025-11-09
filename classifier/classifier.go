// Package classifier implements line classification for CalcMark
package classifier

import (
	"strings"
	"unicode"

	"github.com/CalcMark/go-calcmark/ast"
	"github.com/CalcMark/go-calcmark/constants"
	"github.com/CalcMark/go-calcmark/evaluator"
	"github.com/CalcMark/go-calcmark/lexer"
	"github.com/CalcMark/go-calcmark/parser"
)

// LineType represents the type of a line in a CalcMark document
type LineType int

const (
	Calculation LineType = iota
	Markdown
	Blank
)

func (lt LineType) String() string {
	switch lt {
	case Calculation:
		return "CALCULATION"
	case Markdown:
		return "MARKDOWN"
	case Blank:
		return "BLANK"
	default:
		return "UNKNOWN"
	}
}

// containsOperators checks if token list contains arithmetic or comparison operators
func containsOperators(tokens []lexer.Token) bool {
	operatorTypes := map[lexer.TokenType]bool{
		lexer.PLUS:          true,
		lexer.MINUS:         true,
		lexer.MULTIPLY:      true,
		lexer.DIVIDE:        true,
		lexer.MODULUS:       true,
		lexer.EXPONENT:      true,
		lexer.GREATER_THAN:  true,
		lexer.LESS_THAN:     true,
		lexer.GREATER_EQUAL: true,
		lexer.LESS_EQUAL:    true,
		lexer.EQUAL:         true,
		lexer.NOT_EQUAL:     true,
	}

	for _, token := range tokens {
		if operatorTypes[token.Type] {
			return true
		}
	}
	return false
}

// containsAssignment checks if token list contains an assignment operator
func containsAssignment(tokens []lexer.Token) bool {
	for _, token := range tokens {
		if token.Type == lexer.ASSIGN {
			return true
		}
	}
	return false
}

// allIdentifiersDefined checks if all identifiers in an AST are defined in the context
func allIdentifiersDefined(node ast.Node, context *evaluator.Context) bool {
	switch n := node.(type) {
	case *ast.Identifier:
		// Check if identifier exists in context (which handles boolean keywords)
		return context.Has(n.Name)

	case *ast.UnaryOp:
		return allIdentifiersDefined(n.Operand, context)

	case *ast.BinaryOp:
		return allIdentifiersDefined(n.Left, context) && allIdentifiersDefined(n.Right, context)

	case *ast.ComparisonOp:
		return allIdentifiersDefined(n.Left, context) && allIdentifiersDefined(n.Right, context)

	case *ast.Expression:
		return allIdentifiersDefined(n.Expr, context)

	default:
		// Literals and other nodes don't have identifiers
		return true
	}
}

// ClassifyLine classifies a line as CALCULATION, MARKDOWN, or BLANK
func ClassifyLine(line string, context *evaluator.Context) LineType {
	if context == nil {
		context = evaluator.NewContext()
	}

	// 1. Check empty/whitespace
	if line == "" || strings.TrimSpace(line) == "" {
		return Blank
	}

	// 2. Check markdown prefixes (optimization)
	stripped := strings.TrimLeft(line, constants.Whitespace)

	// Check for markdown headers, quotes, and lists
	if len(stripped) > 0 {
		firstChar := stripped[0]
		if firstChar == '#' || firstChar == '>' {
			return Markdown
		}

		// Check for markdown bullet lists: "- " or "* " (dash/asterisk followed by space)
		if (firstChar == '-' || firstChar == '*') && len(stripped) > 1 && stripped[1:2] == constants.Space {
			return Markdown
		}

		// Numbered list check: digit(s) followed by period and space (e.g., "1. ", "12. ")
		if unicode.IsDigit(rune(firstChar)) {
			// Find the first non-digit
			i := 0
			for i < len(stripped) && unicode.IsDigit(rune(stripped[i])) {
				i++
			}
			// Check if it's followed by ". " (period + space)
			if i < len(stripped)-1 && len(stripped) >= i+2 &&
				stripped[i] == '.' && stripped[i+1:i+2] == constants.Space {
				return Markdown
			}
		}
	}

	// 3. Try to tokenize
	tokens, err := lexer.Tokenize(line)
	if err != nil {
		return Markdown
	}

	// Filter out NEWLINE and EOF tokens for analysis
	var contentTokens []lexer.Token
	for _, t := range tokens {
		if t.Type != lexer.NEWLINE && t.Type != lexer.EOF {
			contentTokens = append(contentTokens, t)
		}
	}

	// Empty content after tokenization
	if len(contentTokens) == 0 {
		return Blank
	}

	// 4. Check for assignment
	if containsAssignment(contentTokens) {
		nodes, err := parser.Parse(line)
		if err != nil {
			return Markdown
		}

		// Must parse to exactly one statement for a single line
		if len(nodes) == 1 {
			if _, ok := nodes[0].(*ast.Assignment); ok {
				// Assignment statements are always calculations
				return Calculation
			}
		}
	}

	// 5. Check for operators
	if containsOperators(contentTokens) {
		nodes, err := parser.Parse(line)
		if err != nil {
			return Markdown
		}

		// Must parse to exactly one statement
		if len(nodes) != 1 {
			return Markdown
		}

		// Verify all identifiers are defined
		if allIdentifiersDefined(nodes[0], context) {
			return Calculation
		}
		return Markdown
	}

	// 6. Single token cases
	if len(contentTokens) == 1 {
		token := contentTokens[0]

		// Try to parse - should result in exactly one statement
		nodes, err := parser.Parse(line)
		if err != nil {
			return Markdown
		}
		if len(nodes) != 1 {
			return Markdown
		}

		// Literals are always calculations
		if token.Type == lexer.NUMBER || token.Type == lexer.CURRENCY || token.Type == lexer.BOOLEAN {
			return Calculation
		}

		// Identifiers only if they exist in context or are boolean keywords
		if token.Type == lexer.IDENTIFIER {
			if context.Has(token.Value) {
				return Calculation
			}
			return Markdown
		}
	}

	// 7. Default: markdown
	return Markdown
}
