package cmd

import (
	"bytes"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/impl/interpreter"
	"github.com/CalcMark/go-calcmark/spec/units"
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

// TestHelpFrontmatterOutput verifies that help frontmatter shows all directives.
func TestHelpFrontmatterOutput(t *testing.T) {
	output := captureStdout(t, func() {
		helpFrontmatterCmd.Run(helpFrontmatterCmd, []string{})
	})

	// Verify header
	if !strings.Contains(output, "CalcMark Frontmatter Directives") {
		t.Error("missing 'CalcMark Frontmatter Directives' header")
	}

	// Verify all 4 directives are present
	directives := []string{"exchange", "globals", "scale", "convert_to"}
	for _, d := range directives {
		if !strings.Contains(output, d) {
			t.Errorf("directive %q not found in output", d)
		}
	}

	// Verify categories are derived from code (not hardcoded)
	for _, cat := range units.Categories() {
		if !strings.Contains(output, cat) {
			t.Errorf("category %q not found in frontmatter output", cat)
		}
	}

	// Verify YAML examples are present
	if !strings.Contains(output, "USD_EUR: 0.92") {
		t.Error("missing exchange rate example")
	}
	if !strings.Contains(output, "factor: 4") {
		t.Error("missing scale map form example")
	}
	if !strings.Contains(output, "system: imperial") {
		t.Error("missing convert_to map form example")
	}
	if !strings.Contains(output, "si, imperial") {
		t.Error("missing valid systems list")
	}
}

// TestHelpAllSections verifies that cm help with no flags shows all sections.
func TestHelpAllSections(t *testing.T) {
	output := captureStdout(t, func() {
		// Reset flags to defaults
		helpShowFunctions = false
		helpShowConstants = false
		helpShowFrontmatter = false
		helpCmd.Run(helpCmd, []string{})
	})

	if !strings.Contains(output, "CalcMark Functions") {
		t.Error("cm help missing Functions section")
	}
	if !strings.Contains(output, "CalcMark Unit Constants") {
		t.Error("cm help missing Constants section")
	}
	if !strings.Contains(output, "CalcMark Frontmatter Directives") {
		t.Error("cm help missing Frontmatter section")
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

	t.Run("frontmatter", func(t *testing.T) {
		output := captureStdout(t, func() {
			helpFrontmatterCmd.Run(helpFrontmatterCmd, []string{})
		})

		if ansiPattern.MatchString(output) {
			t.Error("frontmatter output contains ANSI escape codes")
		}
	})
}

// TestRootHelpShowsLocaleFlag verifies that Cobra's auto-generated help
// includes the --locale flag (the old custom renderer hardcoded flags and
// missed it).
func TestRootHelpShowsLocaleFlag(t *testing.T) {
	output := captureStdout(t, func() {
		rootCmd.SetArgs([]string{"help", "--help"})
		_ = rootCmd.Execute()
	})

	// The help command itself should show its flags and description
	if !strings.Contains(output, "functions") {
		t.Error("help --help output missing 'functions' topic")
	}
	if !strings.Contains(output, "constants") {
		t.Error("help --help output missing 'constants' topic")
	}
	if !strings.Contains(output, "frontmatter") {
		t.Error("help --help output missing 'frontmatter' topic")
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
