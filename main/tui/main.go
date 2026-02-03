package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/TheFellow/go-modular-monolith/app"
)

// Run starts the TUI with the given application and optional initial view.
func Run(application *app.App, initialView View) error {
	model := NewApp(application, initialView)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
