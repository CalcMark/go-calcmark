package lsp

import "github.com/CalcMark/go-calcmark/v2/spec/types"

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

// argTypesCompatible reports whether an actual runtime ArgType satisfies a
// required parameter ArgType.
//
//   - An empty required type accepts anything (used when no filter is active).
//   - A required type of ArgTypeAny accepts every actual type, including
//     unknown/any variables.
//   - An actual type of ArgTypeAny (produced by runtimeTypeToArgType for
//     unmapped runtime types like Boolean or Fraction) does NOT match a
//     specific required type. An unknown-typed variable should not silently
//     pass a type-specific filter — that would leak semantically-wrong
//     completions (e.g., a Boolean variable offered as an accumulate rate).
//   - Otherwise actual and required must match exactly.
func argTypesCompatible(actual, required types.ArgType) bool {
	if required == "" || required == types.ArgTypeAny {
		return true
	}
	// ArgTypeAmount accepts anything that can sit in an additive
	// expression — Number, Quantity, Currency. Percentage, Duration,
	// and Rate are excluded so the autosuggest dropdown for params
	// like `grow.increment` doesn't surface a `growth_rate` (5%)
	// variable as a valid pick.
	if required == types.ArgTypeAmount {
		return actual == types.ArgTypeNumber || actual == types.ArgTypeQuantity
	}
	return actual == required
}
