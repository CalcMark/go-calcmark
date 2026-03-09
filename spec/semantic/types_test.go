package semantic

import "testing"

func TestCheckTypeCompatibility_PercentageMulDiv(t *testing.T) {
	// The runtime normalizes percentages to plain numbers before dispatch,
	// so Percentage * or / any type should be allowed by the semantic checker.
	percentageTypes := []struct {
		name string
		kind TypeKind
	}{
		{"Number", TypeNumber},
		{"Currency", TypeCurrency},
		{"Duration", TypeDuration},
		{"Quantity", TypeQuantity},
		{"Percentage", TypePercentage},
	}

	pctInfo := TypeInfo{Kind: TypePercentage}

	for _, op := range []string{"*", "/"} {
		for _, tt := range percentageTypes {
			otherInfo := TypeInfo{Kind: tt.kind}

			// Percentage op Other
			t.Run("Percentage"+op+tt.name, func(t *testing.T) {
				diag := CheckTypeCompatibility(pctInfo, otherInfo, op, nil)
				if diag != nil {
					t.Errorf("Percentage %s %s should be allowed, got diagnostic: %s", op, tt.name, diag.Message)
				}
			})

			// Other op Percentage
			t.Run(tt.name+op+"Percentage", func(t *testing.T) {
				diag := CheckTypeCompatibility(otherInfo, pctInfo, op, nil)
				if diag != nil {
					t.Errorf("%s %s Percentage should be allowed, got diagnostic: %s", tt.name, op, diag.Message)
				}
			})
		}
	}
}

func TestCheckTypeCompatibility_PercentageWidening(t *testing.T) {
	// value +/- Percentage is valid (percentage widening)
	pctInfo := TypeInfo{Kind: TypePercentage}

	wideningTypes := []struct {
		name string
		kind TypeKind
	}{
		{"Number", TypeNumber},
		{"Currency", TypeCurrency},
		{"Duration", TypeDuration},
	}

	for _, op := range []string{"+", "-"} {
		for _, tt := range wideningTypes {
			otherInfo := TypeInfo{Kind: tt.kind}
			t.Run(tt.name+op+"Percentage", func(t *testing.T) {
				diag := CheckTypeCompatibility(otherInfo, pctInfo, op, nil)
				if diag != nil {
					t.Errorf("%s %s Percentage should be allowed (widening), got diagnostic: %s", tt.name, op, diag.Message)
				}
			})
		}
	}
}
