package editor

import (
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/lipgloss"
)

// FilePickerFocus indicates which part of the file dialog has focus.
type FilePickerFocus int

const (
	FocusFileBrowser FilePickerFocus = iota // Arrow keys navigate directory
	FocusFilename                           // Typing updates filename
)

// FilePickerPurpose indicates why the file picker was opened.
type FilePickerPurpose int

const (
	PickerForSave   FilePickerPurpose = iota // Save or Save As
	PickerForOpen                            // Open file
	PickerForExport                          // Export to format
)

// initFilePicker creates and configures a new filepicker for the editor.
// Starts in the current working directory and shows .cm files prominently.
func initFilePicker() filepicker.Model {
	fp := filepicker.New()

	// Start in current working directory, fall back to home
	if cwd, err := os.Getwd(); err == nil {
		fp.CurrentDirectory = cwd
	} else if home, err := os.UserHomeDir(); err == nil {
		fp.CurrentDirectory = home
	}

	// Configuration - show ALL files and directories for navigation
	// Don't filter by extension - user needs to see directories to navigate
	fp.AllowedTypes = []string{} // Empty = show all files
	fp.DirAllowed = false        // Enter on directory = navigate into it (not select)
	fp.FileAllowed = true        // Enter on file = select for overwrite
	fp.ShowHidden = false
	fp.ShowPermissions = false // Permissions are not useful for save/open and consume space
	fp.ShowSize = true
	fp.SetHeight(15)

	// Apply dark theme styling to match editor
	fp.Styles.Directory = lipgloss.NewStyle().
		Foreground(lipgloss.Color("33")).Bold(true)
	fp.Styles.File = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))
	fp.Styles.Selected = lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).Bold(true)
	fp.Styles.Cursor = lipgloss.NewStyle().
		Foreground(lipgloss.Color("205"))

	return fp
}

// addExtensionIfMissing ensures filename ends with .cm extension.
func addExtensionIfMissing(filename string) string {
	if filepath.Ext(filename) == "" {
		return filename + ".cm"
	}
	return filename
}

// addExportExtension adds the appropriate file extension for an export format.
func addExportExtension(filename, formatName string) string {
	if filepath.Ext(filename) == "" {
		return filename + formatToExtension(formatName)
	}
	return filename
}
