package lexer

// Security limits to prevent DOS attacks
const (
	// MaxIdentifierLength limits identifier length
	// Example: prevents extremely long variable names
	MaxIdentifierLength = 256

	// MaxNumberLength limits number literal length
	// Example: prevents pathological number parsing
	MaxNumberLength = 100
)

// LexerSecurityError represents a security limit violation in the lexer
type LexerSecurityError struct {
	Message string
	Limit   string
	Actual  int
}

func (e *LexerSecurityError) Error() string {
	return e.Message
}
