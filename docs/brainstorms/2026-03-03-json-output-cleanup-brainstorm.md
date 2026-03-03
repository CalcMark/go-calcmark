# JSON Output Cleanup

**Date:** 2026-03-03
**Issue:** [#11 - Feature: More complete JSON output structure](https://github.com/CalcMark/go-calcmark/issues/11)
**Status:** Ready for planning

## What We're Building

A cleaner, more structured JSON output from `cm convert --to=json` that:

1. **Removes redundant block-level fields** (`output` and `variables`) that duplicate information already available in `results[]`
2. **Replaces `raw_value`** with flat, structured decomposition fields (`type`, `numeric_value`, `unit`) that expose the underlying value components
3. **Adds type information** to every result for syntax highlighting and consumer-side dispatch

## Why This Approach

The current JSON output has two problems:

- **Redundancy:** `JSONBlock.Output` only contains the *last* calculation's raw value, while `results[]` already has every calculation's value. `JSONBlock.Variables` is extractable from `results[].variable`. Both are noise.
- **Inconsistency:** `output` uses raw `block.LastValue().String()` (locale-independent) while `value` uses locale-formatted `display.Format()`. Consumers see different representations for the same value (e.g., `"$6500.00"` vs `"$6,500.00"`).

The flat decomposition approach (type + numeric_value + unit) was chosen over nested objects because:
- It's simpler to consume — no nested `quantity` objects to traverse
- It mirrors what syntax highlighters need — a `type` discriminator and optional numeric/unit data
- It's stable — adding new types just adds a new `type` string value, no structural changes

## Key Decisions

### 1. Breaking change: remove `output` and `variables` immediately

The project is young with no known external JSON consumers. A clean break now avoids carrying redundancy forever. This removes `JSONBlock.Output` and `JSONBlock.Variables` entirely.

### 2. Per-result structure: flat with type discriminator

Every result gets a `type` field. Numeric results additionally get `numeric_value` (JSON number) and optionally `unit` (string). Non-numeric types (Date, Time, Boolean) rely on `value` alone. `numeric_value` is serialized as a JSON number; CalcMark uses fixed-precision decimals so no precision loss occurs in practice.

**Before:**
```json
{
  "source": "total = 5000 USD",
  "value": "$5,000.00",
  "raw_value": "$5000.00",
  "variable": "total"
}
```

**After:**
```json
{
  "source": "total = 5000 USD",
  "value": "$5,000.00",
  "type": "currency",
  "numeric_value": 5000,
  "unit": "USD",
  "variable": "total"
}
```

### 3. Type values map to spec/types

| CalcMark Type | `type` value  | Has `numeric_value` | Has `unit`          |
|---------------|---------------|---------------------|---------------------|
| Number        | `"number"`    | Yes                 | No                  |
| Currency      | `"currency"`  | Yes                 | Yes (e.g., `"USD"`) |
| Quantity      | `"quantity"`  | Yes                 | Yes (e.g., `"kg"`)  |
| Rate          | `"rate"`      | Yes                 | Yes (e.g., `"MB/s"`)  |
| Duration      | `"duration"`  | Yes                 | Yes (e.g., `"hours"`) |
| Date          | `"date"`      | No                  | No                  |
| Time          | `"time"`      | No                  | No                  |
| Boolean       | `"boolean"`   | No                  | No                  |

### 4. Rate units use compound string

Rate types represent their unit as a compound string (e.g., `"MB/s"`, `"USD/hour"`) rather than decomposing into separate numerator/denominator fields. Consumers can split on `/` if needed.

### 5. Napkin estimates surface `is_approximate`

Results from napkin estimates include `"is_approximate": true`. Omitted (not `false`) when the value is exact. Useful for syntax highlighters to render approximate values differently (e.g., with a `~` prefix or distinct styling).

### 6. `value` remains the locale-formatted display string

The `value` field continues to be the locale-formatted, human-readable display string. This is the only field affected by `--locale`. All other fields (`type`, `numeric_value`, `unit`) are locale-independent.

## Examples Across Types

**Number:**
```json
{"source": "x = 42", "value": "42", "type": "number", "numeric_value": 42, "variable": "x"}
```

**Currency:**
```json
{"source": "total = 5000 USD", "value": "$5,000.00", "type": "currency", "numeric_value": 5000, "unit": "USD", "variable": "total"}
```

**Quantity:**
```json
{"source": "weight = 5 kg", "value": "5 kg", "type": "quantity", "numeric_value": 5, "unit": "kg", "variable": "weight"}
```

**Rate:**
```json
{"source": "speed = 100 MB/s", "value": "100 MB/s", "type": "rate", "numeric_value": 100, "unit": "MB/s", "variable": "speed"}
```

**Duration:**
```json
{"source": "elapsed = 2.5 hours", "value": "2.5 hours", "type": "duration", "numeric_value": 2.5, "unit": "hours", "variable": "elapsed"}
```

**Date:**
```json
{"source": "d = 2026-03-03", "value": "2026-03-03", "type": "date", "variable": "d"}
```

**Boolean:**
```json
{"source": "flag = true", "value": "true", "type": "boolean", "variable": "flag"}
```

**Napkin estimate:**
```json
{"source": "estimate = ~500 USD", "value": "~$500.00", "type": "currency", "numeric_value": 500, "unit": "USD", "is_approximate": true, "variable": "estimate"}
```

**Error result:**
```json
{"source": "y = unknown_var + 1", "error": "undefined variable: unknown_var"}
```

### 7. Failed results include per-result error

When a calculation line fails, it still appears in `results[]` with `source` and an `error` string, but no `type`, `numeric_value`, or `unit`. This gives consumers line-by-line status. Block-level `diagnostics` remain for structured error details.

## Scope

### In scope
- Update `JSONResult` struct: remove `raw_value`, add `type`, `numeric_value`, `unit`, `is_approximate`
- Update `JSONBlock` struct: remove `Output` and `Variables` fields
- Update `json_formatter.go` to populate new fields via type-switching on the `types.Type` interface
- Update all JSON formatter tests
- Update user-guide, configuration, and CLI reference docs

### Out of scope
- JSON Schema file (`.schema.json`) — can be added later
- Versioned JSON output — not needed at this project stage
- Changes to other output formats (text, markdown, HTML)
