package parser_test

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/parser"
)

// Benchmark simple expression parsing
// Run with: go test -bench=BenchmarkParseSimple -benchmem ./spec/parser
func BenchmarkParseSimple(b *testing.B) {
	input := "1 + 2\n"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark number with multiplier
func BenchmarkParseNumberMultiplier(b *testing.B) {
	input := "1.5M\n"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark complex expression with precedence
func BenchmarkParseComplex(b *testing.B) {
	input := "total = (price + tax) * quantity / 100\n"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark natural language function
func BenchmarkParseNaturalLanguage(b *testing.B) {
	input := "average of 10, 20, 30, 40, 50\n"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark nested function calls
func BenchmarkParseNestedFunctions(b *testing.B) {
	input := "avg(sqrt(16), sqrt(25), sqrt(36))\n"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark currency parsing
func BenchmarkParseCurrency(b *testing.B) {
	input := "$100.50\n"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark currency code
func BenchmarkParseCurrencyCode(b *testing.B) {
	input := "100 USD\n"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark quantity with unit
func BenchmarkParseQuantity(b *testing.B) {
	input := "10 meters\n"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark multi-line program
func BenchmarkParseMultiLine(b *testing.B) {
	input := `x = 10
y = 20
total = x + y
avg = total / 2
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Performance targets (adjust based on actual hardware):
// - Simple: < 5μs, < 500 bytes
// - Complex: < 20μs, < 2KB
// - Natural language: < 10μs, < 1KB
// - Multi-line: < 50μs, < 5KB
//
// Run with:
//   go test -bench=. -benchmem -benchtime=10s ./spec/parser
//
// Profile with:
//   go test -bench=. -cpuprofile=cpu.prof ./spec/parser
//   go tool pprof cpu.prof
