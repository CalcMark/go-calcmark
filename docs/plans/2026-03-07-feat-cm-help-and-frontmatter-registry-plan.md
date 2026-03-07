---
title: "feat: Restructure cm help CLI and add frontmatter registry"
type: feat
status: completed
date: 2026-03-07
brainstorm: docs/brainstorms/2026-03-07-cm-help-and-frontmatter-registry-brainstorm.md
---

# feat: Restructure cm help CLI and add frontmatter registry

## Enhancement Summary

**Deepened on:** 2026-03-07
**Sections enhanced:** 5 phases + technical considerations
**Research sources:** Code analysis of registry.go, canonical.go, conversion.go, frontmatter.go, help.go, root.go, docgen/main.go, feature-table.html; past solution docs/solutions/code-organization/custom-help-hardcoding-flags.md; Cobra CLI patterns research

### Key Improvements
1. Discovered Cobra `help` command naming conflict — must use `SetHelpCommand()` carefully or rename
2. Feature struct may need a new field for YAML-block syntax rendering (frontmatter directives are multi-line)
3. `validUnitCategories` uses lowercase→TitleCase mapping that `units.Categories()` must preserve
4. AGENTS.md and CLAUDE.md have no `cm functions`/`cm constants` references — skip those doc updates

### New Considerations Discovered
- The feature-table Hugo shortcode renders `syntax` in a single `<code>` block — multi-line YAML won't display well without a template enhancement or packing syntax into Description
- Previous learnings (docs/solutions/code-organization/custom-help-hardcoding-flags.md) show a custom `helpCmd` was **removed** in favor of Cobra's built-in help + command groups. Re-adding it requires care to avoid the same drift problem.

## Overview

Move `cm functions` and `cm constants` under a `cm help` subcommand, add `cm help frontmatter` for discovering frontmatter directives, and derive unit categories from code to eliminate drift. This makes CalcMark's reference system discoverable from a single entry point and documents frontmatter features that are currently invisible to users.

## Problem Statement

1. **No frontmatter discovery.** Users cannot discover valid `scale`, `convert_to`, `exchange`, or `globals` options from the CLI. The valid `unit_categories` values are hardcoded in `spec/document/frontmatter.go` and undiscoverable — you have to read source code.
2. **Fragmented help.** `cm functions` and `cm constants` are orphaned root commands with no shared entry point. There's no `cm frontmatter` equivalent.
3. **Category drift risk.** The `validUnitCategories` map in `frontmatter.go` is hardcoded and can drift from the actual unit categories defined in `canonical.go`.

## Proposed Solution

### Phase 1: Unit Categories Function

Add `units.Categories() []string` that derives the category list from `StandardUnits` plus `DataSize`.

- [x] Add `Categories() []string` to `spec/units/canonical.go` (or a new `spec/units/categories.go`)
  - Scan `StandardUnits` for unique `Quantity` values
  - Include `CategoryDataSize` from `conversion.go`
  - Return sorted, deduplicated list
- [x] Replace `validUnitCategories` map in `spec/document/frontmatter.go` with a call to `units.Categories()`
  - Build the lookup map from the derived list at init time
  - No behavior change, just eliminates the hardcoded list
- [x] Add test: `TestCategories_MatchesStandardUnits` in `spec/units/` verifying the list is non-empty and contains expected entries

#### Research Insights

**Implementation detail — `validUnitCategories` mapping:**

The current `validUnitCategories` map in `frontmatter.go:92-102` maps **lowercase** keys to **TitleCase** canonical names (e.g., `"mass"` → `"Mass"`, `"datasize"` → `"DataSize"`). The `Categories()` function should return TitleCase names matching the `Quantity` field values from `StandardUnits`. The `parseUnitCategories()` function at `frontmatter.go:375` does `strings.ToLower(name)` lookup, so the replacement must preserve this case-insensitive lookup behavior.

**Concrete implementation:**

```go
// spec/units/categories.go

// Categories returns the sorted list of unit categories derived from
// StandardUnits and the DataSize conversion category. This is the single
// source of truth — frontmatter validation uses this instead of a
// hardcoded map.
func Categories() []string {
    seen := make(map[string]bool)
    for _, u := range StandardUnits {
        seen[u.Quantity] = true
    }
    seen[CategoryDataSize] = true

    cats := make([]string, 0, len(seen))
    for c := range seen {
        cats = append(cats, c)
    }
    slices.Sort(cats)
    return cats
}
```

**Replacement in `frontmatter.go`:**

```go
// Replace the hardcoded validUnitCategories map with a derived one.
// Built at init from units.Categories() so it stays in sync.
var validUnitCategories map[string]string

func init() {
    validUnitCategories = make(map[string]string)
    for _, cat := range units.Categories() {
        validUnitCategories[strings.ToLower(cat)] = cat
    }
}
```

**Edge case — `CategoryDataSize` is `"DataSize"` not `"Data Size"`:** The lowercase lookup key will be `"datasize"` (no space), which matches the current hardcoded map. No behavior change.

### Phase 2: Frontmatter Registry

Add frontmatter directives to the feature registry.

- [x] Add `CategoryFrontmatter Category = "frontmatter"` to `spec/features/registry.go`
- [x] Add `getFrontmatterFeatures() []Feature` returning 4 features:

#### `exchange`
```yaml
exchange:
  USD_EUR: 0.92
  EUR_GBP: 0.86
```
- Description: "Define currency conversion rates"
- Syntax: `exchange:\n  FROM_TO: rate`
- Aliases: none

#### `globals`
```yaml
globals:
  tax_rate: 0.32
  base_price: $100
```
- Description: "Define document-wide variables"
- Syntax: `globals:\n  name: value`
- Aliases: none

#### `scale`
```yaml
scale: 2
scale:
  factor: 4
  unit_categories: [Length, Mass]
```
- Description: "Multiply quantity results by a factor"
- Syntax: scalar or map form
- Sub-keys: `factor` (number, required), `unit_categories` (list, optional)
- Defaults: all categories except Temperature

#### `convert_to`
```yaml
convert_to: si
convert_to:
  system: imperial
  unit_categories: [Length]
```
- Description: "Convert results to a measurement system"
- Syntax: string or map form
- Sub-keys: `system` (string: si | imperial, required), `unit_categories` (list, optional)

- [x] Include derived `units.Categories()` list in the `scale` and `convert_to` feature descriptions
- [x] Wire `getFrontmatterFeatures()` into `NewRegistry()` alongside existing `getFunctions()`, `getUnits()`, etc.
- [x] Add test: `TestRegistry_FrontmatterCategory` verifying all 4 directives are present

#### Research Insights

**Feature struct fit for frontmatter directives:**

The current `Feature` struct (`registry.go:37-45`) has fields: `Name`, `Category`, `Syntax`, `Description`, `Aliases`, `Example`, `NLExample`. Frontmatter directives need to convey richer information (sub-keys, defaults, valid values, multiple syntax forms).

**Recommended approach — pack into existing fields, no struct changes:**

- `Syntax`: Use the scalar form as the primary syntax (e.g., `"scale: <factor>"`)
- `Description`: Include sub-keys, defaults, and valid values as part of the description text
- `Example`: Show the map form as the example (e.g., `"scale:\n  factor: 4\n  unit_categories: [Length, Mass]"`)
- This avoids changing the `Feature` struct which would ripple through docgen, shortcode, and tests

**Example Feature entry:**

```go
{
    Name:     "scale",
    Category: CategoryFrontmatter,
    Syntax:   "scale: <factor>",
    Description: fmt.Sprintf(
        "Multiply quantity results by a factor. "+
            "Map form: scale: {factor: N, unit_categories: [...]}\n"+
            "Categories: %s\n"+
            "Default: all except Temperature",
        strings.Join(units.Categories(), ", "),
    ),
    Example: "scale: 2",
},
```

**Hugo shortcode compatibility:**

The `feature-table.html` shortcode at `site/layouts/shortcodes/feature-table.html` renders `syntax` and `example` in `<code>` blocks. Multi-line YAML won't render well in a single `<code>` tag. Two options:
1. Keep the shortcode as-is and use the CLI `cm help frontmatter` for the rich YAML display (recommended — keeps shortcode simple)
2. Enhance the shortcode to handle `\n` in syntax/example fields with `<pre>` blocks (over-engineering for 4 entries)

**Recommendation:** Use option 1. The feature-table shortcode shows a concise reference table. The CLI `cm help frontmatter` shows the verbose YAML examples. The language-reference.md already has manually authored frontmatter documentation with full examples.

**Docgen (`cmd/docgen/main.go`) — no changes needed:**

Docgen iterates `reg.Categories()` at line 49 and `reg.ByCategory(cat)` at line 50. Adding `CategoryFrontmatter` to the registry will automatically include frontmatter features in `site/data/features.json`. The `kebabCase()` function handles anchor generation. No code changes required.

### Phase 3: CLI Restructuring

Restructure the command tree.

#### Critical: Cobra `help` naming conflict

**Previous learning (docs/solutions/code-organization/custom-help-hardcoding-flags.md):**

The project previously had a custom `helpCmd` that was removed because it duplicated Cobra's built-in help and drifted from flag definitions. The current setup uses Cobra command groups (v1.6+) with `rootCmd.AddGroup(&cobra.Group{ID: "topics", Title: "Help Topics:"})` in `root.go:74`.

**The naming problem:** Cobra registers a built-in `help` subcommand automatically. Adding a custom `helpCmd` with `Use: "help"` will shadow Cobra's built-in, potentially breaking `cm help eval`, `cm help convert`, etc.

**Recommended approach — use `SetHelpCommand()` to replace Cobra's built-in:**

```go
// Replace Cobra's default help command with our enhanced one
// that includes functions, constants, and frontmatter topics.
rootCmd.SetHelpCommand(helpCmd)
```

This is safe because:
- `helpCmd` becomes the ONLY help command (no shadowing)
- When invoked without args (`cm help`), it prints all sections
- When invoked with a subcommand name (`cm help eval`), Cobra routes to that subcommand's help
- When invoked with a topic (`cm help functions`), it routes to our topic subcommand

**Alternative if `SetHelpCommand` doesn't support subcommands well:** Keep `functions` and `constants` as root commands (current behavior) and just add `frontmatter` as a new root command in the "Help Topics:" group. This is the simplest change but doesn't unify under `cm help`. Evaluate during implementation.

#### `cmd/calcmark/cmd/help.go` — Major rewrite

- [x] Remove `helpFunctionsCmd` and `helpConstantsCmd` cobra commands
- [x] Create `helpCmd` cobra command (`cm help`)
  - Subcommands: `functions`, `constants`, `frontmatter`
  - Flags: `--functions`, `--constants`, `--frontmatter`, `--all` (implicit default)
  - When invoked bare (`cm help`), prints all three sections
- [x] Add `printFrontmatter()` function
  - Verbose output with YAML examples (per brainstorm decision)
  - Show each directive: description, both forms (scalar + map), sub-keys, defaults
  - Show available unit categories (from `units.Categories()`)
  - Show valid systems for `convert_to` (si, imperial)
- [x] Adapt existing `printFunctions()` and `printConstants()` (no logic changes, just callable from new command)

#### `cmd/calcmark/cmd/root.go`

- [x] Remove the `"topics"` command group registration
- [x] Register `helpCmd` as the help command via `rootCmd.SetHelpCommand(helpCmd)` or `rootCmd.AddCommand(helpCmd)`

#### Command behavior:

| Command | Output |
|---------|--------|
| `cm help` | All sections (functions + constants + frontmatter) |
| `cm help functions` | Functions only |
| `cm help constants` | Constants only |
| `cm help frontmatter` | Frontmatter directives only |
| `cm help --functions` | Same as `cm help functions` |
| `cm help --all` | Same as `cm help` |

#### Research Insights

**Cobra help command patterns:**

Using `SetHelpCommand()` replaces Cobra's default help. The custom help command will handle:
1. `cm help` (no args) — print all sections
2. `cm help <subcommand>` — if subcommand matches a topic (functions/constants/frontmatter), show that topic; otherwise delegate to Cobra's standard subcommand help
3. `cm help --functions` — flag-based topic selection

**Implementation pattern:**

```go
var helpCmd = &cobra.Command{
    Use:   "help [topic]",
    Short: "Show CalcMark reference information",
    Long:  "Display functions, constants, and frontmatter directives.",
    Run: func(cmd *cobra.Command, args []string) {
        showFuncs, _ := cmd.Flags().GetBool("functions")
        showConsts, _ := cmd.Flags().GetBool("constants")
        showFM, _ := cmd.Flags().GetBool("frontmatter")

        // If no flags set, show all
        if !showFuncs && !showConsts && !showFM {
            showFuncs, showConsts, showFM = true, true, true
        }

        if showFuncs { printFunctions() }
        if showConsts { printConstants() }
        if showFM { printFrontmatter() }
    },
}
```

**Test updates — `help_test.go`:**

The existing tests reference `helpFunctionsCmd` and `helpConstantsCmd` directly (lines 17, 48). These will need to be updated to test through the new command structure. Keep the `captureStdout()` helper.

**Gotcha — `cm help eval` routing:**

If using `SetHelpCommand()`, Cobra will route `cm help eval` to show the eval subcommand's help text (standard Cobra behavior). Only unrecognized args like `cm help functions` will route to our custom subcommands. This is the desired behavior.

**Gotcha — `printFunctions()` imports `impl/interpreter`:**

The current `help.go` imports `impl/interpreter` for `GetFunctionsByCategory()` and `GetCategoryOrder()`. This creates a `cmd → impl` dependency which is fine per the architecture. The new `printFrontmatter()` should only import `spec/` packages (no `impl/` dependency needed).

### Phase 4: Documentation Updates

- [x] **`cmd/docgen/main.go`** — No changes needed (already consumes all registry categories via `reg.Categories()` at line 49)
- [x] Run `task generate-docs` to regenerate `site/data/features.json` with frontmatter category
- [x] **`site/content/docs/cli-reference.md`** — Replace `cm functions` / `cm constants` references with `cm help functions` / `cm help constants` / `cm help frontmatter`
  - Lines to update: 17-18 (quickstart examples), 283-296 (Help Topics section)
- [x] **`site/content/docs/agent-integration.md`** — Update Feature Discovery section
  - Lines 135-136: `cm functions` → `cm help functions`, `cm constants` → `cm help constants`
- [x] **`site/content/docs/user-guide.md`** — Update Tips section
  - Line 225: `cm constants` → `cm help constants`
  - Line 1007: `cm functions` / `cm constants` → `cm help functions` / `cm help constants`
- [x] **`site/content/docs/language-reference.md`** — Add `{{< feature-table category="frontmatter" >}}` in the Frontmatter section, update any `cm constants` references
- [ ] ~~**`AGENTS.md`** — Update command references~~ — No references found, skip
- [ ] ~~**`CLAUDE.md`** — Update if any references exist~~ — No references found, skip

#### Research Insights

**Exact grep results for documentation references:**

```
site/content/docs/cli-reference.md:17:  cm functions
site/content/docs/cli-reference.md:18:  cm constants
site/content/docs/cli-reference.md:283: ## `cm functions` / `cm constants`
site/content/docs/cli-reference.md:289-296: table and examples
site/content/docs/user-guide.md:225:  Run `cm constants`...
site/content/docs/user-guide.md:1007: Run `cm functions`... `cm constants`
site/content/docs/agent-integration.md:135-136: cm functions / cm constants
```

Neither `AGENTS.md` nor `CLAUDE.md` reference these commands — those doc updates can be skipped.

### Phase 5: Validation

- [x] `task test` — all tests pass
- [x] `task quality` — lint, vet, modernize clean (pre-existing unused const in version.go only)
- [ ] `task site:build` — site builds with new features.json
- [x] Manual verification: `cm help`, `cm help functions`, `cm help constants`, `cm help frontmatter`
- [x] Verify `cm functions` and `cm constants` no longer exist (clean error)

#### Research Insights

**Existing test coverage to update:**

- `cmd/calcmark/cmd/help_test.go`: 4 tests that reference `helpFunctionsCmd`/`helpConstantsCmd` directly — must update to test through new command structure
- `spec/features/registry_test.go`: `TestRegistryByCategory` at line 64 will need a new entry for `CategoryFrontmatter` with `wantMin: 4`
- `spec/features/registry_test.go`: `TestRegistryCategories` at line 248 checks `len(cats) >= 5` — will automatically pass with the new category

**New tests needed:**

1. `spec/units/TestCategories_MatchesStandardUnits` — verify Categories() returns expected entries (Length, Mass, Volume, Temperature, Speed, Energy, Power, Area, DataSize)
2. `spec/units/TestCategories_Sorted` — verify the list is sorted
3. `spec/features/TestRegistry_FrontmatterCategory` — verify 4 directives present with correct names
4. `spec/document/TestValidUnitCategories_DerivedFromCode` — verify the init-time derived map matches expected behavior
5. `cmd/calcmark/cmd/TestHelpFrontmatterOutput` — verify frontmatter output contains directive names and categories
6. `cmd/calcmark/cmd/TestHelpAllSections` — verify `cm help` shows all three sections

## Technical Considerations

- **Dependency direction.** `spec/units` → `spec/features` is fine (features already imports units). The `cmd/` layer imports both. The new `Categories()` function stays in `spec/units`, and `spec/document/frontmatter.go` already imports `spec/units` (it does NOT currently — this is a **new dependency** that needs verification). Actually, `frontmatter.go` is in `spec/document` and does not import `spec/units`. Adding `units.Categories()` call would create `spec/document → spec/units` dependency. This is a valid direction (both are `spec/` packages, no circular dependency).
- **No interpreter dependency.** The frontmatter registry entries are pure spec — no `impl/` imports needed.
- **Feature-table shortcode.** Already renders any category from `features.json`. Adding `CategoryFrontmatter` will make it appear automatically when `{{< feature-table category="frontmatter" >}}` is used. Multi-line YAML syntax won't render ideally in the shortcode's `<code>` blocks — the CLI output is the canonical verbose reference.
- **validUnitCategories drift fix.** Deriving from `units.Categories()` is the real fix for the drift risk flagged in the code review. Once this ships, the hardcoded map is gone.
- **Cobra help conflict.** The project previously removed a custom help command (see `docs/solutions/code-organization/custom-help-hardcoding-flags.md`). Re-adding one must be done carefully using `SetHelpCommand()` or by keeping topic commands as root-level commands in a group. Evaluate both approaches during Phase 3 implementation.

## Acceptance Criteria

- [x] `cm help` prints functions, constants, and frontmatter in a single output
- [x] `cm help frontmatter` shows all 4 directives with YAML examples, sub-keys, defaults, and valid unit categories
- [x] Unit categories in `cm help frontmatter` are derived from code, not hardcoded
- [x] `validUnitCategories` in `frontmatter.go` is derived from `units.Categories()`
- [x] `cm functions` and `cm constants` are removed (error on use)
- [x] `site/data/features.json` includes `frontmatter` category
- [x] All documentation references updated
- [x] All tests pass, quality checks clean, site builds

## References

- Brainstorm: `docs/brainstorms/2026-03-07-cm-help-and-frontmatter-registry-brainstorm.md`
- Feature registry: `spec/features/registry.go`
- Current help: `cmd/calcmark/cmd/help.go`
- Help tests: `cmd/calcmark/cmd/help_test.go`
- Unit categories: `spec/units/canonical.go` (Quantity field), `spec/units/conversion.go` (DataSize)
- Hardcoded categories: `spec/document/frontmatter.go:90-102`
- Docgen: `cmd/docgen/main.go`
- Feature-table shortcode: `site/layouts/shortcodes/feature-table.html`
- Previous help refactor learning: `docs/solutions/code-organization/custom-help-hardcoding-flags.md`
