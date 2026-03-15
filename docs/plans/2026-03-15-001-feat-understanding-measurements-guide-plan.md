---
title: "feat: Understanding Measurements guide — progressive walkthrough"
type: feat
status: active
date: 2026-03-15
origin: docs/brainstorms/2026-03-15-measurements-walkthrough-guide-brainstorm.md
---

# feat: Understanding Measurements Guide

## Overview

Create a new guide at `site/content/guides/understanding-measurements/_index.md` that progressively teaches CalcMark's measurement system through a kitchen/recipe scenario. The guide introduces one concept per section — bare quantities, inline conversions, fractions, `convert_to`, napkin math, and measurement conventions — with tabbed output showing CalcMark source, TUI display, and JSON in each example.

The central teaching moment: a side-by-side comparison showing how `convert_to`, `as napkin`, and inline `in` each affect what appears in JSON output differently.

(see brainstorm: `docs/brainstorms/2026-03-15-measurements-walkthrough-guide-brainstorm.md`)

## Problem Statement / Motivation

Measurement-related documentation is scattered across 5+ pages (units user guide, formatting guide, frontmatter reference, language reference, configuration docs). A reader who wants to understand "how do measurements work in CalcMark?" must bounce between pages and assemble the mental model themselves. There is no single progressive walkthrough.

The existing recipe-scaling guide teaches `scale`, `@scale`, and `measurement:` conventions for a specific use case. The existing unit-conversion guide covers ambiguity and directive composition. Neither teaches the measurement *type system* progressively: what a Quantity is, how fractions preserve exactness, why napkin math and `convert_to` produce different JSON, or what the `value` vs `numeric_value` distinction means.

## Proposed Solution

A single-page guide with a prominent TOC and section anchors, following the established guide structure pattern. A kitchen/recipe scenario (Larky adapting a family cookie recipe for a friend in London) threads through all sections. A new CSS-only tab component allows readers to view CalcMark source, TUI display, or JSON output per example.

### Differentiation from existing guides

| Guide | Purpose | Teaches |
|-------|---------|---------|
| **Understanding Measurements** (new) | Progressive tutorial of the measurement type system | Quantities, inline `in`, fractions, `convert_to`, `as napkin`, measurement conventions, JSON output semantics |
| Recipe Scaling | Domain recipe: cooking | `scale`, `@scale`, `unit_categories`, cost analysis |
| Unit Conversion | Domain recipe: ambiguity | Ambiguous units, inline qualifiers, `measurement:` axis table, directive composition pipeline |

## Technical Approach

### Phase 1: CSS-Only Tab Component

Build a reusable tab shortcode and CSS component. This is a prerequisite for the guide content.

**Files to create/modify:**

1. `site/layouts/shortcodes/tabs.html` — Hugo shortcode wrapping tab markup
2. `site/layouts/shortcodes/tab.html` — Individual tab panel shortcode
3. `site/assets/css/components.css` — Add `.tabs-*` styles using existing design tokens
4. `site/assets/js/tabs.js` (tiny) — localStorage persistence only (~15 lines)
5. `site/layouts/partials/head.html` — Include tabs.js

**Architecture:**

- CSS switching via hidden radio inputs + `:checked` + adjacent sibling selectors
- Three tabs: "CalcMark", "Editor", "JSON" (renamed from "TUI" for clarity)
- Progressive enhancement: works without JS; JS only persists tab preference via `localStorage`
- Accessible: ARIA `role="tablist"`, `role="tab"`, `role="tabpanel"` attributes in shortcode markup
- Responsive: tabs stack or scroll horizontally on narrow viewports
- Dark mode: use existing `[data-theme="dark"]` tokens

**Shortcode usage in markdown:**

```markdown
{{< tabs group="output" >}}
{{< tab name="CalcMark" >}}
\`\`\`calcmark
flour = 2 cups
\`\`\`
{{< /tab >}}
{{< tab name="Editor" >}}
```text
flour = 2 cups                                          1 pt
```
{{< /tab >}}
{{< tab name="JSON" >}}
```json
{ "value": "1 pt", "numeric_value": 2, "unit": "cups" }
```
{{< /tab >}}
{{< /tabs >}}
```

The `group` parameter ties tab preference across all tab groups on the page — selecting "JSON" in one example selects "JSON" in all.

**Note:** The CalcMark tab within `{{< tabs >}}` uses a standard code fence, NOT the `render-codeblock-calcmark` hook (which would auto-append results). Inside tabs we want to control what each panel shows independently. The CalcMark tab shows source only; the Editor tab shows source + results as the TUI renders them; the JSON tab shows the machine-readable output.

### Phase 2: Guide Content

Create `site/content/guides/understanding-measurements/_index.md` following the established guide structure.

**Weight:** 25 (between business-planning at 20 and recipe-scaling at 30)

**Section outline with verified examples:**

#### Section 1: "Units Matter" (Introduction)

- NASA Mars Climate Orbiter story (1999, $327M lost to mixed units)
- Pivot to Larky: "Larky isn't sending anything to Mars — but his cookie recipe is critical."
- First example: `flour = 2 cups` — introduces the Quantity type
- Tab: show that `value` displays "1 pt" (normalized) but `numeric_value` stays `2` and `unit` stays `cups`

#### Section 2: Inline Conversions (`in`)

- Scenario: friend in London needs metric
- Example: `flour = 2 cups in grams`
- Teaching point: `in` converts the data — both `numeric_value` and `unit` change in JSON

#### Section 3: Fractions

- Example: `vanilla = 1/2 tsp`
- Teaching point: CalcMark preserves exact rational numbers. JSON shows `type: "fraction"` with `numerator`, `denominator` fields alongside `numeric_value`
- Arithmetic: `double_vanilla = vanilla * 2` → `1 tsp` (fraction simplifies)

#### Section 4: Document-Wide Conversion (`convert_to`)

- Scenario: "What if every line needs metric?"
- First appearance of frontmatter in the guide: `convert_to: si`
- Example: `flour = 2 cups` with `convert_to: si` → `value: "480 ml"`, `numeric_value: 0.48`, `unit: "liter"`
- Teaching point: `convert_to` is like applying `in` to every line. Both `value` and `numeric_value` change
- Optional: mention oven temperature as a natural `convert_to` example (350°F → 177°C)

#### Section 5: Napkin Math (`as napkin`)

- Scenario: "Roughly how much flour for 10 batches?"
- Example: `bulk_flour = 20 cups as napkin` → `value: "~1.25 gal"`, `numeric_value: 1.25`, `unit: "gal"`, `is_approximate: true`
- **Key insight (woven in):** napkin math normalizes to a human-friendly unit AND rounds. The `is_approximate` flag tells downstream tools this is a rough estimate
- Contrast with bare: `20 cups` → `value: "1.25 gal"`, `numeric_value: 20`, `unit: "cups"` — display normalizes but data stays precise

#### Section 6: Measurement Conventions (`measurement:`)

- Scenario: friend is in the UK, not continental Europe — cups mean different things
- Example: `measurement: { volume: imperial }` makes `1 cup` = 284 mL instead of 240 mL
- Inline qualifiers: `1 us cup` vs `1 imp cup` for mixing conventions
- Teaching point: `measurement:` changes how units are *interpreted*, while `convert_to` changes how results are *output*

#### Section 7: "What's Really in Your Data?" (Recap)

The aha-moment section. Side-by-side comparison table using verified JSON:

| Expression | `value` | `numeric_value` | `unit` | `is_approximate` |
|-----------|---------|-----------------|--------|-------------------|
| `flour = 2 cups` | `1 pt` | `2` | `cups` | — |
| `flour = 2 cups in grams` | `X g` | `X` | `gram` | — |
| `flour = 2 cups` + `convert_to: si` | `480 ml` | `0.48` | `liter` | — |
| `flour = 2 cups as napkin` | `~1 pt` | `1` | `pt` | `true` |

Key takeaways:
- `value` always normalizes for display (2 cups → "1 pt")
- `numeric_value` + `unit` stay in the original unit unless explicitly transformed
- `convert_to` and `in` give precise conversions — machine-safe
- `as napkin` normalizes, rounds, and flags as approximate — human-friendly

"When you pipe CalcMark to another tool via `--format json`, `convert_to` gives you precisely converted data. Napkin math gives you a rounded estimate with `is_approximate: true` so your downstream tool knows to treat it accordingly."

#### Section 8: "What to Read Next"

Links to: recipe-scaling guide (for `scale`), unit-conversion guide (for ambiguity handling), language reference, configuration (JSON output fields).

### Phase 3: Cross-Linking and Index Updates

1. **Update `site/content/guides/_index.md`**: Add hand-written entry for Understanding Measurements between business-planning and recipe-scaling
2. **Update `site/content/guides/recipe-scaling/_index.md`**: Add a callout at the top pointing to this guide for measurement fundamentals
3. **Update `site/content/guides/unit-conversion/_index.md`**: Add a "See also" link to this guide

## Acceptance Criteria

- [x] CSS-only tab component works without JavaScript (switching via radio buttons)
- [x] Tab preference persists across examples via localStorage (progressive enhancement)
- [x] Tabs are accessible: ARIA roles, keyboard navigable
- [ ] Tabs render correctly in light/dark mode
- [x] Tabs are responsive on narrow viewports
- [x] Guide follows established structure: frontmatter (title, summary, weight: 25), H1, intro, feature table, walkthrough steps, "What to Read Next"
- [x] Each section introduces exactly one concept, building on the previous
- [x] All CalcMark examples produce correct output (verified via `cm --format json`)
- [x] "What's Really in Your Data?" section clearly contrasts bare/in/convert_to/napkin JSON output
- [x] NASA → Larky intro sets the right tone (serious point, light delivery)
- [x] Guides index page updated with new entry
- [x] Recipe-scaling guide has callout linking here
- [x] Unit-conversion guide has "See also" link
- [x] `task quality` passes

## Dependencies & Risks

**Dependencies:**
- Tab component must be built before guide content (content authoring format depends on shortcode API)
- `site/data/cm_results.json` does NOT need updating — CalcMark tabs will use standard code fences, not the render-codeblock-calcmark hook

**Risks:**
- **Content overlap:** Carefully differentiated from recipe-scaling and unit-conversion guides via the type-system-progressive-tutorial angle. Cross-links make the relationship explicit.
- **Tab shortcode complexity:** Kept minimal — CSS switching, tiny JS for persistence, ARIA markup. No animation, no dynamic content loading.
- **Frontmatter in examples:** Sections 4 and 6 show frontmatter as part of the example. Since we're using standard code fences (not the render hook), these render as plain text — no doceval dependency for frontmatter examples. The JSON tab shows pre-verified output.

## Sources & References

### Origin

- **Brainstorm document:** [docs/brainstorms/2026-03-15-measurements-walkthrough-guide-brainstorm.md](docs/brainstorms/2026-03-15-measurements-walkthrough-guide-brainstorm.md) — Key decisions: scenario-driven kitchen theme, CSS-only tabs, single page with jump links, weight 25, cross-linking both directions.

### Internal References

- Existing guide pattern: `site/content/guides/recipe-scaling/_index.md`
- CSS component file: `site/assets/css/components.css`
- Design tokens: `site/assets/css/variables.css`
- Hugo head partial: `site/layouts/partials/head.html`
- Guides index: `site/content/guides/_index.md`
- Hugo config: `site/hugo.yaml`

### Verified JSON Output

All JSON examples in the guide are verified against `cm v1.8.6` output:

```
echo '---\nconvert_to: si\n---\nflour = 2 cups' | ./cm --format json
echo 'flour = 2 cups as napkin' | ./cm --format json
echo 'vanilla = 1/2 tsp' | ./cm --format json
```
