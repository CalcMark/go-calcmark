# CalcMark Output Formatter Design (Revised)

## Overview
CalcMark's formatter system centers around the text formatter as the primary interface for both REPL and CLI evaluation. All formatters implement Go's `io.Writer` pattern for consistent output handling.

## Core Architecture

### Formatter Interface
```go
type Formatter interface {
    Format(w io.Writer, doc *Document, opts FormatOptions) error
    Extensions() []string      // File extensions this formatter handles
}

type FormatOptions struct {
    Verbose      bool          // Show calculation steps, types, units
    IncludeErrors bool         // Include error details
    Template     string        // For template-based formatters
    Writer       io.Writer     // Already provided in Format(), but can override
}
```

### Document Model
```go
type Document struct {
    Segments []Segment
    Writer   io.Writer  // Default output (os.Stdout, REPL buffer, etc.)
}

type Segment struct {
    Source   string      
    Output   *Value      
    Error    error       
    Metadata *Metadata   // Unit conversions, intermediate steps
}
```

## Text Formatter as Default

The text formatter is the default for all interactive output:

```go
// REPL evaluation
repl> x = 1 + 2
3

repl> bandwidth = 100 Mbps * 8 users
800 Mbps

// CLI evaluation  
$ echo 'x = 1 + 2' | ./cm eval
3

$ echo 'bandwidth = 100 Mbps * 8 users' | ./cm eval
800 Mbps
```

With `--verbose`:
```go
repl> bandwidth = 100 Mbps * 8 users
100 Mbps * 8 users
  = 100 × 10^6 bits/second * 8       # unit expansion
  = 800 × 10^6 bits/second           # calculation
  = 800 Mbps                         # unit normalization
```

## Formatter Registry

```go
var formatters = map[string]Formatter{
    "text": &TextFormatter{},    // Default
    "cm":   &CalcMarkFormatter{},
    "json": &JSONFormatter{},
    "html": &HTMLFormatter{},
    "md":   &MarkdownFormatter{},
}

func GetFormatter(format string, filename string) Formatter {
    // Explicit format takes precedence
    if format != "" {
        return formatters[format]
    }
    
    // Auto-detect from extension
    ext := strings.ToLower(filepath.Ext(filename))
    for name, fmt := range formatters {
        for _, fmtExt := range fmt.Extensions() {
            if ext == fmtExt {
                return fmt
            }
        }
    }
    
    // Default to text formatter
    return formatters["text"]
}
```

## Evaluation Flow

Both REPL and CLI share the same evaluation and output pipeline:

```go
// Shared evaluation function
func (doc *Document) Evaluate(input string, opts FormatOptions) error {
    segment := doc.ParseAndEval(input)
    doc.Segments = append(doc.Segments, segment)
    
    // Always use text formatter for immediate output
    formatter := formatters["text"]
    return formatter.Format(doc.Writer, 
        &Document{Segments: []Segment{segment}}, opts)
}

// REPL loop
func (r *REPL) Run() {
    scanner := bufio.NewScanner(r.Input)
    opts := FormatOptions{
        Verbose: r.Verbose,
        Writer: r.Output,  // Usually os.Stdout
    }
    
    for scanner.Scan() {
        line := scanner.Text()
        if strings.HasPrefix(line, "/") {
            r.handleCommand(line)
        } else {
            r.document.Evaluate(line, opts)
        }
    }
}

// CLI eval subcommand
func evalCommand(input io.Reader, output io.Writer, verbose bool) error {
    doc := &Document{Writer: output}
    opts := FormatOptions{Verbose: verbose}
    
    scanner := bufio.NewScanner(input)
    for scanner.Scan() {
        doc.Evaluate(scanner.Text(), opts)
    }
    return scanner.Err()
}
```

## Text Formatter Implementation

```go
type TextFormatter struct{}

func (f *TextFormatter) Format(w io.Writer, doc *Document, opts FormatOptions) error {
    for _, seg := range doc.Segments {
        if seg.Error != nil {
            fmt.Fprintf(w, "Error: %v\n", seg.Error)
            if opts.Verbose && seg.Error.Details != nil {
                fmt.Fprintf(w, "  %v\n", seg.Error.Details)
            }
            continue
        }
        
        if seg.Output == nil {
            continue  // Skip pure text segments
        }
        
        if opts.Verbose && seg.Metadata != nil {
            // Show calculation steps
            fmt.Fprintln(w, seg.Source)
            for _, step := range seg.Metadata.Steps {
                fmt.Fprintf(w, "  %s\n", step)
            }
        }
        
        // Always show final output
        fmt.Fprintln(w, seg.Output.String())
    }
    return nil
}
```

## Save/Output Commands

The save command uses the same formatters but writes to files:

```go
// REPL /save command
func (r *REPL) handleSave(args []string) error {
    filename := args[0]
    formatter := GetFormatter("", filename)  // Auto-detect
    
    file, err := os.Create(filename)
    if err != nil {
        return err
    }
    defer file.Close()
    
    return formatter.Format(file, r.document, r.formatOpts)
}

// CLI output subcommand
func outputCommand(inputFile, outputFile string, format string, opts FormatOptions) error {
    doc := LoadAndEvaluate(inputFile)
    
    var w io.Writer = os.Stdout
    if outputFile != "" {
        file, err := os.Create(outputFile)
        if err != nil {
            return err
        }
        defer file.Close()
        w = file
    }
    
    formatter := GetFormatter(format, outputFile)
    return formatter.Format(w, doc, opts)
}
```

## Usage Examples

```bash
# Interactive REPL (text formatter by default)
$ ./cm
cm> x = 100 GB / 8 Mbps
100 hours

# With verbose output
$ ./cm --verbose
cm> x = 100 GB / 8 Mbps
100 GB / 8 Mbps
  = 100 × 10^9 bytes / 8 × 10^6 bits/second
  = 100 × 10^9 × 8 bits / 8 × 10^6 bits/second
  = 10^5 seconds
  = 100,000 seconds
  = 27.78 hours
100 hours

# CLI evaluation (same text formatter)
$ echo 'x = 100 GB / 8 Mbps' | ./cm eval
100 hours

# Save from REPL
cm> /save results.json     # Uses JSON formatter
cm> /save output.txt        # Uses text formatter

# CLI output subcommand
$ ./cm output calc.cm -o results.html   # Auto-detect HTML
$ ./cm output calc.cm --format=json     # Explicit JSON to stdout
```

## Key Design Points

1. **Text formatter is primary**: It's the default for all interactive output
2. **io.Writer everywhere**: Consistent interface for REPL, CLI, files
3. **Shared evaluation path**: REPL and CLI use identical code
4. **Verbose mode**: Shows intermediate steps without changing output format
5. **Auto-detection**: File extensions determine format when not explicit. A strategy patterns ensures that the right tools are used for the right output. For example, spittnig out Markdown to the terminal could use https://github.com/charmbracelet/glamour/tree/master while output to HTML might use Goldmark https://github.com/yuin/goldmark.
6. **Structured content**: The Session and Document models provide a way to access information for structured output. For example, markdown blocks can have a ToHtml method that is used in an HTML template, and calculation.ToHtml will decorate lexer tokens with simple CSS classes for formatting output.

This design makes the text formatter the central piece of the user experience while keeping all formatters pluggable and consistent through the io.Writer interface.
