# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-06)

**Core value:** Fast, offline, verifiable calculations in markdown documents with a simple editor
**Current focus:** v1.1 CalcMark Language — interpreter correctness and editor completion

## Current Position

Phase: 14 (File Operations)
Plan: 0 of TBD in current phase
Status: Not started
Last activity: 2026-02-09 — Completed Phase 13 (Clipboard)

Progress: [███████████████░░░░░] 73%

## Milestone v1.1 Scope

**11 phases, 48 requirements:**
- Phase 9: Interpreter Correctness (6 requirements) - COMPLETE
- Phase 9.1: Separate Validation from Execution (3 plans) - COMPLETE
- Phase 10: Preview Pane (5 requirements) - COMPLETE
- Phase 11: Navigation (6 requirements) - COMPLETE
- Phase 11.1: Bug Fixes (3 plans) - COMPLETE
- Phase 11.2: UX Redesign (3 plans) - COMPLETE
- Phase 12: Undo/Redo (5 requirements) - COMPLETE
- Phase 13: Clipboard (4 requirements) - COMPLETE
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

**Velocity:** 4 min per plan (sample size: 5)

*Updated after each plan completion*

## Accumulated Context

### Decisions

Recent decisions affecting current work:

- [13-03]: Ctrl+C copies when selection exists, quits when no selection (preserves Unix interrupt behavior)
- [13-03]: Paste forces undo boundaries before and after operation per RESEARCH.md
- [13-03]: Multi-line paste splits current line and inserts intermediate lines properly
- [13-03]: insertTextAtCursor and insertMultiLineText use recordEdit for undo integration
- [13-02]: Selection highlighting uses gray background (240) with white foreground (255)
- [13-02]: All navigation keys clear selection before moving cursor
- [13-02]: All typing keys clear selection before inserting/deleting
- [13-02]: Added selectionAnchorLine/Col to Debug() output for test verification
- [13-01]: Use -1 sentinel for selectionAnchorLine to indicate no selection
- [13-01]: HasSelection returns false when anchor equals cursor
- [13-01]: GetSelectionRange normalizes to start <= end for consistent text extraction
- [13-01]: DeleteSelection integrates with undo via recordEdit()
- [12-03]: All edit paths (including autocomplete mode) must call recordEdit() for undo
- [12-03]: transitionToProcessing() called before undo to flush editBuf
- [12-03]: Cursor restoration uses first operation in batch (stores pre-batch state)
- [12-02]: undoGroupingDelay = 1000ms (lower end of 1-2 second range)
- [12-02]: Follow evalDebounceMsg pattern for stale timer detection (batchID vs groupID)
- [12-02]: groupID never resets (stale timers remain invalidated across commits)
- [12-02]: CommitCurrentBatch and ForceBoundary are semantic aliases for CommitBatch
- [12-01]: Clear redo stack on new edits (standard undo/redo behavior)
- [12-01]: Pre-allocate history buffer to maxHistory capacity
- [12-01]: Store cursor/scroll BEFORE operations for restoration
- [12-01]: Operation reversal: Insert<->Delete, Replace swaps Old/New
- [11.1-03]: Only headings (# prefix) and calculation results shown in preview pane
- [11.1-03]: Filtered content (blockquotes, links, paragraphs) shows blank for vertical alignment
- [11.1-02]: Use Mul(targetSeconds/sourceSeconds) instead of Div(sourceSeconds/targetSeconds) for rate conversion to preserve integer precision
- [11.1-02]: Accept tiny precision loss (<1e-10) for large-to-small time unit conversions (unavoidable in shopspring/decimal)
- [11.1-01]: DELETE key fix verified - transitionToEditing() must be called BEFORE editBuf modification
- [11.1-01]: All TUI bug fixes require catwalk test reproducing exact user scenario
- [11.2-03]: File picker uses full-screen modal pattern (like StateHelp, StateCommandMenu)
- [11.2-03]: Two modes: ModePickerBrowse (navigate) and ModePickerNewFile (type filename)
- [11.2-03]: Existing files skip picker, save directly
- [11.2-01]: StateCommandMenu inserted after StateHelp in InputState enum
- [11.2-01]: Command menu captures arrow keys but dismisses on typing (like autocomplete)
- [11.2-01]: Help binding description updated to 'help/commands' to reflect dual purpose
- [11.2]: Ctrl+E restored to export mode (reverted readline navigation)
- [11.2]: Ctrl+A not used for line-start (reserved for select-all in Phase 13)
- [11.2]: Slash commands were removed because / is the CalcMark divide operator. REPL uses : prefix for commands.
- [11.2]: Status bar to show only Ctrl+Q and Ctrl+H (minimal, discoverable)
- [11.2]: Ctrl+H opens command menu popup (not just help overlay)
- [11-03]: Alt+B/F navigation uses same word boundary logic as Ctrl+Arrow
- [11-03]: Word boundary at punctuation (# treated as separate word)
- [11-02]: Use saveCurrentLineAndMoveTo() for scroll adjustment in Ctrl+Home/End
- [11-02]: Ctrl+End moves to last line and end of that line (not just last line)
- ~~[11-01]: Ctrl+E repurposed from export to line-end navigation (readline-style)~~ REVERTED
- ~~[11-01]: Export available via /export command only~~ REVERTED
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

1. ~~`convert_rate` does not show correct preview result — `convert_rate(10 mb/s, per hour)` displays incorrectly (interpreter)~~ FIXED in 11.1-02
2. ~~Delete last character on line bug — deleting last character on a line or under cursor behaves unexpectedly (tui)~~ FIXED in 11.1-01
3. ~~Preview pane extra blank line — cursor on last calc before empty line causes extra blank line in preview (tui)~~ INVESTIGATED in 11.1-03, behavior now correct
4. ~~Preview pane shows markdown — blockquotes, links render in preview when only results should show (tui)~~ FIXED in 11.1-03
5. Add natural language examples to function help messages — e.g., `average of 2, 4, 3` (interpreter)

### Roadmap Evolution

- Phase 11.2 inserted after Phase 11.1: UX Redesign (URGENT) - Holistic command/help system redesign
  - User feedback: Ctrl+E export was broken by Phase 11 changes
  - User feedback: Slash commands conflict with / divide operator
  - User feedback: Status bar and help system inadequate
  - Resolution: Reverted Ctrl+A/E, created dedicated UX phase
- Phase 11.1 inserted after Phase 11: Bug Fixes (URGENT) - NOW COMPLETE
  - Fixed DELETE key behavior
  - Fixed convert_rate precision loss
  - Fixed preview pane markdown filtering
- Phase 9.1 inserted after Phase 9: Separate Validation from Execution (URGENT) - NOW COMPLETE
  - Discovered while fixing error line display bug
  - spec/document/evaluate.go violates architecture rule (imports impl/)
  - Need clean separation: spec=validation, impl=execution
  - Resolution: Deleted spec/document/evaluate.go, all evaluation uses impl/document.Evaluator

### Blockers/Concerns

- spec/classifier/classifier.go also imports impl/interpreter (discovered in 09.1-01, not fixed - out of scope)

## Session Continuity

Last session: 2026-02-09
Stopped at: Completed 13-03-PLAN.md (Clipboard Operations)
Resume file: None
Next: Phase 14 (File Operations) or 13-04 if exists

---
*Updated: 2026-02-09 — Completed 13-03 (Clipboard Operations)*
