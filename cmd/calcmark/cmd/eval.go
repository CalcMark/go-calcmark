package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/CalcMark/go-calcmark/v2/format"
	implDoc "github.com/CalcMark/go-calcmark/v2/impl/document"
	"github.com/CalcMark/go-calcmark/v2/spec/document"
	"github.com/spf13/cobra"
)

var (
	evalVerbose bool
	evalFormat  string
)

var evalCmd = &cobra.Command{
	Use:   "eval [file.cm]",
	Short: "Evaluate CalcMark and print the result",
	Long: `Evaluate a CalcMark file or stdin and print the result.

Examples:
  cm eval calc.cm                Evaluate file and print result
  cm eval -v calc.cm             Evaluate with verbose output (all values)
  cm eval --format json calc.cm  Evaluate and output as JSON
  echo "x = 10" | cm eval       Evaluate from stdin`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runEval(args)
	},
}

func init() {
	evalCmd.Flags().BoolVarP(&evalVerbose, "verbose", "v", false, "Show all intermediate values")
	evalCmd.Flags().StringVar(&evalFormat, "format", "text", "Output format: text, json, html, md, cm")
	rootCmd.AddCommand(evalCmd)
}

// runEval handles the eval subcommand - evaluates and prints the result
func runEval(args []string) error {
	var input string
	var hasFile bool

	if len(args) > 0 {
		filename := args[0]
		hasFile = true

		// Read from file
		if err := validateReadFilePath(filename); err != nil {
			return returnError("invalid file: %w", err)
		}

		bytes, err := os.ReadFile(filename)
		if err != nil {
			return returnError("read file: %w", err)
		}
		if err := validateFileContent(bytes); err != nil {
			return returnError("invalid file: %w", err)
		}
		input = string(bytes)
	}

	if !hasFile {
		// Read from stdin with same 1MB size limit as file input.
		// Without a limit, a piped input could exhaust memory.
		const maxStdinSize = 1*1024*1024 + 1 // 1MB + 1 byte to detect overflow
		bytes, err := io.ReadAll(io.LimitReader(os.Stdin, maxStdinSize))
		if err != nil {
			return returnError("read stdin: %w", err)
		}
		if len(bytes) >= maxStdinSize {
			return returnErrorMsg("stdin input too large (max 1MB)")
		}
		if err := validateStdinContent(bytes); err != nil {
			return returnError("invalid input: %w", err)
		}
		input = string(bytes)

		if strings.TrimSpace(input) == "" {
			return returnErrorMsg("no input — pipe a document or pass a filename (run 'cm eval --help' for details)")
		}
	}

	// Parse and evaluate
	doc, err := document.NewDocument(input)
	if err != nil {
		return returnError("parse error: %w", err)
	}

	eval := implDoc.NewEvaluator()
	eval.SetDisplayFormatter(localeFormatter())
	evalErr := eval.Evaluate(doc)
	if evalErr != nil && !errors.Is(evalErr, implDoc.ErrPartialEvaluation) {
		// Fatal evaluation error (e.g., frontmatter failure) — no output
		return returnError("evaluation error: %w", evalErr)
	}

	// Use the selected formatter for eval output (defaults to "text")
	formatter := format.GetFormatter(evalFormat, "")

	opts := format.Options{
		Verbose:          evalVerbose,
		DisplayFormatter: eval.GetDisplayFormatter(),
	}

	if err := formatter.Format(os.Stdout, doc, opts); err != nil {
		return returnError("format error: %w", err)
	}

	// If partial evaluation, output was formatted with partial results + diagnostics.
	// Return error for non-zero exit code, but don't write JSON error envelope
	// (the formatted output already contains diagnostic information).
	if errors.Is(evalErr, implDoc.ErrPartialEvaluation) {
		return fmt.Errorf("evaluation error: %w", evalErr)
	}

	return nil
}

// returnError wraps an error and, when JSON format is active, writes a JSON
// error envelope to stdout so that agents and pipelines always receive valid
// JSON. The error is still returned for Cobra to print on stderr and set
// exit code 1.
func returnError(wrapFmt string, inner error) error {
	wrapped := fmt.Errorf(wrapFmt, inner)
	if evalFormat == "json" {
		writeJSONError(os.Stdout, wrapped)
	}
	return wrapped
}

// returnErrorMsg is like returnError but for errors without a wrapped inner error.
func returnErrorMsg(msg string) error {
	err := fmt.Errorf("%s", msg)
	if evalFormat == "json" {
		writeJSONError(os.Stdout, err)
	}
	return err
}

// jsonErrorEnvelope wraps an error for JSON output on stdout.
type jsonErrorEnvelope struct {
	Error jsonError `json:"error"`
}

// jsonError represents a structured error in JSON output.
type jsonError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Line    int    `json:"line,omitempty"`
	Code    string `json:"code,omitempty"`
}

// lineCodeMessageRe matches "line N: code: message" patterns in error messages.
var lineCodeMessageRe = regexp.MustCompile(`^line (\d+): (\w+): (.+)$`)

// parseEvalError extracts structured fields from an error message.
// Error messages follow patterns like:
//
//	"evaluation error: line 2: variable_redefinition: cannot reassign 'x'"
//	"parse error: <details>"
//	"frontmatter error: <details>"
func parseEvalError(err error) jsonError {
	msg := err.Error()

	// Extract error type from prefix
	errType := "unknown_error"
	remainder := msg
	for _, prefix := range []string{"evaluation error: ", "parse error: ", "frontmatter error: ", "format error: "} {
		if strings.HasPrefix(msg, prefix) {
			// "evaluation error: " → "evaluation_error"
			errType = strings.ReplaceAll(strings.TrimSuffix(prefix, ": "), " ", "_")
			remainder = msg[len(prefix):]
			break
		}
	}

	je := jsonError{
		Type:    errType,
		Message: remainder,
	}

	// Try to extract "line N: code: message" from remainder
	if m := lineCodeMessageRe.FindStringSubmatch(remainder); m != nil {
		je.Line, _ = strconv.Atoi(m[1])
		je.Code = m[2]
		je.Message = m[3]
	}

	return je
}

// writeJSONError writes a JSON error envelope to w.
func writeJSONError(w io.Writer, err error) {
	envelope := jsonErrorEnvelope{Error: parseEvalError(err)}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(envelope)
}
