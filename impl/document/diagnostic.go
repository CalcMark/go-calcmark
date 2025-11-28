package document

import (
	"strings"

	"github.com/CalcMark/go-calcmark/spec/lexer"
	"github.com/CalcMark/go-calcmark/spec/parser"
)

// DiagnosticSeverity indicates the severity of a diagnostic.
type DiagnosticSeverity int

const (
	// Warning indicates a potential problem that doesn't prevent evaluation.
	Warning DiagnosticSeverity = iota
	// Error indicates a problem that prevents evaluation.
	Error
)

func (s DiagnosticSeverity) String() string {
	switch s {
	case Warning:
		return "warning"
	case Error:
		return "error"
	default:
		return "unknown"
	}
}

// BlockDiagnostic represents a warning or error for a specific line in a block.
type BlockDiagnostic struct {
	BlockID  string             // ID of the block containing the issue
	Line     int                // Line number within block (1-indexed)
	Severity DiagnosticSeverity // Warning or Error
	Code     string             // Diagnostic code (e.g., "LIKELY_CALCULATION")
	Message  string             // Human-readable message
	Source   string             // The problematic line content
}

// Diagnostic codes
const (
	// DiagLikelyCalculation indicates a line that looks like an assignment
	// but failed to parse as a calculation.
	DiagLikelyCalculation = "LIKELY_CALCULATION"
)

// CalculationIndicator defines a pattern that suggests a line was intended
// to be a calculation. Each indicator has a check function and description.
type CalculationIndicator struct {
	Name        string                          // Short name for the indicator
	Description string                          // Human-readable description
	Check       func(tokens []lexer.Token) bool // Returns true if pattern matches
}

// calculationIndicators is the list of patterns that suggest a line was
// intended to be a calculation. This list can be extended with new patterns.
var calculationIndicators = []CalculationIndicator{
	{
		Name:        "assignment",
		Description: "Line starts with identifier = (assignment pattern)",
		Check: func(tokens []lexer.Token) bool {
			if len(tokens) < 2 {
				return false
			}
			return tokens[0].Type == lexer.IDENTIFIER && tokens[1].Type == lexer.ASSIGN
		},
	},
	// Future indicators can be added here:
	// - "starts_with_number": Line starts with a number (e.g., "100 meters +")
	// - "has_currency": Line contains currency symbol at start (e.g., "$100 +")
	// - "function_call": Line starts with identifier( (e.g., "sqrt(16 +")
}

// looksLikeFailedCalculation checks if a line appears to be an intended
// calculation that failed to parse. This helps catch common user mistakes
// where they wrote something like "x = 10 #" thinking it's a calculation.
//
// The function checks against a list of CalculationIndicators. If any
// indicator matches AND the line fails to parse, it's considered a
// likely failed calculation.
//
// Returns (true, parseError) if it looks like a failed calculation.
// Returns (false, nil) if it's likely just text.
func looksLikeFailedCalculation(line string) (bool, error) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false, nil
	}

	// Try to tokenize the line
	lex := lexer.NewLexer(trimmed)
	tokens, lexErr := lex.Tokenize()
	if lexErr != nil {
		// Lexer failed - could be markdown with special chars
		// Check if it at least starts like an assignment (fallback heuristic)
		return startsLikeAssignment(trimmed), lexErr
	}

	// Filter meaningful tokens (exclude NEWLINE, EOF)
	meaningful := filterMeaningful(tokens)
	if len(meaningful) == 0 {
		return false, nil
	}

	// Check against all calculation indicators
	matchedIndicator := false
	for _, indicator := range calculationIndicators {
		if indicator.Check(meaningful) {
			matchedIndicator = true
			break
		}
	}

	if !matchedIndicator {
		return false, nil
	}

	// Matches an indicator - try to parse
	_, parseErr := parser.Parse(trimmed + "\n")
	if parseErr != nil {
		// Matches indicator but fails to parse = likely failed calculation
		return true, parseErr
	}

	// Parses fine - not a failed calculation
	return false, nil
}

// GetCalculationIndicators returns the list of patterns used to detect
// likely calculation attempts. This is useful for documentation and debugging.
func GetCalculationIndicators() []CalculationIndicator {
	return calculationIndicators
}

// startsLikeAssignment checks if a string starts with identifier = pattern
// using simple string matching (for when lexer fails).
func startsLikeAssignment(s string) bool {
	// Look for pattern: word followed by space(s) and =
	idx := strings.Index(s, "=")
	if idx <= 0 {
		return false
	}

	before := strings.TrimSpace(s[:idx])
	// Check if "before" looks like a single identifier (no spaces, starts with letter)
	if strings.Contains(before, " ") {
		return false
	}
	if len(before) == 0 {
		return false
	}

	first := rune(before[0])
	return (first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_'
}

// filterMeaningful returns tokens excluding NEWLINE and EOF.
func filterMeaningful(tokens []lexer.Token) []lexer.Token {
	result := make([]lexer.Token, 0, len(tokens))
	for _, t := range tokens {
		if t.Type != lexer.NEWLINE && t.Type != lexer.EOF {
			result = append(result, t)
		}
	}
	return result
}
