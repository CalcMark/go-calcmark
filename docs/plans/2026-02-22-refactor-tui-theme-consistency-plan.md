---
title: "Refactor: TUI Theme Consistency"
type: refactor
status: completed
date: 2026-02-22
brainstorm: docs/brainstorms/2026-02-22-tui-theme-consistency-brainstorm.md
---

# Refactor: TUI Theme Consistency

## Overview

Eliminate terminal-level color bleed-through in the CalcMark TUI by completing the partially-implemented theme system. Replace ~150 hardcoded `lipgloss.Color(...)` calls across 15+ files with `lipgloss.AdaptiveColor` references from a centralized semantic palette. Provide good default dark and light themes. Add basic block-level syntax highlighting in the source pane.

## Problem Statement

The TUI is unusable on light terminals. Dark theme colors (hardcoded as ANSI 256-color codes like `"236"` or hex strings like `"#1E1E1E"`) render as invisible text and dark rectangles on light backgrounds. The existing theme infrastructure (`config/theme/palette.go` with 15 AdaptiveColor values, `ThemeConfig` with ~40 fields, `BuildStyles()`) was started but never wired into components. Two parallel theme systems exist that are disconnected from each other and from the actual rendering code.

## Proposed Solution

1. **Expand `palette.go`** to ~25 semantic AdaptiveColor values covering all UI surfaces
2. **Rewrite `BuildStyles()`** to construct styles from AdaptiveColor palette values instead of single hex strings
3. **Replace all hardcoded colors** in components, editor views, overlays, and REPL with palette references
4. **Extract shared overlay rendering** to eliminate 4 copies of identical border/padding code
5. **Add block-level syntax highlighting** using 3 new palette colors for frontmatter/markdown/calc lines
6. **Simplify `ThemeConfig`** to ~10-15 user-facing overrides; detect and warn on deprecated keys

## Technical Approach

### Architecture

The color resolution chain becomes:

```
defaults.toml overrides → ThemeConfig (~10 fields)
                              ↓
                         palette.go (fills Dark or Light slot based on color_mode)
                              ↓
                         BuildStyles() → Styles struct (lipgloss.Style with AdaptiveColor)
                              ↓
                         Component Default*Style() functions (source from palette)
                              ↓
                         View rendering (references Styles + component styles)
                              ↓
                         lipgloss resolves Light/Dark at Render() time
```

Key architectural decisions:
- `lipgloss.AdaptiveColor` is the **only** color primitive used in rendering code
- `lipgloss.SetHasDarkBackground()` is called once at startup based on `color_mode` config
- The `BuildStyles()` singleton pattern (via `sync.Once`) remains valid because AdaptiveColor resolves at render time, not at construction time
- Component style structs remain for testability; their `Default*()` constructors source from the palette

### Implementation Phases

#### Phase 1: Foundation — Palette and Config

**Goal:** Expand the palette, simplify ThemeConfig, and update BuildStyles() so the infrastructure is ready for component migration.

**Tasks:**

1. **Expand `cmd/calcmark/config/theme/palette.go`** — Add ~10 new AdaptiveColor values:
   - `SourcePaneBg` / `PreviewPaneBg` (2-3% contrast difference)
   - `PopupBg` / `PopupBorder` / `PopupSelectedBg`
   - `SourceFrontmatter` / `SourceMarkdown` / `SourceCalc` (block-level highlighting)
   - `SelectionBg` / `SelectionFg`
   - `DividerFg`
   - `OverlayBg` / `OverlayBorder`

2. **Simplify `cmd/calcmark/config/types.go`** — Reduce `ThemeConfig` from ~40 fields to ~10-15:
   - Keep: `primary`, `accent`, `error`, `warning`, `muted`, `dimmed`, `output`, `bright`, `separator`
   - Add: `source_pane_bg`, `preview_pane_bg`, `status_bar_bg`
   - Remove: `edit_line_bg`, `edit_line_fg`, `cursor_bg`, `cursor_fg`, `current_line_bg`, `current_line_fg`, `line_number`, `source_text`, `calc_var_name`, `calc_arrow`, `calc_value`, `md_text`, `md_h1_bg`, `md_h2_bg`, `md_heading`, `md_link`, `md_quote`, `md_code`, `md_code_bg`, `context_footer_bg`, `prompt_fg`, `prompt_bg`, `input_fg`, `input_bg`, `input_cursor`

3. **Rewrite `cmd/calcmark/config/theme.go` BuildStyles()**:
   - Import `theme` package for palette values
   - Replace `lipgloss.Color(t.Primary)` with palette AdaptiveColor references
   - User config overrides apply to the palette slot matching their `color_mode` (dark slot for `color_mode="dark"`, light slot for `color_mode="light"`)
   - Add new style fields: `SourceFrontmatter`, `SourceMarkdown`, `SourceCalc` for block-level highlighting

4. **Update `cmd/calcmark/config/defaults.toml`**:
   - Remove entries for deleted ThemeConfig fields
   - Change `color_mode = "auto"` to `color_mode = "dark"` in embedded defaults
   - Keep ~10-15 remaining override fields

5. **Update `cmd/calcmark/config/config.go`**:
   - Add deprecated key detection after Viper unmarshal: compare `viper.AllKeys()` under `[tui.theme]` against known fields, log warnings for unknown keys
   - Keep `DarkMode` bool fallback for one release cycle with deprecation warning
   - `color_mode = "auto"` in user configs triggers deprecation warning but resolves to dark

6. **Remove `cmd/calcmark/config/theme/styles.go`** — Dead code (package-level styles never imported)

**Success criteria:**
- `palette.go` has ~25 AdaptiveColor values
- `BuildStyles()` produces styles using AdaptiveColor
- `ThemeConfig` has ~10-15 fields
- Removed keys in user config produce deprecation warnings on stderr
- `config/theme/styles.go` deleted
- `task test` passes (no breaking changes yet since consumers still use old patterns)

#### Phase 2: Component Migration

**Goal:** Rewrite all component `Default*Style()` functions to source colors from the palette.

**Tasks:**

1. **`cmd/calcmark/tui/components/suggest.go`** — Rewrite `DefaultPopupStyle()` and `DefaultAutosuggestStyle()`:
   - Replace `#5C5C5C`, `#1E1E1E`, `#4A90D9`, `#CCCCCC`, `#888888`, `#666666` with palette references
   - Use `theme.PopupBg`, `theme.PopupBorder`, `theme.PopupSelectedBg`, `theme.Text`, `theme.TextMuted`

2. **`cmd/calcmark/tui/components/statusbar.go`** — Rewrite `DefaultStatusBarStyle()`:
   - Replace `#FFFFFF`, `#FF6B6B`, `#888888`, `#4ECDC4`, `#000000` with palette references
   - Use `theme.StatusBg`, `theme.StatusFg`, `theme.Error`, `theme.Success`

3. **`cmd/calcmark/tui/components/globals.go`** — Rewrite `DefaultGlobalsPanelStyle()`:
   - Replace `#444444`, `#FFFFFF`, `#4ECDC4`, `#FFD93D`, `#333333`, `#666666` with palette references

4. **`cmd/calcmark/tui/components/contextfooter.go`** — Replace ~8 hardcoded ANSI color codes:
   - Replace `"39"`, `"252"`, `"246"`, `"196"`, `"255"`, `"250"`, `"220"`, `"240"` with palette references

5. **`cmd/calcmark/tui/components/errors.go`** and **`cmd/calcmark/tui/components/pinned.go`** — Replace any hardcoded colors

**Success criteria:**
- Zero hardcoded `lipgloss.Color(...)` literals in `cmd/calcmark/tui/components/`
- All `Default*Style()` functions reference `theme.*` palette values
- `task test` passes

#### Phase 3: Shared Overlay Extraction

**Goal:** Extract the duplicated overlay border/padding pattern into a shared component.

**Tasks:**

1. **Create `cmd/calcmark/tui/components/overlay.go`**:
   ```go
   type OverlayStyle struct {
       BorderFg  lipgloss.TerminalColor
       ItemBg    lipgloss.TerminalColor
       SelectedBg lipgloss.TerminalColor
       TextFg    lipgloss.TerminalColor
       MutedFg   lipgloss.TerminalColor
   }

   func DefaultOverlayStyle() OverlayStyle { /* source from palette */ }

   func RenderOverlayBox(content string, width, height int, style OverlayStyle) string
   ```

2. **Refactor `cmd/calcmark/tui/editor/view_overlays.go`**:
   - `renderCommandMenuPopup()` — Use `RenderOverlayBox()` + `OverlayStyle`
   - `renderFilePickerOverlay()` — Use `RenderOverlayBox()` + `OverlayStyle`

3. **Refactor `cmd/calcmark/tui/editor/help_overlay.go`** — Use shared overlay

4. **Refactor `cmd/calcmark/tui/editor/export_overlay.go`** — Use shared overlay

5. **Fix `overlayPadLine()` signature** in `cmd/calcmark/tui/editor/view_util.go`:
   - Change `bg lipgloss.Color` to `bg lipgloss.TerminalColor` for AdaptiveColor compatibility
   - Update all call sites

**Success criteria:**
- Single `OverlayStyle` struct and `RenderOverlayBox()` helper
- Four overlay renderers use the shared code
- `overlayPadLine()` accepts `lipgloss.TerminalColor`
- `task test` passes
- No duplicated border-drawing code

#### Phase 4: Editor View Migration

**Goal:** Replace all ~50 hardcoded colors in editor view rendering with palette references.

**Tasks:**

1. **`cmd/calcmark/tui/editor/view_panes.go`** — Replace ~30 instances of `lipgloss.Color("236")`:
   - Use `m.sourcePaneBg()` and `m.previewPaneBg()` helpers (already exist but unused)
   - Wire these helpers to return palette AdaptiveColor values
   - Every `ensureFullWidth(line, width, lipgloss.Color("236"))` → `ensureFullWidth(line, width, m.sourcePaneBg())`
   - Preview pane uses `m.previewPaneBg()` for its calls

2. **`cmd/calcmark/tui/editor/view.go`** — Remove override-after-construction pattern:
   - Eliminate inline overrides of `DefaultStatusBarStyle()` backgrounds
   - `DefaultStatusBarStyle()` already produces correct themed styles from Phase 2

3. **`cmd/calcmark/tui/editor/view_lines.go`** — Replace hardcoded selection colors:
   - Replace `lipgloss.Color("240")` and `lipgloss.Color("255")` in `renderLineWithSelection()` with `theme.SelectionBg` and `theme.SelectionFg`

4. **`cmd/calcmark/tui/editor/view_footer.go`** — Replace any hardcoded colors

5. **`cmd/calcmark/tui/editor/sidebyside.go`** — Replace divider color:
   - Replace `lipgloss.Color("240")` with `theme.DividerFg`

6. **`cmd/calcmark/tui/editor/filepicker.go`** — Replace hardcoded ANSI colors:
   - Set `fp.Styles.Directory`, `fp.Styles.File`, `fp.Styles.Selected`, `fp.Styles.Cursor` from palette
   - Use conditional values based on `config.IsDarkMode()` if bubbles filepicker doesn't accept AdaptiveColor

7. **`cmd/calcmark/tui/editor/markdown.go`** — Update `createMinimalStyle()`:
   - Create dual style configs: `createLightStyle()` and `createDarkStyle()`
   - Select at `NewMarkdownRenderer()` construction time based on `config.IsDarkMode()`
   - Light style: dark text colors on white background, high-contrast code blocks
   - Dark style: current values (light text on dark background)

**Call-site-to-palette mapping for `ensureFullWidth()` / `padToWidth()`:**

| Call location | Current value | Palette value |
|---|---|---|
| `view_panes.go` renderSourcePaneAligned | `lipgloss.Color("236")` | `m.sourcePaneBg()` |
| `view_panes.go` renderPreviewPaneAligned | `lipgloss.Color("236")` | `m.previewPaneBg()` |
| `view.go` status bar area | `lipgloss.Color("236")` | `theme.StatusBg` |
| `view_overlays.go` overlays | `lipgloss.Color("#1E1E1E")` | `theme.OverlayBg` |
| `view_panes.go` tilde lines | `lipgloss.Color("236")` | `m.sourcePaneBg()` |
| `view_panes.go` separator | `lipgloss.Color("236")` | `m.previewPaneBg()` |

**Success criteria:**
- Zero hardcoded `lipgloss.Color(...)` in `cmd/calcmark/tui/editor/`
- `sourcePaneBg()` and `previewPaneBg()` used consistently
- Selection highlighting uses palette colors
- Glamour has light/dark style configs
- `task test` passes

#### Phase 5: REPL Migration

**Goal:** Replace ~11 inline hardcoded styles in the REPL with palette references.

**Tasks:**

1. **`cmd/calcmark/tui/repl/view.go`** — Replace ANSI 256-color codes:
   - Replace `"252"`, `"236"`, `"3"`, `"9"`, `"240"`, `"6"`, `"15"` with palette AdaptiveColor references
   - Reuse component styles where applicable (e.g., `DefaultStatusBarStyle()` for status rendering)

2. **`cmd/calcmark/tui/repl/model.go`** — Ensure styles field sources from palette

**Success criteria:**
- Zero hardcoded color literals in `cmd/calcmark/tui/repl/`
- REPL reuses component styles where applicable
- `task test` passes

#### Phase 6: Block-Level Syntax Highlighting

**Goal:** Source pane lines are subtly colored by block type (frontmatter, markdown, calc).

**Tasks:**

1. **Determine block type per line** in `cmd/calcmark/tui/editor/view_panes.go`:
   - Frontmatter: `lineNum < m.frontmatterLineCount()` (method may need to be added or use existing `document.Block` classification)
   - Calculation: `LineResult.IsCalc` or `document.NewDetector().IsCalculation(source)`
   - Markdown: everything else (default)

2. **Apply subtle foreground tint** in `renderSourcePaneAligned()`:
   - Before rendering normal source lines, apply the block-type color from `m.styles.SourceFrontmatter`, `m.styles.SourceMarkdown`, or `m.styles.SourceCalc`
   - These are subtle foreground tints, not background changes
   - Must not affect cursor line, edit buffer, or selection rendering

3. **Palette values** (from Phase 1):
   - `SourceFrontmatter`: Light `#6e7781` / Dark `#7d8590` (muted gray — YAML metadata)
   - `SourceMarkdown`: Light `#1a1a1a` / Dark `#e5e5e5` (default text — no change from baseline)
   - `SourceCalc`: Light `#0969da` / Dark `#79c0ff` (subtle blue tint — calculation lines)

**Success criteria:**
- Frontmatter lines have a muted gray tint
- Calculation lines have a subtle blue tint
- Markdown lines use default text color
- No performance impact (block type determined from existing model data, no re-parsing)
- Cursor line, edit buffer, and selection highlighting are unaffected
- `task test` passes

#### Phase 7: Test Regeneration and Validation

**Goal:** Regenerate all golden test expectations and manually validate light/dark modes.

**Tasks:**

1. **Regenerate catwalk golden files**:
   ```bash
   go test ./cmd/calcmark/tui/editor -run Catwalk -v -args -rewrite
   ```
   Note: Catwalk tests use `lipgloss.SetColorProfile(termenv.Ascii)` which strips all ANSI codes. Color changes alone should NOT break these tests. However, structural changes (overlay extraction, new padding) may affect output.

2. **Run full test suite**: `task test`

3. **Manual validation** — Launch the TUI in both modes:
   - `cm edit --color-mode=dark testfile.cm` on a dark terminal
   - `cm edit --color-mode=light testfile.cm` on a light terminal
   - Verify: source pane, preview pane, status bar, globals panel, overlays (help, export, command menu, file picker), autocomplete popup, selection highlighting, syntax highlighting, context footer

4. **Run quality checks**: `task quality`

**Success criteria:**
- `task test` passes with zero failures
- `task quality` passes
- Both dark and light modes produce readable, consistent UI
- No terminal bleed-through in either mode

## Alternative Approaches Considered

1. **Dual-palette ThemeConfig** — Two complete sets of hex defaults (dark and light) in ThemeConfig. Simpler migration but doubles config surface and doesn't leverage lipgloss's built-in AdaptiveColor mechanism. Rejected because it fights the framework.

2. **Single monolithic Theme struct** — Replace all component style structs with one shared Theme. Reduces indirection but couples components to the theme structure, hurting testability and making it harder to test individual components in isolation. Rejected for maintainability.

3. **Terminal auto-detection** — Query terminal background color to auto-detect light/dark. Rejected because it caused escape sequence artifacts before alternate screen in earlier testing. Explicit config is more reliable.

## Acceptance Criteria

### Functional Requirements

- [x] TUI renders correctly on dark terminals with `color_mode = "dark"` (default)
- [x] TUI renders correctly on light terminals with `color_mode = "light"`
- [x] Zero hardcoded `lipgloss.Color(...)` literals in TUI source code (editor + REPL + components)
- [x] All colors resolve from `config/theme/palette.go` AdaptiveColor values
- [x] Component `Default*Style()` functions source from palette
- [x] Overlay rendering uses shared `OverlayStyle` and `RenderOverlayBox()`
- [x] Source pane lines show block-level syntax highlighting (frontmatter/markdown/calc)
- [x] Selection highlighting is readable in both light and dark modes
- [ ] Glamour markdown preview has light/dark style variants
- [x] REPL uses palette colors instead of ANSI 256-color codes

### Non-Functional Requirements

- [x] No measurable performance regression (AdaptiveColor resolution is O(1))
- [x] `task test` and `task quality` pass (3 pre-existing failures unrelated to refactor)
- [x] Backwards compatible: deprecated config keys produce warnings, not errors

### Quality Gates

- [x] Catwalk golden files regenerated and reviewed
- [ ] Manual smoke test on dark terminal (requires interactive terminal)
- [ ] Manual smoke test on light terminal (requires interactive terminal)
- [x] No `lipgloss.Color(` grep matches in `cmd/calcmark/tui/` (excluding test files)

## Dependencies & Prerequisites

- lipgloss `AdaptiveColor` — already available in current dependency version
- `lipgloss.SetHasDarkBackground()` — already called in `applyColorMode()`
- Existing `sourcePaneBg()` / `previewPaneBg()` helpers — ready to be wired
- Existing `palette.go` with 15 AdaptiveColor values — foundation to expand
- Existing `document.Detector` for line classification — needed for syntax highlighting

## Risk Analysis & Mitigation

| Risk | Impact | Mitigation |
|---|---|---|
| Subtle pane contrast (2-3%) invisible on 256-color terminals | Medium | Both hex values may map to same ANSI 234. Accept this; 256-color users get consistent (not contrasted) panes. |
| Users lose custom colors from removed ThemeConfig fields | Low | Log deprecation warnings. Old theme was partially broken anyway. |
| Glamour's `ansi.StyleConfig` doesn't support AdaptiveColor | Medium | Create dual style configs (`createLightStyle()`/`createDarkStyle()`) selected at renderer construction time. |
| `overlayPadLine()` type incompatibility with AdaptiveColor | Low | Fix early in Phase 3 — one-line signature change. |
| File picker (bubbles) may not accept AdaptiveColor | Low | Use conditional palette lookup via `config.IsDarkMode()` at construction time. |
| Catwalk tests break from structural changes | Low | Tests strip ANSI via `termenv.Ascii`. Color changes are invisible. Only structural changes could break tests. Regenerate golden files in Phase 7. |

## Institutional Learnings

From `docs/solutions/`:
- **Always use `GetHorizontalPadding()`** when calculating content widths — important during overlay extraction
- **Test at boundary widths** — overlays and panes should be tested at minimum widths
- **Catwalk TDD pattern** — every user-facing bug must have a catwalk test

## WASM Considerations

WASM is **out of scope** for this refactor. The only WASM file is `spec/document/markdown_wasm.go` (spec layer, not TUI). No WASM build of the TUI exists. No Taskfile target builds TUI for WASM. If WASM TUI support is added later, the palette system naturally supports it via `lipgloss.SetHasDarkBackground()`.

## References & Research

### Internal References
- Brainstorm: `docs/brainstorms/2026-02-22-tui-theme-consistency-brainstorm.md`
- Palette foundation: `cmd/calcmark/config/theme/palette.go`
- Style building: `cmd/calcmark/config/theme.go:60-206`
- Config loading: `cmd/calcmark/config/config.go`
- Source pane bg helpers: `cmd/calcmark/tui/editor/view_panes.go:138-153`
- Overlay duplication: `cmd/calcmark/tui/editor/view_overlays.go`, `help_overlay.go`, `export_overlay.go`
- Catwalk test init: `cmd/calcmark/tui/editor/catwalk_test.go` (init sets `termenv.Ascii`)
- Learnings: `docs/solutions/` (width calculations, boundary testing, TDD)

### Key Patterns
- `lipgloss.AdaptiveColor{Light: "#xxx", Dark: "#yyy"}` resolves at `Render()` time
- `lipgloss.SetHasDarkBackground(bool)` controls global light/dark resolution
- `lipgloss.TerminalColor` interface — accepted by `AdaptiveColor`, `Color`, `NoColor`
- Component `Default*Style()` pattern — pure constructors, testable, palette-sourced
