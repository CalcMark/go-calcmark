---
title: "feat: Add Force/Impulse/Pressure/Acceleration/Frequency units, refactor Speed to martinlindhe, and build Speed-Rate bridge"
type: feat
status: completed
date: 2026-03-16
origin: docs/brainstorms/2026-03-16-units-rates-and-force-brainstorm.md
---

# Add Physical Unit Categories, Refactor Speed, and Build Speed-Rate Bridge

## Overview

Expand CalcMark's unit system with five new physical categories (Force, Impulse, Pressure, Acceleration, Frequency) backed by the martinlindhe library, refactor speed conversions from hardcoded factors to martinlindhe, and build a bidirectional bridge between Speed quantities and Rates. Culminates in a worked NASA Mars Climate Orbiter example demonstrating impulse conversion.

## Problem Statement / Motivation

CalcMark's understanding-measurements guide tells the story of the $327M Mars Climate Orbiter loss due to a pound-force-seconds vs newton-seconds mismatch — but CalcMark cannot actually perform those conversions. More broadly, the unit system lacks Force, Pressure, Acceleration, Frequency, and Impulse — categories already supported by the martinlindhe library that CalcMark depends on. Meanwhile, speed conversions use hardcoded magic numbers instead of the library, and there is no way to bridge between Speed quantities (`60 mph`) and Rates (`60 mi/h`) despite them representing the same physical concept.

(See brainstorm: `docs/brainstorms/2026-03-16-units-rates-and-force-brainstorm.md`)

## Proposed Solution

Four-phase implementation, each shippable independently:

1. **Phase 1 — Speed refactor**: Replace hardcoded speed conversion factors with martinlindhe's Speed type. Pure backend change, identical behavior.
2. **Phase 2 — New unit categories**: Add Force, Impulse, Pressure, Acceleration, Frequency following the established 8-layer pattern.
3. **Phase 3 — Speed-Rate bridge**: Bidirectional coercion between Speed quantities and Rates, enabling `60 mph over 2 hours = 120 miles`.
4. **Phase 4 — NASA worked example**: Add impulse conversion example to understanding-measurements guide.

## Technical Approach

### Phase 1: Speed Refactor (martinlindhe backend)

**Goal**: Replace hardcoded conversion factors in `spec/units/conversion.go:430-456` with martinlindhe Speed type calls, matching the pattern used by Length, Mass, Volume, etc.

**Files**:
- `spec/units/conversion.go` — rewrite `addSpeedConversions()` to use `martinlindhe.Speed` type with `MetersPerSecond`, `KilometersPerHour`, `MilesPerHour`, `Knot` constants
- `spec/units/conversion_test.go` — verify conversion accuracy matches existing hardcoded values within acceptable precision

**Verification**: All existing speed golden tests must pass unchanged. Run `task test` — zero regressions.

**Prerequisite check**: Verify `m/s` slash-form speed parsing works today. Run existing golden tests in `testdata/eval/success/features/speed_units.cm`. The test file explicitly says "NO SLASH UNITS - Parser doesn't support / yet" — this means slash-form speed units (`m/s`, `km/h`) may not parse as units. This is pre-existing behavior and NOT in scope for this phase, but must be documented for Phase 3 bridge work.

---

### Phase 2: New Unit Categories

**Goal**: Add Force, Impulse, Pressure, Acceleration, Frequency as first-class unit categories.

Follow the 8-layer cross-cutting checklist (from `docs/solutions/logic-errors/adding-new-type-fraction-cross-layer-checklist.md`) for each category:

#### Layer 1: `spec/units/canonical.go` — Unit definitions

Add `UnitMapping` entries for each unit. Key units per category:

**Force** (Quantity: `"Force"`):
| Canonical | Symbol | Aliases | System |
|-----------|--------|---------|--------|
| newton | N | newtons | SI |
| kilonewton | kN | kilonewtons | SI |
| dyne | dyn | dynes | CGS |
| kilogram-force | kgf | kilogram-force, kilopond, kiloponds | SI |
| pound-force | lbf | pound-force, pound-forces | Imperial |
| poundal | pdl | poundals | Imperial |

**Impulse** (Quantity: `"Impulse"`) — no martinlindhe type, manual conversion:
| Canonical | Aliases | System |
|-----------|---------|--------|
| newton-second | newton-seconds | SI |
| pound-force-second | pound-force-seconds | Imperial |

Conversion factor: `1 lbf*s = 4.448222 N*s` (same as pound-force to newton, since time unit cancels).

**Pressure** (Quantity: `"Pressure"`):
| Canonical | Aliases | System |
|-----------|---------|--------|
| pascal | pascals, pa | SI |
| kilopascal | kilopascals, kpa | SI |
| megapascal | megapascals, mpa | SI |
| bar | bars | SI |
| millibar | millibars, mbar | SI |
| atmosphere | atmospheres, atm | SI |
| torr | torrs | SI |
| psi | pounds per square inch | Imperial |
| inch of mercury | inches of mercury, inhg | Imperial |

**Acceleration** (Quantity: `"Acceleration"`):
| Canonical | Aliases | System |
|-----------|---------|--------|
| m/s^2 | meters per second squared | SI |
| cm/s^2 | centimeters per second squared | CGS |
| ft/s^2 | feet per second squared | Imperial |
| standard-gravity | standard gravities | SI |

Note: `gal` is NOT registered as an acceleration alias — it conflicts with `gallon` (see brainstorm resolved question). `g` is NOT registered — it conflicts with `grams`.

**Frequency** (Quantity: `"Frequency"`):
| Canonical | Aliases | System |
|-----------|---------|--------|
| hertz | hz | SI |
| kilohertz | khz | SI |
| megahertz | mhz | SI |
| gigahertz | ghz | SI |
| terahertz | thz | SI |

#### Layer 2: `spec/units/conversion.go` — Conversion functions

Add five new `addXxxConversions(r)` calls in `buildConversionRegistry()`:

- `addForceConversions(r)` — base unit: newton. Use `martinlindhe.Force` type with `Newton`, `Dyne`, `KilogramForce`, `PoundForce`, `Poundal` constants.
- `addImpulseConversions(r)` — base unit: newton-second. Manual conversion: `1 lbf*s = 4.448222 N*s`.
- `addPressureConversions(r)` — base unit: pascal. Use `martinlindhe.Pressure` type with `Pascal`, `Kilopascal`, `Bar`, `Atmosphere`, `PoundsPerSquareInch`, `Torr`, `InchOfMercury`, `Millibar` constants.
- `addAccelerationConversions(r)` — base unit: m/s^2. Use `martinlindhe.Acceleration` type with `MeterPerSecondSquared`, `CentimeterPerSecondSquared`, `FootPerSecondSquared`, `StandardGravity` constants.
- `addFrequencyConversions(r)` — base unit: hertz. Use `martinlindhe.Frequency` type with `Hertz`, `Kilohertz`, `Megahertz`, `Gigahertz`, `Terahertz` constants.

Add `defaultTargetUnits` entries:
```
"Force:si": "newton"         "Force:imperial": "pound-force"
"Impulse:si": "newton-second" "Impulse:imperial": "pound-force-second"
"Pressure:si": "pascal"      "Pressure:imperial": "psi"
"Acceleration:si": "m/s^2"   "Acceleration:imperial": "ft/s^2"
"Frequency:si": "hertz"
```

Note: No `Frequency:imperial` entry. When `convert_to: imperial` encounters a frequency value, `GetDefaultTargetUnit` returns empty string, and `applyConvertToQuantity` silently passes through — frequency values are unchanged. This is correct behavior and should be tested explicitly.

#### Layer 3: `spec/lexer/quantity_units.go` — Lexer recognition

Add new unit maps for each category and extend the `IsQuantityUnit()` check:
- `forceUnits` map: newton, newtons, kilonewton, kilonewtons, dyne, dynes, kilogram-force, pound-force, pound-forces, poundal, poundals, lbf, kgf, pdl
- `impulseUnits` map: newton-second, newton-seconds, pound-force-second, pound-force-seconds
- `pressureUnits` map: pascal, pascals, pa, kilopascal, kilopascals, kpa, megapascal, megapascals, mpa, bar, bars, millibar, millibars, mbar, atmosphere, atmospheres, atm, torr, torrs, psi, inhg
- `accelerationUnits` map: m/s^2, cm/s^2, ft/s^2, standard-gravity (NOTE: caret forms must be added to hardcoded unit list for greedy matching — see below)
- `frequencyUnits` map: hertz, hz, kilohertz, khz, megahertz, mhz, gigahertz, ghz, terahertz, thz

**Critical: Acceleration caret notation** — `m/s^2`, `cm/s^2`, `ft/s^2` must be added as hardcoded unit strings in the lexer's greedy unit matching. The lexer must match `m/s^2` as a single unit token rather than tokenizing as `m` `/` `s` `^` `2`. This follows the same approach as `m/s` and `km/h` which are already hardcoded as known slash-form units in `quantity_units.go`.

**Prerequisite**: The existing slash-form speed units (`m/s`, `km/h`) in `quantity_units.go` must actually work in the lexer's greedy matching today. The speed golden tests explicitly avoid slash forms ("NO SLASH UNITS - Parser doesn't support / yet"). Before implementing acceleration caret units, verify that the greedy matching mechanism for `m/s` works. If it does not, the acceleration caret units will need the same fix, and the scope of this layer expands to include fixing slash-form unit tokenization generally.

#### Layer 4: `spec/units/categories.go` — Auto-derived

No manual changes needed. New `Quantity` strings in `StandardUnits` automatically create categories via `Categories()`.

#### Layer 5: Frontmatter — `spec/document/frontmatter.go`

`validUnitCategories` is auto-derived from `units.Categories()` at init time — new categories automatically become valid for `unit_categories` lists in frontmatter. No manual changes needed.

The `measurement` directive (`MeasurementConfig`) currently handles Volume, Mass, and Ton ambiguity. None of the new categories have ambiguous regional names (pound-force is explicitly qualified, PSI is unambiguous). No new axes needed in `MeasurementConfig`.

#### Layer 6: `spec/features/registry.go` — Feature documentation

Add feature entries for the five new categories documenting supported units and conversions.

#### Layer 7: Golden tests

Create TDD test files BEFORE implementing conversions:

- `testdata/eval/success/features/force_units.cm`
- `testdata/eval/success/features/impulse_units.cm`
- `testdata/eval/success/features/pressure_units.cm`
- `testdata/eval/success/features/acceleration_units.cm`
- `testdata/eval/success/features/frequency_units.cm`
- `testdata/eval/errors/features/cross_category_force_frequency.cm` — verify `10 newtons in hertz` errors correctly

Each test file should cover: basic evaluation, `in` conversion within category, cross-system conversion (SI ↔ Imperial where applicable), `convert_to` frontmatter behavior, and error cases.

Also duplicate to `testdata/spec/valid/features/` for spec-level validation.

#### Layer 8: Multi-word unit handling

Check `spec/lexer/multiword.go` for multi-word unit support. Units like "pound-force", "kilogram-force", "newton-second", "pound-force-second", "inch of mercury", "pounds per square inch", "meters per second squared" need multi-word lookup. Hyphenated forms (`pound-force`) may be handled as single tokens. Space-separated forms ("inch of mercury") need the multi-word unit scanner.

---

### Phase 3: Speed-Rate Bridge

**Goal**: Enable bidirectional coercion between Speed quantities and Rates.

#### 3.1: Speed decomposition mapping

Add a new field or function in the spec layer that maps Speed units to their Rate components:

```
mph  → {numerator: "mi",  denominator: "h"}
kph  → {numerator: "km",  denominator: "h"}
mps  → {numerator: "m",   denominator: "s"}
knots → {numerator: "nmi", denominator: "h"}
```

Location: new function `DecomposeSpeedUnit(unit string) (numeratorUnit, timeUnit string, ok bool)` in `spec/units/` — keeps it in the spec layer. Could be a map on `UnitMapping` or a standalone function.

#### 3.2: Speed → Rate coercion (in `evalUnitConversion`)

In `impl/interpreter/unit_conversion_eval.go`, add bridge detection before the existing type assertions:

When source is a `*types.Quantity` with Speed category AND target has a `TargetTimeUnit` (meaning the target is rate-like):
1. Decompose the Speed unit into numerator/denominator
2. Convert the Speed value to the decomposed base (e.g., `60 mph` → convert 60 from mph to mi/h equivalent value)
3. Construct a `*types.Rate{Amount: Quantity{value, numeratorUnit}, PerUnit: denominator}`
4. Delegate to `evalRateUnitConversion()` for the actual rate-to-rate conversion

#### 3.3: Rate → Speed coercion (in `evalUnitConversion`)

When source is a `*types.Rate` AND target unit is a known Speed unit (no `TargetTimeUnit`):
1. Convert the rate to the Speed unit's decomposed form (e.g., rate `120 mi/h` → normalize to match `mph`)
2. Construct a `*types.Quantity` with the Speed unit
3. Return as a standard quantity

#### 3.4: Speed accumulation via `over` and `accumulate()`

In `impl/interpreter/rate_functions.go` (or the `evalAccumulate` dispatcher):
- When first argument is a `*types.Quantity` with Speed category, coerce to Rate using the decomposition mapping, then proceed with standard accumulation
- This handles BOTH `60 mph over 2 hours` (desugared to `accumulate(60 mph, 2 hours)`) AND direct `accumulate(60 mph, 2 hours)` calls

#### 3.5: Speed × Duration arithmetic

In `impl/interpreter/operators.go`, before the `unsupportedOperationError` for `Quantity * Duration`:
- Check if the Quantity is Speed category
- If yes, coerce to Rate and accumulate with the Duration operand
- Produce a distance Quantity result

**Important constraint**: Only fire for Speed-category quantities. `10 kg * 5 hours` must still error.

#### 3.6: Display / formatting

Follow input style (see brainstorm resolved question 4):
- Bridge results that produce a Rate display normally as `value unit/timeunit`
- Bridge results that produce a Quantity use the decomposed numerator unit: `mph` → `mi`, `kph` → `km`
- The decomposition mapping should include preferred display forms

#### 3.7: IsExplicit flag

- Set `IsExplicit = true` on bridge conversion results (both Speed→Rate and Rate→Speed) to prevent `convert_to` from overriding explicit user conversions
- Accumulation results (`over`, `* duration`) do NOT set `IsExplicit` — they are derived values that should respect `convert_to`

#### 3.8: Bridge golden tests

- `testdata/eval/success/features/speed_rate_bridge.cm` — covers all bridge flows:
  - `60 mph in m/min` (Speed → Rate)
  - `120 mi/h in mph` (Rate → Speed)
  - `60 mph over 2 hours` (Speed accumulation via over)
  - `accumulate(60 mph, 2 hours)` (Speed accumulation via function)
  - `60 mph * 2 hours` (Speed × Duration arithmetic)
  - `60 kph in m/s` (Speed → Rate with unit + time conversion)
- `testdata/eval/errors/features/speed_rate_bridge_errors.cm` — verifies non-speed quantities don't accidentally bridge

---

### Phase 4: NASA Worked Example

**Goal**: Add a worked example to the understanding-measurements guide.

**File**: `site/content/guides/understanding-measurements/index.md`

Add a section at the end demonstrating the Mars Climate Orbiter unit mismatch using impulse conversion:

```calcmark
## The Mars Climate Orbiter: A Worked Example

thruster_impulse = 110 pound-force-seconds
expected_impulse = thruster_impulse in newton-seconds
```

Show that the conversion reveals the 4.45x difference between the two unit systems, connecting directly to the story already on the page.

## System-Wide Impact

- **Interaction graph**: New units flow through: lexer → parser → semantic → interpreter → formatter → JSON output. No callbacks or middleware. The `convert_to` transform fires post-evaluation for auto-conversion.
- **Error propagation**: Category mismatch errors (`"cannot convert newton to hertz"`) use the existing `Convert()` error path. No new error types needed.
- **State lifecycle risks**: None — unit conversions are pure functions with no persistent state.
- **API surface parity**: JSON output automatically includes new unit categories. No separate API changes needed.
- **Integration test scenarios**: (1) New unit + `convert_to: imperial` + `unit_categories: [Force]` filter, (2) Speed bridge + `over` accumulation end-to-end, (3) Cross-category error with new units.

## Acceptance Criteria

### Phase 1: Speed Refactor
- [x] `addSpeedConversions()` uses martinlindhe Speed type (no hardcoded factors)
- [x] All existing speed golden tests pass unchanged
- [x] `task test` and `task quality` pass

### Phase 2: New Unit Categories
- [x] Force units convert correctly: `10 newtons in pound-force`
- [x] Impulse units convert correctly: `500 newton-seconds in pound-force-seconds`
- [x] Pressure units convert correctly: `1 atmosphere in psi`
- [x] Acceleration units convert correctly: `1 standard-gravity in feet per second squared` (note: slash-caret notation m/s^2 not yet parseable as input)
- [x] Frequency units convert correctly: `2.4 gigahertz in megahertz`
- [x] `convert_to: imperial` converts Force, Impulse, Pressure, Acceleration to imperial defaults
- [x] `convert_to: si` converts imperial units to SI defaults
- [x] Frequency is unaffected by `convert_to: imperial` (no imperial variant)
- [x] Cross-category errors work: `10 newtons in hertz` → clear error message
- [x] `unit_categories` frontmatter filter works for all new categories
- [x] Golden tests exist and pass for all 5 categories
- [x] `gal` is NOT an acceleration alias (gallon conflict)
- [x] `g` is NOT a standard gravity alias (gram conflict)
- [x] Multi-word units parse correctly: "pound-force", "newton-second", "inch of mercury"

### Phase 3: Speed-Rate Bridge
- [x] `60 kph in m/s` → produces Rate result with correct conversion
- [x] `60 km/h in mph` → produces Speed Quantity result
- [x] `60 mph over 2 hours` → produces `120 mi` distance
- [x] `accumulate(60 mph, 2 hours)` → same result as `over` form (via speed coercion in evalAccumulate)
- [x] `60 kph * 2 hours` → produces distance (not error)
- [x] `10 kg * 5 hours` still errors (bridge only fires for Speed)
- [x] `IsExplicit` set on bridge conversion results
- [x] Display follows input style

### Phase 4: NASA Example
- [x] Worked example added to understanding-measurements guide
- [x] Example uses impulse conversion (pound-force-seconds ↔ newton-seconds)
- [ ] Example renders correctly on the site (requires Hugo build — not verified locally)

## Dependencies & Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Acceleration caret notation lexer challenge | `m/s^2` may not tokenize as a single unit | Hardcode in `quantity_units.go` greedy match list, same as existing `m/s` and `km/h` |
| `m/s` slash parsing may be broken | Speed golden tests avoid slash forms | Verify before Phase 3; bridge may need to work with alias forms only |
| `gal`/gallon alias collision | Direct conflict in canonical units | Do not register `gal` for acceleration |
| Bridge scope creep | `Quantity * Duration` intercept could affect non-speed categories | Guard with explicit Speed category check |
| martinlindhe precision differences | Library may use slightly different conversion factors than hardcoded values | Add conversion_test.go cases comparing martinlindhe output against existing hardcoded factors; accept if delta < 1e-6 relative error |

## Sources & References

### Origin

- **Brainstorm document:** [docs/brainstorms/2026-03-16-units-rates-and-force-brainstorm.md](docs/brainstorms/2026-03-16-units-rates-and-force-brainstorm.md) — Key decisions: Speed stays Quantity with Rate bridge, Impulse as atomic units, caret notation for acceleration, hyphenated long-form for impulse aliases, all martinlindhe categories included.

### Internal References

- 8-layer type checklist: `docs/solutions/logic-errors/adding-new-type-fraction-cross-layer-checklist.md`
- NL/functional syntax parity: `docs/solutions/integration-issues/nl-functional-syntax-parity-and-doc-staleness.md`
- IsExplicit pattern: `docs/solutions/ui-bugs/display-formatter-overrides-explicit-unit-conversion.md`
- Unit definitions: `spec/units/canonical.go`
- Conversion registry: `spec/units/conversion.go`
- Rate type: `spec/types/rate.go`
- Lexer unit recognition: `spec/lexer/quantity_units.go`
- Unit conversion evaluator: `impl/interpreter/unit_conversion_eval.go`
- Rate functions: `impl/interpreter/rate_functions.go`
- Operator dispatch: `impl/interpreter/operators.go`
- Frontmatter parsing: `spec/document/frontmatter.go`
- Transform (convert_to): `spec/transform/transform.go`
- Speed golden tests: `testdata/eval/success/features/speed_units.cm`
- Rate golden tests: `testdata/eval/success/features/rates.cm`, `rate_conversion.cm`, `rate_functions.cm`

### External References

- martinlindhe/unit library: `github.com/martinlindhe/unit` — Force, Pressure, Acceleration, Frequency, Speed types
- NASA Mars Climate Orbiter investigation: referenced in `site/content/guides/understanding-measurements/`
