package views

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
		detailWidth = width - 24
		if detailWidth < 0 {
			detailWidth = 0
		}
		listWidth = width - detailWidth
	}

	return listWidth, detailWidth
}
