---
title: "User Guide"
summary: "Complete guide to CalcMark features, editor shortcuts, and workflows."
weight: 20
---

## Editor Shortcuts

The CalcMark editor provides keyboard shortcuts for common actions. Press **F1** for full help inside the editor.

### File

| Shortcut | Action |
|----------|--------|
| Ctrl+S | Save document |
| Ctrl+O | Open file |
| Ctrl+E | Export to format |
| Ctrl+Q | Quit editor |

### Edit

| Shortcut | Action |
|----------|--------|
| Ctrl+Z | Undo last change |
| Ctrl+Y | Redo last change |
| Ctrl+K | Delete current line |
| Ctrl+F | Add YAML frontmatter |

### View

| Shortcut | Action |
|----------|--------|
| Ctrl+P | Cycle preview mode |

### Navigation

| Shortcut | Action |
|----------|--------|
| Opt+Left / Opt+B | Move to previous word |
| Opt+Right / Opt+F | Move to next word |
| Ctrl+Home | Jump to document start |
| Ctrl+End | Jump to document end |
| Ctrl+D | Scroll down half page |
| Ctrl+U | Scroll up half page |

## Exporting Results

Convert CalcMark files to other formats using `cm convert`:

```bash
cm convert doc.cm --to=html              # HTML to stdout
cm convert doc.cm --to=md -o doc.md      # Markdown file
cm convert doc.cm --to=json              # JSON to stdout
cm convert doc.cm --to=html -T tpl.html  # Custom HTML template
```

Use `cm eval` for quick evaluation:

```bash
cm eval budget.cm            # Print final results
cm eval -v budget.cm         # Show all intermediate values
echo "1 + 2" | cm eval      # Evaluate from stdin
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

Run `cm help constants` for the complete list.

### Unit Conversion

Convert between compatible units using `in` or `as`:

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

{{< feature-table category="function" >}}

See the [Language Reference](/docs/language-reference/) for the complete list of functions, including [natural language syntax](/docs/language-reference/#natural-language-syntax) forms.

### Rates and the `over` keyword {#rates}

Define and work with rates (quantity per time). Use `over` to accumulate a rate over a duration:

```cm
salary = $120000/year
hourly_rate = $75/hour
daily_earnings = hourly_rate over 8 hours
bandwidth = 100 MB/s
monthly_transfer = bandwidth over 30 days
```

### Napkin Math {#napkin-math}

Use `as napkin` to round results to 2 significant figures -- useful for quick estimates:

```cm
monthly_transfer = 100 MB/s over 30 days
monthly_transfer as napkin

exact_servers = 10000 req/s at 450 req/s per server
exact_servers as napkin
```

The `as napkin` modifier normalizes units and adds a `~` prefix to signal the result is an approximation.

### Date Arithmetic

Work with dates and durations:

```cm
project_start = Jan 15 2025
duration = 12 weeks
project_end = project_start + duration

deadline = Jun 1 2025
launch = deadline - 2 weeks
```

### Capacity Planning {#capacity-planning}

Use the `at...per` syntax to calculate how many units you need:

```cm
disks = 10 TB at 2 TB per disk
servers = 10000 req/s at 450 req/s per server
servers_buffered = 10000 req/s at 450 req/s per server with 20% buffer
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

### Use the Preview Pane

Press **Ctrl+P** in the editor to toggle the preview pane, which shows evaluated results alongside your source.

### Get Help on Functions

Run `cm help functions` to see all available functions with descriptions and usage patterns. Run `cm help constants` for unit constants.

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
