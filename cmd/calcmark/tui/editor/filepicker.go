package editor

import (
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/lipgloss"
)

// FilePickerMode distinguishes between browsing and typing a new filename.
type FilePickerMode int

const (
	ModePickerBrowse  FilePickerMode = iota // Browsing directories/files
	ModePickerNewFile                       // Typing new filename
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
	fp.DirAllowed = true         // Allow navigating into directories
	fp.FileAllowed = true        // Allow selecting files (for overwrite)
	fp.ShowHidden = false
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
