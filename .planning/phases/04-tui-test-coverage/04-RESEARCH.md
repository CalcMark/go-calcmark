# Phase 4: TUI Test Coverage - Research

**Researched:** 2026-02-03
**Domain:** Data-driven TUI testing with catwalk, test flakiness elimination
**Confidence:** HIGH

## Summary

Phase 4 focuses on achieving comprehensive catwalk test coverage for all TUI editor interactions and eliminating flaky video-based tests from CI. The project already uses catwalk (knz/catwalk) built on cockroachdb/datadriven for data-driven testing of Bubble Tea models.

The research reveals two primary concerns:
1. **Pre-existing test failures** in TestEditorCatwalk caused by shared document mutation between test files
2. **VHS tape tests** that exist in `testdata/vhs_tapes/` but are NOT currently run in CI (only manual via `task test:vhs`)

**Primary recommendation:** Fix the shared document mutation root cause in TestEditorCatwalk, then expand test coverage systematically. The VHS tapes should be removed or archived since they serve no CI purpose and the behaviors they test should be covered by catwalk tests.

## Standard Stack

The established libraries/tools for this domain:

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| knz/catwalk | latest | Data-driven TUI model testing | Purpose-built for Bubble Tea models, uses datadriven |
| cockroachdb/datadriven | v1.0.2 | Data-driven test file format | Battle-tested at CockroachDB, supports -rewrite flag |
| charmbracelet/bubbletea | v1.3.10 | TUI framework (test target) | Already used in project |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| muesli/termenv | (dep) | Terminal color profile | Force ASCII profile for consistent test output |
| charmbracelet/lipgloss | v1.1.1 | Styling library | Force color profile in test init() |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| catwalk | teatest | teatest runs full tea.Program; heavier, more flaky, harder to isolate |
| catwalk | charmbracelet/x/exp/golden | Golden file comparison only, no key simulation |
| datadriven | table-driven tests | Less readable, harder to maintain, no -rewrite |

**Installation:**
Already in go.mod. No additional dependencies needed.

## Architecture Patterns

### Recommended Test File Structure
```
cmd/calcmark/tui/editor/
├── testdata/                    # Catwalk test data files
│   ├── cursor_navigation        # One file per test scenario
│   ├── viewport_scrolling
│   ├── typing_text
│   ├── text_wrapping_40col
│   ├── evaluation_results
│   └── ...
├── catwalk_test.go             # Main test runner
├── catwalk_wrapping_test.go    # Wrapping-specific tests
├── catwalk_type_mismatch_test.go
└── ...                         # Dedicated test functions for fresh documents
```

### Pattern 1: Dedicated Test Functions with Fresh Documents
**What:** Each test scenario that modifies document state gets its own Go test function that creates a fresh `*document.Document`.
**When to use:** Always for tests that modify document content (typing, inserting lines, deleting)
**Why:** Avoids shared state pollution between test files in datadriven.Walk
**Example:**
```go
// Source: Existing pattern from catwalk_test.go
func TestEditorCatwalkViewportScrolling(t *testing.T) {
    // Create fresh document for this test scenario
    content := `# Viewport Scrolling Test
line 2
line 3
...`
    doc, err := document.NewDocument(content)
    if err != nil {
        t.Fatalf("Failed to create document: %v", err)
    }

    datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
        if !strings.HasSuffix(path, "viewport_scrolling") {
            return
        }

        m := New(doc)
        m.width = 80
        m.height = 16 // Small viewport for scroll testing
        m.previewMode = PreviewFull

        catwalk.RunModel(t, path, m,
            catwalk.WithObserver("debug", ...),
            catwalk.WithObserver("scroll", ...),
        )
    })
}
```

### Pattern 2: Custom Observers for Specific State
**What:** Define observers that extract specific model state for assertions
**When to use:** When `debug` or `view` observers don't capture the right state
**Example:**
```go
// Source: Existing pattern from catwalk_test.go
catwalk.WithObserver("scroll", func(out io.Writer, m tea.Model) error {
    model := m.(Model)
    var buf strings.Builder
    buf.WriteString(fmt.Sprintf("cursorLine=%d scrollOffset=%d totalLines=%d visibleHeight=%d\n",
        model.cursorLine, model.scrollOffset, model.TotalLines(), model.getVisibleHeight()))
    // Additional scroll-specific state...
    _, err := out.Write([]byte(buf.String()))
    return err
})
```

### Pattern 3: Test File Format
**What:** Datadriven test file with directives and expected output
**When to use:** All catwalk tests
**Example:**
```
# Test: Cursor navigation behaviors
# Verifies: arrow keys, Home/End, and line wrapping

# Initial state - cursor at line 0, col 0
run observe=debug
----
-- debug:
mode=0 cursorLine=0 cursorCol=0 ...

# Test 1: Down arrow moves to next logical line
run observe=debug
key down
----
-- debug:
mode=0 cursorLine=1 cursorCol=0 ...
```

### Pattern 4: Color Profile Isolation
**What:** Force ASCII color profile in test init() for consistent output
**When to use:** All catwalk test files
**Why:** Terminal color profiles vary; tests must be deterministic
**Example:**
```go
// Source: Existing pattern from catwalk_test.go
func init() {
    lipgloss.SetColorProfile(termenv.Ascii)
}
```

### Anti-Patterns to Avoid
- **Shared document across datadriven.Walk:** Document mutations from one test file corrupt subsequent files. ALWAYS create fresh documents in dedicated test functions.
- **Relying on test file execution order:** Files execute alphabetically in Walk. Don't assume any order.
- **Testing with view observer for state verification:** Use debug observer for state; view is for visual output only.
- **Timing-dependent tests:** Catwalk uses cmd_timeout (default 20ms) to ignore blinking cursor commands. Don't rely on real time.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Key sequence simulation | Custom tea.KeyMsg generation | catwalk `key` directive | Handles all key types, ctrl modifiers, special keys |
| Text typing simulation | Character-by-character KeyMsg | catwalk `type` directive | Generates proper KeyMsg sequence |
| State comparison | String diff | datadriven expected output format | Built-in diff output, -rewrite flag |
| Terminal resize testing | Manual WindowSizeMsg | catwalk `resize W H` directive | Clean syntax, consistent with other directives |
| Test output regeneration | Manual file editing | `go test ./... -args -rewrite` | Updates all expected outputs from actual |

**Key insight:** Catwalk and datadriven handle all the boilerplate of TUI testing. The only custom code needed is observers and document setup.

## Common Pitfalls

### Pitfall 1: Shared Document Mutation (Current Bug)
**What goes wrong:** TestEditorCatwalk creates one `*document.Document` before `datadriven.Walk` and shares it across all test files. Tests that modify document content (insert_line, type_new_line, scroll_navigation) mutate this shared document, corrupting state for subsequent tests.
**Why it happens:** `datadriven.Walk` iterates over test files alphabetically. Tests run sequentially with the same document instance.
**How to avoid:**
1. Create a fresh document in each dedicated test function
2. Add test files to the skip list in TestEditorCatwalk
3. Create corresponding TestEditorCatwalk<Feature> function
**Warning signs:** Test sees unexpected content like "jjjotestline 1" instead of "# Header"; test fails when run as part of suite but passes in isolation.

### Pitfall 2: editBuf State Confusion
**What goes wrong:** Tests assume j/k navigate between lines, but they actually type 'j'/'k' into editBuf when a line is being edited.
**Why it happens:** The editor is NOT modal (no vim-style modes). Navigation keys only work when editBuf is empty; otherwise they type characters.
**How to avoid:** Check editBuf state in debug output before expecting navigation behavior.
**Warning signs:** Debug output shows `editBuf="jjk"` instead of cursor movement.

### Pitfall 3: Escape Key Behavior
**What goes wrong:** Tests expect ESC to "exit edit mode" but the editor doesn't have modes.
**Why it happens:** Misunderstanding of non-modal architecture.
**How to avoid:** ESC clears editBuf and commits changes; navigation then works. Document this in test comments.
**Warning signs:** ESC doesn't seem to do anything; subsequent navigation keys type characters.

### Pitfall 4: Color Profile Inconsistency
**What goes wrong:** Tests pass locally but fail in CI with different ANSI codes in output.
**Why it happens:** Different terminal emulators/environments have different color profiles.
**How to avoid:** Always set `lipgloss.SetColorProfile(termenv.Ascii)` in test init().
**Warning signs:** Diffs show ANSI escape code differences.

### Pitfall 5: VHS Tape Flakiness
**What goes wrong:** VHS video tests are timing-dependent, terminal-emulator-dependent, and produce inconsistent screenshots.
**Why it happens:** VHS uses real terminal emulation with real timing (Sleep commands). Screen size, font, terminal app all affect output.
**How to avoid:** Don't use VHS for automated testing. Use catwalk for all CI tests.
**Warning signs:** Tests pass locally but fail in CI; tests are inconsistent across runs.

## Code Examples

Verified patterns from existing codebase:

### Complete Test Function Template
```go
// Source: cmd/calcmark/tui/editor/catwalk_test.go
func TestEditorCatwalk<Feature>(t *testing.T) {
    // 1. Create fresh document with appropriate content
    content := `# Test Document
x = 10
y = 20
z = 30`

    doc, err := document.NewDocument(content)
    if err != nil {
        t.Fatalf("Failed to create document: %v", err)
    }

    // 2. Walk testdata and filter to specific test file
    datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
        if !strings.HasSuffix(path, "<feature_name>") {
            return
        }

        // 3. Create fresh model for each test file
        m := New(doc)
        m.width = 80
        m.height = 24
        m.previewMode = PreviewFull

        // 4. Run with appropriate observers
        catwalk.RunModel(t, path, m,
            catwalk.WithObserver("debug", func(out io.Writer, m tea.Model) error {
                _, err := out.Write([]byte(m.(Model).Debug()))
                return err
            }),
            catwalk.WithObserver("lines", func(out io.Writer, m tea.Model) error {
                _, err := out.Write([]byte(m.(Model).DebugLines()))
                return err
            }),
            // Add feature-specific observers as needed
        )
    })
}
```

### Testdata File for Typing Text
```
# Test: Basic text typing
# Verifies: Characters appear in editBuf, cursor advances

# Initial state
run observe=debug
----
-- debug:
mode=0 cursorLine=0 cursorCol=0 ... editBuf=""

# Type some text
run observe=debug
type hello
----
-- debug:
mode=0 cursorLine=0 cursorCol=5 ... editBuf="hello# Header"

# Continue typing
run observe=debug
type  world
----
-- debug:
mode=0 cursorLine=0 cursorCol=11 ... editBuf="hello world# Header"
```

### Testdata File for Text Wrapping at 40 Columns
```
# Test: Text wrapping at narrow width (40 columns)
# Verifies: Long lines wrap correctly, alignment maintained

run observe=lines
----
-- lines:
sourceToVisual: map[0:0 1:1 ...]
Visual lines:
  [0] srcIdx=0 lineNum=1 wrap=false pad=false content="Short line"
  [1] srcIdx=1 lineNum=2 wrap=false pad=false content="This is a longer line th"
  [2] srcIdx=1 lineNum=2 wrap=true pad=false content="at wraps at 40 chars"
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| VHS tape tests | Catwalk data-driven tests | Pre-v1 | VHS is manual-only, not in CI |
| Shared document in Walk | Dedicated test functions | Phase 2-3 | Workaround in place, root fix needed |
| teatest for full programs | catwalk for model testing | Before Phase 1 | Lower-level, more focused tests |

**Deprecated/outdated:**
- VHS tapes: Keep for manual visual verification only. Remove from any CI consideration.
- TestEditorCatwalk shared document pattern: Migrated to dedicated test functions for tests that mutate document.

## Open Questions

Things that couldn't be fully resolved:

1. **Root cause fix for shared document mutation**
   - What we know: The shared document in TestEditorCatwalk causes test pollution. Workaround exists (dedicated test functions).
   - What's unclear: Whether to fix TestEditorCatwalk to create fresh documents per file, or migrate all tests to dedicated functions.
   - Recommendation: Phase 4 should fix the root cause by either:
     a. Make TestEditorCatwalk create fresh documents for each file (requires understanding datadriven.Walk lifecycle)
     b. Migrate ALL tests to dedicated functions (more work but cleaner)

2. **Pre-existing test failures**
   - What we know: insert_at_end, insert_line, scroll_navigation have failing expectations due to document mutation. The expected output in test files is wrong because it was generated with polluted document state.
   - What's unclear: What the correct expected output should be.
   - Recommendation: Fix shared document issue first, then regenerate expected output with `-rewrite` flag.

3. **VHS tape removal scope**
   - What we know: 23 VHS tapes exist in `testdata/vhs_tapes/`. `task test:vhs` runs them manually. They are NOT in CI.
   - What's unclear: Whether all VHS scenarios have catwalk equivalents already.
   - Recommendation: Inventory VHS tapes, ensure catwalk coverage for each scenario, then archive or delete VHS tapes.

## Sources

### Primary (HIGH confidence)
- cmd/calcmark/tui/editor/catwalk_test.go -- existing catwalk test infrastructure
- cmd/calcmark/tui/editor/TESTING.md -- testing documentation
- .planning/STATE.md -- known issues with shared document mutation

### Secondary (MEDIUM confidence)
- [knz/catwalk GitHub](https://github.com/knz/catwalk) -- catwalk library documentation
- [cockroachdb/datadriven GitHub](https://github.com/cockroachdb/datadriven) -- datadriven test format

### Tertiary (LOW confidence)
- WebSearch results for TUI testing best practices -- general guidance only

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - existing codebase uses these libraries with established patterns
- Architecture: HIGH - patterns extracted from existing working tests
- Pitfalls: HIGH - issues documented in STATE.md and verified via test runs

**Research date:** 2026-02-03
**Valid until:** 2026-03-03 (30 days - stable domain, patterns unlikely to change)

## Success Criteria Mapping

| Success Criterion | Research Finding | Implementation Approach |
|-------------------|------------------|-------------------------|
| SC1: Catwalk tests for typing, cursor, wrapping, scrolling, evaluation | Existing patterns in testdata/ | Extend with new test files |
| SC2: No VHS tape tests in CI | VHS already NOT in CI | Verify, then archive/delete tapes |
| SC3: Zero flaky failures in 10 runs | Root cause is shared document | Fix mutation, regenerate expectations |

## Coverage Gap Analysis

### Existing Coverage
| Interaction | Test File | Status |
|-------------|-----------|--------|
| Cursor navigation (arrows) | cursor_navigation | PASS |
| Home/End keys | cursor_navigation | PASS |
| Ctrl+Arrow word movement | word_movement | PASS |
| Viewport scrolling | viewport_scrolling | PASS |
| Page Up/Down | viewport_scrolling | PASS |
| Variable editing | edit_variable_no_redef | PASS |
| Type mismatch errors | error_wrong_line_type_mismatch | PASS |
| Evaluation results | evaluation_debounce | PASS |
| Dependent variables | dependent_results | PASS |
| Text wrapping alignment | wrapping_alignment | PASS |
| Calc line wrapping | wrapping_calc_lines | PASS |
| Layout alignment 80col | layout_alignment_at_80 | PASS |

### Missing/Failing Coverage
| Interaction | Test File | Status | Issue |
|-------------|-----------|--------|-------|
| Insert line below | insert_line | FAIL | Shared document mutation |
| Insert at end | insert_at_end | FAIL | Shared document mutation |
| Scroll after insert | scroll_navigation | FAIL | Shared document mutation |
| Delete empty line | delete_empty_line | FAIL | Shared document mutation |
| Typing text (basic) | NONE | MISSING | Need new test |
| Narrow width (40col) | NONE | MISSING | Need new test |
| Long document scrolling | NONE | MISSING | Need new test |

### Required New Tests (per SC1)
1. **typing_text** - Basic text input, backspace, delete
2. **text_wrapping_40col** - Wrapping behavior at 40 column width
3. **long_document_scroll** - Scrolling through 50+ line document
