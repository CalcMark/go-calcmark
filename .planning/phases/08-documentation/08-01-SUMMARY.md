---
phase: 08-documentation
plan: 01
subsystem: docs
tags: [examples, screenshots, gif, vhs, visual-assets]

# Dependency graph
requires:
  - phase: 07-distribution
    provides: working cm binary for demos
provides:
  - Three focused example files in testdata/examples/
  - TUI screenshot showing two-column layout
  - Hero GIF demonstrating cm eval workflow
  - VHS tape file for reproducible GIF generation
affects: [08-02 (README content), 08-03 (quick-start)]

# Tech tracking
tech-stack:
  added: [vhs]
  patterns: [example-driven documentation, reproducible visual assets]

key-files:
  created:
    - testdata/examples/budget.cm
    - testdata/examples/unit-conversion.cm
    - testdata/examples/engineering.cm
    - docs/images/tui-screenshot.png
    - docs/images/hero.gif
    - docs/images/hero.tape
  modified: []

key-decisions:
  - "Hero GIF uses cm eval workflow (not TUI) because VHS cannot capture interactive TUI apps"
  - "VHS tape runs from project root using ./cm for local binary"
  - "Engineering example uses YAML front matter to demonstrate constants feature"

patterns-established:
  - "Visual assets in docs/images/ with VHS tapes for reproducibility"
  - "Examples in testdata/examples/ with focused single-concept files"

# Metrics
duration: 5min
completed: 2026-02-05
---

# Phase 8 Plan 01: Visual Assets and Examples Summary

**Three calcmark example files and visual assets including TUI screenshot and 14-second hero GIF**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-05T20:15:00Z
- **Completed:** 2026-02-05T20:22:00Z
- **Tasks:** 3
- **Files modified:** 6

## Accomplishments
- Created three focused example files: budget.cm (finance), unit-conversion.cm (measurements), engineering.cm (capacity with YAML front matter)
- Captured TUI screenshot showing source/results two-column layout
- Generated reproducible hero GIF using VHS tape (88KB, 14.5 seconds)

## Task Commits

Each task was committed atomically:

1. **Task 1: Create three focused example files** - `c47ae6f` (docs)
2. **Task 2: Capture screenshot** - User action (no commit - screenshot only)
3. **Task 3: Create VHS tape and generate hero GIF** - `18b071a` (feat)

## Files Created/Modified
- `testdata/examples/budget.cm` - Monthly budget calculation example
- `testdata/examples/unit-conversion.cm` - Distance/weight/temperature conversion example
- `testdata/examples/engineering.cm` - Capacity planning with YAML front matter constants
- `docs/images/tui-screenshot.png` - Static screenshot of TUI two-column layout
- `docs/images/hero.gif` - Animated GIF showing cm eval workflow (88KB)
- `docs/images/hero.tape` - VHS tape file for reproducible GIF generation

## Decisions Made
- Used `cm eval` workflow for hero GIF instead of TUI because VHS cannot properly capture interactive TUI applications that take over the terminal
- VHS tape uses `./cm` (local binary path) since `cm` is not in system PATH
- Engineering example includes YAML front matter to demonstrate the constants feature

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] VHS Require directive failed**
- **Found during:** Task 3 (VHS tape execution)
- **Issue:** `Require cm` failed because cm binary not in PATH
- **Fix:** Changed tape to use `./cm` (local binary) and removed Require directive
- **Files modified:** docs/images/hero.tape
- **Verification:** VHS runs successfully from project root
- **Committed in:** 18b071a

**2. [Rule 1 - Bug] Hero GIF showed shell typing not TUI**
- **Found during:** Task 3 (VHS tape execution)
- **Issue:** Original tape tried to type into TUI after launching cm, but VHS sends keystrokes to shell, not interactive TUI
- **Fix:** Changed demo to use `cm eval` workflow showing file creation and evaluation output
- **Files modified:** docs/images/hero.tape
- **Verification:** GIF shows complete workflow with calculation output
- **Committed in:** 18b071a

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 bug)
**Impact on plan:** Both fixes necessary for VHS compatibility. Demo still effectively shows CalcMark value proposition.

## Issues Encountered
- VHS cannot capture interactive TUI applications that take over the terminal - worked around by using eval workflow which still demonstrates calculation capabilities

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Visual assets ready for README.md integration (08-02)
- Example files can be referenced in quick-start guide (08-03)
- All verification checks pass

---
*Phase: 08-documentation*
*Completed: 2026-02-05*
