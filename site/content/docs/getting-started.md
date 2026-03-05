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

> **macOS Gatekeeper:** If macOS says it cannot verify the binary, run:
> `xattr -d com.apple.quarantine $(which cm)`

**Download binary:**

| Platform | Download |
|----------|----------|
| macOS (Apple Silicon) | [calcmark\_darwin\_arm64.tar.gz](https://github.com/CalcMark/go-calcmark/releases/latest) |
| macOS (Intel) | [calcmark\_darwin\_amd64.tar.gz](https://github.com/CalcMark/go-calcmark/releases/latest) |
| Linux (x64) | [calcmark\_linux\_amd64.tar.gz](https://github.com/CalcMark/go-calcmark/releases/latest) |
| Linux (arm64) | [calcmark\_linux\_arm64.tar.gz](https://github.com/CalcMark/go-calcmark/releases/latest) |
| Windows (x64) | [calcmark\_windows\_amd64.zip](https://github.com/CalcMark/go-calcmark/releases/latest) |

After downloading, extract and move `cm` to a directory in your PATH.

## Quick Start

### Interactive Editor

Open the CalcMark editor:

```bash
cm                    # New document
cm budget.cm          # Open existing file
```

<img src="/images/tui-screenshot.png" alt="CalcMark TUI editor showing a budget calculation with live results" width="700">

### Evaluate a File

Process a file and see results:

```bash
cm eval testdata/examples/system-sizing.cm
```

### Pipe Expressions

Quick calculations from the command line:

```bash
echo "price = 100 USD" | cm
echo "24 celsius in fahrenheit" | cm
echo "500 gram in oz" | cm
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

```cm
base_salary = $85000
bonus_pct = 15%
bonus = base_salary * bonus_pct
total_comp = base_salary + bonus
```

### Units Are First-Class

CalcMark understands physical units and currencies:

```cm
distance = 42.195 km
time = 3 hours + 30 minutes
pace = time / distance

price_usd = 100 USD
price_eur = 85 EUR
```

### Markdown is Ignored

Write prose freely. Only lines that parse as calculations are evaluated:

```cm
# Project Budget

We need to account for both development and infrastructure costs.

dev_team = 5
monthly_salary = $12000
dev_cost = dev_team * monthly_salary * 6 months

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

- Read the full [User Guide]({{< ref "docs/user-guide" >}}) for all features
- Explore the [Examples]({{< ref "docs/examples" >}}) to see CalcMark in action
- Check the [Language Reference]({{< ref "docs/language-reference" >}}) for the formal specification
