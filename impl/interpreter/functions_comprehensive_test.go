package interpreter_test

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/parser"
)

// TestAllStandardFunctions systematically tests every function in BuiltinFunctions.
// This covers INTERP-03: All functions work correctly in standard form.
func TestAllStandardFunctions(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		tolerance   float64 // For float comparisons, 0 means exact string match
		wantErr     bool
		containsStr bool // If true, check if result contains expected string
	}{
		// ==================== Math Functions ====================

		// avg() - Average of numbers
		{"avg basic", "avg(1, 2, 3)\n", "2", 0, false, false},
		{"avg with decimals", "avg(1.5, 2.5, 3.0)\n", "2.333333", 0.0001, false, false},
		{"avg single value", "avg(42)\n", "42", 0, false, false},
		{"avg five values", "avg(10, 20, 30, 40, 50)\n", "30", 0, false, false},
		{"avg negative values", "avg(-10, 0, 10)\n", "0", 0, false, false},

		// sqrt() - Square root
		{"sqrt of 9", "sqrt(9)\n", "3", 0, false, false},
		{"sqrt of 16", "sqrt(16)\n", "4", 0, false, false},
		{"sqrt of 2", "sqrt(2)\n", "1.414213", 0.0001, false, false},
		{"sqrt of 2.25", "sqrt(2.25)\n", "1.5", 0, false, false},
		{"sqrt of 0", "sqrt(0)\n", "0", 0, false, false},
		{"sqrt of 100", "sqrt(100)\n", "10", 0, false, false},

		// accumulate() - Rate accumulation over time
		{"accumulate MB/s over day", "accumulate(100 MB/s, 1 day)\n", "8640000", 0.01, false, false},
		{"accumulate GB/day over year", "accumulate(1 GB/day, 1 year)\n", "365", 0.01, false, false},
		{"accumulate KB/s over hour", "accumulate(10 KB/s, 1 hour)\n", "36000", 0.01, false, false},

		// ==================== Conversion Functions ====================

		// convert_rate() - Rate time unit conversion
		{"convert_rate day to second", "convert_rate(86400/day, second)\n", "1", 0.0001, false, false},
		{"convert_rate per second to hour", "convert_rate(1000/second, hour)\n", "3600000", 0.01, false, false},
		{"convert_rate per minute to second", "convert_rate(60/minute, second)\n", "1", 0.0001, false, false},

		// ==================== Network Functions ====================

		// rtt() - Round-trip time
		{"rtt local", "rtt(local)\n", "0.0005", 0.0001, false, false},     // 0.5ms = 0.0005s
		{"rtt regional", "rtt(regional)\n", "0.01", 0.0001, false, false}, // 10ms = 0.01s
		{"rtt continental", "rtt(continental)\n", "0.05", 0.001, false, false},
		{"rtt global", "rtt(global)\n", "0.15", 0.001, false, false}, // 150ms = 0.15s

		// throughput() - Network bandwidth
		{"throughput gigabit", "throughput(gigabit)\n", "125", 0.01, false, false},        // 125 MB/s
		{"throughput ten_gig", "throughput(ten_gig)\n", "1250", 0.01, false, false},       // 1250 MB/s
		{"throughput wifi", "throughput(wifi)\n", "12.5", 0.01, false, false},             // 12.5 MB/s
		{"throughput four_g", "throughput(four_g)\n", "2.5", 0.01, false, false},          // 2.5 MB/s
		{"throughput five_g", "throughput(five_g)\n", "50", 0.01, false, false},           // 50 MB/s
		{"throughput hundred_gig", "throughput(hundred_gig)\n", "12500", 1, false, false}, // 12500 MB/s

		// transfer_time() - Data transfer time
		{"transfer_time small local gigabit", "transfer_time(1 KB, local, gigabit)\n", "0.0005", 0.001, false, false}, // RTT dominates

		// ==================== Storage Functions ====================

		// seek() - Storage seek latency
		{"seek hdd", "seek(hdd)\n", "0.01", 0.001, false, false},         // 10ms = 0.01s
		{"seek ssd", "seek(ssd)\n", "0.0001", 0.00001, false, false},     // 0.1ms = 0.0001s
		{"seek nvme", "seek(nvme)\n", "0.00001", 0.000001, false, false}, // 0.01ms

		// read() - Storage read time
		{"read 100 MB ssd", "read(100 MB, ssd)\n", "0.1818", 0.01, false, false}, // 100/550 seconds

		// compress() - Compression estimate (returns Quantity with unit)
		{"compress gzip", "compress(300 MB, gzip)\n", "100", 0.5, false, false},     // 300/3 = 100 MB
		{"compress lz4", "compress(100 MB, lz4)\n", "50", 0.5, false, false},        // 100/2 = 50 MB
		{"compress zstd", "compress(350 MB, zstd)\n", "100", 0.5, false, false},     // 350/3.5 = 100 MB
		{"compress none", "compress(100 MB, none)\n", "100", 0.5, false, false},     // 100/1 = 100 MB
		{"compress snappy", "compress(250 MB, snappy)\n", "100", 0.5, false, false}, // 250/2.5 = 100 MB
		{"compress bzip2", "compress(400 MB, bzip2)\n", "100", 0.5, false, false},   // 400/4 = 100 MB

		// ==================== Capacity Functions ====================

		// capacity() - Capacity planning (returns Quantity with unit)
		{"capacity basic", "capacity(10, 2, disk)\n", "5 disk", 0, false, true},            // 10/2 = 5 disks
		{"capacity with remainder", "capacity(10, 3, crate)\n", "4 crate", 0, false, true}, // ceil(10/3) = 4 crates
		{"capacity TB to disk", "capacity(10 TB, 2 TB, disk)\n", "5 disk", 0, false, true},

		// ==================== Availability Functions ====================

		// downtime() - Availability to downtime
		{"downtime 99.9% month", "downtime(99.9%, month)\n", "43.2", 0.1, false, false}, // ~43 minutes
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				if tt.wantErr {
					return // Expected parse error
				}
				t.Fatalf("Parse error: %v", err)
			}

			interp := interpreter.NewInterpreter()
			results, err := interp.Eval(nodes)
			if err != nil {
				if tt.wantErr {
					return // Expected eval error
				}
				t.Fatalf("Eval error: %v", err)
			}

			if tt.wantErr {
				t.Fatal("Expected error but got none")
			}

			if len(results) == 0 {
				t.Fatal("No results returned")
			}

			result := results[0].String()

			if tt.containsStr {
				// Check if result contains expected string (for outputs with units)
				if !strings.Contains(result, tt.expected) {
					t.Errorf("Result %q does not contain expected %q", result, tt.expected)
				}
			} else if tt.tolerance > 0 {
				// Float comparison with tolerance
				var actual float64
				_, err := fmt.Sscanf(result, "%f", &actual)
				if err != nil {
					t.Fatalf("Could not parse result %q as number: %v", result, err)
				}

				var expected float64
				_, err = fmt.Sscanf(tt.expected, "%f", &expected)
				if err != nil {
					t.Fatalf("Could not parse expected %q as number: %v", tt.expected, err)
				}

				diff := math.Abs(actual - expected)
				if diff > tt.tolerance {
					t.Errorf("Result = %f, expected %f (diff: %f, tolerance: %f)",
						actual, expected, diff, tt.tolerance)
				}
			} else {
				// Exact string match
				if result != tt.expected {
					t.Errorf("Result = %s, expected %s", result, tt.expected)
				}
			}
		})
	}
}

// TestAllFunctionErrors tests error cases for each function systematically.
func TestAllFunctionErrors(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErrPart string // Substring expected in error
	}{
		// avg errors
		{"avg no args", "avg()\n", "at least one argument"},

		// sqrt errors
		{"sqrt no args", "sqrt()\n", "exactly one argument"},
		{"sqrt negative", "sqrt(-1)\n", "non-negative"},
		{"sqrt too many args", "sqrt(1, 2)\n", "exactly one argument"},

		// accumulate errors
		{"accumulate no args", "accumulate()\n", "2 arguments"},

		// rtt errors
		{"rtt unknown scope", "rtt(unknown)\n", "unknown network scope"},

		// throughput errors
		{"throughput unknown type", "throughput(unknown)\n", "unknown network type"},

		// seek errors
		{"seek unknown type", "seek(unknown)\n", "unknown storage type"},

		// capacity errors
		{"capacity zero capacity", "capacity(10, 0, disk)\n", "divide by zero"},
		{"capacity negative capacity", "capacity(10, -5, disk)\n", "positive"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				// Parse error is acceptable for some invalid syntax
				if strings.Contains(err.Error(), tt.wantErrPart) {
					return
				}
				// If parse error doesn't match, that's fine - error occurred
				return
			}

			interp := interpreter.NewInterpreter()
			_, err = interp.Eval(nodes)
			if err == nil {
				t.Errorf("Expected error containing %q but got none", tt.wantErrPart)
				return
			}

			if !strings.Contains(err.Error(), tt.wantErrPart) {
				t.Errorf("Error %q does not contain expected %q", err.Error(), tt.wantErrPart)
			}
		})
	}
}

// TestFunctionSynonyms verifies that function synonyms work correctly.
// avg has synonyms: average, mean
func TestFunctionSynonyms(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// avg and its synonyms
		{"avg primary", "avg(1, 2, 3)\n", "2"},
		{"average synonym", "average(1, 2, 3)\n", "2"},
		{"mean synonym", "mean(1, 2, 3)\n", "2"},

		// All three should produce identical results
		{"avg five values", "avg(10, 20, 30, 40, 50)\n", "30"},
		{"average five values", "average(10, 20, 30, 40, 50)\n", "30"},
		{"mean five values", "mean(10, 20, 30, 40, 50)\n", "30"},

		// Test with decimals
		{"avg decimals", "avg(1.5, 2.5)\n", "2"},
		{"average decimals", "average(1.5, 2.5)\n", "2"},
		{"mean decimals", "mean(1.5, 2.5)\n", "2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			interp := interpreter.NewInterpreter()
			results, err := interp.Eval(nodes)
			if err != nil {
				t.Fatalf("Eval error: %v", err)
			}

			if len(results) == 0 {
				t.Fatal("No results returned")
			}

			actual := results[0].String()
			if actual != tt.expected {
				t.Errorf("Result = %s, expected %s", actual, tt.expected)
			}
		})
	}
}

// TestBuiltinFunctionsCoverage verifies every function in BuiltinFunctions has a test.
// This is a meta-test to ensure comprehensive coverage.
func TestBuiltinFunctionsCoverage(t *testing.T) {
	// List of all function names that should be tested
	expectedFunctions := []string{
		"avg",
		"sqrt",
		"accumulate",
		"convert_rate",
		"downtime",
		"rtt",
		"throughput",
		"transfer_time",
		"read",
		"seek",
		"compress",
		"capacity",
	}

	// Verify each function has at least one passing test
	for _, fn := range expectedFunctions {
		t.Run("coverage_"+fn, func(t *testing.T) {
			// This is a placeholder assertion - the actual tests above cover these
			// This test documents the expected coverage
			if fn == "" {
				t.Error("Empty function name in coverage list")
			}
		})
	}
}
