---
title: "feat: Deep clean site documentation"
type: feat
status: active
date: 2026-02-22
---

# Site Documentation Deep Clean

## Overview

The CalcMark documentation site is significantly out of sync with the codebase. REPL commands that don't exist are documented, 10 of 12 functions are missing from the language reference, CLI subcommands/flags are undocumented, and features like `over`, `as napkin`, and NL syntax forms have no documentation at all. This plan delivers an accurate, complete, and maintainable documentation site.

## Problem Statement

Users hitting the site encounter:
- REPL commands (`:open`, `:save`, `:pin`, etc.) that produce errors because they don't exist
- A functions table missing most functions (only `avg` and `sqrt` in the language reference)
- No documentation for `cm convert`, `cm help`, `--color-mode`, or `cm eval -v`
- Zero coverage of `over`, `as napkin`, NL syntax (`read...from`, `compress...using`, etc.)
- A `capacity()` reference that should be `requires()` (or vice versa -- naming mismatch between registries)

## Proposed Solution

### Architecture

Two data sources feed the documentation:

1. **`spec/features/registry.go`** -- The features registry with Name, Category, Syntax, Description, Aliases (Parseable flag), and Example. This is the richer catalog and should be the primary source for the docgen tool.
2. **`impl/interpreter/functions.go`** -- `BuiltinFunctions` with Name, Synonyms, Description, Signature, Category. Used by `cm help functions`.

The docgen tool reads from `spec/features/registry.go` types (imported as a Go package) and writes `site/data/features.json`. Hugo templates consume this data file to render function tables. The JSON is NOT version-controlled; it's generated as a dependency of `site:build`.

Additionally, features not in the registry (like `over`, `as napkin`) get manually-authored sections in the language reference, each with an HTML anchor ID for deep linking.

### Implementation Phases

#### Phase 1: Fix the features registry (code changes)

Add missing entries to `spec/features/registry.go`:

- [x] **Resolve `requires` vs `capacity` naming first** -- the interpreter (`functions.go`) uses `capacity` as the primary name, and the registry (`registry.go`) uses `requires`. Since `capacity` is what the interpreter dispatches on (i.e., what users actually type), verify which name works in input by checking the parser and eval dispatch. Update both registries to use the same canonical name. This gates all downstream documentation.
- [x] Add `over` to `getKeywords()`: `{Name: "over", Category: CategoryKeyword, Syntax: "rate over duration", Description: "Accumulate a rate over time", Example: "100 MB/s over 1 day"}`
- [x] Add `napkin` / `as napkin` to the registry as a keyword: `{Name: "as napkin", Category: CategoryKeyword, Syntax: "expression as napkin", Description: "Round to 2 significant figures for estimates", Example: "432000 MB as napkin"}`
- [x] Add `at` keyword to the registry (used in capacity `at` syntax: `X at Y per Z`): `{Name: "at", Category: CategoryKeyword, Syntax: "X at Y per Z", Description: "Capacity planning with rate and buffer", Example: "1 TB at 100 MB/s per day"}`
- [x] Run `task test` to validate no regressions

**Source files:**
- `spec/features/registry.go:507-543` (getKeywords)
- `impl/interpreter/functions.go:119-125` (capacity FunctionDef)
- `spec/parser/capacity_at_test.go` (capacity `at` syntax tests)

#### Phase 2: Build the docgen tool

Create `cmd/docgen/main.go` -- a Go program that:

- [ ] Imports `spec/features` package
- [ ] Creates a `features.NewRegistry()`
- [ ] Iterates all features, grouped by category
- [ ] Outputs `site/data/features.json` with structure:

```json
{
  "functions": [
    {
      "name": "avg",
      "category": "function",
      "syntax": "avg(a, b, c, ...)",
      "description": "Calculate the average of numbers",
      "aliases": [
        {"name": "average", "parseable": false},
        {"name": "average of", "parseable": true}
      ],
      "example": "avg(10, 20, 30) -> 20",
      "anchor": "avg"
    }
  ],
  "keywords": [...],
  "operators": [...],
  "units": [...],
  "network": [...],
  "storage": [...],
  "compression": [...]
}
```

- [ ] Each entry gets an `anchor` field (kebab-cased name) for deep linking
- [ ] Add Taskfile entries:

```yaml
generate-docs:
  desc: Generate site data from feature registry
  cmds:
    - go run ./cmd/docgen
  sources:
    - spec/features/registry.go
  generates:
    - site/data/features.json

site:build:
  desc: Build the website
  deps:
    - generate-docs
  cmds:
    - hugo --source site --minify
    - echo 'Site built to site/public/'

site:
  desc: Start Hugo dev server for the website
  deps:
    - generate-docs
  cmds:
    - hugo server --source site --buildDrafts
```

- [ ] Add `site/data/` to `.gitignore` (generated, not checked in)
- [ ] Write a test for docgen that verifies the JSON output has expected structure
- [ ] Add a cross-registry completeness check: the docgen test should verify that every function in `impl/interpreter/functions.go` BuiltinFunctions has a corresponding entry in `spec/features/registry.go` (prevents future drift)

**New files:**
- `cmd/docgen/main.go`
- `cmd/docgen/main_test.go`

**Modified files:**
- `Taskfile.yml:217-226` (add generate-docs task, add deps to site tasks)
- `.gitignore` (add site/data/)

#### Phase 3: Create Hugo templates for generated data

Create a Hugo shortcode that renders function tables from generated data:

- [ ] Create `site/layouts/shortcodes/feature-table.html` -- renders features from `.Site.Data.features` grouped by a specified category, with anchor IDs on each entry
- [ ] Shortcode interface: `{{< feature-table category="function" >}}` -- the `category` param selects which group from the JSON to render
- [ ] Each function/keyword renders with an `id` attribute for deep linking (e.g., `<h4 id="avg">`)
- [ ] Include: name, syntax, description, aliases (marking parseable ones with a label), example
- [ ] Verify the shortcode works with a minimal test page before using it in Phase 4/5 content

**New files:**
- `site/layouts/shortcodes/feature-table.html`

#### Phase 4: Rewrite user-guide.md

Major rewrite of `site/content/docs/user-guide.md`:

- [ ] **Remove entire REPL Commands section** (lines 7-32) -- REPL mode is gone
- [ ] **Remove keyboard shortcuts section** (lines 25-31) tied to REPL
- [ ] **Add Editor Shortcuts section** using data from `cmd/calcmark/tui/editor/command_menu.go:24-51`:

| Category | Shortcut | Action |
|----------|----------|--------|
| File | Ctrl+S | Save document |
| File | Ctrl+O | Open file |
| File | Ctrl+E | Export to format |
| File | Ctrl+Q | Quit editor |
| Edit | Ctrl+Z | Undo |
| Edit | Ctrl+Y | Redo |
| Edit | Ctrl+K | Delete current line |
| Edit | Ctrl+F | Insert frontmatter |
| View | Ctrl+P | Toggle preview mode |
| Navigation | Opt+Left/Right | Word navigation |
| Navigation | Ctrl+Home/End | Document start/end |
| Navigation | Ctrl+D/U | Half-page scroll |
| Help | F1 | Full help |

- [ ] **Replace functions table** (lines 148-158) with the `feature-table` shortcode for functions
- [ ] **Remove `cm eval --json` reference** (line 57) -- flag doesn't exist. Replace with `cm convert budget.cm --to=json > results.json`
- [ ] **Remove Output Formats section** (lines 33-58) that references `:save` and `:output` commands. Replace with a brief section about `cm convert`
- [ ] **Fix `capacity()` reference** (line 155) to match the reconciled naming from Phase 1
- [ ] **Remove Tips section** (lines 208-244) -- it still references REPL commands like `:help`, `:vars`, etc. Replace with editor-relevant tips
- [ ] **Add `over` keyword examples** in the Rates section
- [ ] **Add `as napkin` section** explaining napkin math

**Modified files:**
- `site/content/docs/user-guide.md`

#### Phase 5: Rewrite language-reference.md as comprehensive spec

Major update to `site/content/docs/language-reference.md`:

- [ ] **Update Reserved Keywords section** (lines 299-340) to include ALL keywords: `over`, `napkin`, `from`, `at`, `per`, function names (all 12, not just avg/sqrt)
- [ ] **Replace Functions section** (lines 343-367) with comprehensive coverage using `feature-table` shortcode, plus hand-written prose for each category:
  - Math: `avg`, `sqrt`, `accumulate`
  - Conversion: `convert_rate`
  - Network: `downtime`, `rtt`, `throughput`, `transfer_time`
  - Storage: `read`, `seek`, `compress`
  - Capacity: `capacity`/`requires`
- [ ] **Add Natural Language Syntax section** documenting all parseable NL forms:
  - `average of X, Y, Z` -- alias for `avg(X, Y, Z)`
  - `square root of X` -- alias for `sqrt(X)`
  - `read X from Y` -- alias for `read(X, Y)`
  - `compress X using Y` -- alias for `compress(X, Y)`
  - `transfer X across Y Z` -- alias for `transfer_time(X, Y, Z)`
  - `X at Y per Z [with N% buffer]` -- alias for `capacity(X, Y, Z[, N%])`
  - `rate over duration` -- alias for `accumulate(rate, duration)`
  - `rate per unit` -- context-dependent: rate conversion
- [ ] **Add `as napkin` section** with anchor `#as-napkin`:
  - Syntax: `expression as napkin`
  - Behavior: rounds to 2 significant figures, normalizes units, adds `~` prefix
  - Works with: Number, Quantity, Currency, Duration, Rate
  - Example: `432000 MB as napkin` -> `~400 GB`
- [ ] **Add Rates section** documenting rate literals (`100 MB/s`, `$50/hour`) and `over` keyword
- [ ] **Add Date Arithmetic section** documenting date literals, duration math, `from` keyword
- [ ] **Add anchor IDs to every function and keyword** for deep linking (e.g., `## avg {#avg}`, `## as napkin {#as-napkin}`)
- [ ] **Update Type System section** to include Quantity, Rate, Duration types (currently only Number, Currency, Boolean)

**Modified files:**
- `site/content/docs/language-reference.md`

#### Phase 6: Create CLI Reference page

New page `site/content/docs/cli-reference.md`:

- [ ] Document all subcommands with flags, sourced from cobra definitions:

**`cm [file]`** (root)
- `--color-mode` auto|light|dark -- Color mode override

**`cm eval [file.cm]`**
- `-v, --verbose` -- Show all intermediate values
- Reads from file or stdin

**`cm convert <file.cm>`**
- `-t, --to` (required) -- Output format: html, md, json, text, cm
- `-o, --output` -- Write to file instead of stdout
- `-T, --template` -- Custom Go template (html only)

**`cm help [topic]`**
- Topics: `functions`, `constants`

**`cm version`**

**`cm completion [bash|zsh|fish|powershell]`**

- [ ] Add to Hugo sidebar menu in `site/hugo.yaml` with weight 35 (between Language Reference and Configuration)

**New files:**
- `site/content/docs/cli-reference.md`

**Modified files:**
- `site/hugo.yaml` (add cli-reference to docs menu)

#### Phase 7: Validate and fix examples

Run all 5 example `.cm` files through `cm eval` and fix failures:

- [ ] `cm eval docs/examples/household-budget.cm`
- [ ] `cm eval docs/examples/job-offer.cm`
- [ ] `cm eval docs/examples/project-workback.cm` -- known issue: `risk_buffer` undefined on line 110
- [ ] `cm eval docs/examples/recipe-scaling.cm`
- [ ] `cm eval docs/examples/system-sizing.cm`
- [ ] Fix any failures (e.g., define `risk_buffer` in project-workback.cm)
- [ ] Sync the `.cm` file content with the corresponding `site/content/docs/examples/*.md` page
- [ ] Review getting-started.md for minor CLI accuracy fixes

**Modified files:**
- `docs/examples/project-workback.cm` (fix undefined variable)
- `site/content/docs/examples/project-workback.md` (sync fix)
- `site/content/docs/getting-started.md` (minor fixes if needed)
- `site/content/docs/configuration.md` (review for accuracy)

#### Phase 8: Final validation

- [ ] Run `task test` -- all tests pass
- [ ] Run `task quality` -- lint/vet pass
- [ ] Run `task generate-docs` -- produces valid JSON
- [ ] Run `task site:build` -- Hugo builds without errors
- [ ] Manually check deep links work: `/docs/language-reference/#avg`, `#as-napkin`, `#over`, etc.

## Acceptance Criteria

- [ ] No documentation references non-existent features (REPL commands, `--json` flag, etc.)
- [ ] All 12 functions documented with syntax, description, aliases, and examples
- [ ] All NL syntax forms documented
- [ ] `over` and `as napkin` are documented and deep-linkable
- [ ] Every function/keyword has an anchor ID for deep linking
- [ ] CLI Reference page covers all subcommands and flags
- [ ] Editor TUI shortcuts replace stale REPL shortcuts
- [ ] All example `.cm` files evaluate without errors
- [ ] `features.json` is generated at build time, not version-controlled
- [ ] `task site:build` succeeds end-to-end (generate-docs -> hugo)
- [ ] `task test` passes (no regressions from registry changes)

## Dependencies & Risks

- **Registry naming reconciliation** (`requires` vs `capacity`): Changing function names in the features registry could affect help/autocomplete search results. Needs careful testing.
- **Hugo data templates**: The site has never used `.Site.Data` before -- this is new infrastructure. Need to verify Hugo version supports data templates (it's a core Hugo feature, so this is low risk).
- **Example .cm files**: If examples use features that are actually broken, fixing them could require interpreter changes -- but this is unlikely given the test suite.

## References & Research

### Internal References

- Feature registry: `spec/features/registry.go:136-234` (getFunctions)
- Interpreter functions: `impl/interpreter/functions.go:32-126` (BuiltinFunctions)
- Editor shortcuts: `cmd/calcmark/tui/editor/command_menu.go:24-51` (EditorCommands)
- CLI root: `cmd/calcmark/cmd/root.go:11-72`
- CLI eval: `cmd/calcmark/cmd/eval.go:17-35`
- CLI convert: `cmd/calcmark/cmd/convert.go:19-41`
- CLI help: `cmd/calcmark/cmd/help.go:15-61`
- `over` lexer token: `spec/lexer/token.go:77`
- `over` parser: `spec/parser/rdparser.go:532-547`
- Napkin AST: `spec/ast/nodes.go:81-93`
- Napkin eval: `impl/interpreter/napkin_eval.go`
- Napkin display: `format/display/display.go:97`
- NL functions parser: `spec/parser/nl_functions.go`
- Capacity `at` syntax: `spec/parser/capacity_at_test.go`

### Brainstorm

- `docs/brainstorms/2026-02-22-site-docs-deep-clean-brainstorm.md`
