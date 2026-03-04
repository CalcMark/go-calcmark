package parser

// Security limits to prevent DOS attacks
const (
	// MaxNestingDepth limits expression nesting to prevent stack overflow
	// Example: (((((...))))) with 100+ levels
	MaxNestingDepth = 100

	// MaxTokenCount limits total tokens to prevent "token bomb" attacks
	// Example: x1+x2+x3+...+x10000
	MaxTokenCount = 10000

	// MaxCompoundPeriods limits the number of periods in growth functions
	// to prevent CPU exhaustion from large exponent calculations.
	// compound(x, y, 10000) is the maximum allowed.
	MaxCompoundPeriods = 10_000
)

// SecurityError represents a security limit violation
type SecurityError struct {
	Message string
	Limit   string
	Actual  int
}

func (e *SecurityError) Error() string {
	return e.Message
}
