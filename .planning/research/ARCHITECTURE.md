# Architecture Research

**Domain:** TUI Editor + CLI for an Interpreted Language (CalcMark)
**Researched:** 2026-02-02
**Confidence:** HIGH

## Standard Architecture

### System Overview

```
+-------------------------------------------------------------------+
|                       CLI Layer (cobra)                            |
|   root.go  eval.go  convert.go  edit.go  tui.go  version.go      |
+-----+-------------------+-------------------+--------------------+
      |                   |                   |
      v                   v                   v
+------------+   +-----------------+   +---------------+
|  eval cmd  |   |    TUI App      |   |  convert cmd  |
| (headless) |   | (App struct)    |   |  (headless)   |
+-----+------+   +--------+--------+   +-------+-------+
      |                   |                     |
      |          +--------+--------+            |
      |          |                 |            |
      |   +------v------+  +------v------+     |
      |   | REPL Model  |  |Editor Model |     |
      |   | (simple)    |  | (two-pane)  |     |
      |   +------+------+  +------+------+     |
      |          |                 |            |
      +----------+---------+------+            |
                           |                   |
                           v                   v
+-------------------------------------------------------------------+
|                   Shared Component Layer                           |
|  StatusBar  ContextFooter  Suggest  Globals  Pinned  SideBySide   |
+-------------------------------------------------------------------+
                           |
                           v
+-------------------------------------------------------------------+
|                     Pure Computation Layer                         |
|  AlignedModel  LineModel  Results  Geometry  WrapText  Markdown   |
+-------------------------------------------------------------------+
                           |
              +------------+------------+
              |                         |
              v                         v
+------------------------+   +------------------------+
|     spec/ (Language)   |   |  impl/ (Runtime)       |
| lexer, parser, ast,    |   | interpreter, evaluator |
| semantic, document,    |   | document, types, wasm  |
| types, units           |   |                        |
+------------------------+   +------------------------+
```

### Component Responsibilities

| Component | Responsibility | Typical Implementation |
|-----------|----------------|------------------------|
| **CLI Layer** (cobra) | Parse CLI args, dispatch to subcommands, load config | `cmd/calcmark/cmd/` -- thin routing layer |
| **App** | Top-level Bubble Tea model, mode switching (REPL/Editor) | `cmd/calcmark/tui/app.go` -- orchestrator |
| **REPL Model** | Single-line input, scrolling history, slash commands | `cmd/calcmark/tui/repl/` -- standalone model |
| **Editor Model** | Two-pane editor: source (left), results (right) | `cmd/calcmark/tui/editor/` -- the big one |
| **Shared Components** | Reusable pure-rendering components for any TUI mode | `cmd/calcmark/tui/components/` |
| **Pure Computation** | Geometry, alignment, wrapping -- no side effects | `aligned.go`, `linemodel.go`, `sidebyside.go` |
| **spec/** | Language grammar, AST, types, document model | Independent of runtime/UI |
| **impl/** | Interpreter, evaluator, environment, WASM bindings | Depends on spec, never on UI |
| **format/** | Output formatters (text, JSON, HTML, MD, CM) | Depends on spec, never on UI |

### Current Architecture Issues

The codebase has two editor implementations in flight:

1. **Model (v1)** -- `model.go` (53KB, ~1400 lines). Custom line-by-line rendering with manual cursor tracking, scroll management, and alignment. Well-tested with 50+ test files but architecturally bloated. Has `AlignedModel`, `LineModel`, `EditorState`, and complex block-level rendering logic all in one package.

2. **ModelV2** -- `model_v2.go` (18KB, ~666 lines). Uses `bubbles/textarea` for source editing. Simpler but less feature-complete. Currently wired into `app.go` as the active editor. Missing: undo/redo, slash commands, globals panel, search, most V1 features.

**The core problem:** V1 conflates computation and rendering. The `model.go` file handles cursor tracking, document mutation, evaluation orchestration, undo/redo, state machines, and rendering all in one struct with 40+ fields. This makes it hard to test individual concerns and causes cascading bugs when one area changes.

## Recommended Project Structure

```
cmd/
  calcmark/
    main.go                       # Entry point
    cmd/                          # Cobra commands (thin routing)
      root.go                     # REPL or file dispatch
      eval.go                     # Headless evaluation
      convert.go                  # Format conversion
      edit.go                     # Launch editor
      tui.go                      # Launch TUI
      version.go                  # Version info
    config/                       # Configuration loading
      config.go                   # Config struct and loading
      theme.go                    # Theme definitions
      types.go                    # Config types
    tui/                          # All TUI code
      app.go                      # Top-level model, mode switching
      shared/                     # Shared types, key bindings
        keys.go                   # Key bindings (centralized)
        state.go                  # Shared mode/state types
        messages.go               # Cross-component messages
      components/                 # Reusable pure-render components
        statusbar.go              # Status bar rendering
        contextfooter.go          # Context-sensitive footer
        suggest.go                # Autocomplete rendering
        globals.go                # Globals panel
        pinned.go                 # Pinned variables
        errors.go                 # Error display
      repl/                       # REPL mode
        model.go                  # REPL model (self-contained)
        view.go                   # REPL rendering
      editor/                     # Editor mode (NEEDS RESTRUCTURING)
        model.go                  # Core editor state + Update logic
        view.go                   # View rendering
        geometry/                 # NEW: Pure geometry computation
          aligned.go              # Aligned visual line computation
          linemodel.go            # Source-to-visual line mapping
          wrap.go                 # Text wrapping
          sidebyside.go           # Side-by-side pane rendering
          geometry_test.go        # All geometry is testable pure functions
        results.go                # Document -> LineResult bridge
        state.go                  # State machine transitions
        markdown.go               # Markdown preview rendering
        testdata/                 # Catwalk test data files
spec/                             # Language specification (NO impl deps)
  ast/                            # Abstract syntax tree nodes
  classifier/                     # Block classification
  document/                       # Document model (blocks, diagnostics)
  lexer/                          # Tokenizer
  parser/                         # Parser
  semantic/                       # Semantic analysis
  types/                          # Language type system
  units/                          # Unit definitions (canonical.go)
impl/                             # Runtime implementation (NO UI deps)
  interpreter/                    # Expression evaluator
  document/                       # Document-level evaluation, Environment
  types/                          # Runtime type operations
  wasm/                           # WebAssembly bindings
format/                           # Output formatters (NO UI deps)
  display/                        # Display formatting for values
  text_formatter.go
  json_formatter.go
  html_formatter.go
  markdown_formatter.go
  calcmark_formatter.go
```

### Structure Rationale

- **`editor/geometry/`:** The single most important structural change. All pure computation that produces visual line layouts, wrapping, and alignment should live here as pure functions. These functions take inputs (lines, widths, cursor position) and return outputs (visual lines, mappings). No side effects, no lipgloss, no tea.Model. Fully testable with table-driven tests.
- **`components/`:** Already well-structured. Pure rendering functions that take state structs and return strings. Keep this pattern.
- **`shared/`:** Cross-cutting types. Keep thin -- only types and messages used by multiple models.
- **Spec vs Impl separation:** Already correct and working. The `spec/` package defines the language. The `impl/` package executes it. Dependencies flow one way: `impl -> spec`. Never the reverse.

## Architectural Patterns

### Pattern 1: Pure Computation Core (Functional Core, Imperative Shell)

**What:** Separate pure computation from side-effectful Bubble Tea model logic. All geometry, alignment, and wrapping are pure functions. The Bubble Tea model is a thin shell that calls pure functions and wires up state.

**When to use:** Any computation that takes inputs and produces outputs without needing access to the terminal, file system, or Bubble Tea runtime.

**Trade-offs:** Slightly more files and types. Dramatically better testability. Worth it for anything beyond trivial logic.

**Example:**
```go
// Pure function -- fully testable, no dependencies on bubbletea/lipgloss
func CalculateRowGeometry(srcLine string, result string, leftW, rightW int) GeometryResult {
    leftWrapped := WrapText(srcLine, leftW)
    rightWrapped := WrapText(result, rightW)

    height := max(len(leftWrapped), len(rightWrapped))
    if height == 0 {
        height = 1
    }

    // Pad shorter side
    left := make([]string, height)
    right := make([]string, height)
    copy(left, leftWrapped)
    copy(right, rightWrapped)

    return GeometryResult{Height: height, LeftLines: left, RightLines: right}
}

// Imperative shell -- Bubble Tea model calls pure functions
func (m *EditorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // ... handle input, update state ...
    // Recompute geometry from pure functions
    m.geometry = CalculateRowGeometry(m.currentLine, m.result, m.leftWidth, m.rightWidth)
    return m, nil
}
```

**This pattern already exists in the codebase.** `ComputeAlignedModel()` in `aligned.go` is a pure function. `ComputeLineModel()` in `linemodel.go` is a pure function. `RenderContextFooter()` in `contextfooter.go` is a pure function. The recommendation is to formalize and extend this pattern to cover ALL editor geometry.

### Pattern 2: Data-Driven Testing with Catwalk

**What:** Use `knz/catwalk` on top of `cockroachdb/datadriven` for TUI model testing. Test files contain sequences of key presses and expected model state (via custom observers). Tests run without a terminal.

**When to use:** Every TUI behavior that involves key sequences and state transitions. This is the project's primary testing strategy for the editor.

**Trade-offs:** Test files are text-based and can be verbose. The `-rewrite` flag makes regeneration easy. Observer design determines what you can assert on.

**How CalcMark already uses it:**
```
# testdata/delete_empty_line
run observe=debug
key j
key j
----
-- debug:
mode=0 cursorLine=2 cursorCol=0 ... editBuf="" ...
```

**Critical insight:** Catwalk tests the MODEL, not the rendered output. The `debug` observer inspects `Model.Debug()`, which returns structured state. The `results` observer inspects `Model.GetLineResults()`, which returns evaluation results. This is fundamentally different from screenshot testing -- it tests behavior, not pixels.

**Recommendation:** Keep catwalk as the primary testing strategy. Supplement with:
- **Table-driven unit tests** for pure geometry functions (no catwalk needed)
- **Golden file tests** for View() output when visual fidelity matters
- **teatest** only for full-program integration tests (rare, heavy)

### Pattern 3: State Machine for Editor Modes

**What:** Explicit state transitions with invariant checking. The editor has states (Ready, Editing, Processing) with documented invariants and transition functions.

**When to use:** Any UI component with distinct behavioral modes and state that must remain consistent.

**Trade-offs:** More boilerplate for state transitions. Prevents entire classes of bugs where state becomes inconsistent.

**Example from codebase:**
```go
// state.go -- explicit transitions with invariant enforcement
func (m *Model) transitionToReady() {
    // INVARIANT: Document must exist with at least 1 block
    if m.doc == nil || len(m.doc.GetBlocks()) == 0 {
        m.doc, _ = document.NewDocument("\n")
    }
    // INVARIANT: Evaluator must exist
    if m.eval == nil {
        m.eval = implDoc.NewEvaluator()
        _ = m.eval.Evaluate(m.doc)
    }
    // INVARIANT: Cursor at valid position
    // ...
    m.state = StateReady
}
```

**Recommendation:** Keep this pattern. Extend it to the V2 model. The current V2 model lacks explicit state management, which will cause bugs as features are added.

### Pattern 4: Message-Passing for Cross-Component Communication

**What:** Components communicate via Bubble Tea messages, not direct method calls. The `SwitchModeMsg` pattern is already used for REPL-to-Editor switching.

**When to use:** Any communication between sibling or parent-child components that live in different packages.

**Trade-offs:** Slightly more indirection. Prevents tight coupling and makes components independently testable.

**Current examples:**
- `SwitchModeMsg` for mode switching
- `evalDebounceMsg` for debounced evaluation
- `saveResultMsg` for async save results

**Recommendation:** Extend to autocomplete, help system, and any new features. Each feature should define its own message types and handle them in its own Update logic.

## Data Flow

### Evaluation Flow (Source Text to Results)

```
User Types in Source Pane
    |
    v
editBuf updated (immediate, no evaluation)
    |
    v (50ms debounce)
syncDocumentFromTextarea()
    |
    v
specDoc.NewDocument(content)  -->  Parse into blocks (CalcBlock, TextBlock)
    |
    v
implDoc.NewEvaluator().Evaluate(doc)
    |
    v
For each CalcBlock:
    Lexer -> Parser -> Semantic -> Interpreter
    Results stored on CalcBlock.results[]
    Variables stored in Environment
    |
    v
GetLineResults()  -->  Bridge: document blocks -> []LineResult
    |
    v
ComputeAlignedModel(input)  -->  Pure geometry: visual lines for both panes
    |
    v
View()  -->  Render source pane + preview pane side-by-side
```

### Key Data Types in the Flow

| Stage | Type | Purpose |
|-------|------|---------|
| User input | `string` (editBuf) | Raw text being typed |
| Parsed | `*document.Document` | Structured blocks with AST |
| Evaluated | `*implDoc.Evaluator` (owns Environment) | Variable bindings, per-statement results |
| Bridged | `[]LineResult` | Per-line: source, value, error, diagnostics |
| Geometry | `AlignedModel` | Visual lines with alignment, wrapping, cursor |
| Rendered | `string` | Final terminal output from View() |

### State Management

```
EditorModel (single struct, all state)
    |
    +-- Document State: doc, eval, filepath, modified, savedContent
    |
    +-- Cursor State: cursorLine, cursorCol, scrollOffset
    |
    +-- Edit State: state (Ready/Editing/Processing), editBuf, userIsTyping
    |
    +-- UI State: width, height, previewMode, quitting
    |
    +-- Cache: alignedCache, alignedCacheKey (invalidated on any input change)
    |
    +-- Undo/Redo: undoStack, redoStack (content snapshots)
    |
    +-- Features: search, export, save, globals, pinnedVars
```

**The V1 model has ~40 fields.** This is the primary reason for the rewrite. The recommendation is NOT to replicate all 40 fields in V2 but to:
1. Let `textarea.Model` own cursor, scrolling, undo/redo, and text content
2. Keep the editor model focused on: document sync, evaluation, and feature state
3. Move geometry computation to pure functions that take computed inputs

### Key Data Flows

1. **Typing -> Preview:** User types in textarea -> content change detected -> document reparsed -> re-evaluated -> geometry recomputed -> preview pane re-rendered. Debounced at 50ms to avoid excess computation.
2. **Navigation -> Alignment:** User scrolls/navigates -> textarea updates viewport -> geometry extracts visible range -> preview pane aligns to visible source lines only.
3. **Mode Switch:** User types `/edit` in REPL -> `SwitchModeMsg` dispatched -> App creates Editor with current document -> Editor opens with same variable state.

## Scaling Considerations

This is a CLI tool, not a server. "Scaling" means handling large documents efficiently.

| Scale | Architecture Adjustments |
|-------|--------------------------|
| 0-100 lines | Current approach works fine. Full re-evaluation on every change is fast enough. |
| 100-1000 lines | Need incremental evaluation. Only re-evaluate changed blocks, not entire document. The `DependencyAnalyzer` in spec already tracks cross-block dependencies. |
| 1000+ lines | Need virtual viewport. Only compute geometry for visible lines (currently ~24 rows). The `AlignedModel` should only compute visible range. Need lazy evaluation for off-screen blocks. |

### Scaling Priorities

1. **First bottleneck: Re-evaluation latency.** Currently the entire document is reparsed and re-evaluated on every keystroke (after debounce). For documents > 100 lines, this will become noticeable. Fix: `EvaluateBlock()` already exists in `impl/document` for incremental evaluation. Wire it up.
2. **Second bottleneck: Geometry computation.** `ComputeAlignedModel()` iterates all lines to compute visual layout. For large documents, compute only the visible viewport window. Cache alignment for off-screen lines.
3. **Third bottleneck: Memory for undo.** The V1 model stores full document content snapshots for undo/redo. For large documents, consider operational transform or diff-based undo. The V2 textarea has built-in undo which avoids this.

## Anti-Patterns

### Anti-Pattern 1: Monolith Model Struct

**What people do:** Put all state -- cursor, document, evaluation, UI, undo, search, export, save -- in a single struct with 40+ fields.
**Why it's wrong:** Every feature interacts with every other feature. Adding search affects undo. Adding export affects evaluation. Changes cascade. Tests must construct the full state to test any single feature.
**Do this instead:** Compose the model from smaller, independent state structs. Each struct owns one concern. The top-level model delegates to sub-structs. The V1 `Model` is a cautionary example with 40+ fields.

### Anti-Pattern 2: Rendering Inside Update

**What people do:** Compute visual layout (lipgloss styling, line wrapping) inside `Update()` or store pre-rendered strings in the model.
**Why it's wrong:** View computation in Update couples rendering to state management. Pre-rendered strings become stale. Width changes invalidate cached renders but the invalidation logic is scattered.
**Do this instead:** `Update()` should only update logical state (cursor position, document content). `View()` should call pure geometry functions and apply styling. If geometry is expensive, cache the pure computation result (e.g., `AlignedModel`), not the rendered string.

### Anti-Pattern 3: Testing View Output Directly

**What people do:** Assert on the exact string output of `View()`, including ANSI escape codes and lipgloss styling.
**Why it's wrong:** View output is fragile -- it changes with terminal width, color profile, lipgloss version, and theme settings. Tests break constantly for cosmetic reasons. Tests are unreadable because they're full of escape codes.
**Do this instead:** Test the model state through observers (catwalk `debug`, `results`). Test pure geometry functions with table-driven tests. Use `View()` golden tests only for critical visual fidelity checks, and set `lipgloss.SetColorProfile(termenv.Ascii)` to strip ANSI codes.

### Anti-Pattern 4: Custom Cursor Management with textarea

**What people do:** Use `textarea.Model` for editing but also maintain a parallel cursor position in the editor model, leading to synchronization bugs.
**Why it's wrong:** Two sources of truth for cursor position. The textarea has its own cursor. The editor model has `cursorLine`/`cursorCol`. When they disagree, rendering breaks.
**Do this instead:** Let textarea be the single source of truth for cursor position. Use `textarea.Line()` and `textarea.CursorPosition()` to read cursor state. Do not store parallel cursor state in the editor model. The V2 model correctly does this.

### Anti-Pattern 5: Synchronous Evaluation in Update

**What people do:** Call `eval.Evaluate(doc)` directly inside `Update()`, blocking the UI until evaluation completes.
**Why it's wrong:** For complex documents with unit conversions, evaluation can take >50ms, causing visible UI lag.
**Do this instead:** Use Bubble Tea `Cmd` to run evaluation asynchronously. Send the result back as a message. The debounce pattern already exists (`evalDebounceMsg`). Extend it: debounce -> start async eval Cmd -> receive `EvaluationDoneMsg` -> update results. This is the pattern from `code.sh`.

## Integration Points

### Internal Boundaries

| Boundary | Communication | Key Consideration |
|----------|---------------|-------------------|
| **CLI -> TUI** | Direct function call (`NewApp`, `NewEditorApp`) | CLI creates document, passes to TUI. Config loaded before TUI starts. |
| **App -> REPL/Editor** | Bubble Tea message delegation + `SwitchModeMsg` | App.Update delegates to active model. Document shared via `Document()` method. |
| **Editor -> Interpreter** | `spec/document.NewDocument()` + `impl/document.Evaluator` | Editor owns the evaluator. Re-creates on content change. Must NOT hold references across re-parses. |
| **Editor -> Components** | Pure function calls with state structs | Components never hold editor state. They receive state and return rendered strings. |
| **Editor -> Geometry** | Pure function calls with `AlignedModelInput` | Geometry functions never access editor state directly. All inputs are explicit. |
| **spec/ -> impl/** | `impl/` imports `spec/` types | One-way dependency. `spec/` defines, `impl/` executes. |
| **format/ -> spec/** | `format/` imports `spec/` types | One-way dependency. Formatters read document structure. |
| **WASM -> impl/** | WASM wraps interpreter API | No TUI code in WASM. Shares `spec/` and `impl/` only. |

### Dependency Flow (Critical Invariant)

```
spec/  <--  impl/  <--  format/
  ^           ^            ^
  |           |            |
  +-----+----+-----+------+
        |          |
      cmd/calcmark/tui/     (depends on all three)
        |
      cmd/calcmark/cmd/     (depends on tui/ and config/)
```

**Rule:** `spec/` NEVER imports from `impl/`, `format/`, or `cmd/`. This is enforced by convention and tested by successful compilation. Violating this breaks WASM builds and language spec independence.

## Testing Strategy

### Recommended Testing Pyramid

```
                    +---+
                   /     \
                  / VHS   \        <- Visual smoke tests (manual, rare)
                 / tapes   \
                +-----------+
               /             \
              /   Catwalk     \    <- Behavioral integration tests (primary for TUI)
             /  (data-driven)  \
            +-------------------+
           /                     \
          /    Table-driven       \  <- Unit tests for pure functions (fast, many)
         /   geometry, wrapping,   \
        /    alignment, results     \
       +-----------------------------+
      /                               \
     /         spec/ + impl/           \ <- Language tests (separate concern)
    /    lexer, parser, interpreter     \
   +-------------------------------------+
```

### Layer 1: Pure Function Unit Tests (Fastest, Most Numerous)

Test geometry, wrapping, alignment as pure functions with table-driven tests.

```go
func TestCalculateRowGeometry(t *testing.T) {
    tests := []struct {
        name     string
        srcLine  string
        result   string
        leftW    int
        rightW   int
        wantH    int
        wantLeft []string
        wantRight []string
    }{
        {
            name:      "short source, long result wraps",
            srcLine:   "Short",
            result:    "1234567890",
            leftW:     10,
            rightW:    5,
            wantH:     2,
            wantLeft:  []string{"Short", ""},
            wantRight: []string{"12345", "67890"},
        },
        // ... more cases
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := CalculateRowGeometry(tt.srcLine, tt.result, tt.leftW, tt.rightW)
            // assertions
        })
    }
}
```

**What to test this way:**
- `WrapText()` -- all edge cases (empty, exact width, unicode, ANSI codes)
- `ComputeAlignedModel()` -- alignment correctness for various document structures
- `ComputeLineModel()` -- visual line layout
- `GetLineResults()` -- document-to-result bridging
- `FilterSuggestions()` -- autocomplete matching
- Any new geometry function

### Layer 2: Catwalk Behavioral Tests (Primary for TUI)

Test editor behavior through key sequences and state observation. Confidence: HIGH -- this is already proven in the codebase with 12+ test data files.

**When to use catwalk:**
- Navigation behavior (arrow keys, page up/down, home/end)
- Editing behavior (typing, backspace, enter, delete line)
- Evaluation correctness after editing (results observer)
- State transitions (mode switching, slash commands)
- Regression tests for every user-reported bug

**When NOT to use catwalk:**
- Pure geometry computation (use table-driven tests)
- Visual styling (use golden files or manual VHS)
- Performance (use benchmarks)

### Layer 3: Golden File Tests (For Visual Fidelity)

The existing `golden_e2e_test.go` tests CLI output against golden files in `testdata/`. This pattern works well for:
- `eval` command output format
- `convert` command output format
- Document serialization round-tripping

**Not recommended for:** Editor View() output. Terminal rendering is too fragile for golden files.

### Layer 4: VHS Tapes (Manual Visual Smoke Tests)

VHS tapes in `testdata/vhs_tapes/` produce screenshots. These are:
- Run manually (`task test:vhs`)
- Not part of CI (too slow, requires VHS installation)
- Useful for verifying visual themes and layout

**Recommendation:** Keep VHS tapes for manual verification. Do NOT make them part of the automated test suite. They are inherently flaky across terminal emulators and screen sizes.

### Why NOT teatest

`teatest` (from `charmbracelet/x/exp/teatest`) runs a full `tea.Program` in a headless terminal. It is useful for testing program-level behavior (startup, shutdown, full rendering pipeline). However:

- It is experimental (lives in `x/exp`)
- It is heavier than catwalk (spins up the full event loop)
- It tests rendered output (fragile) rather than model state (stable)
- Catwalk already covers the model-testing niche better

**Use teatest only if:** You need to test something that catwalk cannot, such as Cmd execution or full program lifecycle. For CalcMark, catwalk is sufficient and preferred.

## Build Order Implications

The following dependency chain dictates build order for new features:

```
1. Pure geometry functions (no dependencies, test immediately)
   |
2. Editor model (depends on geometry, spec, impl)
   |
3. Components (depend on model state types, not model itself)
   |
4. View rendering (depends on geometry + components + lipgloss)
   |
5. Integration (wire into app.go, cobra commands)
```

**For the TUI editor rewrite:**
1. Extract geometry into pure functions with comprehensive tests
2. Build the new editor model on top of textarea + geometry
3. Add features incrementally (evaluation, alignment, slash commands, undo, search)
4. Each feature: pure function first, wire into model, add catwalk test

**For the help system:**
1. Define help content storage (static strings or embedded files)
2. Build help rendering as a pure component (like ContextFooter)
3. Add help navigation model (simple list/tree with viewport)
4. Wire into app.go as a new mode

**For autocomplete:**
1. Build suggestion engine as a pure function (takes prefix + available names, returns matches)
2. The `SuggestionSource` interface already exists in `components/suggest.go`
3. Add suggestion rendering component (already exists: `RenderSuggestions`, `RenderDropdownSuggestions`)
4. Wire into editor model: detect context, query suggestion engine, render dropdown

## Bubble Tea v2 Consideration

Bubble Tea v2 is in RC stage as of late 2025. Key changes:

| Change | Impact on CalcMark |
|--------|--------------------|
| `View()` returns `tea.View` struct instead of `string` | Major refactor of all View methods |
| New module path (`charm.land/bubbletea/v2`) | Import path changes everywhere |
| New mouse API (split into Click/Release/Wheel/Motion) | Minor -- CalcMark has minimal mouse support |
| Declarative view API | Potentially simplifies alt-screen management |

**Recommendation:** Stay on Bubble Tea v1 (`v0.21.0+` / `v1.3.10`) for the v1 milestone. The v2 API is not yet stable (RC, not final). Migrating mid-rewrite adds risk. Plan a dedicated v2 migration as a separate milestone after the editor rewrite is stable.

## Sources

- CalcMark codebase analysis (HIGH confidence -- direct code reading)
- [Bubble Tea GitHub](https://github.com/charmbracelet/bubbletea) (HIGH confidence)
- [knz/catwalk](https://github.com/knz/catwalk) -- Test library for Bubbletea TUI models (HIGH confidence)
- [Tips for building Bubble Tea programs](https://leg100.github.io/en/posts/building-bubbletea-programs/) (MEDIUM confidence)
- [Bubble Tea State Machine pattern](https://zackproser.com/blog/bubbletea-state-machine) (MEDIUM confidence)
- [Writing Bubble Tea Tests (teatest)](https://charm.land/blog/teatest/) (MEDIUM confidence)
- [Bubble Tea v2 Migration Guide](https://github.com/charmbracelet/bubbletea/discussions/1374) (HIGH confidence)
- [Bubble Tea Component Architecture Discussion](https://github.com/charmbracelet/bubbletea/discussions/286) (MEDIUM confidence)
- [Testing Bubble Tea Interfaces](https://patternmatched.substack.com/p/testing-bubble-tea-interfaces) (LOW confidence -- single blog post)
- code.sh prototype in repository (HIGH confidence -- direct code reading)

---
*Architecture research for: CalcMark TUI Editor and CLI*
*Researched: 2026-02-02*
