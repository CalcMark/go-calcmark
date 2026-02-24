package editor

import (
	"testing"

	"github.com/CalcMark/go-calcmark/spec/document"
	tea "charm.land/bubbletea/v2"
)

// TestWordNavigationDispatch verifies that Ctrl+Left/Right, Alt+Left/Right,
// and Alt+b/f key messages all correctly dispatch to word navigation handlers
// when sent through the model's Update method (the full dispatch chain).
//
// On macOS, Ctrl+Arrow is intercepted by Mission Control, so Alt/Option+Arrow
// and Alt/Option+B/F are the primary working bindings. Ctrl+Arrow works on Linux.
func TestWordNavigationDispatch(t *testing.T) {
	content := "hello world test\n"
	doc, err := document.NewDocument(content)
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	// Key messages that should all trigger word-right navigation
	wordRightKeys := []struct {
		name string
		msg  tea.KeyMsg
	}{
		{"Ctrl+Right", tea.KeyMsg{Type: tea.KeyCtrlRight}},
		{"Alt+Right", tea.KeyMsg{Type: tea.KeyRight, Alt: true}},
		{"Alt+f", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}, Alt: true}},
		{"Alt+F", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'F'}, Alt: true}},
	}

	for _, tc := range wordRightKeys {
		t.Run(tc.name+" moves forward", func(t *testing.T) {
			m := New(doc)
			m.width = 80
			m.height = 24
			m.cursorLine = 0
			m.cursorCol = 0

			result, _ := m.Update(tc.msg)
			m = result.(Model)

			// From col 0 of "hello world test", word-right should move past "hello " to col 6
			if m.cursorCol == 0 {
				t.Errorf("%s: cursor did not move from col 0 (word navigation not dispatched)", tc.name)
			}
			if m.cursorCol != 6 {
				t.Errorf("%s: expected cursorCol=6, got %d", tc.name, m.cursorCol)
			}
		})
	}

	// Key messages that should all trigger word-left navigation
	wordLeftKeys := []struct {
		name string
		msg  tea.KeyMsg
	}{
		{"Ctrl+Left", tea.KeyMsg{Type: tea.KeyCtrlLeft}},
		{"Alt+Left", tea.KeyMsg{Type: tea.KeyLeft, Alt: true}},
		{"Alt+b", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}, Alt: true}},
		{"Alt+B", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'B'}, Alt: true}},
	}

	for _, tc := range wordLeftKeys {
		t.Run(tc.name+" moves backward", func(t *testing.T) {
			m := New(doc)
			m.width = 80
			m.height = 24
			m.cursorLine = 0
			m.cursorCol = 16 // End of "hello world test"
			m.editBuf = "hello world test"

			result, _ := m.Update(tc.msg)
			m = result.(Model)

			// From col 16 of "hello world test", word-left should move to start of "test" at col 12
			if m.cursorCol == 16 {
				t.Errorf("%s: cursor did not move from col 16 (word navigation not dispatched)", tc.name)
			}
			if m.cursorCol != 12 {
				t.Errorf("%s: expected cursorCol=12, got %d", tc.name, m.cursorCol)
			}
		})
	}
}
