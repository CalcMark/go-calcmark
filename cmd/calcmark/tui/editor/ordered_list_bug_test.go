package editor

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/CalcMark/go-calcmark/spec/document"
)

// TestCalculationBeforeOrderedList verifies that a calculation result
// doesn't disappear when starting an ordered list on the next line.
// This reproduces the bug shown in the screenshot.
func TestCalculationBeforeOrderedList(t *testing.T) {
	// Start with: calc on line 1, empty line 2, cursor on line 3
	doc, err := document.NewDocument("a = 2 *5\n\n")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Verify calc result is visible
	results := m.GetLineResults()
	t.Logf("Initial results for %d lines:", len(results))
	for i, r := range results {
		t.Logf("  Line %d: IsCalc=%v, Value=%q, Source=%q", i, r.IsCalc, r.Value, r.Source)
	}

	if len(results) == 0 || !results[0].IsCalc {
		t.Fatal("Line 0 should be a calculation")
	}
	if results[0].Value == "" {
		t.Fatal("Calculation should have a value")
	}

	calcValue := results[0].Value
	t.Logf("Calc result: %s", calcValue)

	// Now move cursor to line 2 (after empty line) and type "1. "
	m.cursorLine = 2
	m.cursorCol = 0

	// Type "1"
	result, _ := m.Update(tea.KeyPressMsg{Code: '.', Text: "."})
	m = result.(Model)

	// Type "."
	result, _ = m.Update(tea.KeyPressMsg{Code: '.', Text: "."})
	m = result.(Model)

	// Type " "
	result, _ = m.Update(tea.KeyPressMsg{Code: '.', Text: "."})
	m = result.(Model)

	t.Logf("After typing '1. ', editBuf=%q", m.editBuf)

	// The calculation should STILL be visible and have a result
	results = m.GetLineResults()
	t.Logf("After typing '1. ', results for %d lines:", len(results))
	for i, r := range results {
		t.Logf("  Line %d: IsCalc=%v, Value=%q, Source=%q", i, r.IsCalc, r.Value, r.Source)
	}

	// CRITICAL: The first line should STILL be a calc with the SAME value
	if len(results) == 0 {
		t.Fatal("No results after typing '1. ' - this is the bug!")
	}

	if !results[0].IsCalc {
		t.Error("Line 0 should STILL be a calculation after typing '1. ' on line 2")
	}

	if results[0].Value == "" {
		t.Error("Calculation result disappeared when typing '1. ' - THIS IS THE BUG!")
	} else if results[0].Value != calcValue {
		t.Errorf("Calculation result changed: was %s, now %s", calcValue, results[0].Value)
	}
}
