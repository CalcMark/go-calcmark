# Roadmap: CalcMark v1

## Overview

CalcMark v1 delivers a polished, distributable calculation notepad for the terminal. The journey starts by fixing broken infrastructure and extracting the pure geometry layer, then rebuilds the TUI editor on a correct two-column foundation, layers on help and discoverability, adds differentiating features (autocomplete and YAML front matter), and finishes with release packaging and documentation. Every phase delivers a verifiable capability that the next phase builds upon.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Foundation** - Fix CI, update deps, extract pure geometry package
- [ ] **Phase 2: TUI Geometry & Layout** - Implement code.sh algorithm for correct two-column rendering
- [ ] **Phase 3: TUI Editor Integration** - Cursor, scrolling, model integration, live evaluation
- [ ] **Phase 4: TUI Test Coverage** - Comprehensive catwalk tests, eliminate flakey video tests
- [ ] **Phase 5: Help System** - CLI help commands, shell completions, TUI help overlay and status bar
- [ ] **Phase 6: Differentiators** - TUI autocomplete and YAML front matter for document constants
- [ ] **Phase 7: Distribution** - GoReleaser, cross-platform binaries, Homebrew, Scoop, WASM
- [ ] **Phase 8: Documentation** - README, examples, generated references, screenshots

## Phase Details

### Phase 1: Foundation
**Goal**: Project builds and tests pass on CI with current Go, dependencies are updated, and pure geometry functions are extracted and fully tested in isolation
**Depends on**: Nothing (first phase)
**Requirements**: FOUND-01, FOUND-02, FOUND-03, FOUND-04, FOUND-05
**Success Criteria** (what must be TRUE):
  1. `task test` passes on CI with Go version matching go.mod (1.24.x)
  2. A `geometry` package exists with pure functions that take dimensions and content as input and return row layout data as output, with zero TUI framework dependencies
  3. `CalculateRowGeometry` correctly computes visual line counts for both columns when text wraps, verified by unit tests covering single-line, multi-wrap, and asymmetric wrap scenarios
  4. `task quality` passes with all dependencies at current stable versions (including adrg/frontmatter added to go.mod)
**Plans:** 2 plans

Plans:
- [x] 01-01-PLAN.md -- Fix CI workflow, update Go to 1.24.12, update deps, fix pre-existing test failures
- [x] 01-02-PLAN.md -- Extract pure geometry package with WrapText and CalculateRowGeometry, wire into editor

### Phase 2: TUI Geometry & Layout
**Goal**: The two-column editor renders source and results side-by-side with correct text wrapping and vertical alignment under all scenarios
**Depends on**: Phase 1 (geometry package)
**Requirements**: EDITOR-01, EDITOR-02, EDITOR-03, EDITOR-04
**Success Criteria** (what must be TRUE):
  1. Opening a .cm file in the TUI shows source on the left and results on the right with no overlapping or misaligned text
  2. Typing a long line in the source pane wraps cleanly at the column boundary without bleeding into the results pane
  3. A result that is wider than the right pane wraps independently without pushing other rows down
  4. When source line N wraps to 3 visual lines and result line N wraps to 1 visual line, both still start at the same vertical position (padding fills the gap)
  5. Resizing the terminal reflows both columns correctly with no rendering artifacts
**Plans:** 2 plans

Plans:
- [ ] 02-01-PLAN.md -- Write integration tests proving all five success criteria for the two-column layout
- [ ] 02-02-PLAN.md -- Fix any layout failures and visually verify two-column rendering

### Phase 3: TUI Editor Integration
**Goal**: The editor is fully interactive with accurate cursor tracking, smooth scrolling, working evaluation pipeline, and results that update as the user types
**Depends on**: Phase 2 (correct layout rendering)
**Requirements**: EDITOR-05, EDITOR-06, EDITOR-07, EDITOR-08, EDITOR-09, EDITOR-10
**Success Criteria** (what must be TRUE):
  1. Cursor is visible at all times and moves to the correct position after each keystroke, including within wrapped lines
  2. Scrolling a document longer than the viewport shows the correct portion of both source and results, with the cursor always visible
  3. Typing `= 2 + 2` on a new line shows `4` in the corresponding results pane within 200ms
  4. Editing a line that defines a variable causes all dependent results to update (e.g., changing `tax = 10%` updates lines using `tax`)
  5. The model layer (geometry + state management) is unified -- there is one code path for computing layout, not separate paths in model and view
**Plans**: TBD

Plans:
- [ ] 03-01: Cursor positioning and visibility in wrapped text
- [ ] 03-02: Viewport scrolling with two-column synchronization
- [ ] 03-03: Debounced evaluation and results rendering pipeline

### Phase 4: TUI Test Coverage
**Goal**: Every editor interaction has a catwalk test, and the CI pipeline contains zero flakey tests
**Depends on**: Phase 3 (editor functionally complete)
**Requirements**: EDITOR-11, EDITOR-12
**Success Criteria** (what must be TRUE):
  1. Catwalk tests exist for: typing text, cursor movement (arrows, home/end, page up/down), text wrapping at narrow widths (40 columns), scrolling through long documents, and evaluation results appearing
  2. No VHS tape or video-based tests remain in CI -- all TUI testing uses catwalk data-driven tests
  3. `task test` completes with zero flakey failures across 10 consecutive runs
**Plans**: TBD

Plans:
- [ ] 04-01: Comprehensive catwalk test suite and flakey test removal

### Phase 5: Help System
**Goal**: Users can discover all CalcMark features through CLI help commands, shell completions, an in-TUI help overlay, and an informative status bar
**Depends on**: Phase 3 (stable editor for TUI help integration)
**Requirements**: HELP-01, HELP-02, HELP-03, HELP-04, HELP-05, HELP-06, HELP-07, HELP-08
**Success Criteria** (what must be TRUE):
  1. Running `cm help functions` prints a complete list of all functions with descriptions and English synonyms, and the output works correctly when piped to `less`
  2. Running `cm help constants` prints all built-in constants from the canonical unit registry
  3. Shell completions work: typing `cm ` and pressing Tab in bash/zsh shows available subcommands
  4. Pressing the help key in the TUI editor shows an overlay listing all keybindings, and pressing it again dismisses the overlay
  5. The status bar at the bottom of the TUI shows cursor position (line:col), calculation count, and displays "EVAL..." while evaluation is in progress
**Plans**: TBD

Plans:
- [ ] 05-01: CLI help commands and shell completions
- [ ] 05-02: TUI help overlay and status bar

### Phase 6: Differentiators
**Goal**: Users can define document-level constants via YAML front matter and discover functions/constants/units through TUI autocomplete while typing
**Depends on**: Phase 3 (stable evaluation pipeline), Phase 5 (help infrastructure for consistent keybinding management)
**Requirements**: HELP-09, HELP-10, HELP-11, HELP-12, YAML-01, YAML-02, YAML-03, YAML-04, YAML-05
**Success Criteria** (what must be TRUE):
  1. A .cm file with `---\ntax_rate: 0.08\n---` at the top makes `tax_rate` available in calculations, and `= price * tax_rate` produces the correct result
  2. Malformed YAML front matter shows a clear error message indicating the problem and line number
  3. Pressing Tab or Ctrl+Space after typing `= av` in the editor shows a dropdown with matching functions (e.g., `average`, `availability`)
  4. The autocomplete dropdown shows English synonyms (e.g., typing `= mean` suggests `average (mean)`)
  5. Selecting an autocomplete suggestion inserts it correctly at the cursor position without disrupting surrounding text
**Plans**: TBD

Plans:
- [ ] 06-01: YAML front matter parsing and evaluation integration
- [ ] 06-02: TUI autocomplete engine and popup rendering

### Phase 7: Distribution
**Goal**: Users can install CalcMark on macOS, Linux, and Windows via a single command or prebuilt binary download, with all release artifacts signed and verified
**Depends on**: Phases 1-6 (all features complete)
**Requirements**: DIST-01, DIST-02, DIST-03, DIST-04, DIST-05, DIST-06, DIST-07, DIST-08, DIST-09, DIST-10, DIST-11, DIST-12
**Success Criteria** (what must be TRUE):
  1. Running `brew install <tap>/calcmark` on macOS installs a working `cm` binary that can open and evaluate .cm files
  2. Downloading the Linux arm64 tarball from GitHub releases, extracting it, and running `./cm --version` prints the correct version
  3. The Windows amd64 zip from GitHub releases contains a working cm.exe
  4. Every release artifact on GitHub has a corresponding SHA256 checksum file
  5. The WASM binary is included in GitHub releases and is under 3MB uncompressed
**Plans**: TBD

Plans:
- [ ] 07-01: GoReleaser configuration and CI release workflow
- [ ] 07-02: Platform packaging (Homebrew tap, Scoop bucket, man pages, shell completions)
- [ ] 07-03: WASM build optimization and release inclusion

### Phase 8: Documentation
**Goal**: A new user can understand what CalcMark is, install it, and start using it within 5 minutes by reading the README and exploring example files
**Depends on**: Phases 1-7 (features and distribution complete, so documentation reflects final product)
**Requirements**: DOCS-01, DOCS-02, DOCS-03, DOCS-04, DOCS-05, DOCS-06, DOCS-07, DOCS-08, DOCS-09, DOCS-10
**Success Criteria** (what must be TRUE):
  1. The README clearly explains CalcMark as "Jupyter notebooks but simpler -- human-readable .cm files with embedded calculations" with installation instructions for all three platforms
  2. The README includes a quick-start example that a user can copy-paste and run immediately
  3. At least 3 example .cm files exist in testdata/ with verified correct calculations covering different use cases (budget, unit conversion, engineering)
  4. Running `cm help functions` and `cm help constants` produces output that matches the generated reference documentation
  5. The README contains at least one screenshot showing the TUI editor with source and results side by side
**Plans**: TBD

Plans:
- [ ] 08-01: README with installation, quick start, and screenshots
- [ ] 08-02: Example .cm files and generated reference documentation
- [ ] 08-03: Contributing guide and CLI reference

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2 -> 3 -> 4 -> 5 -> 6 -> 7 -> 8

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Foundation | 2/2 | Complete | 2026-02-02 |
| 2. TUI Geometry & Layout | 0/2 | Not started | - |
| 3. TUI Editor Integration | 0/3 | Not started | - |
| 4. TUI Test Coverage | 0/1 | Not started | - |
| 5. Help System | 0/2 | Not started | - |
| 6. Differentiators | 0/2 | Not started | - |
| 7. Distribution | 0/3 | Not started | - |
| 8. Documentation | 0/3 | Not started | - |

---
*Roadmap created: 2026-02-02*
*Last updated: 2026-02-02*
