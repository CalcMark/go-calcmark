---
title: "feat: Remote open with URL paste support and CLI subcommand"
type: feat
status: completed
date: 2026-03-06
deepened: 2026-03-06
---

# Remote Open: URL Paste Support and CLI Subcommand

## Enhancement Summary

**Deepened on:** 2026-03-06
**Research agents used:** security-sentinel, architecture-strategist, code-simplicity-reviewer, best-practices-researcher, learnings-researcher, paste-pattern-explorer

### Key Simplifications (from review)
1. **No HTTPStore type** — a standalone `fetchURL()` function in `remote.go` replaces the over-engineered Store interface implementation (3/5 methods would be no-ops)
2. **No WASM stubs for cmd/** — no existing cmd file uses build tags; the CLI binary is never built for WASM
3. **No `--eval` flag** — ship as fetch-and-open-in-editor; users can pipe to `cm eval` later
4. **Single `remote.go` file** — no need to split across 3 files
5. **Paste fix covers ShareTo too** — the description input field in Share To Gist also silently drops pastes

---

## Overview

Two related improvements to CalcMark's remote document opening:

1. **TUI bug fix**: `tea.PasteMsg` is always routed to `handleBracketedPaste()` regardless of mode. Pasting in the Open From Gist overlay (and Share To Gist description field) silently inserts into the document body instead of the overlay input field.

2. **CLI `cm remote` subcommand**: A new subcommand with explicit `--gist` and `--http` flags for opening remote CalcMark documents in the editor from the terminal.

## Problem Statement

**TUI paste bug**: When a user opens the "Open From Gist" overlay and presses Cmd+V to paste a gist URL, the terminal sends a `tea.PasteMsg`. The `Update()` method in `model.go:575-579` routes this directly to `handleBracketedPaste()` which calls `insertPastedText()` — inserting into the document behind the overlay. The overlay's `openFromInput` field is never updated. The same bug affects the Share To Gist description field (`shareDescription`).

**No CLI remote access**: Opening remote documents is only possible through the TUI command menu. There's no way to run `cm remote <url>` from a script or terminal workflow.

## Proposed Solution

### Part 1: TUI Paste Fix

Route `tea.PasteMsg` through mode-aware dispatch. When mode is `StateOpenFrom` or `StateShareTo`, append pasted content to the overlay's input field instead of inserting into the document.

### Part 2: CLI `cm remote` Subcommand

```
cm remote --gist <id-or-url>     Open gist in TUI editor
cm remote --http <url>            Open URL in TUI editor
```

**Explicit flags** (`--gist` / `--http`) are required — no auto-detection. This keeps the CLI predictable and avoids ambiguity.

**Examples:**
```
cm remote --gist abc123def
cm remote --gist https://gist.github.com/user/abc123
cm remote --http https://raw.githubusercontent.com/CalcMark/go-calcmark/refs/heads/main/site/content/docs/examples/napkin-math.md
```

## Technical Approach

### Part 1: TUI Paste in Overlay Input Fields

**File: `cmd/calcmark/tui/editor/model.go`** (line ~575)

Current code:
```go
case tea.PasteMsg:
    return m.handleBracketedPaste(msg.Content)
```

Change to mode-aware dispatch:
```go
case tea.PasteMsg:
    switch m.mode {
    case StateOpenFrom:
        m.openFromInput += msg.Content
        return m, nil
    case StateShareTo:
        if m.shareField == 1 { // description field
            m.shareDescription += msg.Content
            return m, nil
        }
    }
    return m.handleBracketedPaste(msg.Content)
```

### Research Insights: Paste Handling

**From `tui-mode-transitions-formatter-indexing-and-bracketed-paste-fixes.md`:**
- Bracketed paste (`tea.PasteMsg`) requires explicit handler dispatch — terminals intercept Cmd+V and send content as a message type, not a key event.
- The `PasteMsg` handler runs BEFORE `handleKey` in the `Update()` switch, so it bypasses all mode-specific routing.
- Catwalk tests can simulate paste via `tea.PasteMsg{Content: s}` — use this to prove the fix.

**From paste pattern exploration:**
- `StateExport` does NOT need paste support (no text input fields, only selector).
- Other modes (Help, CommandMenu, Globals, FilePicker) also have no text input fields.
- Only `StateOpenFrom` and `StateShareTo` (description field) need paste routing.

### Part 2: CLI `cm remote` Subcommand

**New file: `cmd/calcmark/cmd/remote.go`**

A single file containing:
- Cobra subcommand with `--gist` (string) and `--http` (string) flags
- `fetchURL()` — standalone function for HTTP GET (~30 lines)
- Flag validation (exactly one of `--gist` or `--http` required)
- Dispatch to either `fetchURL()` or `store.GistStore.Open()`, then `runEdit()` with content

**Why no HTTPStore type:** The Store interface requires 5 methods (`Name`, `CheckAvailable`, `CheckAuth`, `Share`, `Open`). For a public HTTP GET, 3 are no-ops and 1 returns an error. When most of an interface's methods are stubs, the type doesn't belong behind that interface. A standalone `fetchURL()` function is clearer and simpler.

**Why no WASM stubs:** No existing file in `cmd/calcmark/cmd/` uses build tags. The CLI binary is never compiled for WASM — only the TUI editor store package has WASM stubs. If WASM becomes relevant for the CLI later, add stubs then (YAGNI).

**Why no `--eval` flag:** The default behavior opens in the editor, which already evaluates. If users need CLI eval from a URL, they can pipe: `curl <url> | cm eval`. Adding `--eval`, `--format`, and `--verbose` flags duplicates `eval.go`'s entire surface area.

### `fetchURL()` Implementation

```go
func fetchURL(rawURL string) (content, filename string, err error) {
    // 1. Validate URL scheme
    u, err := url.Parse(rawURL)
    if err != nil {
        return "", "", fmt.Errorf("invalid URL: %w", err)
    }
    if u.Scheme != "http" && u.Scheme != "https" {
        return "", "", fmt.Errorf("unsupported scheme %q (only http and https)", u.Scheme)
    }
    if u.Host == "" {
        return "", "", fmt.Errorf("URL has no host")
    }

    // 2. Fetch with timeout and size limit
    client := &http.Client{
        Timeout: 30 * time.Second,
        Transport: &http.Transport{
            TLSHandshakeTimeout:   10 * time.Second,
            ResponseHeaderTimeout: 15 * time.Second,
        },
        CheckRedirect: func(req *http.Request, via []*http.Request) error {
            if len(via) >= 10 {
                return fmt.Errorf("too many redirects")
            }
            if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
                return fmt.Errorf("redirect to non-HTTP scheme %q", req.URL.Scheme)
            }
            return nil
        },
    }

    resp, err := client.Get(rawURL)
    if err != nil {
        return "", "", fmt.Errorf("fetch failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return "", "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
    }

    // 3. Read with size limit (+1 to detect overflow)
    const maxSize = 1*1024*1024 + 1
    data, err := io.ReadAll(io.LimitReader(resp.Body, maxSize))
    if err != nil {
        return "", "", fmt.Errorf("read body: %w", err)
    }
    if len(data) >= int(maxSize) {
        return "", "", fmt.Errorf("response exceeds 1MB limit")
    }

    // 4. Validate content
    if err := validateRemoteContent(data); err != nil {
        return "", "", err
    }

    // 5. Infer filename from URL path
    name := path.Base(u.Path)
    if name == "" || name == "." || name == "/" {
        name = "remote.cm"
    }

    return string(data), name, nil
}
```

### Research Insights: HTTP Best Practices

**From security review:**
- `ResponseHeaderTimeout: 15s` catches servers that accept connections but never send headers (distinct from the 30s total timeout).
- `TLSHandshakeTimeout: 10s` prevents stalls during TLS negotiation.
- Validate scheme in `CheckRedirect` to prevent redirect to `file://` or `gopher://`.
- Use `io.LimitReader(resp.Body, maxSize+1)` — the `+1` pattern reads on the body stream and detects overflow without loading unlimited data.

**From best practices research:**
- Never use `http.DefaultClient` — it has zero timeout.
- Set a custom `User-Agent` header: `CalcMark/<version>` (optional, nice-to-have).
- Check `resp.StatusCode` before reading body to fail fast on errors.

### Architecture

```
cmd/calcmark/tui/editor/
  model.go                Mode-aware PasteMsg dispatch (modified)
  open_from_overlay.go    Unchanged (paste arrives via PasteMsg, not key)
  share_overlay.go        Unchanged (paste arrives via PasteMsg, not key)

cmd/calcmark/cmd/
  remote.go               Cobra command + fetchURL() + gist dispatch (new)
  remote_test.go           Unit tests for fetchURL and flag validation (new)
```

**4 files touched total** (2 modified, 2 new). Down from the original 10+ files.

## Acceptance Criteria

### Part 1: TUI Paste Fix

- [ ] Pasting a URL (Cmd+V / bracketed paste) into the Open From Gist overlay populates `openFromInput`
- [ ] Pasting into the Share To Gist description field populates `shareDescription`
- [ ] Pasting in normal editing mode still works as before (no regression)
- [ ] Pasting while in ShareTo visibility selector (field 0) falls through to document paste
- [ ] Catwalk test for paste-in-OpenFrom overlay passes
- [ ] Catwalk test for paste-in-ShareTo description field passes

### Part 2: CLI `cm remote`

- [ ] `cm remote --gist <id>` fetches and opens gist in TUI editor
- [ ] `cm remote --gist <url>` fetches and opens gist in TUI editor
- [ ] `cm remote --http <url>` fetches and opens URL in TUI editor
- [ ] Error when neither `--gist` nor `--http` is provided
- [ ] Error when both `--gist` and `--http` are provided
- [ ] Error for non-HTTP(S) URL schemes
- [ ] Error for content exceeding 1MB
- [ ] Error for HTTP 4xx/5xx responses
- [ ] Timeout after 30 seconds
- [ ] Content validated via `filecheck.ValidateContent()` (or equivalent)
- [ ] Redirects to non-HTTP schemes are rejected
- [ ] `cm remote --help` shows usage with examples
- [ ] Unit tests for `fetchURL` (mock HTTP server via `httptest.NewServer`)
- [ ] Unit tests for flag validation
- [ ] `task test` passes
- [ ] `task quality` passes

## Implementation Phases

### Phase 1: TUI Paste Fix (small, self-contained)

1. Add mode-aware `tea.PasteMsg` dispatch in `model.go` (OpenFrom + ShareTo)
2. Write catwalk tests for paste-in-overlay (OpenFrom and ShareTo)
3. Verify no regression with existing paste tests

**Files modified:**
- `cmd/calcmark/tui/editor/model.go` (~8 lines changed)
- `cmd/calcmark/tui/editor/testdata/open_from_paste` (new catwalk test)
- `cmd/calcmark/tui/editor/testdata/share_to_paste` (new catwalk test)

### Phase 2: CLI `cm remote` Subcommand

1. Create `remote.go` with Cobra command, `fetchURL()`, and gist dispatch
2. Create `remote_test.go` with `httptest.NewServer` tests and flag validation tests
3. Add example line to `root.go` Long description

**Files created:**
- `cmd/calcmark/cmd/remote.go`
- `cmd/calcmark/cmd/remote_test.go`

**Files modified:**
- `cmd/calcmark/cmd/root.go` (add example line to Long description)

## Security Considerations

- **URL scheme validation**: Only `http://` and `https://` allowed. Reject `file://`, `ftp://`, `gopher://`, etc. Validated both on initial URL and on redirects (via `CheckRedirect`).
- **Size limit**: 1MB enforced via `io.LimitReader` on the response body stream — content never exceeds limit in memory.
- **Content validation**: `filecheck.ValidateContent()` (or `validateRemoteContent()`) rejects binary content.
- **No credentials**: HTTP fetcher sends no auth headers. Gist auth delegated to `gh` CLI.
- **Timeout layering**: 30s total (`Client.Timeout`), 15s response headers (`ResponseHeaderTimeout`), 10s TLS (`TLSHandshakeTimeout`).
- **Redirect safety**: `CheckRedirect` validates scheme on every redirect hop, limiting to 10 total.
- **SSRF**: Not a concern — CLI runs locally on the user's machine.

## Edge Cases

- **Gist with no `.cm` files**: Falls back to first file (existing `selectGistFile` behavior)
- **Empty response**: Return clear error "remote document is empty"
- **Binary content**: `filecheck.ValidateContent()` rejects non-text content
- **URL with query params/fragments**: Pass through as-is to HTTP GET
- **Redirect to non-HTTP scheme**: Rejected by `CheckRedirect`
- **Slow server (trickle attack)**: 30s total timeout kills the request
- **Paste multiline text into overlay**: Append entire pasted content to input field (newlines included — overlay input is single-line, so newlines are harmless and will be trimmed by `strings.TrimSpace` on Enter)
- **Gist requiring auth in CLI**: `GistStore.CheckAuth()` detects this; print error directing user to run `gh auth login` first (no interactive auth in CLI mode)

## References

- Existing gist store: `cmd/calcmark/tui/editor/store/gist.go`
- Store interface: `cmd/calcmark/tui/editor/store/store.go`
- Paste handling: `cmd/calcmark/tui/editor/clipboard.go`
- Model update dispatch: `cmd/calcmark/tui/editor/model.go:571-579`
- Open From overlay: `cmd/calcmark/tui/editor/open_from_overlay.go`
- Share overlay: `cmd/calcmark/tui/editor/share_overlay.go`
- CLI eval pattern: `cmd/calcmark/cmd/eval.go`
- Prior gist plan: `docs/plans/2026-02-27-feat-remote-store-gist-plugin-plan.md`
- Solution doc: `docs/solutions/integration-issues/remote-store-gist-plugin-architecture.md`
- Solution doc: `docs/solutions/ui-bugs/tui-mode-transitions-formatter-indexing-and-bracketed-paste-fixes.md`
