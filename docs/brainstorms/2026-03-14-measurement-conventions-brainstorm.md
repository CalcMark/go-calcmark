# Measurement Conventions Brainstorm

**Date:** 2026-03-14
**Status:** Complete

## What We're Building

A system for handling units that have multiple real-world definitions depending on country or domain. Examples: US gallon (3.785 L) vs imperial gallon (4.546 L), avoirdupois ounce vs troy ounce, short ton vs long ton.

**Problem:** Today, CalcMark hardcodes all ambiguous units to US Customary definitions. A UK user writing `10 fl oz` gets 296 mL (US) when they mean 284 mL (imperial) — a ~4% error that compounds in recipes and engineering. There's no way to declare intent or even see which definition is being used.

**Goal:** Let users declare their measurement conventions (document-level or inline), default to US Customary for backwards compatibility, and make the active convention visible in formatted output — all without cluttering the source document.

## Why This Approach

The key insight is that unit ambiguity isn't a single US-vs-UK binary — it's **multiple independent axes**. A UK jeweler needs imperial volume AND troy mass. An Australian baker needs metric cups AND standard tablespoons. These choices are orthogonal.

Rather than a single `measurement: uk` flag, we model each axis independently. This avoids false coupling and lets users be precise about exactly what they mean.

## Key Decisions

### 1. Default behavior: US Customary (non-breaking)

Bare unit names (`gallon`, `ounce`, `pint`) resolve to US Customary definitions — same as today. No existing document changes behavior.

### 2. Frontmatter `measurement` directive (structured map)

```yaml
measurement:
  volume: imperial    # gallon, pint, fl oz, quart → imperial definitions
  mass: troy          # ounce, pound → troy definitions
  ton: long           # ton → long ton (2240 lb)
```

Each key is an independent axis. Only axes that differ from US defaults need to be specified.

### 3. Three axes to implement now, extensible for more

**Implement:**
- `volume`: `us` (default) | `imperial`
- `mass`: `avoirdupois` (default) | `troy`
- `ton`: `short` (default) | `long` | `metric`

**Design for later** (same pattern, just more registrations):
- `horsepower`: `mechanical` | `metric` | `electrical` | `boiler`
- `calorie`: `thermochemical` | `it`
- `tablespoon`: `standard` | `australian`

**Note on cup:** Cup is affected by the `volume` axis (US cup vs imperial cup) but also has metric (250 mL), US legal (240 mL), and Japanese (200 mL) variants that don't map to either system. For now, the `volume` axis handles the US/imperial split. A dedicated `cup` axis may be needed later only if metric/legal/Japanese cup variants are requested.

### 4. Inline prefix overrides

When a user needs to override the document convention for a single expression:

```
gold = 10 troy oz
milk = 2 imp pt
shipping = 5 short ton
```

Prefixes: `us`, `imp`/`imperial`, `troy`, `short`, `long`, `metric` before the unit name. These work regardless of frontmatter — always available as first-class unit aliases.

### 5. Formatter makes conventions visible in output (strict mode, default)

**Source stays clean. Output shows what convention is active.**

With `strict: true` (the default), the formatter annotates bare ambiguous units:

| Source | Frontmatter | Formatted Output |
|--------|-------------|-----------------|
| `2 oz` | (none) | `2 us oz` |
| `2 oz` | `mass: troy` | `2 troy oz` |
| `2 troy oz` | (any) | `2 troy oz` (already explicit, no change) |
| `1 gallon` | (none) | `1 us gal` |
| `1 gallon` | `volume: imperial` | `1 imp gal` |

With `strict: false`, bare units pass through unannotated (`2 oz` → `2 oz`). Explicitly qualified units are always shown as-is.

This is a **formatter concern**, not a parser concern. Controlled by the three-level precedence: `config.toml` < frontmatter < inline qualifier.

### 6. Config integration

Measurement defaults live in `config.toml` alongside locale and formatter settings:

```toml
[measurement]
# volume = "us"           # us | imperial
# mass = "avoirdupois"    # avoirdupois | troy
# ton = "short"           # short | long | metric
# strict = true           # annotate ambiguous units in output
```

`cm config` shows effective measurement settings. `cm config --create` includes the section with defaults commented out. Three-level precedence: `config.toml` (global) < frontmatter (document) < inline qualifier (expression).

### 7. No fork of martinlindhe/unit needed

The `martinlindhe/unit` library already supports all the variants we need (imperial gallon, troy ounce, metric horsepower, etc.). They just aren't registered in CalcMark's `canonical.go` / `conversion.go` yet. The existing `ConversionInfo` + `registerUnit()` abstraction handles this cleanly.

## Interaction with Existing Features

- **`convert_to`**: Works as before. `convert_to: imperial` + `measurement: { volume: imperial }` means source is interpreted as imperial AND converted to imperial (no-op for volume). The two directives are orthogonal: `measurement` controls input interpretation, `convert_to` controls output target.
- **`locale`**: Unchanged. Locale is number formatting (decimals, thousands). Measurement conventions are unit definitions. Fully independent.
- **Explicit conversions**: `10 us gal in imp gal` works regardless of frontmatter — inline qualifiers always take precedence.

## Ambiguous Units Research Summary

7 independent axes of unit ambiguity exist in real-world measurement:

| Axis | Options | Affected Units |
|------|---------|---------------|
| Volume | US Customary / Imperial | gallon, quart, pint, fl oz, gill, cup |
| Mass | Avoirdupois / Troy | ounce, pound, dram, grain |
| Ton | Short / Long / Metric | ton, hundredweight |
| Horsepower | Mechanical / Metric / Electrical / Boiler | hp |
| Calorie | Thermochemical / IT | calorie, BTU |
| Cup | US / US legal / Metric / Japanese | cup (overlaps Volume axis for US/imperial; see note in Key Decisions §3) |
| Tablespoon | Standard / Australian | tbsp |

Plus edge cases: barrel (4+ domain-specific definitions), carat/karat (different physical quantities).

## Resolved Questions

1. **Formatter qualifier style**: Lowercase — `2 us oz`, `2 troy oz`, `1 imp gal`. Consistent with CalcMark's lowercase unit convention.
2. **`convert_to` interaction**: Strictly independent. `convert_to` controls output target system, `measurement` controls input interpretation. They are orthogonal.
3. **Strict mode**: Defaults to `true`. Nested inside the `measurement` map as `strict: true/false`. When strict (default), the formatter annotates bare ambiguous units with their resolved convention (e.g. `2 oz` → `2 us oz`). Users can opt out with `strict: false` for cleaner output when the convention is obvious.
