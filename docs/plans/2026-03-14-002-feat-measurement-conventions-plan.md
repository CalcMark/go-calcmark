---
title: "feat: Measurement conventions with document-level and inline overrides"
type: feat
status: active
date: 2026-03-14
origin: docs/brainstorms/2026-03-14-measurement-conventions-brainstorm.md
---

# Measurement Conventions

## Overview

Add support for ambiguous units that have multiple real-world definitions depending on country or domain. US gallon (3.785 L) vs imperial gallon (4.546 L), standard ounce (28.35g) vs troy ounce (31.10g), short ton (2000 lb) vs long ton (2240 lb).

Users declare their measurement conventions via frontmatter (document-level), config.toml (global), or inline prefixes (per-expression). Bare ambiguous units default to US Customary for backwards compatibility. The formatter annotates ambiguous units in output so readers always know which definition is active.

## Problem Statement

Today, CalcMark hardcodes all ambiguous units to US Customary definitions. A UK user writing `10 fl oz` gets 296 mL (US) when they mean 284 mL (imperial) — a ~4% error that compounds in recipes and engineering. There's no way to declare intent or even see which definition is being used (see brainstorm: docs/brainstorms/2026-03-14-measurement-conventions-brainstorm.md).

## Proposed Solution

Model each axis of ambiguity independently rather than a single `measurement: uk` flag. This avoids false coupling — a UK jeweler needs imperial volume AND troy mass (see brainstorm: §Why This Approach).

### Three axes to implement now

| Axis | Options | Default | Affected Units |
|------|---------|---------|---------------|
| `volume` | `us` / `imperial` | `us` | gallon, quart, pint, fl oz, gill, cup |
| `mass` | `standard` / `troy` | `standard` | ounce, pound, dram, grain |
| `ton` | `short` / `long` / `metric` | `short` | ton, hundredweight |

**Note on `standard`:** "Standard" means avoirdupois — the everyday weight system where 1 lb = 16 oz, 1 oz = 28.35g. This is the system your bathroom scale and grocery store use. It's called "standard" to distinguish it from troy weight (used for precious metals: 1 troy oz = 31.10g). Code comments, diagnostics, and documentation must explain this clearly.

### Three-level precedence

`config.toml` (global default) < frontmatter `measurement:` (document) < inline qualifier (`troy oz`, `imp gal`)

### Frontmatter syntax

```yaml
measurement:
  volume: imperial    # gallon, pint, fl oz, quart → imperial definitions
  mass: troy          # ounce, pound → troy definitions
  ton: long           # ton → long ton (2240 lb)
  strict: false       # opt out of ambiguous unit annotation in output
```

Only axes that differ from US defaults need to be specified.

### Inline prefix overrides

Always available regardless of frontmatter:

```
gold = 10 troy oz
milk = 2 imp pt
shipping = 5 short ton
```

Prefixes: `us`, `imp`/`imperial`, `troy`, `short`, `long`, `metric` before the unit name.

### Formatter strict mode (default: true)

Source stays clean. Output shows which convention is active:

| Source | Frontmatter | Formatted Output |
|--------|-------------|-----------------|
| `2 oz` | (none) | `2 us oz` |
| `2 oz` | `mass: troy` | `2 troy oz` |
| `2 troy oz` | (any) | `2 troy oz` (already explicit) |
| `1 gallon` | (none) | `1 us gal` |
| `1 gallon` | `volume: imperial` | `1 imp gal` |

With `strict: false`, bare units pass through unannotated.

## Technical Approach

### Architecture

The feature spans 6 layers with a clear data flow:

```
config.toml → MeasurementConfig (global defaults)
                ↓
frontmatter → MeasurementConfig (document override)
                ↓
lexer → recognizes multi-word qualified units ("troy oz", "imp gal")
                ↓
conversion registry → resolves bare "oz" using active MeasurementConfig
                ↓
interpreter → passes MeasurementConfig through evaluation
                ↓
formatter → annotates bare ambiguous units in strict mode
```

### Implementation Phases

#### Phase 1: Register unit variants in spec layer

Register all imperial, troy, and ton variants that martinlindhe/unit already provides. No behavioral changes — just making the units available.

**Tasks:**

- [ ] Add `UnitMapping` entries for imperial volume variants to `spec/units/canonical.go`: `imp gal`/`imperial gallon`, `imp qt`/`imperial quart`, `imp pt`/`imperial pint`, `imp fl oz`/`imperial fluid ounce`, `imp cup`/`imperial cup`, `imp gill`/`imperial gill`
- [ ] Add `UnitMapping` entries for troy mass variants: `troy oz`/`troy ounce`, `troy lb`/`troy pound`
- [ ] Add `UnitMapping` entries for ton variants: `short ton`, `long ton`, `metric ton`/`metric tonne`
- [ ] Add `UnitMapping` entries for US-qualified aliases: `us gal`, `us qt`, `us pt`, `us fl oz`, `us cup`, `us oz`, `us lb`
- [ ] Register all new units via `registerUnit()` in `spec/units/conversion.go` with correct martinlindhe/unit conversion factors
- [ ] Add qualified unit names to `IsQuantityUnit()` and `IsMultiWordUnit()` in `spec/lexer/quantity_units.go`
- [ ] Write unit tests in `spec/units/conversion_test.go` verifying conversion accuracy for all new variants (e.g., `1 imp gal` ↔ `4.546 L`, `1 troy oz` ↔ `31.1035 g`, `1 short ton` ↔ `907.185 kg`)
- [ ] Write golden tests in `testdata/eval/success/features/measurement_conventions.cm` exercising inline qualified units: `gold = 10 troy oz in grams`, `milk = 2 imp pt in ml`, `cargo = 5 short ton in kg`
- [ ] Verify `task test` passes — qualified units work as explicit unit names independent of any convention system

**Success criteria:** `10 troy oz in grams` → `311.035 grams`, `1 imp gal in liters` → `4.546 liters`, `1 short ton in kg` → `907.185 kg`. All existing tests still pass.

**Files:**

| File | Change |
|------|--------|
| `spec/units/canonical.go` | Add ~20 new `UnitMapping` entries |
| `spec/units/conversion.go` | Register new units with `registerUnit()` |
| `spec/units/conversion_test.go` | Conversion accuracy tests |
| `spec/lexer/quantity_units.go` | Add qualified names to `IsQuantityUnit()`, `IsMultiWordUnit()` |
| `spec/lexer/quantity_units_test.go` | Test multi-word unit recognition |
| `testdata/eval/success/features/measurement_conventions.cm` | Golden tests |
| `testdata/spec/valid/features/measurement_conventions.cm` | Parser golden tests |

#### Phase 2: Frontmatter `measurement` directive

Add the `measurement:` frontmatter directive following the existing `scale`/`convert_to` pattern.

**Tasks:**

- [ ] Define `MeasurementConfig` struct in `spec/document/frontmatter.go` with fields: `Volume string`, `Mass string`, `Ton string`, `Strict *bool` (pointer for three-state: nil=unset, true, false)
- [ ] Add `Measurement *MeasurementConfig` field to `Frontmatter` struct
- [ ] Add `Measurement any` field to `frontmatterYAML` intermediate struct
- [ ] Implement `parseMeasurementConfig(raw any) (*MeasurementConfig, error)` following `parseConvertToConfig` pattern — validate axis values against known options, return descriptive errors
- [ ] Wire into `ParseFrontmatter()` pipeline
- [ ] Validate axis values: volume must be `us`|`imperial`, mass must be `standard`|`troy`, ton must be `short`|`long`|`metric`. Error messages must explain what "standard" means: `"standard" means avoirdupois — everyday weight (1 oz = 28.35g). Use "troy" for precious metals (1 troy oz = 31.10g).`
- [ ] Write unit tests in `spec/document/frontmatter_test.go`: valid configs, partial configs (only volume specified), invalid axis values, `strict: false`
- [ ] Write golden test files: `testdata/spec/valid/frontmatter/measurement_directive.cm`, `testdata/spec/invalid/frontmatter/measurement_invalid_axis.cm`
- [ ] Add `measurement` to `getFrontmatterFeatures()` in `spec/features/registry.go`

**Success criteria:** Frontmatter parses cleanly, invalid values produce helpful errors, partial configs fill defaults correctly.

**Files:**

| File | Change |
|------|--------|
| `spec/document/frontmatter.go` | `MeasurementConfig` struct, `parseMeasurementConfig()`, wire into pipeline |
| `spec/document/frontmatter_test.go` | Unit tests for all parse paths |
| `spec/features/registry.go` | Register `measurement` feature |
| `testdata/spec/valid/frontmatter/measurement_directive.cm` | Valid frontmatter golden test |
| `testdata/spec/invalid/frontmatter/measurement_invalid_axis.cm` | Invalid frontmatter golden test |

#### Phase 3: Convention-aware unit resolution

Make the conversion registry resolve bare ambiguous units using the active `MeasurementConfig`. This is the core behavioral change.

**Tasks:**

- [ ] Create `spec/units/ambiguous.go` defining which units are ambiguous per axis, and what each convention maps them to:
  ```
  volume:us      → gallon=USLiquidGallon, pint=USLiquidPint, ...
  volume:imperial → gallon=ImperialGallon, pint=ImperialPint, ...
  mass:standard  → oz=AvoirdupoisOunce, lb=AvoirdupoisPound, ...
  mass:troy      → oz=TroyOunce, lb=TroyPound, ...
  ton:short      → ton=ShortTon, ...
  ton:long       → ton=LongTon, ...
  ton:metric     → ton=MetricTonne, ...
  ```
- [ ] Add `ResolveUnit(unitName string, mc *MeasurementConfig) string` function that returns the qualified canonical name for a bare ambiguous unit given a convention. Non-ambiguous units pass through unchanged.
- [ ] Ensure `ResolveUnit` is called in the interpreter before `Convert()` — wire through `impl/interpreter/unit_conversion.go` and `unit_conversion_eval.go`
- [ ] Pass `MeasurementConfig` from frontmatter into interpreter environment via `impl/document/evaluator.go` (follow pattern of how `Scale`/`ConvertTo` are passed)
- [ ] Write tests in `spec/units/ambiguous_test.go` for all resolution paths
- [ ] Write integration tests: document with `measurement: { volume: imperial }` where `1 gallon in liters` → `4.546 liters` instead of `3.785 liters`
- [ ] Verify backwards compatibility: documents WITHOUT `measurement:` frontmatter produce identical output to today
- [ ] Update `testdata/eval/success/features/measurement_conventions.cm` with frontmatter-based tests

**Architectural constraint:** `impl/interpreter` cannot import `spec/document` (import cycle). The `MeasurementConfig` type lives in `spec/document`, so it must be passed through the environment or an interface. Follow the pattern used for exchange rates (see learnings: exchange-rate-frontmatter-validation).

**Success criteria:** `measurement: { volume: imperial }` + `1 gallon in liters` → `4.546 L`. Without frontmatter, `1 gallon in liters` → `3.785 L` (unchanged).

**Files:**

| File | Change |
|------|--------|
| `spec/units/ambiguous.go` | New — ambiguity registry and `ResolveUnit()` |
| `spec/units/ambiguous_test.go` | New — resolution tests |
| `impl/interpreter/unit_conversion.go` | Call `ResolveUnit()` before `Convert()` |
| `impl/interpreter/unit_conversion_eval.go` | Pass measurement config through |
| `impl/document/evaluator.go` | Wire `MeasurementConfig` from frontmatter → interpreter |
| `testdata/eval/success/features/measurement_conventions.cm` | Update with frontmatter tests |

#### Phase 4: Formatter annotation (strict mode)

In strict mode (default), the formatter annotates bare ambiguous units with their resolved convention prefix.

**Tasks:**

- [ ] Add measurement convention context to `format/display/Formatter` (either extend `DisplayConfig` or add a `MeasurementConfig` field)
- [ ] Create `IsAmbiguousUnit(unitName string) bool` helper in `spec/units/ambiguous.go`
- [ ] Create `ConventionPrefix(unitName string, mc *MeasurementConfig) string` that returns the prefix to prepend (e.g., `"us"`, `"troy"`, `"imp"`)
- [ ] In `FormatQuantity()`, if strict mode is active and unit is ambiguous and not already qualified, replace unit with prefixed version
- [ ] Skip annotation for units that already have an inline qualifier (`troy oz` stays `troy oz`)
- [ ] Pass `MeasurementConfig` + strict flag through from evaluator to formatter
- [ ] Write formatter tests: bare ambiguous unit → annotated, already-qualified → unchanged, strict=false → no annotation
- [ ] Update golden tests to verify formatted output

**Success criteria:** Default output annotates ambiguous units. `strict: false` suppresses annotation. Already-qualified units are never double-annotated.

**Files:**

| File | Change |
|------|--------|
| `format/display/formatter.go` | Annotation logic in `FormatQuantity()` |
| `format/display/config.go` | Add measurement context |
| `format/display/formatter_test.go` | Annotation tests |
| `spec/units/ambiguous.go` | Add `IsAmbiguousUnit()`, `ConventionPrefix()` |

#### Phase 5: Config integration

Add `[measurement]` section to `config.toml` as global defaults.

**Tasks:**

- [ ] Add `MeasurementConfig` struct to `cmd/calcmark/config/types.go` with `Volume`, `Mass`, `Ton`, `Strict` fields
- [ ] Add `Measurement MeasurementConfig` to `Config` struct
- [ ] Add commented `[measurement]` section to `cmd/calcmark/config/defaults.toml`
- [ ] Wire config → evaluator: merge config defaults with frontmatter overrides (frontmatter wins)
- [ ] `cm config` shows effective measurement settings
- [ ] `cm config --create` includes the `[measurement]` section with commented defaults
- [ ] Write config parsing tests

**Success criteria:** Config defaults apply when no frontmatter present. Frontmatter overrides config. `cm config` displays measurement settings.

**Files:**

| File | Change |
|------|--------|
| `cmd/calcmark/config/types.go` | Add `MeasurementConfig` struct |
| `cmd/calcmark/config/defaults.toml` | Add `[measurement]` section |
| `cmd/calcmark/config/config.go` | Wire measurement config |
| `cmd/calcmark/config/config_test.go` | Config parsing tests |

#### Phase 6: Documentation and diagnostics

Document the feature on the site and ensure all error messages are helpful.

**Tasks:**

- [ ] Add measurement conventions page to `site/content/docs/` explaining:
  - What axes exist and why they're independent
  - What "standard" means (avoirdupois — everyday weight) vs troy (precious metals)
  - Frontmatter syntax with examples
  - Inline prefix syntax with examples
  - Strict mode behavior with before/after output examples
  - Config integration
- [ ] Ensure all validation errors include concrete fix suggestions:
  - `measurement: { mass: avoidupois }` → `unknown mass convention "avoidupois" — valid options: standard (everyday weight: 1 oz = 28.35g), troy (precious metals: 1 troy oz = 31.10g)`
  - `measurement: { volume: metric }` → `unknown volume convention "metric" — valid options: us, imperial`
- [ ] Add `measurement_conventions` to `spec/features/registry.go` feature list with description
- [ ] Add inline code comments explaining "standard" = avoirdupois wherever the term appears

**Success criteria:** A user who's never heard of avoirdupois can understand the feature from docs and error messages alone.

**Files:**

| File | Change |
|------|--------|
| `site/content/docs/measurement-conventions.md` | New documentation page |
| `spec/features/registry.go` | Feature description |
| Error messages throughout | "standard" explanation in all diagnostics |

## Interaction with Existing Features

These are all orthogonal (see brainstorm: §Interaction with Existing Features):

- **`convert_to`**: Controls output target system. `measurement` controls input interpretation. Independent. `convert_to: imperial` + `measurement: { volume: imperial }` means source is interpreted as imperial AND converted to imperial (no-op for volume).
- **`locale`**: Number formatting (decimals, thousands). Fully independent of measurement conventions.
- **`scale`**: Scales numeric values. Unaffected by measurement conventions.
- **Explicit conversions**: `10 us gal in imp gal` works regardless of frontmatter — inline qualifiers always win.

## System-Wide Impact

### Interaction Graph

Frontmatter parse → `MeasurementConfig` stored on `Frontmatter` struct → passed to `evaluator.Evaluate()` → stored in interpreter environment → used by `ResolveUnit()` before every `Convert()` call → formatter reads config for strict annotation. No callbacks, no observers, no middleware.

### Error & Failure Propagation

Validation errors in frontmatter are caught at parse time and returned as diagnostics (same as invalid `scale` or `convert_to` values). Invalid inline qualifiers like `foobar oz` won't match `IsQuantityUnit()` and will parse as separate tokens, likely producing a parse error. No silent failures.

### State Lifecycle Risks

No persistent state. `MeasurementConfig` is immutable once parsed from frontmatter. No caching, no mutation, no cleanup needed.

### API Surface Parity

All output formats (display, JSON, HTML, markdown) must respect measurement conventions. The `--format json` output includes unit strings — these must be annotated in strict mode just like display format.

### Integration Test Scenarios

1. **Full pipeline with frontmatter**: Document with `measurement: { volume: imperial }` → `1 gallon in liters` → JSON output shows `4.546` (not `3.785`)
2. **Inline override beats frontmatter**: Document with `measurement: { mass: troy }` → `5 us oz in grams` → `141.748g` (US ounce, not troy)
3. **Strict mode annotation in all formats**: `2 oz` with no frontmatter → display shows `2 us oz`, JSON shows unit as `us oz`, HTML shows `2 us oz`
4. **Config + frontmatter precedence**: Config has `volume = "imperial"`, document frontmatter has `measurement: { volume: us }` → US volumes used
5. **Backwards compatibility**: All existing golden tests pass unchanged (no frontmatter = US Customary = current behavior)

## Acceptance Criteria

### Functional Requirements

- [ ] `10 troy oz in grams` → `311.035 grams`
- [ ] `1 imp gal in liters` → `4.546 liters`
- [ ] `1 short ton in kg` → `907.185 kg`
- [ ] `1 long ton in kg` → `1,016.047 kg`
- [ ] Frontmatter `measurement: { volume: imperial }` makes `1 gallon` resolve to imperial gallon
- [ ] Frontmatter `measurement: { mass: troy }` makes `1 oz` resolve to troy ounce
- [ ] Frontmatter `measurement: { ton: long }` makes `1 ton` resolve to long ton
- [ ] Inline qualifiers override frontmatter: `5 us oz` always means standard ounce
- [ ] Config `[measurement]` provides global defaults
- [ ] Frontmatter overrides config
- [ ] Strict mode (default) annotates bare ambiguous units in output
- [ ] `strict: false` suppresses annotation
- [ ] Already-qualified units are never double-annotated
- [ ] All existing tests pass (backwards compatibility)
- [ ] Error messages for invalid measurement values explain options with plain English descriptions

### Non-Functional Requirements

- [ ] No performance regression — `ResolveUnit()` must be O(1) lookup
- [ ] No import cycles between spec and impl layers

### Quality Gates

- [ ] `task test` passes
- [ ] `task quality` passes
- [ ] Golden tests cover all axes, all options, inline overrides, strict/non-strict output
- [ ] Documentation explains "standard" = avoirdupois in every context where the term appears

## Dependencies & Prerequisites

- **martinlindhe/unit library** already provides all needed conversion constants (ImperialGallon, TroyOunce, etc.) — no fork or new dependency needed (see brainstorm: §7)
- **Phase 1 is independent** — qualified units work as standalone feature before convention system exists
- **Phases 2-4 are sequential** — each builds on the previous
- **Phase 5 (config) is independent** of Phases 2-4 structurally but should merge last
- **Phase 6 (docs) can start after Phase 1** and finalize after Phase 4

## Risk Analysis & Mitigation

| Risk | Mitigation |
|------|-----------|
| Multi-word unit parsing: `troy oz` might conflict with existing lexer logic | Lexer already handles up to 3-word units; test early in Phase 1 |
| Backwards compatibility: existing documents silently change meaning | Default is US Customary (same as today); only documents with explicit `measurement:` frontmatter change behavior |
| Frontmatter map ordering non-determinism | Use ordered key slices (see learnings: go-maps-non-deterministic-ordering-frontmatter) |
| Import cycle: impl cannot import spec/document types | Pass `MeasurementConfig` through environment/interface, not direct import |
| NaN/Inf in frontmatter numeric fields | No numeric fields in measurement config — all string enums. Not applicable. |

## Future Considerations

The pattern is extensible for additional axes (see brainstorm: §3):

- `horsepower`: `mechanical` | `metric` | `electrical` | `boiler`
- `calorie`: `thermochemical` | `it`
- `tablespoon`: `standard` | `australian`
- `cup`: dedicated axis for metric (250 mL), US legal (240 mL), Japanese (200 mL) variants beyond the US/imperial split

Each is just a new axis registration — same pattern, no architectural changes.

## Sources & References

### Origin

- **Brainstorm document:** [docs/brainstorms/2026-03-14-measurement-conventions-brainstorm.md](docs/brainstorms/2026-03-14-measurement-conventions-brainstorm.md) — Key decisions: independent axes over single flag, `standard` over `avoirdupois`, strict formatter annotation by default, three-level precedence

### Internal References

- Unit registry: `spec/units/canonical.go:12` (`UnitMapping` struct)
- Conversion system: `spec/units/conversion.go:18` (`ConversionInfo`), `:178` (`registerUnit()`)
- Frontmatter parsing: `spec/document/frontmatter.go:50` (`Frontmatter` struct), `:342` (`parseConvertToConfig` — template)
- Formatter: `format/display/formatter.go:84` (`FormatQuantity`)
- Config: `cmd/calcmark/config/types.go` (`Config` struct)
- Lexer multi-word units: `spec/lexer/lexer.go:370-390`, `spec/lexer/quantity_units.go`
- Feature registry: `spec/features/registry.go`

### Institutional Learnings

- Cross-layer checklist: `docs/solutions/logic-errors/adding-new-type-fraction-cross-layer-checklist.md`
- Frontmatter allowlist gotcha: `docs/solutions/logic-errors/frontmatter-strict-allowlist-rejected-unknown-yaml-keys.md`
- Exchange rate validation (dual-layer pattern): `docs/solutions/logic-errors/exchange-rate-frontmatter-validation.md`
- Display intent preservation: `docs/solutions/ui-bugs/display-formatter-overrides-explicit-unit-conversion.md`
- Map ordering determinism: `docs/solutions/logic-errors/go-maps-non-deterministic-ordering-frontmatter.md`
- NaN/Inf security: `docs/solutions/security-issues/nan-inf-panic-yaml-frontmatter-scale.md`
