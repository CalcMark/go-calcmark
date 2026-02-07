package interpreter

import (
	"testing"

	"github.com/CalcMark/go-calcmark/format/display"
	"github.com/CalcMark/go-calcmark/spec/parser"
	"github.com/CalcMark/go-calcmark/spec/types"
)

// TestNapkinTypePreservation verifies that napkin conversion preserves the input type.
// This is the core test for the type erasure bug fix.
// - Quantity in -> Quantity out (with unit)
// - Currency in -> Currency out (with symbol)
// - Rate in -> Rate out (with Amount.Unit and PerUnit)
// - Duration in -> Duration out (with unit)
// - Number in -> Number out
func TestNapkinTypePreservation(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantType     string // "Quantity", "Currency", "Rate", "Duration", "Number"
		wantUnit     string // Expected unit for Quantity/Duration, symbol for Currency
		wantPerUnit  string // Expected PerUnit for Rate
		checkValue   bool   // Whether to check the rounded value
		approxValue  float64
		tolerance    float64
	}{
		// Quantity type preservation - the main bug case
		{
			name:        "Quantity: data accumulation preserves unit",
			input:       "x = accumulate(5 MB/s, 1 day) as napkin\n",
			wantType:    "Quantity",
			wantUnit:    "GB", // Should normalize 432000 MB to ~400 GB
			checkValue:  true,
			approxValue: 400,
			tolerance:   50,
		},
		{
			name:     "Quantity: simple quantity preserves unit",
			input:    "x = 1234 meters as napkin\n",
			wantType: "Quantity",
			wantUnit: "km", // Should normalize 1234 meters to ~1.2 km
		},
		{
			name:     "Quantity: small quantity keeps same unit",
			input:    "x = 47 MB as napkin\n",
			wantType: "Quantity",
			wantUnit: "MB", // 47 MB stays as MB
		},

		// Currency type preservation
		{
			name:        "Currency: preserves symbol",
			input:       "x = $1234567 as napkin\n",
			wantType:    "Currency",
			wantUnit:    "$", // Symbol preserved
			checkValue:  true,
			approxValue: 1200000,
			tolerance:   100000,
		},
		{
			name:     "Currency: euro symbol preserved",
			input:    "x = €9876 as napkin\n",
			wantType: "Currency",
			wantUnit: "€",
		},

		// Rate type preservation
		{
			name:        "Rate: preserves amount unit and per-unit",
			input:       "x = 100 MB/s as napkin\n",
			wantType:    "Rate",
			wantUnit:    "MB", // Amount unit
			wantPerUnit: "second",
		},
		{
			name:        "Rate: large rate value rounded",
			input:       "x = 1234567 requests/hour as napkin\n",
			wantType:    "Rate",
			wantUnit:    "requests",
			wantPerUnit: "hour",
		},

		// Duration type preservation
		// Note: Parser normalizes duration units to singular form
		{
			name:     "Duration: preserves unit",
			input:    "x = 86400 seconds as napkin\n",
			wantType: "Duration",
			wantUnit: "second", // Parser normalizes to singular
		},
		{
			name:     "Duration: days preserved",
			input:    "x = 365 days as napkin\n",
			wantType: "Duration",
			wantUnit: "day", // Parser normalizes to singular
		},

		// Number type (existing behavior)
		{
			name:        "Number: plain number stays number",
			input:       "x = 1234567 as napkin\n",
			wantType:    "Number",
			checkValue:  true,
			approxValue: 1200000,
			tolerance:   100000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			// Interpret
			interp := NewInterpreter()
			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error: %v", err)
			}

			if len(results) == 0 {
				t.Fatal("No results returned")
			}

			result := results[0]
			if result == nil {
				t.Fatal("Result is nil")
			}

			// Check the type
			switch tt.wantType {
			case "Quantity":
				q, ok := result.(*types.Quantity)
				if !ok {
					t.Errorf("Expected *types.Quantity, got %T", result)
					return
				}
				if q.Unit != tt.wantUnit {
					t.Errorf("Expected unit %q, got %q", tt.wantUnit, q.Unit)
				}
				if tt.checkValue {
					val, _ := q.Value.Float64()
					diff := val - tt.approxValue
					if diff < 0 {
						diff = -diff
					}
					if diff > tt.tolerance {
						t.Errorf("Expected value ~%v, got %v (diff: %v, tolerance: %v)",
							tt.approxValue, val, diff, tt.tolerance)
					}
				}

			case "Currency":
				c, ok := result.(*types.Currency)
				if !ok {
					t.Errorf("Expected *types.Currency, got %T", result)
					return
				}
				if c.Symbol != tt.wantUnit {
					t.Errorf("Expected symbol %q, got %q", tt.wantUnit, c.Symbol)
				}
				if tt.checkValue {
					val, _ := c.Value.Float64()
					diff := val - tt.approxValue
					if diff < 0 {
						diff = -diff
					}
					if diff > tt.tolerance {
						t.Errorf("Expected value ~%v, got %v", tt.approxValue, val)
					}
				}

			case "Rate":
				r, ok := result.(*types.Rate)
				if !ok {
					t.Errorf("Expected *types.Rate, got %T", result)
					return
				}
				if r.Amount.Unit != tt.wantUnit {
					t.Errorf("Expected amount unit %q, got %q", tt.wantUnit, r.Amount.Unit)
				}
				if r.PerUnit != tt.wantPerUnit {
					t.Errorf("Expected per-unit %q, got %q", tt.wantPerUnit, r.PerUnit)
				}

			case "Duration":
				d, ok := result.(*types.Duration)
				if !ok {
					t.Errorf("Expected *types.Duration, got %T", result)
					return
				}
				if d.Unit != tt.wantUnit {
					t.Errorf("Expected unit %q, got %q", tt.wantUnit, d.Unit)
				}

			case "Number":
				n, ok := result.(*types.Number)
				if !ok {
					t.Errorf("Expected *types.Number, got %T", result)
					return
				}
				if tt.checkValue {
					val, _ := n.Value.Float64()
					diff := val - tt.approxValue
					if diff < 0 {
						diff = -diff
					}
					if diff > tt.tolerance {
						t.Errorf("Expected value ~%v, got %v", tt.approxValue, val)
					}
				}

			default:
				t.Fatalf("Unknown expected type: %s", tt.wantType)
			}
		})
	}
}

// TestNapkinQuantityIsNapkinFlag verifies that napkin conversion sets IsNapkin=true
// on Quantity results, enabling tilde prefix in display formatting.
func TestNapkinQuantityIsNapkinFlag(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantDisplay  string
	}{
		{
			name:        "data accumulation shows tilde",
			input:       "x = accumulate(5 MB/s, 1 day) as napkin\n",
			wantDisplay: "~420 GB", // 5 MB/s * 86400s = 432000 MB ≈ 421.875 GB → ~420 GB
		},
		{
			name:        "simple quantity shows tilde",
			input:       "x = 1200 meters as napkin\n",
			wantDisplay: "~1.2 km",
		},
		{
			name:        "small quantity shows tilde",
			input:       "x = 47 MB as napkin\n",
			wantDisplay: "~47 MB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			// Interpret
			interp := NewInterpreter()
			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error: %v", err)
			}

			if len(results) == 0 {
				t.Fatal("No results returned")
			}

			result := results[0]
			q, ok := result.(*types.Quantity)
			if !ok {
				t.Fatalf("Expected *types.Quantity, got %T", result)
			}

			// Verify IsNapkin flag is set
			if !q.IsNapkin {
				t.Error("Expected IsNapkin=true, got false")
			}

			// Verify display format includes tilde
			formatted := display.FormatQuantity(q)
			if formatted != tt.wantDisplay {
				t.Errorf("Expected display %q, got %q", tt.wantDisplay, formatted)
			}
		})
	}
}
