# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-06)

**Core value:** Fast, offline, verifiable calculations in markdown documents with a simple editor
**Current focus:** v1.1 CalcMark Language — interpreter correctness and editor completion

## Current Position

Phase: 9 of 17 (Interpreter Correctness)
Plan: 0 of TBD in current phase
Status: Ready to plan
Last activity: 2026-02-06 — Roadmap created for v1.1

Progress: [░░░░░░░░░░░░░░░░░░░░] 0%

## Milestone v1.1 Scope

**9 phases, 48 requirements:**
- Phase 9: Interpreter Correctness (6 requirements)
- Phase 10: Preview Pane (5 requirements)
- Phase 11: Navigation (6 requirements)
- Phase 12: Undo/Redo (5 requirements)
- Phase 13: Clipboard (4 requirements)
- Phase 14: File Operations (8 requirements)
- Phase 15: Help Update (2 requirements)
- Phase 16: Source Highlighting (6 requirements)
- Phase 17: Testing & Validation (7 requirements)

## Known Bugs

1. `accumulate(5mb/s, 1 day) as napkin` returns "430K" instead of ~400GB
   - Root cause: impl/interpreter/napkin_eval.go line 29 strips units

## Performance Metrics

**Velocity:** Not yet measured (v1.1 just started)

*Updated after each plan completion*

## Accumulated Context

### Decisions

Recent decisions affecting current work:

- [v1.0]: FunctionDef struct as single source of truth for function metadata
- [v1.0]: Alt+b/f for word navigation (works on macOS where Ctrl+Arrow captured)

### Pending Todos

None yet.

### Blockers/Concerns

None yet.

## Session Continuity

Last session: 2026-02-06
Stopped at: Roadmap creation complete
Resume file: None

---
*Updated: 2026-02-06 — v1.1 roadmap created*
