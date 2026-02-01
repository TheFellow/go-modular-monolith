package views

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// ViewModel is the interface all TUI views implement.
// Domain-specific ViewModels live under app/domains/*/surfaces/tui/ and implement this interface.
type ViewModel interface {
	// Init is called once when the view is first mounted (or remounted) and
	// should return any initial command(s) to kick off async work.
	Init() tea.Cmd
	// Update is called for every incoming message (keys, window size, async results).
	// It should return the next ViewModel (often itself) and an optional command.
	Update(msg tea.Msg) (ViewModel, tea.Cmd)
	// View is called after Update to render the current state as a string.
	View() string
	// ShortHelp is queried to render the condensed help footer for this view.
	ShortHelp() []key.Binding
	// FullHelp is queried when the expanded help panel is shown.
	FullHelp() [][]key.Binding
}
