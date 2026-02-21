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

- [x] **Phase 9: Interpreter Correctness** - Fix calculation bugs and audit all conversion paths
- [x] **Phase 9.1: Separate Validation from Execution** - Clean spec/impl boundary (INSERTED)
- [x] **Phase 10: Preview Pane** - Show only calculation results with vertical alignment
- [x] **Phase 11: Navigation** - Word, line, and document movement
- [x] **Phase 11.1: Bug Fixes** - Fix convert_rate preview and TUI display bugs (INSERTED)
- [x] **Phase 11.2: UX Redesign** - Holistic command/help system redesign (INSERTED)
- [x] **Phase 12: Undo/Redo** - Full history with cursor restoration
- [x] **Phase 13: Clipboard** - Select, cut, copy, paste
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

**Plans:** 4 plans

Plans:
- [x] 09-01-PLAN.md — Fix napkin type erasure bug (TDD)
- [x] 09-02-PLAN.md — Audit all conversion paths for type erasure
- [x] 09-03-PLAN.md — Standard function forms and unit roundtrip tests
- [x] 09-04-PLAN.md — Natural language forms and compound unit tests

---

### Phase 9.1: Separate Validation from Execution (INSERTED)

**Goal**: Clean separation between spec (validation) and impl (execution) per architecture rules

**Depends on**: Phase 9 (interpreter correctness)

**Requirements**: None (architectural refactoring)

**Success Criteria** (what must be TRUE):
1. `spec/document/evaluate.go` deleted (not renamed - execution belongs in impl/)
2. `spec/document` has NO imports from `impl/` (architecture rule enforced)
3. All execution (interpreter calls, environment) lives only in `impl/document/evaluator.go`
4. All existing tests pass after migration
5. Document struct contains no interpreter state

**Plans:** 3 plans

Plans:
- [x] 09.1-01-PLAN.md — Remove impl/ imports from spec/document, delete evaluate.go
- [x] 09.1-02-PLAN.md — Migrate spec/document tests to impl/document.Evaluator
- [x] 09.1-03-PLAN.md — Migrate TUI editor tests to impl/document.Evaluator

---

### Phase 10: Preview Pane

**Goal**: The preview pane shows only calculation results, vertically aligned with source lines

**Depends on**: Phase 9.1 (clean architecture foundation)

**Requirements**: PREVIEW-01, PREVIEW-02, PREVIEW-03, PREVIEW-04, PREVIEW-05

**Success Criteria** (what must be TRUE):
1. Preview pane shows calculation results only (no markdown text echoed)
2. Each result line aligns vertically with its corresponding source line
3. Variable assignments display as `variable_name -> result`
4. Anonymous calculations display as `-> result` (arrow only, no placeholder)
5. Non-calculation lines show as blank (preserving vertical spacing)

**Plans:** 5 plans

Plans:
- [ ] 10-01-PLAN.md — Update pane ratio and visual layout
- [ ] 10-02-PLAN.md — Add napkin tilde and thousand separators (TDD)
- [ ] 10-03-PLAN.md — Unify currency display logic (TDD)
- [ ] 10-04-PLAN.md — Improve error presentation and cascading detection
- [ ] 10-05-PLAN.md — Add comprehensive preview pane tests

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

**Plans:** 3 plans

Plans:
- [x] 11-01-PLAN.md — Add Ctrl+A/E line navigation and resolve export conflict
- [x] 11-02-PLAN.md — Implement Ctrl+Home/End document navigation
- [x] 11-03-PLAN.md — Comprehensive Alt+B/F word navigation tests

---

### Phase 11.1: Bug Fixes (INSERTED)

**Goal**: Fix known bugs discovered during v1.1 development

**Depends on**: Phase 11 (navigation complete)

**Requirements**: None (bug fixes from pending todos)

**Success Criteria** (what must be TRUE):
1. `convert_rate(10 mb/s, per hour)` displays correct preview result (~36 GB/hour)
2. Deleting last character on a line behaves correctly
3. Cursor positioning after delete at line boundaries is correct
4. Preview pane maintains consistent vertical alignment regardless of cursor position
5. Preview pane shows only results and headings (no other markdown like quotes, links)

**Plans:** 3 plans

Plans:
- [x] 11.1-01-PLAN.md — Fix DELETE key last character bug (TDD)
- [x] 11.1-02-PLAN.md — Fix convert_rate time unit conversion (TDD)
- [x] 11.1-03-PLAN.md — Fix preview pane markdown filtering

---

### Phase 11.2: UX Redesign (INSERTED)

**Goal**: Holistic redesign of command and help system UX

**Depends on**: Phase 11.1 (bug fixes complete)

**Requirements**: None (UX improvement from user feedback)

**Success Criteria** (what must be TRUE):
1. Status bar shows only Ctrl+Q (quit) and Ctrl+H (help/commands)
2. Ctrl+H opens a command menu popup listing all available actions
3. Accelerators (Ctrl+S, Ctrl+E, etc.) work directly without menu
4. Slash commands were removed because / is the CalcMark divide operator. REPL uses : prefix for commands.
5. Help overlay provides comprehensive command reference
6. Command menu is keyboard-navigable with clear visual feedback

**Plans:** 3 plans

Plans:
- [x] 11.2-01-PLAN.md — Command infrastructure and StateCommandMenu
- [x] 11.2-02-PLAN.md — Command menu popup rendering and status bar simplification
- [x] 11.2-03-PLAN.md — Visual file picker for Save/Save-As operations

---

### Phase 12: Undo/Redo

**Goal**: Full history with cursor position restoration

**Depends on**: Phase 11.2 (UX redesign complete)

**Requirements**: UNDO-01, UNDO-02, UNDO-03, UNDO-04, UNDO-05

**Success Criteria** (what must be TRUE):
1. Ctrl+Z undoes the last edit
2. Ctrl+Y redoes the last undone edit
3. Undo history uses operation-based diffs (not snapshots), 1000-state limit
4. Cursor position is restored to where it was before each edit
5. Undo/redo work correctly for edits spanning multiple lines

**Plans:** 3 plans

Plans:
- [x] 12-01-PLAN.md — UndoManager core implementation (EditOperation, circular buffer)
- [x] 12-02-PLAN.md — Timer-based grouping (natural boundaries, 1s delay)
- [x] 12-03-PLAN.md — Editor integration and Ctrl+Z/Y handlers

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

**Plans:** 3 plans

Plans:
- [ ] 13-01-PLAN.md — Selection state foundation (Model fields, helper methods)
- [ ] 13-02-PLAN.md — Selection highlighting and Ctrl+A with clear on navigation
- [ ] 13-03-PLAN.md — Clipboard operations (Ctrl+X/C/V) with undo integration

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

### Phase 18: UI Polish

**Goal**: Refine the user interface with visual improvements and UX enhancements

**Depends on**: Phase 17 (all features tested and validated)

**Requirements**: None (polish phase, requirements TBD during planning)

**Success Criteria** (what must be TRUE):
1. Error messages are clearly visible and not clipped
2. Status area provides contextual information
3. Visual consistency across all UI states
4. Responsive feel during rapid input
5. Keyboard shortcuts are discoverable

**Plans**: TBD

---

## Progress

**Execution Order:** Phases execute in numeric order: 9 -> 9.1 -> 10 -> 11 -> 11.1 -> 11.2 -> 12 -> 13 -> 14 -> 15 -> 16 -> 17 -> 18

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 9. Interpreter Correctness | v1.1 | 4/4 | Complete | 2026-02-07 |
| 9.1 Separate Validation/Execution | v1.1 | 3/3 | Complete | 2026-02-07 |
| 10. Preview Pane | v1.1 | 5/5 | Complete | 2026-02-06 |
| 11. Navigation | v1.1 | 3/3 | Complete | 2026-02-07 |
| 11.1 Bug Fixes | v1.1 | 3/3 | Complete | 2026-02-07 |
| 11.2 UX Redesign | v1.1 | 3/3 | Complete | 2026-02-07 |
| 12. Undo/Redo | v1.1 | 3/3 | Complete | 2026-02-08 |
| 13. Clipboard | v1.1 | 3/3 | Complete | 2026-02-09 |
| 14. File Operations | v1.1 | 0/TBD | Not started | - |
| 15. Help Update | v1.1 | 0/TBD | Not started | - |
| 16. Source Highlighting | v1.1 | 0/TBD | Not started | - |
| 17. Testing & Validation | v1.1 | 0/TBD | Not started | - |
| 18. UI Polish | v1.1 | 0/TBD | Not started | - |

---

*Roadmap created: 2026-02-06*
*Last updated: 2026-02-08*
