package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	tea "github.com/charmbracelet/bubbletea"
)

func TestDetailViewportUsesPagerKeysWithoutConsumingListArrows(t *testing.T) {
	t.Parallel()
	viewport := NewDetailViewport()
	viewport.SetSize(30, 3)
	lines := make([]string, 10)
	for i := range lines {
		lines[i] = fmt.Sprintf("detail line %d", i+1)
	}
	content := strings.Join(lines, "\n")
	testutil.ErrorIf(t, !strings.Contains(viewport.View(content), "detail line 1"), "%v", "detail did not start at the top")
	testutil.ErrorIf(t, viewport.Update(tea.KeyMsg{Type: tea.KeyDown}), "%v", "detail consumed the list down key")
	testutil.ErrorIf(t, !viewport.Update(tea.KeyMsg{Type: tea.KeyPgDown}), "%v", "detail did not consume page down")
	testutil.ErrorIf(t, strings.Contains(viewport.View(content), "detail line 1"), "%v", "page down did not move detail content")
}

func TestDetailViewportResetsWhenSelectionContentChanges(t *testing.T) {
	t.Parallel()
	viewport := NewDetailViewport()
	viewport.SetSize(30, 2)
	first := "one\ntwo\nthree\nfour"
	viewport.View(first)
	viewport.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	testutil.ErrorIf(t, !strings.Contains(viewport.View("new first\nnew second\nnew third"), "new first"), "%v", "new detail content did not reset to the top")
}
