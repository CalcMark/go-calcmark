# Docs Consolidation Brainstorm

**Date:** 2026-02-23
**Status:** Complete
**Scope:** Consolidate documentation into site/, clean up docs/ for planning-only use

---

## What We're Building

A clean separation of concerns for CalcMark documentation:

- **`site/`** = All user-facing documentation (the canonical source for end users)
- **`docs/`** = Developer planning only (brainstorms, plans, solutions)
- **`testdata/examples/`** = Canonical `.cm` example source files
- **`scripts/`** = Build tooling (VHS .tape scripts for demo GIF generation)

This eliminates stale duplicates, establishes clear ownership, and prevents future doc sprawl.

---

## Why This Approach

The current state has significant duplication and staleness:

| File | Problem |
|------|---------|
| `CONFIG.md` (root) | Uses deprecated `dark_mode` setting; `site/content/docs/configuration.md` is current |
| `docs/README.md` | References REPL commands that no longer exist; `site/content/docs/user-guide.md` supersedes |
| `docs/images/` | Byte-for-byte duplicates of `site/static/images/` (6 files, ~1.7MB) |
| `docs/examples/*.cm` | Embedded verbatim in `site/content/docs/examples/*.md` |
| `docs/THEMING.md` | Developer design spec, not user-facing — belongs with planning docs |
| `format/OUTPUT_FORMATTERS.md` | Developer design doc, not user-facing — belongs with planning docs |
| `ARCHITECURE_FUNCTIONS.md` | Planning doc for future functions — belongs with planning docs |

A single clean sweep is the right approach — this is structural reorganization, not feature work. One atomic pass avoids broken intermediate states.

---

## Key Decisions

1. **Delete stale root docs** — `CONFIG.md` and `docs/README.md` are superseded by site content. Delete outright.

2. **Move planning/design docs to `docs/plans/`:**
   - `docs/THEMING.md` → `docs/plans/2026-02-22-design-tui-theme-specification.md`
   - `format/OUTPUT_FORMATTERS.md` → `docs/plans/2026-02-22-design-output-formatters.md`
   - `ARCHITECURE_FUNCTIONS.md` → `docs/plans/2026-02-22-design-architecture-functions.md`

3. **Move examples to `testdata/examples/`** — The 5 `.cm` files are test fixtures that also feed site content. `testdata/examples/` already exists with other content; these will coexist.

4. **Delete duplicate images from `docs/images/`** — `site/static/images/` is authoritative. Keep the `.tape` VHS scripts (they generate the GIFs).

5. **Move `.tape` scripts to `scripts/`** — They're build tooling, not documentation content.

6. **Update README.md image paths** — Point to `site/static/images/` using relative paths.

7. **Update Taskfile.yml** — The `vhs` task references `docs/images/*.tape` for input and `docs/images/*.gif` for output. Must update to `scripts/*.tape` and `site/static/images/*.gif`.

8. **Update AGENTS.md** — References `OUTPUT_FORMATTERS.md`; update to new path in `docs/plans/`.

9. **Update CONTRIBUTING.md** — Contains `[OUTPUT_FORMATTERS.md](OUTPUT_FORMATTERS.md)` link; update to new path.

10. **Update site example paths** — 6 site content files reference `cm docs/examples/<name>.cm`; update to `cm testdata/examples/<name>.cm`.

11. **Add `docs/README.md`** — A brief file explaining the new structure and that user-facing docs live in `site/`.

12. **Leave historical references alone** — `.planning/` and existing `docs/plans/` files reference old paths as point-in-time records. Do not update these.

---

## File Movement Summary

### Deletions
- `CONFIG.md` (root)
- `docs/README.md`
- `docs/images/*.gif` and `docs/images/*.png` (5 GIFs + 1 PNG = 6 files)
- `docs/images/` directory itself (empty after moves and deletes)

### Moves
| From | To |
|------|----|
| `docs/THEMING.md` | `docs/plans/2026-02-22-design-tui-theme-specification.md` |
| `format/OUTPUT_FORMATTERS.md` | `docs/plans/2026-02-22-design-output-formatters.md` |
| `ARCHITECURE_FUNCTIONS.md` | `docs/plans/2026-02-22-design-architecture-functions.md` |
| `docs/examples/*.cm` (5 files) | `testdata/examples/*.cm` |
| `docs/images/*.tape` (5 files) | `scripts/*.tape` |

### Updates
- `README.md` — Update image paths from `docs/images/` to `site/static/images/`
- `Taskfile.yml` — Update `vhs` task paths from `docs/images/` to `scripts/` (input) and `site/static/images/` (output)
- `AGENTS.md` — Update `OUTPUT_FORMATTERS.md` reference to new path in `docs/plans/`
- `CONTRIBUTING.md` — Update `OUTPUT_FORMATTERS.md` link to new path
- `site/content/docs/examples/_index.md` — Update `docs/examples/` → `testdata/examples/`
- `site/content/docs/examples/*.md` (5 files) — Update `cm docs/examples/` → `cm testdata/examples/`
- `site/content/docs/getting-started.md` — Update `cm eval docs/examples/` → `cm eval testdata/examples/`

### New Files
- `docs/README.md` — Explains the docs/ directory structure (planning-only)

### Not Updated (Historical Records)
- `.planning/phases/08-documentation/*` — Point-in-time phase records
- `docs/plans/2026-02-23-feat-hero-tui-video-recording-plan.md` — References old paths but documents a past plan
- `docs/brainstorms/*` (except the consolidation brainstorm itself) — Historical records

### No Changes Needed
- `site/` structure — Already well-organized with current content
- `docs/brainstorms/` — Already correctly placed
- `docs/plans/` — Already correctly placed (receiving new files)
- `docs/solutions/` — Already correctly placed
- `.planning/` — Separate project management tree, not affected

---

## Open Questions

None — all decisions resolved during brainstorming.
