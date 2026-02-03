# Phase 3: TUI Editor Integration - Context

**Gathered:** 2026-02-03
**Status:** Ready for planning

<domain>
## Phase Boundary

Make the two-column editor fully interactive with accurate cursor tracking, smooth scrolling, working evaluation pipeline, and results that update as the user types. The geometry/layout from Phase 2 is solid — this phase wires up interactivity.

**In scope:** Cursor movement, viewport scrolling, live evaluation, model unification (ModelV2 → Model rename), dirty state tracking, basic editing operations.

**Out of scope:** Undo/redo (Phase 4), text selection (later), autocomplete (Phase 6).

</domain>

<decisions>
## Implementation Decisions

### Cursor behavior in wrapped text
- Arrow keys move by **logical lines only** — down arrow jumps to next line number, not next visual line within a wrap
- Home/End go to **logical line bounds** — Home = column 0, End = end of full line content
- Left/right arrows **wrap to adjacent lines** — left at column 0 goes to end of previous line, right at end goes to start of next
- Cursor is **steady, no blinking** — helps with screen recording and accessibility

### Scrolling & viewport sync
- Both columns **lock by logical line** — logical line N always aligned between source and results, even if visual heights differ
- Viewport adjustment when cursor goes off-screen: **Claude's discretion** (minimal scroll, centering, or adaptive)
- Scroll margin (cursor distance from edges): **Claude's discretion**
- Page Up/Down support: **Claude's discretion**

### Live evaluation timing
- Results update on a **~100ms debounce** after typing pauses
- Debounce value **configurable in code** (constant, not user-facing) for tuning
- During evaluation, **keep previous results** visible — no flicker, no loading indicator
- Non-calculation lines (prose, headings) show **nothing/blank** in results pane

### Error display
- Errors show **both**: indicator (X) in source pane AND full diagnostic message in results/preview pane
- User sees at a glance which lines have errors, and full context in the preview

### Model unification
- **Rename ModelV2 → Model**, rename current Model → ModelOld
- Migrate features from ModelOld to Model **only if needed**
- Delete ModelOld at **end of Phase 3**
- Model contains document, cursor, viewport state together — **single source of truth**
- State accessed through **methods only** (Elm-style, pure functions where possible) — better testability
- Model is **Bubble Tea native** (implements tea.Model) but keep REPL mode in mind
- Model tracks **dirty state** (unsaved changes)
- **Manual save only** (Ctrl+S) — no auto-save

### Quit behavior
- Quit with **Ctrl+Q** (not Esc)
- **Prompt before quit** if unsaved changes — "Unsaved changes. Save? (y/n/cancel)"

### Text editing operations
- Enter **splits line at cursor position** — standard text editor behavior
- Backspace at start of line **joins with previous line**
- Delete key support: **Claude's discretion**
- Tab key: Use existing interpreter whitespace handling — must parse correctly as CalcMark, **UTF-8 compatible**
- **Ctrl+arrow keys move by word** — standard word boundaries (whitespace/punctuation as separators)
- Word deletion (Ctrl+Backspace/Delete): **Claude's discretion** given word boundary complexity in calculations

### Claude's Discretion
- Viewport scroll behavior (minimal vs center vs adaptive)
- Scroll margin amount
- Page Up/Down implementation
- Delete key behavior
- Ctrl+Backspace/Delete word deletion
- Exact debounce tuning within ~100ms baseline

</decisions>

<specifics>
## Specific Ideas

- Interpreter should process large documents in nanoseconds, so 100ms debounce is conservative and may be tuned lower
- "I like Elm architecture and pure functions" — methods-only state access fits this philosophy
- Keep REPL mode in mind even though editor is Bubble Tea native
- Must use existing interpreter libraries for whitespace/UTF-8 — no naive implementations

</specifics>

<deferred>
## Deferred Ideas

- **Undo/redo** — explicitly moved to Phase 4
- **Text selection** (shift+arrows, copy/cut) — not in Phase 3
- **Token-aware word movement** in calculations — standard word boundaries for now

</deferred>

---

*Phase: 03-tui-editor-integration*
*Context gathered: 2026-02-03*
