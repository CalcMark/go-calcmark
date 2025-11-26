package classifier

import (
	"testing"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
)

// TestFunctionCalls tests that standalone function calls are classified as calculations
func TestFunctionCallAvg(t *testing.T) {
	ctx := interpreter.NewEnvironment()

	tests := []struct {
		name string
		line string
		want LineType
	}{
		{"avg function", "avg(1, 2, 3)", Calculation},
		{"avg with spaces", "avg(1, 2, 3, 4, 5)", Calculation},
		{"avg single arg", "avg(42)", Calculation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyLine(tt.line, ctx)
			if got != tt.want {
				t.Errorf("ClassifyLine(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestFunctionCallAverageOf(t *testing.T) {
	ctx := interpreter.NewEnvironment()

	tests := []struct {
		name string
		line string
		want LineType
	}{
		{"average of basic", "average of 1, 2, 3", Calculation},
		{"average of many", "average of 1, 3, 5, 7, 9", Calculation},
		{"average of two", "average of 10, 20", Calculation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyLine(tt.line, ctx)
			if got != tt.want {
				t.Errorf("ClassifyLine(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestFunctionCallSqrt(t *testing.T) {
	ctx := interpreter.NewEnvironment()

	tests := []struct {
		name string
		line string
		want LineType
	}{
		{"sqrt function", "sqrt(16)", Calculation},
		{"sqrt decimal", "sqrt(2.5)", Calculation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyLine(tt.line, ctx)
			if got != tt.want {
				t.Errorf("ClassifyLine(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestFunctionCallSquareRootOf(t *testing.T) {
	ctx := interpreter.NewEnvironment()

	tests := []struct {
		name string
		line string
		want LineType
	}{
		{"square root of basic", "square root of 25", Calculation},
		{"square root of decimal", "square root of 100", Calculation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyLine(tt.line, ctx)
			if got != tt.want {
				t.Errorf("ClassifyLine(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestFunctionInAssignment(t *testing.T) {
	ctx := interpreter.NewEnvironment()

	tests := []struct {
		name string
		line string
		want LineType
	}{
		{"avg in assignment", "result = avg(1, 2, 3)", Calculation},
		{"average of in assignment", "mean = average of 10, 20, 30", Calculation},
		{"sqrt in assignment", "root = sqrt(16)", Calculation},
		{"square root of in assignment", "val = square root of 25", Calculation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyLine(tt.line, ctx)
			if got != tt.want {
				t.Errorf("ClassifyLine(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestFunctionWithVariables(t *testing.T) {
	ctx := interpreter.NewEnvironment()
	// Set up some variables
	interpreter.Evaluate("x = 5\ny = 10\nz = 15", ctx)

	tests := []struct {
		name string
		line string
		want LineType
	}{
		{"avg with variables", "avg(x, y, z)", Calculation},
		{"average of with variables", "average of x, y, z", Calculation},
		{"sqrt with variable", "sqrt(x)", Calculation},
		{"square root of with variable", "square root of y", Calculation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyLine(tt.line, ctx)
			if got != tt.want {
				t.Errorf("ClassifyLine(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}
