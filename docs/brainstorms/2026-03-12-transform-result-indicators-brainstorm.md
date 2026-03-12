# Transform Result Indicators

## What We're Building

Visual indicators in the TUI preview pane that show which transforms have been applied to each result. Currently only scaling shows a `*` suffix in orange/amber. We're adding a conversion indicator and changing the scale symbol to the real multiplication sign.

**Symbol system:**

| Transform | Symbol | Color |
|-----------|--------|-------|
| Scaled only | `×` (U+00D7 MULTIPLICATION SIGN) | Orange/amber (existing `ScaleIndicator` theme color) |
| Converted only | `•` (U+2022 BULLET) | New theme color (distinct from scale) |
| Both scaled + converted | `×•` (both side by side) | Each in its own color |

**Key design decision:** The combined indicator is literally both symbols concatenated, so it visually "adds up" from the individual indicators. No special third symbol needed.

## Why This Approach

- **Composable symbols** — `×•` is obviously `×` + `•`, no learning curve
- **Color-coded** — even without reading the symbol, the color tells you which transform
- **Zero tofu risk** — `×` and `•` render in every terminal font
- **Replaces `*` with `×`** — the current scale indicator uses a plain asterisk, which is ambiguous. The multiplication sign is semantically correct ("this value was multiplied")
- **`•` for conversion** — a bullet/dot meaning "this value was transformed". Small enough not to distract, bold enough to notice with color

## Constraints

- `~` is already used to indicate napkin approximation — cannot reuse
- The `×` must be the real Unicode multiplication sign (U+00D7), not the letter `x`
- Both transforms can apply to the same result (scale first, then convert_to) — the indicator must handle this
- Colors must work on both light and dark themes (like existing `ScaleIndicator`)

## Test Document

This frontmatter exercises all indicator combinations:

```yaml
---
hello: world
scale:
  factor: 2
  unit_categories:
  - Custom
  - Volume
convert_to:
  system: si
  unit_categories:
  - Volume
globals:
  currants: false
---
```

```cm
v = 2 buns
f = 2 cups
fruity = @globals.currants
```

**Expected indicators:**

| Line | Source | Result | Indicator | Why |
|------|--------|--------|-----------|-----|
| `v = 2 buns` | 2 buns | 4 buns | `×` (scaled only) | "Custom" category matches, but no convert_to for Custom |
| `f = 2 cups` | 2 cups | ~946.35 mL | `×•` (both) | Volume is in both scale and convert_to categories |
| `fruity = @globals.currants` | false | false | (none) | Booleans are not scaled or converted |

## Key Decisions

1. **Symbol: `×` for scale** — real multiplication sign, not asterisk or letter x
2. **Symbol: `•` for convert** — bullet, visible with color, not too heavy
3. **Composition: concatenate** — `×•` for both, reads left-to-right matching transform order (scale first, convert second)
4. **Colors: two distinct theme colors** — existing orange/amber for scale, new color for convert (needs to be added to `palette.go`)

## Open Questions

None — all design decisions resolved through dialogue.
