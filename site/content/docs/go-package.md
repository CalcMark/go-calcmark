---
title: "Go Package"
summary: "Use go-calcmark as a library to evaluate CalcMark expressions and documents in your Go applications."
weight: 37
---

go-calcmark is a standard Go module. Import it and evaluate CalcMark expressions in a few lines of code.

Full API documentation is on [pkg.go.dev](https://pkg.go.dev/github.com/CalcMark/go-calcmark).

```bash
go get github.com/CalcMark/go-calcmark
```

## Quick Start: Convert in One Call

The simplest integration — pass CalcMark input, get formatted output:

```go
result, err := calcmark.Convert("price = 100 USD * 1.08", calcmark.Options{
    Format: "html",
})
// result contains styled HTML with the evaluated expression
```

`Convert` handles the full pipeline: parse, evaluate, and format. It supports both CalcMark documents and Markdown with embedded CalcMark blocks:

```go
// Embedded mode: Markdown with ```cm fenced code blocks
result, err := calcmark.Convert(markdownInput, calcmark.Options{
    Mode:   calcmark.Embedded,
    Format: "html",
})
```

Available formats: `"html"`, `"md"`, `"text"`, `"json"`, `"cm"`.

## Example: How CalcMark Lark Works

[CalcMark Lark](https://github.com/CalcMark/calcmark-lark) is a web-based playground built entirely on `calcmark.Convert`. The core is just one function call:

```go
html, err := calcmark.Convert(userInput, calcmark.Options{
    Format:   "html",
    Template: customTemplate, // optional — wraps output in a full HTML page
})
if err != nil {
    // partial results are still useful — display what we have
    log.Printf("convert: %v", err)
}
w.Write([]byte(html))
```

For custom styling, pass a Go `html/template` string via the `Template` option. Custom templates automatically have access to shared partials (`cm-content`, `cm-frontmatter`, `cm-blocks`) — no need to duplicate rendering logic. Override `--cm-*` CSS variables for theming.

See [Custom HTML Templates]({{< ref "docs/custom-templates" >}}) for the full guide, including CSS variable reference, dark mode, and real-world examples like Lark.

## Example: A Minimal REPL

A working CalcMark REPL in under 30 lines. Save this as `repl.go`:

```go
package main

import (
    "bufio"
    "fmt"
    "os"

    calcmark "github.com/CalcMark/go-calcmark"
)

func main() {
    session := calcmark.NewSession()
    scanner := bufio.NewScanner(os.Stdin)

    fmt.Println("CalcMark REPL — type expressions, Ctrl+D to quit")
    for fmt.Print("» "); scanner.Scan(); fmt.Print("» ") {
        result, err := session.Eval(scanner.Text())
        if err != nil {
            fmt.Println("error:", err)
            continue
        }
        if result.Value != nil {
            fmt.Println(result.Value)
        }
    }
}
```

Then run it:

```bash
go mod init myrepl
go mod tidy
go run repl.go
```

Variables persist across lines:

```
» price = $49.99
$49.99
» qty = 3
3
» total = price * qty
$149.97
» tax = 8.5% of total
$12.75
» grand_total = total + tax
$162.72
```

## Evaluate a Single Expression

For one-off evaluations without session state:

```go
result, err := calcmark.Eval("price = 100 USD * 1.08")
if err != nil {
    log.Fatal(err)
}
fmt.Println(result.Value) // $108.00
```

`Eval` creates a fresh session for each call. The returned `Result` contains:

- `Value` — the final computed value (a `types.Type`)
- `AllValues` — all values when the input has multiple lines
- `Diagnostics` — any warnings or hints from semantic analysis

## Sessions: Persistent Variables

Use a `Session` when you need variables to persist across multiple evaluations — like a REPL or an interactive editor:

```go
session := calcmark.NewSession()

session.Eval("base = $85000")
session.Eval("bonus_pct = 15%")
result, _ := session.Eval("total = base + base * bonus_pct")

fmt.Println(result.Value) // $97,750.00

// Look up any variable by name
val, ok := session.GetVariable("total")
if ok {
    fmt.Println(val) // $97,750.00
}
```

Call `session.Reset()` to clear all variables and start fresh.

## Evaluate a Full Document

For complete CalcMark documents — with Markdown, frontmatter, and multiple calc blocks — use the document-level API:

```go
import (
    "github.com/CalcMark/go-calcmark/spec/document"
    implDoc "github.com/CalcMark/go-calcmark/impl/document"
)

source := `---
exchange:
  USD_EUR: 0.92
---

# Budget

revenue = 500K USD
costs = revenue * 0.6
profit = revenue - costs
profit_eur = profit in EUR
`

doc, err := document.NewDocument(source)
if err != nil {
    log.Fatal(err)
}

eval := implDoc.NewEvaluator()
if err := eval.Evaluate(doc); err != nil {
    log.Fatal(err)
}

// Access all variables through the environment
env := eval.GetEnvironment()
profit, _ := env.Get("profit")
fmt.Println(profit) // $200,000.00
```

The document evaluator handles block ordering, dependency resolution, frontmatter directives (exchange rates, globals, scale, measurement), and variable interpolation in text blocks.

### Configuring the Evaluator

The evaluator accepts optional configuration before calling `Evaluate`:

```go
import "github.com/CalcMark/go-calcmark/format/display"

eval := implDoc.NewEvaluator()

// Set locale for number formatting (decimal/thousand separators)
cfg, _ := display.NewConfig("de-DE")
eval.SetDisplayFormatter(display.NewFormatter(cfg))

if err := eval.Evaluate(doc); err != nil {
    log.Fatal(err)
}

// After Evaluate(), the formatter includes any measurement conventions
// from frontmatter. Pass it to output formatters for consistent display.
df := eval.GetDisplayFormatter()
```

Measurement conventions from frontmatter are wired automatically during `Evaluate()`. Library consumers can also set them directly on the interpreter for lower-level control — see `interpreter.SetMeasurement()`.

### Formatting Output

The `format` package provides registry-based output formatters:

```go
import "github.com/CalcMark/go-calcmark/format"

formatter := format.GetFormatter("html", "")
opts := format.Options{
    Verbose:          true,
    DisplayFormatter: eval.GetDisplayFormatter(),
    Template:         customHTMLTemplate, // optional Go template string
}
if err := formatter.Format(os.Stdout, doc, opts); err != nil {
    log.Fatal(err)
}
```

Available formats: `"html"`, `"md"`, `"text"`, `"json"`, `"cm"`.

## The Type System

Every CalcMark value is a `types.Type`. The concrete types are in `github.com/CalcMark/go-calcmark/spec/types`:

| Type | Example | Go Type |
|------|---------|---------|
| `*Number` | `42`, `3.14` | `types.Number` |
| `*Currency` | `$100`, `85 EUR` | `types.Currency` |
| `*Quantity` | `5 kg`, `100 Mbps` | `types.Quantity` |
| `*Fraction` | `1/3`, `2/3 cup` | `types.Fraction` |
| `*Percentage` | `15%`, `20% of 500` | `types.Percentage` |
| `*Duration` | `3 hours`, `2 weeks` | `types.Duration` |
| `*Date` | `2026-03-11` | `types.Date` |
| `*Time` | `2:30 PM`, `14:30` | `types.Time` |
| `*Rate` | `100 MB/s`, `$50/hour` | `types.Rate` |
| `*Boolean` | `true`, `false` | `types.Boolean` |

All numeric values use `shopspring/decimal` for arbitrary-precision arithmetic — no floating-point surprises.

### Extracting Values

Use `types.ToDecimal` to get the numeric value from any numeric type:

```go
import "github.com/CalcMark/go-calcmark/spec/types"

result, _ := calcmark.Eval("weight = 5 kg")
d, err := types.ToDecimal(result.Value)
if err == nil {
    fmt.Println(d) // 5
}
```

For type-specific fields, use a type assertion:

```go
result, _ := calcmark.Eval("price = 100 USD")
if currency, ok := result.Value.(*types.Currency); ok {
    fmt.Println(currency.Value) // 100
    fmt.Println(currency.Code)  // USD
    fmt.Println(currency.Symbol) // $
}
```

## Frontmatter in Code

CalcMark frontmatter (`exchange`, `globals`, `scale`, `convert_to`, `measurement`) is normally written at the top of a `.cm` file. But you can use it anywhere — including `Session.Eval`, `calcmark.Eval`, and `calcmark.Convert`. Just pass the YAML frontmatter block as part of the input:

```go
session := calcmark.NewSession()

// Set up exchange rates via frontmatter
session.Eval(`---
exchange:
  USD_EUR: 0.92
  GBP_USD: 1.27
---`)

// Now currency conversion works
result, _ := session.Eval("price = 100 USD\nprice_eur = price in EUR")
fmt.Println(result.Value) // €92.00
```

This works identically with `calcmark.Convert` for document rendering:

```go
input := `---
exchange:
  USD_EUR: 0.92
globals:
  tax_rate: 0.08
---

revenue = 500K USD
costs = revenue * 0.6
tax = costs * @globals.tax_rate
`

html, _ := calcmark.Convert(input, calcmark.Options{Format: "html"})
```

For Markdown with embedded CalcMark blocks, frontmatter works the same way — it's stripped before goldmark processing and applied to all `cm` blocks:

```go
input := `---
exchange:
  USD_EUR: 0.92
---

# Invoice

` + "```cm" + `
subtotal = 500 USD
total_eur = subtotal in EUR
` + "```" + `
`

html, _ := calcmark.Convert(input, calcmark.Options{
    Mode:   calcmark.Embedded,
    Format: "html",
})
```

## The Environment

For programmatic control without frontmatter, use the `Environment` directly:

```go
import (
    "github.com/CalcMark/go-calcmark/impl/interpreter"
    "github.com/CalcMark/go-calcmark/spec/types"
    "github.com/shopspring/decimal"
)

env := interpreter.NewEnvironment()
env.Set("tax_rate", types.NewNumber(decimal.NewFromFloat(0.08)))
env.SetExchangeRate("USD", "EUR", decimal.NewFromFloat(0.92))

interp := interpreter.NewInterpreterWithEnv(env)
```

Key `Environment` methods:

| Method | Description |
|--------|-------------|
| `Set(name, value)` | Store a variable |
| `Get(name)` | Retrieve a variable (returns value, ok) |
| `Has(name)` | Check if a variable exists |
| `GetAllVariables()` | Get all variables as a map |
| `SetExchangeRate(from, to, rate)` | Set a currency conversion rate |
| `Clone()` | Shallow copy for isolation |

## Diagnostics

Both `Eval` and the document evaluator return diagnostics — structured messages with severity levels:

```go
result, _ := calcmark.Eval("x = unknown_var + 1")
for _, d := range result.Diagnostics {
    fmt.Printf("[%s] %s\n", d.Severity, d.Message)
}
```

Severity levels: `Error`, `Warning`, `Hint`.

## Package Structure

The codebase separates specification from implementation:

| Package | Purpose |
|---------|---------|
| `github.com/CalcMark/go-calcmark` | Top-level `Eval`, `Session`, and `Convert` API |
| `spec/types` | Value types (Number, Currency, Quantity, etc.) |
| `spec/document` | Document model (blocks, parsing) |
| `spec/units` | Canonical unit definitions and conversions |
| `spec/features` | Feature registry (functions, NL syntax) |
| `impl/interpreter` | Expression evaluator and Environment |
| `impl/document` | Document-level evaluator |

Dependencies flow one way: `impl` depends on `spec`, never the reverse.

## Further Reading

- [pkg.go.dev documentation](https://pkg.go.dev/github.com/CalcMark/go-calcmark) — full API reference with all exported types and methods
- [Agent & API Integration]({{< ref "docs/agent-integration" >}}) — using CalcMark via stdin/stdout pipes
- [Language Reference]({{< ref "docs/language-reference" >}}) — formal specification of the CalcMark language
- [Source code on GitHub](https://github.com/CalcMark/go-calcmark) — `spec/` for the language spec, `impl/` for the interpreter
