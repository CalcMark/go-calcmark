# Pitfalls Research

**Domain:** CLI/TUI editor tool reaching v1 release (CalcMark)
**Researched:** 2026-02-02
**Confidence:** MEDIUM-HIGH (mix of project-specific analysis and verified ecosystem patterns)

## Critical Pitfalls

Mistakes that cause rewrites, broken releases, or major user-facing failures.

### Pitfall 1: Two-Pane Alignment Desynchronization Under Wrapping

**What goes wrong:**
The source pane and preview pane lose 1:1 vertical line alignment when text wrapping produces different numbers of visual lines in each pane. The cursor appears to point at the wrong preview result, or padding lines appear in unexpected places. This is already the most complex area of the CalcMark editor -- the entire `aligned.go` architecture exists because of this problem -- and any change to wrapping, pane width ratios, or preview rendering can silently break alignment.

**Why it happens:**
Both panes wrap independently. Source wraps based on `sourceContentWidth` (accounting for line numbers + gutter), while preview wraps based on `previewWidth`. When a source line wraps to 3 visual lines but its preview result wraps to only 1, padding lines must be inserted. The number of visual lines depends on content width, which depends on pane proportions, which can change with terminal resize or preview mode cycling (`PreviewFull` / `PreviewMinimal` / `PreviewHidden`). Any off-by-one in the padding calculation cascades through all subsequent lines.

The existing `computeAlignedPanes` and `ComputeAlignedModel` functions are the single source of truth, but `view.go` then performs *additional* adjustments for the edit buffer at render time (lines 719-778 in `view.go`), creating a second alignment path that can diverge.

**How to avoid:**
- Never modify alignment logic without running the full catwalk test suite (`go test ./cmd/calcmark/tui/editor -run Catwalk -v`).
- Add catwalk tests specifically for wrapping scenarios: long calc lines, long markdown headings, narrow terminal widths (40 columns), and edit buffer wrapping.
- Ensure `AlignedModel` is the ONLY source of truth. If the view layer needs to adjust for the edit buffer, that adjustment should flow through `AlignedModel`, not be patched in render code.
- Test at boundary widths: exactly where content wraps to 2 lines, where it barely fits on 1 line, and where it requires 3+ wrapped lines.

**Warning signs:**
- Preview pane shows results shifted one or more lines from their source lines.
- Tilde (`~`) lines in source pane appear at different positions than tilde lines in preview pane.
- `sourcePreviewMatch=false` in catwalk debug output.
- `highlightMatch=false` in catwalk debug output (cursor highlight at wrong visual line).

**Phase to address:**
TUI Editor Rewrite phase -- this is the core architecture change and must be rock-solid before anything else builds on it.

---

### Pitfall 2: ModelV2 Textarea Integration Losing Custom Editing Semantics

**What goes wrong:**
The `model_v2.go` file shows a transition from a custom editor (Model) to a textarea-based editor (ModelV2). The textarea component from `charmbracelet/bubbles` handles basic editing but does NOT understand CalcMark semantics: block detection, live evaluation, debounced re-evaluation, undo/redo with document snapshots, export dialogs, save prompts, or the alignment architecture. Adopting textarea without carefully preserving these behaviors results in a feature regression where the editor loses its differentiating functionality.

**Why it happens:**
The textarea component is attractive because it handles cursor movement, text insertion, scrolling, line wrapping, and clipboard out of the box. But it treats content as plain text. CalcMark needs to re-parse and re-evaluate on every content change, show live preview of calculation results, maintain two-pane alignment, and provide domain-specific keybindings. The current `ModelV2.syncDocumentFromTextarea()` does a full document rebuild on every keystroke, which defeats the incremental evaluation that `Model.reEvaluate()` and `EvaluateAffectedBlocks` provide.

**How to avoid:**
- Decide early: either use textarea as a pure input component and layer all CalcMark semantics on top (keeping the alignment architecture), OR continue with the custom editor and only borrow specific behaviors from textarea (like cursor blinking).
- If using textarea, implement debounced sync (50ms, matching the existing `evalDebounceDelay`) rather than syncing on every keystroke.
- Port all existing catwalk tests to work with ModelV2 before declaring it ready. Every test that passes on Model must also pass on ModelV2.
- Preserve incremental evaluation: track which blocks changed and use `EvaluateAffectedBlocks` rather than full re-evaluation.

**Warning signs:**
- Keystroke lag in the editor (full document rebuild is O(n) per keystroke instead of O(affected blocks)).
- Missing features compared to Model (undo/redo, export, save-as, search, preview mode cycling).
- Catwalk tests cannot be ported because ModelV2 has a different interface.
- `app.go` already references `editor.ModelV2` -- if the transition is incomplete, the shipped editor may be the less-capable version.

**Phase to address:**
TUI Editor Rewrite phase -- the v1/v2 decision must be made and fully implemented before any dependent work.

---

### Pitfall 3: GitHub Actions Release Workflow Go Version Mismatch

**What goes wrong:**
The release builds fail or produce binaries compiled with the wrong Go version. The project's `go.mod` specifies `go 1.24.4`, but the release workflow (`.github/workflows/release.yml`) pins `go-version: '1.21'`. This means CI cannot compile the project because Go 1.21 does not support language features or standard library APIs available in Go 1.24. Even if the build somehow succeeds, the resulting binaries miss optimizations and runtime improvements from newer Go versions.

**Why it happens:**
The release workflow was written when the project used an earlier Go version and was never updated when `go.mod` was bumped. Go's backward compatibility means the project compiles with newer Go, but CI pins an older version. Additionally, the Taskfile uses `build:all` which cross-compiles for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, and windows/amd64, but the release script may only produce WASM artifacts.

**How to avoid:**
- Update `.github/workflows/release.yml` to use `go-version-file: 'go.mod'` instead of a hardcoded version. This ensures CI always uses the version specified in `go.mod`.
- Add a pre-release check: `go version` output should match `go.mod` requirements.
- Include ALL platform binaries in the release (the Taskfile has `build:all` but the release workflow only runs `release.sh`; verify that `release.sh` calls `build:all`).
- Test the release workflow with `--local` flag before every release.

**Warning signs:**
- CI build fails immediately on `go mod tidy` or `go build` with syntax/feature errors.
- Binaries are compiled with Go 1.21 (check `go version -m <binary>`).
- Missing platform binaries in GitHub Release assets.

**Phase to address:**
Distribution / Release phase -- must be fixed before the v1 release tag is pushed.

---

### Pitfall 4: WASM Binary Size Explosion

**What goes wrong:**
The WASM binary becomes too large for practical web use. A Go WASM binary starts at ~1.3MB for an empty program. CalcMark imports `fmt`, `strings`, `time`, `os`, `path/filepath`, the decimal library, and the entire parser/evaluator stack. Without active management, the WASM binary can easily reach 5-10MB, making it impractical for browser-side use -- CalcMark's stated first-class WASM target.

**Why it happens:**
Go's standard compiler includes the entire Go runtime in every WASM binary. Each `import` pulls in significant code. The `fmt` package alone adds ~400KB. The Taskfile shows both a `build:wasm` (standard Go compiler) and a `build:wasm:tiny` (TinyGo) target, but TinyGo has compatibility limitations and may not compile the full CalcMark codebase.

**How to avoid:**
- Measure the current WASM binary size before starting v1 work: `GOOS=js GOARCH=wasm go build -ldflags "-s -w" -o calcmark.wasm ./cmd/calcmark && ls -lh calcmark.wasm`.
- Set a binary size budget (recommend: under 3MB uncompressed, under 1MB gzip'd for web delivery).
- Profile what contributes to binary size (use `go tool nm` or specialized WASM tools).
- If TinyGo is viable, use it for the WASM target. If not, minimize imports in the WASM build path (use build tags to exclude TUI, CLI, and OS-dependent code from WASM builds).
- Serve with gzip/Brotli compression -- Go WASM compresses well (50%+ reduction).

**Warning signs:**
- WASM binary exceeds 5MB uncompressed.
- TinyGo build fails on CalcMark code (check for unsupported language features).
- `build:wasm:tiny` task has not been run or tested recently.
- No WASM-specific build tags in the codebase separating TUI code from library code.

**Phase to address:**
Distribution / Release phase -- binary size must be measured and optimized before declaring WASM support as v1.

---

## Technical Debt Patterns

Shortcuts that seem reasonable but create long-term problems.

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Full document rebuild on every edit (`redetectBlockTypes` recreates entire document) | Correctness: ensures all block types are accurate | O(n) per keystroke degrades with document size; 100-line documents may feel sluggish | Acceptable in v1 MVP if debounce mitigates; must be optimized before v2 |
| Hardcoded color values (`lipgloss.Color("236")`) throughout `view.go` | Works immediately | Theme changes require find-and-replace across hundreds of lines; impossible for users to customize | Never -- use `m.styles.*` consistently (partially done, but fallback to "236" is widespread) |
| Two alignment code paths (AlignedModel + view.go edit-buffer adjustments) | Edit buffer rendering works correctly | Any alignment change must be made in two places; bugs where they diverge are hard to detect | Acceptable only temporarily; unify before v1 |
| Undo stack stores full document content strings | Simple implementation | 100 undo states * large documents = significant memory; no structural diffing | Acceptable for v1 given 100-entry cap; consider operational transforms post-v1 |
| `getDocumentContent()` iterates all blocks and joins on every call | Simple, correct | Called frequently (cache key computation, save detection, undo); O(n) per call | Should cache or hash; acceptable for v1 with monitoring |

## Integration Gotchas

Common mistakes when connecting components or external services.

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| Cobra CLI + TUI | Help text for TUI keybindings documented in Cobra `--help` but not updated when TUI keybindings change | Generate help text from the same `shared.KeyMap` struct used by the TUI; single source of truth |
| Cobra `--version` + `version.go` | Version constant updated in `version.go` but Cobra root command `Version` field not wired up | Wire `rootCmd.Version = calcmark.Version` in cmd initialization; never hardcode version strings |
| Textarea (bubbles) + custom evaluator | Textarea fires `Update` on every key; syncing document on each fires full re-evaluation | Debounce document sync; only re-evaluate after typing pauses (existing 50ms pattern) |
| GitHub Actions + release.sh | Workflow specifies different Go version than `go.mod` | Use `go-version-file: 'go.mod'` in the workflow |
| WASM build + TUI code | WASM target imports TUI packages that use terminal I/O; fails at runtime in browser | Use build tags (`//go:build !js` on TUI files, `//go:build js` on WASM entry point) |
| Help content + actual keybindings | Help screen says "Ctrl+P = preview" but keybinding was changed to "Tab" | Generate help content from `shared.KeyMap` rather than hardcoding strings |

## Performance Traps

Patterns that work at small scale but fail as usage grows.

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Full document re-parse on every line edit (`redetectBlockTypes` calls `document.NewDocument`) | Input lag on long documents; keystroke delay exceeds 50ms | Use incremental block detection: only re-parse the block containing the edited line; use `EvaluateAffectedBlocks` for evaluation | ~50+ lines with complex calculations |
| `GetLines()` allocates new slice on every call | GC pressure during rapid typing; View() calls GetLines() multiple times per render | Cache the lines slice; invalidate on document change | ~100+ lines with fast typing |
| `computeCacheKey` iterates all lines and XORs character values | Cache key computation is O(n) on total document size | Use a rolling hash or content version counter that increments on each edit | ~500+ lines |
| `lipgloss.Width()` called per-line in render loop | Style measurement is expensive; called once per visual line per render | Pre-compute widths; cache rendered strings between renders when content hasn't changed | ~50+ visual lines (including wrapped lines) |
| `AlignedModel` recomputed on every `View()` call (view.go uses value receiver, cache on pointer receiver doesn't persist) | Double computation per render; `computeAlignedModelFresh` runs even if cached version exists on the pointer | Pass `*Model` in View or restructure to persist cache across View calls | Any document size (constant 2x overhead) |

## Security Mistakes

Domain-specific security issues relevant to a CLI/TUI tool with file I/O.

| Mistake | Risk | Prevention |
|---------|------|------------|
| Export path not validated for directory traversal | User could accidentally or maliciously write exported files to arbitrary paths (noted as intentional in SECURITY.md) | Document this behavior clearly; consider a confirmation prompt for paths outside CWD |
| WASM binary served without CSP headers | Browser consumers could be vulnerable to injection if CalcMark WASM is loaded in an insecure context | Provide documentation for web consumers on proper CSP headers |
| `saveFile` uses `os.WriteFile` with mode 0644 | Correct for most cases, but on shared systems could expose document content | Document that saved files are world-readable; consider using 0600 for new files |
| No input sanitization on filenames from save-as dialog | Filenames with special characters could cause issues on some OSes | Validate filenames (no null bytes, control characters, or OS-reserved names) |

## UX Pitfalls

Common user experience mistakes specific to TUI editors and CLI tools.

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Autocomplete suggestions that fire on every keystroke | Completion popups interrupt typing flow; users feel "fought" by the editor | Only show completions after explicit trigger (e.g., Tab or Ctrl+Space); debounce at minimum 200ms; allow user to dismiss permanently |
| Help text that shows ALL keybindings at once | Users overwhelmed; cannot find the one binding they need | Show context-sensitive help (editing keybindings when editing, navigation when navigating); group by action not by key |
| Help content in TUI that differs from CLI `--help` | Users lose trust when two help systems disagree | Generate both from the same source data; test that they match |
| Status bar showing internal state names | Status bar currently shows `mode=0` style internal state in debug output; if this leaks to production view, users see meaningless numbers | Status bar already hides internal mode (good), but verify no code paths expose it |
| Terminal resize causes flash of unstyled content | User sees white/default-bg lines briefly during resize | Render a complete frame immediately on `WindowSizeMsg` before returning; use lipgloss `Background` on all empty space |
| Undo/redo with no visual feedback | User presses undo but cannot tell what changed | Flash changed lines or show a brief "Undid: [summary]" in status bar |
| Cursor invisible on wrapped lines | When editing a line that wraps, cursor is only visible on the first visual line | Ensure cursor rendering in `renderEditLineWrapped` correctly tracks which wrapped segment contains the cursor (existing implementation does this, but verify under narrow terminals) |

## "Looks Done But Isn't" Checklist

Things that appear complete but are missing critical pieces.

- [ ] **Help system:** Help text written but not tested against actual keybindings -- verify every documented binding actually works by writing a test that presses the key and checks the result
- [ ] **Autocomplete:** Completion appears but does not handle edge cases -- test with: empty input, cursor in middle of word, cursor at end of line, variable name that is a prefix of another
- [ ] **Cross-platform binaries:** Binaries build but are not tested -- download each platform binary on actual hardware or emulator and verify basic operations (open file, edit, save, quit)
- [ ] **WASM build:** WASM compiles but runtime behavior is untested -- load in a real browser (Chrome, Firefox, Safari) and exercise the evaluator; check for missing `syscall/js` bindings
- [ ] **Dark theme:** Theme looks correct on the developer's terminal but breaks on others -- test on: macOS Terminal.app (limited color), iTerm2 (truecolor), VS Code terminal, Windows Terminal, tmux (can strip colors)
- [ ] **Line wrapping:** Wrapping works for ASCII but not for Unicode -- test with: CJK characters (double-width), emoji, combining marks, RTL text
- [ ] **Save/load roundtrip:** Document saves and loads but content differs subtly -- verify with `diff` that saved content, when loaded and re-saved, produces identical bytes
- [ ] **Version in binary:** `version.go` updated but `--version` flag shows wrong value -- verify with `./cm --version` after building with release ldflags
- [ ] **Release notes:** GitHub Release created but notes are auto-generated boilerplate -- write human-readable release notes highlighting what's new for v1 users

## Recovery Strategies

When pitfalls occur despite prevention, how to recover.

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Alignment desync | MEDIUM | Reproduce with catwalk test; fix in `AlignedModel` computation; re-run full test suite |
| ModelV2 regression | HIGH | If v1 ships with incomplete ModelV2, must either fix forward or revert to Model; reverting requires changing `app.go` back |
| Wrong Go version in CI | LOW | Update `release.yml`, delete bad tag, re-tag, re-push |
| WASM too large | MEDIUM | Add build tags to separate TUI from library; audit imports in WASM path; consider TinyGo |
| Help text stale | LOW | Audit all help strings; add test that compares help output to keybinding definitions |
| Binary missing for platform | LOW | Add the missing GOOS/GOARCH to Taskfile `build:all`; re-run release |
| Performance regression in interpreter | HIGH | Requires profiling (`go test -bench`); may need algorithmic changes in evaluator |
| Autocomplete too aggressive | LOW | Change trigger from automatic to explicit (Tab key); add debounce |

## Pitfall-to-Phase Mapping

How roadmap phases should address these pitfalls.

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Two-pane alignment desync | TUI Editor Rewrite | Catwalk tests with wrapping scenarios pass; `sourcePreviewMatch=true` in all tests |
| ModelV2 feature regression | TUI Editor Rewrite | All existing Model catwalk tests ported and passing on ModelV2 |
| Help content drift | Help System | Automated test comparing help output to `shared.KeyMap` definitions |
| Autocomplete UX annoyance | Autocomplete | User testing or manual review; debounce timing verified via test |
| Go version mismatch in CI | Distribution/Release | `go version` step in CI outputs version matching `go.mod` |
| WASM binary size | Distribution/Release | CI step that measures binary size and fails if over budget |
| Cross-platform binary issues | Distribution/Release | Matrix test job that runs basic smoke test on each platform |
| Performance regression | Every Phase | Benchmark tests (`task bench`) run before and after each phase; compare results |
| Documentation/code drift | Documentation | CI check that README examples compile and run |
| Theme/terminal compatibility | TUI Editor Rewrite | Manual test on at least 3 terminals (macOS Terminal.app, iTerm2/WezTerm, Windows Terminal) |

## Sources

- [State of Terminal Emulators 2025 - Jeff Quast](https://www.jeffquast.com/post/state-of-terminal-emulation-2025/) -- Unicode width calculation challenges in terminals (HIGH confidence)
- [Bubbletea Issue #43 - Display issue when content exceeds terminal width](https://github.com/charmbracelet/bubbletea/issues/43) -- Developers must handle wrapping themselves (HIGH confidence, verified via WebFetch)
- [Bubbletea Issue #1036 - View() hangs on AdaptiveColor](https://github.com/charmbracelet/bubbletea/issues/1036) -- Race condition with lipgloss style rendering (MEDIUM confidence)
- [Bubbletea Issue #1225 - lipgloss Width() rendering bug](https://github.com/charmbracelet/bubbletea/issues/1225) -- Dynamic width causes incorrect rendering (MEDIUM confidence)
- [Go Module Release Workflow - Official Go Docs](https://go.dev/doc/modules/release-workflow) -- v1 backward compatibility commitments (HIGH confidence)
- [GoReleaser CGO Limitations](https://goreleaser.com/limitations/cgo/) -- Cross-compilation requires CGO_ENABLED=0 for pure Go projects (HIGH confidence)
- [Cobra User Guide](https://github.com/spf13/cobra/blob/main/site/content/user_guide.md) -- Auto-generated help, subcommand grouping (HIGH confidence)
- [Go WASM Binary Size Analysis](https://dev.bitolog.com/minimizing-go-webassembly-binary-size/) -- fmt adds ~400KB; base binary ~1.3MB (MEDIUM confidence)
- [Tips for building Bubble Tea programs](https://leg100.github.io/en/posts/building-bubbletea-programs/) -- Best practices for Bubbletea architecture (MEDIUM confidence)
- [CLI Guidelines](https://news.ycombinator.com/item?id=25304257) -- CLI help best practices discussion (LOW confidence, community source)
- Project analysis: `go.mod`, `Taskfile.yml`, `release.yml`, `model.go`, `view.go`, `model_v2.go`, `aligned.go`, `state.go`, `RELEASE.md`, `SECURITY.md` -- Direct code inspection (HIGH confidence)

---
*Pitfalls research for: CalcMark CLI/TUI v1 release*
*Researched: 2026-02-02*
