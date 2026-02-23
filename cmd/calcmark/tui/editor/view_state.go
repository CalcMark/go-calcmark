package editor

// view_state.go — View state computation methods.
// These methods compute state that the View() method consumes.

import (
	"fmt"

	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/components"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/tui/geometry"
	"github.com/CalcMark/go-calcmark/format/display"
	"github.com/CalcMark/go-calcmark/spec/document"
)

// computeGlobalsHeight returns the visual height of the globals panel + separator.
// Must exactly mirror renderGlobalsPanel logic to avoid mismatches.
func (m Model) computeGlobalsHeight() int {
	globalsHeight := 1 // collapsed state (always 1 line)
	if m.globalsExpanded {
		if m.frontmatterErr != nil {
			globalsHeight = 2 // header + error detail
		} else if m.getGlobalsCount() == 0 {
			globalsHeight = 2 // header + "(no globals defined)"
		} else {
			globalsHeight = 1 + m.getGlobalsCount()
		}
	}
	globalsHeight++ // +1 for separator line
	return globalsHeight
}

// getGlobalsCount returns the number of global variables.
func (m *Model) getGlobalsCount() int {
	fm := m.doc.GetFrontmatter()
	if fm == nil {
		return 0
	}
	return len(fm.Globals) + len(fm.Exchange)
}

// GetStatusBarState returns state for the status bar.
func (m *Model) GetStatusBarState() components.StatusBarState {
	// Note: mode is an internal implementation detail and not shown to users
	// Status bar shows minimal hints - full command list discoverable via Ctrl+H

	hints := ""
	switch m.mode {
	case StateDefault:
		// Minimal hints - other commands discoverable via Ctrl+H
		hints = "Ctrl+Q quit | Ctrl+H help"
	case StateCommandMenu:
		hints = "Enter select | Esc close"
	case StateHelp:
		hints = "↑↓ navigate | Enter execute | Esc close"
	case StateFilePicker:
		if m.filePickerPurpose == PickerForOpen {
			if m.filePickerFocus == FocusFilename {
				hints = "Enter open | Esc cancel"
			} else {
				hints = "up/down navigate | Tab filename | Esc cancel"
			}
		} else {
			if m.filePickerFocus == FocusFilename {
				hints = "Enter save | Esc cancel"
			} else {
				hints = "up/down navigate | Tab filename | Esc cancel"
			}
		}
	case StateExport:
		hints = "Tab switch | Enter export | Esc cancel"
	case StateSavePrompt:
		hints = "y/n/c"
	}

	return components.StatusBarState{
		Filename:       m.filepath,
		Line:           m.cursorLine + 1,
		Column:         m.cursorCol + 1, // 1-indexed for user display
		TotalLines:     m.TotalLines(),
		CalcCount:      m.CalcBlockCount(),
		Modified:       m.modified,
		Mode:           "", // Mode is internal - not shown to users
		Hints:          hints,
		StatusMsg:      m.statusMsg,
		StatusIsErr:    m.statusIsErr,
		EvalInProgress: m.userIsTyping, // userIsTyping tracks debounce state
	}
}

// GetPinnedPanelState returns state for the pinned panel.
func (m *Model) GetPinnedPanelState(height int) components.PinnedPanelState {
	vars := m.collectPinnedVariables()
	return components.PinnedPanelState{
		Variables: vars,
		ScrollY:   0,
		Height:    height,
	}
}

// collectPinnedVariables gathers pinned variables for display.
func (m *Model) collectPinnedVariables() []components.PinnedVar {
	var result []components.PinnedVar
	seen := make(map[string]bool)

	// Track frontmatter variables
	fmVars := make(map[string]bool)
	if fm := m.doc.GetFrontmatter(); fm != nil {
		for name := range fm.Globals {
			fmVars[name] = true
		}
	}

	// Collect in document order
	for _, node := range m.doc.GetBlocks() {
		if calcBlock, ok := node.Block.(*document.CalcBlock); ok {
			for _, varName := range calcBlock.Variables() {
				if !m.pinnedVars[varName] || seen[varName] {
					continue
				}
				seen[varName] = true

				valueStr := "?"
				if m.eval != nil {
					env := m.eval.GetEnvironment()
					if val, ok := env.Get(varName); ok {
						valueStr = display.Format(val)
					}
				}

				result = append(result, components.PinnedVar{
					Name:          varName,
					Value:         valueStr,
					Changed:       m.changedVars[varName],
					IsFrontmatter: fmVars[varName],
				})
			}
		}
	}

	return result
}

// GetGlobalsPanelState returns state for the globals panel.
func (m *Model) GetGlobalsPanelState() components.GlobalsPanelState {
	var globals []components.GlobalVar
	var errMsg string

	if m.frontmatterErr != nil {
		errMsg = m.frontmatterErr.Error()
	}

	fm := m.doc.GetFrontmatter()
	if fm != nil {
		for _, name := range fm.GlobalKeys() {
			globals = append(globals, components.GlobalVar{
				Name:       name,
				Value:      fmt.Sprintf("%v", fm.Globals[name]),
				IsExchange: false,
			})
		}
		for _, name := range fm.ExchangeKeys() {
			globals = append(globals, components.GlobalVar{
				Name:       name,
				Value:      fm.Exchange[name].StringFixed(4),
				IsExchange: true,
			})
		}
	}

	return components.GlobalsPanelState{
		Globals:    globals,
		Expanded:   m.globalsExpanded,
		FocusIndex: m.globalsFocusIdx,
		Focused:    m.mode == StateGlobals,
		Error:      errMsg,
	}
}

// computeCacheKey computes a cache key from current model state.
func (m *Model) computeCacheKey() alignedCacheKey {
	// Simple hash of content - just use length and first/last chars for speed
	// A proper implementation would use a real hash, but this catches most changes
	lines := m.GetLines()
	var contentHash uint64
	for i, line := range lines {
		contentHash ^= uint64(len(line)) << (uint(i%8) * 8)
		if len(line) > 0 {
			contentHash ^= uint64(line[0]) << 32
			contentHash ^= uint64(line[len(line)-1]) << 40
		}
	}

	return alignedCacheKey{
		contentHash: contentHash,
		cursorLine:  m.cursorLine,
		previewMode: m.previewMode,
		totalLines:  len(lines),
		editBuf:     m.editBuf, // Include editBuf so cache updates while typing
	}
}

// GetAlignedModel returns the cached aligned model, computing it if necessary.
// This is the single source of truth for visual line alignment.
// The cache is automatically invalidated when inputs change.
func (m *Model) GetAlignedModel(sourceWidth, previewWidth int) *AlignedModel {
	currentKey := m.computeCacheKey()

	// Check if cache is valid: same key and same widths
	if m.alignedCache != nil &&
		m.alignedCacheKey == currentKey &&
		m.alignedCacheWidths[0] == sourceWidth &&
		m.alignedCacheWidths[1] == previewWidth {
		return m.alignedCache
	}

	// Cache miss - recompute
	// Calculate content width for source pane (accounting for line numbers)
	lineNumWidth := 4
	sourceContentWidth := max(sourceWidth-lineNumWidth-2, 10)

	input := AlignedModelInput{
		Lines:              m.GetLines(),
		Results:            m.GetLineResults(),
		SourceContentWidth: sourceContentWidth,
		PreviewWidth:       previewWidth,
		CursorLine:         m.cursorLine,
		PreviewMode:        m.previewMode,
		EditBuf:            m.editBuf,
		EditBufLine:        m.cursorLine, // EditBuf applies to current cursor line
	}

	// Compute with render functions that match view.go behavior
	aligned := ComputeAlignedModel(input, m.renderCalcLine, func(line string, width int) []string {
		mdRenderer, _ := NewMarkdownRenderer(width)
		if mdRenderer != nil {
			return mdRenderer.RenderLine(line)
		}
		return geometry.WrapText(line, width)
	})

	// Update cache
	m.alignedCache = &aligned
	m.alignedCacheKey = currentKey
	m.alignedCacheWidths = [2]int{sourceWidth, previewWidth}

	return m.alignedCache
}

// InvalidateAlignedCache explicitly invalidates the cache.
// This is called on key presses, but the cache will also auto-invalidate
// when computeCacheKey() detects changed inputs.
func (m *Model) InvalidateAlignedCache() {
	m.alignedCache = nil
}

// GetAutocompleteState returns the current autocomplete state for rendering.
func (m Model) GetAutocompleteState() components.AutosuggestState {
	return m.autocompleteState
}
