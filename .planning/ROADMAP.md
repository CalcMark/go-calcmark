# Roadmap: CalcMark

## Milestones

- [x] **v1.0 CalcMark** - Phases 1-8 (shipped 2026-02-06)
- [ ] **v1.1 CalcMark Language** - Phases 9-17 (in progress)

## Phases

<details>
<summary>v1.0 CalcMark (Phases 1-8) - SHIPPED 2026-02-06</summary>

See archived roadmap: `.planning/milestones/v1.0-ROADMAP.md`

</details>

### v1.1 CalcMark Language (In Progress)

**Milestone Goal:** Make the interpreter bulletproof and the editor experience complete.

---

- [ ] **Phase 9: Interpreter Correctness** - Fix calculation bugs and audit all conversion paths
- [ ] **Phase 10: Preview Pane** - Show only calculation results with vertical alignment
- [ ] **Phase 11: Navigation** - Word, line, and document movement
- [ ] **Phase 12: Undo/Redo** - Full history with cursor restoration
- [ ] **Phase 13: Clipboard** - Select, cut, copy, paste
- [ ] **Phase 14: File Operations** - Save, quit, new file flows
- [ ] **Phase 15: Help Update** - Accurate keybindings for new features
- [ ] **Phase 16: Source Highlighting** - Syntax coloring with theme support
- [ ] **Phase 17: Testing & Validation** - Comprehensive test coverage

## Phase Details

### Phase 9: Interpreter Correctness

**Goal**: All calculations produce correct results with proper unit handling across all conversion paths

**Depends on**: Phase 8 (v1.0 complete)

**Requirements**: INTERP-01, INTERP-02, INTERP-03, INTERP-04, INTERP-05, INTERP-06

**Success Criteria** (what must be TRUE):
1. `accumulate(5mb/s, 1 day) as napkin` displays ~400GB (not 430K)
2. All unit conversions preserve quantity type through the entire evaluation chain
3. Every function works correctly in standard form (`avg(1,2,3)`) and natural language form (`average of 1, 2, 3`)
4. Unit conversion roundtrips are lossless (meters -> feet -> meters equals original)
5. Compound units like MB/s and km/h evaluate and convert correctly

**Plans**: TBD

---

### Phase 10: Preview Pane

**Goal**: The preview pane shows only calculation results, vertically aligned with source lines

**Depends on**: Phase 9 (correct calculation results to display)

**Requirements**: PREVIEW-01, PREVIEW-02, PREVIEW-03, PREVIEW-04, PREVIEW-05

**Success Criteria** (what must be TRUE):
1. Preview pane shows calculation results only (no markdown text echoed)
2. Each result line aligns vertically with its corresponding source line
3. Variable assignments display as `variable_name -> result`
4. Anonymous calculations display as `# -> result`
5. Non-calculation lines show as blank (preserving vertical spacing)

**Plans**: TBD

---

### Phase 11: Navigation

**Goal**: Users can efficiently navigate within and across lines using keyboard shortcuts

**Depends on**: Phase 10 (stable editor foundation)

**Requirements**: NAV-01, NAV-02, NAV-03, NAV-04, NAV-05, NAV-06

**Success Criteria** (what must be TRUE):
1. Ctrl+Left and Alt+B move cursor one word left
2. Ctrl+Right and Alt+F move cursor one word right
3. Home and Ctrl+A move cursor to start of line
4. End and Ctrl+E move cursor to end of line
5. Ctrl+Home moves to document start; Ctrl+End moves to document end

**Plans**: TBD

---

### Phase 12: Undo/Redo

**Goal**: Users can undo and redo any edit with full cursor position restoration

**Depends on**: Phase 11 (navigation foundation)

**Requirements**: UNDO-01, UNDO-02, UNDO-03, UNDO-04, UNDO-05

**Success Criteria** (what must be TRUE):
1. Ctrl+Z undoes the last edit
2. Ctrl+Y redoes the last undone edit
3. Undo history is unlimited within session (no 100-state cap)
4. Cursor position is restored to where it was before each edit
5. Undo/redo work correctly for edits spanning multiple lines

**Plans**: TBD

---

### Phase 13: Clipboard

**Goal**: Users can select, cut, copy, and paste text using standard keybindings

**Depends on**: Phase 11 (navigation for selection), Phase 12 (undo for paste recovery)

**Requirements**: CLIP-01, CLIP-02, CLIP-03, CLIP-04

**Success Criteria** (what must be TRUE):
1. Ctrl+A selects the entire document
2. Ctrl+X cuts selected text to system clipboard
3. Ctrl+C copies selected text to system clipboard (when selection exists)
4. Ctrl+V pastes from system clipboard at cursor position

**Plans**: TBD

---

### Phase 14: File Operations

**Goal**: Users can save, create, and quit files with proper dirty-state handling

**Depends on**: Phase 12 (undo affects dirty state)

**Requirements**: FILE-01, FILE-02, FILE-03, FILE-04, FILE-05, FILE-06, FILE-07, FILE-08

**Success Criteria** (what must be TRUE):
1. Ctrl+S saves to current file and shows "Saved" in status bar
2. Ctrl+Shift+S prompts for filename (Save As)
3. Ctrl+N creates new file, prompting if current has unsaved changes
4. Ctrl+Q quits with "Save? Y/N/Cancel" prompt if unsaved changes exist
5. Saves are atomic (temp file + rename, no data loss on crash)

**Plans**: TBD

---

### Phase 15: Help Update

**Goal**: Help overlay accurately reflects all v1.1 keybindings

**Depends on**: Phases 11-14 (all keybindings implemented)

**Requirements**: HELP-01, HELP-02

**Success Criteria** (what must be TRUE):
1. F1 or Ctrl+H toggles the help overlay
2. Help overlay lists all keybindings added in v1.1 (navigation, undo/redo, clipboard, file ops)

**Plans**: TBD

---

### Phase 16: Source Highlighting

**Goal**: Editor displays syntax-highlighted source with configurable themes

**Depends on**: Phase 14 (stable editor before visual changes)

**Requirements**: THEME-01, THEME-02, THEME-03, THEME-04, THEME-05, THEME-06

**Success Criteria** (what must be TRUE):
1. Headings, calculations, and markdown have distinct line-level colors
2. Within calculations, variables/numbers/units have distinct inline colors
3. User can override theme via config file (~/.config/calcmark/theme.toml)
4. Built-in dark and light themes adapt to terminal background
5. Typing feels responsive (no jank or flicker during rapid input)

**Plans**: TBD

---

### Phase 17: Testing & Validation

**Goal**: All v1.1 features have comprehensive test coverage with real-world validation

**Depends on**: Phases 9-16 (all features complete)

**Requirements**: TEST-01, TEST-02, TEST-03, TEST-04, TEST-05, TEST-06, TEST-07

**Success Criteria** (what must be TRUE):
1. Property-based tests verify unit conversion roundtrip accuracy
2. Golden tests cover napkin conversion with diverse unit types
3. Catwalk tests exercise undo/redo, save/quit, and clipboard flows
4. Catwalk tests verify preview pane alignment
5. Real-world test suite includes 10+ diverse .cm files with expected outputs

**Plans**: TBD

---

## Progress

**Execution Order:** Phases execute in numeric order: 9 -> 10 -> 11 -> 12 -> 13 -> 14 -> 15 -> 16 -> 17

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 9. Interpreter Correctness | v1.1 | 0/TBD | Not started | - |
| 10. Preview Pane | v1.1 | 0/TBD | Not started | - |
| 11. Navigation | v1.1 | 0/TBD | Not started | - |
| 12. Undo/Redo | v1.1 | 0/TBD | Not started | - |
| 13. Clipboard | v1.1 | 0/TBD | Not started | - |
| 14. File Operations | v1.1 | 0/TBD | Not started | - |
| 15. Help Update | v1.1 | 0/TBD | Not started | - |
| 16. Source Highlighting | v1.1 | 0/TBD | Not started | - |
| 17. Testing & Validation | v1.1 | 0/TBD | Not started | - |

---

*Roadmap created: 2026-02-06*
*Last updated: 2026-02-06*
