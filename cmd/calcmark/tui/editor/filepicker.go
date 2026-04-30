package editor

import (
	"os"
	"path/filepath"

	"charm.land/bubbles/v2/filepicker"
	"charm.land/lipgloss/v2"
	"github.com/CalcMark/go-calcmark/v2/cmd/calcmark/config/theme"
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
// Starts in the current working directory. For open, only .cm/.calcmark files
// are selectable; for save/export, all files are selectable.
func initFilePicker(purpose FilePickerPurpose) filepicker.Model {
	fp := filepicker.New()

	// Start in current working directory, fall back to home
	if cwd, err := os.Getwd(); err == nil {
		fp.CurrentDirectory = cwd
	} else if home, err := os.UserHomeDir(); err == nil {
		fp.CurrentDirectory = home
	}

	// For open, restrict to CalcMark files; for save/export, allow all
	if purpose == PickerForOpen {
		fp.AllowedTypes = []string{".cm", ".calcmark"}
	} else {
		fp.AllowedTypes = []string{} // Empty = show all files
	}
	fp.DirAllowed = false // Enter on directory = navigate into it (not select)
	fp.FileAllowed = true // Enter on file = select for overwrite
	fp.ShowHidden = false
	fp.ShowPermissions = false // Permissions are not useful for save/open and consume space
	fp.ShowSize = true
	fp.SetHeight(15)

	// Apply themed styling to file picker
	fp.Styles.Directory = lipgloss.NewStyle().
		Foreground(theme.FilePickerDir).Bold(true)
	fp.Styles.File = lipgloss.NewStyle().
		Foreground(theme.FilePickerFile)
	fp.Styles.Selected = lipgloss.NewStyle().
		Foreground(theme.FilePickerSelected).Bold(true)
	fp.Styles.Cursor = lipgloss.NewStyle().
		Foreground(theme.FilePickerCursor)

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
