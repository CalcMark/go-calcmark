# Unified Formatting System

**Date:** 2026-03-02
**Status:** Brainstorm

## What We're Building

A centralized, locale-aware formatting system for calcmark that ensures consistent rendering of numbers, quantities, currencies, and rates across all output formats (text, markdown, HTML, JSON) and the TUI preview pane. The system is configurable via a `--locale` CLI flag with a persistent default in cm's config file.

### Problem Statement

Today, formatting of values is inconsistent in several ways:

- **Threshold mismatch:** Plain numbers get K suffix at 1,000 but currency waits until $10,000
- **JSON dual representation:** `Output` uses raw `.String()` while `Results[].Value` uses `display.Format()` — consumers see `"$6500.00"` and `"$6,500.00"` for the same value
- **No thousand separators** for plain numbers (only mid-range currency)
- **No locale support:** Decimal/thousand separators are hardcoded to US conventions
- **Duplicate unit alias maps:** `spec/units/canonical.go` is not consumed by the display layer; `format/display/normalize.go` maintains its own parallel alias system
- **Duplicated result iteration logic** across four formatters
- **`display.Format*` functions convert to float64**, losing precision for large numbers

### Success Criteria

- All output formats produce identical human-readable strings for the same value (except JSON which explicitly provides both raw and formatted)
- Locale configuration (decimal sep, thousand sep, currency placement) works via `--locale` flag and config file
- `spec/units/canonical.go` is the single source of truth for unit data, with display-specific overrides layered on top
- K/M/B suffix behavior remains the default display style
- JSON output has explicit `raw_value` (machine-readable) and `display_value` (locale-formatted) fields
- No regressions in existing golden test outputs (updated to reflect new consistent formatting)
- System is designed so input-locale parsing can be added to the lexer later without rearchitecting

## Why This Approach

**Approach chosen: Centralized `DisplayConfig` struct (Approach 1)**

A `DisplayConfig` struct holds all formatting preferences (locale tag, decimal separator, thousand separator, currency symbol placement, K/M/B thresholds). It flows through the system via `format.Options`. The `display.Format()` family of functions accepts `DisplayConfig` as a parameter.

### Why not alternatives?

- **Formatter middleware (Approach 2):** An output-oriented interface doesn't generalize to input parsing. When input locale is added to the lexer, you'd need a config struct anyway — the interface adds indirection without carrying its weight across both directions.
- **Global config (Approach 3):** Global mutable state prevents concurrent formatting with different locales, makes testing harder, and is un-idiomatic Go.

### Technical approach

- **`golang.org/x/text`** for locale data and detection (CLDR-backed decimal/thousand separator lookup, currency data)
- **Custom formatting logic** for calcmark-specific needs: K/M/B suffixes, unit normalization, napkin estimate markers, rate formatting
- **`DisplayConfig` struct** lives in a shared location (possibly a new `locale` package or within `format/display/`) so it can later be consumed by the lexer for input parsing
- **Unit aliases:** Display layer consumes `spec/units/canonical.go` as base, then applies display-specific overrides (e.g., superscript area symbols) on top

## Key Decisions

1. **Consistency and locale together** — These are deeply intertwined; solving one properly requires solving both.
2. **K/M/B suffixes remain the default** — Users can opt into full numbers via locale/format settings, but K/M/B is the default human-friendly style.
3. **CLI flag + config file** — `--locale` flag on cm commands, persistent default in cm's config file. No in-document directives.
4. **JSON gets both representations** — Explicit `raw_value` (numeric, machine-readable) and `display_value` (locale-formatted string) fields. No ambiguity.
5. **Shared base + display overrides for units** — Display layer derives from `spec/units/canonical.go`, then applies display-specific symbol overrides.
6. **Hybrid Go approach** — `golang.org/x/text` for locale data; custom logic for calcmark-specific formatting.
7. **Output-only locale for now** — Input parsing stays US-style. `DisplayConfig` is designed so it can feed the lexer in a future phase.
8. **Approach 1: DisplayConfig struct** — Centralized config passed explicitly. No globals, no unnecessary interfaces. Naturally extends to input parsing.

## Scope

### In scope

- `DisplayConfig` struct with locale preferences
- Refactored `display.Format*` functions accepting config
- Consistent formatting across text, markdown, HTML formatters
- JSON formatter with explicit raw + formatted fields
- `--locale` CLI flag and config file persistence
- Display layer consuming `spec/units/canonical.go` with overrides
- Eliminating duplicate unit alias maps
- Extracting shared result iteration logic from formatters
- Updated golden tests

### Out of scope

- Locale-aware input parsing in the lexer (future phase)
- In-document locale directives
- Duration normalization (e.g., 90 min -> 1.5 hr)
- New output format types

## Resolved Questions

1. **What locales should be supported at launch?** Minimal set: en-US, de-DE, fr-FR, ja-JP. Covers the major separator conventions (comma-dot, dot-comma, no-separator). More added later.
2. **Config file format?** Viper is already in use. Locale setting integrates with the existing Viper-based config.
3. **Should K/M/B suffixes be locale-aware?** No. K/M/B are universal shorthand in calcmark regardless of locale. Locale only affects separators and currency placement.
4. **Precision policy:** Configurable via `DisplayConfig`. Default matches current behavior (up to 6 decimal places, trailing zeros trimmed, integers show no decimals). Users can override.

## Open Questions

None — all questions resolved during brainstorming.
