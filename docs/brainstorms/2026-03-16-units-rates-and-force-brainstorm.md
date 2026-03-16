# Brainstorm: Units, Rates, and Force

**Date:** 2026-03-16
**Status:** Draft

## What We're Building

Expand CalcMark's unit system with new physical categories, refactor speed conversions to use martinlindhe, and build bidirectional bridges between speed quantities and rates.

### Scope

1. **New unit categories from martinlindhe:**
   - **Force** — newton, kilonewton, dyne, kilogram-force, pound-force, poundal
   - **Impulse** — newton-second, pound-force-second (atomic units, not derived from force×time)
   - **Pressure** — pascal, kilopascal, megapascal, bar, millibar, atmosphere, torr, PSI, inch-of-mercury
   - **Acceleration** — m/s^2, cm/s^2, ft/s^2, gal (note: `g` conflicts with grams, not used)
   - **Frequency** — hertz, kilohertz, megahertz, gigahertz, terahertz

2. **Speed refactor:**
   - Replace hardcoded speed conversion factors with martinlindhe's Speed type
   - Keep speed as an opaque Quantity category (not a Rate)
   - Preserve existing aliases: `kph`, `mph`, `mps`, `knots`

3. **Speed ↔ Rate bridges:**
   - `60 kph in m/min` — coerce Speed quantity to Rate for conversion
   - `120 mi/h in mph` — coerce Rate to Speed quantity
   - `60 mph * 2 hours` — coerce Speed to Rate, accumulate, produce distance
   - Bridges are bidirectional and seamless to the user

4. **Frontmatter `measurement` directive:**
   - Extend to handle new categories: Force (newton ↔ pound-force), Pressure (pascal ↔ PSI), etc.
   - Regional defaults: `measurement: si` → newtons, pascals; `measurement: imperial` → pound-force, PSI

5. **NASA Mars Orbiter worked example:**
   - Add to end of understanding-measurements guide
   - Demonstrate impulse conversion: pound-force-seconds ↔ newton-seconds
   - Direct connection to the story already on the page

## Why This Approach

### Speed stays a Quantity (not a Rate)

Making speed a Rate would impose rate-specific constraints on what is currently a flexible quantity. Engineering and scientific calculations may treat `mph` as a plain value in formulas. Keeping it as a Quantity preserves predictable behavior and avoids regressions.

The bidirectional bridge gives users rate-like behavior (accumulation, time-unit conversion) when they need it, without forcing it.

### Impulse as atomic units (not force × time)

Deriving impulse from `force * time` expressions would require CalcMark to understand dimensional analysis — a much larger language change. Adding `newton-second` and `pound-force-second` as their own Impulse category with hardcoded conversions is simple, explicit, and sufficient for the NASA example and real-world use.

### martinlindhe for Speed

CalcMark currently hardcodes speed conversion factors while using martinlindhe for everything else. Switching to the library for speed is a consistency win and reduces maintenance burden.

### All available martinlindhe categories

Pre-1.0 is the right time to be comprehensive. Pressure, Acceleration, and Frequency are commonly needed in engineering contexts and already implemented in martinlindhe. Adding them now avoids repeated "add unit category X" work later.

## Key Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Speed type | Stays as Quantity | No regressions; rate behavior via bridge |
| Speed backend | martinlindhe library | Consistency with other categories |
| Impulse approach | Atomic unit category | Simpler than dimensional analysis |
| New categories | Force, Impulse, Pressure, Acceleration, Frequency | Comprehensive; martinlindhe supports all |
| Speed ↔ Rate | Bidirectional bridge/coercion | Users get rate math when needed, quantity flexibility always |
| `measurement` directive | Extended for new categories | Regional variants for force, pressure, etc. |
| NASA example | Impulse conversion focus | Direct tie to the story on the page |

## Resolved Questions

1. **Acceleration notation**: Use caret notation (`m/s^2`, `ft/s^2`) alongside named aliases like `gal`. Familiar from programming, commonly accepted. Note: `g` conflicts with grams — do NOT use `g` as an alias for standard gravity.

2. **Bridge scope**: Speed-only for now. YAGNI — generalize to other categories (frequency ↔ cycles/second) only if needed later.

3. **Impulse aliases**: Hyphenated long-form only — `newton-second(s)`, `pound-force-second(s)`. Abbreviations like `Ns` conflict with nanoseconds because CalcMark units are case-insensitive. The broader question of case-sensitive unit abbreviations is a separate future brainstorm.

4. **Compound unit display**: Follow input style. If user wrote `mph`, output `mi`. If they wrote `miles per hour`, output `miles`. Match what the user typed.

5. **Speed × time semantics**: `60 mph * 2 hours` → `120 miles`. This is rate accumulation (same as `60 mi/h over 2 hours`). The bridge coerces the Speed quantity into a rate, accumulates, and produces a distance.

## Open Questions

1. **Case-sensitive units (future)**: SI convention distinguishes `N` (newton) from `n` (nano-). CalcMark is currently case-insensitive. This limits short-form aliases for new categories. Worth a separate brainstorm if abbreviation demand grows.

2. **`measurement` directive mappings**: Research the standard SI ↔ imperial pairings for each new category during implementation. martinlindhe's library defines the correct conversion factors. Note: pound-force is well-defined (1 lbf = 1 avoirdupois pound × standard gravity) — no avoirdupois/imperial ambiguity. Frequency likely has no imperial variant and can be excluded from `measurement` toggling.
