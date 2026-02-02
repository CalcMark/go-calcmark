# Feature Research

**Domain:** CLI/TUI developer tool -- calculation notepad language with REPL and editor
**Researched:** 2026-02-02
**Confidence:** MEDIUM-HIGH (based on analysis of comparable tools Numi/Soulver/Calca, established CLI guidelines from clig.dev and Heroku, Charm ecosystem documentation, and current project state)

## Current State Assessment

Before listing features, here is what CalcMark already has:

| Area | Status | Notes |
|------|--------|-------|
| CLI entry point (cobra) | Working | `cm`, `cm eval`, `cm edit`, `cm convert`, `cm version` |
| REPL mode | Working | Interactive line-by-line evaluation |
| TUI editor (two-pane) | In progress | ModelV2 with textarea, side-by-side source+results |
| File save (Ctrl+S) | Working | Basic save to disk |
| Config system (TOML) | Working | Themes, color mode, formatter options |
| Theming (dark/light) | Working | Configurable hex colors, dark mode detection |
| Status bar component | Defined | Components exist but not fully wired into ModelV2 |
| Autosuggest component | Defined | `SuggestionSource` interface exists, not wired in |
| Shell completions | Disabled | `rootCmd.CompletionOptions.DisableDefaultCmd = true` |
| Release process | Working | `release.sh`, GitHub Actions, WASM artifacts |
| Prebuilt binaries | Partial | `task build:all` exists, no GoReleaser, no Homebrew |
| Man pages | Missing | No man page generation |
| Documentation | Basic | README exists but is library-focused, not user-focused |

## Feature Landscape

### Table Stakes (Users Expect These)

Features users assume exist. Missing these = product feels incomplete or untrustworthy for v1.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **`--help` on every command** | clig.dev consensus: users expect `cmd --help` and `cmd subcmd --help` everywhere. Cobra generates this automatically but text quality matters. | LOW | Cobra auto-generates. Polish the descriptions, add examples to each command's Long text. Currently bare-minimum. |
| **`--version` flag** | Users need to report bugs and verify installations. Must include build info. | LOW | Already exists via `cm version`. Add `--version` flag on root command for convention compliance. |
| **Meaningful error messages** | CLI guidelines (Heroku, clig.dev): errors must explain what went wrong AND suggest what to do. Send errors to stderr. | MEDIUM | Already uses stderr for TUI. Audit all error paths in `cmd/` for actionable messages. |
| **`--no-color` / `NO_COLOR` support** | Universal convention (no-color.org). Users in CI, piped contexts, or with accessibility needs expect this. | LOW | Color detection exists. Add explicit `--no-color` flag and `NO_COLOR` env var check. |
| **Non-zero exit codes on failure** | Scripts and CI depend on exit codes. 0 = success, non-zero = failure. | LOW | Verify all `cmd/` error paths call `os.Exit(1)`. |
| **Shell completions (bash/zsh/fish)** | Expected for any CLI with subcommands. Cobra provides this for free. | LOW | Currently *disabled* (`DisableDefaultCmd = true`). Re-enable and add `cm completion bash/zsh/fish` subcommand. Use portable `ValidArgsFunction` API. |
| **File argument completion** | When typing `cm budget.cm`, shell should complete `.cm` filenames. | LOW | Implement with Cobra's `ValidArgsFunction` + `ShellCompDirectiveFilterFileExt` for `.cm` files. |
| **Ctrl+C graceful exit** | TUI must not leave terminal in bad state. No leaked goroutines, restore terminal settings. | LOW | Already handled via bubbletea alternate screen. Verify cleanup on all exit paths. |
| **Window resize handling** | TUI must reflow on terminal resize without artifacts. | MEDIUM | `tea.WindowSizeMsg` is handled in ModelV2. Need to verify no rendering artifacts on resize (Gemini CLI team found this critical). |
| **Save confirmation on quit** | Editor must warn before losing unsaved changes. | LOW | `modified` flag exists. Add "Unsaved changes. Quit? (y/n)" prompt on Ctrl+Q/Ctrl+C when modified=true. |
| **Status bar with position info** | Users expect to see cursor line/col, file name, modified indicator. Standard in all editors. | LOW | `StatusBarState` and `RenderStatusBar` exist. Wire into ModelV2. |
| **Keyboard shortcut reference** | Users need to discover available keybindings. At minimum, show in status bar footer. | LOW | Use `bubbles/help` component with `key.Binding` definitions. Toggle with `?` key. Already in the Charm ecosystem. |
| **Stdin pipe support** | `echo "5 + 3" \| cm eval` must work. Essential for composability. | LOW | Likely works via cobra. Verify and test. |
| **Prebuilt binaries for major platforms** | Users expect downloadable binaries (Linux amd64/arm64, macOS amd64/arm64, Windows amd64). | MEDIUM | `task build:all` exists. Switch to GoReleaser for automated cross-platform builds with checksums. |

### Differentiators (Competitive Advantage)

Features that set CalcMark apart from Numi, Soulver, Calca, and plain calculators. Not required, but make the tool loved vs merely used.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Live side-by-side results** | CalcMark's core differentiator: type on left, see results update on right in real-time. Soulver/Numi do this in GUI; CalcMark does it in the terminal. No comparable TUI tool exists. | HIGH (in progress) | ModelV2 implements this. Polish is the work: alignment, scrolling, wrapping all need to be solid. |
| **Markdown blending** | Write prose (markdown) interleaved with calculations. Documents that explain AND compute. No competitor does this in a terminal. | HIGH (partially done) | Classifier distinguishes calc vs markdown lines. Rendering markdown sections in the preview pane would be a strong differentiator. |
| **YAML front matter for constants** | Define document-level constants (tax rates, dates, exchange rates) in front matter. Reusable across the document. Like Hugo/Jekyll front matter but for calculations. | MEDIUM | YAML parsing via `gopkg.in/yaml.v3` already in go.mod. Need to parse `---` delimiters, inject variables into evaluator context before document evaluation. |
| **In-TUI autocomplete for variables/functions** | As user types, suggest defined variables, built-in functions (`avg`, `sqrt`), units, and constants (`PI`, `E`). IDE-like experience in the terminal. | HIGH | `SuggestionSource` interface and `AutosuggestState` already defined. Need: trigger detection (typing prefix), popup positioning, Tab acceptance, integration with evaluator context for variable names. |
| **Homebrew tap distribution** | `brew install calcmark/tap/cm` -- frictionless install for macOS/Linux users. | LOW | Standard GoReleaser + homebrew tap pattern. Create `homebrew-tap` repo, configure `.goreleaser.yml`. Well-documented workflow. |
| **Scoop bucket (Windows)** | `scoop install cm` -- frictionless Windows install. | LOW | GoReleaser supports Scoop out of the box alongside Homebrew. |
| **`cm eval --json` structured output** | Machine-readable output for scripting and CI integration. Output variable names, values, types, errors as JSON. | MEDIUM | Aligns with clig.dev recommendation for structured output. Enables piping to `jq`, integration with other tools. |
| **In-TUI contextual help overlay** | Press `?` to see full keybinding reference overlaid on the editor. Not just a footer -- a proper help screen. | MEDIUM | `bubbletea-overlay` package exists for this pattern. Or implement as a simple state toggle that replaces the view. |
| **Man page generation** | `man cm` should work after install. Professional-grade documentation. | LOW | Cobra can generate man pages via `doc.GenManTree()`. GoReleaser can bundle them with Homebrew installs via `extra_install`. |
| **Rich error display in results pane** | When a calc line has an error (undefined variable, division by zero), show the error inline in the results pane with color coding. Not just silence. | MEDIUM | Evaluator already produces diagnostics with severity levels. Render errors in results pane with appropriate styling (red for errors, yellow for warnings). |
| **Document export (HTML, JSON, Markdown)** | `cm convert doc.cm --to=html` already exists as a command. Polish output quality. | MEDIUM | `convert.go` command exists. Ensure HTML output is self-contained, JSON includes all metadata, Markdown round-trips cleanly. |
| **Example files shipped with binary** | Include curated `.cm` example files that users can explore: budget template, unit conversion cheatsheet, date math examples. | LOW | `testdata/*.cm` and `docs/examples/` exist. Curate a user-facing set. Add `cm examples` subcommand or document location. |

### Anti-Features (Commonly Requested, Often Problematic)

Features that seem good but create complexity without proportional value for a v1 release.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| **Vim/Emacs keybinding modes** | Power users want familiar modal editing. | Massive implementation scope. bubbletea textarea is non-modal. Conflicts with CalcMark's keybinding needs (Ctrl+S save, Ctrl+Q quit). Testing matrix explodes. | Stick with standard textarea keybindings (arrows, Home/End, Ctrl+A/E). Document keybindings clearly. Consider post-v1 if heavily requested. |
| **Plugin/extension system** | Users want custom functions, units, or output formats. | Plugin systems are an order-of-magnitude complexity increase. API stability requirements multiply. Security surface expands. | Support YAML front matter for user constants. Keep the built-in function set tight. Consider post-v2. |
| **Real-time currency exchange rates** | Numi does live currency conversion. Users will ask for it. | Network dependency in a calculation tool is an anti-pattern for reproducibility. CalcMark's core value is "verifiable and reproducible" calculations. Live rates break this. | Support currency symbols as unit types (already done). Document that exchange rates are user-defined constants in front matter, not live-fetched. This is a feature, not a limitation -- it ensures reproducibility. |
| **Syntax highlighting in source pane** | Editors highlight syntax. Users expect color-coded variables, operators, numbers. | Significant rendering complexity. bubbletea textarea does not natively support multi-token syntax highlighting. Would require custom rendering layer on top of textarea. | Use cursor-line highlighting (already done) and bold/color for results pane. Consider post-v1 when textarea integration is stable. |
| **Undo/redo history** | Standard editor feature. Users expect Ctrl+Z. | bubbletea textarea has limited undo support. Implementing proper undo/redo with document sync is complex. Must handle both textarea state and evaluator state rollback. | bubbletea textarea provides basic single-level undo. Document this limitation. Full undo/redo is post-v1 scope. |
| **Multi-file/tab support** | Users want to open multiple .cm files simultaneously. | Significant UI complexity: tab bar, tab switching, per-tab state management, multi-document evaluator contexts. | v1 supports one file at a time. `cm file1.cm` opens one file. To switch files, quit and reopen. Simple, predictable, testable. |
| **Configuration GUI / setup wizard** | First-time users might want guided setup. | Massive scope for minimal value. Config file + documentation is the standard for CLI tools. | Ship sensible defaults that work out of the box. Document `~/.config/calcmark/config.toml` clearly. Add `cm config --edit` to open config in $EDITOR if desired. |
| **LSP (Language Server Protocol)** | IDE integration for VS Code, Neovim, etc. | Full LSP implementation is a project unto itself. Completions, diagnostics, hover, go-to-definition all need implementation. | The WASM build and syntax highlighter spec already enable basic editor integration. Full LSP is a separate milestone, not a v1 feature. |
| **Collaborative/shared editing** | Multi-user editing of the same document. | Wrong product category entirely. CalcMark is a local-first calculation tool, not a collaboration platform. | Not applicable. Don't build this. |

## Feature Dependencies

```
[Shell Completions]
    (no deps, re-enable Cobra built-in)

[Status Bar] ──requires──> [Keybinding Definitions (key.Binding)]
    [Help Overlay] ──requires──> [Keybinding Definitions]
                                     └──requires──> [bubbles/help integration]

[In-TUI Autocomplete] ──requires──> [Evaluator Context access]
                           └──requires──> [Trigger detection in textarea]
                           └──requires──> [Popup rendering above cursor]

[YAML Front Matter] ──requires──> [Document parser changes]
                         └──requires──> [Evaluator context pre-seeding]

[Prebuilt Binaries + Homebrew] ──requires──> [GoReleaser setup]
    [Man Pages] ──enhances──> [Prebuilt Binaries + Homebrew]
    [Shell Completion files] ──enhances──> [Prebuilt Binaries + Homebrew]

[Rich Error Display] ──requires──> [Results pane rendering logic]
                          └──enhances──> [Live side-by-side results]

[JSON Output] ──requires──> [eval command output formatting]
                  └──enhances──> [Document export]

[Save Confirmation] ──requires──> [Modified state tracking (exists)]

[Markdown Blending in Preview] ──requires──> [Classifier (exists)]
                                    └──requires──> [glamour or custom markdown rendering]
                                    └──enhances──> [Live side-by-side results]
```

### Dependency Notes

- **Keybinding definitions are foundational**: Both the status bar hints and the help overlay depend on having a proper `keyMap` struct with `key.Binding` definitions. Do this first, then both status bar and help overlay become easy.
- **GoReleaser is the distribution gateway**: Homebrew, Scoop, man pages, shell completion files, and checksums all flow from GoReleaser configuration. Set it up once, get all distribution channels.
- **Autocomplete is the hardest differentiator**: It requires coordination between the textarea (cursor position, current word extraction), the evaluator (variable names), the language spec (function names, units), and the rendering layer (popup positioning). Plan for iterative delivery: variable names first, then functions, then units.
- **YAML front matter requires parser work**: The document parser needs to recognize `---` delimiters and extract YAML. This touches the spec/impl boundary, so it needs careful design to maintain the one-way dependency rule.

## MVP Definition

### Launch With (v1.0)

Minimum viable product -- what's needed to call this a real, shippable tool.

- [ ] **Polished TUI editor with two-pane layout** -- Source on left, live results on right. Scrolling works. Resize works. No rendering artifacts.
- [ ] **Complete CLI help system** -- Every subcommand has `--help` with examples. `--version` on root. Actionable error messages on all paths.
- [ ] **Shell completions (bash/zsh/fish)** -- `cm completion bash/zsh/fish` subcommand. Completions for subcommands and `.cm` file arguments.
- [ ] **Status bar with keybinding hints** -- File name, line position, modified indicator. Short help footer showing key shortcuts.
- [ ] **Help overlay (toggle with ?)** -- Full keybinding reference accessible without leaving the editor.
- [ ] **Save confirmation on quit** -- Prevent accidental data loss.
- [ ] **Prebuilt binaries via GoReleaser** -- Linux (amd64/arm64), macOS (amd64/arm64), Windows (amd64). Checksums. GitHub Release.
- [ ] **Homebrew tap** -- `brew install calcmark/tap/cm` for macOS/Linux users.
- [ ] **NO_COLOR / --no-color support** -- Respect the convention.
- [ ] **Clean README for end users** -- Installation, quick start, examples. Separate from library README.
- [ ] **Curated example files** -- 3-5 example `.cm` files demonstrating key features.

### Add After Validation (v1.x)

Features to add once core is working and users provide feedback.

- [ ] **YAML front matter** -- Trigger: users request document-level constants or templates.
- [ ] **In-TUI autocomplete** -- Trigger: users report friction typing variable names and function calls. Start with variable names, then expand to functions and units.
- [ ] **Rich error display in results pane** -- Trigger: users report confusion about why a line shows no result.
- [ ] **`cm eval --json` structured output** -- Trigger: users want to integrate CalcMark into scripts or CI.
- [ ] **Man pages** -- Trigger: distribution via Homebrew is working and users expect `man cm`.
- [ ] **Scoop bucket (Windows)** -- Trigger: Windows user requests.
- [ ] **Markdown rendering in preview pane** -- Trigger: users with markdown-heavy documents want rendered preview.
- [ ] **Document export polish** -- Trigger: users use `cm convert` and request better HTML/JSON output.

### Future Consideration (v2+)

Features to defer until product-market fit is established.

- [ ] **Syntax highlighting in source pane** -- Requires custom textarea rendering. Wait for bubbletea ecosystem maturity.
- [ ] **Full undo/redo** -- Requires undo stack coordinated with evaluator. Wait for user demand.
- [ ] **LSP server** -- Separate project scope entirely. Wait for editor integration demand.
- [ ] **Plugin system** -- API stability required. Wait for stable v1 API.

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Polished two-pane editor | HIGH | HIGH (in progress) | P1 |
| Shell completions | MEDIUM | LOW | P1 |
| Help system (CLI + TUI) | HIGH | LOW | P1 |
| Status bar integration | MEDIUM | LOW | P1 |
| Save confirmation | HIGH | LOW | P1 |
| NO_COLOR support | MEDIUM | LOW | P1 |
| GoReleaser + prebuilt binaries | HIGH | MEDIUM | P1 |
| Homebrew tap | HIGH | LOW (after GoReleaser) | P1 |
| User-facing README + examples | HIGH | MEDIUM | P1 |
| Help overlay (? key) | MEDIUM | MEDIUM | P1 |
| YAML front matter | HIGH | MEDIUM | P2 |
| In-TUI autocomplete | HIGH | HIGH | P2 |
| Rich error display | MEDIUM | MEDIUM | P2 |
| JSON structured output | MEDIUM | MEDIUM | P2 |
| Man pages | LOW | LOW | P2 |
| Scoop bucket | LOW | LOW | P2 |
| Markdown preview in results | MEDIUM | HIGH | P3 |
| Syntax highlighting | MEDIUM | HIGH | P3 |
| Full undo/redo | MEDIUM | HIGH | P3 |
| LSP server | HIGH | HIGH | P3 |

**Priority key:**
- P1: Must have for v1.0 launch
- P2: Should have, add in v1.x when possible
- P3: Nice to have, future consideration

## Competitor Feature Analysis

| Feature | Numi (macOS/Win/Linux) | Soulver (Apple ecosystem) | Calca (macOS/iOS/Win) | CalcMark (CLI/TUI) |
|---------|----------------------|--------------------------|----------------------|-------------------|
| **Platform** | Desktop GUI | GUI (Mac/iPad/iPhone) | GUI (Mac/iOS/Win) | Terminal (any platform) |
| **Real-time results** | Side panel | Right-aligned | Inline (`=>`) | Two-pane (source + results) |
| **Markdown support** | No | No | Yes (indentation-based) | Yes (CommonMark blending) |
| **Variables** | Yes | Yes | Yes (symbolic) | Yes |
| **Unit conversions** | Yes (live rates) | Yes | Yes | Yes (canonical units) |
| **Currency** | Yes (live rates) | Yes | Yes | Yes (symbols, user-defined rates) |
| **Functions** | JS plugins | Limited | Symbolic math | Built-in (avg, sqrt, etc.) |
| **Document save/load** | No (clipboard) | Yes (files/folders) | Yes | Yes (.cm files) |
| **Reproducibility** | No (live rates change) | No (live rates) | Partial | Yes (core value) |
| **Scriptable** | No | No | No | Yes (pipe, eval, convert) |
| **WASM/browser** | No | No | No | Yes (first-class) |
| **Price** | Free/Setapp | Paid | Paid | Free/open source |
| **Terminal native** | No | No | No | **Yes (unique)** |

CalcMark's unique positioning: the only calculation notepad that is terminal-native, scriptable, reproducible, and open source. This is not a weakness -- it is a deliberate niche serving developers, engineers, and CLI-native users who live in the terminal.

## Sources

### Authoritative (HIGH confidence)
- [Command Line Interface Guidelines (clig.dev)](https://clig.dev/) -- Comprehensive CLI design guide
- [Heroku CLI Style Guide](https://devcenter.heroku.com/articles/cli-style-guide) -- Industry-standard CLI conventions
- [Cobra Shell Completion Docs](https://cobra.dev/docs/how-to-guides/shell-completion/) -- Portable completion API
- [GoReleaser Official Docs](https://goreleaser.com/) -- Release automation for Go
- [bubbles/help package (pkg.go.dev)](https://pkg.go.dev/github.com/charmbracelet/bubbles/help) -- Help component API
- [bubbles/key package (pkg.go.dev)](https://pkg.go.dev/github.com/charmbracelet/bubbles/key) -- Keybinding management
- [BetterCLI.org Help Pages](https://bettercli.org/design/cli-help-page/) -- Help page design patterns

### Verified (MEDIUM confidence)
- [Atlassian: 10 Design Principles for Delightful CLIs](https://www.atlassian.com/blog/it-teams/10-design-principles-for-delightful-clis) -- CLI design principles
- [GoReleaser + GitHub Actions Guide](https://gitgist.com/posts/goreleaser-and-github-actions/) -- Distribution workflow
- [GoReleaser Homebrew Guide (dev.to)](https://dev.to/hadlow/how-to-release-to-homebrew-with-goreleaser-github-actions-and-semantic-release-2gbb) -- Homebrew tap setup
- [Shipping Go CLI Completions with GoReleaser](https://carlosbecker.com/posts/golang-completions-cobra/) -- Completion file distribution
- [GoReleaser Man Page Distribution (appliedgo.net)](https://appliedgo.net/spotlight/install-with-manpage/) -- Man page bundling
- [Soulver App](https://soulver.app/) -- Competitor analysis
- [Numi App](https://numi.app/) -- Competitor analysis
- [X-CMD CLI/TUI Design](https://www.x-cmd.com/start/cli-tui-llm/) -- TUI design philosophy
- [bubbletea-overlay (libraries.io)](https://libraries.io/go/github.com%2Frmhubbert%2Fbubbletea-overlay) -- Overlay pattern for bubbletea

### Unverified (LOW confidence)
- [Awesome TUIs list (GitHub)](https://github.com/rothgar/awesome-tuis) -- TUI ecosystem survey
- [Autocomplete UX best practices (smart-interface-design-patterns.com)](https://smart-interface-design-patterns.com/articles/autocomplete-ux/) -- Autocomplete UX patterns (GUI-focused)

---
*Feature research for: CalcMark CLI/TUI developer tool v1 release*
*Researched: 2026-02-02*
