package editor

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/config"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/geometry"
	"github.com/CalcMark/go-calcmark/format"
	implDoc "github.com/CalcMark/go-calcmark/impl/document"
	specDoc "github.com/CalcMark/go-calcmark/spec/document"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ModelV2 integrates textarea while preserving alignment architecture
type ModelV2 struct {
	// Source editing with textarea
	sourceArea textarea.Model

	// Document state (KEEP existing)
	doc      *specDoc.Document
	eval     *implDoc.Evaluator
	filepath string
	modified bool

	// UI dimensions
	width  int
	height int

	// Theme
	theme config.ThemeConfig

	// App state
	quitting bool
}

// NewModelV2 creates a new textarea-based editor
func NewModelV2(width, height int, theme config.ThemeConfig, filepath string) *ModelV2 {
	// Calculate dimensions
	// Line numbers are rendered by textarea via SetPromptFunc (4 chars wide)
	// Separator between panes: " │ " (3 chars)
	separatorWidth := 3
	availableWidth := width - separatorWidth
	paneWidth := availableWidth / 2
	paneHeight := height - 4

	// Initialize textarea for source
	source := textarea.New()
	source.Focus()
	source.SetWidth(paneWidth)
	source.SetHeight(paneHeight)
	source.ShowLineNumbers = false
	source.Prompt = ""
	source.EndOfBufferCharacter = '~' // Show tildes for lines beyond document

	// Ensure no limits on content
	source.CharLimit = 0 // No character limit
	source.MaxHeight = 0 // No max height (unlimited lines)
	source.MaxWidth = 0  // No max width

	// Note: textarea doesn't have a direct "disable wrap" setting
	// We handle wrapping ourselves in the View layer to maintain proper line number alignment

	// Set up line number rendering that scrolls with textarea
	// This will be updated once we know the actual line count
	source.SetPromptFunc(4, func(lineIdx int) string {
		// Default: show line numbers for all lines
		// This will be replaced in the returned ModelV2
		return fmt.Sprintf("%4d", lineIdx+1)
	})

	// Configure styles - FULL WIDTH LINE HIGHLIGHTING AND BACKGROUNDS
	source.FocusedStyle.Base = lipgloss.NewStyle().
		Background(lipgloss.Color(theme.SourcePaneBg))

	source.FocusedStyle.Text = lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.SourceText)).
		Background(lipgloss.Color(theme.SourcePaneBg))

	source.FocusedStyle.CursorLine = lipgloss.NewStyle().
		Background(lipgloss.Color(theme.EditLineBg)).
		Foreground(lipgloss.Color(theme.EditLineFg))

	source.FocusedStyle.EndOfBuffer = lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Dimmed)).
		Background(lipgloss.Color(theme.SourcePaneBg))

	// Style the line number prompt
	source.FocusedStyle.Prompt = lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Dimmed)).
		Background(lipgloss.Color(theme.SourcePaneBg))

	source.Cursor.Style = lipgloss.NewStyle().
		Background(lipgloss.Color(theme.CursorBg)).
		Foreground(lipgloss.Color(theme.CursorFg))

	// Initialize evaluator
	eval := implDoc.NewEvaluator()

	// Create empty document
	doc, _ := specDoc.NewDocument("")

	return &ModelV2{
		sourceArea: source,
		doc:        doc,
		eval:       eval,
		filepath:   filepath,
		width:      width,
		height:     height,
		theme:      theme,
	}
}

// Init implements tea.Model
func (m *ModelV2) Init() tea.Cmd {
	// Install initial SetPromptFunc
	m.installPromptFunc()
	return textarea.Blink
}

// Update handles input
func (m *ModelV2) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case saveResultMsg:
		if msg.success {
			m.modified = false
		}
		// TODO: Show save status message
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "ctrl+q":
			m.quitting = true
			return m, tea.Quit

		case "ctrl+s":
			if m.filepath != "" {
				return m, m.saveFile()
			}
			// TODO: Handle save-as for new files
			return m, nil

		default:
			// Store old value to detect actual content changes
			oldValue := m.sourceArea.Value()

			// Update source textarea (handles arrow keys, editing, etc.)
			var cmd tea.Cmd
			m.sourceArea, cmd = m.sourceArea.Update(msg)
			cmds = append(cmds, cmd)

			// Install SetPromptFunc to track visible lines during next View() call
			m.installPromptFunc()

			// Only sync and mark modified if content actually changed
			// (arrow keys, page up/down, etc. don't change content)
			if m.sourceArea.Value() != oldValue {
				// Sync back to document and re-evaluate
				m.syncDocumentFromTextarea()

				// Mark as modified
				m.modified = true
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizePanes()
	}

	return m, tea.Batch(cmds...)
}

// View renders the editor
func (m *ModelV2) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Get cursor position from textarea
	cursorLine := m.sourceArea.Line()

	// Calculate source pane dimensions
	sourcePaneWidth := m.sourceArea.Width()
	sourcePaneHeight := m.sourceArea.Height()

	// Determine which logical lines are visible
	// For now, start from line 0 (we'll add viewport scrolling later)
	firstVisibleLogicalLine := 0
	lastVisibleLogicalLine := len(strings.Split(m.sourceArea.Value(), "\n")) - 1

	// Render source pane with proper wrapping and line numbers
	sourceView, actualFirstLine, actualLastLine := m.renderSourcePaneWindow(
		sourcePaneWidth, sourcePaneHeight, cursorLine,
		firstVisibleLogicalLine, lastVisibleLogicalLine)

	// Compute output aligned to source using the visible logical line range
	outputView := m.renderOutputForRange(actualFirstLine, actualLastLine)

	// Combine panes side-by-side using SideBySide renderer
	// This prevents truncation by manually padding each line to exact width
	separatorWidth := 3 // " │ " separator
	sourceWidth := m.sourceArea.Width()
	previewWidth := m.sourceArea.Width()

	// Add separator to source pane
	sourceLines := strings.Split(sourceView, "\n")
	sepStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.Muted)).
		Background(lipgloss.Color(m.theme.SourcePaneBg))
	for i, line := range sourceLines {
		sourceLines[i] = line + sepStyle.Render(" │ ")
	}
	sourceWithSep := strings.Join(sourceLines, "\n")

	// Use SideBySide to render without truncation
	sbs := NewSideBySide(
		sourceWidth+separatorWidth,
		previewWidth,
		lipgloss.Color(m.theme.SourcePaneBg),
		lipgloss.Color(m.theme.SourcePaneBg),
	)
	panes := sbs.Render(sourceWithSep, outputView)

	// Add header
	header := m.renderHeader()

	content := lipgloss.JoinVertical(lipgloss.Left, header, panes)

	// Apply overall background to prevent terminal bleed-through
	// Don't set explicit height - let content determine it
	appStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(m.theme.SourcePaneBg)).
		Width(m.width)

	return appStyle.Render(content)
}

// renderSourcePaneWindow renders the source pane with proper line numbers and wrapping
// Line numbers appear only on the first visual line of each logical line
// Returns: rendered view, first logical line shown, last logical line shown
func (m *ModelV2) renderSourcePaneWindow(paneWidth, paneHeight int, cursorLine int, startLogicalLine, endLogicalLine int) (string, int, int) {
	sourceLines := strings.Split(m.sourceArea.Value(), "\n")

	// Account for line number gutter (4 chars wide: "   1" or " 123")
	gutterWidth := 4
	contentWidth := paneWidth - gutterWidth

	var visualLines []string
	visualLineCount := 0
	actualFirstLine := -1
	actualLastLine := -1

	for logicalLineIdx := startLogicalLine; logicalLineIdx <= endLogicalLine && logicalLineIdx < len(sourceLines); logicalLineIdx++ {
		if visualLineCount >= paneHeight {
			break
		}

		if actualFirstLine == -1 {
			actualFirstLine = logicalLineIdx
		}
		actualLastLine = logicalLineIdx

		lineText := sourceLines[logicalLineIdx]
		isCursorLine := (logicalLineIdx == cursorLine)

		// Wrap this logical line
		wrappedLines := geometry.WrapText(lineText, contentWidth)
		if len(wrappedLines) == 0 {
			wrappedLines = []string{""} // Empty line
		}

		// Render each wrapped visual line
		for wrapIdx, wrappedText := range wrappedLines {
			if visualLineCount >= paneHeight {
				break
			}

			// Line number: show only on first visual line of this logical line
			var lineNum string
			if wrapIdx == 0 {
				lineNum = fmt.Sprintf("%4d", logicalLineIdx+1) // 1-indexed
			} else {
				lineNum = "    " // Blank gutter for wrapped continuation
			}

			// Style the line (highlight cursor line)
			lineStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color(m.theme.SourceText)).
				Background(lipgloss.Color(m.theme.SourcePaneBg))

			if isCursorLine {
				lineStyle = lineStyle.
					Background(lipgloss.Color(m.theme.EditLineBg)).
					Foreground(lipgloss.Color(m.theme.EditLineFg))
			}

			gutterStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color(m.theme.Dimmed)).
				Background(lipgloss.Color(m.theme.SourcePaneBg))

			if isCursorLine {
				gutterStyle = gutterStyle.Background(lipgloss.Color(m.theme.EditLineBg))
			}

			// Combine gutter + content
			visualLine := gutterStyle.Render(lineNum) + lineStyle.Render(wrappedText)
			visualLines = append(visualLines, visualLine)
			visualLineCount++
		}
	}

	// Pad to pane height with tildes (like vim)
	tildeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.Dimmed)).
		Background(lipgloss.Color(m.theme.SourcePaneBg))

	for visualLineCount < paneHeight {
		visualLines = append(visualLines, tildeStyle.Render("   ~"))
		visualLineCount++
	}

	if actualFirstLine == -1 {
		actualFirstLine = 0
	}
	if actualLastLine == -1 {
		actualLastLine = 0
	}

	return strings.Join(visualLines, "\n"), actualFirstLine, actualLastLine
}

// installPromptFunc sets up the SetPromptFunc for line number rendering
// This is called in Update() to ensure it's ready for the next View()
func (m *ModelV2) installPromptFunc() {
	totalLines := m.sourceArea.LineCount()

	m.sourceArea.SetPromptFunc(4, func(lineIdx int) string {
		if lineIdx < totalLines {
			return fmt.Sprintf("%4d", lineIdx+1)
		}
		return "    " // Empty gutter for virtual lines
	})
}

// extractVisibleRange parses the rendered textarea view to extract visible line numbers
// This makes the edit pane the single source of truth for viewport state
func (m *ModelV2) extractVisibleRange(renderedView string) (int, int) {
	lines := strings.Split(renderedView, "\n")

	firstLine := -1
	lastLine := -1

	// Parse each line looking for line numbers in the gutter
	// Format: "   1│content" or " 123│content" (4-char gutter)
	for _, line := range lines {
		// Strip ANSI codes before parsing
		cleanLine := stripAnsi(line)

		if len(cleanLine) < 4 {
			continue
		}

		// Extract the gutter (first 4 characters)
		gutter := cleanLine[:4]

		// Try to parse as a line number
		var lineNum int
		_, err := fmt.Sscanf(strings.TrimSpace(gutter), "%d", &lineNum)
		if err == nil && lineNum > 0 {
			// Convert from 1-indexed display to 0-indexed
			lineIdx := lineNum - 1

			if firstLine == -1 {
				firstLine = lineIdx
			}
			lastLine = lineIdx
		}
	}

	if firstLine == -1 {
		return 0, 0
	}

	return firstLine, lastLine
}

// stripAnsi removes ANSI escape codes from a string
func stripAnsi(s string) string {
	// Simple regex-free approach: skip escape sequences
	var result strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			// Skip until we find the terminating character (letter)
			i += 2
			for i < len(s) && !((s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= 'a' && s[i] <= 'z')) {
				i++
			}
			i++ // Skip the terminating character
		} else {
			result.WriteByte(s[i])
			i++
		}
	}
	return result.String()
}

// syncDocumentFromTextarea updates the document from textarea content
func (m *ModelV2) syncDocumentFromTextarea() {
	content := m.sourceArea.Value()

	// Parse new document
	newDoc, err := specDoc.NewDocument(content)
	if err != nil {
		// Keep old document on parse error
		return
	}

	m.doc = newDoc

	// Re-evaluate
	m.eval = implDoc.NewEvaluator()
	_ = m.eval.Evaluate(m.doc)
}

// renderOutputForRange creates the output pane showing calc results
// Uses aligned visual line pairs to handle wrapping in both panes
// Based on the aligned.go architecture: wraps both panes independently,
// then creates max(source_rows, preview_rows) aligned pairs with padding
func (m *ModelV2) renderOutputForRange(firstVisibleLine, lastVisibleLine int) string {
	sourceLines := strings.Split(m.sourceArea.Value(), "\n")
	sourcePaneWidth := m.sourceArea.Width()
	previewPaneWidth := m.sourceArea.Width() // Same width for now, will adjust
	paneHeight := m.sourceArea.Height()

	totalLines := len(sourceLines)

	// Build aligned visual line pairs for the visible logical line range
	var visualLines []string
	visualLineCount := 0

	for logicalLineIdx := firstVisibleLine; logicalLineIdx <= lastVisibleLine && logicalLineIdx < totalLines; logicalLineIdx++ {
		sourceText := sourceLines[logicalLineIdx]

		// Get calculation result for this logical line
		result := m.getCalcResultForLine(logicalLineIdx, sourceText)

		// Wrap source text to source pane width
		wrappedSourceLines := geometry.WrapText(sourceText, sourcePaneWidth)

		// Wrap preview result to preview pane width
		wrappedPreviewLines := geometry.WrapText(result, previewPaneWidth)

		// Calculate number of aligned visual rows needed
		numSourceRows := len(wrappedSourceLines)
		numPreviewRows := len(wrappedPreviewLines)
		numAlignedRows := numSourceRows
		if numPreviewRows > numAlignedRows {
			numAlignedRows = numPreviewRows
		}

		// Emit aligned visual line pairs
		for rowOffset := 0; rowOffset < numAlignedRows; rowOffset++ {
			var previewLine string
			if rowOffset < numPreviewRows {
				previewLine = wrappedPreviewLines[rowOffset]
			} else {
				// Padding: preview wrapped less than source
				previewLine = ""
			}

			visualLines = append(visualLines, previewLine)
			visualLineCount++

			// Stop if we've filled the pane height
			if visualLineCount >= paneHeight {
				break
			}
		}

		if visualLineCount >= paneHeight {
			break
		}
	}

	// Pad to exact pane height with empty lines
	for visualLineCount < paneHeight {
		visualLines = append(visualLines, "")
		visualLineCount++
	}

	// Join visual lines
	outputText := strings.Join(visualLines, "\n")

	// Style the output
	// NOTE: Do NOT set Width() here - that causes lipgloss to truncate with "..."
	// We handle wrapping manually with geometry.WrapText() above
	outputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.Primary)).
		Background(lipgloss.Color(m.theme.SourcePaneBg))

	return outputStyle.Render(outputText)
}

// getCalcResultForLine extracts calc result for a specific line
func (m *ModelV2) getCalcResultForLine(lineIdx int, lineText string) string {
	trimmed := strings.TrimSpace(lineText)

	// Skip empty lines and non-calc lines
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return ""
	}

	// Check if line has an assignment
	if !strings.Contains(trimmed, "=") {
		return ""
	}

	parts := strings.SplitN(trimmed, "=", 2)
	if len(parts) != 2 {
		return ""
	}

	varName := strings.TrimSpace(parts[0])

	// Get value from evaluator environment
	env := m.eval.GetEnvironment()
	if env == nil {
		return ""
	}

	allVars := env.GetAllVariables()
	if val, ok := allVars[varName]; ok {
		return varName + " → " + val.String()
	}

	return ""
}

// renderHeader creates the file info header
func (m *ModelV2) renderHeader() string {
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(m.theme.Primary)).
		Background(lipgloss.Color(m.theme.SourcePaneBg)).
		Bold(true).
		Width(m.width)

	filename := m.filepath
	if filename == "" {
		filename = "[New Document]"
	}
	if m.modified {
		filename += " [+]"
	}

	return headerStyle.Render(filename)
}

// resizePanes updates dimensions when window resizes
func (m *ModelV2) resizePanes() {
	// Line numbers are rendered by textarea via SetPromptFunc (4 chars wide)
	// Separator between panes: " │ " (3 chars)
	separatorWidth := 3
	availableWidth := m.width - separatorWidth
	paneWidth := availableWidth / 2
	paneHeight := m.height - 4

	m.sourceArea.SetWidth(paneWidth)
	m.sourceArea.SetHeight(paneHeight)
}

// SetValue sets the source content
func (m *ModelV2) SetValue(content string) {
	m.sourceArea.SetValue(content)
	m.syncDocumentFromTextarea()
}

// Value returns the source content
func (m *ModelV2) Value() string {
	return m.sourceArea.Value()
}

// saveFile saves the current content to disk
func (m *ModelV2) saveFile() tea.Cmd {
	return func() tea.Msg {
		content := m.sourceArea.Value()
		err := os.WriteFile(m.filepath, []byte(content), 0644)
		if err != nil {
			return saveResultMsg{err: err}
		}
		return saveResultMsg{success: true}
	}
}

// saveResultMsg reports save operation result
type saveResultMsg struct {
	success bool
	err     error
}

// Quitting returns whether the editor is quitting
func (m ModelV2) Quitting() bool {
	return m.quitting
}

// Document returns the current document
func (m ModelV2) Document() *specDoc.Document {
	return m.doc
}

// NewV2 creates a new ModelV2 from a document
func NewV2(doc *specDoc.Document) *ModelV2 {
	cfg, err := config.Load()
	if err != nil {
		// Fall back to empty theme config if loading fails
		fmt.Fprintf(os.Stderr, "Warning: Failed to load config: %v\n", err)
		cfg = &config.Config{
			TUI: config.TUIConfig{
				Theme: config.ThemeConfig{
					Primary:       "#8BE9FD",
					EditLineBg:    "#44475a",
					EditLineFg:    "#f8f8f2",
					CursorBg:      "#ff79c6",
					CursorFg:      "#282a36",
					Dimmed:        "#6272a4",
					Muted:         "#44475a",
					SourcePaneBg:  "#1C1C1C",
					SourceText:    "#E0E0E0",
					PreviewPaneBg: "#FDFDFB",
				},
			},
		}
	}

	m := NewModelV2(80, 24, cfg.TUI.Theme, "")
	if doc != nil {
		// Serialize document back to source text
		var buf bytes.Buffer
		formatter := &format.CalcMarkFormatter{}
		if err := formatter.Format(&buf, doc, format.Options{}); err == nil {
			m.SetValue(buf.String())
		} else {
			// Fall back to keeping the document as-is
			m.doc = doc
			m.eval = implDoc.NewEvaluator()
			_ = m.eval.Evaluate(m.doc)
		}
	}
	return m
}

// NewV2WithFile creates a new ModelV2 with a filepath
func NewV2WithFile(filepath string, doc *specDoc.Document) *ModelV2 {
	m := NewV2(doc)
	m.filepath = filepath
	return m
}
