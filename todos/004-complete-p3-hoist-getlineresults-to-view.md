---
status: complete
priority: p3
issue_id: "004"
tags: [code-review, performance]
dependencies: []
---

# Hoist GetLineResults to View() Top Level

## Problem Statement

`GetLineResults()` is currently called 2x per `View()` frame (down from 3x after recent fix). It could be hoisted to the top of `View()` and passed as a parameter to sub-renderers, reducing it to 1 call per frame.

## Findings

- Source: performance-oracle agent
- `GetLineResults()` builds the full line result slice on each call
- Called from `renderContextFooter` and line rendering
- Pre-computing once per frame would be a ~50% reduction in remaining calls

## Proposed Solutions

### Option A: Hoist results computation to View()
- **Pros:** 50% fewer GetLineResults() calls, cleaner data flow
- **Cons:** Slightly wider function signatures for sub-renderers
- **Effort:** Medium
- **Risk:** Low

## Acceptance Criteria

- [ ] `GetLineResults()` called exactly once per `View()` frame
- [ ] Results passed as parameter to all sub-renderers that need it
- [ ] No change in visual output
- [ ] `task test` passes
