# Requirements: CalcMark v1

**Defined:** 2026-02-02
**Core Value:** Fast, offline, verifiable calculations in markdown documents with a simple editor

## v1 Requirements

### Foundation

- [ ] **FOUND-01**: Go version in release workflow matches go.mod (1.24.4, not 1.21)
- [ ] **FOUND-02**: Pure geometry functions extracted to separate package with comprehensive tests
- [ ] **FOUND-03**: CalculateRowGeometry function (from code.sh) handles wrapping on both columns
- [ ] **FOUND-04**: Two-column alignment works correctly when left/right have different wrap heights
- [ ] **FOUND-05**: Dependencies updated to current stable versions (adrg/frontmatter added)

### TUI Editor

- [ ] **EDITOR-01**: code.sh geometry algorithm implemented for two-column layout
- [ ] **EDITOR-02**: Left pane (source) wraps text correctly at column boundary
- [ ] **EDITOR-03**: Right pane (results) wraps text correctly and independently
- [ ] **EDITOR-04**: Vertical alignment between source and results maintained under all wrapping scenarios
- [ ] **EDITOR-05**: Cursor positioning works correctly in wrapped text
- [ ] **EDITOR-06**: Cursor visible and tracks user input accurately
- [ ] **EDITOR-07**: Scrolling works correctly with viewport
- [ ] **EDITOR-08**: Model integration (code.sh geometry + model state management) complete
- [ ] **EDITOR-09**: Debounced evaluation triggers on text changes
- [ ] **EDITOR-10**: Results update correctly in right pane after evaluation
- [ ] **EDITOR-11**: Catwalk tests cover all editor interactions (typing, cursor, wrapping, alignment)
- [ ] **EDITOR-12**: No flakey video-based tests in CI (catwalk only)

### Help & Discoverability

- [ ] **HELP-01**: Shell completions enabled (remove DisableDefaultCmd from root.go)
- [ ] **HELP-02**: `cm help` shows general CLI overview with examples
- [ ] **HELP-03**: `cm help functions` lists all functions with descriptions and English synonyms
- [ ] **HELP-04**: `cm help constants` lists all built-in constants
- [ ] **HELP-05**: Help output is pipeable (works with less/more)
- [ ] **HELP-06**: In-TUI help overlay shows keybindings and commands
- [ ] **HELP-07**: Status bar displays current mode, cursor position, calculation count
- [ ] **HELP-08**: Status bar shows "EVAL..." during debounced evaluation
- [ ] **HELP-09**: TUI autocomplete suggests functions while typing `=`
- [ ] **HELP-10**: TUI autocomplete suggests constants from canonical units
- [ ] **HELP-11**: TUI autocomplete shows English synonyms for functions
- [ ] **HELP-12**: Autocomplete popup positioned correctly in textarea

### YAML Front Matter

- [ ] **YAML-01**: YAML front matter parsed from .cm files
- [ ] **YAML-02**: Constants defined in front matter available in calculations
- [ ] **YAML-03**: Front matter syntax documented with examples
- [ ] **YAML-04**: Error messages clear when front matter is malformed
- [ ] **YAML-05**: Integration between spec/ and impl/ maintains dependency flow (spec never depends on impl)

### Distribution

- [ ] **DIST-01**: GoReleaser configuration created and tested
- [ ] **DIST-02**: Prebuilt binaries for macOS (Intel + Apple Silicon)
- [ ] **DIST-03**: Prebuilt binaries for Linux (amd64, arm64)
- [ ] **DIST-04**: Prebuilt binaries for Windows (amd64)
- [ ] **DIST-05**: Homebrew tap configured and working
- [ ] **DIST-06**: Scoop bucket configured and working
- [ ] **DIST-07**: Man pages generated and bundled
- [ ] **DIST-08**: Shell completion files bundled in releases
- [ ] **DIST-09**: WASM build optimized and size monitored
- [ ] **DIST-10**: WASM binary included in GitHub releases
- [ ] **DIST-11**: Release workflow runs in CI successfully
- [ ] **DIST-12**: Checksums and signatures for all release artifacts

### Documentation

- [ ] **DOCS-01**: README explains what CalcMark is (Jupyter-like but simpler)
- [ ] **DOCS-02**: README shows installation instructions (Homebrew, Scoop, prebuilt binaries, from source)
- [ ] **DOCS-03**: README includes quick start example
- [ ] **DOCS-04**: README shows all three use cases (TUI editor, CLI eval, convert/export)
- [ ] **DOCS-05**: Example .cm files with correct calculations in testdata/
- [ ] **DOCS-06**: CLI reference generated from cobra commands
- [ ] **DOCS-07**: Function reference generated from interpreter code
- [ ] **DOCS-08**: Constants reference generated from canonical units
- [ ] **DOCS-09**: Screenshots of TUI editor in README
- [ ] **DOCS-10**: Contributing guide exists

## v2 Requirements

Deferred to future release.

### Advanced Features

- **ADV-01**: Variables with assignment (`:=` syntax)
- **ADV-02**: Multi-file includes/imports
- **ADV-03**: Custom function definitions
- **ADV-04**: Export to PDF format
- **ADV-05**: Syntax highlighting in source pane

### UX Enhancements

- **UX-01**: Full undo/redo history (beyond textarea default)
- **UX-02**: Search and replace in editor
- **UX-03**: Multi-file tabs
- **UX-04**: Vim keybindings mode

## Out of Scope

| Feature | Reason |
|---------|--------|
| Live currency exchange rates | Breaks reproducibility (core value) -- documents must be verifiable offline |
| Plugin system | Adds complexity, security concerns -- defer until clear use case emerges |
| GUI desktop application | Terminal-native is the differentiator -- CalcMark vs Numi/Soulver/Calca |
| Syntax highlighting in source | Low value vs complexity -- markdown is readable without it |
| Collaborative editing | Network dependency breaks offline constraint |
| LSP server | Scope creep -- focus on simple editor first |
| Configuration GUI | CLI tool philosophy -- text config files sufficient |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| FOUND-01 | Phase 1: Foundation | Pending |
| FOUND-02 | Phase 1: Foundation | Pending |
| FOUND-03 | Phase 1: Foundation | Pending |
| FOUND-04 | Phase 1: Foundation | Pending |
| FOUND-05 | Phase 1: Foundation | Pending |
| EDITOR-01 | Phase 2: TUI Geometry & Layout | Pending |
| EDITOR-02 | Phase 2: TUI Geometry & Layout | Pending |
| EDITOR-03 | Phase 2: TUI Geometry & Layout | Pending |
| EDITOR-04 | Phase 2: TUI Geometry & Layout | Pending |
| EDITOR-05 | Phase 3: TUI Editor Integration | Pending |
| EDITOR-06 | Phase 3: TUI Editor Integration | Pending |
| EDITOR-07 | Phase 3: TUI Editor Integration | Pending |
| EDITOR-08 | Phase 3: TUI Editor Integration | Pending |
| EDITOR-09 | Phase 3: TUI Editor Integration | Pending |
| EDITOR-10 | Phase 3: TUI Editor Integration | Pending |
| EDITOR-11 | Phase 4: TUI Test Coverage | Pending |
| EDITOR-12 | Phase 4: TUI Test Coverage | Pending |
| HELP-01 | Phase 5: Help System | Pending |
| HELP-02 | Phase 5: Help System | Pending |
| HELP-03 | Phase 5: Help System | Pending |
| HELP-04 | Phase 5: Help System | Pending |
| HELP-05 | Phase 5: Help System | Pending |
| HELP-06 | Phase 5: Help System | Pending |
| HELP-07 | Phase 5: Help System | Pending |
| HELP-08 | Phase 5: Help System | Pending |
| HELP-09 | Phase 6: Differentiators | Pending |
| HELP-10 | Phase 6: Differentiators | Pending |
| HELP-11 | Phase 6: Differentiators | Pending |
| HELP-12 | Phase 6: Differentiators | Pending |
| YAML-01 | Phase 6: Differentiators | Pending |
| YAML-02 | Phase 6: Differentiators | Pending |
| YAML-03 | Phase 6: Differentiators | Pending |
| YAML-04 | Phase 6: Differentiators | Pending |
| YAML-05 | Phase 6: Differentiators | Pending |
| DIST-01 | Phase 7: Distribution | Pending |
| DIST-02 | Phase 7: Distribution | Pending |
| DIST-03 | Phase 7: Distribution | Pending |
| DIST-04 | Phase 7: Distribution | Pending |
| DIST-05 | Phase 7: Distribution | Pending |
| DIST-06 | Phase 7: Distribution | Pending |
| DIST-07 | Phase 7: Distribution | Pending |
| DIST-08 | Phase 7: Distribution | Pending |
| DIST-09 | Phase 7: Distribution | Pending |
| DIST-10 | Phase 7: Distribution | Pending |
| DIST-11 | Phase 7: Distribution | Pending |
| DIST-12 | Phase 7: Distribution | Pending |
| DOCS-01 | Phase 8: Documentation | Pending |
| DOCS-02 | Phase 8: Documentation | Pending |
| DOCS-03 | Phase 8: Documentation | Pending |
| DOCS-04 | Phase 8: Documentation | Pending |
| DOCS-05 | Phase 8: Documentation | Pending |
| DOCS-06 | Phase 8: Documentation | Pending |
| DOCS-07 | Phase 8: Documentation | Pending |
| DOCS-08 | Phase 8: Documentation | Pending |
| DOCS-09 | Phase 8: Documentation | Pending |
| DOCS-10 | Phase 8: Documentation | Pending |

**Coverage:**
- v1 requirements: 56 total
- Mapped to phases: 56
- Unmapped: 0

---
*Requirements defined: 2026-02-02*
*Last updated: 2026-02-02 after roadmap creation*
