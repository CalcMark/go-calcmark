package classifier

import (
	"testing"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/impl/types"
)

func TestFunctionCallClassification(t *testing.T) {
	tests := []struct {
		name string
		line string
		want LineType
	}{
		{name: "avg with numbers", line: "avg(1, 2, 3)", want: Calculation},
		{name: "sqrt with number", line: "sqrt(16)", want: Calculation},
		{name: "average of with numbers", line: "average of 1, 2, 3", want: Calculation},
		{name: "sum with numbers", line: "sum(1, 2, 3)", want: Calculation},
		{name: "sum of with numbers", line: "sum of 1, 2, 3", want: Calculation},
		{name: "sum of with currency", line: "sum of $100, $200", want: Calculation},
	}

	ctx := interpreter.NewEnvironment()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyLineTest(t, tt.line, ctx)
			if got != tt.want {
				t.Errorf("ClassifyLine(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestUnitConversionClassification(t *testing.T) {
	tests := []struct {
		name string
		line string
		want LineType
	}{
		{name: "meters in feet", line: "10 meters in feet", want: Calculation},
		{name: "cups in ounces", line: "2 cups in ounces", want: Calculation},
	}

	ctx := interpreter.NewEnvironment()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyLineTest(t, tt.line, ctx)
			if got != tt.want {
				t.Errorf("ClassifyLine(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestAsKeywordClassification(t *testing.T) {
	tests := []struct {
		name string
		line string
		want LineType
	}{
		{name: "as valid unit", line: "x as meters", want: Calculation},
		{name: "as currency code", line: "x as EUR", want: Calculation},
		{name: "as unknown identifier", line: "x as xyz", want: Calculation},
		{name: "as napkin", line: "x as napkin", want: Calculation},
		{name: "as precise", line: "x as precise", want: Calculation},
	}

	ctx := interpreter.NewEnvironment()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyLineTest(t, tt.line, ctx)
			if got != tt.want {
				t.Errorf("ClassifyLine(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestRateExpressionsClassification(t *testing.T) {
	tests := []struct {
		name string
		line string
		want LineType
	}{
		{name: "rate with slash", line: "100 MB/s", want: Calculation},
		{name: "rate with per", line: "50 dollars per hour", want: Calculation},
	}

	ctx := interpreter.NewEnvironment()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyLineTest(t, tt.line, ctx)
			if got != tt.want {
				t.Errorf("ClassifyLine(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestCompoundExpressionsClassification(t *testing.T) {
	tests := []struct {
		name string
		line string
		want LineType
	}{
		{name: "quantity arithmetic", line: "10 meters + 5 meters", want: Calculation},
		{name: "currency arithmetic", line: "$100 + $50", want: Calculation},
		{name: "percentage of", line: "10% of 200", want: Calculation},
	}

	ctx := interpreter.NewEnvironment()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyLineTest(t, tt.line, ctx)
			if got != tt.want {
				t.Errorf("ClassifyLine(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestNapkinFormatClassification(t *testing.T) {
	tests := []struct {
		name string
		line string
		want LineType
	}{
		{name: "napkin format", line: "1234567 as napkin", want: Calculation},
		{name: "negative napkin", line: "-1234567 as napkin", want: Calculation},
	}

	ctx := interpreter.NewEnvironment()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyLineTest(t, tt.line, ctx)
			if got != tt.want {
				t.Errorf("ClassifyLine(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestGrowthFunctionClassification(t *testing.T) {
	tests := []struct {
		name string
		line string
		want LineType
	}{
		{name: "compound functional", line: "compound(1000, 5%, 10)", want: Calculation},
		{name: "grow functional", line: "grow(100, 20, 5)", want: Calculation},
		{name: "depreciate functional", line: "depreciate(50000, 15%, 5)", want: Calculation},
		{name: "compound assignment", line: "result = compound(1000, 5%, 10)", want: Calculation},
	}

	ctx := interpreter.NewEnvironment()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyLineTest(t, tt.line, ctx)
			if got != tt.want {
				t.Errorf("ClassifyLine(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestContextSensitiveClassification(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		want    LineType
		setupFn func(*interpreter.Environment)
	}{
		{
			name: "defined variable",
			line: "price",
			want: Calculation,
			setupFn: func(env *interpreter.Environment) {
				val, _ := types.NewCurrency(100, "USD")
				env.Set("price", val)
			},
		},
		{
			name: "undefined variable",
			line: "unknown",
			want: Markdown,
			setupFn: func(env *interpreter.Environment) {
				// No setup needed
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := interpreter.NewEnvironment()
			tt.setupFn(ctx)
			got := classifyLineTest(t, tt.line, ctx)
			if got != tt.want {
				t.Errorf("ClassifyLine(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}
