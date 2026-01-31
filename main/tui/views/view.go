package views

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// ViewModel is the interface all TUI views implement.
// Domain-specific ViewModels live under app/domains/*/surfaces/tui/ and implement this interface.
type ViewModel interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (ViewModel, tea.Cmd)
	View() string
	ShortHelp() []key.Binding
	FullHelp() [][]key.Binding
}

// ViewModelFactory creates a ViewModel for a given view.
// This allows lazy initialization of views.
type ViewModelFactory func() ViewModel
