# CalcMark

## What This Is

CalcMark is an interpreted language that blends CommonMark markdown with calculations, creating verifiable and reproducible calculation documents. Think Jupyter notebooks but simpler—human-readable .cm files with embedded calculations. The `cm` tool provides a TUI editor for authoring CalcMark documents interactively and CLI commands for converting, evaluating, and working with these files. Use cases include engineering notes with live calculations, financial documents where numbers update automatically, and scientific reports with embedded data analysis.

## Core Value

Fast, offline, verifiable calculations in markdown documents with a simple editor. The core interpreter must be correct and blazingly fast—everything else can fail, but calculation accuracy and performance cannot.

## Requirements

### Validated

- ✓ Core interpreter (parser, evaluator, type system) — existing
- ✓ CLI with convert, eval, edit commands — existing
- ✓ Format templates (HTML, MD, JSON, text, cm) — existing
- ✓ Unit system with comprehensive conversions — existing
- ✓ Functions library with English-language synonyms — existing
- ✓ Pure Go implementation with minimal dependencies — existing
- ✓ TUI editor with correct two-column layout — v1.0
- ✓ Help system via CLI and TUI overlay — v1.0
- ✓ TUI autocomplete for functions, constants, and variables — v1.0
- ✓ YAML front matter for document-level constants — v1.0
- ✓ README and documentation — v1.0
- ✓ Prebuilt binaries for Mac/Linux/Windows — v1.0
- ✓ Validated .cm integration test files — v1.0

### Active

<!-- v1.1 CalcMark Language -->

- [ ] Interpreter correctness: fix quantity/unit conversion bugs (e.g., `as napkin` loses unit context)
- [ ] Audit all unit conversion paths for similar issues
- [ ] Test functions in both `func()` and natural language `average of...` forms
- [ ] Comprehensive real-world document test suite
- [ ] Full undo/redo history (unlimited)
- [ ] Save (Ctrl+S)
- [ ] Quit with unsaved changes prompt
- [ ] Save As functionality

### Out of Scope

- Network calls (except future publishing/fetching from URLs) — must work completely offline
- Non-Go dependencies — keep pure Go with stdlib preference
- Live currency exchange rates — breaks reproducibility
- Plugin system — adds complexity, defer until clear use case
- GUI desktop application — terminal-native is the differentiator
- Collaborative editing — network dependency breaks offline constraint
- LSP server — scope creep, focus on simple editor first

## Current Milestone: v1.1 CalcMark Language

**Goal:** Make the interpreter bulletproof and the editor experience complete.

**Target features:**
- Fix interpreter bugs in quantity/unit calculations and conversions
- Audit all functions for correctness in standard and natural language forms
- Comprehensive real-world document testing
- Full undo/redo, save, quit-without-save, save-as

**Known bugs:**
- `accumulate(5mb/s, 1 day) as napkin` returns "430K" instead of ~400GB — napkin formatter loses unit context

## Context

**Codebase:**
- 296 Go files, 63,337 lines of code
- Pure Go with minimal external dependencies
- Performance-critical interpreter (parser avoids time complexity bottlenecks)
- Works completely offline (zero network calls)
- Supports multiple output formats via templates

**v1.0 Shipped:**
- All 51 requirements complete
- 8 phases, 21 plans executed
- Distribution ready for macOS, Linux, Windows

**Known Technical Debt:**
- Pre-existing modernize warnings (~39 across codebase)
- Visual polish items deferred (column headers, divider centering)
- Preview pane UX enhancement (calc outputs only) deferred

## Constraints

- **Pure Go**: Minimal dependencies, prefer standard library where possible
- **Offline-first**: Must work with zero network calls
- **Performance**: Interpreter must be extremely fast, parser avoids O(n²) bottlenecks
- **Stability**: CalcMark language spec is stable, backwards compatibility matters

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| go-version-file in CI | Auto-tracks go.mod, prevents drift | ✓ Good |
| Pure geometry package | Zero TUI framework dependencies, testable in isolation | ✓ Good |
| Dedicated test functions with fresh documents | Avoid shared state mutation between tests | ✓ Good |
| scrollMargin = 3 lines | Good context without excessive scrolling | ✓ Good |
| evalDebounceDelay = 100ms | Conservative default, responsive feel | ✓ Good |
| F1 for help (not ?) | Avoids conflict with calc expressions | ✓ Good |
| FunctionDef struct with init() | Single source of truth, avoids initialization cycle | ✓ Good |
| GoReleaser v2 | Modern release tooling, Homebrew formula generation | ✓ Good |
| "Calculation notepad" positioning | Clear value prop vs Jupyter | ✓ Good |
| Alt+b/f for word navigation | Works on macOS where Ctrl+Arrow is captured | ✓ Good |

---
*Last updated: 2026-02-06 after v1.1 milestone started*
