package interpreter

// rate_widening_test.go — Tests for rate arithmetic widening.
// When a Rate participates in * or / with a non-Rate operand,
// the rate's Amount is extracted and the time denominator is dropped.
// This makes rates interoperable with plain numbers and quantities.

import (
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/parser"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
)

func TestRateWidening_NumberTimesRate(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedValue string
		expectedUnit  string
	}{
		{
			name:          "number * rate with unit",
			input:         "a = 2 posts/week\nb = 2 * a\n",
			expectedValue: "4",
			expectedUnit:  "posts",
		},
		{
			name:          "number * unitless rate",
			input:         "a = 100/second\nb = 3 * a\n",
			expectedValue: "300",
			expectedUnit:  "", // unitless rate → number
		},
		{
			name:          "number * data rate",
			input:         "r = 100 MB/s\nresult = 0.5 * r\n",
			expectedValue: "50",
			expectedUnit:  "MB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}
			interp := NewInterpreter()
			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error: %v", err)
			}
			result := results[len(results)-1]

			if tt.expectedUnit == "" {
				// Expect a Number (unitless rate widens to number)
				num, ok := result.(*types.Number)
				if !ok {
					t.Fatalf("Expected *types.Number, got %T (%v)", result, result)
				}
				if num.Value.String() != tt.expectedValue {
					t.Errorf("Expected %s, got %s", tt.expectedValue, num.Value.String())
				}
			} else {
				// Expect a Quantity
				qty, ok := result.(*types.Quantity)
				if !ok {
					t.Fatalf("Expected *types.Quantity, got %T (%v)", result, result)
				}
				if qty.Value.String() != tt.expectedValue {
					t.Errorf("Expected value %s, got %s", tt.expectedValue, qty.Value.String())
				}
				if qty.Unit != tt.expectedUnit {
					t.Errorf("Expected unit %q, got %q", tt.expectedUnit, qty.Unit)
				}
			}
		})
	}
}

func TestRateWidening_RateTimesNumber_StaysRate(t *testing.T) {
	// Rate * Number → Rate (rate on the left preserves rate type)
	tests := []struct {
		name          string
		input         string
		expectedValue string
		expectedUnit  string
	}{
		{
			name:          "rate * number with unit",
			input:         "a = 2 posts/week\nb = a * 3\n",
			expectedValue: "6",
			expectedUnit:  "posts/week",
		},
		{
			name:          "rate * number unitless",
			input:         "a = 100/second\nb = a * 3\n",
			expectedValue: "300",
			expectedUnit:  "/s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}
			interp := NewInterpreter()
			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error: %v", err)
			}
			result := results[len(results)-1]
			rate, ok := result.(*types.Rate)
			if !ok {
				t.Fatalf("Expected *types.Rate, got %T (%v)", result, result)
			}
			if rate.Amount.Value.String() != tt.expectedValue {
				t.Errorf("Expected amount %s, got %s", tt.expectedValue, rate.Amount.Value.String())
			}
			if rate.CompoundUnit() != tt.expectedUnit {
				t.Errorf("Expected unit %s, got %s", tt.expectedUnit, rate.CompoundUnit())
			}
		})
	}
}

func TestRateWidening_NumberDivRate(t *testing.T) {
	// Number / unitless Rate → Number / Number → Number
	input := "a = 10/second\nb = 100 / a\n"
	nodes, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	interp := NewInterpreter()
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	result := results[len(results)-1]
	num, ok := result.(*types.Number)
	if !ok {
		t.Fatalf("Expected *types.Number, got %T (%v)", result, result)
	}
	if num.Value.String() != "10" {
		t.Errorf("Expected 10, got %s", num.Value.String())
	}
}

func TestRateWidening_NumberDivRateWithUnit_Error(t *testing.T) {
	// Number / Rate-with-unit → Number / Quantity → error (dimensionally invalid)
	input := "a = 5 posts/week\nb = 20 / a\n"
	nodes, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	interp := NewInterpreter()
	_, err = interp.Eval(nodes)
	if err == nil {
		t.Fatal("Expected error for Number / Quantity, got nil")
	}
}

func TestRateWidening_RateDivNumber_StaysRate(t *testing.T) {
	// Rate / Number → Rate (rate on the left preserves rate type)
	tests := []struct {
		name          string
		input         string
		expectedValue string
		expectedUnit  string
	}{
		{
			name:          "rate / number with unit",
			input:         "a = 10 posts/week\nb = a / 2\n",
			expectedValue: "5",
			expectedUnit:  "posts/week",
		},
		{
			name:          "rate / number unitless",
			input:         "a = 100/second\nb = a / 4\n",
			expectedValue: "25",
			expectedUnit:  "/s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}
			interp := NewInterpreter()
			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error: %v", err)
			}
			result := results[len(results)-1]
			rate, ok := result.(*types.Rate)
			if !ok {
				t.Fatalf("Expected *types.Rate, got %T (%v)", result, result)
			}
			if rate.Amount.Value.String() != tt.expectedValue {
				t.Errorf("Expected amount %s, got %s", tt.expectedValue, rate.Amount.Value.String())
			}
			if rate.CompoundUnit() != tt.expectedUnit {
				t.Errorf("Expected unit %s, got %s", tt.expectedUnit, rate.CompoundUnit())
			}
		})
	}
}

func TestRateWidening_EndToEnd(t *testing.T) {
	// Full scenario: rate used in downstream arithmetic
	input := `posts_per_week = 2 posts/week
daily_users = 4000000
weekly_posts = daily_users * posts_per_week
`
	nodes, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	interp := NewInterpreter()
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}

	// weekly_posts = 4000000 * (2 posts/week) → 8000000 posts
	result := results[len(results)-1]
	qty, ok := result.(*types.Quantity)
	if !ok {
		t.Fatalf("Expected *types.Quantity, got %T (%v)", result, result)
	}
	if qty.Value.String() != "8000000" {
		t.Errorf("Expected 8000000, got %s", qty.Value.String())
	}
	if qty.Unit != "posts" {
		t.Errorf("Expected unit 'posts', got %q", qty.Unit)
	}
}

func TestRateWidening_RateDivRate_Unchanged(t *testing.T) {
	// Rate / Rate → Number (existing behavior, should not change)
	input := "a = 10 posts/day\nb = 2 posts/day\nc = a / b\n"
	nodes, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	interp := NewInterpreter()
	results, err := interp.Eval(nodes)
	if err != nil {
		t.Fatalf("Eval error: %v", err)
	}
	result := results[len(results)-1]
	num, ok := result.(*types.Number)
	if !ok {
		t.Fatalf("Expected *types.Number, got %T (%v)", result, result)
	}
	if num.Value.String() != "5" {
		t.Errorf("Expected 5, got %s", num.Value.String())
	}
}
