# CalcMark User Guide

CalcMark is a calculation language that blends seamlessly with markdown. Write your thinking in plain English, then add calculations that reference each other. Change one number and watch everything update.

## Quick Start

### Interactive REPL

Start the interactive environment:

```bash
cm                    # Empty REPL
cm budget.cm          # Load a file and explore
```

### Evaluate a File

Process a file and see results:

```bash
cm eval docs/examples/system-sizing.cm
```

### Pipe Expressions

Quick calculations from the command line:

```bash
echo "price = 100 USD" | cm eval
echo "24 celsius in fahrenheit" | cm eval
echo "500 gram in oz" | cm eval
```

## Core Concepts

### Variables Flow Downward

Variables must be defined before use. Later lines can reference earlier ones:

```
base_salary = $85000
bonus_pct = 15%
bonus = base_salary * bonus_pct
total_comp = base_salary + bonus
```

### Units Are First-Class

CalcMark understands physical units and currencies:

```
distance = 42.195 km              # Marathon distance
time = 3 hours + 30 minutes       # Finishing time
pace = time / distance            # Automatic unit math

price_usd = 100 USD
price_eur = 85 EUR
```

### Markdown is Ignored

Write prose freely. Only lines that parse as calculations are evaluated:

```
# Project Budget

We need to account for both development and infrastructure costs.

dev_team = 5
monthly_salary = $12000
dev_cost = dev_team * monthly_salary * 6 months

Infrastructure will be roughly 20% of dev costs.

infra_pct = 20%
infra_cost = dev_cost * infra_pct
```

## REPL Commands

Press `/` to enter command mode, then type a command:

| Command | Description |
|---------|-------------|
| `/help` | Show help topics |
| `/help units` | List all supported units |
| `/help functions` | List available functions |
| `/open <file>` | Load a CalcMark file |
| `/save <file.cm>` | Save session as CalcMark |
| `/output <file>` | Export to HTML, Markdown, or JSON |
| `/pin` | Pin all variables to the sidebar |
| `/pin <var>` | Pin a specific variable |
| `/unpin <var>` | Unpin a variable |
| `/md` | Enter multi-line markdown mode |
| `/quit` | Exit |

### Keyboard Shortcuts

- `Esc` - Exit current mode (command, markdown, help)
- `Esc Esc` - Clear input line (double-tap quickly)
- `Ctrl+C` or `Ctrl+D` - Quit
- `↑/↓` - Navigate command history
- `PgUp/PgDn` - Scroll help viewer

## Output Formats

### Save Your Work

Save the session as a CalcMark file (calculations + markdown):

```
/save my-budget.cm
```

### Export Results

Export evaluated results in different formats:

```
/output report.html     # Formatted HTML with results
/output summary.md      # Markdown with calculations shown
/output data.json       # Structured JSON for processing
```

### Command Line Export

```bash
cm eval budget.cm --json > results.json
cm eval budget.cm > results.txt
```

## Example Workflows

See the `docs/examples/` directory for complete worked examples:

### System Sizing (`system-sizing.cm`)

Back-of-napkin infrastructure estimation for a 10M user social media app:
- Daily active user calculations
- Storage and bandwidth estimates
- Database replica sizing using `capacity()` function
- Monthly cost projections

```bash
cm docs/examples/system-sizing.cm
```

### Project Workback (`project-workback.cm`)

Working backwards from a launch date:
- Phase duration planning
- Buffer and risk calculations
- Date arithmetic (subtracting durations from dates)
- Resource allocation and cost estimates

```bash
cm docs/examples/project-workback.cm
```

### Recipe Scaling (`recipe-scaling.cm`)

Converting and scaling a recipe:
- Metric to US customary conversions
- Batch scaling (1 loaf → 4 loaves)
- Temperature conversions (Celsius → Fahrenheit)
- Cost per unit calculations

```bash
cm docs/examples/recipe-scaling.cm
```

### Household Budget (`household-budget.cm`)

Monthly budget planning:
- Income and tax calculations
- Fixed vs variable expense tracking
- 50/30/20 rule analysis
- Emergency fund runway calculations

```bash
cm docs/examples/household-budget.cm
```

### Job Offer Comparison (`job-offer.cm`)

Evaluating competing job offers:
- Base salary, bonus, and equity comparison
- RSU vs stock option valuation
- 4-year vesting schedules with cliff
- Risk-adjusted compensation analysis

```bash
cm docs/examples/job-offer.cm
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

Use `/help units` in the REPL for the complete list.

### Unit Conversion

Convert between compatible units using `in`:

```
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
price_usd = $100
price_eur = price_usd in EUR    # → €92.00

salary = 50000 EUR
salary_gbp = salary in GBP      # → £43000.00
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

Note: Globals must be literal values. Expressions like `1 + 1` are not allowed.

### Built-in Functions

| Function | Description | Example |
|----------|-------------|---------|
| `avg()` | Average of values | `avg(10, 20, 30)` |
| `sqrt()` | Square root | `sqrt(144)` |
| `accumulate()` | Rate × time | `accumulate(100/hour, 8 hours)` |
| `capacity()` | Ceiling division with unit | `capacity(1000, 100, server)` |
| `downtime()` | SLA to downtime | `downtime(99.9%, year)` |
| `rtt()` | Network round-trip time | `rtt(regional)` |
| `throughput()` | Network bandwidth | `throughput(gigabit)` |

### Rates

Define and work with rates (quantity per time):

```
salary = $120000/year
hourly_rate = $75/hour
daily_earnings = hourly_rate over 8 hours
bandwidth = 100 MB/s
monthly_transfer = bandwidth over 30 days
```

### Date Arithmetic

Work with dates and durations:

```
project_start = Jan 15 2025
duration = 12 weeks
project_end = project_start + duration

deadline = Jun 1 2025
launch = deadline - 2 weeks
```

### Multiplier Suffixes

Use K, M, B for large numbers:

```
users = 10M                        # 10,000,000
revenue = $5B                      # $5,000,000,000
requests = 100K                    # 100,000
```

### Percentages

Percentages work naturally in calculations:

```
price = $100
discount = 20%
sale_price = price * (1 - discount)

tax_rate = 8.25%
tax = price * tax_rate
```

## Tips

### Reactive Updates

In the REPL, changing a variable automatically updates all dependent values. Pin important variables to the sidebar to watch them change:

```
/pin total_cost
/pin profit_margin
```

### Organize with Markdown

Use headers and prose to structure your thinking:

```
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
/save budget-v2.cm     # Save your iteration
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

## Next Steps

- Explore the example files in `docs/examples/`
- Try `/help` in the REPL to discover features
- Build your own calculation documents!
