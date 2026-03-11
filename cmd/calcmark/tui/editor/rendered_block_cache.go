package editor

import "strings"

// RenderedBlockCache memoizes glamour-rendered TextBlock content for the preview pane.
//
// The cache key is (blockID, content, width). On cache hit, the stored []string
// is returned directly. On miss, the block is rendered through glamour and stored.
//
// Safety: This struct is designed to be held as a *RenderedBlockCache pointer on
// the editor Model. Bubble Tea's View() receives a value copy of Model, so the
// pointer is shared. This is safe because:
//   - Bubble Tea guarantees single-goroutine execution for Update() and View()
//   - The cache is append-only on miss (never deletes during View())
//   - Clear() is only called from Update() (resetForNewDocument)
type RenderedBlockCache struct {
	entries  map[cacheKey]*cacheEntry
	order    []cacheKey // LRU order: most recent at end
	capacity int
	renderer *MarkdownRenderer
	width    int // width the renderer was created for
}

type cacheKey struct {
	blockID string
	content string
	width   int
}

type cacheEntry struct {
	lines []string
}

// NewRenderedBlockCache creates a cache with the given capacity.
func NewRenderedBlockCache(capacity int) *RenderedBlockCache {
	if capacity < 1 {
		capacity = 128
	}
	return &RenderedBlockCache{
		entries:  make(map[cacheKey]*cacheEntry, capacity),
		order:    make([]cacheKey, 0, capacity),
		capacity: capacity,
	}
}

// Render returns glamour-rendered lines for a TextBlock. On cache hit, returns
// stored content. On miss, renders through glamour and caches the result.
//
// interpolatedLines are the TextBlock's InterpolatedSource() lines.
func (c *RenderedBlockCache) Render(blockID string, interpolatedLines []string, width int) []string {
	content := strings.Join(interpolatedLines, "\n")
	key := cacheKey{blockID: blockID, content: content, width: width}

	// Cache hit: move to end of LRU order and return.
	if entry, ok := c.entries[key]; ok {
		c.touch(key)
		return entry.lines
	}

	// Cache miss: render through glamour.
	rendered := c.renderBlock(content, width)

	// Store in cache, evicting oldest if at capacity.
	if len(c.entries) >= c.capacity {
		c.evictOldest()
	}
	c.entries[key] = &cacheEntry{lines: rendered}
	c.order = append(c.order, key)

	return rendered
}

// Clear removes all cached entries. Called on new file open.
func (c *RenderedBlockCache) Clear() {
	c.entries = make(map[cacheKey]*cacheEntry, c.capacity)
	c.order = c.order[:0]
}

// Len returns the number of cached entries.
func (c *RenderedBlockCache) Len() int {
	return len(c.entries)
}

// renderBlock renders a block of markdown text through glamour.
// Uses RenderBlock to preserve blank lines between headings and paragraphs.
// Falls back to raw lines on glamour failure.
func (c *RenderedBlockCache) renderBlock(content string, width int) []string {
	renderer := c.getRenderer(width)
	lines := renderer.RenderBlock(content)
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

// getRenderer returns a cached MarkdownRenderer, creating one if the width changed.
func (c *RenderedBlockCache) getRenderer(width int) *MarkdownRenderer {
	if c.renderer != nil && c.width == width {
		return c.renderer
	}
	r, err := NewMarkdownRenderer(width)
	if err != nil {
		// Fallback: create with a reasonable default.
		r, _ = NewMarkdownRenderer(80)
	}
	c.renderer = r
	c.width = width
	return r
}

// touch moves a key to the end of the LRU order (most recently used).
func (c *RenderedBlockCache) touch(key cacheKey) {
	for i, k := range c.order {
		if k == key {
			c.order = append(c.order[:i], c.order[i+1:]...)
			c.order = append(c.order, key)
			return
		}
	}
}

// evictOldest removes the least recently used entry.
func (c *RenderedBlockCache) evictOldest() {
	if len(c.order) == 0 {
		return
	}
	oldest := c.order[0]
	c.order = c.order[1:]
	delete(c.entries, oldest)
}
