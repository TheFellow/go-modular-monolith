package views

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Placeholder is a temporary view showing "Coming Soon".
type Placeholder struct {
	title  string
	width  int
	height int
}

// NewPlaceholder creates a placeholder view with the given title.
func NewPlaceholder(title string) *Placeholder {
	return &Placeholder{title: title}
}

// Init implements tea.Model.
func (p *Placeholder) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (p *Placeholder) Update(msg tea.Msg) (ViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.height = msg.Height
	}

	return p, nil
}

// View renders the placeholder.
func (p *Placeholder) View() string {
	title := lipgloss.NewStyle().Bold(true).Render(p.title)
	message := lipgloss.NewStyle().Italic(true).Render("Coming Soon")
	content := title + "\n\n" + message

	if p.width <= 0 || p.height <= 0 {
		return content
	}

	return lipgloss.Place(p.width, p.height, lipgloss.Center, lipgloss.Center, content)
}

// ShortHelp returns no bindings; parent view provides help.
func (p *Placeholder) ShortHelp() []key.Binding {
	return nil
}

// FullHelp returns no bindings; parent view provides help.
func (p *Placeholder) FullHelp() [][]key.Binding {
	return nil
}
