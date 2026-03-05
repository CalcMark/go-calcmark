# Brainstorm: Datacenter Build Cost Example

**Date:** 2026-03-05
**Status:** Draft

## What We're Building

A new example page for the CalcMark site that walks a reader through the real-world costs of building a small dedicated datacenter. The page serves two purposes:

1. **Feature showcase** -- demonstrates compound, grow, accumulate, depreciate, exchange rates (front matter), napkin math, unit conversions (`as`/`in`), globals, rates, percentages, and capacity planning in one cohesive document.
2. **Practical guide** -- helps someone actually understand and model datacenter economics (CapEx, OpEx, depreciation, build-vs-colo tradeoff).

### Deliverables

- `testdata/examples/datacenter-cost.cm` -- standalone runnable CalcMark file with markdown prose and calculations
- `site/content/docs/examples/datacenter-cost.md` -- interleaved article that presents the `.cm` snippets with deeper explanations of both the datacenter domain and the CalcMark features being used

## Why This Approach

**Interleaved snippets** (not full-file-then-annotations) because:
- The article reads naturally as a tutorial
- Each snippet is immediately followed by its explanation
- Readers don't need to scroll back and forth between a monolithic code block and annotations
- The standalone `.cm` file exists separately for those who want the runnable version

**Full lifecycle scope** to maximize CalcMark feature coverage across one compelling narrative.

## Key Decisions

1. **Article structure:** Interleaved snippets with prose explanations (not full-file-first)
2. **Scope:** Full lifecycle -- CapEx breakdown, tier comparison, OpEx, depreciation, growth, build-vs-colo, modular alternative, napkin summary
3. **CM file:** Standalone with embedded markdown prose (like system-sizing.cm)
4. **Exchange rate scenario:** International vendor pricing (USD primary, EUR for European equipment, GBP for UK colocation)
5. **Target features to showcase:**
   - Front matter: `exchange` rates (USD_EUR, USD_GBP) and `globals`
   - `compound()` -- projecting OpEx growth over years
   - `grow()` -- linear capacity additions with modular builds
   - `accumulate()` -- total costs over time periods (rates over duration)
   - `depreciate()` -- equipment and facility depreciation with salvage floor
   - `as napkin` -- human-readable summary figures
   - `in EUR` / `in GBP` -- currency conversion for international vendors
   - `as` / `in` for unit conversions (e.g., kW to MW)
   - Percentages (`40% of total_build`)
   - Rates (`$X/month`, `$/sqft`)
   - Multiplier suffixes (`1M`, `7.7M`)
   - Comparison operators for decision logic

## Planned Sections (CM file structure)

1. **Front matter** -- exchange rates (USD_EUR, USD_GBP), globals (facility_sqft)
2. **Facility Sizing & Baseline CapEx** -- sqft, cost/sqft ranges, total build cost
3. **CapEx Breakdown** -- electrical (40-45%), building fit-out (20-25%), land/shell (15-20%), HVAC/cooling (15-20%)
4. **Tier Comparison** -- Tier II vs III cost multipliers
5. **International Equipment** -- European cooling vendor in EUR, convert to USD
6. **Power Infrastructure** -- cost per MW, kW to MW conversion
7. **Operating Expenses** -- annual OpEx, electricity rates, maintenance percentages
8. **OpEx Growth Projection** -- compound() for annual cost escalation
9. **Equipment Depreciation** -- depreciate() servers, cooling, with salvage floor
10. **Build vs. Colocation** -- 3-year TCO comparison
11. **Modular/Prefab Alternative** -- grow() for pay-as-you-grow capacity
12. **Summary** -- napkin math for all key figures

## Open Questions

None -- all key decisions resolved.
