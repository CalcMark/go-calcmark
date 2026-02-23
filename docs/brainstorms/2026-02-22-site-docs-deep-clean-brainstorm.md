# Site Documentation Deep Clean

**Date:** 2026-02-22
**Status:** Brainstorm

## What We're Building

A comprehensive audit and rewrite of the CalcMark documentation site (`site/`) to eliminate stale, misleading, and incomplete content. The site currently references features that don't exist, omits features that do exist, and presents an outdated architecture (REPL mode) that has been superseded by the editor TUI + `cm eval` workflow.

### Scope

1. **Remove all REPL command references** -- there is no standalone REPL mode anymore
2. **Fix CLI documentation** -- validate and document actual subcommands/flags from cobra definitions
3. **Generate function reference data from registry.go** -- hybrid approach: auto-generate JSON data, hand-craft surrounding prose
4. **Update language reference to comprehensive spec** -- cover all features including NL syntax, rates, dates, storage, compression, network
5. **Rewrite keyboard shortcuts for editor TUI** -- replace stale REPL shortcuts
6. **Validate all examples** -- run every example through `cm eval` and fix failures
7. **Add CLI Reference page** -- new dedicated page for all subcommands and flags

## Why This Approach

### Hybrid data generation (registry.go -> JSON -> Hugo)

The feature registry (`spec/features/registry.go`) already has strongly-typed fields: Name, Category, Syntax, Description, Aliases (with Parseable flag), and Example. Rather than manually transcribing this into docs (which will drift), we generate a `site/data/functions.json` file from the registry and use Hugo data templates for accurate function tables. Hand-crafted prose wraps the generated tables for context and guidance.

**Generator location:** `cmd/docgen/` -- a small Go tool run via `task generate-docs`. This avoids spec->site dependency violations and keeps it explicit in the Taskfile. The generated JSON is NOT checked into version control; instead, the site build Taskfile entry calls `generate-docs` before Hugo, so the data is always fresh from the registry.

### Remove REPL, document what exists

The user guide documents `:open`, `:save`, `:output`, `:pin`, `:unpin`, `:md` commands that don't exist. The actual interaction model is:
- `cm` / `cm file.cm` -- opens the editor TUI
- `cm eval [file]` -- batch evaluation
- `cm convert` -- format conversion

There is no standalone REPL mode anymore. All REPL command references should be removed.

### Comprehensive language reference

Rather than splitting formal spec from practical docs, the language reference should be the complete, authoritative specification covering ALL features: grammar, type system, functions (all 13+), NL syntax forms, rates, dates, units, storage, compression, and network features.

## Key Decisions

1. **No REPL docs** -- REPL mode is gone; remove all `:command` references
2. **Hybrid function docs** -- Generate JSON data from registry.go, use Hugo data templates, write prose by hand
3. **Generator as cmd/docgen/ + Taskfile** -- Most maintainable: explicit, visible in `task --list`, no spec->site dependency violation. Output is NOT version-controlled; site build task calls generate-docs before Hugo
4. **Keyboard shortcuts rewritten for editor TUI** -- Replace REPL shortcuts with actual editor TUI bindings
5. **New CLI Reference page** -- Dedicated page documenting all subcommands and flags (eval, convert, help, version, completion, --color-mode)
6. **Language reference as comprehensive spec** -- Update to cover all features, not just grammar basics
7. **Validate all examples** -- Run through `cm eval` to confirm they work

## Gap Analysis

### What's documented but doesn't exist

| Item | Location | Status |
|------|----------|--------|
| `:open <file>` command | user-guide.md | Does not exist |
| `:save <file>` command | user-guide.md | Does not exist |
| `:output <file>` command | user-guide.md | Does not exist |
| `:pin` / `:unpin` commands | user-guide.md | Does not exist |
| `:md` command | user-guide.md | Does not exist |
| REPL keyboard shortcuts | user-guide.md | REPL mode removed |
| `cm eval --json` flag | user-guide.md | Flag doesn't exist on eval |
| `capacity()` function name | user-guide.md | Actually `requires()` |

### What exists but isn't documented

| Item | Source | Status |
|------|--------|--------|
| `convert_rate()` function | registry.go | Not in any docs |
| `requires()` function | registry.go | Not in any docs (listed as `capacity()`) |
| `transfer_time()` function | registry.go | Not in any docs |
| `read()` function | registry.go | Not in any docs |
| `seek()` function | registry.go | Not in any docs |
| `compress()` function | registry.go | Not in any docs |
| NL: `read X from Y` | registry.go | Not in any docs |
| NL: `compress X using Y` | registry.go | Not in any docs |
| NL: `transfer X across Y Z` | registry.go | Not in any docs |
| NL: `X at Y per Z [with N% buffer]` | parser tests | Not in any docs |
| `cm convert` subcommand | convert.go | Not documented |
| `cm help functions/constants` | help.go | Not documented |
| `cm version` subcommand | version.go | Not documented |
| `cm completion` subcommand | completion.go | Not documented |
| `--color-mode` flag | root.go | Not documented |
| `cm eval -v` flag | eval.go | Not documented |
| `--to`, `--output`, `--template` flags | convert.go | Not documented |

### What's documented but inaccurate

| Item | Issue |
|------|-------|
| Language reference functions table | Only lists avg/sqrt, missing 11+ functions |
| Language reference reserved keywords | Only lists avg/sqrt as function names |
| User guide functions table | Lists 7 functions, missing 6+ |
| User guide "capacity()" | Function is actually `requires()` |

## Affected Pages

1. `site/content/docs/user-guide.md` -- Major rewrite (remove REPL, fix functions, add TUI shortcuts)
2. `site/content/docs/language-reference.md` -- Major update (comprehensive spec)
3. `site/content/docs/getting-started.md` -- Minor fixes (CLI examples)
4. `site/content/docs/configuration.md` -- Review for accuracy
5. **NEW:** `site/content/docs/cli-reference.md` -- CLI subcommands and flags
6. **NEW:** `cmd/docgen/main.go` -- Registry-to-JSON generator
7. **NEW:** `site/data/functions.json` -- Generated at build time (not checked in)
8. `site/content/docs/examples/*.md` -- Validate all 5 examples

## Open Questions

_None remaining -- all questions resolved during brainstorm._
