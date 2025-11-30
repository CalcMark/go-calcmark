package parser_test

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/spec/ast"
	"github.com/CalcMark/go-calcmark/spec/parser"
)

// TestRecursiveDescentBasics tests the basic functionality of the new parser.
func TestRecursiveDescentBasics(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "simple number",
			input:   "42\n",
			wantErr: false,
		},
		{
			name:    "simple addition",
			input:   "1 + 2\n",
			wantErr: false,
		},
		{
			name:    "simple assignment",
			input:   "x = 10\n",
			wantErr: false,
		},
		{
			name:    "parentheses",
			input:   "(1 + 2) * 3\n",
			wantErr: false,
		},
		{
			name:    "incomplete expression",
			input:   "1 +\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := parser.NewRecursiveDescentParser(tt.input)
			nodes, err := p.Parse()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if len(nodes) == 0 {
					t.Errorf("expected nodes but got none")
				}
			}
		})
	}
}

// TestFrontmatterAssignment tests parsing of @namespace.property = value syntax.
func TestFrontmatterAssignment(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantNamespace string
		wantProperty  string
		wantErr       bool
		errContains   string
	}{
		{
			name:          "exchange rate assignment",
			input:         "@exchange.USD_EUR = 0.92\n",
			wantNamespace: "exchange",
			wantProperty:  "USD_EUR",
		},
		{
			name:          "global variable assignment",
			input:         "@global.tax_rate = 0.32\n",
			wantNamespace: "global",
			wantProperty:  "tax_rate",
		},
		{
			name:          "exchange rate with expression",
			input:         "@exchange.EUR_GBP = 1 / 1.17\n",
			wantNamespace: "exchange",
			wantProperty:  "EUR_GBP",
		},
		{
			name:          "global with currency value",
			input:         "@global.budget = $1000\n",
			wantNamespace: "global",
			wantProperty:  "budget",
		},
		{
			name:        "unknown namespace",
			input:       "@unknown.foo = 42\n",
			wantErr:     true,
			errContains: "unknown frontmatter namespace",
		},
		{
			name:        "missing property",
			input:       "@exchange = 0.92\n",
			wantErr:     true,
			errContains: "expected '.'",
		},
		{
			name:        "missing value",
			input:       "@global.rate =\n",
			wantErr:     true,
			errContains: "expected",
		},
		{
			name:        "missing equals",
			input:       "@global.rate 0.32\n",
			wantErr:     true,
			errContains: "expected '='",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := parser.NewRecursiveDescentParser(tt.input)
			nodes, err := p.Parse()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(nodes) == 0 {
				t.Fatal("expected nodes but got none")
			}

			fmAssign, ok := nodes[0].(*ast.FrontmatterAssignment)
			if !ok {
				t.Fatalf("expected FrontmatterAssignment, got %T", nodes[0])
			}

			if fmAssign.Namespace != tt.wantNamespace {
				t.Errorf("namespace = %q, want %q", fmAssign.Namespace, tt.wantNamespace)
			}
			if fmAssign.Property != tt.wantProperty {
				t.Errorf("property = %q, want %q", fmAssign.Property, tt.wantProperty)
			}
			if fmAssign.Value == nil {
				t.Error("value is nil")
			}
		})
	}
}
