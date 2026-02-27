---
title: "fix: Status bar message truncation and Ctrl-O file type validation"
type: fix
status: active
date: 2026-02-27
deepened: 2026-02-27
---

# fix: Status bar message truncation and Ctrl-O file type validation

## Enhancement Summary

**Deepened on:** 2026-02-27
**Sections enhanced:** 6
**Research agents used:** pattern-recognition-specialist, performance-oracle, security-sentinel, code-simplicity-reviewer, architecture-strategist, learnings-researcher, context7 (lipgloss, bubbletea)

### Key Improvements from Research

1. **Extract extension validation to `filecheck` package** — Prevents DRY violation between CLI (`security.go:65`) and TUI (`file_operations.go`). Both callers share a single source of truth for accepted extensions.
2. **Simplified error messages** — Single message format using existing `"Open failed: ..."` prefix pattern for consistency with all other `openFile` error paths.
3. **Drop unnecessary `maxMsgWidth` floor** — `TruncateWithEllipsis` already handles degenerate inputs gracefully. The floor is YAGNI.
4. **Fix misleading comment in `TruncateWithEllipsis`** — Says "binary search" but is actually a linear scan. Opportunistic fix while touching the file.
5. **Reduced test matrix** — Testing multiple wrong extensions (`.txt`, `.json`, `.html`) is redundant since they all hit the same branch. 3 extension tests instead of 7.

### New Considerations Discovered

- **File picker error visibility**: When the picker stays open after failure, the error in `m.statusMsg` is behind the overlay and not immediately visible. The `statusMsg` clears on the next keypress in the picker (falls through to `default` in `key_dispatch.go:21`). This matches save flow behavior but should be noted.
- **`TruncateWithEllipsis` iterates by byte, not rune** — Can split multi-byte UTF-8 sequences on intermediate iterations. Not a correctness issue (lipgloss handles it) but wastes iterations. Consider fixing opportunistically.
- **`RenderMinimalStatusBar` uses `len()` not `lipgloss.Width()`** — Latent bug for non-ASCII filenames. Out of scope but tracked.

## Overview

Two related UX issues in the TUI editor when opening non-CalcMark files via Ctrl-O:

1. **Long error messages overflow the status bar** — `RenderStatusBar` renders `StatusMsg` at full length with no width clamping. Messages like `"Parse error: frontmatter: unknown frontmatter key 'title'; user variables must go under 'globals:'"` (85+ chars) overflow the terminal width, breaking the `StatusBarHeight = 2` contract and causing bubbletea rendering artifacts.

2. **No file extension validation on Ctrl-O open** — The CLI's `validateFileConstraints` rejects non-`.cm`/`.calcmark` files, but the TUI's `openFile()` skips this check entirely. Opening a `.md` file with YAML frontmatter produces a cryptic parse error about "unknown frontmatter key" instead of a clear "not a CalcMark file" message.

## Problem Statement

When a user presses Ctrl-O and selects `site/content/docs/examples/dates-and-durations.md`, the editor:
1. Passes `filecheck.ValidateContent` (it's valid UTF-8 text)
2. Fails at `document.NewDocument` because the Hugo YAML frontmatter (`title`, `summary`, `weight`) uses keys not in CalcMark's reserved set
3. Shows `Parse error: frontmatter: unknown frontmatter key 'title'; user variables must go under 'globals:'` in the status bar
4. The message is too long to read — it overflows the status bar width with no truncation or ellipsis

## Proposed Solution

### Change 1: Status bar message truncation

Add `TruncateWithEllipsis` to `RenderStatusBar` before styling the message. The utility already exists in the same package (`components/errors.go:135`) and is used by `RenderContextFooter`.

**File**: `cmd/calcmark/tui/components/statusbar.go`

Truncate `state.StatusMsg` to fit within the available content width before applying the style. The available width is `width - 2` (1-char left pad from `buildLine` + 1 right margin).

```go
// Status message takes the full line
if state.StatusMsg != "" {
    var msgStyle lipgloss.Style
    if state.StatusIsErr {
        msgStyle = style.StatusErr
    } else {
        msgStyle = style.StatusOK
    }
    // Truncate before styling to prevent overflow and preserve StatusBarHeight contract.
    // Must truncate the raw text, not the ANSI-styled string.
    maxMsgWidth := width - 2 // 1 left pad + 1 right margin
    truncatedMsg := TruncateWithEllipsis(state.StatusMsg, maxMsgWidth)
    msgStr := msgStyle.Render(truncatedMsg)
    line1 := buildLine(msgStr, lipgloss.Width(truncatedMsg))
    line2 := StyledPadding(width, barBg)
    return line1 + "\n" + line2
}
```

#### Research Insights

**Why truncate in the renderer, not the business logic:**
- `RenderStatusBar` is a pure function that owns fitting content to `width`. This is a rendering concern.
- `GetStatusBarState()` in `view_state.go` does not know the terminal width — truncating there would require threading width through the state projection layer.
- Business logic (`openFile`) should produce semantically complete messages. If the terminal later resizes, a pre-truncated message could not be re-expanded.
- This follows the existing pattern: `RenderContextFooter` already truncates via `TruncateWithEllipsis` at `contextfooter.go:121,151,198`.

**Performance (from performance-oracle):**
- `TruncateWithEllipsis` short-circuits via `lipgloss.Width(s) <= maxWidth` — zero allocations when no truncation needed (the common case).
- For a 90-char message needing 10 chars trimmed: ~10 iterations of the linear scan, each doing `lipgloss.Width` on a ~80-char string. Well under a microsecond.
- `RenderStatusBar` is called every frame, but `TruncateWithEllipsis` is already called 3+ times per frame by `RenderContextFooter`. One more call is negligible.

**Opportunistic fix — `TruncateWithEllipsis` comment:**
The comment at `errors.go:141` says "Binary search" but the algorithm is a linear scan from the end. Fix the comment while in the file:
```go
// Linear scan from end to find truncation point
// (accounting for variable-width characters)
```

### Change 2: Extract extension validation to `filecheck` package + use in `openFile`

**Rationale (from pattern-recognition and architecture reviews):** The CLI's `validateFileConstraints` at `security.go:64-67` and the proposed TUI check both encode the same extension list (`.cm`, `.calcmark`). Extracting to `filecheck` — which already serves as the shared validation layer between CLI and TUI (`filecheck.ValidateContent` is called by both) — eliminates this DRY violation.

**File**: `cmd/calcmark/filecheck/filecheck.go` — add shared predicate:

```go
// IsCalcMarkExtension reports whether the file at path has a recognized
// CalcMark extension (.cm or .calcmark, case-insensitive).
func IsCalcMarkExtension(path string) bool {
    ext := strings.ToLower(filepath.Ext(path))
    return ext == ".cm" || ext == ".calcmark"
}
```

**File**: `cmd/calcmark/tui/editor/file_operations.go` — use in `openFile`:

```go
func (m *Model) openFile(filename string) {
    // Get absolute path
    absPath, err := filepath.Abs(filename)
    if err != nil {
        m.statusMsg = fmt.Sprintf("Invalid path: %v", err)
        m.statusIsErr = true
        return
    }

    // Validate file extension (case-insensitive, matching CLI behavior)
    if !filecheck.IsCalcMarkExtension(absPath) {
        ext := filepath.Ext(absPath)
        if ext == "" {
            ext = "no extension"
        }
        m.statusMsg = fmt.Sprintf("Open failed: unsupported file type (%s)", ext)
        m.statusIsErr = true
        return
    }

    // ... rest of openFile unchanged ...
}
```

**File**: `cmd/calcmark/cmd/security.go` — refactor to use shared predicate:

```go
func validateFileConstraints(absPath string) error {
    // Security: Check file extension (case-insensitive)
    if !filecheck.IsCalcMarkExtension(absPath) {
        return fmt.Errorf("invalid file extension: expected .cm or .calcmark")
    }
    // ... rest unchanged ...
}
```

#### Research Insights

**Error message format (from pattern-recognition and simplicity reviews):**
- Use the existing `"Open failed: ..."` prefix to match `file_operations.go` conventions (lines 163, 171, 179, 186 all use prefixed messages).
- Single message format — no need to differentiate "no extension" vs "wrong extension" at the code level. The message `"Open failed: unsupported file type (.md)"` or `"Open failed: unsupported file type (no extension)"` covers both naturally.
- Include the actual extension in the message so the user can verify they selected the right file.

**Security (from security-sentinel):**
- `strings.ToLower` is sufficient for ASCII extensions like `.cm` and `.calcmark`. Go's `strings.ToLower` handles Unicode correctly, but file extensions are conventionally ASCII.
- Double extensions (`.cm.bak`) are correctly handled: `filepath.Ext` returns `.bak`, which will fail the check.
- Symlinks are not a concern here — the extension check validates the link target's name, not the link itself, since `filepath.Abs` resolves the path.
- The extension check is defense-in-depth, not a security boundary. `filecheck.ValidateContent` remains the primary content check.

### Change 3: Keep file picker open on open failure

Currently, `file_picker_handler.go` calls `exitOverlay()` unconditionally after `openFile()`. The save flow already keeps the picker open on failure (line 37: `if !m.statusIsErr`). Apply the same pattern to the open flow.

**File**: `cmd/calcmark/tui/editor/file_picker_handler.go`

Both code paths that call `openFile` must be guarded:

**Path 1 — FocusFilename + Enter (line 47-52):**
```go
} else if m.filePickerPurpose == PickerForOpen {
    if m.filePickerFocus == FocusFilename {
        if m.newFileName != "" {
            path := filepath.Join(m.filePicker.CurrentDirectory, m.newFileName)
            m.openFile(path)
            if !m.statusIsErr {
                m.exitOverlay()
            }
        }
        return m, nil
    }
```

**Path 2 — DidSelectFile (line 94-99):**
```go
if didSelect, path := m.filePicker.DidSelectFile(msg); didSelect {
    if m.filePickerPurpose == PickerForOpen {
        m.openFile(path)
        if !m.statusIsErr {
            m.exitOverlay()
        }
        return m, cmd
    }
```

#### Research Insights

**State machine analysis (from architecture review):**
- `exitOverlay()` at `mode_transitions.go:18-22` sets `m.mode = StateDefault` and resets `m.pendingSaveAction` and `m.newFileName`. Skipping it on error keeps the user in `StateFilePicker` with their input intact — they can try another file or press Esc.
- This is architecturally consistent with the save flow at line 37.

**UX concern — error visibility behind overlay:**
- When the picker stays open, the status bar error is rendered behind the overlay and is not immediately visible. The overlay (`renderFilePickerOverlay`) does not display status messages.
- However, pressing Esc to close the picker reveals the error. And the next keypress in the picker clears the status message (falls through to `default` in `key_dispatch.go:21-27`).
- This matches the save flow behavior. If this UX is problematic, a future enhancement could display the error within the overlay frame — but that is a separate concern.

## Acceptance Criteria

### Functional

- [x] Status bar messages longer than terminal width are truncated with `...`
- [x] Opening a `.md` file via Ctrl-O shows `"Open failed: unsupported file type (.md)"`
- [x] Opening a file with no extension shows `"Open failed: unsupported file type (no extension)"`
- [x] `.CM` and `.CALCMARK` (uppercase) are accepted
- [x] `.cm` files with parse errors still show the parse error (not extension error)
- [x] File picker stays open when open fails, allowing user to try another file
- [x] Status bar never exceeds `StatusBarHeight = 2` lines
- [x] `filecheck.IsCalcMarkExtension` is used by both CLI and TUI

### Testing

- [x] Unit tests in `components_test.go`: status bar truncation at 80 and 40 widths with 85-char message
- [x] Unit tests in `components_test.go`: status bar with message exactly at width (no truncation)
- [x] Unit tests in `components_test.go`: status bar with message under width (no truncation)
- [x] Unit tests in `filecheck/filecheck_test.go`: `IsCalcMarkExtension` with `.cm`, `.calcmark`, `.CM`, `.md`, `""` (no ext)
- [x] Unit tests in `file_operations_test.go`: open `.md` file → extension error
- [x] Unit tests in `file_operations_test.go`: open `.CM` (uppercase) → success
- [x] Unit tests in `file_operations_test.go`: open `.calcmark` → success
- [x] Unit tests in `file_operations_test.go`: open `.cm` → success (existing test passes)
- [x] Unit tests in `file_operations_test.go`: open `.cm` with parse errors → shows parse error, not extension error
- [x] Unit test: file picker stays open after open failure
- [ ] Catwalk test: Ctrl-O → select non-.cm file → error shown → picker stays open → Esc → error visible in status bar
- [x] `task test` passes with zero regressions
- [x] `task quality` passes

## Technical Considerations

### Status bar padding math

The `buildLine` helper in `RenderStatusBar` adds 1-char left padding. The `Bar` style has `Padding(0, 1)` but `buildLine` manages its own padding manually (it doesn't call `style.Bar.Render()`). The truncation width should be `width - 2` to account for the 1-char left pad and 1-char right margin.

Per the institutional learning in `docs/solutions/ui-bugs/tui-editor-rendering-divider-status-bar-error-line.md`:
- Always call `style.GetHorizontalPadding()` before computing content width
- Test at exact boundary widths (79, 80, 81) to catch off-by-one errors
- The previous status bar wrapping bug was caused by ignoring style padding — same class of error to avoid

### Extension check does not replace content validation

The extension check is a fast pre-filter. The `filecheck.ValidateContent` (binary/null/UTF-8) check remains after `os.ReadFile` as defense-in-depth. A `.cm` file that's actually binary still gets caught. The validation order in `openFile` is:

```
1. filepath.Abs             — path resolution
2. filecheck.IsCalcMarkExtension — extension pre-filter (NEW)
3. os.ReadFile              — I/O
4. filecheck.ValidateContent — binary/null/UTF-8 check
5. document.NewDocument     — parse
6. evaluator.Evaluate       — eval (non-fatal errors)
```

### `RenderMinimalStatusBar` is out of scope

`RenderMinimalStatusBar` does not render status messages at all and is not called by `View()`. Two latent issues tracked but not addressed:
1. It uses `len(name)` instead of `lipgloss.Width()` for filename truncation — incorrect for non-ASCII filenames.
2. It does not display `StatusMsg` at all — if ever promoted to active use, it needs the same truncation treatment.

### `saveFile` extension validation is out of scope

`saveFile` does not validate extensions either — a user could save as `foo.txt`. This is an existing gap that predates this plan. Out of scope.

## Dependencies & Risks

- **Low risk**: All changes are in the TUI layer; no spec or interpreter changes.
- **Dependency**: `TruncateWithEllipsis` already exists and is tested. No new utilities needed.
- **New shared function**: `filecheck.IsCalcMarkExtension` is a pure function with no dependencies — low risk addition.
- **Risk**: Off-by-one in truncation width calculation. Mitigated by boundary-width tests at 79, 80, 81 columns.
- **Risk**: File picker staying open on error — the error message is behind the overlay and not immediately visible. Mitigated by matching the existing save flow behavior. The user can press Esc to see the error, or navigate to another file.
- **Risk**: Extension list divergence if new extensions are added — mitigated by extracting to `filecheck.IsCalcMarkExtension` (single source of truth).

## References & Research

### Internal References

- Status bar rendering: `cmd/calcmark/tui/components/statusbar.go:95-122`
- `TruncateWithEllipsis`: `cmd/calcmark/tui/components/errors.go:135-150`
- File open: `cmd/calcmark/tui/editor/file_operations.go:159-203`
- CLI extension check: `cmd/calcmark/cmd/security.go:63-68`
- File picker handler: `cmd/calcmark/tui/editor/file_picker_handler.go:45-54, 94-99`
- File content validation: `cmd/calcmark/filecheck/filecheck.go:52-76`
- Status bar width bug learning: `docs/solutions/ui-bugs/tui-editor-rendering-divider-status-bar-error-line.md`
- Ctrl-O stale state learning: `docs/solutions/ui-bugs/ctrl-o-stale-state-and-unsaved-changes-detection.md`
- Background bleed-through learning: `docs/solutions/ui-bugs/lipgloss-background-bleed-through.md`
- Mode transition learning: `docs/solutions/ui-bugs/tui-mode-transitions-formatter-indexing-and-bracketed-paste-fixes.md`

### Framework Documentation

- lipgloss `Width()` measures visual width by stripping ANSI and counting runes — safe for styled and unstyled text
- lipgloss `Style.Width(n)` constrains rendered output to n columns with wrapping — an alternative to manual truncation, but wrapping would violate `StatusBarHeight = 2`
- bubbletea Elm architecture: state changes in Update, presentation in View — truncation correctly placed in View layer
