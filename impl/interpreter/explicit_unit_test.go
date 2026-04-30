package interpreter

import (
	"testing"

	"github.com/CalcMark/go-calcmark/v2/format/display"
	"github.com/CalcMark/go-calcmark/v2/spec/parser"
	"github.com/CalcMark/go-calcmark/v2/spec/types"
)

// TestExplicitUnitConversionDisplay verifies that explicit unit conversions
// via `in`/`as` display in the user's chosen unit, not auto-scaled.
func TestExplicitUnitConversionDisplay(t *testing.T) {
	f := display.DefaultFormatter()

	tests := []struct {
		name            string
		input           string
		expectedDisplay string
		expectExplicit  bool
	}{
		{
			name:            "200 kW in MW displays as 0.2 MW",
			input:           "200 kilowatts in megawatts\n",
			expectedDisplay: "0.2 megawatts",
			expectExplicit:  true,
		},
		{
			name:            "500 kW in MW displays as 0.5 MW",
			input:           "500 kilowatts in megawatts\n",
			expectedDisplay: "0.5 megawatts",
			expectExplicit:  true,
		},
		{
			name:            "1 m in mm displays as 1,000 mm",
			input:           "1 meter in millimeters\n",
			expectedDisplay: "1,000 millimeters",
			expectExplicit:  true,
		},
		{
			name:            "1000 kW in MW displays as 1 MW",
			input:           "1000 kilowatts in megawatts\n",
			expectedDisplay: "1 megawatts",
			expectExplicit:  true,
		},
		{
			name:            "arithmetic drops explicit flag",
			input:           "(200 kilowatts in megawatts) * 2\n",
			expectedDisplay: "400 kW",
			expectExplicit:  false,
		},
		{
			name:            "chained conversion works",
			input:           "(1 meter in feet) in meters\n",
			expectedDisplay: "1 meters",
			expectExplicit:  true,
		},
		{
			name:            "napkin on explicit re-scales normally",
			input:           "200 kilowatts in megawatts as napkin\n",
			expectedDisplay: "~200 kW",
			expectExplicit:  false,
		},
		{
			name:            "precise on explicit shows full precision",
			input:           "10 meters in feet as precise\n",
			expectedDisplay: "32.808399 feet",
			expectExplicit:  true,
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

			if len(results) == 0 {
				t.Fatal("No results returned")
			}

			result := results[len(results)-1]

			// Check IsExplicit flag
			if qty, ok := result.(*types.Quantity); ok {
				if qty.IsExplicit != tt.expectExplicit {
					t.Errorf("IsExplicit = %v, want %v", qty.IsExplicit, tt.expectExplicit)
				}
			}

			// Check formatted display
			displayed := f.Format(result)
			if displayed != tt.expectedDisplay {
				t.Errorf("display = %q, want %q", displayed, tt.expectedDisplay)
			}
		})
	}
}
