# Flat Line Buffer for TUI Source Pane

**Date:** 2026-02-26
**Status:** Ready for planning

## What We're Building

Decouple the source pane's text buffer from the spec layer's document model. Instead of every keystroke flowing through `updateCurrentLine` -> `NewDocument()`, the source pane will:

1. Maintain a flat `[]string` buffer (all lines, including frontmatter, treated identically)
2. Only sync to the document model when evaluation is needed (debounce timer, navigation, save)
3. Eliminate all `if cursorLine < fmCount` branches from editing operations

Replace the current single-line `editBuf` cache with this full `[]string` line buffer that becomes the source of truth for all editing. The document model (`m.doc`) becomes a derived artifact, rebuilt from the buffer at sync points rather than being mutated on every keystroke.

**Principle:** The YAML should NOT be treated any differently than other text in the source pane.

## Why This Approach

### The Problem

Today, every mutation point in the editing pipeline (`updateCurrentLine`, `deleteLine`, `handleEnterKey`, `DeleteSelection`) branches on whether the cursor is in the frontmatter region:

- **Frontmatter path:** Full `document.NewDocument()` rebuild because the spec layer treats frontmatter as a single parsed YAML unit. Includes `SetRawSource()` fallback for invalid YAML.
- **Body path:** Targeted `ReplaceBlockSource()` update on a single block.

This creates 9+ locations across 4 files with `if cursorLine < fmCount` branching, each with its own error handling for YAML parse failures. The complexity cascades into undo/redo, which must handle document state changes from rebuilds (line count changes, embedded newlines splitting).

### The Fix

A `[]string` buffer decouples editing from the document model entirely:

1. **Editing** operates on the buffer (insert rune at position, split line, join lines, delete range). No branching. No document rebuilds. No YAML error handling.
2. **Syncing** happens at debounce boundaries: `strings.Join(buffer, "\n")` -> `NewDocument(content)`. One code path, always a full rebuild.
3. **Rendering** reads from the buffer for the source pane. The document model provides block types for color-coding and the preview pane.

## Key Decisions

### 1. Sync Trigger: Same debounce pattern
Keep the existing 100ms debounce timer + navigation sync points. The `[]string` buffer replaces `editBuf` as the source of truth during editing. The document model only updates when the debounce fires or the user navigates away.

### 2. Undo Model: Keep fine-grained operations
Keep the current `EditOperation` types (`OpInsert`, `OpDelete`, `OpReplace`, `OpInsertLine`, `OpDeleteLine`) but record and apply them against the `[]string` buffer instead of the document model. The operations stay the same, just the target changes. `OpDocReplace` may become unnecessary since the buffer handles frontmatter insertion as regular line operations.

### 3. Initialization: Document -> Buffer on load
On file open, parse the document normally via `NewDocument()`, then materialize all lines into the `[]string` buffer. From that point forward, the buffer is the editing source of truth. The document model is rebuilt from buffer content at sync points.

### 4. Sync Method: Always full rebuild
On sync: `strings.Join(lines, "\n")` -> `NewDocument(content)`. One code path, no branching between frontmatter and body. `redetectBlockTypes()` already does exactly this today. The debounce means it runs at most ~10 times/sec.

### 5. Text Fidelity: Preserve exact user text
The buffer holds exactly what the user typed. No YAML normalization or reformatting. The document model parses the buffer's exact text at sync time. This eliminates the current `Serialize()` round-trip which can reorder keys or normalize whitespace.

## What Changes

### Eliminated
- `editBuf` / `editBufLoaded` (replaced by `lines []string` + `cursorLine` index)
- All `if cursorLine < fmCount` branches in `updateCurrentLine`, `deleteLine`, `handleEnterKey`, `DeleteSelection`
- `frontmatterErr` tracking and `SetRawSource()` fallback paths in the editor (YAML errors surface only at sync time)
- `loadCurrentLineIntoEditBuffer()` / `transitionToEditing()` (buffer is always loaded)
- `saveCurrentLineAndMoveTo()` complexity (no per-line flush needed; just move cursor)
- The critical ordering constraint: "redetectBlockTypes MUST run before loadCurrentLineIntoEditBuffer"

### Simplified
- `GetLines()` becomes a simple accessor returning `m.lines` (no reconstruction)
- `TotalLines()` becomes `len(m.lines)`
- Navigation (up/down/page) just changes `cursorLine` index — no flush/reload cycle
- Undo apply/reverse operates on `m.lines` directly — no `updateCurrentLine()` indirection
- `transitionToProcessing()` becomes: join lines, `NewDocument()`, evaluate. One path.

### Preserved
- View layer reads block types from document model for color-coding (unchanged)
- Preview pane reads evaluation results from document model (unchanged)
- Undo operation types and batching logic (unchanged, just retargeted)
- Debounce timer pattern (unchanged)
- `redetectBlockTypes()` full rebuild (already exists, becomes the only sync path)

## What Gets Harder

- **Buffer/document divergence during typing:** While the user types, the buffer is ahead of the document model. The view must read from the buffer for source pane content and from the document model for block types/colors. This is already the case today with `editBuf` substitution, but it extends to all lines.
- **YAML error feedback:** Currently `frontmatterErr` surfaces immediately on each keystroke. With the buffer approach, YAML errors only surface at sync time (100ms debounce). This is likely fine — the user won't notice 100ms delay on error feedback.
- **File save must sync first:** Before saving, the buffer must be synced to the document model. This is already the pattern today (`transitionToProcessing()` before save).

## Scope

### In Scope
- New `lines []string` field replacing `editBuf`/`editBufLoaded`
- Rewrite of mutation operations to target `lines` directly
- Rewrite of `transitionToProcessing()` as the single sync path
- Rewrite of undo apply/reverse to target `lines` directly
- Updated catwalk tests and unit tests

### Out of Scope
- Changes to the spec layer (`document.NewDocument`, `Frontmatter`, block types)
- Changes to the preview pane or evaluation pipeline
- Changes to the undo batching/grouping logic
- Performance optimization of the full rebuild (already fast enough with debounce)
