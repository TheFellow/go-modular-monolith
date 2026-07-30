package dialog

import "github.com/charmbracelet/lipgloss"

// DialogStyles defines styles used by dialogs.
type DialogStyles struct {
	Modal         lipgloss.Style
	Title         lipgloss.Style
	Message       lipgloss.Style
	Button        lipgloss.Style
	ButtonFocused lipgloss.Style
	DangerButton  lipgloss.Style
}
