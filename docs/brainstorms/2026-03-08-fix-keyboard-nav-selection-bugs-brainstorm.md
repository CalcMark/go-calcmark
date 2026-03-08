---
title: "fix: Keyboard navigation and selection state inconsistencies"
type: fix
date: 2026-03-08
issue: 38
terminal: Ghostty (Kitty protocol)
---

# fix: Keyboard navigation and selection state inconsistencies

## What We're Building

Fix multiple keyboard navigation and selection state bugs in the TUI editor, primarily affecting Cmd+Arrow key combos in Ghostty.

## Problem

### Bug 1: Cmd+Right opens Export dialog

Pressing Cmd+Right triggers the Export dialog instead of moving the cursor to end-of-line. This is a regression — the key dispatch routes through Ctrl+E (Export) instead of the Super (Cmd) block that maps Cmd+Right to End-of-line.

### Bug 2: Cmd+Left selects and duplicates text

Pressing Cmd+Left appears to select text and possibly duplicate some of it, rather than moving to start-of-line.

### Bug 3: Cmd+Down/Up don't navigate to document boundaries

Cmd+Down should jump to end of document, Cmd+Up to start. With Shift held, should select to those boundaries.

### Bug 4: Alt+Up/Down not handled (Terminal.app fallback)

In Terminal.app, Cmd keys are consumed by the terminal (switch tabs, etc.) and never reach the app. The macOS convention is to use Opt (Alt) as a fallback: Opt+Up/Down should jump to document start/end. Currently, Alt+Up/Down are not handled at all — the Alt block in `key_dispatch.go` only covers Alt+Left/Right and Alt+B/F for word navigation.

### Bug 5: Selection state inconsistencies

Selection sometimes gets lost or behaves unexpectedly during navigation.

## Key Discovery

**Cmd+letter keys (A, C, V, Z) work correctly** — the `ModSuper` detection is functional for letter keys. **Cmd+Arrow keys are broken** — they bypass the Super block and fall through to other handlers. This means Ghostty encodes Cmd+Arrow differently than Cmd+letter, possibly as escape sequences that don't carry the `ModSuper` modifier.

## Why This Approach

**Debug-first**: Add a key debug mode to log exactly what Ghostty sends for each Cmd+Arrow combo. The dispatch code *looks* correct for Kitty protocol keys (line 98 of `key_dispatch.go`), so the bug is likely in how the terminal encodes these key events, not in the logic itself.

Once we know what Ghostty actually sends, we fix the dispatch to match reality rather than guessing.

## Key Decisions

1. **Debug-first approach** — Log raw key events before changing dispatch logic
2. **Ghostty is the primary target** — User's daily terminal, supports Kitty protocol
3. **Standard macOS conventions** — Cmd+Arrow = line/doc boundaries, Shift extends selection
4. **Catwalk tests required** — Every fix must have a data-driven test proving the bug exists and the fix works (per CLAUDE.md)

## Relevant Code

| File | What it does |
|------|-------------|
| `cmd/calcmark/tui/editor/key_dispatch.go:15-89` | Global shortcuts (Ctrl+E = Export at line 47) |
| `cmd/calcmark/tui/editor/key_dispatch.go:98-142` | Super (Cmd) block for arrow keys and letter keys |
| `cmd/calcmark/tui/editor/key_dispatch.go:149-167` | Ctrl+key clipboard shortcuts |
| `cmd/calcmark/tui/editor/key_dispatch.go:172-210` | Shift+navigation for selection extension |
| `cmd/calcmark/tui/editor/navigation.go` | Movement handlers (Home/End/CtrlHome/CtrlEnd) |
| `cmd/calcmark/tui/editor/selection.go` | Selection state management |

## Resolved Questions

1. **Debug mode** → Runtime flag (`cm --debug-keys`) that logs raw key events to stderr
2. **Terminal scope** → Focus on Ghostty but verify across Terminal.app, iTerm2
3. **Terminal.app behavior** → Cmd keys are consumed by Terminal.app itself (never reach app). Opt+Left/Right work for word nav. Opt+Up/Down should be the doc-boundary fallback but are unhandled.

## Open Questions

1. Does Ghostty send `ModSuper` for Cmd+Arrow, or something else (e.g., just `ModCtrl` with an arrow code)? → Answer via `--debug-keys` flag
