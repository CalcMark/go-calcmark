---
title: "feat: Add cm config subcommand"
type: feat
status: completed
date: 2026-03-04
---

## Enhancement Summary

**Deepened on:** 2026-03-04
**Sections enhanced:** 7
**Research agents used:** security-sentinel, code-simplicity-reviewer, architecture-strategist, pattern-recognition-specialist, performance-oracle, best-practices-researcher, learnings-researcher

### Key Improvements from Deepening
1. **Dropped source attribution** — no popular CLI (gh, kubectl, hugo) does this; saves ~140 LOC of complexity
2. **Atomic file creation** with `os.OpenFile(O_CREATE|O_EXCL)` instead of stat-then-write (TOCTOU race fix)
3. **Respect `$XDG_CONFIG_HOME`** — proper XDG spec compliance in new path helpers
4. **Use `pelletier/go-toml/v2` Marshal** for show (already indirect dep via Viper, zero bloat)
5. **Reuse embedded `defaults.toml` verbatim** for `--create` template (simplest approach)
6. **Reduced to 5 focused tests** from 10 (avoid duplicating config package test coverage)
7. **Refactor `load()` to use new path helpers** — single source of truth for config paths

---

# feat: Add `cm config` subcommand

## Overview

Add a `cm config` command that prints the current effective configuration as valid TOML to stdout. Add a `--create` flag that scaffolds a commented-out baseline config file at the XDG path.

## Problem Statement / Motivation

Users have no way to inspect their resolved configuration. When debugging theme or locale issues, they must manually compare their config file against the defaults.toml in the source code. There is also no `cm`-native way to create a starter config file — the docs tell users to manually `mkdir -p` and `touch`.

## Proposed Solution

A single `cm config` command with a `--create` flag:

- **`cm config`** — Print the fully-resolved effective configuration (embedded defaults merged with user overrides and CLI flag overrides) as valid TOML to stdout. Status messages go to stderr.
- **`cm config --create`** — Write the embedded `defaults.toml` (with `key = value` lines commented out) to the XDG config path. Refuses to overwrite an existing file (atomic `O_CREATE|O_EXCL`). Prints confirmation to stderr and the created file's contents to stdout.

### Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Output format | Valid TOML to stdout | Pipe-friendly (`cm config > backup.toml`), consistent with config file format |
| Source attribution | **None** — just print resolved values | No popular CLI does per-key attribution (gh, kubectl, hugo skip it). Eliminates ~140 LOC. Add later if users request. |
| `--create` target path | `$XDG_CONFIG_HOME/calcmark/config.toml` (falls back to `~/.config/calcmark/config.toml`) | Proper XDG spec compliance |
| `--create` + existing file | Atomic error via `O_CREATE\|O_EXCL`, exit 1 | Prevents data loss, eliminates TOCTOU race (CWE-367) |
| CLI flag interaction | Show resolved config WITH flag overrides applied | Shows "what the current session sees" |
| OS-specific defaults | Same defaults on all platforms | No OS-specific values exist today; keep simple |
| Deprecated keys in output | Omit from TOML output | Per institutional learning: deprecated keys only in user configs |
| Colorization | None | Plain TOML, consistent with `cm version` |
| `--create` template source | Embedded `defaults.toml` with values commented out | Runtime line-by-line transformation (~15 lines of code) |
| TOML serialization for show | `pelletier/go-toml/v2` Marshal | Already indirect dep via Viper; struct-driven stays in sync with `types.go` |
| Run vs RunE | `RunE` | Command performs I/O that can fail; matches `eval.go`, `convert.go` pattern |
| Short flag for `--create` | None (`BoolVar` not `BoolVarP`) | No natural single-letter; matches `convert.go` `--show-template` precedent |

## Technical Approach

### New Files

#### `cmd/calcmark/cmd/config.go`

```go
package cmd

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/CalcMark/go-calcmark/cmd/calcmark/config"
    toml "github.com/pelletier/go-toml/v2"
    "github.com/spf13/cobra"
)

var configCreate bool

var configCmd = &cobra.Command{
    Use:   "config",
    Short: "Print or create CalcMark configuration",
    Long: `Print the current effective configuration as TOML.

With --create, write a starter config file to the XDG config path
with all values commented out and descriptive comments included.`,
    RunE: func(cmd *cobra.Command, args []string) error {
        if configCreate {
            return runConfigCreate()
        }
        return runConfigShow()
    },
}

func init() {
    configCmd.Flags().BoolVar(&configCreate, "create", false,
        "Create a starter config file at ~/.config/calcmark/config.toml")
    rootCmd.AddCommand(configCmd)
}
```

#### `cmd/calcmark/cmd/config_test.go`

Test cases (TDD — write before implementation):

```
TestConfigShow_DefaultsOnly    — No user config, output is valid TOML containing all expected keys
TestConfigCreate_NewFile       — Creates dir + file, file contains commented-out values, stdout has contents
TestConfigCreate_ExistingFile  — Errors (does not modify existing file)
TestConfigCreate_ExistingDir   — Dir exists, creates file successfully
TestConfigCreate_NoHome        — $HOME unset, errors gracefully
```

### Research Insights: Test Strategy

- **Do NOT re-test config merge logic** — the config package already tests merge precedence in `config_test.go`
- **Do NOT test CLI flag override effects** — that's `root.go`'s `PersistentPreRunE` responsibility
- **DO test the command's own responsibilities**: TOML serialization format, file creation safety, overwrite protection
- Use `captureStdout()` from `help_test.go:125-145` and `t.Setenv("HOME", ...)` + `config.Reload()` from `config_test.go`

### Config Package Changes

#### `cmd/calcmark/config/config.go` — Expose helpers and refactor `load()`

```go
// DefaultsTOML returns the raw embedded defaults.toml content.
func DefaultsTOML() string {
    return defaultsToml
}

// XDGConfigPath returns the XDG config file path for the current user.
// Respects $XDG_CONFIG_HOME per the XDG Base Directory Specification.
// Falls back to ~/.config/calcmark/config.toml if not set.
func XDGConfigPath() (string, error) {
    if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
        return filepath.Join(xdg, "calcmark", "config.toml"), nil
    }
    home, err := os.UserHomeDir()
    if err != nil || home == "" {
        return "", fmt.Errorf("cannot determine home directory: %w", err)
    }
    return filepath.Join(home, ".config", "calcmark", "config.toml"), nil
}

// FallbackConfigPath returns the legacy dotfile config path (~/.calcmarkrc.toml).
func FallbackConfigPath() (string, error) {
    home, err := os.UserHomeDir()
    if err != nil || home == "" {
        return "", fmt.Errorf("cannot determine home directory: %w", err)
    }
    return filepath.Join(home, ".calcmarkrc.toml"), nil
}
```

### Research Insights: Config Package

- **Refactor `load()` to call `XDGConfigPath()` and `FallbackConfigPath()`** internally. Currently `load()` computes paths inline (lines 71-82). After the change, there is a single source of truth for path computation.
- **`DefaultsTOML()` is a function, not an exported var** — matches the `format.DefaultHTMLTemplate()` precedent in `convert.go`.
- **`$XDG_CONFIG_HOME` support must also be added to `load()`** for consistency. If `--create` writes to `$XDG_CONFIG_HOME/calcmark/config.toml`, `load()` must also read from there.

### Implementation Logic

#### `runConfigShow()`

1. Call `config.Get()` to get the resolved config (already loaded by `PersistentPreRunE` with CLI overrides applied)
2. Add a TOML header comment to stdout (`# CalcMark effective configuration`)
3. Marshal with `pelletier/go-toml/v2` — produces struct-ordered TOML that stays in sync with `types.go`
4. Write to stdout

```go
func runConfigShow() error {
    cfg := config.Get()
    fmt.Println("# CalcMark effective configuration")
    fmt.Println()
    bytes, err := toml.Marshal(cfg)
    if err != nil {
        return fmt.Errorf("failed to marshal config: %w", err)
    }
    os.Stdout.Write(bytes)
    return nil
}
```

### Research Insights: Show Command

- **No source attribution** — `gh config list`, `kubectl config view`, and `hugo config` all output resolved values without per-key provenance. If users need to know why a value is set, they inspect their config file directly.
- **Struct-driven marshal** via `pelletier/go-toml/v2` keeps output in sync with `types.go` automatically as fields are added/removed. No manual key list to maintain.
- **Deprecated fields (`DarkMode`)**: The `mapstructure:"dark_mode"` tag on `TUIConfig.DarkMode` will appear in TOML output only if `pelletier/go-toml` respects the tag. Verify during implementation — may need a `toml:"-"` tag to suppress it, or use a separate output struct that excludes deprecated fields.

#### `runConfigCreate()`

1. Call `config.XDGConfigPath()` to get target path
2. Create parent directory with `os.MkdirAll(dir, 0755)`
3. Use `os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)` for atomic creation
4. Transform `config.DefaultsTOML()`: comment out `key = value` lines, replace header
5. Write transformed content to file
6. Print confirmation to stderr: `Created <path>`
7. Print file contents to stdout

```go
func runConfigCreate() error {
    path, err := config.XDGConfigPath()
    if err != nil {
        return err
    }

    // Create parent directory
    if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
        return fmt.Errorf("cannot create config directory: %w", err)
    }

    // Atomic create — fails if file already exists (O_EXCL)
    f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
    if err != nil {
        if os.IsExist(err) {
            return fmt.Errorf("config file already exists: %s", path)
        }
        return fmt.Errorf("cannot create config file: %w", err)
    }
    defer f.Close()

    content := commentOutDefaults(config.DefaultsTOML())
    if _, err := f.WriteString(content); err != nil {
        return fmt.Errorf("failed to write config: %w", err)
    }

    fmt.Fprintf(os.Stderr, "Created %s\n", path)
    fmt.Print(content)
    return nil
}
```

#### `commentOutDefaults()`

Transform the embedded `defaults.toml` into a user-facing template:

```go
func commentOutDefaults(defaults string) string {
    var b strings.Builder
    b.WriteString("# CalcMark Configuration\n")
    b.WriteString("# Uncomment and modify values below to customize.\n")
    b.WriteString("# See: https://calcmark.dev/docs/configuration/\n")

    // Skip the original header block (lines starting with # before first blank line or key)
    lines := strings.Split(defaults, "\n")
    pastHeader := false
    for _, line := range lines {
        if !pastHeader {
            if strings.HasPrefix(line, "#") || line == "" {
                if line == "" { pastHeader = true; b.WriteString("\n") }
                continue
            }
            pastHeader = true
        }

        trimmed := strings.TrimSpace(line)
        switch {
        case trimmed == "":
            b.WriteString("\n")
        case strings.HasPrefix(trimmed, "#"):
            b.WriteString(line + "\n")          // Keep existing comments
        case strings.HasPrefix(trimmed, "["):
            b.WriteString("\n" + line + "\n")    // Keep section headers
        default:
            b.WriteString("# " + line + "\n")   // Comment out key = value
        }
    }
    return b.String()
}
```

### Research Insights: Create Command

- **`O_CREATE|O_EXCL` is the industry-standard pattern** for safe file creation. Eliminates TOCTOU race (CWE-367) that exists with stat-then-write. Used by `gh` CLI, ssh-keygen, and gpg.
- **Reusing `defaults.toml` verbatim** (with line transformation) is the simplest approach. The alternative — a second embedded file — creates a synchronization burden with no benefit.
- **File permissions**: `0644` for the config file, `0755` for directories. Standard for non-secret config. Matches existing test patterns in `config_test.go`.
- **No `--force` flag in v1** — if the file exists, error out. Users who want to reset can delete the file manually. Adding `--force` is trivial later if requested.

### Example Output

#### `cm config` (no user config)

```toml
# CalcMark effective configuration

locale = 'en-US'

[tui]
color_mode = 'dark'

[tui.theme]
primary = ''
accent = ''
error = ''
warning = ''
muted = ''
dimmed = ''
output = ''
bright = ''
separator = ''
source_pane_bg = ''
preview_pane_bg = ''
status_bar_bg = ''

[formatter]
verbose = false
include_errors = true
default_format = 'text'
```

#### `cm config --create` (generated file)

```toml
# CalcMark Configuration
# Uncomment and modify values below to customize.
# See: https://calcmark.dev/docs/configuration/

# Display locale for number formatting (decimal/thousand separators).
# Supported: "en-US", "de-DE", "fr-FR"
# locale = "en-US"

[tui]
# Color mode: "light" or "dark"
# Set to "light" if you use a light terminal background.
# color_mode = "dark"

[tui.theme]
# These overrides replace the corresponding palette color for your color_mode.
# Leave empty to use the built-in adaptive palette defaults.

# Primary brand color - titles, prompts, variable names
# primary = ""
# Accent color - borders, highlights
# accent = ""
# Error messages
# error = ""
# Changed/modified indicator
# warning = ""
# Help text, secondary info
# muted = ""
# Hints, suggestions, preview text
# dimmed = ""
# Calculation results
# output = ""
# Syntax emphasis
# bright = ""
# Divider lines
# separator = ""

# Pane backgrounds (leave empty for palette defaults)
# source_pane_bg = ""
# preview_pane_bg = ""
# status_bar_bg = ""

[formatter]
# verbose = false
# include_errors = true
# default_format = "text"
```

## Acceptance Criteria

- [x] `cm config` prints resolved TOML config to stdout (valid TOML, parseable by any TOML library)
- [x] `cm config --create` creates config file at `$XDG_CONFIG_HOME/calcmark/config.toml` (or `~/.config/calcmark/config.toml`) with commented-out values
- [x] `cm config --create` atomically errors if the file already exists (exit 1, no partial write)
- [x] `cm config --create` creates the directory if it does not exist
- [x] `cm config --create` prints file contents to stdout, confirmation to stderr
- [x] All tests pass (`task test`)
- [x] `cm config` appears in `cm --help` output
- [x] Site docs updated: `cli-reference.md` and `configuration.md`
- [x] `load()` refactored to use `XDGConfigPath()` and `FallbackConfigPath()` internally
- [x] Deprecated `DarkMode` field excluded from `cm config` TOML output

## Dependencies & Risks

**Dependencies:**
- `github.com/pelletier/go-toml/v2` — promote from indirect to direct dependency (already in go.mod via Viper, zero binary size impact)

**Risks:**
- **Low:** `pelletier/go-toml/v2` Marshal field ordering. Mitigation: struct fields are marshaled in declaration order, which matches the natural grouping. Verify during implementation.
- **Low:** Deprecated `DarkMode bool` field appearing in marshal output. Mitigation: add `toml:"-"` tag to suppress, or use a separate output-only struct.

**Institutional learnings applied:**
- Never include deprecated keys in embedded defaults (`viper-isset-embedded-defaults-deprecation.md`)
- Prefer Cobra's built-in help system for the new command (`custom-help-hardcoding-flags.md`)
- Go maps have non-deterministic ordering (`go-maps-non-deterministic-ordering-frontmatter.md`) — irrelevant here since we use struct-based marshal, but confirms the decision to avoid map-based TOML generation.

## Implementation Order

1. Add `toml:"-"` tag to `TUIConfig.DarkMode` in `types.go` (suppress deprecated field in marshal)
2. Add `DefaultsTOML()`, `XDGConfigPath()`, `FallbackConfigPath()` to `config/config.go`
3. Refactor `load()` to call the new path helpers (single source of truth)
4. Write tests in `cmd/calcmark/cmd/config_test.go` (TDD — 5 tests)
5. Implement `cmd/calcmark/cmd/config.go` with `runConfigShow()` and `runConfigCreate()`
6. Run `task test` and iterate
7. Update `site/content/docs/cli-reference.md` with `cm config` section
8. Update `site/content/docs/configuration.md` to reference `cm config --create`
9. Run `task quality`

## References & Research

### Internal References

- Config loading: `cmd/calcmark/config/config.go:56-104`
- Config types: `cmd/calcmark/config/types.go:6-68`
- Embedded defaults: `cmd/calcmark/config/defaults.toml`
- Cobra command pattern: `cmd/calcmark/cmd/version.go` (simplest example)
- CLI flag integration: `cmd/calcmark/cmd/root.go:53-87`
- `captureStdout()` test utility: `cmd/calcmark/cmd/help_test.go:125-145`
- Config test patterns: `cmd/calcmark/config/config_test.go`
- `format.DefaultHTMLTemplate()` precedent: `cmd/calcmark/cmd/convert.go:37`

### Institutional Learnings

- `docs/solutions/logic-errors/viper-isset-embedded-defaults-deprecation.md` — Never include deprecated keys in defaults
- `docs/solutions/code-organization/custom-help-hardcoding-flags.md` — Prefer Cobra built-in help
- `docs/solutions/logic-errors/go-maps-non-deterministic-ordering-frontmatter.md` — Use struct ordering, not map iteration

### External Research

- **gh config list**: Outputs resolved key=value pairs, no source attribution
- **kubectl config view**: Outputs merged YAML, supports `--output` format flag
- **hugo config**: Outputs resolved TOML, supports `--format` flag (toml/yaml/json)
- **pelletier/go-toml/v2**: Supports `comment` struct tags and `toml:"-"` for field exclusion
- **Atomic file creation**: `O_CREATE|O_EXCL` is the POSIX-standard pattern (CWE-367 mitigation)

### User-Facing Docs

- `site/content/docs/configuration.md` — Will add `cm config --create` mention in Quick Start
- `site/content/docs/cli-reference.md` — Will add `cm config` section
