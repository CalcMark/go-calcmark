package editor

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/CalcMark/go-calcmark/v2/spec/document"
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
		{"Ctrl+Right", tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModCtrl}},
		{"Alt+Right", tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModAlt}},
		{"Alt+f", tea.KeyPressMsg{Code: 'f', Mod: tea.ModAlt}},
		{"Alt+F", tea.KeyPressMsg{Code: 'F', Mod: tea.ModAlt}},
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
		{"Ctrl+Left", tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModCtrl}},
		{"Alt+Left", tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModAlt}},
		{"Alt+b", tea.KeyPressMsg{Code: 'b', Mod: tea.ModAlt}},
		{"Alt+B", tea.KeyPressMsg{Code: 'B', Mod: tea.ModAlt}},
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
