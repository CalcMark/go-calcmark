# Status Area Redesign Plan

## Problem
The status area currently grows dynamically when showing detailed information (function signatures, errors, etc.), which pushes the main panes up in a jarring way. This disrupts the user's visual focus.

## Design Goals
1. **Fixed geometry**: Main panes should never shift due to status content
2. **Progressive disclosure**: Brief info always visible, details on demand
3. **Context-appropriate**: Show info where the user is already looking

## Proposed Solution: Hybrid Approach

### Option A: Fixed Toast + Preview Pane Details (Recommended)

**Status Bar (always 1-2 lines fixed height):**
- Line 1: Filename, position, mode indicators
- Line 2: Brief hint/error (truncated with "..." if too long)

**Preview Pane Integration:**
- When cursor is on a function call, show signature in preview pane header
- When there's an error, show full error message in preview area
- Preview pane already exists and doesn't shift layout

**Benefits:**
- Zero layout shift
- Natural place for detailed info (preview is for "results")
- User's eyes already scan between source and preview

### Option B: Expandable Tooltip

**Implementation:**
- Floating tooltip positioned near cursor (like autocomplete popup)
- Shows function signature, error details, etc.
- Auto-dismisses after timeout or on keystroke
- Pure geometry calculation (reuse popup positioning)

**Benefits:**
- Info appears exactly where user is looking
- Doesn't take permanent screen space

### Option C: Collapsible Status Footer

**Implementation:**
- Status area has fixed 2-line minimum
- Shows expand indicator (▼) when more content available
- User presses key (e.g., Ctrl+I) to toggle expanded view
- Expanded view overlays content (doesn't push it)

## Implementation Plan

### Phase 1: Fixed Status Bar Height
1. Set status bar to fixed 2-line height
2. Truncate long messages with "..." indicator
3. Store full message for detail view

### Phase 2: Preview Pane Integration
1. Add "context header" to preview pane
2. Show function signature when cursor on function call
3. Show full error when line has error
4. Style to distinguish from result content

### Phase 3: Optional Tooltip (Future)
1. Reuse popup positioning from autocomplete
2. Create tooltip component for hover-style info
3. Triggered by dwell time or explicit key

## Files to Modify
- `cmd/calcmark/tui/components/statusbar.go` - Fixed height
- `cmd/calcmark/tui/components/contextfooter.go` - Truncation logic
- `cmd/calcmark/tui/editor/view.go` - Preview pane context header
- `cmd/calcmark/tui/geometry/popup.go` - Reuse for tooltips (Phase 3)

## Success Criteria
- [ ] Main panes never shift vertically during editing
- [ ] User can access full error/hint details when needed
- [ ] No loss of functionality from current status area
