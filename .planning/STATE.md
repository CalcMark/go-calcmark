# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-06)

**Core value:** Fast, offline, verifiable calculations in markdown documents with a simple editor
**Current focus:** v1.1 CalcMark Language — interpreter correctness and editor completion

## Current Position

Phase: 11 (Navigation)
Plan: 0 of TBD in current phase
Status: Ready to plan
Last activity: 2026-02-06 — Phase 10 complete (verified)

Progress: [████████░░░░░░░░░░░░] 28%

## Milestone v1.1 Scope

**9 phases, 48 requirements:**
- Phase 9: Interpreter Correctness (6 requirements) - COMPLETE
- Phase 9.1: Separate Validation from Execution (3 plans) - COMPLETE
- Phase 10: Preview Pane (5 requirements) - COMPLETE
- Phase 11: Navigation (6 requirements)
- Phase 12: Undo/Redo (5 requirements)
- Phase 13: Clipboard (4 requirements)
- Phase 14: File Operations (8 requirements)
- Phase 15: Help Update (2 requirements)
- Phase 16: Source Highlighting (6 requirements)
- Phase 17: Testing & Validation (7 requirements)

## Known Bugs

1. ~~`accumulate(5mb/s, 1 day) as napkin` returns "430K" instead of ~400GB~~ FIXED in 09-01
   - Root cause: impl/interpreter/napkin_eval.go line 29 strips units
   - Fix: 52af9f3 - type-preserving napkin conversion

2. `request_data = 50kb` displays as `48.8 KB` instead of `50 KB`
   - Reported: 2026-02-06
   - Expected: 50 KB (user assigned 50 kilobytes)
   - Actual: 48.8 KB (unexpected conversion)
   - Investigate: Is `kb` being interpreted differently than expected? Check unit parsing for kb/KB/Kb

3. `daily_load * request_data` fails with "eval error" when multiplying dimensionless count by data quantity
   - Reported: 2026-02-06
   - Context: `rps = 1.2K`, `daily_load = accumulate(rps/s, 1 day)` -> 103.68M, then `daily_load * request_data` fails
   - Expected: Should multiply count (103.68M) by data size (50 KB) to get total data
   - Investigate: Type system may not allow Number * Quantity multiplication

## Performance Metrics

**Velocity:** 4 min per plan (sample size: 4)

*Updated after each plan completion*

## Accumulated Context

### Decisions

Recent decisions affecting current work:

- [10-05]: TestEditorCatwalkPreviewPane uses per-test documents (not shared document)
- [10-05]: PREVIEW-XX tests in sidebyside_test.go alongside pane tests
- [10-04]: Show full error messages in preview (not abbreviated hints)
- [10-04]: Cascading errors show "blocked" instead of repeating root cause
- [10-04]: Don't set VarName on error lines (preserve original behavior)
- [10-03]: All currency codes convert to symbols when available (USD -> $, EUR -> €)
- [10-03]: Mid-range currency values (1000-9999) use thousand separators
- [10-03]: Negative sign before symbol (-$50.00, not $-50.00)
- [10-02]: IsNapkin field on Quantity struct (not separate type)
- [10-02]: Tilde applied in FormatQuantity, not during evaluation
- [10-01]: Preview pane header is "Results" (not "Preview")
- [10-01]: Anonymous calculations display as "-> result" with arrow prefix
- [10-01]: Source/preview pane ratio is fixed at 60/40
- [09.1-03]: Use impldoc alias for impl/document import in TUI editor tests
- [09.1-02]: External test package (document_test) to avoid spec/impl import cycle
- [09.1-02]: impldoc alias for impl/document import in tests
- [09.1-01]: EnvironmentWriter interface with Set() and SetExchangeRate() for dependency inversion
- [09-02]: All types.NewNumber usages in interpreter are intentional (no type erasure bugs)
- [09-02]: Currency / Number division is a language limitation, not a bug
- [09-02]: avg() and sqrt() correctly return Number (aggregate/transform functions)
- [09-03]: Tolerance-based float comparison (0.0001 relative) for roundtrip tests
- [09-03]: Document unsupported conversions (compound speed, multi-word area) rather than fail
- [09-04]: NL forms consume expressions greedily - use parentheses for explicit grouping
- [09-04]: Rate * scalar supported, but scalar * rate is not commutative
- [09-04]: Rate + rate direct addition not implemented - use accumulate
- [09-01]: Use display.NormalizeForDisplay for Quantity unit normalization in napkin conversion
- [09-01]: Duration units preserved exactly (no auto-normalization to larger units)
- [v1.0]: FunctionDef struct as single source of truth for function metadata
- [v1.0]: Alt+b/f for word navigation (works on macOS where Ctrl+Arrow captured)

### Pending Todos

1. `convert_rate` does not show correct preview result — `convert_rate(10 mb/s, per hour)` displays incorrectly (interpreter)

### Roadmap Evolution

- Phase 9.1 inserted after Phase 9: Separate Validation from Execution (URGENT) - NOW COMPLETE
  - Discovered while fixing error line display bug
  - spec/document/evaluate.go violates architecture rule (imports impl/)
  - Need clean separation: spec=validation, impl=execution
  - Resolution: Deleted spec/document/evaluate.go, all evaluation uses impl/document.Evaluator

### Blockers/Concerns

- spec/classifier/classifier.go also imports impl/interpreter (discovered in 09.1-01, not fixed - out of scope)

## Session Continuity

Last session: 2026-02-07
Stopped at: Completed 10-05-PLAN.md (Phase 10 complete)
Resume file: None
Next phase: Phase 11 (Navigation)

---
*Updated: 2026-02-07 — Completed 10-05-PLAN.md (Preview Pane Tests)*
