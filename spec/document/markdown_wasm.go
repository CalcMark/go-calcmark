//go:build wasm
// +build wasm

package document

// Render is a no-op in WASM builds.
// Markdown rendering should be done client-side in JavaScript.
// This method exists to maintain API compatibility but will panic if called.
func (tb *TextBlock) Render() string {
	panic("TextBlock.Render() is not available in WASM builds. Use client-side markdown rendering (e.g., marked.js)")
}

// renderMarkdown is not available in WASM builds.
func renderMarkdown(source string) string {
	return "" // Not used
}
