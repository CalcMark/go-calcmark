package editor

// key_dispatch.go — Main key handling dispatch.
// Routes key events to mode-specific handlers and global shortcuts.

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// handleKey is the main key dispatch. It routes key events based on:
// 1. Global shortcuts (Ctrl+C, Ctrl+Q, Ctrl+S, Ctrl+E, Ctrl+O, Ctrl+H)
// 2. Mode-specific handlers (help, autocomplete, command menu, file picker, etc.)
// 3. Default editing mode
func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Invalidate aligned model cache - state may change
	m.InvalidateAlignedCache()

	// Clear status message on any key — but only in modes that don't use
	// the status bar for prompts. Modal overlays manage their own state.
	switch m.mode {
	case StateExport, StateSavePrompt:
		// These modes manage statusMsg themselves — don't clear it
	default:
		m.statusMsg = ""
		m.statusIsErr = false
	}

	// Global shortcuts that work in all modes.
	// Delegated to executeCommandByName to avoid duplicating dispatch logic.
	switch msg.String() {
	case "ctrl+c":
		// Ctrl+C: copy if selection exists, quit if no selection.
		// This preserves Unix interrupt behavior while enabling copy.
		newModel, cmd, handled := m.handleCopy()
		if handled {
			return newModel, cmd
		}
		// No selection — fall through to quit behavior.
		m.quitting = true
		return m, tea.Quit
	case "ctrl+q":
		return m.executeCommandByName("Quit")
	case "ctrl+s":
		return m.executeCommandByName("Save")
	case "ctrl+e":
		return m.executeCommandByName("Export")
	case "ctrl+o":
		return m.executeCommandByName("Open")
	case "ctrl+f":
		return m.insertFrontmatter()
	}

	// Handle command menu toggle (Ctrl+H/F1) - works regardless of mode
	if key.Matches(msg, m.keys.Help) {
		if m.mode == StateCommandMenu {
			m.mode = StateDefault
		} else {
			m.mode = StateCommandMenu
			m.commandMenuState.Selected = 0 // Reset selection when opening
		}
		return m, nil
	}

	// Mode-specific handling for UI overlays
	switch m.mode {
	case StateHelp:
		return m.handleHelpOverlayKey(msg)
	case StateAutocomplete:
		return m.handleAutocompleteKey(msg)
	case StateGlobals:
		return m.handleGlobalsKey(msg)
	case StateCommandMenu:
		return m.handleCommandMenuKey(msg)
	case StateFilePicker:
		return m.handleFilePickerKey(msg)
	case StateExport:
		return m.handleExportOverlayKey(msg)
	case StateSavePrompt:
		return m.handleSavePromptKey(msg)
	default:
		// StateDefault - user is always editing
		return m.handleDefaultKey(msg)
	}
}

// handleDefaultKey processes keys in the default editing mode.
// The user is ALWAYS able to type and edit - this is the only mode they experience.
func (m Model) handleDefaultKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Handle Shift+navigation for text selection.
	// Must be checked before Alt+Arrow and the main switch because
	// shift+arrow key strings won't match plain arrow cases.
	if msg.Mod.Contains(tea.ModShift) {
		hasCtrl := msg.Mod.Contains(tea.ModCtrl)
		hasAlt := msg.Mod.Contains(tea.ModAlt)

		switch msg.Code {
		case tea.KeyUp:
			if !hasCtrl && !hasAlt {
				return m.handleShiftUp()
			}
		case tea.KeyDown:
			if !hasCtrl && !hasAlt {
				return m.handleShiftDown()
			}
		case tea.KeyLeft:
			if hasCtrl || hasAlt {
				return m.handleShiftCtrlLeft()
			}
			return m.handleShiftLeft()
		case tea.KeyRight:
			if hasCtrl || hasAlt {
				return m.handleShiftCtrlRight()
			}
			return m.handleShiftRight()
		case tea.KeyHome:
			if hasCtrl {
				return m.handleShiftCtrlHome()
			}
			return m.handleShiftHome()
		case tea.KeyEnd:
			if hasCtrl {
				return m.handleShiftCtrlEnd()
			}
			return m.handleShiftEnd()
		}
	}

	// Handle Alt+Arrow for word navigation (Option+Arrow on macOS)
	// This is an alternative to Ctrl+Arrow which is often captured by macOS
	if msg.Mod.Contains(tea.ModAlt) {
		switch msg.String() {
		case "alt+left":
			return m.handleCtrlLeftKey()
		case "alt+right":
			return m.handleCtrlRightKey()
		}
		// On macOS terminals, Option+Arrow often sends ESC+b or ESC+f
		// These appear as Alt+b and Alt+f (standard readline/emacs bindings)
		// In v2, modifier combos have the character in Code, not Text.
		if msg.Code == 'b' || msg.Code == 'B' {
			return m.handleCtrlLeftKey() // backward word
		}
		if msg.Code == 'f' || msg.Code == 'F' {
			return m.handleCtrlRightKey() // forward word
		}
	}

	switch msg.String() {
	case "tab":
		// Tab triggers autocomplete
		return m.triggerAutocomplete()
	case "up":
		return m.handleUpKey()
	case "down":
		return m.handleDownKey()
	case "left":
		return m.handleLeftKey()
	case "right":
		return m.handleRightKey()
	case "ctrl+left":
		return m.handleCtrlLeftKey()
	case "ctrl+right":
		return m.handleCtrlRightKey()
	case "pgup":
		return m.handlePageUpKey()
	case "pgdown":
		return m.handlePageDownKey()
	case "home":
		return m.handleHomeKey()
	case "end":
		return m.handleEndKey()
	case "ctrl+home":
		return m.handleCtrlHomeKey()
	case "ctrl+end":
		return m.handleCtrlEndKey()
	case "esc":
		// ESC does nothing in normal editing mode - it's only for canceling special modes
		// (like globals panel, export mode, save-as dialog, etc.)
		return m, nil
	case "enter":
		return m.handleEnterKey()
	case "backspace":
		return m.handleBackspaceKey()
	case "delete":
		return m.handleDeleteKey()
	case "ctrl+p":
		return m.handleCtrlP()
	case "ctrl+d":
		return m.handleCtrlD()
	case "ctrl+u":
		return m.handleCtrlU()
	case "ctrl+k":
		// Delete current line
		m.deleteLine()
		return m, nil
	case "ctrl+z":
		return m.handleUndo()
	case "ctrl+y":
		return m.handleRedo()
	case "ctrl+a":
		// Select all - Ctrl+A
		m.SelectAll()
		return m, nil
	case "ctrl+x":
		// Cut - Ctrl+X
		return m.handleCut()
	case "ctrl+v":
		// Paste - Ctrl+V
		return m.handlePaste()
	case "space":
		return m.handleSpaceKey()
	default:
		// Handle text input (printable characters)
		if msg.Text != "" {
			return m.handleRuneInput([]rune(msg.Text))
		}
	}

	return m, nil
}
