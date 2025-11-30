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
	convertFormat   string
	convertOutput   string
	convertTemplate string
)

var convertCmd = &cobra.Command{
	Use:   "convert <file.cm>",
	Short: "Convert CalcMark to another format",
	Long: `Convert a CalcMark file to HTML, Markdown, JSON, text, or CalcMark format.

Examples:
  cm convert doc.cm --to=html              Convert to HTML (stdout)
  cm convert doc.cm --to=md -o doc.md      Convert to Markdown file
  cm convert doc.cm --to=json              Convert to JSON
  cm convert doc.cm --to=html -T tpl.html  Use custom HTML template`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConvert(args[0])
	},
}

func init() {
	convertCmd.Flags().StringVarP(&convertFormat, "to", "t", "", "Output format: html, md, json, text, cm (required)")
	convertCmd.Flags().StringVarP(&convertOutput, "output", "o", "", "Write to file instead of stdout")
	convertCmd.Flags().StringVarP(&convertTemplate, "template", "T", "", "Custom Go template (html only)")
	_ = convertCmd.MarkFlagRequired("to")
	rootCmd.AddCommand(convertCmd)
}

// runConvert handles the convert subcommand
func runConvert(filename string) error {
	// Validate file path
	if err := validateFilePath(filename); err != nil {
		return fmt.Errorf("invalid file: %w", err)
	}

	// Read input file
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	// Parse document
	doc, err := document.NewDocument(string(content))
	if err != nil {
		return fmt.Errorf("parse error: %w", err)
	}

	// Evaluate
	eval := implDoc.NewEvaluator()
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
		Verbose:  true,
		Template: templateContent,
	}
	if err := formatter.Format(out, doc, opts); err != nil {
		return fmt.Errorf("format error: %w", err)
	}

	return nil
}
