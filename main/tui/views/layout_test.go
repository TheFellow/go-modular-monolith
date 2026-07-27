package views

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/charmbracelet/lipgloss"
)

func TestPaneContentWidthsFitAllocatedViewport(t *testing.T) {
	t.Parallel()
	listStyle := lipgloss.NewStyle().Padding(0, 1)
	detailStyle := lipgloss.NewStyle().Padding(0, 1).BorderLeft(true)
	listOuter, detailOuter := SplitListDetailWidths(120)
	listContent := PaneContentWidth(listStyle, listOuter)
	detailContent := PaneContentWidth(detailStyle, detailOuter)

	list := listStyle.Width(PaneStyleWidth(listStyle, listContent)).Render("")
	detail := detailStyle.Width(PaneStyleWidth(detailStyle, detailContent)).Render("")
	testutil.Equals(t, lipgloss.Width(list)+lipgloss.Width(detail), 120)
}
