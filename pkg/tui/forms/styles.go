package forms

import "github.com/charmbracelet/lipgloss"

// FormStyles contains styles for the form and its fields.
type FormStyles struct {
	Form          lipgloss.Style
	Label         lipgloss.Style
	LabelRequired lipgloss.Style
	Input         lipgloss.Style
	InputFocused  lipgloss.Style
	Error         lipgloss.Style
	Help          lipgloss.Style
}

// FieldStyles contains styles for individual fields.
type FieldStyles struct {
	Label        lipgloss.Style
	Input        lipgloss.Style
	InputFocused lipgloss.Style
	Error        lipgloss.Style
}
