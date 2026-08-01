package tui_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/TheFellow/go-modular-monolith/pkg/testutil"
	"github.com/TheFellow/go-modular-monolith/pkg/toolkits/tui"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestListDetailOwnsStandardLoadingResultAndLayoutStates(t *testing.T) {
	t.Parallel()

	styles := tui.ListViewStyles{
		Selected:   lipgloss.NewStyle().Bold(true),
		ListPane:   lipgloss.NewStyle().Padding(0, 1),
		DetailPane: lipgloss.NewStyle().Padding(0, 1),
		ErrorText:  lipgloss.NewStyle(),
	}
	shell := tui.NewListDetail("Things", "Loading things...", styles)
	shell.SetSize(100, 30)
	if got := shell.View("detail"); !strings.Contains(got, "Loading things") {
		testutil.ErrorIf(t, true, "loading view = %q", got)
	}

	shell.SetResult(nil, errors.New("unavailable"))
	if got := shell.View("detail"); !strings.Contains(got, "unavailable") {
		testutil.ErrorIf(t, true, "error view = %q", got)
	}

	shell.SetResult([]list.Item{tui.NewListItem(42, "Answer", "A detail", "")}, nil)
	got := shell.View("selected detail")
	for _, want := range []string{"Things", "Answer", "selected detail"} {
		if !strings.Contains(got, want) {
			testutil.ErrorIf(t, true, "loaded view missing %q: %q", want, got)
		}
	}

	shell.SetSize(80, 20)
	_ = shell.Update(tea.KeyMsg{Type: tea.KeyDown})
}
