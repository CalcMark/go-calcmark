---
title: "Consolidate docs into site/ and clean up docs/ for planning-only use"
type: refactor
status: completed
date: 2026-02-23
brainstorm: docs/brainstorms/2026-02-23-docs-consolidation-brainstorm.md
---

# Consolidate Documentation Structure

## Overview

Establish a clean separation: `site/` owns all user-facing docs, `docs/` is strictly for developer planning (brainstorms, plans, solutions). Delete stale duplicates, move planning docs, and update all references.

## Problem Statement

Documentation is scattered across root-level files, `docs/`, `format/`, and `site/` with stale duplicates and broken ownership. `CONFIG.md` uses deprecated settings, `docs/README.md` references a REPL that no longer exists, and `docs/images/` duplicates `site/static/images/` byte-for-byte.

## Acceptance Criteria

- [x] No user-facing documentation exists outside `site/`
- [x] `docs/` contains only brainstorms, plans, and solutions
- [x] All planning/design docs live in `docs/plans/`
- [x] Example `.cm` source files live in `testdata/examples/`
- [x] VHS `.tape` scripts live in `scripts/`
- [x] No broken references in README.md, AGENTS.md, CONTRIBUTING.md, Taskfile.yml, or site content
- [x] `task build` and `task test` pass
- [x] `hugo --source site` builds with zero errors

## Implementation

### Phase 1: Delete Stale Docs

- [x] Delete `CONFIG.md` (root) — superseded by `site/content/docs/configuration.md`
- [x] Delete `docs/README.md` — superseded by `site/content/docs/user-guide.md`

### Phase 2: Move Planning/Design Docs to `docs/plans/`

- [x] `docs/THEMING.md` → `docs/plans/2026-02-22-design-tui-theme-specification.md`
- [x] `format/OUTPUT_FORMATTERS.md` → `docs/plans/2026-02-22-design-output-formatters.md`
- [x] `ARCHITECURE_FUNCTIONS.md` → `docs/plans/2026-02-22-design-architecture-functions.md`

### Phase 3: Move Examples to `testdata/examples/`

Move 5 `.cm` files (coexisting with the 3 already there: budget.cm, unit-conversion.cm, engineering.cm):

- [x] `docs/examples/household-budget.cm` → `testdata/examples/household-budget.cm`
- [x] `docs/examples/job-offer.cm` → `testdata/examples/job-offer.cm`
- [x] `docs/examples/project-workback.cm` → `testdata/examples/project-workback.cm`
- [x] `docs/examples/recipe-scaling.cm` → `testdata/examples/recipe-scaling.cm`
- [x] `docs/examples/system-sizing.cm` → `testdata/examples/system-sizing.cm`
- [x] Remove empty `docs/examples/` directory

### Phase 4: Consolidate Images and Tapes

- [x] Move `.tape` files to `scripts/`:
  - `docs/images/hero.tape` → `scripts/hero.tape`
  - `docs/images/feature-variables.tape` → `scripts/feature-variables.tape`
  - `docs/images/feature-eval.tape` → `scripts/feature-eval.tape`
  - `docs/images/feature-export.tape` → `scripts/feature-export.tape`
  - `docs/images/feature-autocomplete.tape` → `scripts/feature-autocomplete.tape`
- [x] Delete duplicate images from `docs/images/` (6 files: 5 GIFs + 1 PNG)
- [x] Remove empty `docs/images/` directory

### Phase 5: Update References

**README.md** (lines 5, 66):
```markdown
# Before
![CalcMark TUI](docs/images/hero.gif)
![CalcMark TUI](docs/images/tui-screenshot.png)

# After
![CalcMark TUI](site/static/images/hero.gif)
![CalcMark TUI](site/static/images/tui-screenshot.png)
```

**AGENTS.md** (line 30):
```markdown
# Before
Look at OUTPUT_FORMATTERS.md for details and ./format for implementation.

# After
Look at docs/plans/2026-02-22-design-output-formatters.md for details and ./format for implementation.
```

**CONTRIBUTING.md** (line 198):
```markdown
# Before
- See [OUTPUT_FORMATTERS.md](OUTPUT_FORMATTERS.md) for output format details

# After
- See [Output Formatters Design](docs/plans/2026-02-22-design-output-formatters.md) for output format details
```

**Taskfile.yml** `record-demos` task (lines 244-262):
```yaml
# Before: docs/images/*.tape → docs/images/*.gif → cp to site/static/images/
# After: scripts/*.tape → site/static/images/*.gif (direct output, no cp step)
record-demos:
  desc: Regenerate all demo GIFs for the website
  cmds:
    - vhs scripts/hero.tape
    - vhs scripts/feature-variables.tape
    - vhs scripts/feature-eval.tape
    - vhs scripts/feature-export.tape
    - echo '✓ All demo GIFs regenerated to site/static/images/'
  sources:
    - scripts/*.tape
  generates:
    - site/static/images/hero.gif
    - site/static/images/feature-variables.gif
    - site/static/images/feature-eval.gif
    - site/static/images/feature-export.gif
```

Note: The `.tape` files themselves need their `Output` directives updated from `docs/images/*.gif` to `site/static/images/*.gif` so VHS writes directly to the site. This eliminates the `cp` step entirely.

**Site content files** (7 files):
- `site/content/docs/examples/_index.md` — update `docs/examples/` → `testdata/examples/`
- `site/content/docs/examples/household-budget.md` — update path
- `site/content/docs/examples/job-offer.md` — update path
- `site/content/docs/examples/project-workback.md` — update path
- `site/content/docs/examples/recipe-scaling.md` — update path
- `site/content/docs/examples/recipe-scaling.md` — update path
- `site/content/docs/examples/system-sizing.md` — update path
- `site/content/docs/getting-started.md` — update `cm eval docs/examples/` → `cm eval testdata/examples/`

### Phase 6: Add docs/README.md

Create a new `docs/README.md` explaining the structure:

```markdown
# docs/

This directory contains **developer planning documents only**. User-facing documentation lives at [calcmark.org](https://calcmark.org) (source: `site/`).

## Structure

- `brainstorms/` — Feature exploration and design decisions
- `plans/` — Implementation plans and design specifications
- `solutions/` — Post-mortem records for solved problems
```

### Phase 7: Verify

- [x] `task test` passes
- [x] `task build` passes
- [x] `hugo --source site` builds with zero errors
- [x] No broken links in README.md (images render on GitHub)
- [x] `task record-demos` references correct paths (scripts/*.tape)

## Not Updated (Historical Records)

These files reference old paths but are point-in-time records — do not modify:
- `.planning/phases/08-documentation/*`
- `docs/plans/2026-02-23-feat-hero-tui-video-recording-plan.md`
- `docs/brainstorms/*` (other than the consolidation brainstorm)

## References

- Brainstorm: `docs/brainstorms/2026-02-23-docs-consolidation-brainstorm.md`
- Hugo site structure: `docs/solutions/infrastructure/hugo-site-structure-from-scratch.md`
- Active site docs cleanup plan: `docs/plans/2026-02-22-feat-site-docs-deep-clean-plan.md`
