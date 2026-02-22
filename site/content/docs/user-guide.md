---
title: "User Guide"
summary: "Complete guide to CalcMark features, REPL commands, and workflows."
weight: 20
---

## REPL Commands

Press `:` to enter command mode, then type a command:

| Command | Description |
|---------|-------------|
| `:help` | Show help topics |
| `:help units` | List all supported units |
| `:help functions` | List available functions |
| `:open <file>` | Load a CalcMark file |
| `:save <file.cm>` | Save session as CalcMark |
| `:output <file>` | Export to HTML, Markdown, or JSON |
| `:pin` | Pin all variables to the sidebar |
| `:pin <var>` | Pin a specific variable |
| `:unpin <var>` | Unpin a variable |
| `:md` | Enter multi-line markdown mode |
| `:quit` | Exit |

### Keyboard Shortcuts

- `Esc` - Exit current mode (command, markdown, help)
- `Esc Esc` - Clear input line (double-tap quickly)
- `Ctrl+C` or `Ctrl+D` - Quit
- `Up/Down` - Navigate command history
- `PgUp/PgDn` - Scroll help viewer

## Output Formats

### Save Your Work

Save the session as a CalcMark file (calculations + markdown):

```bash
:save my-budget.cm
```

### Export Results

Export evaluated results in different formats:

```bash
:output report.html     # Formatted HTML with results
:output summary.md      # Markdown with calculations shown
:output data.json       # Structured JSON for processing
```

### Command Line Export

```bash
cm eval budget.cm --json > results.json
cm eval budget.cm > results.txt
```

## Language Features

### Supported Units

CalcMark supports a wide range of units across categories:

- **Length**: m, km, ft, mi, in, cm, mm
- **Mass**: kg, g, lb, oz
- **Volume**: L, mL, gal, cup, tbsp, tsp
- **Time**: second, minute, hour, day, week, month, year
- **Temperature**: C, F, K
- **Data**: byte, KB, MB, GB, TB
- **Area**: m2, ft2, km2, acre
- **Speed**: mph, km/h, m/s
- **Data Rate**: Mbps, Gbps

Use `:help units` in the REPL for the complete list.

### Unit Conversion

Convert between compatible units using `in`:

```cm
distance = 5 miles
distance_km = distance in km

temp_c = 20 celsius
temp_f = temp_c in fahrenheit

file_size = 1.5 GB
file_size_mb = file_size in MB
```

### Currency Conversion

Convert between currencies using `in` with exchange rates defined in YAML frontmatter:

```yaml
---
exchange:
  USD/EUR: 0.92
  EUR/GBP: 0.86
---
```

```cm
price_usd = $100
price_eur = price_usd in EUR

salary = 50000 EUR
salary_gbp = salary in GBP
```

Exchange rates use the format `FROM/TO: rate` where 1 unit of FROM equals `rate` units of TO.

### Global Variables

Define reusable values in the frontmatter that can be referenced throughout your document:

```yaml
---
globals:
  base_date: Jan 15 2025
  tax_rate: 0.32
  base_price: $100
  sprint_length: 2 weeks
  bandwidth: 100 MB/s
---
```

```cm
net_price = base_price * (1 - tax_rate)
project_end = base_date + sprint_length * 6
monthly_transfer = bandwidth over 30 days
```

Globals support all CalcMark literal types:

- **Numbers**: `42`, `3.14`, `1.5K`, `25%`
- **Quantities**: `10 meters`, `5 kg`, `100 MB`
- **Currencies**: `$100`, `50 EUR`
- **Dates**: `Jan 15 2025`
- **Durations**: `5 days`, `2 weeks`
- **Rates**: `100 MB/s`, `$50/hour`
- **Booleans**: `true`, `false`

Globals must be literal values. Expressions like `1 + 1` are not allowed.

### Built-in Functions

| Function | Description | Example |
|----------|-------------|---------|
| `avg()` | Average of values | `avg(10, 20, 30)` |
| `sqrt()` | Square root | `sqrt(144)` |
| `accumulate()` | Rate x time | `accumulate(100/hour, 8 hours)` |
| `capacity()` | Ceiling division with unit | `capacity(1000, 100, server)` |
| `downtime()` | SLA to downtime | `downtime(99.9%, year)` |
| `rtt()` | Network round-trip time | `rtt(regional)` |
| `throughput()` | Network bandwidth | `throughput(gigabit)` |

### Rates

Define and work with rates (quantity per time):

```cm
salary = $120000/year
hourly_rate = $75/hour
daily_earnings = hourly_rate over 8 hours
bandwidth = 100 MB/s
monthly_transfer = bandwidth over 30 days
```

### Date Arithmetic

Work with dates and durations:

```cm
project_start = Jan 15 2025
duration = 12 weeks
project_end = project_start + duration

deadline = Jun 1 2025
launch = deadline - 2 weeks
```

### Multiplier Suffixes

Use K, M, B for large numbers:

```cm
users = 10M
revenue = $5B
requests = 100K
```

### Percentages

Percentages work naturally in calculations:

```cm
price = $100
discount = 20%
sale_price = price * (1 - discount)

tax_rate = 8.25%
tax = price * tax_rate
```

## Tips

### Reactive Updates

In the REPL, changing a variable automatically updates all dependent values. Pin important variables to the sidebar to watch them change:

```bash
:pin total_cost
:pin profit_margin
```

### Organize with Markdown

Use headers and prose to structure your thinking:

```cm
# Q1 Budget

## Revenue Assumptions
monthly_revenue = $50000
q1_months = 3
total_revenue = monthly_revenue * q1_months

## Cost Breakdown
fixed_costs = $20000
variable_pct = 30%
variable_costs = total_revenue * variable_pct
```

### Iterate Quickly

Load a file, tweak values in the REPL, then save when satisfied:

```bash
cm budget.cm           # Load and explore
# ... make changes ...
:save budget-v2.cm     # Save your iteration
```

## Troubleshooting

### "Undefined variable"

Variables must be defined before use. Check that:
1. The variable is spelled correctly
2. It's defined on an earlier line
3. No typos in the name

### "Incompatible units"

You can't add meters to kilograms. Check that operations make physical sense.

### "Parse error"

The line isn't valid CalcMark syntax. Common issues:
- Missing operator between values
- Unclosed parentheses
- Invalid characters
