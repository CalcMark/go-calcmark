package interpreter

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/parser"
	"github.com/CalcMark/go-calcmark/spec/types"
)

// TestNLFunctionEquivalence verifies that plain language syntax produces
// identical results to parenthesized function-call syntax.
func TestNLFunctionEquivalence(t *testing.T) {
	tests := []struct {
		name      string
		nlInput   string // Natural language form
		funcInput string // Parenthesized form
	}{
		{
			name:      "read 100 MB from ssd",
			nlInput:   "read 100 MB from ssd\n",
			funcInput: "read(100 MB, ssd)\n",
		},
		{
			name:      "read 1 GB from nvme",
			nlInput:   "read 1 GB from nvme\n",
			funcInput: "read(1 GB, nvme)\n",
		},
		{
			name:      "read 500 MB from hdd",
			nlInput:   "read 500 MB from hdd\n",
			funcInput: "read(500 MB, hdd)\n",
		},
		{
			name:      "compress 1 GB using gzip",
			nlInput:   "compress 1 GB using gzip\n",
			funcInput: "compress(1 GB, gzip)\n",
		},
		{
			name:      "compress 500 MB using lz4",
			nlInput:   "compress 500 MB using lz4\n",
			funcInput: "compress(500 MB, lz4)\n",
		},
		{
			name:      "compress 2 GB using zstd",
			nlInput:   "compress 2 GB using zstd\n",
			funcInput: "compress(2 GB, zstd)\n",
		},
		{
			name:      "transfer 1 GB across regional gigabit",
			nlInput:   "transfer 1 GB across regional gigabit\n",
			funcInput: "transfer_time(1 GB, regional, gigabit)\n",
		},
		{
			name:      "transfer 500 MB across global wifi",
			nlInput:   "transfer 500 MB across global wifi\n",
			funcInput: "transfer_time(500 MB, global, wifi)\n",
		},
		{
			name:      "transfer 100 MB across local ten_gig",
			nlInput:   "transfer 100 MB across local ten_gig\n",
			funcInput: "transfer_time(100 MB, local, ten_gig)\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse and eval the NL form
			nlNodes, err := parser.Parse(tt.nlInput)
			if err != nil {
				t.Fatalf("NL parse error: %v", err)
			}
			nlInterp := NewInterpreter()
			nlResults, err := nlInterp.Eval(nlNodes)
			if err != nil {
				t.Fatalf("NL eval error: %v", err)
			}
			if len(nlResults) == 0 {
				t.Fatal("NL produced no results")
			}

			// Parse and eval the function-call form
			funcNodes, err := parser.Parse(tt.funcInput)
			if err != nil {
				t.Fatalf("Func parse error: %v", err)
			}
			funcInterp := NewInterpreter()
			funcResults, err := funcInterp.Eval(funcNodes)
			if err != nil {
				t.Fatalf("Func eval error: %v", err)
			}
			if len(funcResults) == 0 {
				t.Fatal("Func produced no results")
			}

			// Compare results
			nlStr := nlResults[0].String()
			funcStr := funcResults[0].String()
			if nlStr != funcStr {
				t.Errorf("NL result %q != func result %q", nlStr, funcStr)
			}
			t.Logf("NL: %q → %s", tt.nlInput[:len(tt.nlInput)-1], nlStr)
			t.Logf("Fn: %q → %s", tt.funcInput[:len(tt.funcInput)-1], funcStr)
		})
	}
}

// TestNLFunctionResults verifies that NL forms produce expected result types.
func TestNLFunctionResults(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		checkResult func(*testing.T, types.Type)
	}{
		{
			name:  "read returns duration",
			input: "read 100 MB from ssd\n",
			checkResult: func(t *testing.T, result types.Type) {
				_, ok := result.(*types.Duration)
				if !ok {
					t.Fatalf("Expected Duration, got %T: %s", result, result.String())
				}
				t.Logf("Read result: %s", result.String())
			},
		},
		{
			name:  "compress returns quantity in data size",
			input: "compress 1 GB using gzip\n",
			checkResult: func(t *testing.T, result types.Type) {
				qty, ok := result.(*types.Quantity)
				if !ok {
					t.Fatalf("Expected Quantity, got %T: %s", result, result.String())
				}
				t.Logf("Compress result: %s (unit: %s)", result.String(), qty.Unit)
			},
		},
		{
			name:  "transfer returns duration",
			input: "transfer 1 GB across regional gigabit\n",
			checkResult: func(t *testing.T, result types.Type) {
				_, ok := result.(*types.Duration)
				if !ok {
					t.Fatalf("Expected Duration, got %T: %s", result, result.String())
				}
				t.Logf("Transfer result: %s", result.String())
			},
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
				t.Fatal("No results")
			}
			tt.checkResult(t, results[0])
		})
	}
}
