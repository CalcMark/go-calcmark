package parser

import (
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/ast"
)

func TestRateParsing(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		checkAST    func(*testing.T, []ast.Node)
	}{
		{
			name:        "rate with slash",
			input:       "100 MB/s\n",
			expectError: false,
			checkAST: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				// RateLiteral is returned directly, not wrapped
				rate, ok := nodes[0].(*ast.RateLiteral)
				if !ok {
					t.Fatalf("Expected RateLiteral, got %T", nodes[0])
				}
				if rate.PerUnit != "s" {
					t.Errorf("Expected per unit 's', got '%s'", rate.PerUnit)
				}
			},
		},
		{
			name:        "rate with per keyword",
			input:       "5 GB per day\n",
			expectError: false,
			checkAST: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				rate, ok := nodes[0].(*ast.RateLiteral)
				if !ok {
					t.Fatalf("Expected RateLiteral, got %T", nodes[0])
				}
				if rate.PerUnit != "day" {
					t.Errorf("Expected per unit 'day', got '%s'", rate.PerUnit)
				}
			},
		},
		{
			name:        "cost rate per hour",
			input:       "$0.10 per hour\n",
			expectError: false,
			checkAST: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				rate, ok := nodes[0].(*ast.RateLiteral)
				if !ok {
					t.Fatalf("Expected RateLiteral, got %T", nodes[0])
				}
				if rate.PerUnit != "hour" {
					t.Errorf("Expected per unit 'hour', got '%s'", rate.PerUnit)
				}
			},
		},
		{
			name:        "regular division not a rate",
			input:       "10 / 5\n",
			expectError: false,
			checkAST: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				// Should be BinaryOp (division), not RateLiteral
				_, ok := nodes[0].(*ast.BinaryOp)
				if !ok {
					t.Fatalf("Expected BinaryOp for division, got %T", nodes[0])
				}
			},
		},
		{
			name:        "rate with minutes",
			input:       "1000 req/min\n",
			expectError: false,
			checkAST: func(t *testing.T, nodes []ast.Node) {
				if len(nodes) != 1 {
					t.Fatalf("Expected 1 node, got %d", len(nodes))
				}
				rate, ok := nodes[0].(*ast.RateLiteral)
				if !ok {
					t.Fatalf("Expected RateLiteral, got %T", nodes[0])
				}
				if rate.PerUnit != "min" {
					t.Errorf("Expected per unit 'min', got '%s'", rate.PerUnit)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.checkAST != nil && !tt.expectError {
				tt.checkAST(t, nodes)
			}
		})
	}
}

// TestIdentifierSlashDisambiguation verifies that whitespace around / controls
// whether identifier/time_unit is parsed as a rate or division.
//
//	"weekly_posts/week"  → tight slash → RateLiteral (rate intent)
//	"total / days"       → spaced slash → BinaryOp (division intent)
func TestIdentifierSlashDisambiguation(t *testing.T) {
	// Spaced slash: identifier / time_unit → division
	divisionTests := []struct {
		name  string
		input string
	}{
		{name: "spaced: total / days", input: "total / days\n"},
		{name: "spaced: cost / hours", input: "cost / hours\n"},
		{name: "spaced: revenue / month", input: "revenue / month\n"},
	}

	for _, tt := range divisionTests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Unexpected parse error: %v", err)
			}
			if len(nodes) != 1 {
				t.Fatalf("Expected 1 node, got %d", len(nodes))
			}
			binOp, ok := nodes[0].(*ast.BinaryOp)
			if !ok {
				t.Fatalf("Expected BinaryOp for spaced division, got %T", nodes[0])
			}
			if binOp.Operator != "/" {
				t.Errorf("Expected operator '/', got '%s'", binOp.Operator)
			}
		})
	}

	// Tight slash: identifier/time_unit → rate
	rateTests := []struct {
		name    string
		input   string
		perUnit string
	}{
		{name: "tight: weekly_posts/week", input: "weekly_posts/week\n", perUnit: "week"},
		{name: "tight: count/day", input: "count/day\n", perUnit: "day"},
		{name: "tight: bandwidth/s", input: "bandwidth/s\n", perUnit: "s"},
	}

	for _, tt := range rateTests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Unexpected parse error: %v", err)
			}
			if len(nodes) != 1 {
				t.Fatalf("Expected 1 node, got %d", len(nodes))
			}
			rate, ok := nodes[0].(*ast.RateLiteral)
			if !ok {
				t.Fatalf("Expected RateLiteral for tight slash, got %T", nodes[0])
			}
			if rate.PerUnit != tt.perUnit {
				t.Errorf("Expected per unit '%s', got '%s'", tt.perUnit, rate.PerUnit)
			}
		})
	}
}

// TestIdentifierPerDesugarsToConvertRate tests that "identifier per time_unit"
// desugars to convert_rate(identifier, time_unit) at parse time (issue #87).
func TestIdentifierPerDesugarsToConvertRate(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		targetUnit string
	}{
		{
			name:       "identifier per year",
			input:      "d per year\n",
			targetUnit: "year",
		},
		{
			name:       "identifier per second",
			input:      "r per second\n",
			targetUnit: "second",
		},
		{
			name:       "identifier per month",
			input:      "rate per month\n",
			targetUnit: "month",
		},
		{
			name:       "identifier per quarter",
			input:      "rate per quarter\n",
			targetUnit: "quarter",
		},
		{
			name:       "identifier per quarterly (alias)",
			input:      "rate per quarterly\n",
			targetUnit: "quarterly",
		},
	}

	// Sanity: when RHS is also a bare identifier (potentially a variable),
	// the parser still emits convert_rate; runtime decides validity.
	// Covered by TestIdentifierPerAcceptsVariableOrDurationRHS below.

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Unexpected parse error: %v", err)
			}

			if len(nodes) != 1 {
				t.Fatalf("Expected 1 node, got %d", len(nodes))
			}

			// Must be a FunctionCall to convert_rate, NOT a RateLiteral
			fc, ok := nodes[0].(*ast.FunctionCall)
			if !ok {
				t.Fatalf("Expected FunctionCall (convert_rate), got %T", nodes[0])
			}

			if fc.Name != "convert_rate" {
				t.Errorf("Expected function name 'convert_rate', got '%s'", fc.Name)
			}

			if len(fc.Arguments) != 2 {
				t.Fatalf("Expected 2 arguments, got %d", len(fc.Arguments))
			}

			// First arg should be the identifier
			_, isIdent := fc.Arguments[0].(*ast.Identifier)
			if !isIdent {
				t.Errorf("Expected first arg to be Identifier, got %T", fc.Arguments[0])
			}

			// Second arg should be the target time unit identifier
			targetIdent, ok := fc.Arguments[1].(*ast.Identifier)
			if !ok {
				t.Fatalf("Expected second arg to be Identifier, got %T", fc.Arguments[1])
			}
			if targetIdent.Name != tt.targetUnit {
				t.Errorf("Expected target unit '%s', got '%s'", tt.targetUnit, targetIdent.Name)
			}
		})
	}
}

// TestLiteralPerStillCreatesRate ensures that literal expressions with per
// still create RateLiterals (not convert_rate calls).
// TestIdentifierPerAcceptsVariableOrDurationRHS extends the NL form
// `<rate> per <period>` so the period can be a variable name (resolved
// at runtime to a Duration) or a duration literal. Both cases desugar
// to convert_rate with the RHS expression passed as-is — the runtime
// inspects the value to extract the time unit.
//
// Bare time-unit identifiers (covered by TestIdentifierPerDesugarsToConvertRate)
// still take the fast path: RHS is an Identifier node with the unit name.
func TestIdentifierPerAcceptsVariableOrDurationRHS(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		checkRHS func(t *testing.T, rhs ast.Node)
	}{
		{
			name:  "rate variable per variable (runtime resolves)",
			input: "a per p\n",
			checkRHS: func(t *testing.T, rhs ast.Node) {
				ident, ok := rhs.(*ast.Identifier)
				if !ok {
					t.Fatalf("Expected Identifier, got %T", rhs)
				}
				if ident.Name != "p" {
					t.Errorf("Expected ident 'p', got %q", ident.Name)
				}
			},
		},
		{
			name:  "rate variable per duration literal",
			input: "a per 1 day\n",
			checkRHS: func(t *testing.T, rhs ast.Node) {
				dur, ok := rhs.(*ast.DurationLiteral)
				if !ok {
					t.Fatalf("Expected DurationLiteral, got %T", rhs)
				}
				if dur.Unit != "day" && dur.Unit != "days" {
					t.Errorf("Expected unit day/days, got %q", dur.Unit)
				}
			},
		},
		{
			name:  "rate variable per multi-day duration literal",
			input: "a per 5 days\n",
			checkRHS: func(t *testing.T, rhs ast.Node) {
				dur, ok := rhs.(*ast.DurationLiteral)
				if !ok {
					t.Fatalf("Expected DurationLiteral, got %T", rhs)
				}
				if dur.Unit != "days" && dur.Unit != "day" {
					t.Errorf("Expected unit day/days, got %q", dur.Unit)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Unexpected parse error: %v", err)
			}
			if len(nodes) != 1 {
				t.Fatalf("Expected 1 node, got %d", len(nodes))
			}
			fc, ok := nodes[0].(*ast.FunctionCall)
			if !ok {
				t.Fatalf("Expected FunctionCall (convert_rate), got %T", nodes[0])
			}
			if fc.Name != "convert_rate" {
				t.Errorf("Expected convert_rate, got %q", fc.Name)
			}
			if len(fc.Arguments) != 2 {
				t.Fatalf("Expected 2 args, got %d", len(fc.Arguments))
			}
			tt.checkRHS(t, fc.Arguments[1])
		})
	}
}

// TestRateLiteralPerAcceptsVariableOrDurationRHS pins the same behavior
// for the rate-literal LHS path: `(5/day) per p`, `(5/day) per 1 hour`.
func TestRateLiteralPerAcceptsVariableOrDurationRHS(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"rate-literal per variable", "(5/day) per p\n"},
		{"rate-literal per duration", "(5/day) per 1 hour\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Unexpected parse error: %v", err)
			}
			if len(nodes) != 1 {
				t.Fatalf("Expected 1 node, got %d", len(nodes))
			}
			fc, ok := nodes[0].(*ast.FunctionCall)
			if !ok {
				t.Fatalf("Expected FunctionCall (convert_rate), got %T", nodes[0])
			}
			if fc.Name != "convert_rate" {
				t.Errorf("Expected convert_rate, got %q", fc.Name)
			}
		})
	}
}

func TestLiteralPerStillCreatesRate(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "quantity per day",
			input: "5 GB per day\n",
		},
		{
			name:  "number per hour",
			input: "100 per hour\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Unexpected parse error: %v", err)
			}
			if len(nodes) != 1 {
				t.Fatalf("Expected 1 node, got %d", len(nodes))
			}
			// Must be RateLiteral, NOT FunctionCall
			_, ok := nodes[0].(*ast.RateLiteral)
			if !ok {
				t.Fatalf("Expected RateLiteral, got %T", nodes[0])
			}
		})
	}
}

// TestRateUnitConversionParsing tests parsing of rate-to-rate unit conversion.
// Examples: "10 m/s in inch/s", "60 km/h in mph"
func TestRateUnitConversionParsing(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectError    bool
		targetUnit     string
		targetTimeUnit string
	}{
		{
			name:           "meters per second to inches per second",
			input:          "10 m/s in inch/s\n",
			expectError:    false,
			targetUnit:     "inch",
			targetTimeUnit: "s",
		},
		{
			name:           "km per hour to miles per hour",
			input:          "60 km/h in mile/h\n",
			expectError:    false,
			targetUnit:     "mile",
			targetTimeUnit: "h",
		},
		{
			name:           "rate conversion with per keyword",
			input:          "10 m/s in inch per second\n",
			expectError:    false,
			targetUnit:     "inch",
			targetTimeUnit: "second",
		},
		{
			name:           "rate conversion changing time unit",
			input:          "60 m/s in m/min\n",
			expectError:    false,
			targetUnit:     "m",
			targetTimeUnit: "min",
		},
		{
			name:        "invalid time unit in rate conversion",
			input:       "10 m/s in inch/foo\n",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := Parse(tt.input)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected parse error: %v", err)
			}

			if len(nodes) != 1 {
				t.Fatalf("Expected 1 node, got %d", len(nodes))
			}

			conv, ok := nodes[0].(*ast.UnitConversion)
			if !ok {
				t.Fatalf("Expected UnitConversion, got %T", nodes[0])
			}

			if conv.TargetUnit != tt.targetUnit {
				t.Errorf("Expected target unit '%s', got '%s'", tt.targetUnit, conv.TargetUnit)
			}

			if conv.TargetTimeUnit != tt.targetTimeUnit {
				t.Errorf("Expected target time unit '%s', got '%s'", tt.targetTimeUnit, conv.TargetTimeUnit)
			}

			// Verify the source is a RateLiteral
			_, isRate := conv.Quantity.(*ast.RateLiteral)
			if !isRate {
				t.Errorf("Expected source to be RateLiteral, got %T", conv.Quantity)
			}
		})
	}
}
