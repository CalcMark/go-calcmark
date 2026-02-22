# TUI Theme Consistency

**Date:** 2026-02-22
**Status:** Brainstorm (reviewed)

## What We're Building

A consistent, complete theming system for the CalcMark TUI that eliminates terminal-level color bleed-through and provides good default dark and light themes. The core problems today:

1. **Dark theme on light terminal is broken** — hardcoded dark colors render as unreadable text (dark-on-dark, white rectangles bleeding through) when the user has a light terminal background.
2. **~70+ hardcoded `lipgloss.Color(...)` calls** scattered across view_panes.go, view_overlays.go, view.go, repl/view.go, suggest.go, globals.go, statusbar.go, filepicker.go, export_overlay.go, and help_overlay.go that bypass the existing theme system entirely.
3. **Two parallel theme systems** that aren't connected: `ThemeConfig` (hex strings in defaults.toml) and `config/theme/palette.go` (AdaptiveColor pairs, unused).

## Why This Approach

**AdaptiveColor throughout, with semantic palette in code:**

- `lipgloss.AdaptiveColor` is the idiomatic bubbletea/lipgloss mechanism for light/dark support. Each color has `{Light: "#xxx", Dark: "#yyy"}` and lipgloss selects the right one at render time based on `HasDarkBackground()`.
- A semantic palette of ~15-20 named AdaptiveColor values in `palette.go` (e.g., `PaneBg`, `PopupBg`, `TextPrimary`, `BorderDim`) provides a single source of truth. Components reference these by purpose, not appearance.
- Component-level style structs (`PopupStyle`, `StatusBarStyle`, `GlobalsPanelStyle`, `AutosuggestStyle`) **stay as-is** for testability and pure rendering, but their `Default*Style()` functions are rewritten to source colors from the palette.
- The user-facing `[tui.theme]` config section is slimmed to ~10-15 high-level overrides (primary, accent, error, etc.). Internal structural colors (popup border shades, separator grays) are not user-configurable — they derive from the palette.

**Alternatives considered:**
- *Dual-palette ThemeConfig* (two sets of hex defaults): simpler migration but doesn't leverage lipgloss's built-in adaptive mechanism, and doubles the config surface.
- *Single shared Theme struct* replacing all component styles: reduces indirection but couples components to the theme structure, hurting testability.

## Key Decisions

1. **AdaptiveColor as the color primitive** — Replace all `lipgloss.Color("...")` literals with references to `lipgloss.AdaptiveColor` values from `palette.go`. This includes converting ANSI 256-color codes (e.g., `"236"`, `"252"`) in the REPL to hex-based AdaptiveColor values.

2. **Explicit color_mode, no auto-detection** — Default to dark. Users set `color_mode = "light"` in config if they have a light terminal. No terminal background query (which caused escape-sequence artifacts before alternate screen). The existing `color_mode = "auto"` is preserved as a deprecated alias for `"dark"` with a logged warning, then removed in a future release.

3. **Semantic palette in code, minimal user config** — ~15-20 AdaptiveColor values defined in `config/theme/palette.go`. Only high-level "brand" colors (primary, accent, error, warning, output) exposed in `[tui.theme]` config.toml. Everything else is internal.

4. **Keep component style structs** — `PopupStyle`, `StatusBarStyle`, `GlobalsPanelStyle`, `AutosuggestStyle` remain for clean component APIs and testability. Their `Default*()` constructors are rewritten to pull from the palette. The override-after-construction pattern in `view.go` (where `DefaultStatusBarStyle()` is modified inline with theme backgrounds) is eliminated — `Default*()` functions directly produce the correct themed styles.

5. **Subtle pane contrast** — Both source and preview panes have similar backgrounds, with preview very slightly lighter/darker (2-3%) for visual separation. This means `ensureFullWidth()` and `padToWidth()` calls must use context-dependent palette values (`SourcePaneBg` vs `PreviewPaneBg`) rather than a single hardcoded `"236"`.

6. **Block-level syntax highlighting in source pane** — Three distinct colors for frontmatter lines (YAML between `---`), markdown lines, and calculation lines. The document parser already classifies lines by type. No token-level parsing. Three new AdaptiveColor values: `SourceFrontmatter`, `SourceMarkdown`, `SourceCalc`. **Note:** This is a feature addition that should ideally be a separate phase from the theme refactor to reduce risk, but may be implemented together since it touches the same rendering code.

7. **Glamour (markdown preview) stays separate** — The `createMinimalStyle()` in `markdown.go` keeps its own color system. However, if the palette changes the preview pane background, glamour's base background should be updated to match to avoid clashing. This is a targeted change to `createMinimalStyle()`, not a full integration.

8. **ThemeConfig struct simplification** — The current ~40-field `ThemeConfig` struct is reduced to ~10-15 user-facing fields (primary, accent, error, warning, output, muted, dimmed, bright, separator). User overrides apply to the light or dark slot of an AdaptiveColor based on the configured `color_mode`. Removed keys (e.g., `edit_line_bg`, `cursor_bg`, `calc_value`) are detected at load time and logged as deprecation warnings. No automatic mapping from old to new — users must update their config.

9. **Remove dead `config/theme/styles.go` package-level styles** — The pre-built styles in `theme/styles.go` (`TextStyle`, `MutedStyle`, `HeaderStyle`, etc.) are unused and redundant with the component `Default*Style()` pattern. Remove them to avoid confusion about which layer owns styling.

10. **Extract shared overlay style** — The four overlays (help, export, command menu, file picker) share duplicated border-drawing code with a `pad()` closure. Extract a shared `OverlayStyle` struct with a `DefaultOverlayStyle()` that sources from the palette, and a `RenderOverlayBox()` helper. This eliminates ~4 copies of the same rendering pattern.

## Scope / Affected Files

**Core palette (new/modified):**
- `cmd/calcmark/config/theme/palette.go` — Expand from ~12 to ~20+ AdaptiveColor values
- `cmd/calcmark/config/theme/styles.go` — **Remove** (dead code)
- `cmd/calcmark/config/theme.go` — Rewrite `BuildStyles()` to use AdaptiveColor from palette
- `cmd/calcmark/config/types.go` — Simplify `ThemeConfig` to ~10-15 fields
- `cmd/calcmark/config/defaults.toml` — Update to match simplified config
- `cmd/calcmark/config/config.go` — Update `applyColorMode()`, deprecate "auto", add migration warnings

**Components (replace hardcoded colors):**
- `cmd/calcmark/tui/components/suggest.go` — `DefaultPopupStyle()`, `DefaultAutosuggestStyle()`
- `cmd/calcmark/tui/components/statusbar.go` — `DefaultStatusBarStyle()`
- `cmd/calcmark/tui/components/globals.go` — `DefaultGlobalsPanelStyle()`
- `cmd/calcmark/tui/components/errors.go` — any hardcoded colors
- `cmd/calcmark/tui/components/pinned.go` — any hardcoded colors
- `cmd/calcmark/tui/components/contextfooter.go` — any hardcoded colors
- **New:** `cmd/calcmark/tui/components/overlay.go` — Shared `OverlayStyle` struct and `RenderOverlayBox()`

**Source pane syntax highlighting (new behavior):**
- `cmd/calcmark/tui/editor/view_panes.go` — Render source lines with block-type-aware colors
- `cmd/calcmark/tui/editor/view_lines.go` — Apply frontmatter/markdown/calc coloring
- Uses existing `document.Detector` or block classification already available in the model

**Editor views (biggest change — ~50+ hardcoded colors):**
- `cmd/calcmark/tui/editor/view.go` — Remove override-after-construction pattern
- `cmd/calcmark/tui/editor/view_panes.go` — Replace `lipgloss.Color("236")` with palette values
- `cmd/calcmark/tui/editor/view_overlays.go` — Refactor to use shared `OverlayStyle`
- `cmd/calcmark/tui/editor/view_lines.go`
- `cmd/calcmark/tui/editor/view_footer.go`
- `cmd/calcmark/tui/editor/view_util.go`
- `cmd/calcmark/tui/editor/help_overlay.go` — Refactor to use shared `OverlayStyle`
- `cmd/calcmark/tui/editor/export_overlay.go` — Refactor to use shared `OverlayStyle`
- `cmd/calcmark/tui/editor/filepicker.go`
- `cmd/calcmark/tui/editor/sidebyside.go`
- `cmd/calcmark/tui/editor/markdown.go` — Update `createMinimalStyle()` background to match palette

**REPL views:**
- `cmd/calcmark/tui/repl/view.go` — Convert ~15 ANSI 256-color codes to palette references. The REPL currently inlines all styles (no component structs); it should use the existing component style structs where applicable (e.g., `DefaultStatusBarStyle()` for its status rendering).

## Risks / Constraints

- **Backwards compatibility**: Users with existing `[tui.theme]` config.toml will see deprecation warnings for removed keys. Their customizations for removed fields (e.g., `cursor_bg`) will be silently ignored. This is acceptable since the old theme system was partially broken anyway.
- **Test updates**: Catwalk tests compare rendered output. Color changes will require regenerating golden test expectations. Verify whether `task test` has a golden-update flag. Plan a dedicated test-regeneration step.
- **WASM target**: In WASM, `lipgloss.HasDarkBackground()` has no terminal to query. The WASM build must call `SetHasDarkBackground()` explicitly. The JavaScript host should communicate theme preference via a WASM export/import. This needs a build-tag guard or WASM-specific initialization path. **Concrete decision needed during planning.**
- **AdaptiveColor lazy resolution**: The `BuildStyles()` singleton pattern in `config.go` (called once via `sync.Once`) is compatible with AdaptiveColor because lipgloss resolves Light/Dark at `Render()` time, not at `NewStyle()` time. **Verify this assumption early in implementation.**

## Resolved Questions

1. **Approach**: Use AdaptiveColor throughout (not dual-palette ThemeConfig).
2. **Config scope**: Semantic palette in code; only ~10-15 high-level keys in user config.
3. **Component styles**: Keep style structs, wire their defaults from the palette.
4. **Detection**: Require explicit color_mode config; default to dark; no auto-detection.
5. **Glamour**: Stays separate but gets background-only update for consistency.
6. **Pane contrast**: Subtle (2-3% difference) rather than stark contrast.
7. **Syntax highlighting**: Block-level only (frontmatter/markdown/calc).
8. **Auto deprecation**: `color_mode = "auto"` becomes deprecated alias for dark.
9. **Dead code**: `config/theme/styles.go` package-level vars removed.
10. **Overlay duplication**: Extract shared OverlayStyle struct.
