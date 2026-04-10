// Package classifier implements line classification for CalcMark
package classifier

import (
	"strings"
	"sync"
	"unicode"

	"github.com/CalcMark/go-calcmark/constants"
	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/features"
	"github.com/CalcMark/go-calcmark/spec/lexer"
	"github.com/CalcMark/go-calcmark/spec/parser"
)

// nlTriggerKeywords is the set of NL function trigger keywords, derived from
// the feature registry. Computed once on first use.
var (
	nlTriggerOnce     sync.Once
	nlTriggerKeywords map[string]bool
)

func getNLTriggerKeywords() map[string]bool {
	nlTriggerOnce.Do(func() {
		r := features.DefaultRegistry()
		nlTriggerKeywords = make(map[string]bool)
		for _, kw := range r.NLTriggerKeywords() {
			nlTriggerKeywords[kw] = true
		}
	})
	return nlTriggerKeywords
}

// IdentifierResolver checks whether a variable name is defined in the current
// evaluation context. This interface decouples the spec/classifier package
// from impl/interpreter, preserving the spec → impl dependency direction.
// impl/interpreter.Environment satisfies this interface.
type IdentifierResolver interface {
	Has(name string) bool
}

// emptyResolver is an IdentifierResolver that reports no identifiers as defined.
type emptyResolver struct{}

func (emptyResolver) Has(string) bool { return false }

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

// containsFunctions checks if token list contains function calls
func containsFunctions(tokens []lexer.Token) bool {
	functionTypes := map[lexer.TokenType]bool{
		lexer.FUNC_AVG:            true,
		lexer.FUNC_SQRT:           true,
		lexer.FUNC_SUM:            true,
		lexer.FUNC_NUMBER:         true,
		lexer.FUNC_AVERAGE_OF:     true,
		lexer.FUNC_SQUARE_ROOT_OF: true,
		lexer.FUNC_SUM_OF:         true,
	}

	triggers := getNLTriggerKeywords()

	for _, token := range tokens {
		if functionTypes[token.Type] {
			return true
		}
		// NL functions use IDENTIFIER tokens (not reserved keywords).
		// The trigger set is derived from the feature registry.
		if token.Type == lexer.IDENTIFIER {
			if triggers[strings.ToLower(string(token.Value))] {
				return true
			}
		}
	}
	return false
}

// containsSpecialKeywords checks if token list contains special expression keywords
// like IN (unit conversion), PER (rates), AS/NAPKIN (formatting), or OF (percentage)
func containsSpecialKeywords(tokens []lexer.Token) bool {
	specialTypes := map[lexer.TokenType]bool{
		lexer.IN:     true, // "10 meters in feet"
		lexer.OF:     true, // "10% of 200"
		lexer.PER:    true, // "50 dollars per hour"
		lexer.AS:     true, // "1234567 as napkin"
		lexer.NAPKIN: true, // napkin keyword
	}

	for _, token := range tokens {
		if specialTypes[token.Type] {
			return true
		}
	}
	return false
}

// containsDateKeywords checks if token list contains date-related keywords or duration literals.
func containsDateKeywords(tokens []lexer.Token) bool {
	for _, token := range tokens {
		switch token.Type {
		case lexer.DATE_TODAY, lexer.DATE_TOMORROW, lexer.DATE_YESTERDAY,
			lexer.DATE_THIS_WEEK, lexer.DATE_THIS_MONTH, lexer.DATE_THIS_YEAR,
			lexer.DATE_NEXT_WEEK, lexer.DATE_NEXT_MONTH, lexer.DATE_NEXT_YEAR,
			lexer.DATE_LAST_WEEK, lexer.DATE_LAST_MONTH, lexer.DATE_LAST_YEAR,
			lexer.DATE_WEEKDAY, lexer.DATE_THIS_WEEKDAY, lexer.DATE_NEXT_WEEKDAY, lexer.DATE_LAST_WEEKDAY,
			lexer.DATE_THIS_MONTH_NAME, lexer.DATE_NEXT_MONTH_NAME, lexer.DATE_LAST_MONTH_NAME,
			lexer.DATE_THIS_QUARTER, lexer.DATE_NEXT_QUARTER, lexer.DATE_LAST_QUARTER,
			lexer.DATE_THIS_FISCAL_QUARTER, lexer.DATE_NEXT_FISCAL_QUARTER, lexer.DATE_LAST_FISCAL_QUARTER,
			lexer.DATE_THIS_FISCAL_YEAR, lexer.DATE_NEXT_FISCAL_YEAR, lexer.DATE_LAST_FISCAL_YEAR,
			lexer.AGO, lexer.CALENDAR_QUARTER_LITERAL, lexer.FISCAL_QUARTER_LITERAL,
			lexer.FISCAL_YEAR_LITERAL, lexer.CALENDAR_YEAR_LITERAL,
			lexer.DATE_LITERAL, lexer.DURATION_LITERAL:
			return true
		}
	}
	return false
}

// containsDirective checks if token list contains a directive reference (@scale, @globals.x)
func containsDirective(tokens []lexer.Token) bool {
	for _, token := range tokens {
		if token.Type == lexer.AT_SIGN {
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
func allIdentifiersDefined(node ast.Node, env IdentifierResolver) bool {
	switch n := node.(type) {
	case *ast.Identifier:
		// Check if identifier exists in context (which handles boolean keywords)
		return env.Has(n.Name)

	case *ast.UnaryOp:
		return allIdentifiersDefined(n.Operand, env)

	case *ast.BinaryOp:
		return allIdentifiersDefined(n.Left, env) && allIdentifiersDefined(n.Right, env)

	case *ast.ComparisonOp:
		return allIdentifiersDefined(n.Left, env) && allIdentifiersDefined(n.Right, env)

	case *ast.Expression:
		return allIdentifiersDefined(n.Expr, env)

	case *ast.DirectiveRef:
		// Directives are always "defined" from the classifier's perspective.
		// Semantic validation (e.g., missing frontmatter) is the interpreter's job.
		return true

	case *ast.QuantityLiteral:
		// Recurse into expression-based quantities (e.g., "@scale meters")
		if n.Expr != nil {
			return allIdentifiersDefined(n.Expr, env)
		}
		return true

	default:
		// Literals and other nodes don't have identifiers
		return true
	}
}

// ClassifyLine classifies a line as CALCULATION, MARKDOWN, or BLANK.
// Returns an error for critical syntax errors (like inline octothorpe).
func ClassifyLine(line string, env IdentifierResolver) (LineType, error) {
	if env == nil {
		env = emptyResolver{}
	}

	// 1. Check empty/whitespace (per ENCODING_SPEC.md)
	if constants.IsBlankLine(line) {
		return Blank, nil
	}

	// 2. Check markdown prefixes (optimization)
	stripped := strings.TrimLeft(line, constants.Whitespace)

	// Check for markdown headers, quotes, and lists
	// Convert to runes for Unicode-safe indexing
	if len(stripped) > 0 {
		runes := []rune(stripped)
		firstChar := runes[0]

		if firstChar == '#' || firstChar == '>' {
			return Markdown, nil
		}

		// Check for markdown bullet lists: "- " or "* " (dash/asterisk followed by space)
		if (firstChar == '-' || firstChar == '*') && len(runes) > 1 && runes[1] == ' ' {
			return Markdown, nil
		}

		// Numbered list check: digit(s) followed by period and space (e.g., "1. ", "12. ")
		if unicode.IsDigit(firstChar) {
			// Find the first non-digit
			i := 0
			for i < len(runes) && unicode.IsDigit(runes[i]) {
				i++
			}
			// Check if it's followed by ". " (period + space)
			if i < len(runes)-1 && runes[i] == '.' && runes[i+1] == ' ' {
				return Markdown, nil
			}
		}
	}

	// 3. Try to tokenize
	lex := lexer.NewLexer(line)
	tokens, err := lex.Tokenize()
	if err != nil {
		// Check if this is a critical lexer error that should be propagated
		// (like octothorpe) rather than treated as Markdown
		if lexErr, ok := err.(*lexer.LexerError); ok {
			// Octothorpe errors are critical syntax errors, not ambiguous Markdown
			if strings.Contains(lexErr.Message, "octothorpe") || strings.Contains(lexErr.Message, "#") {
				return Markdown, err // Propagate the error
			}
		}
		// Other tokenization errors likely mean it's valid Markdown prose
		return Markdown, nil
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
		return Blank, nil
	}

	// 4. Check for assignment
	if containsAssignment(contentTokens) {
		nodes, err := parser.Parse(line)
		if err != nil {
			return Markdown, nil
		}

		// Must parse to exactly one statement for a single line
		if len(nodes) == 1 {
			if _, ok := nodes[0].(*ast.Assignment); ok {
				// Assignment statements are always calculations
				return Calculation, nil
			}
		}
	}

	// 5. Check for functions
	if containsFunctions(contentTokens) {
		nodes, err := parser.Parse(line)
		if err != nil {
			return Markdown, nil
		}

		// Must parse to exactly one statement
		if len(nodes) == 1 {
			return Calculation, nil
		}
	}

	// 5b. Check for special keywords (IN, PER, AS, NAPKIN, OF)
	// These indicate unit conversions, rates, napkin formatting, or percentage operations
	if containsSpecialKeywords(contentTokens) {
		nodes, err := parser.Parse(line)
		if err != nil {
			return Markdown, nil
		}

		// Must parse to exactly one statement
		if len(nodes) == 1 {
			return Calculation, nil
		}
	}

	// 5c. Check for date keywords (today, next Friday, this quarter, etc.)
	if containsDateKeywords(contentTokens) {
		nodes, err := parser.Parse(line)
		if err != nil {
			return Markdown, nil
		}

		// Must parse to exactly one statement
		if len(nodes) == 1 {
			return Calculation, nil
		}
	}

	// 5d. Check for directive references (@scale, @globals.x)
	if containsDirective(contentTokens) {
		nodes, err := parser.Parse(line)
		if err != nil {
			return Markdown, nil
		}

		// Must parse to exactly one statement
		if len(nodes) == 1 {
			return Calculation, nil
		}
	}

	// 6. Check for operators
	if containsOperators(contentTokens) {
		nodes, err := parser.Parse(line)
		if err != nil {
			return Markdown, nil
		}

		// Must parse to exactly one statement
		if len(nodes) != 1 {
			return Markdown, nil
		}

		// Verify all identifiers are defined
		if allIdentifiersDefined(nodes[0], env) {
			return Calculation, nil
		}
		return Markdown, nil
	}

	// 7. Single token cases
	if len(contentTokens) == 1 {
		token := contentTokens[0]

		// Try to parse - should result in exactly one statement
		nodes, err := parser.Parse(line)
		if err != nil {
			return Markdown, nil
		}
		if len(nodes) != 1 {
			return Markdown, nil
		}

		// Literals are always calculations
		if token.Type == lexer.NUMBER || token.Type == lexer.CURRENCY || token.Type == lexer.QUANTITY || token.Type == lexer.BOOLEAN || token.Type == lexer.FRACTION {
			return Calculation, nil
		}

		// Identifiers only if they exist in context or are boolean keywords
		if token.Type == lexer.IDENTIFIER {
			if env.Has(token.Value) {
				return Calculation, nil
			}
			return Markdown, nil
		}
	}

	// 8. Default: markdown
	return Markdown, nil
}
