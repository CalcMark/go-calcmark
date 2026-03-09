package types

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

var hundred = decimal.NewFromInt(100)

// Percentage represents a percentage value.
// The Value field stores the fractional form (e.g., 0.32 for 32%).
// Percentage preserves its identity through variables and applies
// widening on + and - operators: value + pct = value * (1 + pct).
type Percentage struct {
	Value decimal.Decimal
}

// NewPercentage creates a Percentage from a fractional decimal value.
// For 32%, pass 0.32.
func NewPercentage(value decimal.Decimal) *Percentage {
	return &Percentage{Value: value}
}

// String returns the percentage in human-readable form (e.g., "32%").
func (p *Percentage) String() string {
	display := p.Value.Mul(hundred)

	// Strip trailing zeros for clean display
	s := display.String()
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}

	return fmt.Sprintf("%s%%", s)
}
