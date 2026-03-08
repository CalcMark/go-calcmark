package editor

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/CalcMark/go-calcmark/cmd/calcmark/config/theme"
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
				{Name: "New", Accelerator: "Ctrl+N", Kind: HelpActionable, CommandName: "New"},
				{Name: "Save", Accelerator: "Ctrl+S", Kind: HelpActionable, CommandName: "Save"},
				{Name: "Save As", Accelerator: "", Kind: HelpActionable, CommandName: "Save As"},
				{Name: "Open", Accelerator: "Ctrl+O", Kind: HelpActionable, CommandName: "Open"},
				{Name: "Export", Accelerator: "Ctrl+T", Kind: HelpActionable, CommandName: "Export"},
				{Name: "Quit", Accelerator: "Ctrl+Q", Kind: HelpActionable, CommandName: "Quit"},
			},
		},
		{
			Name: "Edit",
			Items: []HelpItem{
				{Name: "Undo", Accelerator: "Ctrl+Z / ⌘Z", Kind: HelpActionable, CommandName: "Undo"},
				{Name: "Redo", Accelerator: "Ctrl+Y / ⌘⇧Z", Kind: HelpActionable, CommandName: "Redo"},
				{Name: "Delete Line", Accelerator: "Ctrl+K", Kind: HelpActionable, CommandName: "Delete Line"},
				{Name: "Insert Frontmatter", Accelerator: "Ctrl+F", Kind: HelpActionable, CommandName: "Insert Frontmatter"},
				{Name: "New Line", Accelerator: "Enter", Kind: HelpAdvisory},
				{Name: "Backspace", Accelerator: "Bksp", Kind: HelpAdvisory},
				{Name: "Delete Word", Accelerator: "Ctrl+Bksp", Kind: HelpAdvisory},
				{Name: "Select All", Accelerator: "⌘A", Kind: HelpAdvisory},
				{Name: "Copy", Accelerator: "Ctrl+C / ⌘C", Kind: HelpAdvisory},
				{Name: "Cut", Accelerator: "Ctrl+X / ⌘X", Kind: HelpAdvisory},
				{Name: "Paste", Accelerator: "Ctrl+V / ⌘V", Kind: HelpAdvisory},
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
				{Name: "Word Left", Accelerator: "Opt+← / Opt+B", Kind: HelpAdvisory},
				{Name: "Word Right", Accelerator: "Opt+→ / Opt+F", Kind: HelpAdvisory},
				{Name: "Line Start", Accelerator: "⌘← / Ctrl+A / Home", Kind: HelpAdvisory},
				{Name: "Line End", Accelerator: "⌘→ / Ctrl+E / End", Kind: HelpAdvisory},
				{Name: "Page Up", Accelerator: "PgUp", Kind: HelpAdvisory},
				{Name: "Page Down", Accelerator: "PgDn", Kind: HelpAdvisory},
				{Name: "Doc Start", Accelerator: "⌘↑ / Opt+↑", Kind: HelpAdvisory},
				{Name: "Doc End", Accelerator: "⌘↓ / Opt+↓", Kind: HelpAdvisory},
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
func (m Model) handleHelpOverlayKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	indices := actionableIndices()
	if len(indices) == 0 {
		// No actionable items — only Esc closes
		if msg.String() == "esc" {
			m.exitOverlay()
		}
		return m, nil
	}

	switch msg.String() {
	case "esc":
		m.exitOverlay()
		return m, nil

	case "up":
		// Move to previous actionable item
		m.helpState.Selected = prevActionableIndex(indices, m.helpState.Selected-1)
		return m, nil

	case "down":
		// Move to next actionable item
		m.helpState.Selected = nextActionableIndex(indices, m.helpState.Selected+1)
		return m, nil

	case "enter":
		// Execute selected actionable item
		items := flatHelpItems()
		if m.helpState.Selected >= 0 && m.helpState.Selected < len(items) {
			item := items[m.helpState.Selected]
			if item.Kind == HelpActionable && item.CommandName != "" {
				m.exitOverlay()
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

	o := NewOverlayStyle(52)

	var lines []string
	lines = append(lines, o.TopBorder)

	// Title
	lines = append(lines, o.WrapRow(o.PadLine(" CalcMark Help", theme.TextBright, o.ItemBg, true)))

	// Separator
	lines = append(lines, o.SepRow())

	// Track flat index across categories
	flatIdx := 0
	for catIdx, cat := range categories {
		// Category header
		lines = append(lines, o.WrapRow(o.PadLine(fmt.Sprintf(" %s", cat.Name), theme.TextMuted, o.ItemBg, true)))

		for _, item := range cat.Items {
			// Build the display line: prefix + accelerator + name
			var prefix string
			if item.Kind == HelpActionable && flatIdx == selectedIdx {
				prefix = " ▸ "
			} else {
				prefix = "   "
			}

			accel := fmt.Sprintf("%-18s", item.Accelerator)
			content := fmt.Sprintf("%s%s%s", prefix, accel, item.Name)

			if item.Kind == HelpActionable && flatIdx == selectedIdx {
				lines = append(lines, o.WrapRow(o.PadLine(content, theme.PopupSelectedFg, o.SelectedBg, true)))
			} else if item.Kind == HelpActionable {
				lines = append(lines, o.WrapRow(o.PadLine(content, theme.Text, o.ItemBg, false)))
			} else {
				lines = append(lines, o.WrapRow(o.PadLine(content, theme.Hint, o.ItemBg, false)))
			}

			flatIdx++
		}

		// Add empty line between categories (except last)
		if catIdx < len(categories)-1 {
			lines = append(lines, o.WrapRow(o.PadLine("", theme.Hint, o.ItemBg, false)))
		}
	}

	// Separator before hints
	lines = append(lines, o.SepRow())

	// Hints
	lines = append(lines, o.HintRow(" ↑↓ navigate  Enter execute  Esc close"))

	lines = append(lines, o.BottomBorder)
	return strings.Join(lines, "\n")
}
