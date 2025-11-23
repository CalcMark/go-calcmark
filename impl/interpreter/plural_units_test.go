package interpreter_test

import (
	"testing"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/parser"
)

// TestPluralUnitMismatch documents that plural forms are treated as different units
func TestPluralUnitMismatch(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"goose vs geese", "1 goose + 2 geese\n"},
		{"cat vs cats", "5 cat + 3 cats\n"},
		{"person vs people", "10 person + 5 people\n"},
		{"mouse vs mice", "2 mouse + 3 mice\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				// Parse succeeded, continue to eval
				t.Logf("Parse succeeded for %q", tt.input)
			}

			interp := interpreter.NewInterpreter()
			_, err = interp.Eval(nodes)

			if err == nil {
				t.Errorf("Expected error for mismatched units %q but got none", tt.input)
			} else {
				t.Logf("Correctly errored: %v", err)
			}
		})
	}
}
