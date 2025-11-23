package calcmark

import (
	"github.com/CalcMark/go-calcmark/impl/interpreter"
)

// Session maintains state for live editor use.
// Variables persist across Eval calls within the same session.
type Session struct {
	env *interpreter.Environment
}

// NewSession creates a new stateful evaluation session.
// Variables are preserved across Eval calls.
//
// Example:
//
//	session := calcmark.NewSession()
//	session.Eval("x = 10")
//	result, _ := session.Eval("x + 5")
//	fmt.Println(result.Value) // 15
func NewSession() *Session {
	return &Session{
		env: interpreter.NewEnvironment(),
	}
}

// Eval evaluates an expression in this session's context.
// Variables are preserved across calls.
func (s *Session) Eval(input string) (*Result, error) {
	return evaluate(input, s.env)
}

// Reset clears all variables in this session.
func (s *Session) Reset() {
	s.env = interpreter.NewEnvironment()
}

// GetVariable retrieves a variable value by name.
// Returns the value and true if found, nil and false otherwise.
func (s *Session) GetVariable(name string) (interface{}, bool) {
	return s.env.Get(name)
}
