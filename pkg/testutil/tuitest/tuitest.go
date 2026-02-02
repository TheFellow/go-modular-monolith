package tuitest

import (
	"testing"

	"github.com/TheFellow/go-modular-monolith/main/tui/views"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type listViewStyles interface {
	~struct {
		Title       lipgloss.Style
		Subtitle    lipgloss.Style
		Muted       lipgloss.Style
		Selected    lipgloss.Style
		ListPane    lipgloss.Style
		DetailPane  lipgloss.Style
		ErrorText   lipgloss.Style
		WarningText lipgloss.Style
	}
}

type listViewKeys interface {
	~struct {
		Up      key.Binding
		Down    key.Binding
		Enter   key.Binding
		Refresh key.Binding
		Back    key.Binding
		Create  key.Binding
		Edit    key.Binding
		Delete  key.Binding
		Adjust  key.Binding
		Set     key.Binding
		Publish key.Binding
	}
}

// DefaultListViewStyles returns a minimal style set for view model tests.
func DefaultListViewStyles[T listViewStyles]() T {
	return T{
		Title:       lipgloss.NewStyle(),
		Subtitle:    lipgloss.NewStyle(),
		Muted:       lipgloss.NewStyle(),
		Selected:    lipgloss.NewStyle(),
		ListPane:    lipgloss.NewStyle(),
		DetailPane:  lipgloss.NewStyle(),
		ErrorText:   lipgloss.NewStyle(),
		WarningText: lipgloss.NewStyle(),
	}
}

// DefaultListViewKeys returns key bindings for view model tests.
func DefaultListViewKeys[T listViewKeys]() T {
	return T{
		Up:      key.NewBinding(key.WithKeys("up")),
		Down:    key.NewBinding(key.WithKeys("down")),
		Enter:   key.NewBinding(key.WithKeys("enter")),
		Refresh: key.NewBinding(key.WithKeys("r")),
		Back:    key.NewBinding(key.WithKeys("esc")),
		Create:  key.NewBinding(key.WithKeys("c")),
		Edit:    key.NewBinding(key.WithKeys("e")),
		Delete:  key.NewBinding(key.WithKeys("d")),
		Adjust:  key.NewBinding(key.WithKeys("a")),
		Set:     key.NewBinding(key.WithKeys("s")),
		Publish: key.NewBinding(key.WithKeys("p")),
	}
}

// InitAndLoad runs Init and processes the resulting commands.
func InitAndLoad(t testing.TB, model views.ViewModel) views.ViewModel {
	t.Helper()
	cmd := model.Init()
	msgs := RunCmds(cmd)
	for _, msg := range msgs {
		updated, _ := model.Update(msg)
		model = updated
	}
	return model
}

// RunCmds executes a tea.Cmd and flattens any batch messages.
func RunCmds(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if msg == nil {
		return nil
	}
	switch typed := msg.(type) {
	case tea.BatchMsg:
		var out []tea.Msg
		for _, sub := range typed {
			out = append(out, RunCmds(sub)...)
		}
		return out
	default:
		return []tea.Msg{typed}
	}
}
