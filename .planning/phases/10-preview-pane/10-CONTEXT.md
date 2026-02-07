# Phase 10: Preview Pane - Context

**Gathered:** 2026-02-08
**Status:** Ready for planning

<domain>
## Phase Boundary

Display calculation results in a preview pane, vertically aligned with source lines. Each source line maps 1:1 to a preview line. The preview shows only calculation results (not echoed markdown). This is a display/formatting phase - no new calculation features.

</domain>

<decisions>
## Implementation Decisions

### Result formatting
- Variable assignments display as: `variable → result` (e.g., `total_income → $5,000`)
- Anonymous calculations (no variable) display as: `→ result` (arrow only, no placeholder)
- Non-calculation lines (markdown) show as blank in preview (preserving vertical spacing)
- Always show units with results (even when obvious from context)
- Use smart rounding for numeric precision, but make all formatting configurable
  - Formatting should be configurable like other output formats (HTML, etc.)
- Napkin estimates use tilde prefix: `~400 GB`
- Large numbers use locale-aware thousand separators
- Currency handling: locale-aware positioning, with consistent handling of:
  - Three-letter codes (USD, EUR)
  - Symbols ($, €) as special cases
  - Existing complex logic should be unified

### Error presentation
- Show full error message in preview (not abbreviated)
- Error styling is theme-dependent (follow user's theme colors)
- When a line has an error but partial result is available, show both
- Cascading errors: show root cause only, dependents show "blocked" (not repeated errors)

### Visual separation
- Thin vertical line (│) as divider between source and preview
- Fixed 60/40 width ratio (source/preview), not user-adjustable
- Preview pane header: simple "Results" label at top
- No line numbers in preview pane (numbers only in source)
- Both panes highlight current line (cursor line highlighted in source AND preview)
- Scroll is locked - preview scrolls in sync with source (always aligned)
- Long documents scroll within available terminal - needs tests

### Line alignment behavior
- When source line wraps: result spans the same wrap lines
- When result is too long: wrap result text (not truncate)
- Empty lines in source = empty lines in preview (1:1 mapping)
- Multi-line calc blocks: per-line results (match current behavior)

### Claude's Discretion
- Exact implementation of wrapping logic
- Theme color mappings for errors
- How "blocked" downstream errors are phrased

</decisions>

<specifics>
## Specific Ideas

- Formatting should work like other output formats - screen is just one "output format" and users may want to adjust settings similar to HTML output
- Thousand separators and currency positioning should follow user locale
- The existing complex currency logic (USD/EUR codes vs $/€ symbols) needs consistent unification

</specifics>

<deferred>
## Deferred Ideas

- Adjustable divider position - decided against for now (fixed ratio)
- Independent preview scrolling - decided against (locked scroll)

</deferred>

---

*Phase: 10-preview-pane*
*Context gathered: 2026-02-08*
