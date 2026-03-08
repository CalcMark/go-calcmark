package editor

// key_dispatch.go — Main key handling dispatch.
// Routes key events to mode-specific handlers and global shortcuts.

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// handleKey is the main key dispatch. It routes key events based on:
// 1. Global shortcuts (Ctrl+C, Ctrl+Q, Ctrl+N, Ctrl+S, Ctrl+E, Ctrl+O, Ctrl+H)
// 2. Mode-specific handlers (help, autocomplete, command menu, file picker, etc.)
// 3. Default editing mode
func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.debugKeys && m.debugKeyFile != nil {
		fmt.Fprintf(m.debugKeyFile, "[KEY] code=%d mod=%v text=%q str=%q\n",
			msg.Code, msg.Mod, msg.Text, msg.String())
	}

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
	// Only respond to actual Ctrl+key, not Cmd (Super) key on macOS.
	// Terminals using the Kitty keyboard protocol send Super separately;
	// legacy terminals map Cmd→Ctrl and we can't distinguish them.
	//
	if msg.Mod.Contains(tea.ModCtrl) && !msg.Mod.Contains(tea.ModSuper) {
		switch msg.Code {
		case 'c':
			// Ctrl+C: copy if selection exists, quit if no selection.
			newModel, cmd, handled := m.handleCopy()
			if handled {
				return newModel, cmd
			}
			return m.executeCommandByName("Quit")
		case 'q':
			return m.executeCommandByName("Quit")
		case 'n':
			return m.executeCommandByName("New")
		case 's':
			return m.executeCommandByName("Save")
		case 'o':
			return m.executeCommandByName("Open")
		case 'f':
			return m.insertFrontmatter()
		}
	}

	// Handle command menu toggle (Ctrl+H/F1) - works regardless of mode
	if key.Matches(msg, m.keys.Help) {
		if m.mode == StateCommandMenu {
			m.exitOverlay()
		} else {
			m.enterCommandMenu()
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
	case StateShareTo:
		return m.handleShareOverlayKey(msg)
	case StateOpenFrom:
		return m.handleOpenFromOverlayKey(msg)
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
	// Handle Super (Cmd on macOS) key combinations.
	// With the Kitty keyboard protocol, Cmd sends ModSuper (not ModCtrl).
	// Cmd+Arrow maps to Home/End, Cmd+Shift+Arrow extends selection.
	// Cmd+A/C/X/V/Z/Y provide standard macOS editing shortcuts.
	if msg.Mod.Contains(tea.ModSuper) {
		hasShift := msg.Mod.Contains(tea.ModShift)
		switch msg.Code {
		case tea.KeyLeft:
			if hasShift {
				return m.handleShiftHomeKey()
			}
			return m.handleHomeKey()
		case tea.KeyRight:
			if hasShift {
				return m.handleShiftEndKey()
			}
			return m.handleEndKey()
		case tea.KeyUp:
			if hasShift {
				return m.handleShiftCtrlHomeKey()
			}
			return m.handleCtrlHomeKey()
		case tea.KeyDown:
			if hasShift {
				return m.handleShiftCtrlEndKey()
			}
			return m.handleCtrlEndKey()
		case 'a':
			m.SelectAll()
			return m, nil
		case 'c':
			newModel, cmd, handled := m.handleCopy()
			if handled {
				return newModel, cmd
			}
			return m, nil
		case 'x':
			return m.handleCut()
		case 'v':
			return m.handlePaste()
		case 'z':
			if hasShift {
				return m.handleRedo() // Cmd+Shift+Z = Redo (macOS convention)
			}
			return m.handleUndo()
		case 'y':
			return m.handleRedo() // Cmd+Y = Redo (cross-platform alternative)
		}
	}

	// Handle Ctrl+key clipboard and undo shortcuts via modifier+code.
	// Uses the same msg.Mod/msg.Code pattern as the Super block above for
	// reliability across terminal protocols (Kitty, legacy, etc.).
	// Legacy macOS terminals map Cmd→Ctrl, so Cmd+Shift+Z arrives as
	// Ctrl+Shift+Z — handled here for consistent redo behavior.
	if msg.Mod.Contains(tea.ModCtrl) && !msg.Mod.Contains(tea.ModSuper) {
		hasShift := msg.Mod.Contains(tea.ModShift)
		switch msg.Code {
		case 'z':
			if hasShift {
				return m.handleRedo() // Ctrl+Shift+Z = Redo
			}
			return m.handleUndo()
		case 'y':
			return m.handleRedo()
		case 'x':
			return m.handleCut()
		case 'v':
			return m.handlePaste()
		case 'a':
			m.SelectAll()
			return m, nil
		}
	}

	// Handle Shift+navigation for text selection.
	// Must be checked before Alt+Arrow and the main switch because
	// shift+arrow key strings won't match plain arrow cases.
	if msg.Mod.Contains(tea.ModShift) {
		hasCtrl := msg.Mod.Contains(tea.ModCtrl)
		hasAlt := msg.Mod.Contains(tea.ModAlt)

		switch msg.Code {
		case tea.KeyUp:
			if hasCtrl || hasAlt {
				return m.handleShiftCtrlHomeKey() // Select to document start
			}
			return m.handleShiftUpKey()
		case tea.KeyDown:
			if hasCtrl || hasAlt {
				return m.handleShiftCtrlEndKey() // Select to document end
			}
			return m.handleShiftDownKey()
		case tea.KeyLeft:
			if hasCtrl || hasAlt {
				return m.handleShiftCtrlLeftKey()
			}
			return m.handleShiftLeftKey()
		case tea.KeyRight:
			if hasCtrl || hasAlt {
				return m.handleShiftCtrlRightKey()
			}
			return m.handleShiftRightKey()
		case tea.KeyHome:
			if hasCtrl {
				return m.handleShiftCtrlHomeKey()
			}
			return m.handleShiftHomeKey()
		case tea.KeyEnd:
			if hasCtrl {
				return m.handleShiftCtrlEndKey()
			}
			return m.handleShiftEndKey()
		case tea.KeyPgUp:
			return m.handleShiftPageUpKey()
		case tea.KeyPgDown:
			return m.handleShiftPageDownKey()
		}
	}

	// Handle Alt+Arrow for word/document navigation (Option+Arrow on macOS).
	// Alt+Left/Right = word navigation (alternative to Ctrl+Arrow).
	// Alt+Up/Down = document start/end (fallback for Terminal.app where Cmd is consumed).
	if msg.Mod.Contains(tea.ModAlt) {
		switch msg.String() {
		case "alt+left":
			return m.handleCtrlLeftKey()
		case "alt+right":
			return m.handleCtrlRightKey()
		case "alt+up":
			return m.handleCtrlHomeKey()
		case "alt+down":
			return m.handleCtrlEndKey()
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
	case "ctrl+k":
		// Delete current line
		return m.handleDeleteLine()
	case "space":
		return m.handleRuneInput([]rune{' '})
	default:
		// Handle text input (printable characters)
		if msg.Text != "" {
			return m.handleRuneInput([]rune(msg.Text))
		}
	}

	return m, nil
}
