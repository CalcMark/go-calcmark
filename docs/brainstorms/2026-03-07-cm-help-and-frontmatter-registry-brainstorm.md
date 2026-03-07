---
topic: cm help restructuring and frontmatter registry
date: 2026-03-07
status: decided
---

# cm help Restructuring and Frontmatter Registry

## What We're Building

Two related changes:

1. **Restructure CLI help** — Move `cm functions` and `cm constants` under a `cm help` subcommand. Add a new `cm help frontmatter` topic that documents all frontmatter directives with valid options, unit categories, and YAML examples.

2. **Frontmatter registry** — Add frontmatter directives to the existing feature registry (`spec/features/registry.go`) so docgen can generate structured documentation for them, just like it does for functions and units.

## Why This Approach

- Nowhere in the CLI or docs can a user discover the valid `unit_categories` values for `scale` or `convert_to`. They have to read source code.
- `cm functions` and `cm constants` are good but they're orphaned root commands. Grouping under `cm help` creates a single discoverable entry point.
- The feature registry already feeds docgen and the Hugo feature-table shortcode. Adding frontmatter to it means generated docs come for free.

## Key Decisions

### CLI Shape: `cm help <topic>`

`cm functions` and `cm constants` become `cm help functions` and `cm help constants`. New topic: `cm help frontmatter`.

Both subcommand and flag forms work:
- `cm help functions` = `cm help --functions`
- `cm help` with no args = `cm help --all` (shows everything)

**Backwards compat:** Remove old `cm functions` and `cm constants` root commands. Pre-1.0, clean break is fine.

### Frontmatter Registry Location: `spec/features/registry.go`

Add a new category `CategoryFrontmatter = "frontmatter"` to the existing feature registry. Each directive (exchange, globals, scale, convert_to) becomes a Feature with:
- Name, description, syntax (YAML examples)
- Sub-options (factor, unit_categories, system) as part of the description or a new field
- Valid values (si/imperial for systems, derived category list for unit_categories)

### Unit Categories: Derived from Code

Scan `spec/units/canonical.go`'s `StandardUnits` for unique `Quantity` field values at init time. Also include `DataSize` from `spec/units/conversion.go`. This prevents drift — categories always match what the unit system supports.

A function like `units.Categories() []string` returns the sorted, deduplicated list.

### Output Format: Verbose with Examples

`cm help frontmatter` shows each directive with:
- Description
- YAML examples (scalar and map forms)
- Valid sub-keys and their types
- Available unit categories (derived)
- Default behavior

Example output:
```
## scale
Multiply quantity results by a factor.

  scale: 2
  scale:
    factor: 4
    unit_categories: [Length, Mass]

Categories: Length, Mass, Volume, Temperature,
  Speed, Power, Energy, Area, DataSize
Default: all except Temperature

## convert_to
Convert to a measurement system.

  convert_to: si
  convert_to:
    system: imperial
    unit_categories: [Length]

Systems: si, imperial
```

## Scope of Changes

### Code Changes
- `cmd/calcmark/cmd/help.go` — Restructure as `cm help` with subcommands + flags
- `spec/features/registry.go` — Add `CategoryFrontmatter` and frontmatter features
- `spec/units/canonical.go` or `spec/units/units.go` — Add `Categories() []string` function
- `cmd/docgen/main.go` — Already consumes the registry; frontmatter will appear in features.json

### Documentation Updates
- `site/content/docs/cli-reference.md` — Update command reference
- `site/content/docs/agent-integration.md` — Update `cm functions` → `cm help functions`
- `site/content/docs/user-guide.md` — Update references to `cm functions` and `cm constants`
- `site/content/docs/language-reference.md` — Update references
- `AGENTS.md` — Update `cm functions` and `cm constants` references
- `CLAUDE.md` — Update if it references these commands

### Hugo Site
- `site/data/features.json` — Will now include frontmatter category
- Feature-table shortcode — Should render frontmatter section on language reference page

## Open Questions

None — all decisions resolved during brainstorming.
