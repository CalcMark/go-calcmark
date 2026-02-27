package editor

// file_picker_handler.go — Key handling for the file picker overlay.

import (
	"path/filepath"

	tea "charm.land/bubbletea/v2"
)

func (m Model) handleFilePickerKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Global keys that work in both modes
	switch msg.String() {
	case "esc":
		// Cancel and return to editing
		m.exitOverlay()
		return m, nil

	case "tab":
		// Toggle focus between filename and browser
		if m.filePickerFocus == FocusFilename {
			m.filePickerFocus = FocusFileBrowser
		} else {
			m.filePickerFocus = FocusFilename
		}
		return m, nil

	case "enter":
		if m.filePickerPurpose == PickerForSave {
			// Save mode: Enter with filename input saves the file
			if m.filePickerFocus == FocusFilename {
				if m.newFileName != "" {
					filename := addExtensionIfMissing(m.newFileName)
					path := filepath.Join(m.filePicker.CurrentDirectory, filename)
					m.saveFile(path)
					m.newFileName = ""
					if !m.statusIsErr {
						// Save succeeded — complete whatever action triggered the save
						return m.completePendingSaveAction()
					}
				}
				return m, nil
			}
			// If browser has focus, let filepicker handle Enter (navigate/select)
		} else if m.filePickerPurpose == PickerForOpen {
			// Open mode: Enter on a file opens it
			if m.filePickerFocus == FocusFilename {
				if m.newFileName != "" {
					path := filepath.Join(m.filePicker.CurrentDirectory, m.newFileName)
					m.openFile(path)
					if !m.statusIsErr {
						m.exitOverlay()
					}
				}
				return m, nil
			}
			// If browser has focus, file selection handled below via DidSelectFile
		} else if m.filePickerPurpose == PickerForExport {
			// Export mode: Enter with filename exports in the selected format
			if m.filePickerFocus == FocusFilename {
				if m.newFileName != "" {
					format := m.exportFormatOpts[m.exportState.FormatIdx]
					filename := addExportExtension(m.newFileName, format)
					path := filepath.Join(m.filePicker.CurrentDirectory, filename)
					m.exportFile(path, format)
					m.exitOverlay()
				}
				return m, nil
			}
			// If browser has focus, let filepicker handle Enter (navigate/select)
		}
	}

	// Filename input focus - handle typing
	if m.filePickerFocus == FocusFilename {
		switch msg.String() {
		case "backspace":
			if len(m.newFileName) > 0 {
				m.newFileName = m.newFileName[:len(m.newFileName)-1]
			}
			return m, nil
		default:
			if msg.Text != "" {
				m.newFileName += msg.Text
				return m, nil
			}
		}
		return m, nil
	}

	// Browser focus - pass to filepicker for navigation
	var cmd tea.Cmd
	m.filePicker, cmd = m.filePicker.Update(msg)

	// Check if file was selected (existing file)
	if didSelect, path := m.filePicker.DidSelectFile(msg); didSelect {
		if m.filePickerPurpose == PickerForOpen {
			// Open mode: selecting a file opens it immediately
			m.openFile(path)
			if !m.statusIsErr {
				m.exitOverlay()
			}
			return m, cmd
		}
		// Save/Export mode: put filename in the input field for confirmation
		m.newFileName = filepath.Base(path)
		return m, cmd
	}

	return m, cmd
}
