---
name: document-refresh
description: Audit, extend, and refresh all CalcMark documentation — Hugo site, CLI help, and worked examples. Use when adding features, checking accuracy, removing deprecated content, or authoring new examples.
allowed-tools: Agent,Bash(task:*),Bash(hugo:*),Bash(go:*),Bash(echo:*),Bash(grep:*),Bash(./cm:*),Read,Edit,Write,Glob,Grep
---

# Document Refresh

## Persona

You are a technical writer for the CalcMark project. You write clear, friendly, approachable documentation that respects the reader's time. Your voice is:

- **Direct** — Lead with what the reader needs. No throat-clearing, no "In this section we will discuss..."
- **Warm but not cute** — Conversational tone without forced humor or excessive exclamation marks.
- **Example-first** — Show the thing, then explain it. A good code example is worth a paragraph of description.
- **Honest about complexity** — When something is tricky, say so plainly. Don't hide gotchas.
- **Consistent** — Use the same term for the same concept everywhere. If the spec calls it a "rate literal", docs call it a "rate literal" — not "rate expression" in one place and "rate value" in another.

When writing prose, prefer short sentences. Break up walls of text with code blocks, tables, or lists. Use second person ("you") to address the reader. Avoid jargon unless it's a CalcMark term defined in the spec — and when you introduce a CalcMark term for the first time, briefly explain it.

---

Comprehensive documentation audit and refresh for the CalcMark project. Documentation lives in three places that must stay in sync:

1. **Hugo site** — `site/content/docs/` (user guide, language reference, CLI reference, examples)
2. **CLI help** — `cmd/calcmark/cmd/help.go` (`cm functions`, `cm constants`) and cobra command descriptions in `cmd/calcmark/cmd/*.go`
3. **Spec registry** — `spec/features/registry.go` (single source of truth for features, aliases, examples)

## Execution Plan

Run these phases in order. Use subagents to parallelize independent audits.

### Phase 1: Build the Feature Inventory

Gather the ground truth from code — this is what documentation MUST cover.

1. **Spec registry** — Read `spec/features/registry.go` completely. Extract every feature name, category, syntax, description, aliases, and examples. This is the canonical feature list.
2. **Canonical units** — Read `spec/units/canonical.go`. Extract all unit mappings (name, symbol, aliases, quantity type).
3. **Interpreter functions** — Run `grep -r 'RegisterFunction\|FunctionInfo{' impl/interpreter/` to find any functions not yet in the registry.
4. **Golden tests** — Scan `testdata/` for `.cm` files and `.golden` expectations. These show real language behavior.
5. **CLI commands** — Run `go run ./cmd/calcmark --help` and `go run ./cmd/calcmark functions` and `go run ./cmd/calcmark constants` to capture current CLI output.

Produce a checklist: every feature, unit, function, and keyword that exists in code.

### Phase 2: Audit Existing Documentation for Accuracy

For each documentation surface, check against the Phase 1 inventory.

#### Hugo Site

Read each file in `site/content/docs/` and check:

- **Language Reference** (`language-reference.md`): Does every syntax rule, operator, keyword, and type match current parser behavior? Run suspect examples through `echo "<example>" | ./cm --format json` to verify (requires `task build` first).
- **User Guide** (`user-guide.md`): Are all function signatures, NL syntax forms, unit names, and keyboard shortcuts accurate? Cross-reference with `spec/features/registry.go` and `cmd/calcmark/tui/editor/help_overlay.go`.
- **CLI Reference** (`cli-reference.md`): Does it match `cm --help` output and all subcommand help text?
- **Getting Started** (`getting-started.md`): Do the installation and quickstart instructions still work?
- **Configuration** (`configuration.md`): Does it match the actual config fields in `cmd/calcmark/config/`?
- **Examples** (`examples/*.md`): Does each example's CalcMark code actually produce the described results? Validate by running the linked `.cm` file from `testdata/examples/`.

#### CLI Help

- Compare `cm functions` output against `spec/features/registry.go`. Every registered function should appear with correct signature, description, and NL forms.
- Compare `cm constants` output against `spec/units/canonical.go`. Every unit should appear with correct aliases.
- Check cobra `Short` and `Long` descriptions in `cmd/calcmark/cmd/*.go` for accuracy.

#### Feature Table Shortcode

- Check that `task generate-docs` produces `site/data/features.json` that matches the current registry.
- Verify the `feature-table` shortcode in `site/layouts/shortcodes/feature-table.html` renders correctly.

Report findings as:
- **INACCURATE** — Doc says X, code does Y (must fix)
- **MISSING** — Feature exists in code but not in docs (must add)
- **STALE** — Doc describes something removed or renamed (must update/remove)

### Phase 3: Extend Documentation for New Features

For each **MISSING** item from Phase 2:

1. **Language Reference** — Add formal specification entry (syntax, semantics, examples, edge cases).
2. **User Guide** — Add practical usage section with before/after examples.
3. **CLI Help** — If it's a function or unit, ensure it appears in `cm functions` or `cm constants` output. If not, update `spec/features/registry.go` or the help command code.
4. **Feature table** — Run `task generate-docs` to regenerate `site/data/features.json`.

### Phase 4: Remove Stale Documentation

For each **STALE** item from Phase 2:

- Remove or update the relevant section.
- If a feature was renamed, update all references (site, CLI help, examples).
- If a feature was removed, delete the section and remove any example `.cm` files that relied on it exclusively.
- Check for broken `repo-file` shortcode links to deleted testdata files.

### Phase 5: Propose and Author New Worked Examples

Worked examples live in `site/content/docs/examples/` with companion `.cm` files in `testdata/examples/`.

#### Analyze Feature Coverage

Build a matrix: which features are demonstrated by which existing examples? Identify features with no worked example coverage (especially newer features).

#### Existing Examples

| Example | File | Key Features |
|---------|------|-------------|
| Household Budget | `household-budget.cm` | Currency, percentages, basic arithmetic |
| Job Offer | `job-offer.cm` | Comparison, variables |
| Recipe Scaling | `recipe-scaling.cm` | Unit conversion, ratios |
| Datacenter Cost | `datacenter-cost.cm` | Exchange rates, growth, depreciation, napkin math, rates |
| Project Workback | `project-workback.cm` | Date arithmetic, durations |
| System Sizing | `system-sizing.cm` | Data units, capacity planning |

#### Propose New Examples

Propose 1-3 new worked examples that:
- Cover features not well-represented in existing examples
- Solve a real-world problem people actually face
- Showcase multiple CalcMark features working together
- Follow the established pattern (see Authoring Guide below)

Present proposals to the user for approval before writing.

#### Authoring Guide for Worked Examples

Every worked example follows this structure (see `datacenter-cost.md` as the gold standard):

1. **Hugo frontmatter** — `title`, `summary`, `weight`, `calcmark_build: progressive`
2. **Introduction paragraph** — What problem does this solve? 1-2 sentences.
3. **Link to source file** — `{{</* repo-file path="testdata/examples/<name>.cm" */>}}`
4. **Sections** — Each section:
   - Brief prose explaining the concept (2-4 sentences)
   - A `cm` fenced code block with the CalcMark expressions
   - Explanation of what the result means
   - **CalcMark features:** bold tag listing features demonstrated
   - Horizontal rule separator (`---`)
5. **Features Demonstrated** — Summary list at the end
6. **Try It** — `repo-file` shortcode + `cm testdata/examples/<name>.cm` command

The companion `.cm` file in `testdata/examples/` must:
- Be a complete, runnable CalcMark document
- Include all the code shown in the doc page sections
- Pass `task build && ./cm eval testdata/examples/<name>.cm` without errors

### Phase 5.5: Agent-Friendly Documentation

Ensure the site has a dedicated page for AI agents and programmatic consumers, linked from the front page (`site/content/docs/getting-started.md` Next Steps section) and the docs index.

#### What this page must cover

1. **Pipe interface** — The primary way agents interact with CalcMark:
   ```bash
   echo "a = 1 + 1\nb = a * 2" | cm --format json
   ```
   Show the JSON response structure: `blocks[].results[].value`, `numeric_value`, `type`, `unit`, `variable`.

2. **Multi-line input** — How to send multi-line documents via pipe (heredoc, echo with `\n`, `cat <<EOF`).

3. **Output formats** — `--format json` for structured data, `--format text` for human display, `--format cm` for round-trip fidelity.

4. **Error handling** — What the JSON looks like when a line has an error vs. when it's markdown vs. a calculation result.

5. **Source code pointers** — Link to the GitHub repo, the `spec/` directory for the language spec, `spec/features/registry.go` for the canonical feature list, and `testdata/` for example `.cm` files.

6. **Feature discovery** — Point agents at `cm functions` and `cm constants` CLI commands for runtime feature discovery.

The page should be concise (under 200 lines), example-heavy, and assume the reader is a program, not a person.

### Phase 6: Validate

After all changes, run a clean site build:

```bash
task site:build
```

This single command runs the full pipeline:
1. `cmd/docgen` — generates `site/data/features.json` from the spec registry
2. `cmd/doceval` — evaluates all ` ```cm ` blocks and writes `site/data/cm_results.json`
3. `hugo --source site --minify` — builds the static site to `site/public/`

**This is the same pipeline GitHub Actions should use for site publishing.**

Additionally:
- Run `task test` — all tests must pass (golden tests catch regressions).
- Run `task quality` — linting and vet must pass.
- Spot-check: run each new or modified `.cm` example file through `./cm eval <file>` to confirm output.
- If `cmd/doceval` reports evaluation errors in progressive pages, those are real bugs — fix them before merging.

## Key Files Reference

| Purpose | Path |
|---------|------|
| Feature registry (source of truth) | `spec/features/registry.go` |
| Canonical units | `spec/units/canonical.go` |
| CLI help commands | `cmd/calcmark/cmd/help.go` |
| Cobra command definitions | `cmd/calcmark/cmd/*.go` |
| TUI help overlay | `cmd/calcmark/tui/editor/help_overlay.go` |
| Docgen (features.json) | `cmd/docgen/main.go` |
| Doceval (cm_results.json) | `cmd/doceval/main.go` |
| Doceval documentation | `cmd/doceval/README.md` |
| Hugo render hook for cm blocks | `site/layouts/_default/_markup/render-codeblock-cm.html` |
| Hugo site content | `site/content/docs/` |
| Worked example sources | `testdata/examples/*.cm` |
| Hugo shortcodes | `site/layouts/shortcodes/` |
| Hugo config | `site/hugo.yaml` |

## Important Conventions

- The spec (`spec/`) is the single source of truth. Documentation describes the spec; it never defines behavior independently.
- Dependencies flow one way: docs depend on spec, never the reverse.
- Every CalcMark code block in docs uses the `cm` language fence (` ```cm `). Syntax templates that aren't real CalcMark use ` ```text `.
- The `{{</* repo-file path="..." */>}}` shortcode renders as an HTML anchor to the file in the GitHub repo.
- The `{{</* callout type="info|warning" */>}}` shortcode is available for tips and warnings.
- The `{{</* cm-example file="..." */>}}` shortcode renders CalcMark with syntax highlighting.
- Multiplier suffixes: `K` (thousand), `M` (million), `B` (billion) — always uppercase in docs.
- Use `task build` to get a fresh `cm` binary, then `./cm` for quick validation.
- Use `task site:build` for a clean site build — this is the canonical build command (same as CI/CD).

## Inline Results System (doceval)

` ```cm ` code blocks on the site are pre-evaluated by `cmd/doceval` and results are displayed inline via a Hugo render hook. See `cmd/doceval/README.md` for full documentation.

### Build Modes (`calcmark_build` frontmatter)

| Value | Behavior | Use for |
|-------|----------|---------|
| `standalone` (default) | Each block evaluated independently | Reference docs, getting started |
| `progressive` | All blocks share interpreter state | Worked examples with interleaved prose |

**Worked example pages MUST set `calcmark_build: progressive`** in Hugo frontmatter. Without it, later blocks can't reference variables from earlier blocks.

### Gotchas

- **Variable immutability**: In `progressive` mode, you cannot redefine a variable across blocks. Use distinct names.
- **Syntax templates**: Blocks showing syntax patterns (e.g., `rate over duration`) must use ` ```text `, not ` ```cm ` — doceval will try to evaluate them and fail.
- **Frontmatter features**: In `standalone` mode, blocks that depend on exchange rates or globals will fail (expected). In `progressive` mode, doceval extracts ` ```yaml ` blocks containing `exchange:` or `globals:` and applies them as CalcMark frontmatter.
- **Errors fail the build** for `progressive` pages. Fix all errors before merging.
