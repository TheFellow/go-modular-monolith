package tui

import "github.com/charmbracelet/lipgloss"

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

func PaneContentWidth(style lipgloss.Style, width int) int {
	return max(width-style.GetHorizontalFrameSize(), 0)
}

func PaneStyleWidth(style lipgloss.Style, contentWidth int) int {
	return contentWidth + style.GetPaddingLeft() + style.GetPaddingRight()
}
