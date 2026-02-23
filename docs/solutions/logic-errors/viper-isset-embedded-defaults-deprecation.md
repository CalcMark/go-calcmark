---
title: "Remove dark_mode default from embedded config to fix repeated deprecation warnings"
date: 2026-02-22
category: logic-errors
tags:
  - config
  - deprecation
  - viper
  - defaults
  - backwards-compatibility
component: "cmd/calcmark/config/"
severity: high
symptoms:
  - "Deprecation warning printed on every CLI invocation regardless of user config"
  - "Warning appears during cm convert, cm eval, and all other commands"
  - "Warning present even when no user config file exists"
---

# Viper `IsSet()` Treats Embedded Defaults as User-Provided Values

## Problem

Every `cm` invocation printed this warning to stderr:

```
calcmark: dark_mode is deprecated. Use color_mode="dark" or color_mode="light" instead.
```

This happened even with no user config file. The warning was supposed to fire only when a user's config explicitly set the deprecated `dark_mode` field.

## Root Cause

Viper's `IsSet()` method does not distinguish between values loaded from embedded defaults and values from user config files — it returns `true` for any loaded value regardless of source.

The embedded `defaults.toml` contained both the deprecated key and its replacement:

```toml
[tui]
color_mode = "dark"
dark_mode = true     # ← This caused the false positive
```

The warning logic in `warnDeprecatedColorMode()` checked `v.IsSet("tui.dark_mode")`, which returned `true` because the embedded default had set it — not because any user had.

## Investigation Steps

1. Ran `cm convert --to=md file.cm` and observed the warning on stderr
2. Traced the warning to `config.go:134` in `warnDeprecatedColorMode()`
3. Checked the condition: `v.IsSet("tui.dark_mode")` — returns true
4. Found `dark_mode = true` in the embedded `defaults.toml`
5. Confirmed: Viper's `IsSet()` conflates "value exists" with "user set it"

## Fix

**Removed `dark_mode = true` from `defaults.toml`** — only `color_mode = "dark"` remains as the active default:

```toml
# Before
[tui]
color_mode = "dark"
dark_mode = true

# After
[tui]
color_mode = "dark"
```

**Preserved backward compatibility** — the `DarkMode bool` field in `TUIConfig` (`types.go:14`) still exists so that user configs with `dark_mode` unmarshal correctly and trigger the deprecation warning.

**Updated test** — `TestLoad_DefaultsOnly` now expects `DarkMode == false` when loading only embedded defaults.

### Key Files

- `cmd/calcmark/config/defaults.toml` — removed `dark_mode = true`
- `cmd/calcmark/config/config.go:129-136` — `warnDeprecatedColorMode()` function (unchanged)
- `cmd/calcmark/config/config_test.go` — updated default assertion
- `cmd/calcmark/config/types.go:14` — `DarkMode` field kept for backward compat

## Verification

- `cm convert --show-template` outputs cleanly with no warning
- `cm convert --to=md file.cm` produces output with no warning
- All config tests pass including new `TestThemeOverride_EndToEnd`
- User configs with `dark_mode = true` still trigger the warning correctly

## Prevention Rules

- **Never include deprecated keys in embedded defaults.** Keep the struct field for backward-compat unmarshalling, but remove the key from `defaults.toml` immediately upon deprecation.
- **Don't rely on `v.IsSet()` for user-intent detection** when using embedded defaults. `IsSet()` returns true for all loaded values regardless of source.
- **Test deprecation warnings with real user config files only.** Write integration tests that create a temp config with the deprecated key and verify the warning fires — then verify it does NOT fire with only embedded defaults.

## General Principle

Viper's `IsSet()` conflates "value exists" with "user set it" when embedded defaults are used. Deprecated features should never live in embedded defaults; treat embedded defaults as the baseline implementation and reserve user configs as the only source for detecting user choices.

## See Also

- `docs/THEMING.md` — Color resolution flow and configuration hierarchy
- `docs/plans/2026-02-22-refactor-tui-theme-consistency-plan.md` — Phase 1 deprecated key detection
- `cmd/calcmark/config/config.go:109-126` — `warnDeprecatedThemeKeys()` uses `AllKeys()` instead of `IsSet()` (the correct pattern)
