# Brainstorm: Measurements Walkthrough Guide

**Date:** 2026-03-15
**Status:** Draft

---

## What We're Building

A new guide under `site/content/guides/measurements/` that provides a comprehensive, scenario-driven walkthrough of CalcMark's measurement system. The guide follows a kitchen/recipe scenario that progressively builds from simple quantities to complex conversions, weaving in fractions, `as napkin`, and `convert_to` — and explicitly showing readers how each feature affects what they see in the TUI vs. what appears in JSON output.

**The core problem:** Measurement-related documentation is scattered across 5+ pages (units user guide, formatting guide, frontmatter reference, language reference, configuration docs). There is no single page where a reader can build a complete mental model by starting simple and gradually adding complexity.

**Note:** A recipe-scaling guide already exists at `guides/recipe-scaling/`, but it's focused on the `scale` directive and measurement conventions (US vs Imperial). This new guide has a different purpose: teaching the *measurement type system* progressively — what quantities are, how conversions work, what fractions and napkin math do to your data, and why JSON output differs between them.

---

## Why This Approach

### Scenario-driven, single page with jump links

- **Scenario:** Kitchen/recipe — universally relatable, naturally requires unit conversions, fractions, and rough estimates.
- **Structure:** Single page with a prominent TOC and section anchors. Readers who want the full story read top-to-bottom; readers who need one concept jump directly to it.
- **Output display:** Tabbed examples (CalcMark source / TUI display / JSON output) so readers pick their preferred view. Tabs persist across examples.
- **Napkin vs convert_to aha moment:** Introduced organically when the scenario first needs it, then crystallized in a dedicated comparison section near the end.

### Why not expand the existing recipe-scaling guide?

Different pedagogical goals. Recipe-scaling teaches `scale`, `@scale`, and `measurement:` conventions for a specific use case. This guide teaches the measurement *type system* — what a Quantity is, how `in` conversions work, what fractions preserve, why `as napkin` is a display hint while `convert_to` is a data transformation. They complement each other; linking between them is valuable.

---

## Key Decisions

1. **Placement:** New guide at `site/content/guides/understanding-measurements/_index.md`, added to the sidebar under Domain Recipes.
2. **Narrative arc:** Scenario-driven, kitchen/recipe theme. Opens with the NASA Mars Climate Orbiter story, pivots to Larky's cookie recipe as a humorous but pointed parallel. Progressive complexity — each section builds exactly one concept on the previous.
3. **Output tabs:** CSS-only three-tab view (CalcMark / TUI / JSON) using radio buttons + `:checked`. No JS for switching; tiny `localStorage` script only for persisting preference. New reusable component for the site.
4. **Relationship to recipe-scaling guide:** Complementary, cross-linked in both directions. This guide teaches measurement fundamentals; recipe-scaling teaches the `scale` directive.
5. **Napkin vs convert_to:** Woven in naturally at first use, then a dedicated recap section at the end ("What's Really in Your Data?").
6. **Single page with TOC:** Jump links for readers who need a specific concept; continuous story for readers going top-to-bottom.

---

## Proposed Section Outline

### 1. Introduction — "Units Matter" (no frontmatter needed)
- Open with the NASA Mars Climate Orbiter story (1999): a $327 million spacecraft lost because one team used pound-force seconds and the other used newton-seconds. Mixed units caused a real engineering catastrophe.
- Pivot to Larky: "Larky isn't sending anything to Mars — but his cookie recipe is *critical*. And when his friend in London asks for it in metric, getting the butter wrong isn't an option." Light, humorous tone that makes the point: unit management matters at every scale.
- First CalcMark snippet: just `flour = 2 cups` — show what a Quantity is in all three tabs.

### 2. Inline Conversions (`in`)
- Friend needs metric: `flour = 2 cups in grams`
- Show how `in` changes both TUI display and JSON `unit` field.
- Introduce `as` as an alias: `butter = 1 stick as grams`

### 3. Fractions
- `vanilla = 1/2 tsp` — CalcMark treats this as an exact rational number.
- Show TUI displays `1/2 tsp` (fraction preserved), JSON shows `numeric_value: 0.5`.
- Arithmetic with fractions: `double_vanilla = vanilla * 2` → `1 tsp`

### 4. Document-Wide Conversion (`convert_to`)
- "What if every line needs to be metric?"
- Introduce `convert_to: si` frontmatter — first time frontmatter appears in the guide.
- Show how all quantities change in output. Emphasize: this *actually converts the data*.

### 5. Napkin Math (`as napkin`)
- "Roughly how much flour do I need to buy for 10 batches?"
- `bulk_flour = flour * 10 as napkin` → TUI shows `~4.7 kg`, JSON shows `is_approximate: true` with original `numeric_value`.
- **Key insight (woven in):** "Notice the JSON still has the precise value — `as napkin` is a *display hint*, not a conversion."

### 6. Measurement Conventions (`measurement:`)
- Friend is in the UK, not continental Europe — cups mean different things.
- Introduce `measurement: { volume: imperial }`.
- Show how the same `1 cup` resolves to 284 mL (UK) vs 240 mL (US default).
- Inline qualifiers: `us_cup = 1 us cup` for mixing conventions.

### 7. "What's Really in Your Data?" (the aha recap)
- Side-by-side comparison table:
  - `convert_to: si` → JSON `numeric_value` and `unit` both change. Data is transformed.
  - `as napkin` → JSON `numeric_value` stays precise, `is_approximate: true` added. Display hint only.
  - `in kg` (inline) → Same as convert_to but per-line.
  - Fractions → `numeric_value` is the decimal equivalent; TUI may show the fraction.
- Short paragraph: "When you pipe CalcMark to another tool via `--format json`, convert_to gives you converted data. Napkin math gives you the original data with a hint that the display was rounded."

### 8. What to Read Next
- Links to: recipe-scaling guide, unit-conversion guide, language reference, configuration (JSON output fields).

---

## Resolved Questions

1. **Tab implementation:** CSS-only tabs using radio buttons + `:checked` selectors. Zero JS for functionality; a tiny `localStorage` script only for persisting the user's tab preference across examples.

2. **Guide URL:** `guides/understanding-measurements/` — signals this is a learning resource, not a reference page.

3. **Cross-linking:** Both directions. Add a callout at the top of the existing recipe-scaling guide pointing here for measurement fundamentals. This guide links to recipe-scaling for the `scale` directive.

## Resolved (cont.)

4. **Sidebar weight:** Before recipe-scaling — positioned as a prerequisite. Readers learn measurement fundamentals first, then apply them in recipe-scaling.

## Open Questions

None remaining.
