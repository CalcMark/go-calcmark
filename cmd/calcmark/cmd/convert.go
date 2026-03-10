package cmd

import (
	"fmt"
	"os"

	"github.com/CalcMark/go-calcmark/format"
	implDoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/spec/document"
	"github.com/spf13/cobra"
)

var (
	convertFormat       string
	convertOutput       string
	convertTemplate     string
	convertShowTemplate bool
)

var convertCmd = &cobra.Command{
	Use:   "convert <file.cm>",
	Short: "Convert CalcMark to another format",
	Long: `Convert a CalcMark file to HTML, Markdown, JSON, text, or CalcMark format.

Use --show-template to print the default HTML template. This is useful as a
starting point for custom templates passed via --template.

Examples:
  cm convert doc.cm --to=html              Convert to HTML (stdout)
  cm convert doc.cm --to=md -o doc.md      Convert to Markdown file
  cm convert doc.cm --to=json              Convert to JSON
  cm convert doc.cm --to=html -T tpl.html  Use custom HTML template
  cm convert --show-template               Print default HTML template`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if convertShowTemplate {
			fmt.Print(format.DefaultHTMLTemplate())
			return nil
		}
		if len(args) == 0 {
			return fmt.Errorf("missing file — usage: cm convert <file.cm> --to=<format>")
		}
		return runConvert(args[0])
	},
}

func init() {
	convertCmd.Flags().StringVarP(&convertFormat, "to", "t", "", "Output format: html, md, json, text, cm (required)")
	convertCmd.Flags().StringVarP(&convertOutput, "output", "o", "", "Write to file instead of stdout")
	convertCmd.Flags().StringVarP(&convertTemplate, "template", "T", "", "Custom Go template (html only)")
	convertCmd.Flags().BoolVar(&convertShowTemplate, "show-template", false, "Print the default HTML template and exit")
	rootCmd.AddCommand(convertCmd)
}

// runConvert handles the convert subcommand
func runConvert(filename string) error {
	// --to is required for conversion
	if convertFormat == "" {
		return fmt.Errorf("missing --to flag — usage: cm convert <file.cm> --to=<format> (valid: html, md, json, text, cm)")
	}

	// Validate file path
	if err := validateReadFilePath(filename); err != nil {
		return fmt.Errorf("invalid file: %w", err)
	}

	// Read input file
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	// Security: Reject binary/non-text content before parsing
	if err := validateFileContent(content); err != nil {
		return fmt.Errorf("invalid file: %w", err)
	}

	// Parse document
	doc, err := document.NewDocument(string(content))
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	// Evaluate with display formatter for {{var}} interpolation
	eval := implDoc.NewEvaluator()
	eval.SetDisplayFormatter(localeFormatter())
	if err := eval.Evaluate(doc); err != nil {
		return fmt.Errorf("evaluation error: %w", err)
	}

	// Validate template option
	if convertTemplate != "" && convertFormat != "html" {
		return fmt.Errorf("--template is only valid with --to=html")
	}

	// Load custom template if provided
	var templateContent string
	if convertTemplate != "" {
		tplContent, err := os.ReadFile(convertTemplate)
		if err != nil {
			return fmt.Errorf("read template: %w", err)
		}
		templateContent = string(tplContent)
	}

	// Validate format name
	validFormats := map[string]bool{
		"html": true, "md": true, "json": true, "text": true, "cm": true,
	}
	if !validFormats[convertFormat] {
		return fmt.Errorf("unknown format: %s (valid: html, md, json, text, cm)", convertFormat)
	}

	// Get formatter
	formatter := format.GetFormatter(convertFormat, convertOutput)

	// Determine output destination
	var out *os.File
	if convertOutput != "" {
		out, err = os.Create(convertOutput)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer out.Close()
	} else {
		out = os.Stdout
	}

	// Format and write
	opts := format.Options{
		Verbose:          true,
		Template:         templateContent,
		DisplayFormatter: localeFormatter(),
	}
	if err := formatter.Format(out, doc, opts); err != nil {
		return fmt.Errorf("format error: %w", err)
	}

	return nil
}
