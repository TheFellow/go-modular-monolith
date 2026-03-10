package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/TheFellow/go-modular-monolith/app"
)

// Run starts the TUI with the given application.
func Run(application *app.App) error {
	model := NewApp(application)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
