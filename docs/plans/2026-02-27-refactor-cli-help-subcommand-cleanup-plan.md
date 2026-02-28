---
title: "Refactor: CLI Help & Subcommand Cleanup"
type: refactor
status: completed
date: 2026-02-27
---

# Refactor: CLI Help & Subcommand Cleanup

## Enhancement Summary

**Deepened on:** 2026-02-27
**Agents used:** code-simplicity-reviewer, architecture-strategist, pattern-recognition-specialist, best-practices-researcher, learnings-researcher

### Key Improvements
1. Use Cobra's built-in `Deprecated` field instead of `Hidden: true` + manual stderr warning — idiomatic, less code, auto-hides from help and completions
2. Simplified from 5 phases to 2 — removed no-op phases and redundant tests
3. Added VHS tape script updates to prevent deprecation warnings in recorded demos

## Overview

The `cm edit` subcommand is redundant with the root command (`cm [file]`) — both call `runEdit()` with identical behavior. This plan deprecates the `edit` subcommand using Cobra's built-in `Deprecated` field and updates tests for consistency.

## Problem Statement

1. **`cm edit` is a redundant alias**: `cm budget.cm` and `cm edit budget.cm` produce identical behavior via the shared `runEdit()` function in `tui.go`.
2. **Help output lists `edit` alongside real commands**: This suggests `edit` is a distinct operation when it's just the default behavior restated.
3. **REPL mode is unreachable from CLI**: `tui.NewApp()` (REPL mode) is never called from any cobra command. Self-contained and tested but out of scope for this change.

## Proposed Solution

**Deprecate `cm edit` using Cobra's `Deprecated` field.** When set to a non-empty string, Cobra automatically:
- Prints `Command "edit" is deprecated, <message>` when the command is invoked
- Hides the command from help output (`IsAvailableCommand()` returns false)
- Hides the command from shell completions

This preserves backwards compatibility — `cm edit` still works but warns users to migrate.

### Why `Deprecated` over `Hidden: true` + manual warning?

Research confirmed Cobra v1.10.2 (our version) has a first-class `Deprecated` field that handles everything. Using `Hidden: true` + `fmt.Fprintln(os.Stderr, ...)` reinvents this, adds custom code, and diverges from the framework's intended pattern. The `Deprecated` field is the approach used by kubectl, gh CLI, and other major Go CLIs.

### Why deprecation over hard removal?

The project has "strong backwards compatibility requirements" per CLAUDE.md. Hard removal would break scripts, aliases, or muscle memory using `cm edit`. Deprecation gives users a migration path across releases.

## Acceptance Criteria

- [x] `cm edit` still works but prints Cobra's deprecation notice
- [x] `cm edit` is hidden from `cm help` output
- [x] `cm edit` is hidden from shell completions (`cm <TAB>`)
- [x] `cm help` lists only: completion, convert, eval, version
- [x] Help test updated to not expect "edit" in command list
- [x] VHS tape scripts updated from `cm edit` to `cm`
- [x] All tests pass (`task test`)
- [x] Quality checks pass (`task quality`)

## Technical Approach

### Phase 1: Deprecate `cm edit` and Update VHS Scripts

**File: `cmd/calcmark/cmd/edit.go`**

Replace `Hidden: true` + manual warning with Cobra's `Deprecated` field:

```go
package cmd

import (
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit [file.cm]",
	Short: "Open the CalcMark document editor",
	Long: `Open the split-pane document editor for working with CalcMark files.
The editor shows source on the left and computed results on the right.

Examples:
  cm edit                   Open editor with file picker
  cm edit budget.cm         Open specific file in editor`,
	Deprecated: "use 'cm' or 'cm <file>' instead.",
	Args:       cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			runEdit(args[0])
		} else {
			runEdit("")
		}
	},
}

func init() {
	rootCmd.AddCommand(editCmd)
}
```

The `Deprecated` field alone achieves hiding + warning. No `Hidden: true` needed (redundant — `Deprecated` already causes `IsAvailableCommand()` to return false). No manual `fmt.Fprintln` needed.

**VHS tape scripts** (4 files under `scripts/`):

Update `cm edit` to `cm` in:
- `scripts/hero.tape`
- `scripts/feature-variables.tape`
- `scripts/feature-export.tape`
- `scripts/feature-autocomplete.tape`

These are project-internal tooling, not user-facing contracts, so updating them is safe and prevents deprecation warnings from appearing in recorded demos.

### Phase 2: Update Tests and Validate

**File: `cmd/calcmark/cmd/help_test.go` (line 120)**

Remove `"edit"` from the expected commands list:
```go
for _, cmd := range []string{"eval", "convert", "version"} {
```

The existing `TestHelpCmdShowsCLIOverview` already validates that help output contains the correct command set. Once `"edit"` is removed from the expected list, the test correctly validates the new behavior. The help rendering loop in `help.go:36` already skips commands where `IsAvailableCommand()` returns false (which includes deprecated commands), so no additional "hidden from help" test is needed.

**Validate:**
- Run `task test` — all tests must pass
- Run `task quality` — lint, vet, and quality checks must pass

## Not in Scope

REPL code, planning docs, internal `:edit` REPL command, and `editor/command_menu.go` `"edit"` category are all unrelated and out of scope.

## Files Changed

| File | Change |
|------|--------|
| `cmd/calcmark/cmd/edit.go` | Replace current impl with `Deprecated` field |
| `cmd/calcmark/cmd/help_test.go` | Remove `"edit"` from expected commands list |
| `scripts/hero.tape` | `cm edit` → `cm` |
| `scripts/feature-variables.tape` | `cm edit` → `cm` |
| `scripts/feature-export.tape` | `cm edit` → `cm` |
| `scripts/feature-autocomplete.tape` | `cm edit` → `cm` |

## Research Insights

### Cobra `Deprecated` vs `Hidden` Fields

| Behavior | `Hidden: true` | `Deprecated: "msg"` |
|----------|----------------|---------------------|
| Command runs normally | Yes | Yes |
| Hidden from help | Yes | Yes |
| Prints warning on use | No | Yes |
| Excluded from completions | Yes | Yes |

Setting both is redundant. `Deprecated` is the correct choice for user-facing migration.

### Known Pitfall: Cobra Deprecation Output Stream

Cobra's deprecation message goes through `c.Printf` → `OutOrStderr()`, which defaults to stderr. This is correct for our use case. If `rootCmd.SetOut()` were ever called (it is not), the destination would change. Acceptable risk.

### Deprecation Lifecycle

| Phase | Action | Timeline |
|-------|--------|----------|
| Deprecate | Set `Deprecated` field (this plan) | v0.x.0 |
| Keep functional | Command works with warning | 2-3 minor releases |
| Remove | Delete `edit.go` entirely | v0.(x+2).0 or v1.0.0 |

## References

- CLAUDE.md: backwards compatibility requirements
- `docs/solutions/logic-errors/viper-isset-embedded-defaults-deprecation.md`: established deprecation pattern — warn only on user action, keep code path for backwards compat
- Cobra `Deprecated` field: [pkg.go.dev/github.com/spf13/cobra](https://pkg.go.dev/github.com/spf13/cobra)
- [spf13/cobra#1993](https://github.com/spf13/cobra/issues/1993): deprecation output stream behavior
