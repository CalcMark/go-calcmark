package display

import (
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/types"
	"github.com/shopspring/decimal"
)

// Arrays display as `[a, b]` with each element formatted the way it
// would be on its own; Text displays verbatim (go-calcmark#118, R14).
func TestFormat_ArrayAndText(t *testing.T) {
	f := DefaultFormatter()
	arr, _ := types.NewArray([]types.Type{
		types.NewCurrency(decimal.NewFromInt(1250), "$"),
		types.NewCurrency(decimal.NewFromFloat(75.5), "$"),
	})
	if got := f.Format(arr); got != "[$1,250.00, $75.50]" {
		t.Errorf("Format(array) = %q", got)
	}
	empty, _ := types.NewArray(nil)
	if got := f.Format(empty); got != "[]" {
		t.Errorf("Format(empty array) = %q", got)
	}
	if got := f.Format(types.NewText("Senior")); got != "Senior" {
		t.Errorf("Format(text) = %q", got)
	}
}
