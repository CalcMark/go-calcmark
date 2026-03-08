package editor

import (
	"testing"
)

// TestCommandNameConsistency verifies that all command names referenced
// in EditorCommands and helpCategories are handled by executeCommandByName.
// A typo in any location would cause a silent no-op.
func TestCommandNameConsistency(t *testing.T) {
	dispatched := dispatchedCommandNames()

	// Every command menu item must be handled by executeCommandByName
	for _, cmd := range EditorCommands {
		if !dispatched[cmd.Name] {
			t.Errorf("EditorCommands has %q but executeCommandByName does not handle it", cmd.Name)
		}
	}

	// Every actionable help item must be handled by executeCommandByName
	for _, cat := range helpCategories() {
		for _, item := range cat.Items {
			if item.Kind == HelpActionable && item.CommandName != "" {
				if !dispatched[item.CommandName] {
					t.Errorf("helpCategories has actionable %q but executeCommandByName does not handle it", item.CommandName)
				}
			}
		}
	}
}

// dispatchedCommandNames returns the set of command names handled by
// executeCommandByName. Maintained manually — update when adding commands.
func dispatchedCommandNames() map[string]bool {
	return map[string]bool{
		"New":                true,
		"Save":               true,
		"Save As":            true,
		"Open":               true,
		"Export":             true,
		"Quit":               true,
		"Undo":               true,
		"Redo":               true,
		"Delete Line":        true,
		"Insert Frontmatter": true,
		"Toggle Preview":     true,
		"Word Left":          true,
		"Word Right":         true,
		"Doc Start":          true,
		"Doc End":            true,
		"Full Help":          true,
		"Share To Gist":      true,
		"Open From Gist":     true,
	}
}
