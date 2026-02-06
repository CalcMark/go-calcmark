# Project Research Summary

**Project:** CalcMark v1.1 - Interpreter Correctness & Editor Completion
**Domain:** CLI/TUI Calculation Notepad with Interpreted Language
**Researched:** 2026-02-06
**Confidence:** HIGH

## Executive Summary

CalcMark v1.1 focuses on three critical areas: fixing interpreter correctness issues (particularly the napkin unit bug), completing editor UX features (undo/redo with keyboard shortcuts), and implementing robust file operations. The existing architecture is solid—no new dependencies are required. The Go ecosystem and bubbletea TUI framework are stable choices that already power the current implementation. The core challenge is surgical fixes to existing code rather than new feature development.

The recommended approach is to address interpreter correctness first (napkin bug), followed by undo/redo keyboard bindings, then comprehensive testing. The napkin bug is a type-erasure problem in `evalNapkinConversion()` that strips units from quantities during formatting. The fix is localized to one function but requires careful testing across all type conversions. Undo/redo infrastructure exists but lacks keyboard shortcuts and has a 100-state limit that should be removed. File operations are mostly complete but need atomic save implementation to prevent data loss.

Key risks are: (1) unit type preservation through all transformation pipelines, (2) cursor position tracking in undo states, and (3) atomic file writes to prevent corruption. These are all well-understood problems with established solutions. The codebase already demonstrates good architectural patterns—pure computation cores, explicit state machines, and clean dependency flow. v1.1 is polish and correctness, not architectural change.

## Key Findings

### Recommended Stack

The existing stack is sufficient—no new dependencies needed for v1.1 features. Go 1.24.12 (current stable with security patches), charmbracelet/bubbletea v1.3.10 (TUI framework), and charmbracelet/bubbles v0.21.0 (textarea component) provide all required functionality. The project correctly stays on v1 stable releases rather than v2 release candidates. Testing uses knz/catwalk for data-driven TUI tests, which is critical for reproducing user interaction bugs.

**Core technologies:**
- **Go 1.24.12**: Current stable runtime, patch security fixes. Stay on 1.24 for stability.
- **bubbletea v1.3.10**: Elm-architecture TUI framework. v1 stable, v2 still at RC—correct choice.
- **bubbles v0.21.0**: textarea component for multi-line editing with cursor and scrolling.
- **shopspring/decimal**: Decimal arithmetic for precise calculations without floating-point errors.
- **knz/catwalk**: Data-driven TUI testing—essential for validating editor key sequences.

**Implementation patterns:**
- **Memento pattern for undo/redo**: Store content snapshots rather than command pattern. Simpler and fits document-based editing.
- **State machine for file operations**: Existing `InputState` enum handles save prompts, save-as dialogs correctly.
- **Property-based testing for unit conversions**: Round-trip tests (A->B->A == A) validate conversion accuracy.

### Expected Features

CalcMark v1.1 is a correctness and polish milestone, not a feature expansion. All expected features are already partially implemented and need completion or bug fixes.

**Must have (table stakes):**
- Ctrl+Z/Ctrl+Y for undo/redo—universal editor shortcuts, currently missing
- Napkin preserves units—`as napkin` must show "~400 GB" not "430K"
- Quit with unsaved changes prompt—already implemented, needs verification
- Unlimited undo within session—current 100-state limit is arbitrary
- Atomic file saves—prevent data loss on crash during write

**Should have (quality improvements):**
- Character batching in undo—typing "hello" should undo as one action, not 5
- Cursor position restoration on undo—return cursor to where the edit occurred
- Unit conversion audit—verify MB vs MiB vs Mb handling is consistent
- Save-as with overwrite confirmation—prevent accidental file overwrites

**Defer (out of scope for v1.1):**
- Persistent undo across sessions—breaks user mental model, not expected
- Undo tree (vim style)—complex UX, niche use case
- Auto-save on timer—scope creep, breaks file-based mental model

### Architecture Approach

CalcMark uses a three-layer architecture: specification layer (spec/), interpreter layer (impl/), and TUI layer (cmd/calcmark/tui/). Dependencies flow one-way: spec never imports impl or cmd, impl never imports cmd. This enables WASM builds and language spec independence. The TUI layer uses pure computation cores (functional) wrapped in imperative Bubble Tea models (shell). State transitions are explicit with documented invariants.

**Major components:**
1. **Undo/Redo State** (model.go): Current `undoStack []string` stores content snapshots. Needs enhancement to store cursor position and remove 100-state limit.
2. **Unit System** (three-layer): canonical.go defines unit names, unit_library.go provides conversion functions, normalize.go handles display formatting.
3. **Napkin Formatter** (napkin_eval.go): Rounds numeric values for human readability. Current bug: strips unit type, returning bare Number instead of preserving Quantity/Rate/Currency types.
4. **File Operations** (model.go): Save, save-as, quit-with-prompt already implemented. Needs atomic write pattern (temp file + rename) to prevent corruption.
5. **Editor State Machine** (state.go): Explicit StateReady/StateEditing/StateProcessing transitions with invariant checking. Separate InputState for UI modals (save prompts, autocomplete).

**Key patterns:**
- **Pure computation cores**: All geometry, alignment, wrapping are pure functions. Bubble Tea models call these and manage state.
- **Document rebuild on change**: Content changes trigger full document re-parse and re-evaluate. Makes undo simple—just restore content string.
- **Memento for undo**: Store snapshots, not commands. Fits document editing better than command pattern granularity.

### Critical Pitfalls

1. **Unit Context Lost During Type Transformations** — The napkin formatter at `impl/interpreter/napkin_eval.go` line 29 extracts numeric value from Quantity but returns a bare Number, losing the unit. Fix: Preserve type hierarchy—if input is Quantity, return Quantity with rounded value and same unit. Test with `accumulate(5 MB/s, 1 day) as napkin` expecting "~400 GB".

2. **Undo/Redo Cursor Position Not Tracked** — Current `undoStack []string` stores only content, not cursor position. Undo restores content but leaves cursor at wrong location, causing disorientation and potential out-of-bounds errors. Fix: Store `UndoState{Content, CursorLine, CursorCol}` instead of plain strings.

3. **File Save Not Atomic (Data Loss Risk)** — `os.WriteFile()` truncates file to 0 bytes then writes. Power failure mid-write results in empty file. Fix: Write to temp file, atomically rename to target (POSIX guarantees atomic rename). Pattern: `os.WriteFile(tmpFile); os.Rename(tmpFile, target)`.

4. **Save Doesn't Flush Pending Edits** — Debounce mechanism means `editBuf` isn't committed until 100ms after typing stops. Ctrl+S during this window saves old content, missing last keystrokes. Fix: Flush `editBuf` to document before calling `getDocumentContent()`.

5. **Redo Stack Cleared on Any Edit** — Standard linear history behavior, but users may expect to make small edits without losing all redo history. Mitigation: Document this behavior clearly, or implement branching undo tree (more complex).

## Implications for Roadmap

Based on research, the milestone should be executed in three sequential phases addressing interpreter correctness, editor enhancements, and comprehensive testing. This ordering prioritizes user trust (correct calculations) before convenience (keyboard shortcuts).

### Phase 1: Interpreter Correctness (FIRST priority)

**Rationale:** Incorrect calculations destroy user trust in the tool. The napkin bug affects the core use case (accumulating rates over time). Fix this before any other work to ensure test infrastructure validates correct behavior.

**Delivers:**
- Napkin formatter preserves unit types through conversion
- Accumulate function returns correctly formatted quantities
- Test coverage for all type transformations (Quantity, Rate, Currency, Duration)

**Addresses:**
- Napkin preserves units (must-have from FEATURES.md)
- Unit conversion correctness requirements
- Type preservation through transformations

**Avoids:**
- Pitfall 1: Unit context lost during type transformations
- Pitfall 7: Unit prefix confusion (MB vs MiB vs Mb)

**Complexity:** LOW - localized fix in one function, but requires careful testing across type system.

### Phase 2: Undo/Redo Enhancement

**Rationale:** Keyboard shortcuts are table stakes for any text editor. Users expect Ctrl+Z/Y to work without thinking. The infrastructure exists (undo/redo methods), just needs wiring to key handlers. Removing the 100-state limit and adding cursor tracking are low-complexity enhancements.

**Delivers:**
- Ctrl+Z for undo, Ctrl+Y for redo (keyboard bindings)
- Unlimited undo within session (remove 100-state cap)
- Cursor position restored on undo (enhance UndoState struct)
- Edit buffer flushed before undo operations

**Addresses:**
- Ctrl+Z/Ctrl+Y shortcuts (must-have from FEATURES.md)
- Unlimited undo (must-have from FEATURES.md)
- Cursor position restoration (quality improvement)

**Avoids:**
- Pitfall 2: Undo/redo cursor position not tracked
- Pitfall 8: Undo before editBuf commit loses edit
- Pitfall 5: Redo stack cleared on any edit (document behavior)

**Uses:** Existing memento pattern from STACK.md recommendation. No new dependencies.

**Complexity:** LOW - wire existing functionality to keys, enhance snapshot struct.

### Phase 3: File Operations & Atomic Saves

**Rationale:** File corruption risk is critical but less common than interpreter bugs or keyboard shortcuts. Existing save/quit/save-as functionality works, but needs atomic write pattern to prevent data loss on crash. Quick verification phase to ensure edge cases are covered.

**Delivers:**
- Atomic file saves (temp file + rename pattern)
- Edit buffer flushed before save
- Existing quit-with-prompt verified via catwalk tests
- Save-as overwrite confirmation added

**Addresses:**
- Quit with unsaved changes prompt (must-have, verify existing)
- Atomic file saves (quality improvement)
- Save-as overwrite confirmation (should-have)

**Avoids:**
- Pitfall 3: File save not atomic (data loss risk)
- Pitfall 4: Save doesn't flush pending edits
- Pitfall 9: Save-as overwrites without warning

**Implements:** State machine architecture component—existing InputState handles prompts correctly.

**Complexity:** LOW - existing infrastructure works, add atomic write wrapper.

### Phase 4: Comprehensive Testing & Documentation

**Rationale:** After correctness fixes and UX enhancements, validate entire system with property-based tests, catwalk tests for TUI interactions, and golden tests for edge cases. This phase catches regressions and documents expected behavior.

**Delivers:**
- Property-based unit conversion tests (round-trip invariants)
- Catwalk tests for undo/redo/save/quit key sequences
- Golden tests for napkin formatting edge cases
- Unit conversion audit (MB vs MiB handling)

**Addresses:**
- Unit conversion correctness (must-have)
- Character batching in undo (defer to v1.2 if complex)
- Comprehensive real-world document testing (milestone goal)

**Avoids:**
- Pitfall 6: Undo granularity too fine (evaluate during testing)
- Pitfall 11: Rate time unit normalization inconsistency

**Complexity:** MEDIUM - test writing is straightforward but comprehensive coverage takes time.

### Phase Ordering Rationale

- **Interpreter first** because incorrect results destroy user trust. No point polishing UX if calculations are wrong.
- **Undo/redo second** because keyboard shortcuts are expected in any editor. Quick win with existing infrastructure.
- **File operations third** because atomic saves prevent rare but catastrophic data loss. Existing functionality works, just needs hardening.
- **Testing last** to validate all changes and catch regressions before declaring v1.1 complete.

**Dependencies:** Phase 1 must complete before Phase 4 (tests depend on correct interpreter behavior). Phases 2 and 3 are independent and could run parallel, but sequential is safer for git commit hygiene.

### Research Flags

**Phases with standard patterns (skip research-phase):**
- **Phase 1 (Interpreter):** Type preservation is well-understood. Direct codebase analysis identified the bug location and fix.
- **Phase 2 (Undo/Redo):** Memento pattern is established. Keyboard bindings are straightforward bubbletea message handling.
- **Phase 3 (File Operations):** Atomic write pattern is industry-standard (npm/write-file-atomic reference). State machine already implemented.
- **Phase 4 (Testing):** Catwalk usage is documented in project TESTING.md. Property-based testing patterns are well-known.

**No phases need `/gsd:research-phase` during planning.** All patterns are established, implementation is straightforward, codebase structure is clear. This is a correctness and polish milestone, not exploratory development.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Existing dependencies are stable, no additions needed. Verified via official release pages and codebase analysis. |
| Features | HIGH | All features are partially implemented. Requirements derived from user expectations and editor standards. |
| Architecture | HIGH | Codebase directly analyzed, patterns documented, bug locations identified. Three-layer architecture is solid. |
| Pitfalls | HIGH | Bug locations confirmed in source (napkin_eval.go line 29), patterns validated via editor UX research and atomic write references. |

**Overall confidence:** HIGH

### Gaps to Address

- **Character batching in undo** — Feature research identifies this as expected behavior, but implementation complexity is unclear. Recommend deferring to v1.2 unless testing reveals user complaints. Current debounce-based undo may be sufficient for typical document editing.

- **Unit prefix case sensitivity** — Canonical unit definitions in `spec/units/canonical.go` need audit to verify MB vs mb vs Mb handling. Research flags this as moderate pitfall, but exact current behavior is unclear. Should be validated during Phase 4 testing.

- **Temperature conversion** — Non-linear conversions (32F = 0C) are different from linear unit conversions. canonical.go lists temperature units, but conversion implementation wasn't verified. Test during Phase 4 to determine if this is a gap.

- **Save-as directory resolution** — Current code uses `filepath.Abs()` which defaults to cwd. User expectation is to default to current file's directory. Research identifies this as minor pitfall; validate during Phase 3 and add status bar feedback showing full path after save.

## Sources

### Primary (HIGH confidence)
- CalcMark codebase direct analysis: model.go, napkin_eval.go, state.go, rate.go, canonical.go, unit_library.go, normalize.go
- [Bubble Tea releases](https://github.com/charmbracelet/bubbletea/releases) — v1.3.10 verified as latest stable
- [Bubbles releases](https://github.com/charmbracelet/bubbles/releases) — v0.21.0 verified as latest stable
- [Go 1.24 release notes](https://go.dev/doc/go1.24) — 1.24.12 security patches confirmed
- [knz/catwalk](https://github.com/knz/catwalk) — TUI testing tool verification
- CalcMark TESTING.md — Catwalk usage patterns documented in project

### Secondary (MEDIUM confidence)
- [Undo/Redo Implementations in Text Editors](https://www.mattduck.com/undo-redo-text-editors) — Memento pattern recommendation
- [You Don't Know Undo/Redo](https://dev.to/isaachagoel/you-dont-know-undoredo-4hol) — User expectations for character batching
- [super_editor Wiki: Undo Redo](https://github.com/superlistapp/super_editor/wiki/Design-Thoughts:-Undo-Redo) — Cursor tracking requirements
- [Towards Atomic File Modifications](https://dev.to/martinhaeusler/towards-atomic-file-modifications-2a9n) — Atomic save patterns
- [npm/write-file-atomic](https://github.com/npm/write-file-atomic) — Reference implementation
- [Refactoring Guru: Memento Pattern](https://refactoring.guru/design-patterns/memento/go/example) — Go implementation example

### Tertiary (LOW confidence)
- [NIST Metrication Errors](https://www.nist.gov/pml/owm/metrication-errors-and-mishaps) — Real-world unit conversion disasters (context)
- [NASA Mars Climate Orbiter](https://sites.google.com/view/onlineunitconversions/four-tragedies-caused-by-erroneous-unit-conversion) — Famous unit mixup example (context)

---
*Research completed: 2026-02-06*
*Ready for roadmap: yes*
