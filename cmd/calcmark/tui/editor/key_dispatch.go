package editor

// key_dispatch.go — Main key handling dispatch.
// Routes key events to mode-specific handlers and global shortcuts.

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// handleKey is the main key dispatch. It routes key events based on:
// 1. Global shortcuts (Ctrl+C, Ctrl+Q, Ctrl+S, Ctrl+E, Ctrl+O, Ctrl+H)
// 2. Mode-specific handlers (help, autocomplete, command menu, file picker, etc.)
// 3. Default editing mode
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
	switch msg.Type {
	case tea.KeyCtrlC:
		// Ctrl+C: copy if selection exists, quit if no selection.
		// This preserves Unix interrupt behavior while enabling copy.
		newModel, cmd, handled := m.handleCopy()
		if handled {
			return newModel, cmd
		}
		// No selection — fall through to quit behavior.
		m.quitting = true
		return m, tea.Quit
	case tea.KeyCtrlQ:
		return m.executeCommandByName("Quit")
	case tea.KeyCtrlS:
		return m.executeCommandByName("Save")
	case tea.KeyCtrlE:
		return m.executeCommandByName("Export")
	case tea.KeyCtrlO:
		return m.executeCommandByName("Open")
	}

	// Ctrl+F: Insert Frontmatter (global shortcut)
	if msg.Type == tea.KeyCtrlF {
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
func (m Model) handleDefaultKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle Alt+Arrow for word navigation (Option+Arrow on macOS)
	// This is an alternative to Ctrl+Arrow which is often captured by macOS
	if msg.Alt {
		switch msg.Type {
		case tea.KeyLeft:
			return m.handleCtrlLeftKey()
		case tea.KeyRight:
			return m.handleCtrlRightKey()
		case tea.KeyRunes:
			// On macOS terminals, Option+Arrow often sends ESC+b or ESC+f
			// These appear as Alt+b and Alt+f (standard readline/emacs bindings)
			if len(msg.Runes) == 1 {
				switch msg.Runes[0] {
				case 'b', 'B':
					return m.handleCtrlLeftKey() // backward word
				case 'f', 'F':
					return m.handleCtrlRightKey() // forward word
				}
			}
		}
	}

	switch msg.Type {
	case tea.KeyTab:
		// Tab triggers autocomplete
		return m.triggerAutocomplete()
	case tea.KeyUp:
		return m.handleUpKey()
	case tea.KeyDown:
		return m.handleDownKey()
	case tea.KeyLeft:
		return m.handleLeftKey()
	case tea.KeyRight:
		return m.handleRightKey()
	case tea.KeyCtrlLeft:
		return m.handleCtrlLeftKey()
	case tea.KeyCtrlRight:
		return m.handleCtrlRightKey()
	case tea.KeyPgUp:
		return m.handlePageUpKey()
	case tea.KeyPgDown:
		return m.handlePageDownKey()
	case tea.KeyHome:
		return m.handleHomeKey()
	case tea.KeyEnd:
		return m.handleEndKey()
	case tea.KeyCtrlHome:
		return m.handleCtrlHomeKey()
	case tea.KeyCtrlEnd:
		return m.handleCtrlEndKey()
	case tea.KeyEsc:
		// ESC does nothing in normal editing mode - it's only for canceling special modes
		// (like globals panel, export mode, save-as dialog, etc.)
		return m, nil
	case tea.KeyEnter:
		return m.handleEnterKey()
	case tea.KeyBackspace:
		return m.handleBackspaceKey()
	case tea.KeyDelete:
		return m.handleDeleteKey()
	case tea.KeyCtrlP:
		return m.handleCtrlP()
	case tea.KeyCtrlD:
		return m.handleCtrlD()
	case tea.KeyCtrlU:
		return m.handleCtrlU()
	case tea.KeyCtrlK:
		// Delete current line
		m.deleteLine()
		return m, nil
	case tea.KeyCtrlZ:
		return m.handleUndo()
	case tea.KeyCtrlY:
		return m.handleRedo()
	case tea.KeyCtrlA:
		// Select all - Ctrl+A
		m.SelectAll()
		return m, nil
	case tea.KeyCtrlX:
		// Cut - Ctrl+X
		return m.handleCut()
	case tea.KeyCtrlV:
		// Paste - Ctrl+V
		return m.handlePaste()
	case tea.KeySpace:
		return m.handleSpaceKey()
	case tea.KeyRunes:
		return m.handleRuneInput(msg.Runes)
	}

	return m, nil
}
