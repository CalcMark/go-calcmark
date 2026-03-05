---
title: "Editor and config tests fail due to host user config leaking into test environment"
date: "2026-03-04"
category: "test-failures"
severity: "medium"
components: ["cmd/calcmark/tui/editor", "cmd/calcmark/config"]
tags: ["test-isolation", "color-mode", "golden-files", "catwalk", "environment-contamination"]
symptoms:
  - "Catwalk golden file mismatches in editor package when user has ~/.config/calcmark/config.toml"
  - "ANSI color codes in test output differ from expectations (light mode vs dark mode)"
  - "TestLoad_DefaultsOnly fails when user config.toml exists on host"
  - "Selection highlighting test fails on machines with light color_mode configured"
root_cause: "config.Load() called in init() functions reads the real ~/.config/calcmark/config.toml from the host system, contaminating editor and config tests with the user's color_mode preference instead of using default (dark) settings"
---

## Problem

10 tests across 2 packages (`cmd/calcmark/tui/editor` and `cmd/calcmark/config`) failed when the host machine had `~/.config/calcmark/config.toml` with `color_mode = "light"`. Catwalk golden files contained dark mode ANSI escape sequences, producing mismatches. The `TestLoad_DefaultsOnly` config test also failed because it expected the embedded default (`dark`) but got the user override (`light`).

Additionally, the selection highlighting test only checked for the dark mode selection color (`#1E3A5F` / `48;2;30;58;95`) but the rendering correctly used the light mode color (`#DBEAFE` / `48;2;219;234;254`), causing a false failure.

## Root Cause

`config.Load()` called in test `init()` functions (in `model_basics_test.go` and `function_result_display_test.go`) read the real user config from the host filesystem. Since `Load()` uses `sync.Once`, the first call set the global config state for the entire test process. All editor tests then rendered with light mode colors, but catwalk golden files expected dark mode ANSI codes.

The `init()` → `config.Load()` pattern is problematic because:
- `init()` runs before `TestMain`, making it impossible to isolate `HOME` in time
- `sync.Once` means the first (contaminated) load wins
- No mechanism to detect or prevent host config leakage

## Solution

### 1. Add `TestMain` for config isolation (central fix)

```go
// testmain_test.go
package editor

import (
    "os"
    "testing"
    "github.com/CalcMark/go-calcmark/cmd/calcmark/config"
)

func TestMain(m *testing.M) {
    tmpHome, err := os.MkdirTemp("", "calcmark-editor-test-*")
    if err != nil {
        panic("failed to create temp home: " + err.Error())
    }
    defer os.RemoveAll(tmpHome)
    os.Setenv("HOME", tmpHome)

    // Reload config with isolated HOME (overrides any init() Load calls).
    config.Reload()

    os.Exit(m.Run())
}
```

Even though `init()` runs before `TestMain`, calling `config.Reload()` resets the `sync.Once` and reloads from the isolated HOME (which has no user config), falling back to embedded defaults (dark mode).

### 2. Remove redundant `init()` config loading

Removed `func init() { config.Load() }` from `model_basics_test.go` and `function_result_display_test.go` since `TestMain` handles it.

### 3. Isolate per-test in config package

```go
func TestLoad_DefaultsOnly(t *testing.T) {
    t.Setenv("HOME", t.TempDir())
    cfg, err := Reload()
    // ...
}
```

### 4. Fix color mode assertion in selection highlighting test

Added light mode Selection color (`48;2;219;234;254`) alongside the existing dark mode check (`48;2;30;58;95`).

### 5. Regenerate stale catwalk golden files

Ran `go test -args -rewrite` to regenerate `testdata/autocomplete` golden file with isolated dark mode config. Updated `testdata/selection` to expect `StateDefault` (not `StateAutocomplete`) after typing 1 character, matching the current `minAutocompletePrefix=2` behavior.

## Prevention Strategies

- **Default to isolated HOME in every package that touches config.** Any package that reads `~/.config` or `$XDG_CONFIG_HOME` must override `HOME` before tests run. `TestMain` is the correct place.
- **Treat host environment leakage as a test bug.** If a test passes on your machine but fails elsewhere, suspect environment bleed before logic errors.
- **Prefer `TestMain` over `init()` for test setup.** `TestMain` provides full lifecycle control, cleanup hooks, and is discoverable (one per package).
- **Use `t.Setenv` for single-test isolation.** Go 1.17+ auto-restores the env after each test, but use `TestMain` when the entire package needs isolation.

### Checklist for new test files

- [ ] Does this package read user config, home directory, or XDG paths?
- [ ] If yes, does a `TestMain` already isolate `HOME`?
- [ ] Does the package cache config at package-load time (e.g., `init()` or package-level `var`)? If so, `TestMain` is mandatory.
- [ ] For ANSI color assertions, check both light and dark mode palette values.

## Cross-References

- Related solution: [Viper IsSet() pitfall](../logic-errors/viper-isset-embedded-defaults-deprecation.md) — similar config isolation concern
- Config loading pipeline: `config.go` `Load()` → `Reload()` (state reset with `sync.Once`)
- Color application: `config.go` `applyColorMode()` → `compat.HasDarkBackground`
- Palette colors: `config/theme/palette.go` — defines adaptive light/dark color pairs
- Catwalk testing guide: `cmd/calcmark/tui/editor/TESTING.md`
