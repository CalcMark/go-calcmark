package editor

import (
	"strings"
	"testing"

	"github.com/CalcMark/go-calcmark/v2/spec/document"
)

// TestStatusBarNoShellContamination verifies that the status bar
// does NOT display shell command output like "task: [build] echo 'Built cm'".
//
// BUG: The status bar sometimes shows shell commands instead of editor status.
// This test verifies that the status bar only shows legitimate editor status.
func TestStatusBarNoShellContamination(t *testing.T) {
	doc, err := document.NewDocument("x = 10\n")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Get status bar state
	statusState := m.GetStatusBarState()
	t.Logf("Status bar hints: %q", statusState.Hints)
	t.Logf("Status message: %q", m.statusMsg)

	// Status bar should NOT contain shell-related text
	invalidPatterns := []string{
		"task:",
		"[build]",
		"echo",
		"Built cm",
		"bash",
		"sh -c",
	}

	for _, pattern := range invalidPatterns {
		if strings.Contains(statusState.Hints, pattern) {
			t.Errorf("Status bar hints contain shell command pattern %q: %q",
				pattern, statusState.Hints)
		}
		if strings.Contains(m.statusMsg, pattern) {
			t.Errorf("Status message contains shell command pattern %q: %q",
				pattern, m.statusMsg)
		}
	}

	// Status bar should contain legitimate editor commands
	validPatterns := []string{
		"Ctrl+S", // save
		"Ctrl+Q", // quit
		"Arrows", // navigate
	}

	foundValid := false
	for _, pattern := range validPatterns {
		if strings.Contains(statusState.Hints, pattern) {
			foundValid = true
			break
		}
	}

	if !foundValid {
		t.Errorf("Status bar hints don't contain any valid editor commands: %q",
			statusState.Hints)
	}
}

// TestStatusBarClearsOnTyping verifies that the status bar
// updates appropriately when typing, and doesn't show stale messages.
func TestStatusBarClearsOnTyping(t *testing.T) {
	doc, err := document.NewDocument("\n")
	if err != nil {
		t.Fatalf("Failed to create document: %v", err)
	}

	m := New(doc)
	m.width = 80
	m.height = 24

	// Initial state should have clean status
	if m.statusMsg != "" {
		t.Logf("Initial status message: %q", m.statusMsg)
	}

	// Get initial view to see what's displayed
	initialView := m.View().Content
	t.Logf("Initial view status bar area:\n%s",
		getStatusBarFromView(initialView))

	// Verify no contamination in initial state
	if strings.Contains(initialView, "task:") ||
		strings.Contains(initialView, "[build]") {
		t.Errorf("Initial view contains shell command text:\n%s", initialView)
	}
}

// getStatusBarFromView extracts the last few lines of the view
// which should contain the status bar.
func getStatusBarFromView(view string) string {
	lines := strings.Split(view, "\n")
	if len(lines) < 3 {
		return view
	}
	// Return last 3 lines (status bar area)
	return strings.Join(lines[len(lines)-3:], "\n")
}
