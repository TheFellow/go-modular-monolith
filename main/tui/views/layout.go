package views

import (
	sharedtui "github.com/TheFellow/go-modular-monolith/pkg/tui"
	"github.com/charmbracelet/lipgloss"
)

// SplitListDetailWidths returns list and detail widths for split-pane layouts.
// Uses shared defaults to keep list views consistent.
func SplitListDetailWidths(width int) (int, int) {
	return sharedtui.SplitListDetailWidths(width)
}

// PaneContentWidth converts an allocated outer width into the width available
// to content after a Lip Gloss style adds padding and borders.
func PaneContentWidth(style lipgloss.Style, width int) int {
	return sharedtui.PaneContentWidth(style, width)
}

// PaneStyleWidth converts content width to the value expected by Style.Width.
// Lip Gloss includes padding in Width but adds borders outside it.
func PaneStyleWidth(style lipgloss.Style, contentWidth int) int {
	return sharedtui.PaneStyleWidth(style, contentWidth)
}
