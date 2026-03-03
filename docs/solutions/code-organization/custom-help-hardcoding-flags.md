---
title: Fix missing --locale flag in cm help by replacing custom help with Cobra's built-in
date: 2026-03-02
category: code-organization
tags: [cli, help-command, cobra, flags, locale]
severity: medium
component: cmd/calcmark/cmd
symptoms:
  - cm help output missing --locale flag while cm --help showed it correctly
  - new flags not automatically included in custom help renderer
  - custom helpCmd duplicated Cobra's built-in help functionality
root_cause: rootCmd.SetHelpCommand(&cobra.Command{Hidden: true}) suppressed Cobra's help, forcing manual duplication that couldn't stay in sync with flag definitions
resolution: Removed custom help command and used Cobra v1.6+ command groups to display functions and constants as a "Help Topics:" section
---

## Problem

The CalcMark CLI had a custom `helpCmd` in `help.go` that manually rendered commands, flags, and topics. The flags section was hardcoded:

```go
fmt.Println("Flags:")
fmt.Println("  --color-mode string   Color mode: 'auto', 'light', or 'dark'")
```

When `--locale` was added as a persistent flag on `rootCmd`, it appeared in `cm --help` (Cobra's built-in) but was missing from `cm help` (the custom renderer). Any new flag required manual addition to the hardcoded list.

## Root Cause

In `root.go`, Cobra's built-in help command was explicitly suppressed:

```go
rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
```

This forced all help rendering through the custom `helpCmd`, which duplicated what Cobra already does — listing commands, flags, and usage — but couldn't stay in sync with flag definitions.

## Solution

### 1. root.go — Replace suppression with command group

**Before:**
```go
func init() {
    rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
}
```

**After:**
```go
func init() {
    rootCmd.AddGroup(&cobra.Group{ID: "topics", Title: "Help Topics:"})
}
```

Also appended the GitHub Gist CLI note to `rootCmd.Long` (previously only in the custom renderer).

### 2. help.go — Delete custom helpCmd, register topics on rootCmd

**Before:**
```go
var helpCmd = &cobra.Command{
    Use: "help [topic]",
    Run: func(cmd *cobra.Command, args []string) {
        // 30+ lines of manual rendering with hardcoded flags
    },
}

func init() {
    rootCmd.AddCommand(helpCmd)
    helpCmd.AddCommand(helpFunctionsCmd)
    helpCmd.AddCommand(helpConstantsCmd)
}
```

**After:**
```go
func init() {
    helpFunctionsCmd.GroupID = "topics"
    helpConstantsCmd.GroupID = "topics"
    rootCmd.AddCommand(helpFunctionsCmd, helpConstantsCmd)
}
```

The `helpFunctionsCmd`, `helpConstantsCmd`, and all helper functions (`printFunctions()`, `printConstants()`, etc.) were kept unchanged.

### 3. help_test.go — Replace overview test with flag presence test

```go
func TestRootHelpShowsLocaleFlag(t *testing.T) {
    output := captureStdout(t, func() {
        rootCmd.SetArgs([]string{"--help"})
        _ = rootCmd.Execute()
    })

    if !strings.Contains(output, "--locale") {
        t.Error("root help output missing --locale flag")
    }
    if !strings.Contains(output, "--color-mode") {
        t.Error("root help output missing --color-mode flag")
    }
}
```

## Verification

```bash
cm help             # Shows commands, flags (including --locale), and "Help Topics:" group
cm help functions   # Cobra routes to cm functions --help, shows function list
cm functions        # Also works directly as a top-level command
cm constants        # Works directly
cm --help           # Same as cm help
task test           # All 26 packages pass
```

## Prevention

- **Prefer framework features over custom implementations.** Cobra handles help rendering, flag inheritance, and command routing. Custom help commands duplicate this and inevitably drift.
- **If customization is needed, use Cobra's template system** (`cmd.SetUsageTemplate()`, `cmd.SetHelpTemplate()`) rather than replacing the help command entirely.
- **Use Cobra command groups** (v1.6+) to organize domain-specific topics alongside standard commands.
- **Test flag presence in help output.** `TestRootHelpShowsLocaleFlag` catches this class of bug — if a new persistent flag is added but doesn't appear in help, something is wrong.

## Related Documentation

- [docs/solutions/ui-bugs/locale-formatting-bypass-in-tui.md](../ui-bugs/locale-formatting-bypass-in-tui.md) — Locale formatting bypasses in TUI
- [docs/plans/2026-02-27-refactor-cli-help-subcommand-cleanup-plan.md](../../plans/2026-02-27-refactor-cli-help-subcommand-cleanup-plan.md) — CLI help subcommand cleanup plan
- [docs/plans/2026-03-02-feat-unified-locale-aware-formatting-plan.md](../../plans/2026-03-02-feat-unified-locale-aware-formatting-plan.md) — Unified locale-aware formatting plan
