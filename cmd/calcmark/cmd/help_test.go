package cmd

import (
	"bytes"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
)

// TestHelpFunctionsOutput verifies that help functions shows all function names.
func TestHelpFunctionsOutput(t *testing.T) {
	output := captureStdout(t, func() {
		helpFunctionsCmd.Run(helpFunctionsCmd, []string{})
	})

	// Verify header is present
	if !strings.Contains(output, "CalcMark Functions") {
		t.Error("missing 'CalcMark Functions' header")
	}

	// Verify every registered function name appears in the output
	for _, fn := range interpreter.GetAllFunctions() {
		if !strings.Contains(output, fn.Name) {
			t.Errorf("function %q not found in output", fn.Name)
		}
	}

	// Verify every category from the registry appears in the output
	for _, cat := range interpreter.GetCategoryOrder() {
		if !strings.Contains(output, cat) {
			t.Errorf("category %q not found in output", cat)
		}
	}

	// Verify synonym is shown for avg
	if !strings.Contains(output, "average") {
		t.Error("synonym 'average' not shown for avg function")
	}
}

// TestHelpConstantsOutput verifies that help constants shows unit information.
func TestHelpConstantsOutput(t *testing.T) {
	output := captureStdout(t, func() {
		helpConstantsCmd.Run(helpConstantsCmd, []string{})
	})

	// Verify header is present
	if !strings.Contains(output, "CalcMark Unit Constants") {
		t.Error("missing 'CalcMark Unit Constants' header")
	}

	// Verify some key units are present
	expectedUnits := []string{
		"meter", "kilogram", "liter", "celsius",
	}

	for _, unit := range expectedUnits {
		if !strings.Contains(output, unit) {
			t.Errorf("unit %q not found in output", unit)
		}
	}

	// Verify quantity categories are present
	expectedQuantities := []string{"Length", "Mass", "Volume", "Temperature"}
	for _, qty := range expectedQuantities {
		if !strings.Contains(output, qty) {
			t.Errorf("quantity %q not found in output", qty)
		}
	}
}

// TestHelpOutputPipeable verifies that output contains no ANSI escape codes.
func TestHelpOutputPipeable(t *testing.T) {
	// ANSI escape code regex
	ansiPattern := regexp.MustCompile(`\x1b\[[0-9;]*m`)

	t.Run("functions", func(t *testing.T) {
		output := captureStdout(t, func() {
			helpFunctionsCmd.Run(helpFunctionsCmd, []string{})
		})

		if ansiPattern.MatchString(output) {
			t.Error("functions output contains ANSI escape codes")
		}
	})

	t.Run("constants", func(t *testing.T) {
		output := captureStdout(t, func() {
			helpConstantsCmd.Run(helpConstantsCmd, []string{})
		})

		if ansiPattern.MatchString(output) {
			t.Error("constants output contains ANSI escape codes")
		}
	})
}

// TestRootHelpShowsLocaleFlag verifies that Cobra's auto-generated help
// includes the --locale flag (the old custom renderer hardcoded flags and
// missed it).
func TestRootHelpShowsLocaleFlag(t *testing.T) {
	output := captureStdout(t, func() {
		rootCmd.SetArgs([]string{"--help"})
		_ = rootCmd.Execute()
	})

	if !strings.Contains(output, "--locale") {
		t.Error("root help output missing --locale flag")
	}
	if !strings.Contains(output, "--color-mode") {
		t.Error("root help output missing --color-mode flag")
	}
}

// captureStdout captures stdout during execution of fn.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	return buf.String()
}
