# Phase 9: Interpreter Correctness - Context

**Gathered:** 2026-02-06
**Status:** Ready for planning

<domain>
## Phase Boundary

Fix calculation bugs so all computations produce correct results with proper unit handling. This includes fixing the napkin conversion bug, auditing all conversion paths for type erasure, ensuring functions work in both standard and natural language forms, and verifying unit conversion roundtrips.

</domain>

<decisions>
## Implementation Decisions

### Napkin Behavior
- Fix type preservation: input type → output type (Quantity stays Quantity, not Number)
- Preserve type for all cases: Quantity, Rate, Duration, Currency
- No tilde (~) prefix in display — just the rounded value
- Auto-normalize to human-friendly units (422000 MB → 400 GB)
- Known bug location: `impl/interpreter/napkin_eval.go` line 29 extracts value, line 99 returns wrong type

### Error Handling
- Show error with suggestion: "Cannot add meters to seconds. Did you mean...?"
- Always include line number: "Line 5: Cannot add meters to seconds"
- Function type errors show signature: "avg(numbers...) - got string at position 1"
- Errors styled in different color (use existing Error color from palette.go)

### Natural Language Tolerance
- Follow lexer/semantic parser exactly — spec defines allowed forms
- Do not add new tolerance — just ensure spec is implemented correctly
- Comprehensive test coverage for ALL natural language forms defined in lexer

### Precision and Rounding
- Internal: highest precision that makes sense (no arbitrary truncation)
- Zero precision loss on unit conversion roundtrips
- Output formatters (Preview Pane, HTML, etc.) can display fewer digits for readability
- Calculation accuracy is non-negotiable; display formatting is separate concern

### Claude's Discretion
- Exact wording of error suggestions
- How to determine "human-friendly" unit scale for napkin normalization
- Test organization and naming conventions

</decisions>

<specifics>
## Specific Ideas

- "The semantic parser or lexer should already completely describe what `as napkin` does" — follow spec exactly
- Precision configuration is a future concern, not Phase 9 scope
- "Zero loss, but the OUTPUT format may use fmt" — internal accuracy vs display formatting

</specifics>

<deferred>
## Deferred Ideas

- Configurable precision per output formatter — future phase
- Temperature conversion validation (non-linear C/F/K) — noted in requirements as CORR-01

</deferred>

---

*Phase: 09-interpreter-correctness*
*Context gathered: 2026-02-06*
