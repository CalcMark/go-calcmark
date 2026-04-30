package interpreter

import (
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/types"
	"github.com/shopspring/decimal"
)

// Benchmarks for evalBinaryOperation — the interpreter's hot path.
// Run with: go test -bench=BenchmarkEvalBinaryOp -benchmem ./impl/interpreter

func BenchmarkEvalBinaryOp_NumberNumber(b *testing.B) {
	left := types.NewNumber(decimal.NewFromInt(42))
	right := types.NewNumber(decimal.NewFromInt(17))

	b.ResetTimer()
	for b.Loop() {
		_, _ = evalBinaryOperation(left, right, "+")
	}
}

func BenchmarkEvalBinaryOp_NumberMultiply(b *testing.B) {
	left := types.NewNumber(decimal.NewFromInt(100))
	right := types.NewNumber(decimal.NewFromFloat(1.08))

	b.ResetTimer()
	for b.Loop() {
		_, _ = evalBinaryOperation(left, right, "*")
	}
}

func BenchmarkEvalBinaryOp_CurrencyNumber(b *testing.B) {
	left := &types.Currency{
		Value:  decimal.NewFromFloat(99.99),
		Symbol: "$",
		Code:   "USD",
	}
	right := types.NewNumber(decimal.NewFromFloat(1.08))

	b.ResetTimer()
	for b.Loop() {
		_, _ = evalBinaryOperation(left, right, "*")
	}
}

func BenchmarkEvalBinaryOp_QuantityQuantity(b *testing.B) {
	left := &types.Quantity{Value: decimal.NewFromInt(100), Unit: "meters"}
	right := &types.Quantity{Value: decimal.NewFromInt(50), Unit: "meters"}

	b.ResetTimer()
	for b.Loop() {
		_, _ = evalBinaryOperation(left, right, "+")
	}
}

func BenchmarkEvalBinaryOp_PercentageWidening(b *testing.B) {
	left := types.NewNumber(decimal.NewFromInt(1000))
	right := types.NewPercentage(decimal.NewFromFloat(0.20))

	b.ResetTimer()
	for b.Loop() {
		_, _ = evalBinaryOperation(left, right, "+")
	}
}

func BenchmarkEvalBinaryOp_DurationDuration(b *testing.B) {
	left := &types.Duration{Value: decimal.NewFromInt(5), Unit: "days"}
	right := &types.Duration{Value: decimal.NewFromInt(3), Unit: "days"}

	b.ResetTimer()
	for b.Loop() {
		_, _ = evalBinaryOperation(left, right, "+")
	}
}
