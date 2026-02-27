---
title: "feat: Add remote store plugin system with GitHub Gists backend"
type: feat
status: active
date: 2026-02-27
brainstorm: docs/brainstorms/2026-02-27-remote-store-plugins-brainstorm.md
---

# feat: Add remote store plugin system with GitHub Gists backend

## Overview

Add "Share To Gist" and "Open From Gist" commands to the TUI editor,
enabling users to share CalcMark documents as GitHub Gists and open gists
as new local documents. The implementation uses subprocess calls to the
`gh` CLI, delegating auth and network operations entirely to the external
tool.

## Problem Statement / Motivation

CalcMark documents contain verifiable calculations that users want to share
with others. Currently, sharing requires manually copying file contents to
external services. A built-in "Share To" / "Open From" flow reduces
friction and keeps users in the editor.

## Proposed Solution

A subprocess-based store layer that shells out to `gh` for all GitHub Gist
operations. Two new TUI overlay states provide the user interface. No sync
state, no credential handling, no new network code in CalcMark itself.

### Architecture: Two-Phase Subprocess Pattern

The critical architectural decision is how to handle subprocess execution
when `gh` might need interactive auth:

- **`tea.Exec`** suspends the TUI and gives the subprocess terminal
  control. Good for auth prompts, but stdout goes to the screen (not
  captured by CalcMark).
- **`exec.Command` with pipes** captures stdout/stderr but cannot show
  interactive auth prompts.

**Solution: two-phase approach.**

1. **Auth check** — run `gh auth status` with piped output. If exit code
   is non-zero, use `tea.Exec` to run `gh auth login` interactively, then
   retry the auth check.
2. **Operation** — run the actual `gh gist create` or `gh gist view` with
   piped stdout/stderr. Since auth is already confirmed, no interactive
   prompts are needed.

This gives us both interactive auth (when needed) and captured output
(always).

## Technical Considerations

### Subprocess Security

- **Shell injection prevention**: Always use `exec.Command("gh", "gist",
  "create", "-d", description, ...)` with separate argument slices. Never
  shell string concatenation.
- **Content validation**: Fetched gist content must pass
  `filecheck.ValidateContent()` (binary check) and the 1MB size limit from
  SECURITY.md before parsing.
- **No credentials in CalcMark**: All auth is handled by `gh auth`. CalcMark
  never reads, stores, or transmits tokens.

### WASM Build Guards

Subprocess operations are unavailable in WASM. Use build tags:
- `//go:build !js && !wasm` on files that import `os/exec`
- `//go:build js || wasm` on stub files that return "not available in
  browser" errors

The command menu entries for Share/Open should be conditionally excluded
from WASM builds.

### Multi-File Gists

`gh gist view <id> -r` without `--filename` concatenates all files. To
handle this:
1. Run `gh gist view <id> --json files` to inspect the file list
2. If exactly one file, fetch it with `-r`
3. If multiple files, pick the first `.cm` file. If none have `.cm`
   extension, pick the first file
4. Use `--filename <name>` to fetch the specific file

### Overlay Pattern (Not Huh)

The brainstorm proposed Charm's Huh library for forms, but SpecFlow
analysis shows the forms are simple enough for the existing overlay
pattern. The Share To form needs a binary select + text input. The Open
From form needs a single text input. Both are simpler than the existing
Export overlay. **Reuse existing overlay patterns** — no new dependency
needed.

### No Dedicated Keybindings

V1 commands are accessible via the command menu only (Ctrl+H / F1). No
dedicated Ctrl+key shortcuts. This avoids consuming scarce key slots for an
infrequently-used feature.

## Acceptance Criteria

### Functional Requirements

- [x] "Share To Gist" command in the command menu creates a new GitHub Gist
  from the current document and displays the gist URL in the status bar
- [x] "Open From Gist" command in the command menu loads a gist's content
  as a new untitled local document
- [x] User can choose public or secret visibility when sharing
- [x] User can optionally provide a description when sharing
- [x] Gist URL is copied to the system clipboard after sharing
- [x] Open From triggers unsaved-changes guard if document is modified
- [x] Clear error message if `gh` CLI is not installed (with install URL)
- [x] Clear error message on auth failure, network error, or invalid gist
- [x] Esc cancels the Share/Open overlay at any point
- [x] Multi-file gists are handled (pick first `.cm` file or first file)

### Non-Functional Requirements

- [x] No shell injection possible via description or filename fields
- [x] Fetched content validated: binary check + 1MB size limit
- [x] WASM builds compile without `os/exec` — commands excluded from menu
- [x] No new dependencies added (`go.mod` unchanged except build tags)
- [x] `task quality` passes (lint, modernize, staticcheck)
- [x] `task test` passes with no regressions

### Testing Requirements

- [ ] Catwalk tests: trigger Share To via command menu, verify overlay, Esc
  cancels
- [ ] Catwalk tests: trigger Open From via command menu, verify overlay,
  Esc cancels
- [ ] Catwalk tests: trigger Open From with unsaved changes, verify save
  prompt appears
- [x] Unit tests: `gh` CLI availability check
- [x] Unit tests: gist URL parsing from `gh` stdout
- [x] Unit tests: multi-file gist handling (JSON parsing, file selection)
- [x] Unit tests: content validation on fetched gist data
- [x] Unit tests: filename resolution (named file, untitled document)
- [x] WASM build: verify commands absent from menu (`go build` with
  `GOOS=js GOARCH=wasm`)

## Implementation Phases

### Phase 1: Store Abstraction + Gist Implementation

Create the subprocess interaction layer independent of the TUI.

**New files:**

```
cmd/calcmark/tui/editor/store/
  store.go           -- Store interface + errors (build-tag guarded)
  store_stub.go      -- WASM stubs (//go:build js || wasm)
  gist.go            -- GitHub Gist implementation
  gist_test.go       -- Unit tests with mock exec
```

**`store.go`** — core interface:

```go
//go:build !js && !wasm

// ShareResult holds the outcome of a Share operation.
type ShareResult struct {
    URL string
}

// OpenResult holds the outcome of an Open operation.
type OpenResult struct {
    Content  string
    Filename string
}

// Store defines the operations a remote store must support.
type Store interface {
    // Name returns the display name (e.g., "GitHub Gist").
    Name() string

    // CheckAvailable verifies the backing CLI tool is installed.
    CheckAvailable() error

    // CheckAuth verifies authentication status.
    CheckAuth() error

    // Share creates a new remote resource from content.
    Share(content, filename, description string, public bool) (ShareResult, error)

    // Open fetches content from a remote resource by identifier.
    Open(identifier string) (OpenResult, error)
}
```

**`gist.go`** — GitHub Gist implementation:

- `CheckAvailable()` — `exec.LookPath("gh")`
- `CheckAuth()` — `exec.Command("gh", "auth", "status")` with piped
  stderr, check exit code
- `Share()` — write content to temp file, run
  `gh gist create [-d desc] [--public] -f filename tempfile`, parse URL
  from stdout, clean up temp file
- `Open()` — run `gh gist view <id> --json files` to get file list,
  select appropriate file, run `gh gist view <id> -r --filename <name>`,
  validate content (binary check + size limit), return content + filename

**Testing**: Table-driven tests using a mock command executor injected via
an `Executor` interface:

```go
type Executor interface {
    Run(name string, args ...string) (stdout, stderr []byte, err error)
    LookPath(name string) (string, error)
}
```

Production uses `realExecutor` wrapping `os/exec`. Tests use
`mockExecutor` with canned responses.

### Phase 2: TUI Integration

Wire the store layer into the editor's overlay system.

**Modified files:**

| File | Changes |
|------|---------|
| `model.go` | Add `StateShareTo`, `StateOpenFrom` to `InputState`. Add store-related fields to `Model`. |
| `mode_transitions.go` | Add `enterShareTo()`, `enterOpenFrom()`, update `exitOverlay()`. |
| `command_menu.go` | Add "Share To Gist" and "Open From Gist" to `EditorCommands`. Add cases in `executeCommandByName()`. |
| `key_dispatch.go` | Add `StateShareTo` and `StateOpenFrom` cases in the mode switch. |
| `save_prompt_handler.go` | Add `PendingOpenFromRemote` to `PendingAction` enum and handle in `completePendingSaveAction()`. |

**New files:**

| File | Purpose |
|------|---------|
| `share_overlay.go` | Renders the Share To form (visibility select + description input). Handles key dispatch within the overlay. |
| `share_overlay_test.go` | Catwalk + unit tests for Share To overlay rendering and key handling. |
| `open_from_overlay.go` | Renders the Open From form (URL/ID text input). Handles key dispatch within the overlay. |
| `open_from_overlay_test.go` | Catwalk + unit tests for Open From overlay rendering and key handling. |
| `store_commands.go` | Orchestrates the two-phase subprocess flow: auth check → `tea.Exec` for auth if needed → piped command for operation. Sends result messages back to the model. |
| `store_commands_stub.go` | WASM stub (//go:build js \|\| wasm) — returns "not available" errors. |

**Share To overlay state:**

```go
// Fields added to Model
shareVisibility int    // 0 = public, 1 = secret
shareDescription string
shareField      int    // 0 = visibility select, 1 = description input
```

**Key handling in Share To overlay:**
- Up/Down or Tab: toggle between visibility and description fields
- Left/Right (on visibility field): toggle public/secret
- Character input (on description field): append to description
- Enter: submit form → launch subprocess
- Esc: cancel, return to StateDefault

**Open From overlay state:**

```go
// Fields added to Model
openFromInput string
```

**Key handling in Open From overlay:**
- Character input: append to `openFromInput`
- Backspace: delete last character
- Enter: submit → launch subprocess
- Esc: cancel, return to StateDefault

**Subprocess orchestration in `store_commands.go`:**

```go
func (m *Model) executeShareToGist() tea.Cmd {
    store := gist.New(realExecutor{})

    // Check availability
    if err := store.CheckAvailable(); err != nil {
        return m.setStatusError("gh CLI not found. Install: https://cli.github.com")
    }

    // Check auth — if fails, return tea.Exec for interactive login
    if err := store.CheckAuth(); err != nil {
        return tea.Exec(exec.Command("gh", "auth", "login"), func(err error) tea.Msg {
            if err != nil {
                return shareResultMsg{err: fmt.Errorf("auth failed: %w", err)}
            }
            return retryShareMsg{} // triggers re-attempt with piped command
        })
    }

    // Auth OK — run in background, return result via message
    return func() tea.Msg {
        filename := m.resolveFilename()
        result, err := store.Share(
            m.getDocumentContent(),
            filename,
            m.shareDescription,
            m.shareVisibility == 0,
        )
        return shareResultMsg{result: result, err: err}
    }
}
```

**Result handling in `Update()`:**

```go
case shareResultMsg:
    if msg.err != nil {
        m.setStatus(msg.err.Error(), true)
    } else {
        // Copy URL to clipboard
        clipboard.Write(msg.result.URL)
        m.setStatus("Shared: "+msg.result.URL+" (copied)", false)
    }
    m.exitOverlay()

case openResultMsg:
    if msg.err != nil {
        m.setStatus(msg.err.Error(), true)
        m.exitOverlay()
    } else {
        // Load content as new document (same path as openFile but from string)
        m.loadDocumentFromString(msg.result.Content, msg.result.Filename)
    }
```

**Filename resolution:**

```go
func (m *Model) resolveFilename() string {
    if m.filepath == "" {
        return "untitled.cm"
    }
    return filepath.Base(m.filepath)
}
```

### Phase 3: Testing + Polish

- Write catwalk tests for all overlay key sequences
- Verify WASM build (`GOOS=js GOARCH=wasm go build ./...`)
- Update help overlay to list new commands
- Update SECURITY.md to mention subprocess plugin architecture
- Run `task quality` and `task test`
- Run `task security` for fuzzing

## Dependencies & Risks

| Risk | Mitigation |
|------|------------|
| `gh` CLI not installed on user's system | Clear error with install URL. Feature degrades gracefully (menu items still show but fail with helpful message). |
| `gh` CLI output format changes between versions | Parse output conservatively. Use `--json` where available for structured output. |
| `tea.Exec` state restoration bugs | Thorough catwalk tests for TUI state after subprocess exit. Invalidate aligned pane cache on resume. |
| Clipboard not available (SSH, headless) | Graceful fallback — show URL in status bar without "(copied)" suffix. |
| Multi-file gist with no `.cm` files | Fall back to first file. CalcMark parses any text (non-calc lines become TextBlocks). |

## Success Metrics

- User can share a CalcMark document as a gist in under 5 seconds
  (excluding first-time auth)
- User can open a gist by pasting a URL and editing within 3 seconds
- Zero security vulnerabilities (no shell injection, no credential
  exposure)
- No regressions in existing TUI functionality

## References & Research

### Internal References

- Brainstorm: `docs/brainstorms/2026-02-27-remote-store-plugins-brainstorm.md`
- TUI editor model: `cmd/calcmark/tui/editor/model.go` (InputState enum,
  lines 142-153)
- Mode transitions: `cmd/calcmark/tui/editor/mode_transitions.go`
- Command menu: `cmd/calcmark/tui/editor/command_menu.go`
- Key dispatch: `cmd/calcmark/tui/editor/key_dispatch.go`
- File operations: `cmd/calcmark/tui/editor/file_operations.go`
- Overlay style: `cmd/calcmark/tui/editor/overlay_style.go`
- Export overlay (pattern to follow):
  `cmd/calcmark/tui/editor/export_overlay.go`
- File validation: `cmd/calcmark/filecheck/`
- Save prompt handler: `cmd/calcmark/tui/editor/save_prompt_handler.go`
- Clipboard: `cmd/calcmark/tui/editor/clipboard.go`
- Security policy: `SECURITY.md`

### Institutional Learnings

- Mode transitions must be centralized in `mode_transitions.go` with
  consistent field resets
  (`docs/solutions/ui-bugs/tui-mode-transitions-formatter-indexing-and-bracketed-paste-fixes.md`)
- File open must reset all mutable editor state
  (`docs/solutions/ui-bugs/ctrl-o-stale-state-and-unsaved-changes-detection.md`)
- Every user-facing TUI feature needs catwalk tests — unit tests miss
  timing-dependent bugs
  (`docs/solutions/ui-bugs/bubble-tea-v2-migration-selection-undo-clipboard-fixes.md`)
- Overlay rendering must use raw text for column arithmetic, then style
  (`docs/solutions/code-organization/split-view-go-into-cohesive-modules.md`)

### External References

- `gh gist create` docs: https://cli.github.com/manual/gh_gist_create
- `gh gist view` docs: https://cli.github.com/manual/gh_gist_view
- `tea.Exec` in bubbletea: https://pkg.go.dev/charm.land/bubbletea/v2#Exec
