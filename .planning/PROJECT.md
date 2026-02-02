# CalcMark

## What This Is

CalcMark is an interpreted language that blends CommonMark markdown with calculations, creating verifiable and reproducible calculation documents. Think Jupyter notebooks but simpler—human-readable .cm files with embedded calculations. The `cm` tool provides a TUI editor for authoring CalcMark documents interactively and CLI commands for converting, evaluating, and working with these files. Use cases include engineering notes with live calculations, financial documents where numbers update automatically, and scientific reports with embedded data analysis.

## Core Value

Fast, offline, verifiable calculations in markdown documents with a simple editor. The core interpreter must be correct and blazingly fast—everything else can fail, but calculation accuracy and performance cannot.

## Requirements

### Validated

<!-- Existing codebase capabilities that work and are relied upon -->

- ✓ Core interpreter (parser, evaluator, type system) — existing
- ✓ CLI with convert, eval, edit commands — existing
- ✓ Format templates (HTML, MD, JSON, text, cm) — existing
- ✓ Unit system with comprehensive conversions — existing
- ✓ Functions library with English-language synonyms — existing
- ✓ Pure Go implementation with minimal dependencies — existing

### Active

<!-- Current scope for v1 GitHub release -->

- [ ] TUI editor works correctly with proper two-column layout (adopt code.sh algorithm to fix wrapping/alignment bugs)
- [ ] Help system accessible via CLI (`cm help`, `cm help functions`, `cm help constants`) and within TUI
- [ ] TUI autocomplete for functions, constants, and English synonyms while typing
- [ ] YAML front matter for document-level constants (exchange rates, assumptions)
- [ ] README and documentation explaining what CalcMark is and how to use it
- [ ] Prebuilt binaries for Mac/Linux/Windows available via GitHub releases
- [ ] Validated .cm integration test files with verified correct calculations

### Out of Scope

- Network calls (except future publishing/fetching from URLs) — must work completely offline
- Non-Go dependencies — keep pure Go with stdlib preference
- Shell autocomplete (bash/zsh completion) — TUI autocomplete only
- Documented requirements not on critical release path — ignore until explicitly prioritized
- New language features beyond YAML front matter — CalcMark spec is stable

## Context

**Existing Codebase:**
- Brownfield Go project with working interpreter and CLI
- Implementation IS the spec (no formal language specification)
- Core interpreter well-tested but integration test .cm files need validation
- TUI has fundamental architectural issues: wrapping, two-column alignment, cursor positioning
- Video recorder tests are flakey due to underlying TUI architecture problems

**Technical Environment:**
- Pure Go codebase with minimal external dependencies
- Performance-critical interpreter (parser avoids time complexity bottlenecks)
- Must work completely offline (zero network calls)
- Supports multiple output formats via templates

**Known Issues:**
- TUI two-column layout fundamentally broken (code.sh describes fix)
- Discoverability problem: even creator forgets available functions/constants
- Integration tests may contain incorrect calculations
- Flakey video-based TUI tests mask real architectural problems

## Constraints

- **Pure Go**: Minimal dependencies, prefer standard library where possible
- **Offline-first**: Must work with zero network calls (future: optional publishing features)
- **Performance**: Interpreter must be extremely fast, parser avoids O(n²) bottlenecks
- **Stability**: CalcMark language spec is stable, backwards compatibility matters after v1
- **Focus**: Ship release bar first, ignore distracting documented requirements

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Adopt code.sh TUI architecture | Current two-column layout has unfixable wrapping/alignment bugs | — Pending |
| TUI autocomplete only (no shell) | Authoring workflow needs discovery, shell autocomplete less critical | — Pending |
| YAML front matter for constants | Makes document assumptions explicit and discoverable | — Pending |
| Offline-only for v1 | Simplicity and reliability over cloud features | — Pending |

---
*Last updated: 2026-02-02 after initialization*
