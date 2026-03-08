package editor

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/CalcMark/go-calcmark/spec/document"
)

// TestScaleChangeViaDebounce verifies that editing a frontmatter value
// and waiting for the debounce timer (no navigation) updates the preview.
// This simulates the real TUI flow: type → wait → debounce fires.
func TestScaleChangeViaDebounce(t *testing.T) {
	content := "---\nscale: 1\nglobals:\n  tax_rate: 0.085\n---\n@globals.tax_rate * 2"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	var model tea.Model = m

	getValue := func() string {
		ed := model.(Model)
		for _, lr := range ed.GetLineResults() {
			if lr.IsCalc && lr.Value != "" {
				return lr.Value
			}
		}
		return ""
	}

	t.Logf("Initial: result=%q", getValue())

	// Navigate to scale line
	model = sendKey(t, model, "down") // line 1: "scale: 1"
	model = sendKey(t, model, "end")  // end of line

	// Delete "1" and type "3" (to change scale: 1 → scale: 3)
	model = sendKey(t, model, "backspace")
	model = typeText(t, model, "3")

	// DO NOT navigate — just fire the debounce timer
	ed := model.(Model)
	t.Logf("Before debounce: editBuf=%q cursorLine=%d userIsTyping=%v",
		ed.editBuf, ed.cursorLine, ed.userIsTyping)

	// Simulate debounce firing
	debounceMsg := evalDebounceMsg{editBufSnapshot: ed.editBuf}
	model, _ = model.Update(debounceMsg)

	result := getValue()
	t.Logf("After debounce: result=%q", result)

	// @globals.tax_rate * 2 = 0.085 * 2 = 0.17
	// scale:3 doesn't affect plain numbers, so result should still be 0.17
	if result == "" {
		ed = model.(Model)
		// Check for errors
		for _, lr := range ed.GetLineResults() {
			if lr.Error != "" {
				t.Logf("Error on line %d: %q", lr.LineNum, lr.Error)
			}
		}
		t.Errorf("result is empty after debounce — expression not evaluated")
	} else if result != "0.17" {
		t.Errorf("result=%q, want 0.17", result)
	}
}

// TestScaleChangeViaNavigation verifies the navigation path works.
func TestScaleChangeViaNavigation(t *testing.T) {
	content := "---\nscale: 1\nglobals:\n  tax_rate: 0.085\n---\n@globals.tax_rate * 2"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	var model tea.Model = m

	getValue := func() string {
		ed := model.(Model)
		for _, lr := range ed.GetLineResults() {
			if lr.IsCalc && lr.Value != "" {
				return lr.Value
			}
		}
		return ""
	}

	t.Logf("Initial: result=%q", getValue())

	// Navigate to scale line and change
	model = sendKey(t, model, "down")
	model = sendKey(t, model, "end")
	model = sendKey(t, model, "backspace")
	model = typeText(t, model, "3")

	// Navigate away to flush
	for range 6 {
		model = sendKey(t, model, "down")
	}

	result := getValue()
	t.Logf("After navigation flush: result=%q", result)

	if result != "0.17" {
		t.Errorf("result=%q, want 0.17", result)
	}
}

// TestScaleChangeUpdatesScaledCurrency verifies currency scaling via TUI.
func TestScaleChangeUpdatesScaledCurrency(t *testing.T) {
	content := "---\nscale:\n  factor: 1\n  unit_categories: [Currency]\n---\ncost = $5.00"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24
	var model tea.Model = m

	getValue := func() string {
		ed := model.(Model)
		for _, lr := range ed.GetLineResults() {
			if lr.VarName == "cost" {
				return lr.Value
			}
		}
		return ""
	}

	if v := getValue(); v != "$5.00" {
		t.Fatalf("Initial cost=%q, want $5.00", v)
	}

	// Navigate to factor line and change 1→2
	model = sendKey(t, model, "down") // scale:
	model = sendKey(t, model, "down") // factor: 1
	model = sendKey(t, model, "end")
	model = sendKey(t, model, "backspace")
	model = typeText(t, model, "2")
	model = sendKey(t, model, "down")

	if cost := getValue(); cost != "$10.00" {
		t.Errorf("cost=%q with factor:2, want $10.00", cost)
	}
}
