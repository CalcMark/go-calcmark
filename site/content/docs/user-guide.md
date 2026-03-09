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
  - [@Directive References](#directive-references) -- `@scale` and `@globals.name` in expressions
  - [Scale and Convert](#scale-convert) -- Document-wide scaling and unit system conversion
  - [Built-in Functions](#built-in-functions) -- All 15 functions
  - [Function Reference](#function-reference) -- Detailed examples for every function
  - [Natural Language Syntax](#natural-language-syntax) -- NL forms reference table
  - [Network Functions](#network-functions) -- RTT, throughput, transfer time, downtime
  - [Storage Functions](#storage-functions) -- Read, seek, compress
  - [Growth Functions](#growth-functions) -- Compound, linear, and depreciation
  - [Rates and `over`](#rates) -- Rate literals, accumulation, and conversion
  - [Date Arithmetic](#date-arithmetic) -- Date literals, durations, and `from`
  - [Napkin Math](#napkin-math) -- Quick estimates with `as napkin`
  - [Precise Display](#precise-display) -- Full precision with `as precise`
  - [Capacity Planning](#capacity-planning) -- `at...per` syntax
  - [Multiplier Suffixes](#multiplier-suffixes) -- K, M, B shortcuts
  - [Percentages](#percentages) -- Percentage calculations
- [Worked Examples](#worked-examples) -- Complete calculation scenarios
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
| Ctrl+A | Select all |
| Ctrl+C | Copy selection |
| Ctrl+X | Cut selection |
| Ctrl+V | Paste |
| Ctrl+Backspace | Delete word |

### View {#shortcuts-view}

| Shortcut | Action |
|----------|--------|
| Ctrl+P | Cycle preview mode |
| Ctrl+H / F1 | Open command menu |

### Navigation {#shortcuts-navigation}

| Shortcut | Action |
|----------|--------|
| Opt+Left / Opt+B | Move to previous word |
| Opt+Right / Opt+F | Move to next word |
| Ctrl+Home | Jump to document start |
| Ctrl+End | Jump to document end |
| Ctrl+Left / Ctrl+Right | Move word left/right |
| Ctrl+D | Scroll down half page |
| Ctrl+U | Scroll up half page |
| Shift+Arrow | Extend selection |

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

```calcmark
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

Use `cm eval` for quick evaluation, or pipe directly into `cm`:

```bash
cm eval budget.cm            # Print final results
cm eval -v budget.cm         # Show all intermediate values
echo "1 + 2" | cm           # Evaluate piped input (auto-detects pipe)
echo "1 + 2" | cm --format json  # JSON output for scripting/agents
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

- **Area**: cm², m², km², ha, in², ft², yd², mi², acre
- **DataSize**: byte, KB, MB, GB, TB, PB (and binary: KiB, MiB, GiB, TiB)
- **Energy**: J, kJ, cal, kcal, kWh
- **Length**: m, cm, mm, km, in, ft, yd, mi, nmi (nautical mile)
- **Mass**: mg, g, kg, metric ton (t), oz, lb
- **Power**: W, kW, MW, hp
- **Speed**: m/s, km/h, mph, knot
- **Temperature**: C, F, K
- **Volume**: mL, L, tsp, tbsp, cup, pt, qt, gal

Time units (`second`, `minute`, `hour`, `day`, `week`, `month`, `year`) are used in durations and rates but are not a conversion category.

Run `cm help constants` for the complete list with aliases and descriptions.

### Unit Conversion {#unit-conversion}

Convert between compatible units using `in` or `as`:

```calcmark
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
  USD_EUR: 0.92
  EUR_GBP: 0.86
---
```

```calcmark
price_usd = $100
price_eur = price_usd in EUR

salary = 50000 EUR
salary_gbp = salary in GBP
```

Exchange rates use the format `FROM_TO: rate` where 1 unit of FROM equals `rate` units of TO.

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

Reference globals in expressions with the `@globals.` prefix:

```text
net_price = @globals.base_price * (1 - @globals.tax_rate)
project_end = @globals.base_date + @globals.sprint_length * 6
monthly_transfer = @globals.bandwidth over 30 days
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

### @Directive References {#directive-references}

Use `@scale` and `@globals.name` to reference frontmatter values in expressions:

```text
per_unit = total_cost / @scale
tax = income * @globals.tax_rate
```

`@scale` resolves to the numeric scale factor from frontmatter. `@globals.name` resolves to the typed value of a named global. These are the only supported directives — `@exchange`, `@convert_to`, and other names produce errors.

`@globals` without a field name (e.g., bare `@globals`) is a parser error. Use `@globals.name` to reference a specific global.

### Scale and Convert {#scale-convert}

Use `scale` and `convert_to` in the frontmatter to transform results across an entire document. This is useful for recipes, engineering estimates, and any document where you want to change units or multiply quantities globally.

#### Scaling

Scaling is explicit — you specify which categories to scale in `unit_categories`. A bare `scale: 2` sets the factor (accessible via `@scale`) but scales nothing on its own.

```yaml
---
scale:
  factor: 2
  unit_categories: [Mass, Volume]
---
```

```text
flour = 1.5 cups       → 3 cups (doubled — Volume)
eggs = 3 eggs           → 3 eggs (unchanged — custom unit, not in categories)
oven = 350 fahrenheit   → 350 fahrenheit (unchanged — not in categories)
price = $5.00           → $5.00 (unchanged — not in categories)
```

`flour` displays as `3 cups` because Volume is in the category list. `eggs` is unchanged because `eggs` is a custom unit — add `Custom` to `unit_categories` to scale it. Temperature and currency are unaffected unless you add `Temperature` or `Currency` to `unit_categories`.

In the editor, scaled values show a `*` suffix in the results pane. The context footer shows `@scale = N` when you're on a scaled line.

#### Converting

Convert all quantities to a target measurement system:

```yaml
---
convert_to: si
---
```

```text
distance = 5 miles       -> 8.05 km
weight = 10 pounds       -> 4.54 kg
temp = 72 fahrenheit     -> 22.2222 celsius
```

`distance` displays as `8.05 km`, `weight` as `4.54 kg`, and `temp` as `22.2222 celsius`. If you use the `in` keyword explicitly (e.g., `distance in feet`), that conversion takes priority over `convert_to`.

#### Both Together

When both are present, scale applies first, then convert:

```yaml
---
scale:
  factor: 2
  unit_categories: [Mass]
convert_to: si
---
```

```text
butter = 4 ounces       -> 227 g (doubled, then converted)
```

`butter` is first doubled to `8 ounces` (Mass is in `unit_categories`), then converted to `227 g`.

#### Filtering by Category

Limit scale or convert to specific unit categories:

```yaml
---
scale:
  factor: 4
  unit_categories: [Mass, Volume]
convert_to:
  system: si
  unit_categories: [Length]
---
```

Valid categories: `Area`, `Currency`, `Custom`, `DataSize`, `Energy`, `Length`, `Mass`, `Number`, `Power`, `Speed`, `Temperature`, `Volume`. `Custom` matches units not in the standard library (e.g., `bananas`, `eggs`). These are derived from the unit definitions — run `cm help frontmatter` for the current list. Use `unit_categories: [All]` to scale every category.

See the [Recipe Scaling](/docs/examples/recipe-scaling/) example for a complete walkthrough, and the [Language Reference — Frontmatter](/docs/language-reference/#frontmatter) for the full specification.

### Built-in Functions {#built-in-functions}

{{< feature-table category="function" >}}

See the [Function Reference](#function-reference) below for detailed examples of every function, including natural language forms.

### Function Reference {#function-reference}

Detailed examples for every built-in function, showing both function-call and natural language syntax.

#### `avg` / `average of` {#avg-examples}

```calcmark
avg(10, 20, 30)                    -> 20
avg(1, 2, 3, 4, 5)                -> 3
average of 100, 200, 300           -> 200  (NL form)
avg($100, $200, $300)              -> $200.00  (preserves currency)
```

#### `sqrt` / `square root of` {#sqrt-examples}

```calcmark
sqrt(16)                           -> 4
sqrt(2)                            -> 1.4142...
square root of 144                 -> 12  (NL form)
sqrt($100)                         -> $10.00  (preserves currency)
```

#### `accumulate` / `over` {#accumulate-examples}

```calcmark
accumulate(100 MB/s, 1 hour)       -> 360000 MB
accumulate($75/hour, 8 hours)      -> $600
100 MB/s over 1 day                -> ~8.64 TB  (keyword form)
$120000/year over 1 month          -> $10000
```

#### `convert_rate` {#convert-rate-examples}

```calcmark
convert_rate(1000 req/s, minute)   -> 60000 req/min
convert_rate($120000/year, month)  -> $10000/month
```

#### `capacity` / `at...per` {#capacity-examples}

```calcmark
capacity(10 TB, 2 TB, disk)              -> 5 disks
capacity(10000 req/s, 500 req/s, server) -> 20 servers
10 TB at 2 TB per disk                   -> 5 disks  (NL form)
10000 req/s at 450 req/s per server with 20% buffer -> 27 servers
```

#### `downtime` {#downtime-examples}

```calcmark
downtime(0.999, year)              -> 8.76 hours
downtime(0.999, month)             -> 43.2 minutes
downtime(0.9999, month)            -> 4.32 minutes
```

#### `rtt` {#rtt-examples}

```calcmark
rtt(local)                         -> 0.5 ms
rtt(regional)                      -> 10 ms
rtt(continental)                   -> 50 ms
rtt(global)                        -> 150 ms
```

#### `throughput` {#throughput-examples}

```calcmark
throughput(gigabit)                 -> 125 MB/s
throughput(ten_gig)                -> 1250 MB/s
throughput(wifi)                   -> 12.5 MB/s
throughput(five_g)                 -> 50 MB/s
```

#### `transfer_time` / `transfer...across` {#transfer-time-examples}

```calcmark
transfer_time(1 GB, regional, gigabit)   -> ~8 seconds
transfer_time(500 MB, continental, gigabit)
transfer 1 GB across regional gigabit    -> (NL form)
transfer 100 MB across local ten_gig

data = 500 KB
transfer data across continental ten_gig -> (variable reference)
```

#### `read` / `read...from` {#read-examples}

```calcmark
read(100 MB, ssd)                  -> ~0.18 seconds
read(1 GB, nvme)                   -> ~0.29 seconds
read 100 MB from ssd               -> (NL form)
read 10 GB from pcie_ssd

data = 5 MB
read data from nvme                -> (variable reference)
```

#### `seek` {#seek-examples}

```calcmark
seek(hdd)                          -> 10 ms
seek(ssd)                          -> 0.1 ms
seek(nvme)                         -> 0.01 ms
db_query_hdd = seek(hdd) + read(5 MB, hdd)
```

#### `compress` / `compress...using` {#compress-examples}

```calcmark
compress(1 GB, gzip)               -> ~333 MB
compress(500 MB, lz4)              -> ~250 MB
compress(2 GB, zstd)               -> ~571 MB
compress 1 GB using gzip            -> (NL form)
compress 500 MB using lz4

data = 1 GB
compress data using zstd            -> (variable reference)
```

#### `compound` / `compound...by...over` {#compound-examples}

```calcmark
compound($1000, 5%, 10)                              -> $1628.89
compound(500 customers, 20%, 12)                     -> 4458.05 customers
compound($1000, 5%, 10 years, monthly)               -> $1647.01
compound($1000, 5%, 10 years, quarterly)             -> $1643.62
compound $1000 by 5% over 10 years                   -> $1628.89  (NL form)
compound $1000 by 5% monthly over 10 years
compound $1000 by 5% per month over 12 months
```

#### `grow` / `grow...by...over` {#grow-examples}

```calcmark
grow(100, 20, 5)                   -> 200
grow($500, $100, 36)               -> $4100.00
grow 100 by 20 over 5 months       -> 200  (NL form)
```

#### `depreciate` / `depreciate...by...over` {#depreciate-examples}

```calcmark
depreciate($50000, 15%, 5)                -> $22185.27
depreciate($50000, 15%, 20, $5000)        -> $5000.00  (salvage floor)
depreciate $50000 by 15% over 5 years     -> $22185.27  (NL form)
depreciate $50000 by 15% over 5 years to $5000
```

#### Unit Handling in Functions

**Same units are preserved:**

```calcmark
avg($100, $200, $300) -> $200.00
sqrt($100) -> $10.00
```

**Mixed units are dropped:**

```calcmark
avg($100, €200) -> 150  (no units)
average of $50, €100, £150 -> 100  (no units)
```

### Natural Language Syntax {#natural-language-syntax}

CalcMark supports natural language forms for many functions. These are equivalent to the function-call syntax. Arguments can be literal values (`100 MB`) or variable references (`data`).

#### Function Aliases

| Natural Language | Equivalent | Example |
|-----------------|------------|---------|
| `average of X, Y, Z` | `avg(X, Y, Z)` | `average of 10, 20, 30` |
| `square root of X` | `sqrt(X)` | `square root of 144` |
| `read X from Y` | `read(X, Y)` | `read 100 MB from ssd` |
| `compress X using Y` | `compress(X, Y)` | `compress 1 GB using gzip` |
| `transfer X across Y Z` | `transfer_time(X, Y, Z)` | `transfer 1 GB across regional gigabit` |
| `compound X by Y% over Z` | `compound(X, Y%, Z)` | `compound $1000 by 5% over 10 years` |
| `compound X by Y% monthly over Z` | `compound(X, Y%, Z, monthly)` | `compound $1000 by 5% monthly over 10 years` |
| `compound X by Y% per P over Z` | `compound(X, Y%, Z, P)` | `compound $1000 by 5% per month over 12 months` |
| `compound X by Y% compounded F over Z` | `compound(X, Y%, Z, compounded F)` | `compound $1000 by 12% compounded monthly over 10 years` |
| `grow X by Y over Z` | `grow(X, Y, Z)` | `grow 100 by 20 over 5 months` |
| `depreciate X by Y% over Z` | `depreciate(X, Y%, Z)` | `depreciate $50000 by 15% over 5 years` |
| `depreciate X by Y% over Z to W` | `depreciate(X, Y%, Z, W)` | `depreciate $50000 by 15% over 5 years to $5000` |

#### Capacity Planning Syntax {#capacity-syntax}

The `at...per` syntax is a natural language form for the `capacity()` function:

```text
demand at capacity per unit
demand at capacity per unit with N% buffer
```

Examples:

```calcmark
10 TB at 2 TB per disk                         -> 5 disks
10000 req/s at 450 req/s per server             -> 23 servers
10000 req/s at 450 req/s per server with 20%    -> 27 servers
100 apples at 30 per crate                      -> 4 crates
```

The slash syntax also works: `10 TB at 2 TB/disk`.

#### Rate Accumulation with `over` {#over}

The `over` keyword accumulates a rate over a time duration:

```text
rate over duration
```

This is equivalent to `accumulate(rate, duration)`:

```calcmark
100 MB/s over 1 day         -> total data transferred
$75/hour over 8 hours       -> daily earnings
1000 req/s over 1 hour      -> total requests
```

#### Rate Conversion with `per`

The `per` keyword in a rate context creates a rate literal:

```calcmark
1000 requests per second    -> 1000 req/s
$50 per hour                -> $50/hour
```

### Network Functions {#network-functions}

#### Round-Trip Time

```calcmark
rtt(local)          -> 0.5 ms   (same datacenter)
rtt(regional)       -> 10 ms    (same region)
rtt(continental)    -> 50 ms    (cross-continent)
rtt(global)         -> 150 ms   (worldwide)
```

#### Throughput

```calcmark
throughput(gigabit)      -> 125 MB/s
throughput(ten_gig)      -> 1.22 GB/s
throughput(hundred_gig)  -> 12500 MB/s
throughput(wifi)         -> 12.5 MB/s
throughput(four_g)       -> 2.5 MB/s
throughput(five_g)       -> 50 MB/s
```

#### Transfer Time

Calculate data transfer time across a network:

```calcmark
transfer_time(1 GB, regional, gigabit)
transfer 1 GB across regional gigabit       (NL form)
```

#### Downtime from Availability

```calcmark
downtime(99.9%, year)     -> ~8.76 hours
downtime(99.99%, month)   -> ~4.32 minutes
```

### Storage Functions {#storage-functions}

#### Read Time

```calcmark
read(1 GB, ssd)       read from SATA SSD (~550 MB/s)
read(1 GB, nvme)      read from NVMe SSD (~3.5 GB/s)
read(1 GB, pcie_ssd)  read from PCIe Gen4 SSD (~7 GB/s)
read(1 GB, hdd)       read from 7200 RPM HDD (~150 MB/s)

read 100 MB from ssd  (NL form)
```

#### Seek Latency

```calcmark
seek(ssd)       -> 0.1 ms
seek(nvme)      -> 0.01 ms
seek(pcie_ssd)  -> 0.01 ms
seek(hdd)       -> 10 ms
```

#### Compression

```calcmark
compress(1 GB, gzip)     -> 333 MB   (3:1 ratio)
compress(1 GB, lz4)      -> 512 MB   (2:1 ratio)
compress(1 GB, zstd)     -> 293 MB   (3.5:1 ratio)
compress(1 GB, bzip2)    -> 250 MB   (4:1 ratio)
compress(1 GB, snappy)   -> 400 MB   (2.5:1 ratio)
compress(1 GB, none)     -> 1 GB     (1:1, no compression)

compress 1 GB using gzip (NL form)
```

### Growth Functions {#growth-functions}

#### Compound Growth

The 4th argument controls which formula `compound()` uses. This matters -- `monthly` and `month` give different answers:

**Without a frequency modifier** -- simple compound growth:

```text
A = P × (1 + r)^n
```

```calcmark
compound($1000, 5%, 10)                              -> $1628.89
compound $1000 by 5% over 10 years                   -> $1628.89
```

The rate is applied once per period for `n` periods. The 3rd argument is a dimensionless count -- `10` means "10 times", not "10 years". Writing `10 years` is equivalent; the unit is ignored in simple mode.

**With a frequency adverb** (`monthly`, `quarterly`, `weekly`, `daily`, `yearly`) -- financial compounding:

```text
A = P × (1 + r/n)^(n × t)
```

where `r` is the annual rate, `n` is the compounding frequency per year, and `t` is time in years.

```calcmark
compound($1000, 5%, 10, monthly)                     -> $1647.01
compound($1000, 5%, 10, quarterly)                    -> $1643.62
compound($1000, 5%, 24 months, monthly)              -> $1103.43
compound $1000 by 5% monthly over 10 years            -> $1647.01
```

The 5% annual rate is split into 12 monthly applications (5%/12 per month), compounded 120 times over 10 years. More frequent compounding yields higher returns: monthly ($1,647) > quarterly ($1,643) > yearly ($1,629).

**Arguments:**

| # | Name | Description |
|---|------|-------------|
| 1 | principal | Starting amount (number, currency, or quantity) |
| 2 | rate | Growth rate (treated as annual rate when a frequency modifier is present) |
| 3 | periods | Period count or duration. With a frequency modifier, a plain number is treated as years; a duration like `24 months` is converted to years |
| 4 | modifier | Optional: `monthly`, `quarterly`, `daily`, `weekly`, `yearly`. Switches to the financial compounding formula |

{{< callout type="info" >}}
**`month` vs `monthly`**: The noun form `month` is a label -- `compound($1000, 5%, 12, month)` means "5% applied 12 times" (simple growth). The adverb `monthly` switches formula -- `compound($1000, 5%, 10, monthly)` means "5% annual rate, compounded 12x/year for 10 years." The 3rd argument becomes a duration in years, and the rate becomes an annual rate.
{{< /callout >}}

#### Linear Growth

Calculate linear (additive) growth over time:

```calcmark
grow($500, $100, 36)               -> $4100.00
grow(100, 20, 5)                   -> 200

grow 100 by 20 over 5 months       (NL form)
```

**Arguments:**

| # | Name | Description |
|---|------|-------------|
| 1 | initial | Starting amount |
| 2 | increment | Amount added each period |
| 3 | periods | Number of periods (number or duration) |

#### Depreciation

Calculate declining-balance depreciation with optional salvage floor:

```calcmark
depreciate($50000, 15%, 5)                -> $22185.27
depreciate($50000, 15%, 20, $5000)        -> $5000.00  (salvage floor)

depreciate $50000 by 15% over 5 years     (NL form)
depreciate $50000 by 15% over 5 years to $5000
```

**Arguments:**

| # | Name | Description |
|---|------|-------------|
| 1 | value | Starting value |
| 2 | rate | Depreciation rate as percentage |
| 3 | periods | Number of periods (number or duration) |
| 4 | salvage | Optional: minimum floor value |

### Rates and the `over` keyword {#rates}

#### Rate Literals

Create rates using the slash syntax:

```calcmark
net_bandwidth = 100 MB/s
salary = $120000/year
load = 1000 req/s
```

#### Rate Accumulation with `over`

Use `over` to calculate the total from a rate over time:

```calcmark
link_speed = 100 MB/s
daily_transfer = link_speed over 1 day

hourly_rate = $75/hour
daily_earnings = hourly_rate over 8 hours
```

#### Rate Conversion

Convert rates to different time units using `convert_rate()`:

```calcmark
convert_rate(1000 req/s, minute)    -> 60000 req/min
convert_rate($120000/year, month)   -> $10000/month
```

#### Rate Arithmetic Widening

When you multiply or divide **by** a rate (rate on the right), CalcMark drops the time denominator and uses the rate's amount. This is called **widening** — the rate widens into a plain quantity.

When the rate is on the **left** side, it stays a rate. This lets you scale rates naturally with `rate * number`.

```text
posts_rate = 2 posts/week
scaled = posts_rate * 3           -> 6 posts/week  (rate * number → rate)
total  = 3 * posts_rate           -> 6 posts       (number * rate → quantity)
```

The rule is simple: **left operand wins**. If you start with a rate, you get a rate. If you start with a number or quantity and multiply by a rate, you get a number or quantity.

See the [Language Reference: Rate Arithmetic Widening](/docs/language-reference/#rate-arithmetic-widening) for the full type dispatch table.

### Date Arithmetic {#date-arithmetic}

#### Date Literals

```calcmark
project_start = Jan 15 2025
christmas = Dec 25 2025
now = today
```

CalcMark recognizes `today`, `tomorrow`, and `yesterday` as date keywords.

#### Duration Arithmetic

```calcmark
project_start = Jan 15 2025
duration = 12 weeks
project_end = project_start + duration

deadline = Jun 1 2025
launch = deadline - 2 weeks
```

#### The `from` Keyword

```calcmark
7 days from Jan 1 2025   -> Wednesday, January 8, 2025
2 weeks from today       -> (today + 14 days)
```

### Napkin Math {#napkin-math}

The `as napkin` modifier rounds results to 2 significant figures and normalizes units. It adds a `~` prefix to signal the result is an approximation.

**Syntax:** `expression as napkin`

**Works with:** Number, Quantity, Currency, Duration, Rate

```calcmark
432000 MB as napkin                 -> ~420 GB
100 MB/s over 30 days as napkin    -> ~248 TB
$1234567 as napkin                  -> ~$1.2M
```

This is useful for quick back-of-the-envelope calculations where exact precision is not needed.

### Precise Display {#precise-display}

The `as precise` modifier is the opposite of `as napkin`. It shows full float precision, skipping all display rounding. This is useful when you need exact values from unit conversions.

**Syntax:** `expression as precise`

```calcmark
10 meters as feet                  -> 32.8 feet
10 meters as feet as precise       -> 32.808399 feet
```

Explicit unit conversions are rounded by default for readability. Use `as precise` when you need the exact value.

### Capacity Planning {#capacity-planning}

Use the `at...per` syntax to calculate how many units you need:

```calcmark
disks = 10 TB at 2 TB per disk
servers = 10000 req/s at 450 req/s per server
servers_buffered = 10000 req/s at 450 req/s per server with 20% buffer
```

### Multiplier Suffixes {#multiplier-suffixes}

Use K, M, B for large numbers:

```calcmark
users = 10M
revenue = $5B
requests = 100K
```

### Percentages {#percentages}

Percentages work naturally in calculations:

```calcmark
price = $100
discount = 20%
sale_price = price * (1 - discount)

tax_rate = 8.25%
tax = price * tax_rate
```

---

## Worked Examples {#worked-examples}

### Basic Calculations

```calcmark
# Simple Math

5 + 3
10 - 2
4 * 5
20 / 4
2 ^ 3
10 % 3
```

### Variables

```calcmark
# Budget

salary = $5000
bonus = $500
expenses = $3000

net = salary + bonus - expenses
```

### Comparisons

```calcmark
# Checks

salary = $50000
threshold = $60000

is_high_earner = salary > threshold
needs_raise = salary < $40000
meets_target = salary >= $50000
```

### Complex Expressions

```calcmark
# Mortgage

principal = $200000
rate = 0.04
years = 30
months = years * 12

monthly_rate = rate / 12
payment = principal * (monthly_rate * (1 + monthly_rate) ^ months) / ((1 + monthly_rate) ^ months - 1)
```

### System Sizing

```calcmark
# Server Capacity Planning

peak_load = 50000 req/s
server_capacity = 2000 req/s
servers = peak_load at server_capacity per server with 25% buffer

# Storage
daily_data = 100 MB/s over 1 day
monthly_storage = daily_data * 30
disks = monthly_storage at 2 TB per disk
```

### Mixed Markdown

```calcmark
# My Monthly Budget

I earn a salary and get bonuses.

## Income

monthly_salary = $5000
annual_bonus = $3000
monthly_bonus = annual_bonus / 12

Total monthly income:
total_income = monthly_salary + monthly_bonus

## Expenses

- Rent: $1500
- Food: $600
- Utilities: $200

rent = $1500
food = $600
utilities = $200
total_expenses = rent + food + utilities

## Summary

Monthly surplus:
surplus = total_income - total_expenses

Can I save 20%?
savings_goal = total_income * 0.20
can_save = surplus >= savings_goal
```

---

## Tips {#tips}

### Organize with Markdown {#tip-markdown}

Use headers and prose to structure your thinking:

```calcmark
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

Run `cm help functions` to see all available functions with descriptions and usage patterns. Run `cm help constants` for unit constants, or `cm help frontmatter` for frontmatter directives.

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
