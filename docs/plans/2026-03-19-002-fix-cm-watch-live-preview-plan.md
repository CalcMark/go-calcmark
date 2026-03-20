---
title: "fix: cm watch live preview — styling, file watching, and logging"
type: fix
status: active
date: 2026-03-19
---

# fix: cm watch live preview — styling, file watching, and logging

## Overview

The `cm watch` command has three bugs that make it unusable:

1. **No styling** — The watch page shell (`watchPageTemplate`) has ~5 lines of CSS, but the rendered content uses class names (`.calc-block`, `.calc-line`, `.calc-source`, `.calc-inline-result`, `.calc-error`, `.text-block`, `.frontmatter`, etc.) from `preview.gohtml` that have **no matching CSS rules**. The full styles exist in `default.gohtml` (~200 lines) but are not included in the watch page.

2. **File changes not detected on save** — `fsnotify` is set up to watch the file directly (`watcher.Add(filename)`) but many editors (vim, VS Code, JetBrains, nano) use **atomic saves**: write to a temp file, then rename over the original. This removes the original inode, causing fsnotify to lose the watch silently. The fix is to watch the **parent directory** and filter events for the target filename.

3. **No stderr/stdout logging** — Only startup messages ("Watching..." and "Preview: ...") are printed. No logging for: file change detected, re-render completed, WebSocket client connect/disconnect, or debounce activity. Makes debugging impossible.

## Proposed Solution

### Bug 1: Include `default.gohtml` styles in watch page

Extract the CSS from `default.gohtml` into the `watchPageTemplate` constant in `watch.go`. The content classes are identical between `preview.gohtml` and `default.gohtml`, so the styles transfer directly. Add the watch-specific status indicator styles.

**Files:** `cmd/calcmark/cmd/watch.go` (update `watchPageTemplate` constant)

### Bug 2: Watch parent directory, filter for target file

Replace `watcher.Add(filename)` with `watcher.Add(filepath.Dir(absPath))`. In the event handler, compare `event.Name` against the absolute path of the target file. Handle `Rename`/`Remove` events (editor atomic save creates these) in addition to `Write`/`Create`.

**Files:** `cmd/calcmark/cmd/watch.go` (update `runWatch` and `watchLoop`)

### Bug 3: Add structured stderr logging

Log key events to stderr:
- `[watch] change detected: <filename>` — when a matching file event fires
- `[watch] re-rendered (<duration>)` — after successful re-render
- `[watch] render error: <err>` — on render failure (already exists)
- `[watch] client connected (<n> total)` — WebSocket connect
- `[watch] client disconnected (<n> total)` — WebSocket disconnect
- `[watch] notified <n> client(s)` — after broadcasting reload

**Files:** `cmd/calcmark/cmd/watch.go` (update `watchLoop`, `addClient`, `removeClient`)

## Acceptance Criteria

- [ ] Watch preview page renders with full `default.gohtml` styling (calc blocks, text blocks, frontmatter, tables, code, blockquotes)
- [ ] File changes are detected when editors use atomic save (write-temp-rename pattern)
- [ ] File changes are detected on normal `Write` events
- [ ] stderr shows structured log lines for change detection, re-render, and WebSocket lifecycle
- [ ] Existing security properties preserved (loopback-only, session token, CSP headers, origin validation)
- [ ] All existing tests pass
- [ ] New tests written TDD-style proving each bug before fixing

## TDD Test Plan

Tests must be written **first** to prove the bugs exist, then the fixes make them pass.

### Test 1: Watch page includes calc-block styling (`watch_test.go`)

```go
func TestHandlePage_IncludesCalcStyles(t *testing.T)
```
- Create a `watchServer` with rendered HTML containing `.calc-block` markup
- Call `handlePage` via `httptest.NewRecorder`
- Assert response body contains `.calc-block {` CSS rule
- **Proves bug 1:** Currently fails because `watchPageTemplate` has no calc styles

### Test 2: Watch detects atomic-save file changes (`watch_test.go`)

```go
func TestWatchLoop_AtomicSave(t *testing.T)
```
- Create a temp dir with a `.cm` file
- Start fsnotify watcher on the **directory** (the fix)
- Simulate atomic save: write temp file, rename over original
- Assert the watchServer's `html` field updates within a timeout
- **Proves bug 2:** Currently fails because watcher watches the file, not directory

### Test 3: Logging on file change (`watch_test.go`)

```go
func TestWatchLoop_LogsOnChange(t *testing.T)
```
- Capture stderr output during a file change event
- Assert log line contains `[watch] change detected`
- **Proves bug 3:** Currently fails because no logging exists

### Test 4: WebSocket client logging (`watch_test.go`)

```go
func TestAddRemoveClient_Logs(t *testing.T)
```
- Capture stderr during addClient/removeClient
- Assert log lines contain `[watch] client connected` and `[watch] client disconnected`

## Implementation Order

1. Write failing tests (all 4)
2. Fix bug 1: CSS in watchPageTemplate
3. Fix bug 2: Directory-level watching
4. Fix bug 3: Structured logging
5. Run `task test` — all pass
6. Run `task quality` — clean
7. Manual smoke test with `task build && ./cm watch <file>`

## Sources

- `cmd/calcmark/cmd/watch.go` — all watch logic
- `cmd/calcmark/cmd/watch_test.go` — existing tests
- `format/templates/default.gohtml` — CSS styles to extract
- `format/templates/preview.gohtml` — content fragment template
- fsnotify docs: atomic save handling requires directory-level watching
