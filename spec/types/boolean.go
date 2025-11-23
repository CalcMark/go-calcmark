package types

// Boolean represents a boolean value (true/false).
// CalcMark supports multiple boolean representations: true/false, yes/no, t/f, y/n.
type Boolean struct {
	Value bool
}

// NewBoolean creates a new Boolean with the given value.
func NewBoolean(value bool) *Boolean {
	return &Boolean{Value: value}
}

// String returns "true" or "false".
func (b *Boolean) String() string {
	if b.Value {
		return "true"
	}
	return "false"
}
