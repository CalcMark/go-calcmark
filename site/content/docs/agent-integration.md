---
title: "Agent & API Integration"
summary: "Use CalcMark as a calculation engine from scripts, AI agents, and pipelines."
weight: 36
---

CalcMark works as a command-line calculation engine. Pipe expressions in, get structured JSON out. No server, no SDK -- just stdin/stdout.

## Quick Start

```bash
echo "price = $100\ntax = price * 0.08\ntotal = price + tax" | cm --format json
```

Every result includes the display value, numeric value, type, and unit -- everything a program needs to use the result downstream.

## Input

Send CalcMark expressions to stdin. Use `\n` to separate lines:

```bash
echo "a = 1 + 1\nb = a * 2" | cm --format json
```

For longer documents, use a heredoc:

```bash
cm --format json <<'EOF'
distance = 42.195 km
time = 3.5 hours
pace = time / distance
EOF
```

Or pipe a file:

```bash
cm eval budget.cm --format json
```

## JSON Response Structure

The response is an array of **blocks**. Each block is either `"calculation"` or `"text"` (markdown):

```json
{
  "blocks": [
    {
      "type": "text",
      "source": ["# Budget"],
      "html": "<h1 id=\"budget\">Budget</h1>\n"
    },
    {
      "type": "calculation",
      "source": ["price = $100", "tax = price * 0.08"],
      "results": [
        {
          "source": "price = $100",
          "value": "$100.00",
          "type": "currency",
          "numeric_value": 100,
          "unit": "USD",
          "variable": "price"
        },
        {
          "source": "tax = price * 0.08",
          "value": "$8.00",
          "type": "currency",
          "numeric_value": 8,
          "unit": "USD",
          "variable": "tax"
        }
      ]
    }
  ]
}
```

### Result Fields

| Field | Always present | Description |
|-------|---------------|-------------|
| `source` | yes | The original CalcMark expression |
| `value` | yes | Display-formatted result (locale-aware) |
| `type` | yes | CalcMark type: `number`, `currency`, `quantity`, `duration`, `rate`, `date`, `boolean` |
| `numeric_value` | yes | Machine-readable number (always uses `.` decimal, no formatting) |
| `variable` | when assigned | Variable name if this was an assignment (`x = ...`) |
| `unit` | when typed | Unit identifier: `USD`, `km`, `second`, etc. |
| `is_explicit` | when converted | `true` if the result used an explicit `in`/`as` conversion |
| `is_approximate` | when napkin | `true` if `as napkin` rounding was applied |
| `date_value` | for dates | ISO 8601 date string (`2025-01-08`) |

Use `type` for dispatch, `numeric_value` + `unit` for computation, and `value` for display.

## Output Formats

| Flag | Use case |
|------|----------|
| `--format json` | Structured data for programs and agents |
| `--format text` | Human-readable plain text |
| `--format html` | HTML document with rendered markdown |
| `--format md` | Markdown with results inline |
| `--format cm` | Normalized CalcMark source (round-trip safe) |

## Error Handling

Undefined variables and type errors produce a non-zero exit code and an error message on stderr:

```bash
echo "x = unknown + 1" | cm --format json
# Exit code 1
# stderr: evaluation error: undefined_variable: undefined variable "unknown"
```

Markdown lines (prose, headers, lists) are not errors -- they appear as `"type": "text"` blocks in the JSON output.

## Verbose Mode

By default, only the last result is shown in text mode. Use `-v` to get all intermediate values:

```bash
echo "a = 10\nb = a * 2\nc = b + 1" | cm -v
# 10
# 20
# 21
```

In JSON mode (`--format json`), all results are always included regardless of `-v`.

## Feature Discovery

List all available functions and units at runtime:

```bash
cm functions    # All functions with signatures and NL forms
cm constants    # All unit constants with aliases
```

## What CalcMark Can Do

CalcMark handles arithmetic, unit conversions, currency, dates, rates, capacity planning, and growth modeling. A few examples:

```bash
# Unit conversion
echo "marathon = 26.2 miles in km" | cm --format json

# Date arithmetic
echo "deadline = Jan 15 2025 + 90 days" | cm --format json

# Rate accumulation
echo "cost = $0.10/hour over 30 days" | cm --format json

# Capacity planning
echo "servers = 50000 req/s at 2000 req/s per server with 25% buffer" | cm --format json

# Napkin math (2 significant figures)
echo "1234567 as napkin" | cm --format json
```

See the [Language Reference](/docs/language-reference/) for the full specification and the [User Guide](/docs/user-guide/) for all features.

## Source Code

CalcMark is open source: [github.com/CalcMark/go-calcmark](https://github.com/CalcMark/go-calcmark)

Key directories for understanding the language:

| Path | What's there |
|------|-------------|
| `spec/` | Language specification (grammar, types, parser) |
| `spec/features/registry.go` | Canonical feature list with all functions, units, and aliases |
| `spec/units/canonical.go` | All supported physical units |
| `impl/interpreter/` | Evaluation engine |
| `testdata/` | Golden tests and example `.cm` files |
| `testdata/examples/` | Complete worked examples (runnable) |
