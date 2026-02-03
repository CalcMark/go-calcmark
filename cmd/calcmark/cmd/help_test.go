package cmd

import (
	"bytes"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"
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

	// Verify all 12 function names are present
	expectedFunctions := []string{
		"avg", "sqrt", "accumulate", "convert_rate",
		"downtime", "rtt", "throughput", "transfer_time",
		"read", "seek", "compress", "capacity",
	}

	for _, fn := range expectedFunctions {
		if !strings.Contains(output, fn) {
			t.Errorf("function %q not found in output", fn)
		}
	}

	// Verify categories are present
	expectedCategories := []string{"Math", "Conversion", "Network", "Storage", "Capacity"}
	for _, cat := range expectedCategories {
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

// TestHelpCmdTopics verifies that help command shows available topics.
func TestHelpCmdTopics(t *testing.T) {
	output := captureStdout(t, func() {
		helpCmd.Run(helpCmd, []string{})
	})

	if !strings.Contains(output, "functions") {
		t.Error("help output missing 'functions' topic")
	}
	if !strings.Contains(output, "constants") {
		t.Error("help output missing 'constants' topic")
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
