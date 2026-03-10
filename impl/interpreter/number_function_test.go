package interpreter_test

import (
	"testing"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/parser"
	"github.com/CalcMark/go-calcmark/spec/types"
	"github.com/shopspring/decimal"
)

func TestNumberFunction(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		// Extract number from plain number
		{"number from integer", "number(42)\n", "42", false},
		{"number from decimal", "number(3.14)\n", "3.14", false},

		// Extract number from quantity
		{"number from quantity", "number(10 kg)\n", "10", false},
		{"number from quantity expr", "x = 5 MB\nnumber(x)\n", "5", false},

		// Extract number from currency
		{"number from currency", "number($100)\n", "100", false},
		{"number from currency expr", "price = $49.99\nnumber(price)\n", "49.99", false},

		// Extract number from percentage
		{"number from percentage", "number(25%)\n", "0.25", false},

		// Extract number from duration
		{"number from duration", "number(3 hours)\n", "3", false},

		// Errors
		{"number no args", "number()\n", "", true},
		{"number too many args", "number(1, 2)\n", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if tt.wantErr {
				if err != nil {
					return // parse error is acceptable
				}
				interp := interpreter.NewInterpreter()
				_, err = interp.Eval(nodes)
				if err == nil {
					t.Fatalf("expected error for %q, got none", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse error for %q: %v", tt.input, err)
			}
			interp := interpreter.NewInterpreter()
			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error for %q: %v", tt.input, err)
			}
			if len(results) == 0 {
				t.Fatalf("no results for %q", tt.input)
			}
			last := results[len(results)-1]
			num, ok := last.(*types.Number)
			if !ok {
				t.Fatalf("expected *types.Number, got %T for %q", last, tt.input)
			}
			if num.Value.String() != tt.want {
				t.Errorf("number() = %s, want %s for %q", num.Value.String(), tt.want, tt.input)
			}
		})
	}
}

func TestQuantityCurrencyMultiplication(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantValue  string
		wantSymbol string
	}{
		// Quantity * Currency → Currency
		{"qty times currency", "3 people * $100\n", "300", "$"},
		{"qty times currency expr", "hc = 5 people\nrate = $200\nhc * rate\n", "1000", "$"},

		// Currency * Quantity → Currency
		{"currency times qty", "$100 * 3 people\n", "300", "$"},
		{"currency times qty expr", "rate = $150\nhc = 4 people\nrate * hc\n", "600", "$"},

		// Works with custom units
		{"custom unit times currency", "12 dogs * $50\n", "600", "$"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error for %q: %v", tt.input, err)
			}
			interp := interpreter.NewInterpreter()
			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error for %q: %v", tt.input, err)
			}
			if len(results) == 0 {
				t.Fatalf("no results for %q", tt.input)
			}
			last := results[len(results)-1]
			cur, ok := last.(*types.Currency)
			if !ok {
				t.Fatalf("expected *types.Currency, got %T for %q", last, tt.input)
			}
			want := decimal.RequireFromString(tt.wantValue)
			if !cur.Value.Equal(want) {
				t.Errorf("got %s, want %s for %q", cur.Value, tt.wantValue, tt.input)
			}
			if cur.Symbol != tt.wantSymbol {
				t.Errorf("got symbol %q, want %q for %q", cur.Symbol, tt.wantSymbol, tt.input)
			}
		})
	}
}
