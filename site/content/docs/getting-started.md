---
title: "Getting Started"
summary: "Install CalcMark and run your first calculation."
weight: 10
---

## Installation

**macOS/Linux (Homebrew):**

```bash
brew install calcmark/tap/calcmark
```

**Download binary:**

| Platform | Download |
|----------|----------|
| macOS (Apple Silicon) | [calcmark\_darwin\_arm64.tar.gz](https://github.com/CalcMark/go-calcmark/releases/latest) |
| macOS (Intel) | [calcmark\_darwin\_amd64.tar.gz](https://github.com/CalcMark/go-calcmark/releases/latest) |
| Linux (x64) | [calcmark\_linux\_amd64.tar.gz](https://github.com/CalcMark/go-calcmark/releases/latest) |
| Linux (arm64) | [calcmark\_linux\_arm64.tar.gz](https://github.com/CalcMark/go-calcmark/releases/latest) |
| Windows (x64) | [calcmark\_windows\_amd64.zip](https://github.com/CalcMark/go-calcmark/releases/latest) |

After downloading, extract and move `cm` to a directory in your PATH. On macOS, you may need to run `xattr -d com.apple.quarantine ./cm` before first use.

## Quick Start

### Interactive Editor

Open a worked example directly — no files needed. Click {{< repo-file path="testdata/examples/household-budget.cm" show_file="false" >}} to copy the `cm remote` command to your clipboard. Paste it into your terminal to launch CalcMark with the [Household Budget]({{< ref "docs/examples/household-budget" >}}) example loaded in the editor. Press **Ctrl+H** to open the help menu and see all keyboard shortcuts. Press **Ctrl+Q** to exit.

> You'll see the {{< repo-file path="testdata/examples/household-budget.cm" show_file="false" >}} icon throughout the docs and [Examples]({{< ref "docs/examples" >}}) — it always copies a `cm remote` command you can paste into your terminal.

<img src="/images/tui-screenshot.png" alt="CalcMark TUI editor showing a budget calculation with live results" width="700">

You can also start from scratch or open a local file:

```bash
cm                    # New document
cm budget.cm          # Open existing file
```

Browse more examples in the [Examples]({{< ref "docs/examples" >}}) section.

### Evaluate a File

Process a local file and see results:

```bash
cm eval budget.cm
```

### Pipe Expressions

Quick calculations from the command line:

```bash
echo "price = 100 USD" | cm
echo "24 celsius in fahrenheit" | cm
echo "500 gram in oz" | cm
echo "1 atmosphere in psi" | cm
echo "1 + 1" | cm --format json    # JSON output for scripting
```

> **Note:** When stdin is piped, `cm` automatically evaluates and prints results instead of opening the editor. Use `--format json` for structured output.

## The Editor

The editor includes **autosuggest** that helps you discover functions and units as you type. Type at least 2 characters and suggestions appear automatically:

<img src="/images/feature-autocomplete.gif" alt="CalcMark autocomplete popup showing function and NL suggestions" width="600">

Every function has both a traditional `fn(args)` form and a natural language form. The autocomplete shows both -- select the function name for `compound(...)` syntax, or the NL row for `compound $1000 by 5% over 10 years`:

<img src="/images/feature-growth-autocomplete.gif" alt="CalcMark autocomplete showing growth function NL completion" width="600">

## Core Concepts

### Variables Flow Downward

Variables must be defined before use. Later lines can reference earlier ones:

```calcmark
base_salary = $85000
bonus_pct = 15%
bonus = base_salary * bonus_pct
total_comp = base_salary + bonus
```

### Units Are First-Class

CalcMark understands physical units and currencies:

```calcmark
marathon_pace = 12.06 km/hour
race_time = 3 hours + 30 minutes
distance_covered = marathon_pace over race_time

price_usd = 100 USD
price_eur = 85 EUR
```

### Markdown is Ignored

Write prose freely. Only lines that parse as calculations are evaluated:

```calcmark
# Project Budget

We need to account for both development and infrastructure costs.

dev_team = 5
monthly_salary = $12000
dev_cost = dev_team * monthly_salary * 6

Infrastructure will be roughly 20% of dev costs.

infra_pct = 20%
infra_cost = dev_cost * infra_pct
```

## Optional: GitHub Gist Integration

To share and open CalcMark documents via GitHub Gist, install the [GitHub CLI](https://cli.github.com):

```bash
brew install gh
gh auth login
```

See [Sharing with GitHub Gist](/docs/user-guide/#sharing-gist) in the User Guide for details.

## Next Steps

**Learn by domain** — pick a [Guide](/guides/) that matches your use case:
- [System Sizing](/guides/system-sizing/) — capacity planning, bandwidth, SLA budgets
- [Business Planning](/guides/business-planning/) — P&L, budgets, financial modeling
- [Recipe Scaling](/guides/recipe-scaling/) — fractions, measurement conventions, scaling
- [Unit Conversion](/guides/unit-conversion/) — measurement systems, ambiguous units

**Go deeper:**
- [User Guide](/docs/user-guide/) — task-oriented reference for every feature
- [The Editor](/docs/editor/) — preview modes, shortcuts, autocomplete
- [Examples](/docs/examples/) — complete worked documents
- [Language Reference](/docs/language-reference/) — formal specification

**Integrate:**
- [Agent & API Integration](/docs/agent-integration/) — use CalcMark from code or AI agents
- [Go Package](/docs/go-package/) — embed in your Go application
- {{< lark "" "CalcMark Lark" >}} — try in your browser, no install needed
