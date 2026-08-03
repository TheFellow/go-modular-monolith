package components

import "github.com/charmbracelet/lipgloss"

// EmptyState displays a message when there's no data.
type EmptyState struct {
	message string
	style   lipgloss.Style
}

func NewEmptyState(message string, style lipgloss.Style) EmptyState {
	return EmptyState{
		message: message,
		style:   style,
	}
}

func (e EmptyState) View() string {
	return e.style.Render(e.message)
}
