# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-06)

**Core value:** Fast, offline, verifiable calculations in markdown documents with a simple editor
**Current focus:** v1.1 CalcMark Language — interpreter correctness and editor completion

## Current Position

Phase: 9.1 (Separate Validation from Execution) - COMPLETE
Plan: 3 of 3 in current phase
Status: Phase complete
Last activity: 2026-02-07 — Completed 09.1-03-PLAN.md

Progress: [████░░░░░░░░░░░░░░░░] 18%

## Milestone v1.1 Scope

**9 phases, 48 requirements:**
- Phase 9: Interpreter Correctness (6 requirements) - COMPLETE
- Phase 9.1: Separate Validation from Execution (3 plans) - COMPLETE
- Phase 10: Preview Pane (5 requirements)
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

## Performance Metrics

**Velocity:** 4 min per plan (sample size: 3)

*Updated after each plan completion*

## Accumulated Context

### Decisions

Recent decisions affecting current work:

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

None yet.

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
Stopped at: Completed 09.1-03-PLAN.md (Phase 9.1 COMPLETE)
Resume file: None

---
*Updated: 2026-02-07 — Phase 9.1 complete, ready for Phase 10*
