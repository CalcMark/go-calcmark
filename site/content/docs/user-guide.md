---
title: "User Guide"
summary: "Complete guide to CalcMark features, editor shortcuts, and workflows."
weight: 20
---

## Contents {#contents}

- [Editor Shortcuts](#editor-shortcuts) -- Keyboard shortcuts for the CalcMark editor
- [Locale Formatting](#locale-formatting) -- Display numbers in your locale
- [Exporting Results](#exporting-results) -- `cm convert` and `cm eval`
- [Sharing with GitHub Gist](#sharing-gist) -- Share and open documents via GitHub Gist
- [Language Features](#language-features)
  - [Supported Units](#units) -- Physical, data, and currency units
  - [Unit Conversion](#unit-conversion) -- `in` and `as` keywords
  - [Currency Conversion](#currency-conversion) -- Exchange rates in frontmatter
  - [Global Variables](#global-variables) -- Reusable values in frontmatter
  - [Built-in Functions](#built-in-functions) -- All 15 functions
  - [Growth Functions](#growth-functions) -- Compound, linear, and depreciation
  - [Rates and `over`](#rates) -- Rate literals and accumulation
  - [Napkin Math](#napkin-math) -- Quick estimates with `as napkin`
  - [Date Arithmetic](#date-arithmetic) -- Date literals and duration math
  - [Capacity Planning](#capacity-planning) -- `at...per` syntax
  - [Multiplier Suffixes](#multiplier-suffixes) -- K, M, B shortcuts
  - [Percentages](#percentages) -- Percentage calculations
- [Tips](#tips) -- Organize with markdown, preview pane, getting help
- [Troubleshooting](#troubleshooting) -- Common errors and fixes

---

## Editor Shortcuts {#editor-shortcuts}

The CalcMark editor provides keyboard shortcuts for common actions. Press **F1** for full help inside the editor.

### File {#shortcuts-file}

| Shortcut | Action |
|----------|--------|
| Ctrl+N | New empty document |
| Ctrl+S | Save document |
| Ctrl+O | Open file |
| Ctrl+E | Export to format |
| Ctrl+Q | Quit editor |

### Edit {#shortcuts-edit}

| Shortcut | Action |
|----------|--------|
| Ctrl+Z | Undo last change |
| Ctrl+Y | Redo last change |
| Ctrl+K | Delete current line |
| Ctrl+F | Add YAML frontmatter |

### View {#shortcuts-view}

| Shortcut | Action |
|----------|--------|
| Ctrl+P | Cycle preview mode |

### Navigation {#shortcuts-navigation}

| Shortcut | Action |
|----------|--------|
| Opt+Left / Opt+B | Move to previous word |
| Opt+Right / Opt+F | Move to next word |
| Ctrl+Home | Jump to document start |
| Ctrl+End | Jump to document end |
| Ctrl+D | Scroll down half page |
| Ctrl+U | Scroll up half page |

## Locale Formatting {#locale-formatting}

CalcMark can format output numbers using locale-specific decimal and thousands separators. This affects how results are displayed in all output modes (editor, REPL, eval, convert).

### Setting Your Locale

**Per-command** with the `--locale` flag:

```bash
cm eval --locale=de-DE budget.cm
cm convert doc.cm --to=html --locale=fr-FR
```

**Permanently** in your config file:

```toml
# ~/.config/calcmark/config.toml
locale = "de-DE"
```

The `--locale` flag overrides the config file for that invocation.

### Example: Same Document, Different Locales

Given a CalcMark document:

```cm
price = $1500
pi = 3.14159
users = 1500000
weight = 50.5 kg
```

| Value | en-US (default) | de-DE | fr-FR |
|-------|-----------------|-------|-------|
| `price` | `$1,500.00` | `$1.500,00` | `$1 500,00` |
| `pi` | `3.14159` | `3,14159` | `3,14159` |
| `users` | `1.5M` | `1,5M` | `1,5M` |
| `weight` | `50.5 kg` | `50,5 kg` | `50,5 kg` |

### What Stays the Same

Regardless of locale, these never change:

- **Input syntax** -- always use `.` for decimals and `,` or `_` for thousands in your source
- **K/M/B/T suffixes** -- always English letters
- **Currency symbols** -- always prefix position (`$`, `€`, etc.)
- **CalcMark format** -- `cm convert --to=cm` is always locale-independent

### JSON Output

When exporting to JSON, each result includes a locale-formatted display value and structured type information:

```bash
cm convert budget.cm --to=json --locale=de-DE
```

```json
{
  "source": "price = $1500",
  "value": "$1.500,00",
  "type": "currency",
  "numeric_value": 1500,
  "unit": "USD",
  "variable": "price"
}
```

Use `type` for dispatch, `numeric_value` + `unit` for computation, and `value` for display. The `type`, `numeric_value`, and `unit` fields are always locale-independent. See [Configuration: JSON Output](/docs/configuration/#json-raw-value) for details.

## Exporting Results {#exporting-results}

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
cm eval --locale=de-DE budget.cm  # German number formatting
```

All export formats respect the `--locale` flag (or config setting) except `cm` format, which stays locale-independent.

## Sharing with GitHub Gist {#sharing-gist}

Share CalcMark documents as GitHub Gists directly from the editor. This feature requires the [GitHub CLI (`gh`)](https://cli.github.com) to be installed and authenticated.

### Prerequisites

Install and authenticate the GitHub CLI:

```bash
# Install
brew install gh          # macOS
sudo apt install gh      # Debian/Ubuntu

# Authenticate
gh auth login
```

### Share To Gist

Open the command menu (**Ctrl+H** or **F1**), select **Share To Gist**, then choose visibility (public or secret) and add an optional description. Press **Enter** to share. The Gist URL is copied to your clipboard.

If you are not authenticated, CalcMark will launch `gh auth login` interactively and retry after you sign in.

### Open From Gist

Open the command menu, select **Open From Gist**, then paste a Gist URL or ID. CalcMark fetches the Gist content and loads it into the editor. If the Gist contains multiple files, `.cm` files are preferred.

> **Note:** Sharing with GitHub Gist is not available in the browser (WASM) build.

## Language Features {#language-features}

### Supported Units {#units}

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

### Unit Conversion {#unit-conversion}

Convert between compatible units using `in` or `as`:

```cm
distance = 5 miles
distance_km = distance in km

temp_c = 20 celsius
temp_f = temp_c in fahrenheit

file_size = 1.5 GB
file_size_mb = file_size in MB
```

### Currency Conversion {#currency-conversion}

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

### Global Variables {#global-variables}

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

### Built-in Functions {#built-in-functions}

{{< feature-table category="function" >}}

See the [Language Reference](/docs/language-reference/) for the complete list of functions, including [natural language syntax](/docs/language-reference/#natural-language-syntax) forms.

### Growth Functions {#growth-functions}

Model compound growth, linear growth, and depreciation over time:

```cm
# Compound growth
compound($1000, 5%, 10)                          -> $1628.89
compound $1000 by 5% over 10 years               (NL form)

# Financial compounding
compound($1000, 5%, 10 years, compounded monthly) -> $1647.01

# Linear growth
grow($500, $100, 36)                              -> $4100.00
grow 100 by 20 over 5 months                      (NL form)

# Depreciation with salvage floor
depreciate($50000, 15%, 5)                        -> $22185.27
depreciate $50000 by 15% over 5 years to $5000
```

See the [Language Reference: Growth Functions](/docs/language-reference/#growth) for the full argument reference.

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

### Date Arithmetic {#date-arithmetic}

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

### Multiplier Suffixes {#multiplier-suffixes}

Use K, M, B for large numbers:

```cm
users = 10M
revenue = $5B
requests = 100K
```

### Percentages {#percentages}

Percentages work naturally in calculations:

```cm
price = $100
discount = 20%
sale_price = price * (1 - discount)

tax_rate = 8.25%
tax = price * tax_rate
```

## Tips {#tips}

### Organize with Markdown {#tip-markdown}

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

### Use the Preview Pane {#tip-preview}

Press **Ctrl+P** in the editor to toggle the preview pane, which shows evaluated results alongside your source.

### Get Help on Functions {#tip-help}

Run `cm help functions` to see all available functions with descriptions and usage patterns. Run `cm help constants` for unit constants.

## Troubleshooting {#troubleshooting}

### "Undefined variable" {#error-undefined}

Variables must be defined before use. Check that:
1. The variable is spelled correctly
2. It's defined on an earlier line
3. No typos in the name

### "Incompatible units" {#error-incompatible}

You can't add meters to kilograms. Check that operations make physical sense.

### "Parse error" {#error-parse}

The line isn't valid CalcMark syntax. Common issues:
- Missing operator between values
- Unclosed parentheses
- Invalid characters
