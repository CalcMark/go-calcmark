---
title: "Remote store plugin system with GitHub Gist backend for TUI editor"
date: 2026-02-27
category: integration-issues
tags:
  - tui
  - remote-store
  - github-gist
  - subprocess
  - wasm-build-tags
  - bubbletea
  - overlay-system
  - feature
components:
  - cmd/calcmark/tui/editor/store
  - cmd/calcmark/tui/editor
  - cmd/calcmark/filecheck
severity: medium
symptom: "TUI editor lacked the ability to share documents to or open documents from remote storage backends such as GitHub Gists"
root_cause: "No remote store abstraction existed; required a two-phase subprocess pattern for piped output vs interactive terminal auth, WASM build-tag gating to exclude subprocess code from browser builds, and adaptation to Bubbletea v2 API"
resolution: "Introduced a store abstraction package with a GitHub Gist backend, integrated Share To Gist and Open From Gist overlays into the TUI command menu, gated subprocess code behind build tags for WASM compatibility, and added catwalk tests covering both overlay flows"
---

# Remote Store Plugin Architecture: GitHub Gist Backend

## Problem

CalcMark's TUI editor had no way to share documents remotely or load documents from remote sources. Users could only work with local files. The editor needed a plugin system for remote data stores, with GitHub Gists as the first backend, enabling "Share To Gist" and "Open From Gist" commands from the command menu (Ctrl+H / F1).

## Investigation

Several key design decisions were evaluated:

- **Store abstraction vs direct integration**: A `Store` interface was defined in `store/store.go` to allow future backends (e.g., Pastebin, S3, GitLab Snippets). This keeps overlay logic backend-agnostic.
- **`gh` CLI vs GitHub API with OAuth**: Using the `gh` CLI avoids embedding OAuth credentials, leverages existing user authentication, and reduces dependency surface. Tradeoff: requires `gh` to be installed.
- **Subprocess execution vs Go HTTP client**: The `gh` CLI approach requires subprocess management with two incompatible modes: piped I/O for non-interactive commands and terminal handoff for interactive auth.
- **WASM compatibility**: Build tags were chosen over runtime checks because they prevent compilation failures entirely.

## Root Cause / Design Challenge

Four core engineering challenges drove the implementation:

1. **Two-phase subprocess lifecycle**: Auth status checks and gist operations need captured stdout/stderr (piped mode), while `gh auth login` needs full terminal control (interactive mode). These cannot use the same execution path.

2. **WASM build exclusion**: The `os/exec` package is unavailable in WASM targets. Any file importing it causes compilation failures. The entire subprocess layer must be invisible to the WASM compiler.

3. **Bubbletea v2 API differences**: The v1 pattern `tea.Exec(tea.WrapExecCommand(cmd), callback)` does not exist in v2. The correct v2 API is `tea.ExecProcess(cmd, callback)`.

4. **Modal overlay integration with save-prompt flow**: "Open From Gist" with unsaved changes must show a save prompt first, then resume the open-from-gist workflow after the user responds.

## Solution

### Store Interface and Executor Abstraction

The `Store` interface defines operations, and `Executor` decouples subprocess calls for testability:

```go
// store/store.go
type Store interface {
    CheckAvailable() error
    CheckAuth() error
    Share(content, description string, public bool) (*ShareResult, error)
    Open(id string) (*OpenResult, error)
}

type Executor interface {
    Run(name string, args ...string) (stdout, stderr []byte, err error)
}

var (
    ErrCLINotFound      = errors.New("store: CLI not found")
    ErrNotAuthenticated  = errors.New("store: not authenticated")
)
```

`RealExecutor` wraps `exec.Command` for production use. Tests use a `mockExecutor` with canned responses.

### Build-Tag Architecture for WASM

Native-only code uses `//go:build !js && !wasm`, WASM stubs use `//go:build js || wasm`:

```go
// store_commands.go — native only
//go:build !js && !wasm

func init() {
    EditorCommands = append(EditorCommands,
        Command{Name: "Share To Gist", ...},
        Command{Name: "Open From Gist", ...},
    )
}
```

The `init()` pattern ensures commands are automatically excluded from WASM builds without conditional logic at call sites.

### Two-Phase Subprocess Pattern

- **Phase 1 (piped)**: `store.Executor.Run()` for auth checks and gist operations — captures stdout/stderr programmatically.
- **Phase 2 (interactive)**: `tea.ExecProcess(cmd, callback)` for `gh auth login` — hands terminal control to the subprocess.
- **Retry messages**: After interactive auth completes, a retry message (e.g., `retryShareMsg{}`) resumes the workflow.

```go
// Check auth — if not authenticated, launch interactive login
if err := gist.CheckAuth(); err != nil {
    if errors.Is(err, store.ErrNotAuthenticated) {
        c := exec.Command(ghPath, "auth", "login")
        return m, tea.ExecProcess(c, func(err error) tea.Msg {
            if err != nil {
                return shareResultMsg{err: fmt.Errorf("auth failed: %w", err)}
            }
            return retryShareMsg{} // Resume workflow after auth
        })
    }
}
```

### Modal Overlay Integration

New overlays follow the established pattern:

1. Add `InputState` constants (`StateShareTo`, `StateOpenFrom`)
2. Add `enterShareTo()` / `enterOpenFrom()` in `mode_transitions.go`
3. Add key dispatch cases in `key_dispatch.go`
4. Add render cases in `view.go`
5. Add `PendingOpenFromRemote` to `save_prompt_handler.go` for unsaved-changes guard

## Verification

- **17 unit tests** in `store/gist_test.go` with `mockExecutor` covering all error paths
- **3 catwalk test suites** in `testdata/`:
  - `share_to_overlay` — overlay lifecycle, field cycling, cancel
  - `open_from_overlay` — text input, cancel
  - `open_from_unsaved` — save-prompt interception flow
- **Full suite**: `task test` and `task quality` pass cleanly
- **WASM build**: Store package compiles under WASM with stubs

## Best Practices

### Adding New Store Backends

1. Define the backend in a build-tagged file (`//go:build !js && !wasm`). Register via `init()` for automatic WASM exclusion.
2. Implement the `Store` interface. Keep it narrow.
3. Use `Executor` for all subprocess calls — never call `os/exec.Command` directly.
4. Use the two-phase subprocess pattern for interactive auth: piped `Executor.Run()` first, then `tea.ExecProcess()` if auth is needed.
5. Validate all fetched content with `filecheck.ValidateContent()` + size limits.

### Adding New Overlays

1. Add `InputState` constant
2. Add enter/exit methods in `mode_transitions.go`
3. Add key dispatch case in `key_dispatch.go`
4. Add render case in `view.go`
5. Integrate with save-prompt handler if the action could lose unsaved work
6. Keep overlay state in dedicated model fields

### Testing Strategy

- **Unit tests**: Mock `Executor` to test store logic without real CLI tools
- **Catwalk tests**: Every user-facing TUI behavior must have a catwalk test reproducing the exact key sequence
- **WASM build**: Verify after any change touching `os/exec` usage

## Common Pitfalls

### WASM Build Tags

- **Forgetting `//go:build !js && !wasm`** on files importing `os/exec` causes indirect WASM compilation failures (missing `syscall` symbols).
- **Build tags must cover the entire dependency chain.** Use `init()` registration to decouple the registry from backends.

### Bubbletea v2 API

- **`tea.ExecProcess` suspends Bubbletea** while the external process runs. No messages arrive until the process exits.
- **The callback must return a `tea.Msg`** — design specific message types for each workflow step.
- **Never block in `Update`** — run long operations in `tea.Cmd` functions.

### Content Handling

- **Remote content is untrusted.** Never skip `filecheck.ValidateContent()` even for the user's own Gists.
- **Enforce size limits** before content reaches the editor model.
- **Parse stderr for actionable errors.** Many CLI tools put human-readable messages on stderr.

## Related Documentation

- [Brainstorm](../../brainstorms/2026-02-27-remote-store-plugins-brainstorm.md) — Plugin architecture exploration
- [Implementation Plan](../../plans/2026-02-27-feat-remote-store-gist-plugin-plan.md) — Three-phase execution plan
- [TUI Mode Transitions](../ui-bugs/tui-mode-transitions-formatter-indexing-and-bracketed-paste-fixes.md) — Centralized mode transition patterns
- [Ctrl-O Stale State](../ui-bugs/ctrl-o-stale-state-and-unsaved-changes-detection.md) — File open must reset editor state
- [Bubbletea v2 Migration](../ui-bugs/bubble-tea-v2-migration-selection-undo-clipboard-fixes.md) — v2 API patterns and catwalk testing
- [Overlay Compositing](../ui-bugs/overlay-compositing-ansi-state-bleed-through.md) — ANSI state management for overlays
- [Editor Testing Guide](../../cmd/calcmark/tui/editor/TESTING.md) — Catwalk test specification

### External References

- [gh CLI documentation](https://cli.github.com)
- [Bubbletea v2](https://github.com/charmbracelet/bubbletea)
- [Catwalk testing](https://github.com/knz/catwalk)
- [datadriven](https://github.com/cockroachdb/datadriven)
