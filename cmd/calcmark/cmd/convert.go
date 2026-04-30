package cmd

import (
	"fmt"
	"os"

	"github.com/CalcMark/go-calcmark"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/config"
	"github.com/CalcMark/go-calcmark/format"
	"github.com/spf13/cobra"
)

var (
	convertFormat       string
	convertOutput       string
	convertTemplate     string
	convertShowTemplate bool
	convertEmbedded     bool
)

var convertCmd = &cobra.Command{
	Use:   "convert <file.cm>",
	Short: "Convert CalcMark to another format",
	Long: `Convert a CalcMark file to HTML, Markdown, JSON, text, or CalcMark format.

Use --show-template to print the default HTML template. This is useful as a
starting point for custom templates passed via --template.

Use --embedded to process a standard Markdown file that contains cm/calcmark
fenced code blocks. Each block is evaluated independently and replaced with
its formatted output. All other content passes through unchanged.

Examples:
  cm convert doc.cm --to=html              Convert to HTML (stdout)
  cm convert doc.cm --to=md -o doc.md      Convert to Markdown file
  cm convert doc.cm --to=json              Convert to JSON
  cm convert doc.cm --to=html -T tpl.html  Use custom HTML template
  cm convert --show-template               Print default HTML template
  cm convert report.md --embedded          Process embedded CalcMark blocks
  cm convert report.md --embedded --to=html  Embedded CalcMark to HTML`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if convertShowTemplate {
			fmt.Println("{{/* === CalcMark shared partials (parsed automatically) ===")
			fmt.Println("   Available: cm-content, cm-frontmatter, cm-blocks, cm-calc-block, cm-text-block")
			fmt.Println("   Call via {{template \"cm-content\" .}} or individual partials.")
			fmt.Println("   Data model: .Style (template.CSS), .Frontmatter, .Blocks []TemplateBlock")
			fmt.Println("*/}}")
			fmt.Println()
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
	convertCmd.Flags().BoolVar(&convertEmbedded, "embedded", false, "Process a Markdown file with embedded cm/calcmark blocks")
	rootCmd.AddCommand(convertCmd)
}

// runConvert handles the convert subcommand for both cm and embedded modes.
func runConvert(filename string) error {
	// Determine mode and validate file path.
	mode := calcmark.CM
	if convertEmbedded {
		mode = calcmark.Embedded
		if err := validateReadFilePathEmbedded(filename); err != nil {
			return fmt.Errorf("invalid file: %w", err)
		}
	} else {
		if err := validateReadFilePath(filename); err != nil {
			return fmt.Errorf("invalid file: %w", err)
		}
	}

	// --to is required for cm mode; for embedded it defaults to "md".
	formatName := convertFormat
	if formatName == "" {
		if convertEmbedded {
			formatName = "md"
		} else {
			return fmt.Errorf("missing --to flag — usage: cm convert <file.cm> --to=<format> (valid: html, md, json, text, cm)")
		}
	}

	// Validate format name.
	validFormats := map[string]bool{
		"html": true, "md": true, "json": true, "text": true, "cm": true,
	}
	if !validFormats[formatName] {
		return fmt.Errorf("unknown format: %s (valid: html, md, json, text, cm)", formatName)
	}

	// Embedded mode only supports md and html.
	if convertEmbedded && formatName != "md" && formatName != "html" {
		return fmt.Errorf("--embedded only supports md and html output; --to=%s is not valid with --embedded", formatName)
	}

	// Validate template option.
	if convertTemplate != "" && formatName != "html" {
		return fmt.Errorf("--template is only valid with --to=html")
	}

	// Read input file.
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	// Security: Reject binary/non-text content.
	if err := validateFileContent(content); err != nil {
		return fmt.Errorf("invalid file: %w", err)
	}

	// Resolve template content.
	templateContent, err := resolveTemplate(formatName)
	if err != nil {
		return err
	}

	// Build conversion options.
	cfg := config.Get()
	opts := calcmark.Options{
		Mode:             mode,
		Format:           formatName,
		Template:         templateContent,
		Locale:           cfg.Locale,
		DateFormat:       cfg.Formatter.DateFormat,
		PeriodDateFormat: cfg.Formatter.PeriodDateFormat,
	}

	// Convert.
	result, convErr := calcmark.Convert(string(content), opts)

	// Write output first — even if there were partial errors (e.g., some embedded
	// blocks failed), the result contains inline error blockquotes and should be written.
	if result != "" {
		if err := writeOutput(result); err != nil {
			return err
		}
	}

	return convErr
}

// resolveTemplate determines the template content for the conversion.
// For HTML format: uses custom template if provided, otherwise the default full template.
// For embedded mode: no default template (goldmark output is self-contained).
// For other formats: no template.
func resolveTemplate(formatName string) (string, error) {
	if formatName != "html" {
		return "", nil
	}
	if convertTemplate != "" {
		content, err := os.ReadFile(convertTemplate)
		if err != nil {
			return "", fmt.Errorf("read template: %w", err)
		}
		return string(content), nil
	}
	// Embedded mode: no default template — goldmark HTML is self-contained.
	if convertEmbedded {
		return "", nil
	}
	// CM mode: CLI uses the full default template (not fragment) for backwards compatibility.
	return format.DefaultHTMLTemplate(), nil
}

// writeOutput writes the conversion result to the output destination.
func writeOutput(result string) error {
	if convertOutput != "" {
		if err := os.WriteFile(convertOutput, []byte(result), 0o644); err != nil {
			return fmt.Errorf("write output file: %w", err)
		}
		return nil
	}
	fmt.Print(result)
	return nil
}
