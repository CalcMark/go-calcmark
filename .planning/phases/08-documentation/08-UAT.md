---
status: testing
phase: 08-documentation
source: [08-01-SUMMARY.md, 08-02-SUMMARY.md]
started: 2026-02-05T21:00:00Z
updated: 2026-02-05T21:00:00Z
---

## Current Test

number: complete
name: All tests passed
status: UAT complete

## Tests

### 1. Example files run without errors
expected: Run `cm eval testdata/examples/*.cm` - all three files should evaluate and produce calculation output
result: issue
reported: "the output for calculations appears to have a blank line between each calculation output in the plain text piped version. There should be one calculation output per line only."
severity: minor
fixed: "e3f07ed - Refactored text formatter to output one result per line in non-verbose mode"

### 2. TUI screenshot shows two-column layout
expected: Open docs/images/tui-screenshot.png - should show source code on left pane and calculated results on right pane
result: pass

### 3. Hero GIF shows calculation workflow
expected: Open docs/images/hero.gif - should show typing calculations and seeing results (via cm eval workflow)
result: pass

### 4. README explains what CalcMark is
expected: Open README.md - first few lines should explain CalcMark as a "calculation notepad" with calculations embedded in markdown
result: pass

### 5. README shows Homebrew installation
expected: README.md contains `brew install calcmark/tap/calcmark` in Installation section
result: pass

### 6. README Quick Start shows all three use cases
expected: Quick Start section shows: 1) TUI editor (`cm budget.cm`), 2) CLI eval (`cm eval budget.cm`), 3) Convert (`cm convert budget.cm --to=html`)
result: pass

### 7. README links work
expected: Click links in README - testdata/examples/*.cm files exist, docs/README.md exists, spec/LANGUAGE_SPEC.md exists
result: pass

## Summary

total: 7
passed: 6
issues: 1 (fixed)
pending: 0
skipped: 0

## Gaps

### Gap 1: Ctrl+Arrow word navigation not working on macOS
reported: "The help files says that CTRL arrows should work to move one word left or right, but that is not implemented"
analysis: Implementation existed in model.go but Ctrl+Arrow keys are captured by macOS for Mission Control. On macOS terminals, Option+Arrow sends Alt+b/Alt+f escape sequences.
fix: Added handling for Alt+b and Alt+f (what macOS terminals actually send for Option+Arrow). Used keydebug tool to verify actual key events.
files: cmd/calcmark/tui/editor/model.go, cmd/calcmark/tui/shared/keys.go
commit: d1cb95c
status: verified
