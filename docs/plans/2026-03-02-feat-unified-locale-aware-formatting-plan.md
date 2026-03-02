---
title: "feat: Unified locale-aware formatting system"
type: feat
status: active
date: 2026-03-02
deepened: 2026-03-02
brainstorm: docs/brainstorms/2026-03-02-unified-formatting-brainstorm.md
---

# Unified Locale-Aware Formatting System

## Enhancement Summary

**Deepened on:** 2026-03-02
**Research agents used:** architecture-strategist, pattern-recognition-specialist, performance-oracle, security-sentinel, code-simplicity-reviewer, best-practices-researcher, Context7 (golang.org/x/text docs)

### Key Improvements
1. Restructured from 6 phases to 4 (merged refactors, extracted independent unit cleanup)
2. Decoupled `AlignResults` from formatting — returns raw `types.Type`, formatters apply display independently
3. Pass `*display.Formatter` through `format.Options` (not `DisplayConfig`) — one mechanism everywhere
4. Added concrete implementation patterns: probe technique for separator lookup, `decimal.StringFixed()` pattern for precision-safe formatting
5. Removed `Precision` field (YAGNI), reduced launch locales from 4 to 3
6. Added security hardening: locale input validation, string length guards, U+202F invariant tests
7. Added P0 performance fixes: 3 per-call map allocations to hoist, pre-allocate slices, `defaultFormatter` singleton

### New Considerations Discovered
- `golang.org/x/text/internal/number.InfoFromTag` provides direct separator lookup but is an internal package — use probe technique on `message.NewPrinter` instead
- No existing Go library combines K/M/B suffixes with locale-aware decimals — custom implementation required
- `getCurrencyDecimals()`, `abbreviateTimeUnit()`, and `normalizeUnitSymbol()` each allocate a map on every call — fix as prerequisite
- `golang.org/x/text@v0.33.0` already in `go.mod` — no new module dependency, just deeper usage

---

## Overview

Centralize and make consistent how numbers, quantities, currencies, and rates are formatted across all output formats (text, markdown, HTML, JSON) and the TUI (editor preview, REPL). Add user-configurable locale via `--locale` CLI flag and Viper config file. Use `golang.org/x/text` for locale data with custom calcmark-specific formatting logic.

## Problem Statement

Formatting of calculated values is inconsistent across output channels:

- **Threshold mismatch:** Plain numbers get K suffix at 1,000; currency waits until $10,000
- **JSON inconsistency:** `JSONBlock.Output` uses raw `block.LastValue().String()` while `JSONResult.Value` uses `display.Format()` — consumers see `"$6500.00"` and `"$6,500.00"` for the same value
- **No thousand separators** for plain numbers (only mid-range currency gets them)
- **No locale support:** Decimal/thousand separators hardcoded to US conventions
- **Duplicate unit aliases:** `spec/units/canonical.go` is the canonical unit spec but `format/display/normalize.go` maintains a parallel alias system with divergent symbols (e.g., `sq m` vs `m²`)
- **Result iteration duplication:** 4 formatters independently implement source-line/result-index alignment with a documented history of index bugs
- **Precision loss:** `display.Format*` functions convert `decimal.Decimal` to `float64` for formatting
- **Per-call map allocations:** `getCurrencyDecimals()`, `abbreviateTimeUnit()`, and `normalizeUnitSymbol()` each allocate a new map on every call (P0 performance fix)

## Proposed Solution

A `Formatter` struct that holds a `DisplayConfig` with all formatting preferences. It flows explicitly through the system via `format.Options` — no globals, no hidden state. `golang.org/x/text` provides locale data (separators, currency info); custom Go logic handles calcmark-specific formatting (K/M/B suffixes, unit normalization, napkin estimates).

## Technical Approach

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│  Config Layer                                                   │
│  cmd/calcmark/config/types.go                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ Config.Locale string  (top-level, e.g., "de-DE")        │    │
│  │ Loaded from: --locale flag > config.toml > en-US        │    │
│  └─────────────────────────────────────────────────────────┘    │
├─────────────────────────────────────────────────────────────────┤
│  Display Layer                                                  │
│  format/display/config.go (NEW)                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ DisplayConfig {                                         │    │
│  │   Tag          language.Tag    // parsed locale          │    │
│  │   DecimalSep   string          // "." or ","             │    │
│  │   ThousandSep  string          // "," or "." or "\u202F" │    │
│  │ }                                                        │    │
│  │ DefaultConfig() → en-US                                  │    │
│  │ NewConfig(locale string) → (DisplayConfig, error)        │    │
│  └─────────────────────────────────────────────────────────┘    │
│  format/display/formatter.go (NEW)                              │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ Formatter struct { cfg DisplayConfig }  // value type    │    │
│  │ NewFormatter(cfg DisplayConfig) Formatter                │    │
│  │ (f Formatter) Format(t types.Type) string                │    │
│  │ (f Formatter) FormatNumber(v decimal.Decimal) string     │    │
│  │ (f Formatter) FormatCurrency(c *types.Currency) string   │    │
│  │ (f Formatter) FormatQuantity(q *types.Quantity) string   │    │
│  │ (f Formatter) FormatRate(r *types.Rate) string           │    │
│  └─────────────────────────────────────────────────────────┘    │
│  format/display/display.go (EXISTING - backward compat)         │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ var defaultFormatter Formatter  // init() singleton      │    │
│  │ Format(t types.Type) string → defaultFormatter.Format(t) │    │
│  └─────────────────────────────────────────────────────────┘    │
├─────────────────────────────────────────────────────────────────┤
│  Formatter Layer                                                │
│  format/formatter.go                                            │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ Options {                                               │    │
│  │   Verbose, IncludeErrors, Template     (existing)        │    │
│  │   Formatter  display.Formatter         (NEW)             │    │
│  │ }                                                        │    │
│  └─────────────────────────────────────────────────────────┘    │
│  format/align.go (NEW)                                          │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ AlignResults(block) []AlignedStatement                   │    │
│  │ Pure alignment — no display dependency                   │    │
│  └─────────────────────────────────────────────────────────┘    │
├─────────────────────────────────────────────────────────────────┤
│  TUI Layer                                                      │
│  cmd/calcmark/tui/editor/ and tui/repl/                         │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ Model stores display.Formatter (value type)              │    │
│  │ Created from config.Get().Locale at startup              │    │
│  │ Same type as Options.Formatter — one mechanism           │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

### Research Insights: Key Design Decisions

**Formatter as value type (~64 bytes):** `DisplayConfig` contains a `language.Tag` (24 bytes), 2 strings (16 bytes each), totaling ~56 bytes. Value type avoids heap allocation and GC pressure. Stack-allocatable. The `defaultFormatter` singleton in `display.go` avoids per-call construction.

**`format.Options` carries `Formatter`, not `DisplayConfig`:** The caller (CLI command or TUI model) creates the `Formatter` once and passes it everywhere. No formatter needs to construct its own. TUI and batch formatters use the same mechanism — one way to get formatted output.

**Separator lookup via probe technique:** Format a known probe value `1234.5` through `message.NewPrinter(tag)` at `NewConfig()` time to extract locale separators. Uses only the public `golang.org/x/text` API, backed by real CLDR data. Runs once at config construction, not per-format call.

```go
// extractSeparators discovers separators by formatting a probe value.
func extractSeparators(tag language.Tag) (decimalSep, thousandSep string) {
    p := message.NewPrinter(tag)
    formatted := p.Sprintf("%v", number.Decimal(1234.5))
    // "1,234.5" (en-US), "1.234,5" (de-DE), "1\u202F234,5" (fr-FR)
    // Extract separators by position relative to known digits.
    idx4 := strings.IndexByte(formatted, '4')
    idx5 := strings.LastIndexByte(formatted, '5')
    if idx4 >= 0 && idx5 > idx4+1 {
        decimalSep = formatted[idx4+1 : idx5]
    }
    idx1 := strings.IndexByte(formatted, '1')
    idx2 := strings.IndexByte(formatted, '2')
    if idx1 >= 0 && idx2 > idx1+1 {
        thousandSep = formatted[idx1+1 : idx2]
    }
    return
}
```

**Decimal formatting without float64:** Use `decimal.StringFixed()` + string manipulation (the `leekchan/accounting` pattern). Split on `"."`, insert thousand separators right-to-left into the integer part, rejoin with locale decimal separator. ~2x slower than float64 but preserves precision.

```go
func insertGroupSeparators(digits, thousSep string) string {
    n := len(digits)
    if n <= 3 { return digits }
    var b strings.Builder
    b.Grow(n + (n/3)*len(thousSep))
    remainder := n % 3
    if remainder > 0 {
        b.WriteString(digits[:remainder])
        if n > remainder { b.WriteString(thousSep) }
    }
    for i := remainder; i < n; i += 3 {
        b.WriteString(digits[i : i+3])
        if i+3 < n { b.WriteString(thousSep) }
    }
    return b.String()
}
```

**AlignResults decoupled from formatting:** Returns `[]AlignedStatement` with raw `types.Type`. Each formatter applies `display.Formatter` to `Result` as needed. JSON can produce both `raw_value` (from `Result.String()`) and `display_value` (from `formatter.Format(Result)`) from a single `AlignResults` call.

```go
type AlignedStatement struct {
    Source       string
    Result       types.Type  // raw, unformatted — nil if no result
    Variable     string      // extracted from assignment expressions
    IsBlank      bool        // blank source line (no result)
    IsResultLine bool        // previous result comment (skip)
}

func AlignResults(block *document.CalcBlock) []AlignedStatement { ... }
```

### Implementation Phases

#### Phase 1: Formatter struct, AlignResults extraction, and P0 performance fixes

**Goal:** Introduce the core abstractions without changing any behavior. Fix per-call map allocations. All existing output remains identical (en-US default).

**Tasks:**

*P0 Performance Fixes (prerequisite):*
- [x] Hoist `getCurrencyDecimals()` map to a `switch` statement (`display.go:186`)
- [x] Hoist `abbreviateTimeUnit()` map to package-level `var` (`display.go:305`)
- [x] Hoist `normalizeUnitSymbol()` map to package-level `var` (`normalize.go:376`) — or unify with existing `aliasToCanonical`
- [x] Pre-allocate `results` slice in `GetLineResults()` with `make([]LineResult, 0, len(allLines))` (`results.go:30`)

*DisplayConfig and Formatter:*
- [x] Create `format/display/config.go` with `DisplayConfig` struct (Tag, DecimalSep, ThousandSep — no Precision field) and `DefaultConfig()` returning en-US defaults
- [x] Add `Validate() error` method on `DisplayConfig` — reject empty separators to prevent zero-value trap
- [x] Create `format/display/formatter.go` with `Formatter` value type wrapping `DisplayConfig`
- [x] Move all formatting logic from `display.go` free functions onto `Formatter` methods
- [x] Keep existing `display.Format()` as backward-compatible wrapper via package-level `defaultFormatter` singleton (constructed in `init()`, not per-call)
- [x] Add `Formatter display.Formatter` field to `format.Options` (not `DisplayConfig` — one mechanism for both formatters and TUI)

*AlignResults extraction:*
- [x] Create `format/align.go` with `AlignResults(block *document.CalcBlock) []AlignedStatement`
- [x] `AlignedStatement` struct: `Source`, `Result types.Type`, `Variable`, `IsBlank`, `IsResultLine`
- [x] Refactor `text_formatter.go`, `markdown_formatter.go`, `html_formatter.go`, `json_formatter.go` to use `AlignResults()`, each applying `opts.Formatter.Format()` to `Result` as needed
- [x] Add blank-line regression tests for each formatter (per institutional learning)
- [x] Add test for `AlignResults` directly with various source patterns

*Threading:*
- [x] Thread `Formatter` through all output formatter `Format()` implementations
- [x] All existing tests pass with zero output changes
- [x] Explicit test: `DefaultConfig()` produces identical output to current `display.Format()` for all existing test cases

**Files changed:**
- `format/display/config.go` (new)
- `format/display/formatter.go` (new)
- `format/display/display.go` (existing functions become wrappers, P0 fixes)
- `format/display/normalize.go` (hoist `normalizeUnitSymbol` map)
- `format/formatter.go` (Options gains Formatter field)
- `format/align.go` (new)
- `format/align_test.go` (new)
- `format/text_formatter.go`, `markdown_formatter.go`, `html_formatter.go`, `json_formatter.go` (use AlignResults + opts.Formatter)
- `cmd/calcmark/tui/editor/results.go` (pre-allocate slice)

**Success criteria:**
- `task test` passes — zero output changes
- `display.Format()` still works for all existing consumers
- `defaultFormatter` singleton — no per-call allocation
- New `Formatter` struct is tested independently
- Blank-line regression tests pass

---

#### Phase 2: Locale-aware separator formatting + JSON `raw_value`

**Goal:** `Formatter` produces locale-specific output. JSON gets explicit raw + formatted fields. `golang.org/x/text` wired in.

**Tasks:**

*Locale support:*
- [x] Implement `NewConfig(locale string) (DisplayConfig, error)` with:
  - Length bound (64 bytes) and ASCII-only validation before `language.Parse()` (security finding)
  - Probe technique to extract separators via `message.NewPrinter`
  - Invalid locale falls back to en-US with warning to stderr
- [x] Replace hardcoded `addThousandSeparators()` with locale-aware `insertGroupSeparators()` on Formatter
- [x] Replace hardcoded decimal separator in `formatSmallNumber()`, `formatCurrencyWithSeparators()`, etc.
- [ ] Eliminate `float64` conversion — use `decimal.StringFixed()` + string manipulation
- [x] Add string length guard (1000 chars) before separator insertion for pathological values (security finding)
- [x] K/M/B suffixes: English letters always; decimal separator within suffix numbers localizes (`1,5M` in de-DE)
- [x] Currency symbol positioning stays fixed — locale only affects separators
- [x] Napkin estimates (`~`) use locale separators
- [x] Frontmatter exchange rates in verbose output remain en-US

*JSON schema:*
- [x] Add `RawValue string` field to `JSONResult` struct (`json:"raw_value"`)
- [x] `RawValue` uses `types.Type.String()` — always ASCII, machine-readable
- [x] Existing `Value` field becomes locale-formatted via `opts.Formatter.Format()`
- [x] Add test asserting `raw_value` is always ASCII-only (no U+202F, no locale-specific characters)
- [x] Document the JSON schema in a comment on the struct

*Testing:*
- [x] Table-driven tests for en-US, de-DE, fr-FR covering: numbers, currency, quantities, rates, napkin, negatives, zero, very large, very small
- [x] Use `%q` format verb in test errors to make U+202F visible
- [x] Named constant `NoBreakSpace = "\u00a0"` for test readability (Go's x/text uses U+00A0, not U+202F)
- [x] Round-trip test: `DefaultConfig()` and `NewConfig("en-US")` produce identical output
- [ ] Test `go-runewidth` width calculation for U+202F in TUI context

**Locale behavior matrix:**

| Value | en-US | de-DE | fr-FR |
|-------|-------|-------|-------|
| 1234.56 | 1.23K | 1,23K | 1,23K |
| $1500.00 | $1,500.00 | $1.500,00 | $1\u202F500,00 |
| 100 JPY | 100 JPY | 100 JPY | 100 JPY |
| 1000000 | 1M | 1M | 1M |
| 50.5 kg | 50.5 kg | 50,5 kg | 50,5 kg |
| ~400 GB | ~400 GB | ~400 GB | ~400 GB |
| -999.99 | -999.99 | -999,99 | -999,99 |

**Note:** fr-FR thousand separator is CLDR narrow no-break space (U+202F), not a regular space. Tests must use the correct Unicode character. `golang.org/x/text@v0.33.0` already in `go.mod` — `language` and `number` subpackages require no new module dependency.

**Files changed:**
- `format/display/config.go` (NewConfig with locale parsing + validation)
- `format/display/formatter.go` (locale-aware separator methods)
- `format/display/display.go` (update internal helpers)
- `format/display/display_test.go` (locale-parameterized tests)
- `format/json_formatter.go` (add RawValue field, populate it)
- `format/json_formatter_test.go` (verify both fields, ASCII invariant)

**Success criteria:**
- Existing en-US tests pass unchanged
- New locale tests verify separator behavior for de-DE, fr-FR
- Invalid locale produces warning and falls back to en-US
- JSON `raw_value` is always ASCII
- `task test` passes

---

#### Phase 3: CLI flag and config integration

**Goal:** Users can configure locale via `--locale` flag and config file. TUI respects locale.

**Tasks:**
- [x] Add `Locale string` field to `Config` struct in `config/types.go` (top-level — locale is application-wide, will eventually affect input parsing too)
- [x] Add `locale = "en-US"` to `defaults.toml`
- [x] Add `--locale` persistent flag to root command in `root.go` (mirrors `--color-mode` pattern: direct flag var + manual override, NOT `viper.BindPFlag` per institutional learning)
- [x] In `PersistentPreRunE`: if `--locale` flag set, override `cfg.Locale`. Add comment documenting the precedence chain at the resolution site.
- [x] Wire `config.Get().Locale` → `display.NewConfig(locale)` → `display.NewFormatter(cfg)` → `format.Options.Formatter` in `eval.go` and `convert.go`
- [x] Wire into TUI: pass `display.Formatter` to `editor.Model` and `repl.Model` constructors (same value type as `Options.Formatter`)
- [x] Preserve locale across TUI mode switches (`switchMode()` in `app.go` must carry formatter through)
- [x] Add `--locale` to `-h` help text with examples
- [x] Config tests: verify locale loading from TOML, CLI override, default fallback. Use `t.TempDir()` and `Reload()` pattern per existing config tests.
- [x] Integration test: `cm eval --locale de-DE` produces German-formatted output

**Precedence order:** CLI flag > config.toml > en-US default

**Files changed:**
- `cmd/calcmark/config/types.go` (add Locale field)
- `cmd/calcmark/config/defaults.toml` (add locale default)
- `cmd/calcmark/cmd/root.go` (add --locale flag)
- `cmd/calcmark/cmd/eval.go` (wire locale into Options)
- `cmd/calcmark/cmd/convert.go` (wire locale into Options)
- `cmd/calcmark/tui/app.go` (pass Formatter to editor/REPL, preserve across mode switch)
- `cmd/calcmark/tui/editor/` (accept and use Formatter)
- `cmd/calcmark/tui/repl/model.go` (accept and use Formatter)
- `cmd/calcmark/config/config_test.go` (locale config tests)

**Success criteria:**
- `cm eval --locale de-DE file.cm` shows German-formatted numbers
- Config file locale setting works
- CLI flag overrides config file
- No flag defaults to en-US (current behavior)
- TUI mode switch preserves locale
- `task test` and `task quality` pass

---

#### Phase 4 (Independent): Unit alias consolidation

**Note:** This phase is independent of locale support. It can be done before, after, or in parallel with Phases 1-3. Extracted as a separate deliverable to reduce scope coupling.

**Goal:** Display layer derives unit data from `spec/units/canonical.go` with display-specific overrides. Eliminate duplicate alias maps.

**Tasks:**
- [ ] Add `ToBase decimal.Decimal` field to `spec/units/UnitMapping` — conversion factors are language-level knowledge
- [ ] Add `Family string` field to `spec/units/UnitMapping` (e.g., `"si_length"`, `"data_storage"`) — enables auto-grouping without display-side hardcoding
- [ ] Create `format/display/units.go` that builds `unitFamilies` from `spec/units/canonical.go` data at init time, grouping by `Family`
- [ ] Register display-specific symbol overrides in explicit map: `m2` → `m²`, `km2` → `km²`, `ft2` → `ft²`
- [ ] Remove hardcoded `normalizeUnitSymbol()` function and its 90+ line alias map from `normalize.go`
- [ ] Export `AbbreviateTimeUnit` from `spec/types/rate.go` — remove duplicate from `display.go`
- [ ] Verify all existing normalize tests pass with derived data
- [ ] Benchmark `NormalizeForDisplay` to ensure no performance regression

**Files changed:**
- `spec/units/canonical.go` (add ToBase, Family fields)
- `format/display/units.go` (new — build families from canonical)
- `format/display/normalize.go` (remove duplicate maps, use units.go)
- `format/display/display.go` (remove duplicate `abbreviateTimeUnit`, import from spec)
- `spec/types/rate.go` (export `AbbreviateTimeUnit`)
- `format/display/normalize_test.go` (verify, benchmark)

**Success criteria:**
- `spec/units/canonical.go` is the single source for unit data
- Display overrides are explicit and minimal
- No duplicate alias maps remain
- `NormalizeForDisplay` benchmark shows no regression
- `task test` passes

## Alternative Approaches Considered

1. **Formatter middleware (interface-based):** Output-oriented interface doesn't generalize to future input locale parsing. Would need a config struct anyway.
2. **Global config:** Package-level `SetLocale()` prevents concurrent formatting, makes testing harder, un-idiomatic Go.
3. **Full CLDR from day one:** Over-scoped. Minimal locale set covers the major separator conventions and is testable.
4. **`golang.org/x/text/internal/number` for separator lookup:** Direct API via `InfoFromTag(tag).Symbol(SymDecimal)` but uses an internal package with no compatibility guarantee. Proposal to expose as public API accepted ([#53872](https://github.com/golang/go/issues/53872)) but not yet shipped. Probe technique is safer.

See [brainstorm](../brainstorms/2026-03-02-unified-formatting-brainstorm.md) for full analysis.

## Acceptance Criteria

### Functional Requirements

- [ ] All output formats (text, markdown, HTML) produce identical human-readable strings for the same value and locale
- [ ] JSON output includes both `value` (locale-formatted) and `raw_value` (machine-readable, always ASCII) per result
- [ ] `--locale` CLI flag works on all subcommands (eval, convert)
- [ ] Config file `locale` setting persists across sessions
- [ ] CLI flag overrides config file setting
- [ ] Default behavior (no locale set) produces identical output to current behavior (en-US)
- [ ] Supported locales: en-US, de-DE, fr-FR
- [ ] Invalid locale produces warning to stderr and falls back to en-US
- [ ] K/M/B suffixes always use English letters regardless of locale
- [ ] TUI editor and REPL respect locale setting

### Non-Functional Requirements

- [ ] `display.Format()` backward compatible via `defaultFormatter` singleton (no per-call allocation)
- [ ] `Formatter` is a value type (~64 bytes, stack-allocatable)
- [ ] No performance regression in TUI rendering (formatter created once at startup)
- [ ] `decimal.Decimal` precision preserved — no `float64` conversion for separator insertion
- [ ] Locale string validated: max 64 bytes, ASCII-only, before reaching `language.Parse()`
- [ ] Numeric strings over 1000 chars skip separator insertion (pathological value guard)

### Quality Gates

- [ ] `task test` passes at each phase boundary
- [ ] `task quality` passes at completion
- [ ] Table-driven locale tests cover en-US, de-DE, fr-FR with `%q` format for non-ASCII visibility
- [ ] Round-trip test: `DefaultConfig()` == `NewConfig("en-US")`
- [ ] `raw_value` ASCII-only invariant test
- [ ] Blank-line regression tests for all formatters
- [ ] Benchmark for `NormalizeForDisplay` shows no regression
- [ ] Golden tests updated where formatting output changes

## Scope Exclusions

- **Locale-aware input parsing** in the lexer (future phase — `DisplayConfig` is designed to extend to this)
- **In-document locale directives** (e.g., `<!-- locale: de-DE -->`)
- **Duration normalization** (e.g., 90 min → 1.5 hr)
- **Date/time locale formatting** (future phase)
- **Boolean/time locale formatting** (explicitly excluded)
- **CalcMark round-trip format** locale formatting (`.cm` stays locale-independent for portability)
- **Environment variable detection** (`LANG`/`LC_ALL`) — consider in a follow-up
- **Currency symbol repositioning** per locale (symbol position stays fixed; only separators change)
- **Hot-reload of locale** mid-TUI-session (set at startup only)
- **Locale-adapted suffixes** (always K/M/B, never locale-specific abbreviations)
- **Unit alias consolidation** (Phase 4 is independent and can be delivered separately)
- **ja-JP locale** at launch (identical separator behavior to en-US; add when needed)
- **Configurable precision** (current behavior hardcoded; add when a user-facing control is needed)

## Dependencies & Prerequisites

- `golang.org/x/text` (language, number, message subpackages) — already in `go.mod` at v0.33.0
- No other new dependencies
- Phases 1-3 are sequential; Phase 4 is independent

## Risk Analysis & Mitigation

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Breaking external `display.Format()` consumers | Low | High | Package-level `defaultFormatter` singleton; this is a CLI app with no known external consumers |
| JSON schema change breaks downstream | Medium | High | Add `raw_value` alongside existing `value` — additive, not destructive |
| Golden test churn | High | Low | Phase 1 produces zero output changes; Phase 2+ updates tests deliberately |
| Performance regression in TUI | Low | Medium | Formatter created once at startup; P0 map allocation fixes; benchmark |
| `float64` precision loss during formatting | Known | Medium | Replace with `decimal.StringFixed()` + string manipulation |
| Viper `IsSet()` gotcha with defaults | Known | Medium | Direct flag var + manual override, not `viper.BindPFlag` (per institutional learning) |
| Locale string injection / DoS | Low | Medium | Length bound (64 bytes), ASCII-only validation before `language.Parse()` |
| U+202F display manipulation | Medium | Low | Document in output paths; `raw_value` is always ASCII; test `go-runewidth` width |
| Pathological number strings after float64 removal | Low | Medium | String length guard (1000 chars) before separator insertion |

## Documentation Plan

- [ ] Update `--help` text for `--locale` flag with examples
- [ ] Add locale section to site/content documentation
- [ ] Document supported locales and their behavior
- [ ] Document JSON schema change (`raw_value` field)
- [ ] Document that locale-formatted values may contain non-ASCII whitespace (U+202F for fr-FR)

## References & Research

### Internal References

- Brainstorm: `docs/brainstorms/2026-03-02-unified-formatting-brainstorm.md`
- Display formatting: `format/display/display.go:26` — central `Format()` function
- Unit normalization: `format/display/normalize.go` — duplicate alias maps
- Canonical units: `spec/units/canonical.go` — single source of truth
- Config loading: `cmd/calcmark/config/config.go:57` — Viper init
- Config types: `cmd/calcmark/config/types.go` — Config struct
- CLI flags: `cmd/calcmark/cmd/root.go:54` — persistent flag pattern
- Format options: `format/formatter.go:20` — Options struct
- JSON schema: `format/json_formatter.go:26-52` — JSON output types
- TUI editor results: `cmd/calcmark/tui/editor/results.go:175` — display.Format() call
- TUI REPL: `cmd/calcmark/tui/repl/model.go:123,398,492` — display.Format() calls
- TUI mode switch: `cmd/calcmark/tui/app.go:106` — switchMode()
- P0 map allocations: `format/display/display.go:186,305`, `format/display/normalize.go:376`

### Institutional Learnings

- Formatter index alignment bug: `docs/solutions/ui-bugs/tui-mode-transitions-formatter-indexing-and-bracketed-paste-fixes.md`
- Currency spacing: `docs/solutions/ui-bugs/currency-code-output-spacing.md`
- Viper IsSet gotcha: `docs/solutions/logic-errors/viper-isset-embedded-defaults-deprecation.md`
- Code organization: `docs/solutions/code-organization/split-view-go-into-cohesive-modules.md`
- Map ordering: `docs/solutions/logic-errors/go-maps-non-deterministic-ordering-frontmatter.md` — use ordered slices for deterministic iteration
- Validation at multiple entry points: `docs/solutions/logic-errors/exchange-rate-frontmatter-validation.md` — validate locale consistently across config + CLI

### External References

- [golang.org/x/text/number package](https://pkg.go.dev/golang.org/x/text/number) — public API for locale-aware number formatting
- [golang.org/x/text/language package](https://pkg.go.dev/golang.org/x/text/language) — BCP 47 tag parsing and matching
- [x/text/number: provide way to query number system (#53872)](https://github.com/golang/go/issues/53872) — accepted proposal for public separator query API
- [Keeping Your Modules Compatible (Go Blog)](https://go.dev/blog/module-compatibility) — backward compatibility patterns
- [leekchan/accounting](https://github.com/leekchan/accounting/blob/master/formatnumber.go) — `decimal.Decimal` string formatting without float64
- [CVE-2022-32149](https://github.com/advisories/GHSA-69ch-w2m2-3vjp) — DoS in `golang.org/x/text/language` (fixed in v0.3.8, project uses v0.33.0)
