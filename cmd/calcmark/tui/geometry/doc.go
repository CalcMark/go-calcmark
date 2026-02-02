// Package geometry provides pure layout computation for two-column terminal rendering.
// It has zero dependencies on TUI frameworks (lipgloss, bubbletea, bubbles).
// All functions are pure: same inputs always produce same outputs.
//
// This package is the foundation for the CalcMark editor's two-column layout.
// It handles text wrapping with unicode-aware width calculation and computes
// row geometry for aligning source and result columns.
package geometry
