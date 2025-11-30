package repl

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/config"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/shared"
	"github.com/CalcMark/go-calcmark/spec/document"
	tea "github.com/charmbracelet/bubbletea"
)

func init() {
	// Initialize config for tests
	config.Load()
}

func TestNewModel(t *testing.T) {
	// Test with nil document
	m := New(nil)
	if m.doc == nil {
		t.Error("Expected document to be initialized")
	}
	if m.eval == nil {
		t.Error("Expected evaluator to be initialized")
	}
	if m.inputMode != shared.InputNormal {
		t.Errorf("Expected InputNormal mode, got %v", m.inputMode)
	}

	// Test with existing document
	doc, _ := document.NewDocument("x = 10\n")
	m = New(doc)
	if m.doc != doc {
		t.Error("Expected document to be set")
	}
	if !m.pinnedVars["x"] {
		t.Error("Expected x to be auto-pinned")
	}
}

func TestHandleKeyCtrlC(t *testing.T) {
	m := New(nil)
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	result := newModel.(Model)

	if !result.quitting {
		t.Error("Ctrl+C should set quitting=true")
	}
	if cmd == nil {
		t.Error("Ctrl+C should return quit command")
	}
}

func TestHandleKeyCtrlD(t *testing.T) {
	m := New(nil)
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	result := newModel.(Model)

	if !result.quitting {
		t.Error("Ctrl+D should set quitting=true")
	}
	if cmd == nil {
		t.Error("Ctrl+D should return quit command")
	}
}

func TestSlashModeEntry(t *testing.T) {
	m := New(nil)

	// Simulate typing '/' on empty input
	newModel, _ := m.Update(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'/'},
	})
	result := newModel.(Model)

	if result.inputMode != shared.InputSlash {
		t.Error("Typing / should enter slash mode")
	}
	if result.input.Prompt != "/ " {
		t.Errorf("Expected '/ ' prompt, got %q", result.input.Prompt)
	}
}

func TestSlashModeEscapeExit(t *testing.T) {
	m := New(nil)
	m.inputMode = shared.InputSlash
	m.input.Prompt = "/ "

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	result := newModel.(Model)

	if result.inputMode != shared.InputNormal {
		t.Error("Escape should exit slash mode")
	}
	if result.input.Prompt != "> " {
		t.Errorf("Expected '> ' prompt after escape, got %q", result.input.Prompt)
	}
}

func TestHandleCommandPin(t *testing.T) {
	doc, _ := document.NewDocument("x = 5\ny = 10\n")
	m := New(doc)

	// Unpin all first
	m, _ = m.handleCommand("unpin")
	if len(m.pinnedVars) != 0 {
		t.Error("unpin should clear all")
	}

	// Pin all
	m, _ = m.handleCommand("pin")
	if !m.pinnedVars["x"] || !m.pinnedVars["y"] {
		t.Error("pin should pin all variables")
	}

	// Unpin specific
	m, _ = m.handleCommand("unpin x")
	if m.pinnedVars["x"] {
		t.Error("unpin x should unpin x")
	}
	if !m.pinnedVars["y"] {
		t.Error("y should still be pinned")
	}

	// Pin specific
	m, _ = m.handleCommand("pin x")
	if !m.pinnedVars["x"] {
		t.Error("pin x should pin x")
	}
}

func TestHandleCommandQuit(t *testing.T) {
	m := New(nil)

	m, _ = m.handleCommand("quit")
	if !m.quitting {
		t.Error("quit should set quitting=true")
	}

	m = New(nil)
	m, _ = m.handleCommand("q")
	if !m.quitting {
		t.Error("q should set quitting=true")
	}
}

func TestHandleCommandHelp(t *testing.T) {
	m := New(nil)
	initialLen := len(m.outputHistory)

	m, _ = m.handleCommand("help")
	if len(m.outputHistory) != initialLen+1 {
		t.Error("help should add to output history")
	}

	// Test aliases
	m, _ = m.handleCommand("h")
	if len(m.outputHistory) != initialLen+2 {
		t.Error("h should add to output history")
	}

	m, _ = m.handleCommand("?")
	if len(m.outputHistory) != initialLen+3 {
		t.Error("? should add to output history")
	}
}

func TestHandleCommandUnknown(t *testing.T) {
	m := New(nil)
	m, _ = m.handleCommand("notarealcommand")

	if m.err == nil {
		t.Error("Unknown command should set error")
	}
	if !strings.Contains(m.err.Error(), "unknown command") {
		t.Errorf("Expected 'unknown command' error, got: %v", m.err)
	}
}

func TestHandleCommandEdit(t *testing.T) {
	m := New(nil)
	_, cmd := m.handleCommand("edit")

	if cmd == nil {
		t.Error("/edit should return a command")
	}

	// Execute the command to get the message
	msg := cmd()
	switchMsg, ok := msg.(shared.SwitchModeMsg)
	if !ok {
		t.Fatalf("Expected SwitchModeMsg, got %T", msg)
	}
	if switchMsg.Mode != shared.ModeEditor {
		t.Errorf("Expected ModeEditor, got %v", switchMsg.Mode)
	}

	// Test with filepath
	_, cmd = m.handleCommand("edit test.cm")
	msg = cmd()
	switchMsg = msg.(shared.SwitchModeMsg)
	if switchMsg.Filepath != "test.cm" {
		t.Errorf("Expected filepath 'test.cm', got %q", switchMsg.Filepath)
	}
}

func TestHistoryNavigation(t *testing.T) {
	m := New(nil)

	// Add some history
	m.history = []string{"x = 1", "y = 2", "z = 3"}

	// Navigate up
	tm, _ := m.handleHistoryUp()
	result := tm.(Model)
	if result.input.Value() != "z = 3" {
		t.Errorf("Expected 'z = 3', got %q", result.input.Value())
	}

	tm, _ = result.handleHistoryUp()
	result = tm.(Model)
	if result.input.Value() != "y = 2" {
		t.Errorf("Expected 'y = 2', got %q", result.input.Value())
	}

	// Navigate down
	tm, _ = result.handleHistoryDown()
	result = tm.(Model)
	if result.input.Value() != "z = 3" {
		t.Errorf("Expected 'z = 3', got %q", result.input.Value())
	}

	tm, _ = result.handleHistoryDown()
	result = tm.(Model)
	if result.input.Value() != "" {
		t.Errorf("Expected empty input after history end, got %q", result.input.Value())
	}
}

func TestExtractLastToken(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"x", "x"},
		{"x = 1 + foo", "foo"},
		{"x = bar * baz", "baz"},
		{"test_var", "test_var"},
		{"x + ", ""},
	}

	for _, tt := range tests {
		got := extractLastToken(tt.input)
		if got != tt.want {
			t.Errorf("extractLastToken(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsIdentChar(t *testing.T) {
	valid := []byte{'a', 'z', 'A', 'Z', '0', '9', '_'}
	invalid := []byte{' ', '+', '-', '*', '/', '(', ')'}

	for _, ch := range valid {
		if !isIdentChar(ch) {
			t.Errorf("isIdentChar(%c) should be true", ch)
		}
	}

	for _, ch := range invalid {
		if isIdentChar(ch) {
			t.Errorf("isIdentChar(%c) should be false", ch)
		}
	}
}

func TestGetSlashCommandSuggestions(t *testing.T) {
	commands := shared.DefaultSlashCommands()

	// Empty input returns all
	all := GetSlashCommandSuggestions("", commands)
	if len(all) != len(commands) {
		t.Errorf("Empty input should return all commands")
	}

	// Prefix matching
	qMatches := GetSlashCommandSuggestions("q", commands)
	if len(qMatches) != 2 { // "quit" and "q"
		t.Errorf("Expected 2 matches for 'q', got %d", len(qMatches))
	}

	// No match
	noMatch := GetSlashCommandSuggestions("xyz", commands)
	if len(noMatch) != 0 {
		t.Error("Expected no matches for 'xyz'")
	}
}

func TestRenderHelpLine(t *testing.T) {
	// Normal mode - should mention history and commands
	result := RenderHelpLine(false, 60)
	if !strings.Contains(result, "history") {
		t.Error("Normal mode should mention history navigation")
	}
	if !strings.Contains(result, "/help") {
		t.Error("Normal mode should mention /help")
	}
	if !strings.Contains(result, "/quit") {
		t.Error("Normal mode should mention /quit")
	}

	// Slash mode - should mention help and quit
	result = RenderHelpLine(true, 100)
	if !strings.Contains(result, "/help") {
		t.Error("Slash mode should mention /help")
	}
	if !strings.Contains(result, "Esc") {
		t.Error("Slash mode should mention Esc to cancel")
	}
}
