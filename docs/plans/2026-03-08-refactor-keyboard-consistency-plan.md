---
title: "refactor: Keyboard shortcut consistency audit and cleanup"
type: refactor
status: active
date: 2026-03-08
---

# refactor: Keyboard shortcut consistency audit and cleanup

## Overview

The TUI editor's keyboard shortcuts contain vim-isms (now removed: Ctrl+D/U) and other inconsistencies with native terminal editor conventions. This plan covers a complete audit and alignment of all keyboard bindings across multiple platforms (macOS, Linux) to ensure the non-modal editor feels native.

## Problem Statement

CalcMark is explicitly **non-modal** — the user is always editing, and Ctrl+key combos are accelerator keys for app commands, not mode switches. However, several bindings conflict with platform conventions or each other:

| Key | Current action | Platform standard | Issue |
|-----|---------------|-------------------|-------|
| Ctrl+E | Export | End of line (emacs/readline), Cmd+Right in legacy terminals | **Collides with Cmd+Right in terminals that map Cmd→Ctrl** |
| Ctrl+F | Insert Frontmatter | Find (universal: VS Code, nano, Sublime, etc.) | **Hijacks the most common search shortcut** |
| Ctrl+K | Delete Line | Kill to end of line (emacs), Delete line (nano) | Minor — nano uses Ctrl+K for cut-line too |
| Ctrl+P | Toggle Preview | Print (GUI), Up (emacs/readline) | Acceptable — app-specific |
| Ctrl+H | Help/Command Menu | Backspace (some terminals) | Minor — F1 is the primary trigger |

## Proposed Solution

### Phase 1: Diagnose Cmd+Arrow in Ghostty

**Prerequisite**: Use `cm --debug-keys` (now implemented) to capture exactly what Ghostty sends for Cmd+Right, Cmd+Left, Cmd+Up, Cmd+Down.

**Expected findings**:
- If Ghostty sends `ModSuper + KeyRight` → dispatch already works; bug is elsewhere
- If Ghostty sends `ModCtrl + 'e'` (Cmd mapped to Ctrl) → Ctrl+E handler fires before Super block
- If Ghostty sends something else entirely → new dispatch case needed

#### Actions based on findings

- [ ] Run `cm --debug-keys` in Ghostty, press each Cmd+Arrow combo
- [ ] Run `cm --debug-keys` in Terminal.app for comparison
- [ ] Run `cm --debug-keys` in iTerm2 for comparison
- [ ] Document raw key events for each combo in each terminal
- [ ] Fix dispatch based on actual Ghostty encoding

### Phase 2: Resolve Ctrl+E collision

The Ctrl+E = Export shortcut is the most likely cause of Bug #1 (Cmd+Right opens Export). In legacy terminals, Cmd+Right sends Ctrl+Right which may be misinterpreted.

**Options**:
1. **Keep Ctrl+E for Export** but add a guard: if the key arrives with an arrow code or from a navigation context, don't trigger Export
2. **Move Export to a different shortcut** (e.g., Ctrl+Shift+E, or make it command-menu-only)
3. **Remap Ctrl+E to end-of-line** (emacs convention) and make Export command-menu-only

**Recommendation**: Option 3. Export is used infrequently and doesn't warrant a top-level Ctrl shortcut. Making it command-menu-only (Ctrl+H → navigate → Enter) is sufficient. This frees Ctrl+E for emacs end-of-line, which is the expected behavior in terminals.

- [ ] Decide on Ctrl+E replacement strategy
- [ ] Remove Ctrl+E global handler
- [ ] Add Ctrl+E → end-of-line in default key dispatch (or leave it to terminal convention)
- [ ] Update help overlay and command menu
- [ ] Write catwalk test verifying Ctrl+E no longer opens Export

### Phase 3: Resolve Ctrl+F collision

Ctrl+F = Insert Frontmatter conflicts with the universal Find shortcut. CalcMark doesn't have Find yet, but reserving Ctrl+F for it follows the principle of least surprise.

**Options**:
1. **Move Insert Frontmatter to command-menu-only** — it's a one-time action, not frequent
2. **Remap to Ctrl+Shift+F** — less discoverable but keeps a dedicated shortcut
3. **Keep Ctrl+F** and accept the divergence

**Recommendation**: Option 1. Frontmatter insertion is typically done once per document. Command menu discovery is sufficient.

- [ ] Remove Ctrl+F global handler for Insert Frontmatter
- [ ] Keep Insert Frontmatter available in command menu
- [ ] Reserve Ctrl+F for future Find feature
- [ ] Update help overlay
- [ ] Write catwalk test

### Phase 4: Audit remaining shortcuts

Review each remaining shortcut for platform consistency:

- [ ] Ctrl+K (Delete Line) — keep; matches nano convention
- [ ] Ctrl+P (Toggle Preview) — keep; app-specific, no conflict
- [ ] Ctrl+H (Help/Command Menu) — keep; F1 is primary, Ctrl+H is secondary
- [ ] Ctrl+C (Copy or Quit) — keep; standard behavior
- [ ] Document all shortcuts in a reference table in the help overlay

### Phase 5: Cross-terminal verification

- [ ] Test all shortcuts in Ghostty (Kitty protocol)
- [ ] Test all shortcuts in Terminal.app (legacy)
- [ ] Test all shortcuts in iTerm2 (hybrid)
- [ ] Document any terminal-specific behavior

## Acceptance Criteria

- [ ] `--debug-keys` output captured for Ghostty, Terminal.app, iTerm2
- [ ] Cmd+Right no longer opens Export in Ghostty
- [ ] Ctrl+E freed from Export (or collision resolved)
- [ ] Ctrl+F freed from Insert Frontmatter (reserved for Find)
- [ ] All shortcuts documented in help overlay
- [ ] `task test` passes
- [ ] `task quality` passes
- [ ] Tested across Ghostty, Terminal.app, iTerm2

## Technical Considerations

- The global Ctrl handler (key_dispatch.go line 31) runs BEFORE mode-specific handlers. Any removal of a Ctrl shortcut must verify no fallthrough behavior changes.
- Legacy terminals (Terminal.app) map Cmd→Ctrl, so Cmd+E = Ctrl+E in those terminals. Removing Ctrl+E from Export prevents this collision.
- The `key.Matches(msg, m.keys.Help)` handler (line 56) uses the keybinding system differently from the switch/case blocks. Consistency between these dispatch methods should be verified.

## Dependencies

- Depends on `--debug-keys` flag (completed in fix/keyboard-nav-selection-bugs branch)
- Issue #38 covers the initial Cmd+Arrow bugs; this plan covers the broader consistency refactor

## References

- Brainstorm: `docs/brainstorms/2026-03-08-fix-keyboard-nav-selection-bugs-brainstorm.md`
- Fix plan: `docs/plans/2026-03-08-fix-keyboard-nav-selection-bugs-plan.md`
- Key dispatch: `cmd/calcmark/tui/editor/key_dispatch.go`
- Help overlay: `cmd/calcmark/tui/editor/help_overlay.go`
- Command menu: `cmd/calcmark/tui/editor/command_menu.go`
- Prior art: `docs/solutions/ui-bugs/frontmatter-editing-keyboard-dispatch-fixes.md`
