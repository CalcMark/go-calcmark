package lsp

import "github.com/CalcMark/go-calcmark/spec/types"

// runtimeTypeToArgType maps a concrete evaluator runtime type to the ArgType
// string used by ParamSpec. Unknown or nil values collapse to ArgTypeAny.
//
// Currency maps to ArgTypeQuantity because a currency value is semantically
// a quantity (amount + unit) for parameter-filtering purposes.
func runtimeTypeToArgType(v types.Type) types.ArgType {
	switch v.(type) {
	case nil:
		return types.ArgTypeAny
	case *types.Number:
		return types.ArgTypeNumber
	case *types.Percentage:
		return types.ArgTypePercentage
	case *types.Quantity:
		return types.ArgTypeQuantity
	case *types.Rate:
		return types.ArgTypeRate
	case *types.Duration:
		return types.ArgTypeDuration
	case *types.Currency:
		return types.ArgTypeQuantity
	default:
		return types.ArgTypeAny
	}
}

// argTypesCompatible reports whether an actual runtime ArgType satisfies
// a required parameter ArgType. ArgTypeAny on either side is always compatible;
// an empty required type accepts anything (used when no filter is active).
func argTypesCompatible(actual, required types.ArgType) bool {
	if required == "" || required == types.ArgTypeAny {
		return true
	}
	if actual == types.ArgTypeAny {
		return true
	}
	return actual == required
}
