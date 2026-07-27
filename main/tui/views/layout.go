package views

import "github.com/charmbracelet/lipgloss"

// SplitListDetailWidths returns list and detail widths for split-pane layouts.
// Uses shared defaults to keep list views consistent.
func SplitListDetailWidths(width int) (int, int) {
	if width <= 0 {
		return 0, 0
	}

	listWidth := int(float64(width) * 0.6)
	if listWidth < 32 {
		listWidth = width / 2
	}
	detailWidth := width - listWidth
	if detailWidth < 24 {
		detailWidth = max(width-24, 0)
		listWidth = width - detailWidth
	}

	return listWidth, detailWidth
}

// PaneContentWidth converts an allocated outer width into the width available
// to content after a Lip Gloss style adds padding and borders.
func PaneContentWidth(style lipgloss.Style, width int) int {
	return max(width-style.GetHorizontalFrameSize(), 0)
}

// PaneStyleWidth converts content width to the value expected by Style.Width.
// Lip Gloss includes padding in Width but adds borders outside it.
func PaneStyleWidth(style lipgloss.Style, contentWidth int) int {
	return contentWidth + style.GetPaddingLeft() + style.GetPaddingRight()
}
