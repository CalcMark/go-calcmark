package geometry

// PopupPosition represents the calculated screen position for a popup.
type PopupPosition struct {
	Row     int  // Screen row (0-indexed from top)
	Col     int  // Screen column (0-indexed from left)
	Below   bool // True if popup is below cursor, false if above
	MaxH    int  // Maximum height available for popup
	Clipped bool // True if position was adjusted to fit screen
}

// PopupBounds describes the constraints for popup positioning.
type PopupBounds struct {
	ScreenWidth   int // Total screen width
	ScreenHeight  int // Total screen height
	ContentTop    int // First row of content area (after headers)
	ContentBottom int // Last row of content area (before footer)
}

// CalculatePopupPosition computes where to place a popup near the cursor.
// This is a pure function: given cursor position, viewport, and bounds,
// it returns the optimal popup position.
//
// Parameters:
//   - cursorRow: visual row of cursor within content area (0-indexed)
//   - cursorCol: visual column of cursor (0-indexed)
//   - scrollOffset: how many lines are scrolled up
//   - popupWidth: desired popup width
//   - popupHeight: desired popup height (may be clipped)
//   - bounds: screen boundary constraints
//
// The popup prefers to appear below the cursor, but flips above if there's
// more space above and not enough below.
func CalculatePopupPosition(
	cursorRow, cursorCol, scrollOffset int,
	popupWidth, popupHeight int,
	bounds PopupBounds,
) PopupPosition {
	// Calculate visual cursor position on screen
	// cursorRow is already the visual row within the content area
	screenRow := bounds.ContentTop + cursorRow - scrollOffset + 1 // +1 to place below cursor

	// Calculate available space above and below cursor
	spaceBelow := bounds.ContentBottom - screenRow
	spaceAbove := screenRow - bounds.ContentTop - 1 // -1 because cursor takes 1 row

	// Decide whether to place above or below
	below := true
	maxHeight := spaceBelow

	if popupHeight > spaceBelow && spaceAbove > spaceBelow {
		// Not enough space below and more space above - flip to above
		below = false
		maxHeight = spaceAbove
		screenRow = screenRow - popupHeight - 1 // Position above cursor
	}

	// Clip height to available space
	actualHeight := max(1, min(popupHeight, maxHeight))

	// Calculate horizontal position - try to align with cursor
	screenCol := cursorCol
	// Ensure popup doesn't go off right edge
	if screenCol+popupWidth > bounds.ScreenWidth {
		screenCol = bounds.ScreenWidth - popupWidth
	}
	// Ensure popup doesn't go off left edge
	if screenCol < 0 {
		screenCol = 0
	}

	// Ensure row is within bounds
	clipped := false
	if screenRow < bounds.ContentTop {
		screenRow = bounds.ContentTop
		clipped = true
	}
	if screenRow+actualHeight > bounds.ContentBottom {
		screenRow = bounds.ContentBottom - actualHeight
		clipped = true
	}

	return PopupPosition{
		Row:     screenRow,
		Col:     screenCol,
		Below:   below,
		MaxH:    actualHeight,
		Clipped: clipped,
	}
}

// PopupDimensions calculates ideal popup dimensions based on content.
func PopupDimensions(suggestions []string, maxWidth, maxHeight int) (width, height int) {
	// Calculate width based on longest suggestion
	width = 20 // minimum width
	for _, s := range suggestions {
		w := StringWidth(s)
		if w+4 > width { // +4 for padding and selection indicator
			width = w + 4
		}
	}

	// Cap at max width
	width = min(width, maxWidth)

	// Height is number of items, capped at max
	height = max(1, min(len(suggestions), maxHeight))

	return width, height
}
