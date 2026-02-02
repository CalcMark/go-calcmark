# Project Research Summary

**Project:** CalcMark v1 Release Polish
**Domain:** CLI/TUI developer tool for calculation notepad with live preview
**Researched:** 2026-02-02
**Confidence:** HIGH

## Executive Summary

CalcMark is a terminal-native calculation notepad that blends CommonMark markdown with live calculation evaluation, built as a Go CLI/TUI application. The project has a working interpreter, REPL, and two-pane editor but needs v1 release polish across six key areas: TUI editor rewrite (from custom Model to textarea-based ModelV2), help system, autocomplete, YAML front matter, documentation, and prebuilt binary distribution.

The recommended approach is to ship v1 on the proven Charm ecosystem v1 stack (bubbletea v1.3.10, bubbles v0.21.0, lipgloss v1.x) rather than migrating to v2 RCs. The editor rewrite must converge on ModelV2 with textarea as the source-of-truth for editing, while preserving CalcMark's unique two-pane alignment architecture through extracted pure geometry functions. Distribution should adopt GoReleaser for cross-platform builds and Homebrew/Scoop packaging. The existing catwalk test infrastructure is the project's strength and must expand to cover all ModelV2 behavior.

The critical risk is two-pane alignment desynchronization under text wrapping - this is the hardest technical problem in the codebase and must be solved through rigorous pure-function geometry computation with comprehensive catwalk testing before any other features build on top of it. Secondary risks include: shipping ModelV2 with feature regression from Model, WASM binary size explosion, and CI/release workflow Go version mismatches.

## Key Findings

### Recommended Stack

CalcMark's existing stack is validated and current. The only additions needed are adrg/frontmatter for YAML front matter parsing and GoReleaser v2.13.3 for release automation. The project should remain on Charm ecosystem v1 for the v1 release - bubbletea v2, bubbles v2, and lipgloss v2 are in RC but not stable. Migration to Charm v2 should be planned as a post-v1 milestone.

**Core technologies:**
- **Go 1.24.12** (update from 1.24.4): Current stable with security patches. Go 1.25 exists but 1.24 with 12 patch releases is safer for v1.
- **bubbletea v1.3.10**: Industry standard TUI framework with Elm architecture. Stay on v1 - v2 is RC2 with breaking changes to View(), imports, and mouse API.
- **bubbles v0.21.0**: Provides textarea component for multi-line editing with line numbers, cursor, scrolling. This is the foundation for ModelV2.
- **lipgloss v1.x**: Terminal styling and layout. Pin to stable tag instead of current pseudo-version. Do NOT migrate to v2 beta.
- **catwalk v0.1.4**: Data-driven TUI testing. Critical for editor testing - enables key-sequence reproduction of bugs.
- **cobra v1.10.2** (update from v1.10.1): CLI framework with built-in shell completion and man page generation via cobra/doc.
- **adrg/frontmatter** (new): YAML front matter parsing. Clean API, zero dependencies beyond go-yaml which is already a dep.
- **GoReleaser v2.13.3** (new tool): Release automation for cross-compilation, GitHub releases, Homebrew taps, checksums, SBOM generation.

**Version compatibility note:** The project's go.mod specifies Go 1.24.4 but GitHub Actions release workflow pins Go 1.21 - this is a critical mismatch that will break CI builds. Must update to `go-version-file: 'go.mod'`.

### Expected Features

CalcMark's competitive positioning is as the only terminal-native, scriptable, reproducible calculation notepad. This is a deliberate niche serving developers and engineers who live in the terminal, not a limitation. The v1 feature set should emphasize polish of core capabilities over feature breadth.

**Must have (table stakes):**
- Polished two-pane TUI editor with live source-to-results alignment, no rendering artifacts on resize or wrapping
- Complete CLI help system: `--help` on all commands with examples, `--version` flag, actionable error messages
- Shell completions (bash/zsh/fish) via Cobra's built-in system
- Status bar showing file name, cursor position, modified indicator, keybinding hints
- Help overlay (toggle with `?`) for full keybinding reference without leaving editor
- Save confirmation on quit to prevent data loss
- NO_COLOR / --no-color support for CI and accessibility
- Prebuilt binaries via GoReleaser: Linux amd64/arm64, macOS amd64/arm64, Windows amd64, with checksums
- Homebrew tap for frictionless macOS/Linux install
- User-facing README with installation, quick start, curated example files

**Should have (competitive differentiators):**
- YAML front matter for document-level constants (tax rates, dates, exchange rates)
- In-TUI autocomplete for variables, functions, units - IDE-like experience in terminal
- Rich error display in results pane - inline errors with color coding, not silence
- `cm eval --json` structured output for scripting and CI integration
- Man page generation via cobra/doc bundled with Homebrew installs
- Markdown rendering in preview pane for markdown-heavy documents
- Scoop bucket for Windows users

**Defer (v2+):**
- Syntax highlighting in source pane (requires custom textarea rendering)
- Full undo/redo (textarea has basic undo; full undo/redo needs undo stack coordinated with evaluator)
- LSP server (separate project scope)
- Plugin system (requires API stability commitment)
- Vim/Emacs keybinding modes (testing matrix explosion, conflicts with CalcMark keybindings)

**Anti-features (reject):**
- Live currency exchange rates (breaks reproducibility - CalcMark's core value)
- Multi-file/tab support (significant UI complexity for v1)
- Configuration GUI (CLI tools use config files + documentation)

### Architecture Approach

The project follows the Functional Core, Imperative Shell pattern: pure computation functions (geometry, alignment, wrapping) separated from side-effectful Bubble Tea model logic. The critical architectural issue is that the codebase has two editor implementations in flight: Model (v1, 1400 lines, custom cursor/scroll) and ModelV2 (666 lines, textarea-based, simpler but less feature-complete). The v1 model conflates computation and rendering - 40+ fields handling cursor, document, evaluation, UI, undo, search, and export all in one struct.

**Major components:**
1. **CLI Layer (cobra)** - thin routing to subcommands (eval, convert, edit, tui)
2. **App (bubbletea top-level)** - mode switching between REPL and Editor via SwitchModeMsg
3. **Editor Model** - two-pane editor core: source (left), results (right) with live alignment
4. **Pure Geometry Layer (NEW)** - extract to editor/geometry/ with pure functions: aligned.go, linemodel.go, wrap.go, sidebyside.go - fully testable, no lipgloss/tea dependencies
5. **Shared Components** - reusable pure rendering: StatusBar, ContextFooter, Suggest, Globals
6. **spec/ (Language)** - lexer, parser, AST, semantic, document, types, units - NO impl dependencies
7. **impl/ (Runtime)** - interpreter, evaluator, Environment, WASM bindings - depends on spec only
8. **format/** - output formatters (text, JSON, HTML, MD, CM) - depends on spec only

**Critical pattern: Data-driven testing with catwalk.** The project already has 50+ catwalk test files for Model. Every user-facing TUI bug must have a catwalk test that reproduces the exact key sequence, proves the bug exists (test fails), then validates the fix (test passes). Catwalk tests the MODEL state, not rendered output - this is fundamentally better than screenshot testing because it tests behavior, not pixels.

**Architecture decision required:** The editor rewrite must converge on ModelV2 with textarea as source-of-truth for cursor, scrolling, undo, and text content. The editor model should focus on: document sync, evaluation, and feature state. Geometry computation must move to pure functions in editor/geometry/. All existing Model catwalk tests must be ported to ModelV2 before declaring it ready.

### Critical Pitfalls

1. **Two-pane alignment desynchronization under wrapping:** Source and preview panes lose 1:1 vertical alignment when text wrapping produces different numbers of visual lines. This is already the most complex area - the entire aligned.go architecture exists because of this problem. Any change to wrapping, pane width ratios, or preview rendering can silently break alignment. The existing computeAlignedPanes is the single source of truth, but view.go performs additional adjustments for the edit buffer at render time, creating a second alignment path that can diverge. **Prevention:** Never modify alignment without running full catwalk suite. Add catwalk tests for wrapping at narrow terminal widths (40 columns), long calc lines, long markdown headings. Ensure AlignedModel is ONLY source of truth - if view layer needs adjustments, flow through AlignedModel. Test at boundary widths where content wraps to 2/3 lines.

2. **ModelV2 textarea integration losing custom editing semantics:** The textarea component treats content as plain text but CalcMark needs to re-parse and re-evaluate on every change, maintain two-pane alignment, and provide domain-specific keybindings. The current ModelV2.syncDocumentFromTextarea() does full document rebuild on every keystroke, defeating the incremental evaluation that Model.reEvaluate() provides. **Prevention:** Implement debounced sync (50ms) rather than per-keystroke. Port all existing Model catwalk tests to ModelV2 before declaring ready. Preserve incremental evaluation with EvaluateAffectedBlocks.

3. **GitHub Actions release workflow Go version mismatch:** The project's go.mod specifies Go 1.24.4 but .github/workflows/release.yml pins go-version: '1.21'. CI cannot compile the project. **Prevention:** Update release.yml to use go-version-file: 'go.mod'. Add pre-release check that go version output matches go.mod. Test release workflow with --local flag before every release.

4. **WASM binary size explosion:** Go WASM binary starts at ~1.3MB. CalcMark imports fmt (~400KB), decimal library, entire parser/evaluator. Without management, binary can reach 5-10MB. The Taskfile shows both build:wasm (standard) and build:wasm:tiny (TinyGo) but TinyGo compatibility is unverified. **Prevention:** Measure current WASM size. Set binary size budget: under 3MB uncompressed, under 1MB gzip'd. Profile size contributors. Use build tags to exclude TUI/CLI code from WASM builds. Serve with gzip compression.

5. **Full document rebuild on every edit degrades with document size:** The redetectBlockTypes function recreates the entire document on every keystroke. This is O(n) per keystroke and will feel sluggish on 100+ line documents. **Prevention:** Use incremental block detection - only re-parse the block containing the edited line. Use EvaluateAffectedBlocks for evaluation. This is acceptable for v1 MVP with debounce but must be optimized before v2.

## Implications for Roadmap

Based on research, the work divides into 3 major phases with 1 foundation phase and 1 distribution phase:

### Phase 0: Foundation (Pre-work)
**Rationale:** Fix critical infrastructure issues before feature work begins. The Go version mismatch and alignment architecture must be resolved or they will block all other work.
**Delivers:** Clean foundation for feature development
**Addresses:**
- Update go.mod to Go 1.24.12
- Update GitHub Actions to use go-version-file: 'go.mod'
- Update cobra to v1.10.2
- Verify all tests pass with updated dependencies
- Extract pure geometry functions to editor/geometry/ with comprehensive unit tests

**Avoids:** Go version mismatch pitfall (CI breakage), building features on unstable geometry foundation

### Phase 1: TUI Editor Rewrite (Core Product)
**Rationale:** This is the product's primary interface and must be rock-solid before other features build on it. The Model vs ModelV2 decision blocks all other TUI work. Alignment is the hardest technical problem and must be solved first.
**Delivers:** Production-ready ModelV2 with textarea integration, perfect two-pane alignment under all wrapping scenarios, comprehensive catwalk test coverage
**Addresses:**
- Converge on ModelV2 with textarea as source-of-truth
- Implement debounced document sync (50ms)
- Port all Model catwalk tests to ModelV2
- Add wrapping-specific catwalk tests: narrow terminals (40 cols), long lines, unicode
- Wire geometry functions into ModelV2
- Implement incremental evaluation with EvaluateAffectedBlocks
- Add status bar integration (file name, cursor pos, modified indicator)
- Add save confirmation on quit (when modified=true)
- Verify window resize handling with no rendering artifacts

**Uses:** bubbletea v1.3.10, bubbles v0.21.0 textarea, lipgloss v1.x, catwalk v0.1.4
**Implements:** Editor Model, Pure Geometry Layer, Shared Components (StatusBar)
**Avoids:** Two-pane alignment pitfall, ModelV2 regression pitfall, performance trap from full document rebuild

### Phase 2: Help & Discoverability (UX Polish)
**Rationale:** Users cannot use features they cannot discover. Help system has low implementation cost (Cobra provides most infrastructure) but high user value. Shell completions are currently disabled despite being built into Cobra. This phase makes CalcMark feel professional and complete.
**Delivers:** Complete discoverability layer across CLI and TUI
**Addresses:**
- Re-enable Cobra shell completions (currently disabled: DisableDefaultCmd = true)
- Add completion bash/zsh/fish subcommands
- Implement ValidArgsFunction for .cm file argument completion
- Define centralized shared.KeyMap with key.Binding definitions
- Build help overlay component (toggle with `?`) using bubbles/help
- Generate help content from shared.KeyMap (single source of truth)
- Polish CLI --help text with examples on all commands
- Add --version flag to root command
- Audit all error paths for actionable error messages
- Add NO_COLOR / --no-color support

**Uses:** cobra v1.10.2 built-in completion, bubbles/help, bubbles/key
**Implements:** Help overlay component, Keybinding definitions
**Avoids:** Help content drift pitfall, autocomplete UX annoyance (help first, autocomplete later)

### Phase 3: Advanced Features (Differentiators)
**Rationale:** Features that set CalcMark apart from Numi/Soulver/Calca. YAML front matter and autocomplete are the highest-value differentiators but require the editor foundation from Phase 1. These can be implemented incrementally and independently.
**Delivers:** Document-level constants via front matter, IDE-like autocomplete, rich error display
**Addresses:**
- Add adrg/frontmatter dependency
- Implement YAML front matter parsing (--- delimiters)
- Inject front matter variables into evaluator context before document evaluation
- Build suggestion engine as pure function (prefix + available names → matches)
- Wire SuggestionSource interface to evaluator context for variable names
- Add dropdown rendering above cursor (RenderDropdownSuggestions already exists)
- Trigger autocomplete on explicit action (Tab or Ctrl+Space), not automatic
- Render errors inline in results pane with color coding (red for errors, yellow for warnings)
- Add markdown rendering in preview pane using glamour for markdown blocks

**Uses:** adrg/frontmatter (new), existing SuggestionSource interface, glamour v0.10.0
**Implements:** Front matter parsing in document layer, Autocomplete trigger logic, Error display component
**Avoids:** Autocomplete UX pitfall (explicit trigger, not automatic), feature creep (no live currency rates)

### Phase 4: Distribution & Release (Go-to-Market)
**Rationale:** Users cannot adopt what they cannot install. GoReleaser provides the standard Go distribution workflow: cross-platform builds, checksums, Homebrew/Scoop packaging, release notes. This phase unlocks frictionless installation.
**Delivers:** Professional distribution infrastructure for all major platforms
**Addresses:**
- Add .goreleaser.yml configuration (v2 format)
- Configure builds: linux amd64/arm64, darwin amd64/arm64, windows amd64
- Add GitHub Actions workflow triggered on tag push (v*)
- Configure Homebrew tap in .goreleaser.yml
- Configure Scoop bucket in .goreleaser.yml
- Generate man pages with cobra/doc.GenManTree()
- Bundle man pages and shell completion files in release archives
- Add ldflags for version embedding: -X main.Version={{.VERSION}}
- Include WASM binary in release (existing build:wasm task)
- Measure WASM binary size, enforce <3MB budget
- Add build tags to exclude TUI code from WASM build if size exceeds budget
- Test release workflow with goreleaser release --snapshot --clean
- Write human-readable release notes for v1.0

**Uses:** GoReleaser v2.13.3, cobra/doc for man pages
**Implements:** Release automation, cross-platform binaries, Homebrew tap, Scoop bucket
**Avoids:** Go version mismatch (already fixed in Phase 0), WASM size explosion pitfall, cross-platform binary issues

### Phase 5: Documentation (Last Mile)
**Rationale:** Documentation comes after features are stable. Writing docs for moving targets wastes effort. This phase captures the v1 product as-shipped.
**Delivers:** Complete user-facing documentation
**Addresses:**
- Write user-facing README (installation, quick start, basic usage)
- Curate 3-5 example .cm files (budget template, unit conversion, date math)
- Add JSON output format: cm eval --json for scripting
- Polish cm convert output quality (HTML self-contained, JSON with metadata, markdown round-trip)
- Write language reference documentation in docs/ (markdown files)
- Auto-generate CLI reference with cobra/doc.GenMarkdownTree()
- Document configuration in docs/configuration.md (~/.config/calcmark/config.toml)
- Add cm examples subcommand or document examples location

**Uses:** cobra/doc.GenMarkdownTree()
**Implements:** Documentation in docs/, example files in docs/examples/
**Avoids:** Documentation/code drift pitfall (docs written after features stable)

### Phase Ordering Rationale

- **Phase 0 before all others:** Infrastructure bugs (Go version mismatch) will block CI. Geometry extraction provides stable foundation for Phase 1.
- **Phase 1 (Editor) must come first:** This is the core product. All TUI features (help overlay, autocomplete) depend on a stable editor model. Alignment is the hardest problem and blocks everything else.
- **Phase 2 (Help) before Phase 3 (Features):** Users must discover existing features before adding new ones. Help system is low-cost, high-value, and independent of editor internals.
- **Phase 3 (Features) after editor stable:** YAML front matter and autocomplete depend on document evaluation flow which must be solid from Phase 1. These features are independently implementable.
- **Phase 4 (Distribution) and Phase 5 (Docs) are parallelizable:** Once Phase 3 features are implemented, distribution and documentation can proceed in parallel. However, docs should follow features to avoid documenting moving targets.

**Alternative ordering:** Phase 4 (Distribution) could come before Phase 3 (Features) to enable early-adopter testing of the core editor. This trades feature completeness for faster feedback loops. Acceptable if the v1.0 definition is "polished editor" rather than "editor + differentiators."

### Research Flags

**Phases likely needing deeper research during planning:**
- **Phase 3 (YAML Front Matter):** Integration point between document parser (spec/) and evaluator (impl/) requires careful design to maintain one-way dependency. Consider /gsd:research-phase for "How to inject front matter variables into evaluator context without coupling spec/ to impl/?"
- **Phase 3 (Autocomplete):** Trigger detection and popup positioning in textarea-based editor needs research. Consider /gsd:research-phase for "How to detect word-under-cursor in bubbles/textarea and position dropdown above cursor?"

**Phases with standard patterns (skip research-phase):**
- **Phase 0 (Foundation):** Dependency updates follow standard Go module workflow. Geometry extraction is internal refactoring with existing tests as specification.
- **Phase 1 (Editor):** Architecture already defined in model_v2.go. Catwalk tests provide behavioral specification. This is implementation, not research.
- **Phase 2 (Help):** Cobra and bubbles/help are well-documented with examples. This is wiring, not research.
- **Phase 4 (Distribution):** GoReleaser is extensively documented with quick-start guides and examples. This is configuration, not research.
- **Phase 5 (Documentation):** Writing docs does not need research.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Existing stack verified against current releases via WebSearch. Charm ecosystem v1/v2 status confirmed from official GitHub. GoReleaser version checked. Only addition is adrg/frontmatter which is well-documented. |
| Features | MEDIUM-HIGH | Based on competitor analysis (Numi, Soulver, Calca), CLI design guidelines (clig.dev, Heroku), and current project state analysis. Feature prioritization is opinionated but grounded in table-stakes analysis. Anti-features identified through reproducibility requirement. |
| Architecture | HIGH | Direct codebase analysis. Catwalk testing strategy proven with 50+ existing tests. Functional Core / Imperative Shell pattern is standard. Model vs ModelV2 decision is clear from code inspection. Dependency flow (spec → impl → tui) is enforced and correct. |
| Pitfalls | MEDIUM-HIGH | Alignment pitfall confirmed from aligned.go complexity. ModelV2 regression risk clear from comparing Model (1400 lines) vs ModelV2 (666 lines) feature sets. Go version mismatch verified in release.yml. WASM size pitfall estimated from Go WASM base size + imports. Performance trap is O(n) analysis of redetectBlockTypes. |

**Overall confidence:** HIGH

The stack and architecture confidence is high because it's based on direct code inspection and official documentation. Feature confidence is slightly lower because it involves predicting user expectations, but it's grounded in competitor analysis and CLI best practices. Pitfall confidence is high for identified issues (they exist in the code) but medium for estimating their impact (requires implementation experience to fully validate).

### Gaps to Address

**Gap: Exact WASM binary size and TinyGo compatibility**
- Current status: Unknown if build:wasm:tiny task works. Unknown current WASM size.
- How to handle: Measure early in Phase 0 or Phase 4. Run `GOOS=js GOARCH=wasm go build -ldflags "-s -w" -o calcmark.wasm ./cmd/calcmark && ls -lh calcmark.wasm`. If size exceeds 3MB, profile and add build tags. If TinyGo builds fail, document and use standard Go compiler.

**Gap: ModelV2 feature parity with Model**
- Current status: ModelV2 is simpler (666 lines vs 1400) but feature comparison incomplete. Unclear which features are intentionally deferred vs accidentally lost.
- How to handle: During Phase 1 planning, create a feature matrix: Model features (undo/redo, search, export, slash commands, globals panel, pinned vars) vs ModelV2 features. Decide for each: implement in Phase 1, defer to Phase 3, or reject as unnecessary complexity.

**Gap: Incremental evaluation implementation complexity**
- Current status: EvaluateAffectedBlocks exists in impl/document but integration with ModelV2 is unclear. Dependency analysis for cross-block variable references is complex.
- How to handle: Phase 1 can ship with full document rebuild if debounce mitigates lag. Incremental evaluation becomes a performance optimization task post-v1 if users report issues on 100+ line documents.

**Gap: Terminal compatibility testing matrix**
- Current status: Project has VHS tapes for visual verification but no systematic cross-terminal testing. Dark theme and color handling tested on developer's terminal only.
- How to handle: Phase 1 should include manual testing on at least 3 terminals: macOS Terminal.app (limited color), iTerm2/WezTerm (truecolor), Windows Terminal. Phase 4 should add CI smoke tests on each platform (basic open/edit/save/quit).

**Gap: Help content generation from keybinding definitions**
- Current status: shared.KeyMap does not exist yet. Help text and actual keybindings can drift.
- How to handle: Phase 2 must create shared/keys.go with key.Binding definitions as the single source of truth. Both TUI help overlay and CLI --help text generate from this. Add a test that compares help output to key definitions.

## Sources

### Primary (HIGH confidence)
- CalcMark codebase direct analysis: go.mod, Taskfile.yml, .github/workflows/release.yml, cmd/calcmark/tui/editor/*.go, spec/, impl/
- [Bubble Tea GitHub](https://github.com/charmbracelet/bubbletea) - v1.3.10 latest stable, v2 RC2 status
- [Bubbles GitHub](https://github.com/charmbracelet/bubbles) - v0.21.0 latest stable, textarea component API
- [Lipgloss GitHub](https://github.com/charmbracelet/lipgloss) - v1.x stable, v2 beta status
- [Glamour GitHub](https://github.com/charmbracelet/glamour) - v0.10.0 latest
- [Cobra pkg.go.dev](https://pkg.go.dev/github.com/spf13/cobra) - v1.10.2, shell completion API, cobra/doc
- [GoReleaser releases](https://github.com/goreleaser/goreleaser/releases) - v2.13.3 latest
- [GoReleaser quick start](https://goreleaser.com/quick-start/) - configuration best practices
- [adrg/frontmatter GitHub](https://github.com/adrg/frontmatter) - YAML front matter parsing API
- [knz/catwalk](https://github.com/knz/catwalk) - Test library for Bubbletea TUI models
- [Go 1.24 release notes](https://go.dev/doc/go1.24) - patch history, 1.24.12 latest

### Secondary (MEDIUM confidence)
- [Command Line Interface Guidelines (clig.dev)](https://clig.dev/) - CLI design consensus
- [Heroku CLI Style Guide](https://devcenter.heroku.com/articles/cli-style-guide) - industry standards
- [Bubble Tea v2 Migration Guide](https://github.com/charmbracelet/bubbletea/discussions/1374) - breaking changes
- [Tips for building Bubble Tea programs](https://leg100.github.io/en/posts/building-bubbletea-programs/) - architecture patterns
- [Bubble Tea State Machine pattern](https://zackproser.com/blog/bubbletea-state-machine) - state management
- [Soulver App](https://soulver.app/), [Numi App](https://numi.app/) - competitor analysis
- [Atlassian: 10 Design Principles for Delightful CLIs](https://www.atlassian.com/blog/it-teams/10-design-principles-for-delightful-clis)
- [GoReleaser + GitHub Actions Guide](https://gitgist.com/posts/goreleaser-and-github-actions/)
- [State of Terminal Emulators 2025 - Jeff Quast](https://www.jeffquast.com/post/state-of-terminal-emulation-2025/) - Unicode width challenges
- [Go WASM Binary Size Analysis](https://dev.bitolog.com/minimizing-go-webassembly-binary-size/) - size optimization

### Tertiary (LOW confidence)
- [Awesome TUIs list (GitHub)](https://github.com/rothgar/awesome-tuis) - ecosystem survey
- [Autocomplete UX best practices](https://smart-interface-design-patterns.com/articles/autocomplete-ux/) - GUI-focused, adapted for TUI

---
*Research completed: 2026-02-02*
*Ready for roadmap: yes*
