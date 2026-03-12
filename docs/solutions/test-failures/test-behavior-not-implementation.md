---
title: Test Expected Behavior, Not Implementation Details
category: test-failures
date: 2026-03-12
tags: [testing, tdd, behavioral-tests, tui, bubbletea]
components: [editor/dynamic_footer_height_test.go]
---

# Test Expected Behavior, Not Implementation Details

## Problem

Tests written to validate a dynamic footer height feature were **tautological** — they re-implemented the production formula and checked that the formula equaled itself. These tests could never fail unless the test was also updated, making them useless for catching regressions.

Examples of what went wrong:

```go
// BAD: Re-implements the formula and checks it equals itself
func TestGetVisibleHeightMatchesViewFormula(t *testing.T) {
    results := m.GetLineResults()
    footerHeight := m.contextFooterHeight(results)
    expected := max(m.height-components.StatusBarHeight-footerHeight-2, 5)
    got := m.getVisibleHeight()
    if got != expected { ... } // This can NEVER fail
}

// BAD: Checks internal state instead of user-visible output
func TestLayoutSumsToTerminalHeight(t *testing.T) {
    contentHeight := m.getVisibleHeight()
    footerHeight := m.contextFooterHeight(results)
    total := contentHeight + footerHeight + components.StatusBarHeight + 2
    if total != m.height { ... } // Tests arithmetic, not behavior
}
```

## Root Cause

Confusing "testing the system" with "re-deriving the system." When a test duplicates production logic, it proves nothing — it's a mirror, not a validator. The test should express **what the user sees**, not **how the code computes it**.

## Solution

Render the full `View()` output and assert on **user-visible behavior**:

```go
// GOOD: Tests what the user sees
func TestFooterShowsFullHintOnErrorLine(t *testing.T) {
    doc, _ := document.NewDocument("result = undefined_var * 2\n")
    m := New(doc)
    m.width = 80
    m.height = 24
    m.cursorLine = 0

    view := m.View().Content
    plain := stripAnsi(view)

    // The hint should be visible in the rendered output
    if !strings.Contains(plain, "Defined variables") {
        t.Errorf("Footer should show the full diagnostic hint.\nView:\n%s", plain)
    }
}

// GOOD: Tests a rendering invariant the user depends on
func TestViewHasCorrectLineCountAlways(t *testing.T) {
    tests := []struct {
        name       string
        source     string
        cursorLine int
        height     int
    }{
        {"normal doc", "x = 10\ny = 20\n", 0, 24},
        {"cursor on error", "result = bad * 2\n", 0, 24},
        {"small terminal with error", "result = bad * 2\n", 0, 12},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            doc, _ := document.NewDocument(tt.source)
            m := New(doc)
            m.width = 80
            m.height = tt.height
            m.cursorLine = tt.cursorLine

            view := m.View().Content
            lines := strings.Split(view, "\n")
            if len(lines) != tt.height {
                t.Errorf("View should have exactly %d lines, got %d",
                    tt.height, len(lines))
            }
        })
    }
}
```

### The Pattern

| Aspect | Bad (Implementation) | Good (Behavioral) |
|--------|---------------------|-------------------|
| What it tests | Internal formula | Rendered output |
| How it validates | Re-derives the computation | Checks user-visible content |
| Regression value | Zero (tautological) | High (catches real breaks) |
| Maintenance cost | Must update with every refactor | Stable across refactors |

### Behavioral Test Categories for TUI

1. **Content presence** — "Does the rendered view contain this text?" (`strings.Contains(plain, "Defined variables")`)
2. **Rendering invariants** — "Does the view always produce exactly N lines?" (`len(lines) == m.height`)
3. **Visibility** — "Is the cursor line visible when the footer expands?" (`strings.Contains(plain, "undefined_var")`)
4. **State transitions** — "Does moving cursor between error/non-error lines produce different views?" (render twice, compare)
5. **Priority suppression** — "Does autocomplete suppress footer expansion?" (set mode, check line count)

### When Pure Function Tests Are Still Valuable

Pure helper functions like `countWrappedLines(text, width)` are fine to test directly — they have clear inputs/outputs with no coupling to rendering state:

```go
// GOOD: Pure function with obvious expected values
func TestCountWrappedLines(t *testing.T) {
    tests := []struct{ text string; width, want int }{
        {"hello world", 80, 1},
        {"123456789012345678901", 20, 2},
    }
    // ...
}
```

The rule: test *behavior* at the integration boundary (rendered View), test *computation* at the unit boundary (pure functions). Never test by re-implementing the thing you're testing.

## Prevention

- Before writing a test, ask: "If I deleted the production code and rewrote it differently, would this test still catch a bug?" If not, the test is tautological.
- For TUI tests, default to rendering `View()` and asserting on the output string. Only test internal helpers when they're pure functions with obvious expected values.
- Name tests after what the *user* expects, not what the *code* does: `TestFooterShowsFullHintOnErrorLine` not `TestContextFooterHeightReturns4`.
