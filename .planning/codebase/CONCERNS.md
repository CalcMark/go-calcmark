# Codebase Concerns

**Analysis Date:** 2026-02-01

## Tech Debt

### Sqrt Implementation Uses Float64 Conversion

**Issue:** The `sqrt()` function in the interpreter loses precision by converting to float64 and back.
- Files: `impl/interpreter/functions.go:351-356`
- Impact: Arbitrary precision is compromised for square root calculations; decimal.Decimal precision advantages are negated for this operation
- Fix approach: Implement Newton's method for arbitrary-precision square root calculation using decimal.Decimal arithmetic throughout

### Missing Save-As Functionality in ModelV2

**Issue:** New files (without a filepath) cannot be saved via Ctrl+S.
- Files: `cmd/calcmark/tui/editor/model_v2.go:146`
- Impact: Users cannot save newly created documents; creating a new document and trying to save it fails silently
- Fix approach: Implement save-as dialog that prompts for filename when filepath is empty

### Incomplete Save Status Feedback

**Issue:** No user feedback when file save completes successfully.
- Files: `cmd/calcmark/tui/editor/model_v2.go:133`
- Impact: Users cannot confirm whether save operations succeeded; no visual confirmation in UI
- Fix approach: Display temporary status message (e.g., "Saved" notification) that appears for 2 seconds then clears

### Debug Comment in Editor Model

**Issue:** Production code contains debug logging comment that isn't implemented consistently.
- Files: `cmd/calcmark/tui/editor/model.go:1036`
- Impact: Error debugging is inconsistent; no centralized error logging strategy
- Fix approach: Either implement the debug logging system or remove the comment; establish consistent error reporting pattern

### Parser Date Validation Tests Incomplete

**Issue:** Date semantic validation tests are marked TODO and skipped.
- Files: `spec/parser/date_test.go:58, 94, 132, 145-148`
- Impact: Month-only and day-only date parsing is not validated; invalid dates like "December 32" are not caught
- Fix approach: Implement date semantic validation in parser to reject invalid dates; enable skipped tests

### Parser Quantity Tests Incomplete

**Issue:** Node type verification for quantity literals is marked TODO.
- Files: `spec/parser/quantity_test.go:65-66, 188, 205`
- Impact: Cannot verify that parser produces correct AST node types for quantities; hard to catch parser regressions
- Fix approach: Add assertions to verify QuantityLiteral vs CurrencyLiteral node types; verify unit normalization in AST

### Unit Conversion Test Placeholder

**Issue:** Unit conversion test has a TODO for a feature that may not be supported.
- Files: `impl/interpreter/in_unit_test.go:32`
- Impact: Chained operations like "10 meters * 2 in feet" may not work; feature gap or test coverage gap is unclear
- Fix approach: Clarify if multiply-then-convert should be supported; either implement or document as limitation

## Known Bugs

### Calculation Result Disappears on Ordered List Start

**Issue:** When starting an ordered list on a new line after a calculation, the calculation result disappears from the results.
- Symptoms: User types calc like "a = 2 * 5", presses Enter twice, then types "1. " on line 3; the calculation on line 1 is no longer evaluated
- Files: `cmd/calcmark/tui/editor/test_ordered_list_bug.go`
- Trigger: Type calculation → press Enter twice → type "1. " on new line
- Workaround: None; requires parser/evaluator fix
- Status: Reproducible test exists but bug is not fixed

### Empty Document Line Count Inconsistency

**Issue:** Empty documents show incorrect line counts after operations.
- Symptoms: Empty document should have 1 line but sometimes shows 2 lines or cursor line is negative
- Files: `cmd/calcmark/tui/editor/visual_state_test.go:71, 83, 88, 91, 121, 126`
- Trigger: Open empty document → press Enter
- Impact: Cursor positioning is unreliable for empty documents; rendering may fail

### Visual Line Alignment Failures During Typing

**Issue:** Preview pane line count diverges from source pane line count while typing.
- Symptoms: Preview shows different number of lines than source during active editing
- Files: `cmd/calcmark/tui/editor/alignment_test.go:112, 386`, `cmd/calcmark/tui/editor/interactive_alignment_test.go:91, 208`
- Trigger: Type content → preview width triggers rewrap → alignment breaks
- Impact: Visual 1:1 pane alignment is lost; renders look misaligned and confusing
- Root cause: Preview pane rewraps at different points than source; timing issues with cache invalidation

### Heading Preview Not Visible While Typing

**Issue:** Heading content typed in editor doesn't immediately appear in preview pane.
- Symptoms: Type heading text like "# Test" → preview doesn't show the heading while typing
- Files: `cmd/calcmark/tui/editor/interactive_alignment_test.go:208`
- Trigger: Type heading content in editor
- Impact: Live preview doesn't update with editBuf content; user can't see what they're typing in preview
- Root cause: Preview rendering may not use editBuf for heading content; cache invalidation timing issue

### Ordered List Bug Heading Rendering

**Issue:** Heading preview line is empty even though heading is in editBuf.
- Symptoms: When typing heading content, the preview line shows as empty
- Files: `cmd/calcmark/tui/editor/navigation_eval_test.go:276`
- Trigger: Type "1. " on line after calculation → preview line 1 doesn't show heading
- Impact: Preview pane gives no feedback on heading content until line is complete

### Scroll Offset Out of Bounds

**Issue:** ScrollOffset can exceed the actual visual line count, causing rendering issues.
- Symptoms: After scrolling, scrollOffset becomes >= visual line count, causing array bounds errors
- Files: `cmd/calcmark/tui/editor/model_test.go:3269-3278`
- Trigger: Scroll past end of document with cursor at very end of file
- Impact: Rendering crashes or shows incorrect lines; document becomes unviewable

## Security Considerations

### Token Count Limit Implemented But Not Everywhere

**Issue:** MaxTokenCount limit (10,000 tokens) is checked in parser, but may not apply to all code paths.
- Risk: Token bomb attacks could still succeed if input bypasses token limit checks
- Files: `spec/parser/rdparser.go:46-50`, `spec/parser/limits.go:9-11`
- Current mitigation: Token count validated in main parse path
- Recommendations: Audit all entry points to ensure token limits are checked consistently; add security tests for token bomb patterns

### Nesting Depth Limit Is Implemented

**Issue:** MaxNestingDepth (100 levels) is implemented correctly.
- Files: `spec/parser/rdparser.go:21-23, 155-170`
- Current mitigation: Depth tracking with security error on violation
- Status: ✅ Appears secure

### Security Roadmap Incomplete

**Issue:** Planned security features are not yet implemented.
- Risk: No security audit completed; no fuzzing infrastructure
- Files: `SECURITY.md:121-129`
- Missing:
  - [ ] Security audit (v1.0.0)
  - [ ] Fuzzing infrastructure (v0.3.0)
  - [ ] Security benchmarks (v0.2.0)
- Recommendations: Prioritize fuzzing infrastructure before public release; conduct security audit with third party

### WASM Global Context Not Reset Between Evaluations

**Issue:** Global context in WASM persists across calls, potentially leaking state between different evaluations.
- Risk: If WASM is used with untrusted inputs, variable state from one evaluation could affect another
- Files: `impl/wasm/main.go:29`
- Current mitigation: `resetContext()` function exists but must be called explicitly
- Recommendations: Document that resetContext() must be called between user sessions; consider auto-resetting per evaluation if isolation is required

## Performance Bottlenecks

### Large File Line Count Operations

**Issue:** Operations on documents with 100+ lines may have performance issues due to alignment computation.
- Problem: AlignedModel computation for every render with large documents
- Files: `cmd/calcmark/tui/editor/view.go:63`, `cmd/calcmark/tui/editor/aligned.go`
- Cause: Alignment model is recomputed on every View() call when cache is invalidated; no incremental updates for small changes
- Improvement path: Implement incremental alignment updates for single-line changes; cache alignment per block instead of whole document

### Cache Invalidation Too Aggressive

**Issue:** Aligned model cache is invalidated on every keystroke, forcing full recomputation.
- Problem: Every character typed invalidates the cache and triggers realignment computation
- Files: `cmd/calcmark/tui/editor/model.go:1563-1586` (cache includes editBuf state)
- Cause: Cache key includes editBuf which changes on every keystroke
- Improvement path: Use smarter invalidation; only recompute pane alignment if width changes, defer full alignment until debounce completes

### Linear Search for Changed Blocks

**Issue:** Finding changed blocks uses linear iteration over entire document.
- Problem: O(n) search every debounce cycle for changed block IDs
- Files: `cmd/calcmark/tui/editor/model.go:1075`
- Cause: No index of changed blocks; relies on iterating changedBlockIDs map
- Improvement path: Maintain separate index of changed block line ranges for O(1) lookup

## Fragile Areas

### Two-Pane Alignment Architecture

**Issue:** Maintaining 1:1 visual line alignment between source and preview is inherently fragile.
- Files: `cmd/calcmark/tui/editor/aligned.go`, `cmd/calcmark/tui/editor/view.go:23-24`
- Why fragile:
  - Wrapping at different character positions in each pane
  - Preview width differs from source width
  - Line count changes during typing (insertions, wraps)
  - Cache invalidation timing issues
  - Bidirectional mapping must stay in sync or scrolling breaks
- Safe modification: Use catwalk tests (see `./cmd/calcmark/tui/editor/TESTING.md`) to verify alignment before changing wrapping or width logic
- Test coverage: `cmd/calcmark/tui/editor/alignment_test.go` (445 lines) covers many cases but gaps remain for edge cases

### Model State During Typing

**Issue:** Model has multiple state variables (editBuf, doc, cursorLine, results) that must stay in sync.
- Files: `cmd/calcmark/tui/editor/model.go:100-160` (Model struct fields)
- Why fragile:
  - EditBuf is temporary state that may diverge from document content
  - Results must be recomputed when editBuf changes but before debounce fires
  - Cursor line may be stale if document was reparsed with different line count
  - ChanhedBlockIDs tracking may miss blocks if code path is incomplete
- Safe modification: Ensure every keystroke: (1) updates editBuf, (2) recomputes results, (3) invalidates cache, (4) syncs cursor to document
- Test coverage: `cmd/calcmark/tui/editor/model_test.go` (4469 lines) has extensive tests but timing-sensitive bugs still occur

### View Rendering Pipeline

**Issue:** View() function has complex control flow for rendering aligned panes and headers.
- Files: `cmd/calcmark/tui/editor/view.go` (1003 lines)
- Why fragile:
  - Dimension calculations are interdependent (content height, pane height, globals height)
  - Off-by-one errors in height/width calculations cause misalignment
  - Globals panel height affects source pane padding (line 76-79)
  - Background color handling must be consistent across all components
- Safe modification: Use catwalk tests to verify visual output; change one dimension calculation at a time
- Test coverage: `cmd/calcmark/tui/editor/block_render_test.go` (504 lines) tests individual renderers but integration is complex

### Wrapped Line Handling

**Issue:** Wrapped lines must be tracked separately in both panes but render differently.
- Files: `cmd/calcmark/tui/editor/aligned.go:63-77` (AlignedLineKind constants)
- Why fragile:
  - Continuation lines don't get line numbers but must align with preview lines
  - Padding lines inserted when panes have different visual heights
  - Wrapped lines must be distinguishable from blank lines
  - Visual to source mapping breaks if wrapping changes
- Safe modification: Test wrapping with files containing long lines; verify alignments don't shift
- Test coverage: No dedicated wrapped line alignment tests

## Scaling Limits

### Document Size Scalability

**Current capacity:** Documents tested up to 100+ lines with manual scrolling
- Limit: Line count operations are O(n); alignment recomputation is O(n*m) for n lines and m panes
- No hard limit on file size in TUI (unlike CLI 1MB limit)
- Scaling path: At 1000+ lines, cache invalidation on every keystroke becomes noticeable; incremental alignment updates needed

### Search/Navigation Complexity

**Current capacity:** No search implemented
- Limit: N/A - feature doesn't exist
- Scaling path: If search is added, will need indexed search structures; naive string search would be O(n)

### Evaluation Performance

**Current capacity:** Single expressions evaluated sub-millisecond for typical calculations
- Limit: Complex expressions with deep nesting approach MaxNestingDepth limit
- Scaling path: No optimization for repeated subexpressions; caching would help for dependencies

## Dependencies at Risk

### Shopspring Decimal Precision Edge Cases

**Issue:** Arbitrary precision math library may have corner cases with very large or very small numbers.
- Risk: sqrt() float64 conversion loses precision; other operations might have similar issues
- Impact: Calculations with extreme values may be inaccurate
- Migration plan: Consider mpmath library if more operations need arbitrary precision; audit all float64 conversions

### Bubbletea/Lipgloss TUI Framework

**Issue:** Heavy reliance on bubbletea for event loop and lipgloss for rendering.
- Risk: TUI framework updates could break visual output; terminal compatibility issues
- Impact: Changes to library APIs require code updates; terminal-specific bugs are hard to diagnose
- Migration plan: No alternative identified; framework is appropriate for TUI; maintain version constraints

### Go Version Constraints

**Issue:** WASM compilation requires specific Go version.
- Risk: If Go changes WASM interface, compilation could fail
- Impact: Cannot update Go minor versions without testing WASM build
- Migration plan: Maintain CI tests for WASM compilation; pin Go version if needed

## Missing Critical Features

### Revert/Undo Functionality

**Issue:** No undo stack implemented in editor.
- Problem: User typing mistakes cannot be undone; requires manual correction
- Blocks: Cannot call this a production editor without undo
- Priority: High - essential for usability

### Find/Replace

**Issue:** No find or find-and-replace functionality.
- Problem: Users cannot navigate to specific content in large documents
- Blocks: Cannot efficiently edit documents with 100+ lines
- Priority: Medium - important for usability but not blocking basic use

### Documentation/Help System

**Issue:** Help viewer exists but content may be incomplete or not integrated.
- Problem: Users have no in-app reference for language features or keyboard shortcuts
- Blocks: New users cannot discover how to use the editor
- Priority: Medium - needed before public release

## Test Coverage Gaps

### Empty Document Edge Cases

**Issue:** Multiple tests fail or are fragile for empty documents.
- What's not tested: Comprehensive empty document state machine testing
- Files: `cmd/calcmark/tui/editor/empty_editor_test.go` (765 lines) has tests but edge cases remain
- Risk: Empty document operations are unreliable; cursor position, line count, and rendering fail in different ways
- Priority: High - empty documents are the first state users see

### Ordered List and Markdown Interaction

**Issue:** Ordered list introduction breaks calculation results.
- What's not tested: Full catwalk test sequence for ordered list bug
- Files: `cmd/calcmark/tui/editor/test_ordered_list_bug.go` - test exists but is not in catwalk format
- Risk: User loses work when typing lists after calculations; reproducible but not in test suite
- Priority: High - blocks calculation + list workflows

### Alignment Stress Tests

**Issue:** Many alignment bugs only appear under specific wrapping and resizing conditions.
- What's not tested: Stress tests for terminal resize during editing; edge cases with very narrow/wide panes
- Files: `cmd/calcmark/tui/editor/alignment_test.go` - existing tests may not cover all resize scenarios
- Risk: Panes become misaligned under resize operations; visual corruption
- Priority: Medium - intermittent but recoverable with refresh

### Date Arithmetic

**Issue:** Date operations like "today + 7 days" may not be fully tested.
- What's not tested: Comprehensive date arithmetic test suite; leap years, month boundaries, etc.
- Files: `impl/interpreter/date_test.go` (594 lines) has tests but date edge cases are complex
- Risk: Date calculations produce wrong results for boundary cases
- Priority: Medium - feature exists but edge cases unknown

### WASM Integration Testing

**Issue:** WASM module is built but integration with JavaScript may have gaps.
- What's not tested: Full integration tests with JavaScript consumers; state management across calls
- Files: `impl/wasm/main.go` - no test files in wasm directory
- Risk: WASM consumers discover bugs in production
- Priority: Medium - WASM is first-class citizen but untested

## Architecture Concerns

### Model V2 Parallel Implementation

**Issue:** New ModelV2 exists alongside original Model; both are partially implemented.
- Files: `cmd/calcmark/tui/editor/model.go` (2001 lines), `cmd/calcmark/tui/editor/model_v2.go` (665 lines)
- Problem: Two implementations cause confusion; unclear which one is "current"; both have bugs
- Impact: Maintenance burden; fixes go in one version and not the other; code review confusion
- Fix approach: Decide which is the target implementation; migrate fully or revert

### Direct Document Mutation During Editing

**Issue:** Document object is mutated directly during EditBuf updates.
- Files: `cmd/calcmark/tui/editor/model.go:syncDocumentFromTextarea()`
- Problem: editBuf and document can diverge; mutation is not atomic
- Impact: If update fails partway, document state is inconsistent
- Fix approach: Use immutable document updates or clearly document the contract

---

*Concerns audit: 2026-02-01*
