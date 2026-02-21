package editor

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HelpItemKind distinguishes actionable items (selectable, executable)
// from advisory items (read-only informational).
type HelpItemKind int

const (
	HelpActionable HelpItemKind = iota // Selectable + executable via Enter
	HelpAdvisory                       // Read-only informational
)

// HelpItem represents a single item in the help overlay.
type HelpItem struct {
	Name        string       // Display name: "Save", "Up/Down"
	Accelerator string       // Key hint: "Ctrl+S", "↑↓"
	Kind        HelpItemKind // Actionable or advisory
	CommandName string       // For actionable: maps to executeCommandByName()
}

// HelpCategory groups related help items under a section header.
type HelpCategory struct {
	Name  string
	Items []HelpItem
}

// HelpOverlayState tracks navigation within the help overlay.
type HelpOverlayState struct {
	Selected     int // Index into the flat list of ALL items (actionable ones are selectable)
	ScrollOffset int // Vertical scroll offset for the overlay content
}

// helpCategories returns the organized help content.
// Actionable items can be executed via Enter; advisory items are read-only.
func helpCategories() []HelpCategory {
	return []HelpCategory{
		{
			Name: "File",
			Items: []HelpItem{
				{Name: "Save", Accelerator: "Ctrl+S", Kind: HelpActionable, CommandName: "Save"},
				{Name: "Save As", Accelerator: "", Kind: HelpActionable, CommandName: "Save As"},
				{Name: "Open", Accelerator: "Ctrl+O", Kind: HelpActionable, CommandName: "Open"},
				{Name: "Export", Accelerator: "Ctrl+E", Kind: HelpActionable, CommandName: "Export"},
				{Name: "Quit", Accelerator: "Ctrl+Q", Kind: HelpActionable, CommandName: "Quit"},
			},
		},
		{
			Name: "Edit",
			Items: []HelpItem{
				{Name: "Undo", Accelerator: "Ctrl+Z", Kind: HelpActionable, CommandName: "Undo"},
				{Name: "Redo", Accelerator: "Ctrl+Y", Kind: HelpActionable, CommandName: "Redo"},
				{Name: "Delete Line", Accelerator: "Ctrl+D", Kind: HelpActionable, CommandName: "Delete Line"},
				{Name: "New Line", Accelerator: "Enter", Kind: HelpAdvisory},
				{Name: "Backspace", Accelerator: "Bksp", Kind: HelpAdvisory},
				{Name: "Delete Word", Accelerator: "Ctrl+Bksp", Kind: HelpAdvisory},
				{Name: "Select All", Accelerator: "Ctrl+A", Kind: HelpAdvisory},
				{Name: "Copy", Accelerator: "Ctrl+C", Kind: HelpAdvisory},
				{Name: "Cut", Accelerator: "Ctrl+X", Kind: HelpAdvisory},
				{Name: "Paste", Accelerator: "Ctrl+V", Kind: HelpAdvisory},
			},
		},
		{
			Name: "View",
			Items: []HelpItem{
				{Name: "Toggle Preview", Accelerator: "Ctrl+P", Kind: HelpActionable, CommandName: "Toggle Preview"},
			},
		},
		{
			Name: "Navigation",
			Items: []HelpItem{
				{Name: "Up / Down", Accelerator: "↑ ↓", Kind: HelpAdvisory},
				{Name: "Left / Right", Accelerator: "← →", Kind: HelpAdvisory},
				{Name: "Word Left", Accelerator: "Ctrl+←", Kind: HelpAdvisory},
				{Name: "Word Right", Accelerator: "Ctrl+→", Kind: HelpAdvisory},
				{Name: "Line Start", Accelerator: "Home", Kind: HelpAdvisory},
				{Name: "Line End", Accelerator: "End", Kind: HelpAdvisory},
				{Name: "Page Up", Accelerator: "PgUp", Kind: HelpAdvisory},
				{Name: "Page Down", Accelerator: "PgDn", Kind: HelpAdvisory},
				{Name: "Doc Start", Accelerator: "Ctrl+Home", Kind: HelpAdvisory},
				{Name: "Doc End", Accelerator: "Ctrl+End", Kind: HelpAdvisory},
			},
		},
		{
			Name: "Other",
			Items: []HelpItem{
				{Name: "Help Toggle", Accelerator: "Ctrl+H / F1", Kind: HelpAdvisory},
				{Name: "Command Menu", Accelerator: "Ctrl+H / F1", Kind: HelpAdvisory},
				{Name: "Cancel / Close", Accelerator: "Esc", Kind: HelpAdvisory},
			},
		},
	}
}

// flatHelpItems returns all help items as a flat list.
func flatHelpItems() []HelpItem {
	var items []HelpItem
	for _, cat := range helpCategories() {
		items = append(items, cat.Items...)
	}
	return items
}

// actionableIndices returns the indices of actionable items in the flat list.
func actionableIndices() []int {
	var indices []int
	items := flatHelpItems()
	for i, item := range items {
		if item.Kind == HelpActionable {
			indices = append(indices, i)
		}
	}
	return indices
}

// nextActionableIndex finds the next actionable index at or after pos.
func nextActionableIndex(indices []int, pos int) int {
	for _, idx := range indices {
		if idx >= pos {
			return idx
		}
	}
	// Wrap to first
	if len(indices) > 0 {
		return indices[0]
	}
	return 0
}

// prevActionableIndex finds the previous actionable index at or before pos.
func prevActionableIndex(indices []int, pos int) int {
	for i := len(indices) - 1; i >= 0; i-- {
		if indices[i] <= pos {
			return indices[i]
		}
	}
	// Wrap to last
	if len(indices) > 0 {
		return indices[len(indices)-1]
	}
	return 0
}

// handleHelpOverlayKey processes keys when the interactive help overlay is visible.
func (m Model) handleHelpOverlayKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	indices := actionableIndices()
	if len(indices) == 0 {
		// No actionable items — only Esc closes
		if msg.Type == tea.KeyEsc {
			m.mode = StateDefault
		}
		return m, nil
	}

	switch msg.Type {
	case tea.KeyEsc:
		m.mode = StateDefault
		return m, nil

	case tea.KeyUp:
		// Move to previous actionable item
		m.helpState.Selected = prevActionableIndex(indices, m.helpState.Selected-1)
		return m, nil

	case tea.KeyDown:
		// Move to next actionable item
		m.helpState.Selected = nextActionableIndex(indices, m.helpState.Selected+1)
		return m, nil

	case tea.KeyEnter:
		// Execute selected actionable item
		items := flatHelpItems()
		if m.helpState.Selected >= 0 && m.helpState.Selected < len(items) {
			item := items[m.helpState.Selected]
			if item.Kind == HelpActionable && item.CommandName != "" {
				m.mode = StateDefault
				return m.executeCommandByName(item.CommandName)
			}
		}
		return m, nil
	}

	return m, nil
}

// renderHelpOverlay renders a centered help panel with interactive navigation.
// Actionable items are highlighted and selectable via cursor; advisory items are dimmed.
func (m Model) renderHelpOverlay() string {
	categories := helpCategories()
	selectedIdx := m.helpState.Selected

	innerWidth := 52

	// Border and background colors (shared palette across all overlays)
	borderFg := lipgloss.Color("#5C5C5C")
	borderStyle := lipgloss.NewStyle().Foreground(borderFg)
	itemBg := lipgloss.Color("#1E1E1E")
	selectedBg := lipgloss.Color("#4A90D9")

	topBorder := borderStyle.Render("╭" + strings.Repeat("─", innerWidth) + "╮")
	bottomBorder := borderStyle.Render("╰" + strings.Repeat("─", innerWidth) + "╯")
	leftBorder := borderStyle.Render("│")
	rightBorder := borderStyle.Render("│")
	sepLine := borderStyle.Render(strings.Repeat("─", innerWidth))

	// pad creates a line padded to innerWidth with the given foreground/background
	pad := func(content string, fg, bg lipgloss.Color, bold bool) string {
		style := lipgloss.NewStyle().Foreground(fg).Background(bg)
		if bold {
			style = style.Bold(true)
		}
		visualWidth := lipgloss.Width(content)
		if visualWidth < innerWidth {
			content += strings.Repeat(" ", innerWidth-visualWidth)
		}
		return style.Render(content)
	}

	var lines []string
	lines = append(lines, topBorder)

	// Title
	lines = append(lines, leftBorder+pad(" CalcMark Help", "#FFFFFF", itemBg, true)+rightBorder)

	// Separator
	lines = append(lines, leftBorder+sepLine+rightBorder)

	// Track flat index across categories
	flatIdx := 0
	for catIdx, cat := range categories {
		// Category header
		lines = append(lines, leftBorder+pad(fmt.Sprintf(" %s", cat.Name), "#AAAAAA", itemBg, true)+rightBorder)

		for _, item := range cat.Items {
			// Build the display line: prefix + accelerator + name
			var prefix string
			if item.Kind == HelpActionable && flatIdx == selectedIdx {
				prefix = " ▸ "
			} else {
				prefix = "   "
			}

			accel := fmt.Sprintf("%-14s", item.Accelerator)
			content := fmt.Sprintf("%s%s%s", prefix, accel, item.Name)

			if item.Kind == HelpActionable && flatIdx == selectedIdx {
				lines = append(lines, leftBorder+pad(content, "#FFFFFF", selectedBg, true)+rightBorder)
			} else if item.Kind == HelpActionable {
				lines = append(lines, leftBorder+pad(content, "#CCCCCC", itemBg, false)+rightBorder)
			} else {
				lines = append(lines, leftBorder+pad(content, "#777777", itemBg, false)+rightBorder)
			}

			flatIdx++
		}

		// Add empty line between categories (except last)
		if catIdx < len(categories)-1 {
			lines = append(lines, leftBorder+pad("", "#777777", itemBg, false)+rightBorder)
		}
	}

	// Separator before hints
	lines = append(lines, leftBorder+sepLine+rightBorder)

	// Hints
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Background(itemBg).
		Italic(true)
	hint := " ↑↓ navigate  Enter execute  Esc close"
	visualWidth := lipgloss.Width(hint)
	if visualWidth < innerWidth {
		hint += strings.Repeat(" ", innerWidth-visualWidth)
	}
	lines = append(lines, leftBorder+hintStyle.Render(hint)+rightBorder)

	lines = append(lines, bottomBorder)
	return strings.Join(lines, "\n")
}
