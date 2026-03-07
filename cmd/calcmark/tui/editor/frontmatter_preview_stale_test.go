package editor

// frontmatter_preview_stale_test.go — Tests for stale preview values when
// frontmatter YAML is malformed.
//
// Bug: When frontmatter has parse errors (missing closing ---, empty value,
// invalid YAML), the preview pane still shows values from the last successful
// parse. Users see incorrect results for variables that no longer exist or
// have been modified to invalid states.
//
// Core invariant: GetGlobalsPanelState() must return ZERO globals when
// frontmatterErr is non-nil. Stale data must never leak into the preview.

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/CalcMark/go-calcmark/spec/document"
)

// TestPreviewStale_MissingClosingDelimiter tests that when the closing ---
// delimiter is removed, no preview values are shown (frontmatter is malformed).
//
// Scenario: Well-formed frontmatter → delete closing --- character by character
// → debounce → preview must show no globals.
func TestPreviewStale_MissingClosingDelimiter(t *testing.T) {
	content := "---\nexchange:\n  USD_EUR: 0.92\nglobals:\n  my_var: 42\n---\nx = @globals.my_var + 1"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	var model tea.Model = m

	// Verify initial state has globals
	state := m.GetGlobalsPanelState()
	if len(state.Globals) == 0 {
		t.Fatal("Expected globals in initial state")
	}
	if state.Error != "" {
		t.Fatalf("Expected no error in initial state, got %q", state.Error)
	}
	t.Logf("Initial globals count: %d", len(state.Globals))

	// Navigate to closing --- line (line 5)
	for range 5 {
		model = sendKey(t, model, "down")
	}
	model = sendKey(t, model, "end")

	ed := model.(Model)
	if ed.editBuf != "---" {
		t.Fatalf("Expected editBuf='---', got %q", ed.editBuf)
	}

	// Delete all 3 dashes, debouncing after each
	for i := range 3 {
		model = sendKey(t, model, "backspace")
		model = simulateDebounce(t, model)

		ed = model.(Model)
		state = ed.GetGlobalsPanelState()
		t.Logf("After backspace %d: editBuf=%q frontmatterErr=%v globals=%d error=%q",
			i+1, ed.editBuf, ed.frontmatterErr, len(state.Globals), state.Error)

		// After removing characters from ---, frontmatter becomes malformed.
		// The preview must show NO globals when there's a parse error.
		if ed.frontmatterErr != nil && len(state.Globals) > 0 {
			t.Errorf("BUG: Stale globals shown when frontmatterErr is set (backspace %d)", i+1)
			for _, g := range state.Globals {
				t.Errorf("  stale global: %s = %s", g.Name, g.Value)
			}
		}
	}
}

// TestPreviewStale_EmptyYAMLValue tests that when a YAML value is emptied
// (e.g., "USD_EUR: 0.92" → "USD_EUR:"), the preview does not show the old value.
//
// This is the exact user-reported scenario: empty value shows stale "0.9200".
func TestPreviewStale_EmptyYAMLValue(t *testing.T) {
	content := "---\nexchange:\n  USD_EUR: 0.92\nglobals:\n  my_var: 42\n---\nx = @globals.my_var + 1"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	var model tea.Model = m

	// Navigate to USD_EUR line (line 2: "  USD_EUR: 0.92")
	model = sendKey(t, model, "down")
	model = sendKey(t, model, "down")
	model = sendKey(t, model, "end")

	ed := model.(Model)
	t.Logf("On USD_EUR line: editBuf=%q cursorLine=%d", ed.editBuf, ed.cursorLine)

	// Delete the value "0.92" (4 characters)
	for range 4 {
		model = sendKey(t, model, "backspace")
	}

	ed = model.(Model)
	t.Logf("After clearing value: editBuf=%q", ed.editBuf)
	if ed.editBuf != "  USD_EUR: " {
		t.Fatalf("Expected editBuf='  USD_EUR: ', got %q", ed.editBuf)
	}

	// Also delete the trailing space to get "USD_EUR:"
	model = sendKey(t, model, "backspace")

	ed = model.(Model)
	t.Logf("After removing space: editBuf=%q", ed.editBuf)

	// Navigate down to flush the edit
	model = sendKey(t, model, "down")

	ed = model.(Model)
	state := ed.GetGlobalsPanelState()
	t.Logf("After flush: frontmatterErr=%v globals=%d error=%q",
		ed.frontmatterErr, len(state.Globals), state.Error)

	// When USD_EUR has an empty value, validateExchangeRate should fail
	// (rate must be positive). If frontmatterErr is set, NO globals should appear.
	if ed.frontmatterErr != nil && len(state.Globals) > 0 {
		t.Error("BUG: Stale globals shown when frontmatterErr is set")
		for _, g := range state.Globals {
			t.Errorf("  stale global: %s = %s (isExchange=%v)", g.Name, g.Value, g.IsExchange)
		}
	}

	// Even if frontmatterErr is nil somehow, the value must not be "0.9200"
	for _, g := range state.Globals {
		if g.Name == "USD_EUR" && g.Value == "0.9200" {
			t.Errorf("BUG: Stale value '0.9200' shown for USD_EUR after clearing")
		}
	}
}

// TestPreviewStale_InvalidYAMLShowsNoGlobals tests that completely invalid
// YAML (e.g., duplicate keys, syntax errors) shows no globals.
func TestPreviewStale_InvalidYAMLShowsNoGlobals(t *testing.T) {
	content := "---\nexchange:\n  USD_EUR: 0.92\nglobals:\n  my_var: 42\n---\nx = 10"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	var model tea.Model = m

	// Verify initial state
	state := m.GetGlobalsPanelState()
	if len(state.Globals) != 2 {
		t.Fatalf("Expected 2 globals initially, got %d", len(state.Globals))
	}

	// Navigate to "globals:" line (line 3) and type "globals:" again to create
	// a duplicate key error
	for range 3 {
		model = sendKey(t, model, "down")
	}
	model = sendKey(t, model, "end")
	model = sendKey(t, model, "enter")
	model = typeText(t, model, "globals:")

	// Navigate away to flush
	model = sendKey(t, model, "down")

	ed := model.(Model)
	state = ed.GetGlobalsPanelState()
	t.Logf("After duplicate key: frontmatterErr=%v globals=%d error=%q",
		ed.frontmatterErr, len(state.Globals), state.Error)

	if ed.frontmatterErr != nil && len(state.Globals) > 0 {
		t.Error("BUG: Stale globals shown with duplicate YAML key error")
		for _, g := range state.Globals {
			t.Errorf("  stale global: %s = %s", g.Name, g.Value)
		}
	}
	if ed.frontmatterErr != nil && state.Error == "" {
		t.Error("BUG: frontmatterErr is set but Error string is empty in panel state")
	}
}

// TestPreviewStale_FixFrontmatterRestoresGlobals tests that after fixing
// malformed frontmatter, globals reappear in the preview.
//
// Scenario: Break frontmatter by clearing the exchange rate value (making it
// invalid), verify no globals shown, then type a valid rate back.
func TestPreviewStale_FixFrontmatterRestoresGlobals(t *testing.T) {
	content := "---\nexchange:\n  USD_EUR: 0.92\nglobals:\n  my_var: 42\n---\nx = @globals.my_var + 1"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	var model tea.Model = m

	// Verify initial state: 2 globals (USD_EUR + my_var)
	state := m.GetGlobalsPanelState()
	if len(state.Globals) != 2 {
		t.Fatalf("Expected 2 globals initially, got %d", len(state.Globals))
	}

	// Step 1: Break frontmatter by clearing the USD_EUR value
	// Navigate to line 2 ("  USD_EUR: 0.92"), end, delete "0.92" + space
	model = sendKey(t, model, "down")
	model = sendKey(t, model, "down")
	model = sendKey(t, model, "end")
	for range 5 {
		model = sendKey(t, model, "backspace") // delete " 0.92" → leaves "  USD_EUR:"
	}

	// Flush via navigation
	model = sendKey(t, model, "down")

	ed := model.(Model)
	state = ed.GetGlobalsPanelState()
	t.Logf("After breaking: frontmatterErr=%v globals=%d error=%q",
		ed.frontmatterErr, len(state.Globals), state.Error)

	// Frontmatter should be broken (empty exchange rate) - no globals
	if ed.frontmatterErr == nil {
		t.Fatal("Expected frontmatterErr after clearing exchange rate value")
	}
	if len(state.Globals) > 0 {
		t.Errorf("BUG: Globals should be empty when frontmatter is broken, got %d", len(state.Globals))
	}

	// Step 2: Fix frontmatter by navigating back and typing a valid rate
	model = sendKey(t, model, "up")
	model = sendKey(t, model, "end")
	model = typeText(t, model, " 0.85")

	// Flush via navigation
	model = sendKey(t, model, "down")

	ed = model.(Model)
	state = ed.GetGlobalsPanelState()
	t.Logf("After fixing: frontmatterErr=%v globals=%d", ed.frontmatterErr, len(state.Globals))

	// Frontmatter should be fixed - globals should reappear
	if ed.frontmatterErr != nil {
		t.Errorf("Expected no frontmatterErr after fix, got %v", ed.frontmatterErr)
	}
	if len(state.Globals) == 0 {
		t.Error("BUG: Globals should reappear after fixing frontmatter")
	}

	// Verify specific globals
	foundUSD := false
	foundMyVar := false
	for _, g := range state.Globals {
		if g.Name == "USD_EUR" {
			foundUSD = true
			if g.Value != "0.8500" {
				t.Errorf("Expected USD_EUR=0.8500, got %s", g.Value)
			}
		}
		if g.Name == "my_var" {
			foundMyVar = true
		}
	}
	if !foundUSD {
		t.Error("Expected USD_EUR to reappear after fix")
	}
	if !foundMyVar {
		t.Error("Expected my_var to reappear after fix")
	}
}

// TestPreviewStale_BuildFrontmatterValueMapEmpty tests that buildFrontmatterValueMap
// returns an empty map when frontmatterErr is set.
func TestPreviewStale_BuildFrontmatterValueMapEmpty(t *testing.T) {
	content := "---\nexchange:\n  USD_EUR: 0.92\nglobals:\n  my_var: 42\n---\nx = 10"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Initially should have values
	valueMap := m.buildFrontmatterValueMap()
	if len(valueMap) == 0 {
		t.Fatal("Expected non-empty value map initially")
	}
	if _, ok := valueMap["USD_EUR"]; !ok {
		t.Error("Expected USD_EUR in initial value map")
	}

	// Simulate a frontmatter error
	m.frontmatterErr = fmt.Errorf("frontmatter: test error")

	// Value map should now be empty
	valueMap = m.buildFrontmatterValueMap()
	if len(valueMap) != 0 {
		t.Errorf("BUG: Value map should be empty when frontmatterErr is set, got %d entries", len(valueMap))
		for k, v := range valueMap {
			t.Errorf("  stale entry: %s = %s", k, v.value)
		}
	}
}

// TestPreviewStale_GetGlobalsCountZeroOnError tests that getGlobalsCount
// returns 0 when frontmatterErr is set.
func TestPreviewStale_GetGlobalsCountZeroOnError(t *testing.T) {
	content := "---\nexchange:\n  USD_EUR: 0.92\nglobals:\n  my_var: 42\n---\nx = 10"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Initially should have 2 (USD_EUR + my_var)
	if count := m.getGlobalsCount(); count != 2 {
		t.Fatalf("Expected 2 globals initially, got %d", count)
	}

	// Set error
	m.frontmatterErr = fmt.Errorf("frontmatter: test error")

	// Should be 0
	if count := m.getGlobalsCount(); count != 0 {
		t.Errorf("BUG: getGlobalsCount should return 0 on error, got %d", count)
	}

	// Clear error — count should restore
	m.frontmatterErr = nil
	if count := m.getGlobalsCount(); count != 2 {
		t.Errorf("Expected 2 globals after clearing error, got %d", count)
	}
}
