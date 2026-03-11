package editor

// mode_transitions.go — Centralized mode transition methods.
//
// All mode changes (m.mode = StateXxx) should go through these methods
// to ensure associated fields are consistently reset. This prevents bugs
// where forgetting to reset a field leaves the editor in an inconsistent state.

import (
	tea "charm.land/bubbletea/v2"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/components"
	implDoc "github.com/CalcMark/go-calcmark/impl/document"
	"github.com/CalcMark/go-calcmark/spec/document"
)

// exitOverlay returns to StateDefault, resetting overlay-specific fields.
// Handles: Help, CommandMenu, Export, ShareTo, OpenFrom, SavePrompt, Globals, FilePicker.
func (m *Model) exitOverlay() {
	m.mode = StateDefault
	m.pendingSaveAction = PendingNone
	m.newFileName = ""
	m.shareDescription = ""
	m.shareVisibility = 0
	m.shareField = 0
	m.openFromInput = ""
}

// enterShareTo opens the Share To Gist overlay.
func (m *Model) enterShareTo() {
	m.mode = StateShareTo
	m.shareVisibility = 0 // Default: public
	m.shareDescription = ""
	m.shareField = 0 // Start on visibility select
}

// enterOpenFrom opens the Open From Gist overlay.
func (m *Model) enterOpenFrom() {
	m.mode = StateOpenFrom
	m.openFromInput = ""
}

// enterCommandMenu opens the command menu overlay.
func (m *Model) enterCommandMenu() {
	m.mode = StateCommandMenu
	m.commandMenuState.Selected = 0
}

// enterHelp opens the help overlay.
func (m *Model) enterHelp() {
	m.mode = StateHelp
	m.helpState = HelpOverlayState{Selected: 0, ScrollOffset: 0}
}

// enterSavePrompt shows the save confirmation dialog.
func (m *Model) enterSavePrompt(action PendingAction, promptMsg string) {
	m.pendingSaveAction = action
	m.mode = StateSavePrompt
	m.statusMsg = promptMsg
}

// enterFilePicker opens the file picker for save/open/export.
func (m *Model) enterFilePicker(purpose FilePickerPurpose, focus FilePickerFocus) tea.Cmd {
	m.filePicker = initFilePicker(purpose)
	m.filePickerFocus = focus
	m.filePickerPurpose = purpose
	m.mode = StateFilePicker
	return m.filePicker.Init()
}

// enterExportMode opens the export format overlay.
func (m *Model) enterExportMode() {
	m.mode = StateExport
	m.exportState = ExportOverlayState{FormatIdx: 0}
	m.statusMsg = ""
}

// exitAutocomplete dismisses autocomplete without inserting.
func (m *Model) exitAutocomplete() {
	m.mode = StateDefault
	m.autocompleteState = components.AutosuggestState{}
}

// resetForNewDocument resets all mutable editor state for a freshly opened document.
// Called by openFile() to prevent stale state from leaking across documents.
func (m *Model) resetForNewDocument(doc *document.Document, eval *implDoc.Evaluator, absPath, content string) {
	m.doc = doc
	m.eval = eval
	m.filepath = absPath
	m.modified = false
	m.savedContent = content

	m.cursorLine = 0
	m.cursorCol = 0
	m.scrollOffset = 0

	m.editBuf = ""
	m.editBufLoaded = false
	m.userIsTyping = false
	m.frontmatterErr = nil
	m.changedBlockIDs = make(map[string]bool)
	m.selectionAnchorLine = -1
	m.selectionAnchorCol = -1

	m.autocompleteState = components.AutosuggestState{}
	m.pendingSaveAction = PendingNone
	m.newFileName = ""

	m.undoManager.Clear()
	if m.renderCache != nil {
		m.renderCache.Clear()
	}

	m.pinnedVars = make(map[string]bool)
	m.changedVars = make(map[string]bool)
	m.autoPinVariables()

	m.transitionToReady()
}
