---
title: "Bracketed paste routed to document body instead of overlay input fields"
category: ui-bugs
tags: [tui, paste, bracketed-paste, overlay, mode-dispatch, open-from-gist, share-to-gist]
module: cmd/calcmark/tui/editor
symptom: "Pasting a URL via Cmd+V (bracketed paste) into the Open From Gist or Share To Gist overlay inserted text into the document body instead of the overlay's input field"
root_cause: "tea.PasteMsg in Update() was routed unconditionally to handleBracketedPaste(), bypassing mode-aware dispatch; overlay states (StateOpenFrom, StateShareTo) never received paste events"
date_solved: 2026-03-06
---

# Bracketed Paste Routed to Document Body Instead of Overlay Input Fields

## Problem

When an overlay dialog (such as "Open From Gist" or "Share To Gist") was active in the TUI editor, pressing Cmd+V to paste clipboard content would insert the text into the document body behind the overlay instead of into the overlay's input field. This meant users could not paste a gist URL into the "Open From" dialog or paste a description into the "Share To" dialog — the paste silently corrupted the document instead.

## Investigation

The bug was traced through the message dispatch chain in `Update()` within `cmd/calcmark/tui/editor/model.go`. Key presses are dispatched through `handleKey()` which has mode-aware routing (checking `m.mode` to decide where input goes). However, `tea.PasteMsg` — a separate Bubble Tea message type sent when the terminal intercepts bracketed paste (Cmd+V) — was handled in its own `case` branch, completely bypassing the mode-aware key dispatch logic. Paste events had no knowledge of which overlay was active.

## Root Cause

In Bubble Tea, `tea.KeyPressMsg` and `tea.PasteMsg` are distinct message types that travel through separate code paths. The `handleKey` method naturally dispatches based on `m.mode`, so key-by-key typing into overlay input fields works correctly. But `tea.PasteMsg` had its own `case` branch in `Update()` that unconditionally called `handleBracketedPaste()`, which inserts text into the document body.

**Before (broken):**
```go
case tea.PasteMsg:
    return m.handleBracketedPaste(msg.Content)
```

## Solution

Mode-aware dispatch was added before the default paste handler. The fix checks `m.mode` and routes paste content to the appropriate overlay input field when an overlay is active.

**After (fixed) — `model.go` lines 575-589:**
```go
case tea.PasteMsg:
    // Route paste to overlay input fields when active.
    // Without this, pasted text goes to the document body behind the overlay.
    switch m.mode {
    case StateOpenFrom:
        m.openFromInput += msg.Content
        return m, nil
    case StateShareTo:
        if m.shareField == 1 { // description field active
            m.shareDescription += msg.Content
            return m, nil
        }
    }
    // Default: bracketed paste into document editor.
    return m.handleBracketedPaste(msg.Content)
```

When `m.mode == StateOpenFrom`, pasted text appends to `m.openFromInput`. When `m.mode == StateShareTo` and the description field is active (`m.shareField == 1`), pasted text appends to `m.shareDescription`. All other states fall through to the original `handleBracketedPaste()` behavior.

## Verification

Two catwalk data-driven tests were added:

**`testdata/open_from_paste`** — Opens the "Open From Gist" overlay via the command menu, pastes a gist URL using `paste "https://gist.github.com/user/abc123"`, then cancels with Esc and verifies the document body was not modified (`editBuf=""`, `totalSource` unchanged).

**`testdata/share_to_paste`** — Opens the "Share To Gist" overlay, tabs to the description field, pastes text using `paste "My budget calculations"`, then cancels and verifies the document was not modified.

Both tests use the `debug` observer to assert that `sourcePreviewMatch=true` and `editBuf=""` after the paste, confirming that pasted content was routed to the overlay field and never reached the document body.

## Prevention

### The Core Pattern

This is a class of bug where **two dispatch sites must stay in sync**: the `tea.KeyPressMsg` routing (inside `handleKey`) and the `tea.PasteMsg` routing (inside `Update`). Any new overlay state with a text input field must be added to both.

### Checklist for New Overlay States

Every new `InputState` that accepts text input **must**:

1. **`handleKey`** — route `tea.KeyPressMsg` to the overlay's input field based on `m.mode`.
2. **`Update` PasteMsg case** — add a `case` for the new state under `tea.PasteMsg` that appends `msg.Content` to the same field.
3. **Catwalk test** — add a `paste` test that proves pasted text lands in the overlay field, not the document.

### Testing Checklist

For every overlay with a text input field, write a catwalk test that:

1. **Paste arrives in the overlay field** — navigate to the overlay, paste text, verify `editBuf=""` (document unchanged).
2. **Cancel preserves document integrity** — after pasting, press Esc and verify `totalSource` and `editBuf` are unchanged.
3. **Multi-field overlays** — test paste in each field separately (e.g., ShareTo has visibility selector + description field; only the description field accepts paste).

**Catwalk paste template:**
```
run observe=debug
paste "test content"
----
-- debug:
mode=StateXxx ... editBuf=""
```

The key assertion is `editBuf=""` — confirming pasted text did not leak into the document's edit buffer.

## Related

- [TUI mode transitions, formatter alignment, and bracketed paste fixes](../ui-bugs/tui-mode-transitions-formatter-indexing-and-bracketed-paste-fixes.md) — earlier work that added the initial `tea.PasteMsg` handler and centralized mode transitions in `mode_transitions.go`
- [Overlay compositing ANSI state bleed-through](../ui-bugs/overlay-compositing-ansi-state-bleed-through.md) — overlay rendering issues
