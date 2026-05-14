package semantic

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/CalcMark/go-calcmark/v2/spec/ast"
	"github.com/CalcMark/go-calcmark/v2/spec/document"
)

// CheckFrontmatter validates the populated CalcMark fields of a Frontmatter
// and returns diagnostics for malformed values. It is a separate exported
// function (not a method on Checker) because frontmatter validation is purely
// structural — it has no dependency on accumulated Checker state and can be
// called independently by tooling (LSP, calcmark-web) that wants to surface
// frontmatter problems without setting up a full Checker.
//
// Scope: only registered keys (see document.Registry) are validated. Entries in
// fm.Extra are passthrough by design — they carry no CalcMark semantics — and
// produce zero diagnostics here.
//
// What this DOES NOT duplicate from the parser:
//   - YAML shape errors (e.g., "globals: 42") — the parser already returns a
//     fatal error and the resulting Frontmatter is never constructed.
//   - convert_to/scale/measurement sub-key validation — the parser's
//     validateConvertToConfig, validateScaleConfig, and parseMeasurementConfig
//     already enforce those at parse time.
//
// What this DOES catch:
//   - Programmatically constructed Frontmatter values that bypass the parser
//     (e.g., a test or a future library caller that builds a Frontmatter by
//     hand and sets ConvertTo.System = "xyz").
//   - Cases the parser is too permissive about reaching the typed value, such
//     as exchange rates that became zero or negative after construction.
//
// Diagnostic anchors come from fm.KeyRanges (Unit 3). When a registered key is
// absent from KeyRanges (e.g., the Frontmatter was built programmatically
// without source positions), the diagnostic still emits but with a zero-value
// ast.Range as a fallback so callers can rely on Range != nil.
//
// Complexity: O(|registered keys present in fm|). Per D9/D10, the registry is
// small and walked linearly; no auxiliary indices are built.
func CheckFrontmatter(fm document.Frontmatter) []Diagnostic {
	var diags []Diagnostic

	// convert_to: System must be in the registry's EnumValues.
	// Comparison is case-sensitive — the parser already lowercases input, so any
	// non-canonical value reaching here came from programmatic construction and
	// should be flagged exactly as written.
	if fm.ConvertTo != nil {
		if d, ok := checkConvertTo(fm.ConvertTo, rangeFor(fm, "convert_to")); ok {
			diags = append(diags, d)
		}
	}

	// exchange: every rate must be positive (zero and negative rates would
	// silently yield zero or sign-flipped conversions). Walk in deterministic
	// (sorted-key) order so the diagnostic stream is stable across runs.
	if len(fm.Exchange) > 0 {
		exchangeRange := rangeFor(fm, "exchange")
		keys := make([]string, 0, len(fm.Exchange))
		for k := range fm.Exchange {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			rate := fm.Exchange[k]
			if !rate.IsPositive() {
				diags = append(diags, Diagnostic{
					Severity: Error,
					Code:     DiagFrontmatterValidation,
					Message: fmt.Sprintf(
						"exchange rate %q must be positive, got %s",
						k, rate.String(),
					),
					Range: exchangeRange,
				})
			}
		}
	}

	// scale: Factor must be positive. The parser's validateScaleConfig enforces
	// this for parsed input, but a programmatic caller can bypass it.
	if fm.Scale != nil {
		if !fm.Scale.Factor.IsPositive() {
			diags = append(diags, Diagnostic{
				Severity: Error,
				Code:     DiagFrontmatterValidation,
				Message: fmt.Sprintf(
					"scale factor must be positive, got %s",
					fm.Scale.Factor.String(),
				),
				Range: rangeFor(fm, "scale"),
			})
		}
	}

	// Sort by Range.Start.Line ascending so multi-key diagnostics arrive in
	// source order. A zero-value Range (line 0) sorts to the front, which is
	// fine — those are the no-source-position fallback cases.
	sort.SliceStable(diags, func(i, j int) bool {
		li, lj := 0, 0
		if diags[i].Range != nil {
			li = diags[i].Range.Start.Line
		}
		if diags[j].Range != nil {
			lj = diags[j].Range.Start.Line
		}
		return li < lj
	})

	return diags
}

// checkConvertTo validates the System field against the registered enum values.
// Returns (Diagnostic{}, false) when valid.
func checkConvertTo(cfg *document.ConvertToConfig, r *ast.Range) (Diagnostic, bool) {
	key, ok := document.LookupKey("convert_to")
	if !ok {
		// Defensive: the registry should always contain convert_to. If not,
		// we have nothing to validate against.
		return Diagnostic{}, false
	}
	if slices.Contains(key.EnumValues, cfg.System) {
		return Diagnostic{}, false
	}
	return Diagnostic{
		Severity: Error,
		Code:     DiagFrontmatterValidation,
		Message: fmt.Sprintf(
			"convert_to value %q is not valid; expected one of: %s",
			cfg.System, strings.Join(key.EnumValues, ", "),
		),
		Range: r,
	}, true
}

// rangeFor returns a non-nil *ast.Range for the named key. When KeyRanges
// lacks an entry (e.g., programmatic construction with no source positions),
// it returns a pointer to a zero-value Range so callers can always rely on
// Diagnostic.Range != nil.
func rangeFor(fm document.Frontmatter, key string) *ast.Range {
	if r, ok := fm.KeyRanges[key]; ok {
		return &r
	}
	return &ast.Range{}
}
