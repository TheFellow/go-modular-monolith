package components

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Spinner wraps the bubbles spinner with styling.
type Spinner struct {
	spinner spinner.Model
	message string
	style   lipgloss.Style
}

func NewSpinner(message string, style lipgloss.Style) Spinner {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return Spinner{
		spinner: s,
		message: message,
		style:   style,
	}
}

func (s Spinner) Init() tea.Cmd {
	return s.spinner.Tick
}

func (s Spinner) Update(msg tea.Msg) (Spinner, tea.Cmd) {
	var cmd tea.Cmd
	s.spinner, cmd = s.spinner.Update(msg)
	return s, cmd
}

func (s Spinner) View() string {
	return s.style.Render(s.spinner.View() + " " + s.message)
}
