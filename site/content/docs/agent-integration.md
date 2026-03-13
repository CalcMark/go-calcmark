---
title: "Agent & API Integration"
summary: "Use CalcMark as a calculation engine from scripts, AI agents, and pipelines."
weight: 36
---

CalcMark works as a command-line calculation engine. Pipe expressions in, get structured JSON out. No server, no SDK -- just stdin/stdout.

## Installation

Check if `cm` is available:

```bash
cm version
```

If not found, install it:

**macOS / Linux (Homebrew):**

```bash
brew install calcmark/tap/calcmark
```

**Linux (direct download):**

```bash
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  armv6l)  ARCH="armv6" ;;
esac
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
curl -sL "https://github.com/CalcMark/go-calcmark/releases/latest/download/calcmark_${OS}_${ARCH}.tar.gz" | tar xz -C /usr/local/bin cm
```

Always ask the user before installing software.

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

Or evaluate a file:

```bash
cm eval budget.cm --format json
```

## Essential Syntax

### Variables and Expressions

```calcmark
price = $100
quantity = 12
total = price * quantity
```

### Units and Conversions

```calcmark
distance = 42.195 km
distance in miles
weight = 5 kg in pounds
```

### Currency

Supported symbols: `$` (USD), `€` (EUR), `£` (GBP), `¥` (JPY). Any 3-letter ISO code works as a postfix: `100 CHF`, `50 CAD`.

### Percentages

```calcmark
marked_up = $100 + 15%
discounted = $250 - 20%
15% of $2000
```

When added/subtracted with another type, percentages widen: `$100 + 15%` = `$115.00`.

### Dates and Durations

```calcmark
deadline = Jan 15 2025
remaining = deadline - today
launch = today + 90 days
2 weeks from today
```

### Rates

```calcmark
speed = 100 MB/s
cost = $50/hour
throughput = 1000 req/s
```

Rate accumulation with `over`:

```calcmark
monthly_cost = $50/hour over 30 days
data = 100 MB/s over 1 day
```

### Capacity Planning

```calcmark
servers = 50000 req/s at 2000 req/s per server with 25% buffer
```

### Napkin Math

Round to 2 significant figures with magnitude suffix:

```calcmark
1234567 as napkin
```

Result: `~1.2M`

### Multiplier Suffixes

```calcmark
10K              -> 10,000
5M               -> 5,000,000
2B               -> 2,000,000,000
1.5T             -> 1,500,000,000,000
```

## Functions

Prefer the natural language (NL) form where available:

| Function | NL Form | Example |
|----------|---------|---------|
| `compound(p, r, t)` | `compound P by R over T` | `compound $10000 by 7% over 30 years` |
| `depreciate(v, r, t)` | `depreciate V by R over T` | `depreciate $50000 by 15% over 5 years` |
| `grow(start, step, n)` | `grow S by X over N` | `grow $500 by $100 over 36` |
| `accumulate(r, t)` | `rate over duration` | `1000 req/s over 1 day` |
| `capacity(d, c, u, b)` | `demand at cap per unit` | `50000 req/s at 2000 req/s per server with 25% buffer` |
| `convert_rate(r, t)` | `rate per time` | `1200 req/s per minute` |
| `read(size, type)` | `read X from Y` | `read 100 MB from ssd` |
| `compress(size, algo)` | `compress X using Y` | `compress 1 GB using gzip` |
| `transfer_time(s, scope, net)` | `transfer X across Y Z` | `transfer 500 KB across regional gigabit` |
| `avg(a, b, c)` | `average of a, b, c` | `average of $100, $200, $300` |
| `sum(a, b, c)` | `sum of a, b, c` | `sum of $50, $75, $100` |
| `sqrt(x)` | `square root of x` | `square root of 144` |
| `number(x)` | | `number($100)` → `100` (strips unit) |

Run `cm help functions` for the complete list with signatures.

### Smart Type Handling

CalcMark automatically handles many type combinations without `number()`:

```calcmark
monthly_users = 2500 customers
price = $49
revenue = price * monthly_users
```

Result: `$122,500.00` -- currency * quantity gives currency, dropping the custom unit.

More examples:
- `$100 + 15%` → `$115.00` (percentage widening)
- `€4500 in USD` → converts using frontmatter exchange rates
- `$450 * servers` → currency, where `servers` is a quantity from capacity planning

Use `number()` only when you need a dimensionless value, such as dividing two currencies for a ratio: `number(ltv) / number(cac)`.

## Frontmatter

Documents can start with YAML frontmatter for configuration:

```yaml
---
title: Budget Analysis
locale: en-US
globals:
  tax_rate: 0.32
  headcount: 12
exchange:
  USD_EUR: 0.92
  EUR_USD: 1.09
---
```

Reference globals with `@globals.name`:

```calcmark
team_cost = salary * @globals.headcount
```

An agent can research current exchange rates and insert them into frontmatter for accurate multi-currency documents.

Run `cm help frontmatter` for all directives including `scale` and `convert_to`.

## Template Interpolation

Embed calculated values in prose with `{{variable_name}}`:

```calcmark
total = $1500
servers = 50000 req/s at 2000 req/s per server with 25% buffer

The project costs {{total}} and requires {{servers}} app servers.
```

When combined with `cm convert --to=html`, this produces readable reports with inline results.

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
| `cm convert --to=html` | HTML document with rendered markdown and results |
| `cm convert --to=md` | Markdown with results inline |

Convert to a file with `-o`:

```bash
cm convert report.cm --to=html -o report.html
```

## Error Handling

Undefined variables and type errors produce a non-zero exit code and an error message on stderr:

```bash
echo "x = unknown + 1" | cm --format json
# Exit code 1
# stderr: evaluation error: undefined_variable: undefined variable "unknown"
```

**Errors go to stderr as plain text, not JSON.** Always check the exit code.

**Silent misinterpretation:** If CalcMark doesn't recognize an expression, it treats it as markdown prose. The JSON will contain `"type": "text"` blocks instead of `"type": "calculation"`. Always verify your output contains calculation blocks when you expect them.

## Common Pitfalls

### Variables Are Immutable

Variables cannot be reassigned. This will error:

```calcmark
x = 10
x = 20   # ERROR: variable_redefinition: cannot reassign 'x'
```

Use distinct names for each step of a calculation:

```calcmark
base_cost = $100
adjusted_cost = base_cost + 15%
```

### Unit Propagation

Arithmetic on quantities preserves the unit of the numerator. This can produce unexpected results:

```calcmark
customers = 343 customers
servers_raw = customers / 10   # Result: 34.3 customers (NOT servers!)
```

Use the NL `capacity` form to get correct unit conversion:

```calcmark
servers = 343 customers at 10 customers per server   # Result: 35 server ✓
```

### Prefer NL Forms Over Raw Arithmetic

CalcMark's natural language functions handle rounding, units, and edge cases that raw division does not. When a built-in function exists, use it:

| Instead of | Use |
|------------|-----|
| `total / num_servers` | `demand at cap per unit` (capacity) |
| `principal * (1 + rate) ^ years` | `compound P by R over T` |
| `start + (step * periods)` | `grow S by X over N` |

Run `cm help functions` to discover all available NL forms.

### Error Output Goes to stderr

Errors are plain text on stderr, not JSON. If you pipe `cm` output directly into a JSON parser and an error occurs, the parser will receive empty input and fail silently. Always check the exit code:

```bash
if ! result=$(cm eval file.cm --format json 2>/dev/null); then
  cm eval file.cm 2>&1  # re-run to see the error message
fi
```

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

List all available functions, units, and frontmatter directives at runtime:

```bash
cm help functions    # All functions with signatures and NL forms
cm help constants    # All unit constants with aliases
cm help frontmatter  # All frontmatter directives with valid options
cm help --all        # Everything at once
```

## Workflow Patterns

### Quick Calculation

Pipe a one-liner and extract the result from JSON:

```bash
echo "servers = 50000 req/s at 2000 req/s per server with 25% buffer" | cm --format json
```

### Research Artifact

Write a `.cm` file as evidence of analytical work:

1. Write the `.cm` file with calculations and prose
2. Evaluate: `cm eval analysis.cm --format json`
3. Both the source file and JSON results serve as artifacts

### Document Deliverable

Create a CalcMark document and convert it for the user:

1. Write the `.cm` file with frontmatter, headers, calculations, and `{{template}}` interpolation
2. Convert: `cm convert report.cm --to=html -o report.html`
3. Deliver the HTML (or markdown) to the user

### Iterative Analysis

Build up a `.cm` file across multiple steps:

1. Start with initial assumptions and calculations
2. Evaluate and review results
3. Adjust variables based on findings
4. Re-evaluate to see updated results
5. Repeat until the analysis is complete

## Security

- Always ask the user before installing `cm`
- Write `.cm` files only in the project directory or temp directory
- Never modify shell profiles or global PATH
- Never run `cm` with elevated privileges

## Agent Skill

A distributable CalcMark skill is available for AI coding agents. It wraps this page's content with platform-specific metadata for Claude Code, Cursor, Copilot CLI, and Gemini CLI:

[github.com/CalcMark/agent-skill](https://github.com/CalcMark/agent-skill)

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
