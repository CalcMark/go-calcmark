# Building and Testing the cm CLI

## Quick Build

```bash
cd /Users/bitsbyme/projects/go-calcmark
go build -o cm ./cmd/calcmark
```

This creates the `cm` executable in the current directory.

## Usage

```bash
# Evaluate expressions from stdin
echo "today + 7 days" | ./cm eval

# Evaluate from a file
./cm eval examples/dates_and_units.cm

# Interactive REPL
./cm repl

# Show version
./cm version
```

## Installation (Optional)

```bash
# Install to $GOPATH/bin
go install ./cmd/calcmark

# Then use as 'calcmark'
echo "5 + 10" | calcmark eval
```

## Current Status

**Working:**
- ✅ Dates: `today + 7 days`
- ✅ Numbers: `5 + 10`
- ✅ Currency: `$100`

**Not Yet Working:**
- ❌ Quantities: `5 kg + 10 lb` (parser issue)

The interpreter code is complete, just needs final parser debugging.
