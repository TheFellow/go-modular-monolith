package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/TheFellow/go-modular-monolith/app"
	"github.com/TheFellow/go-modular-monolith/pkg/middleware"
)

// Run starts the TUI with the given application and optional initial view.
func Run(ctx *middleware.Context, application *app.App, initialView View) error {
	model := NewApp(ctx, application, initialView)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
