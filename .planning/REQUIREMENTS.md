# Requirements: CalcMark v1.1

**Defined:** 2026-02-06
**Core Value:** Fast, offline, verifiable calculations in markdown documents with a simple editor

## v1.1 Requirements

### Interpreter Correctness

- [ ] **INTERP-01**: Napkin conversion preserves unit types (`accumulate(5mb/s, 1 day) as napkin` shows ~400GB not 430K)
- [ ] **INTERP-02**: All conversion paths audit — verify no type erasure in any conversion function
- [ ] **INTERP-03**: Standard function forms work correctly (`avg()`, `sum()`, `accumulate()`, etc.)
- [ ] **INTERP-04**: Natural language function forms work correctly (`average of...`, `sum of...`, etc.)
- [ ] **INTERP-05**: Unit conversion roundtrips are accurate (e.g., meters → feet → meters)
- [ ] **INTERP-06**: Compound units handle correctly (e.g., MB/s, km/h)

### Preview Pane

- [ ] **PREVIEW-01**: Preview pane shows ONLY calculation results (not markdown text)
- [ ] **PREVIEW-02**: Results vertically aligned with source calculation lines
- [ ] **PREVIEW-03**: Variable assignments display as `variable_name -> result`
- [ ] **PREVIEW-04**: Anonymous calculations display as `# -> result`
- [ ] **PREVIEW-05**: Non-calculation lines show blank in preview (spacing preserved)

### File Operations

- [ ] **FILE-01**: Ctrl+S saves to current file, shows "Saved" in status bar
- [ ] **FILE-02**: Ctrl+Shift+S prompts for filename (Save As)
- [ ] **FILE-03**: Ctrl+N creates new file (prompts if unsaved)
- [ ] **FILE-04**: Ctrl+Q quits with unsaved changes prompt ("Save? Y/N/Cancel")
- [ ] **FILE-05**: Ctrl+C force quits immediately without prompt
- [ ] **FILE-06**: Atomic saves using temp file + rename pattern (no data loss on crash)
- [ ] **FILE-07**: editBuf flushed before save (no lost characters during typing)
- [ ] **FILE-08**: Modified indicator accurate in status bar

### Undo/Redo

- [ ] **UNDO-01**: Ctrl+Z triggers undo
- [ ] **UNDO-02**: Ctrl+Y triggers redo
- [ ] **UNDO-03**: Unlimited undo history within session (remove 100-state limit)
- [ ] **UNDO-04**: Cursor position restored on undo/redo
- [ ] **UNDO-05**: Undo/redo work correctly across line boundaries

### Clipboard Operations

- [ ] **CLIP-01**: Ctrl+A selects entire document
- [ ] **CLIP-02**: Ctrl+X cuts selection to clipboard
- [ ] **CLIP-03**: Ctrl+C copies selection to clipboard (when text selected)
- [ ] **CLIP-04**: Ctrl+V pastes from clipboard

### Navigation

- [ ] **NAV-01**: Ctrl+← or Alt+B moves cursor one word left
- [ ] **NAV-02**: Ctrl+→ or Alt+F moves cursor one word right
- [ ] **NAV-03**: Home or Ctrl+A moves to start of line
- [ ] **NAV-04**: End or Ctrl+E moves to end of line
- [ ] **NAV-05**: Ctrl+Home moves to start of document
- [ ] **NAV-06**: Ctrl+End moves to end of document

### Help

- [ ] **HELP-01**: F1 or Ctrl+H toggles help overlay
- [ ] **HELP-02**: Help overlay shows all keybindings accurately

### Source Highlighting

- [ ] **THEME-01**: Line-level highlighting by block type (headings purple+bold, calculations blue, markdown gray)
- [ ] **THEME-02**: Inline highlighting within calculations (variables purple, numbers blue, units green)
- [ ] **THEME-03**: User-overridable theme via config file (~/.config/calcmark/theme.toml or similar)
- [ ] **THEME-04**: Built-in dark and light themes that respect terminal background
- [ ] **THEME-05**: No jank or flickering during typing (debounce or skip highlighting during rapid input)
- [ ] **THEME-06**: Fallback to line-level only if inline highlighting causes performance issues

### Testing

- [ ] **TEST-01**: Property-based tests for unit conversion roundtrips
- [ ] **TEST-02**: Golden tests for napkin conversion with various units
- [ ] **TEST-03**: Catwalk tests for undo/redo flows
- [ ] **TEST-04**: Catwalk tests for save/quit/new flows
- [ ] **TEST-05**: Catwalk tests for clipboard operations
- [ ] **TEST-06**: Catwalk tests for preview pane alignment
- [ ] **TEST-07**: Real-world document test suite (10+ diverse .cm files)

## Future Requirements

Deferred to v1.2 or later.

### UX Enhancements

- **UX-01**: Character batching in undo (continuous typing as single action)
- **UX-02**: Search and replace in editor
- **UX-03**: Multiple selections

### Correctness Extensions

- **CORR-01**: Temperature conversion validation (non-linear C/F/K)
- **CORR-02**: Unit prefix case sensitivity audit (MB vs MiB vs Mb)

## Out of Scope

| Feature | Reason |
|---------|--------|
| Tree-based redo history | Complexity not justified for v1.1 |
| Vim keybindings | Testing matrix explosion, defer to future |
| LSP server | Separate project scope |
| Plugin system | Requires API stability commitment |
| Markdown rendering in preview | Preview now shows only calculation results |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| INTERP-01 | TBD | Pending |
| INTERP-02 | TBD | Pending |
| INTERP-03 | TBD | Pending |
| INTERP-04 | TBD | Pending |
| INTERP-05 | TBD | Pending |
| INTERP-06 | TBD | Pending |
| PREVIEW-01 | TBD | Pending |
| PREVIEW-02 | TBD | Pending |
| PREVIEW-03 | TBD | Pending |
| PREVIEW-04 | TBD | Pending |
| PREVIEW-05 | TBD | Pending |
| FILE-01 | TBD | Pending |
| FILE-02 | TBD | Pending |
| FILE-03 | TBD | Pending |
| FILE-04 | TBD | Pending |
| FILE-05 | TBD | Pending |
| FILE-06 | TBD | Pending |
| FILE-07 | TBD | Pending |
| FILE-08 | TBD | Pending |
| UNDO-01 | TBD | Pending |
| UNDO-02 | TBD | Pending |
| UNDO-03 | TBD | Pending |
| UNDO-04 | TBD | Pending |
| UNDO-05 | TBD | Pending |
| CLIP-01 | TBD | Pending |
| CLIP-02 | TBD | Pending |
| CLIP-03 | TBD | Pending |
| CLIP-04 | TBD | Pending |
| NAV-01 | TBD | Pending |
| NAV-02 | TBD | Pending |
| NAV-03 | TBD | Pending |
| NAV-04 | TBD | Pending |
| NAV-05 | TBD | Pending |
| NAV-06 | TBD | Pending |
| HELP-01 | TBD | Pending |
| HELP-02 | TBD | Pending |
| THEME-01 | TBD | Pending |
| THEME-02 | TBD | Pending |
| THEME-03 | TBD | Pending |
| THEME-04 | TBD | Pending |
| THEME-05 | TBD | Pending |
| THEME-06 | TBD | Pending |
| TEST-01 | TBD | Pending |
| TEST-02 | TBD | Pending |
| TEST-03 | TBD | Pending |
| TEST-04 | TBD | Pending |
| TEST-05 | TBD | Pending |
| TEST-06 | TBD | Pending |
| TEST-07 | TBD | Pending |

**Coverage:**
- v1.1 requirements: 48 total
- Mapped to phases: 0 (pending roadmap)
- Unmapped: 48

---
*Requirements defined: 2026-02-06*
*Last updated: 2026-02-06 after UX review*
